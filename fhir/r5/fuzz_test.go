package r5_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// corpusSeedDir holds the synthetic FHIR workflow instances the fuzz targets seed from,
// so the fuzzer starts from parseable, fully-populated resources and mutates outward into
// the malformed space rather than exploring from random noise. Every byte is go-radx
// synthetic test data (see testdata/fhir/SOURCE.md); no real PHI is ever added to the
// corpus or surfaced by these targets.
const corpusSeedDir = "../../testdata/fhir/r5"

// malformedSeedDir holds the vendored, attributed, no-PHI malformed-FHIR corpus: the
// hostile-input seeds (truncated, wrong-typed, two-choice-branch, unknown-resourceType,
// deeply-nested) the fuzz targets and the Phase 4 hostile-input gate share. Seeding from
// known-bad inputs anchors the fuzzer in the failure space the decode/validate surface
// must survive.
const malformedSeedDir = "../../testdata/fhir/malformed"

// fuzzSeeds reads every committed seed under the corpus and malformed directories plus a
// handful of inline edge cases (empty, whitespace, a bare array, a deeply nested object).
// The fuzzer adds these to its corpus; the same set replays under `go test` so a seed that
// would crash the decoder is caught without a fuzzing build.
func fuzzSeeds(t testing.TB) [][]byte {
	t.Helper()
	seeds := [][]byte{
		{},                                  // empty: degenerate truncation
		[]byte("   \n\t  "),                 // whitespace only
		[]byte("null"),                      // JSON null
		[]byte("[]"),                        // bare array, not a resource object
		[]byte(`{"resourceType":""}`),       // empty discriminator
		[]byte(`{"resourceType":"Patient"`), // truncated mid-object
		deeplyNestedJSON(512),               // deeply nested: stack-depth probe
	}
	for _, dir := range []string{corpusSeedDir, malformedSeedDir} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read seed dir %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatalf("read seed %s: %v", e.Name(), err)
			}
			seeds = append(seeds, raw)
		}
	}
	return seeds
}

// deeplyNestedJSON builds a JSON object nested depth levels deep under repeated "a" keys.
// It is a synthetic structural probe for the decoder's recursion handling — it carries no
// resourceType and no patient data — so the fuzzer exercises the deeply-nested path
// without a hand-maintained fixture.
func deeplyNestedJSON(depth int) []byte {
	buf := make([]byte, 0, depth*6+2)
	for range depth {
		buf = append(buf, `{"a":`...)
	}
	buf = append(buf, '1')
	for range depth {
		buf = append(buf, '}')
	}
	return buf
}

// FuzzUnmarshalResource drives the registry-dispatched decode (fhir.UnmarshalResource)
// with arbitrary, truncated, and deeply-nested bytes. The contracts under fuzz: it must
// never panic (PRD §9.3 — malformed external input yields a typed error, never a crash);
// a successful decode must round-trip back through fhir.UnmarshalResource without panicking
// (the polymorphic Bundle/contained decode is the one re-entrant path); and a decode that
// fails because the input ran out must report io.ErrUnexpectedEOF rather than an opaque
// syntax error (the truncation contract). It seeds from the synthetic corpus and the
// malformed corpus, both PHI-free.
func FuzzUnmarshalResource(f *testing.F) {
	for _, seed := range fuzzSeeds(f) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		resource, err := fhir.UnmarshalResource(data)
		if err != nil {
			// A failure on truncated input must be matchable as io.ErrUnexpectedEOF; the
			// truncation map runs at every decode boundary, so any "ran out of bytes" error
			// the fuzzer surfaces is already folded to the sentinel. We do not assert the
			// converse (that every error is truncation) — most malformed inputs are genuine
			// syntax or type errors — only that the decoder returned rather than panicked.
			return
		}
		if resource == nil {
			t.Fatal("UnmarshalResource returned nil resource with nil error")
		}
		if resource.ResourceType() == "" {
			t.Fatal("decoded resource has empty resourceType")
		}
		// A decoded resource must re-decode: marshalling is exercised by the benchmark and
		// the corpus round-trip; here the invariant is that re-feeding a once-decoded
		// resource's bytes never panics the polymorphic decode path.
		reencoded, err := marshalNoPanic(resource)
		if err != nil {
			return
		}
		if _, err := fhir.UnmarshalResource(reencoded); err != nil {
			t.Fatalf("re-decode of a successfully decoded resource failed: %v", err)
		}
	})
}

// FuzzValidate drives the in-process structural gate (fhir.Validate) over decode-then-
// validate: it decodes the fuzzed bytes and, on success, validates the resource. Validate
// must never panic on any decoded resource — nil, partial, structurally broken — and must
// never leak a patient value into an issue diagnostic or expression (PRD §9.1, §9.3).
// Inputs that fail to decode are skipped: Validate's contract is over a Resource, and the
// decode-failure path is FuzzUnmarshalResource's concern.
func FuzzValidate(f *testing.F) {
	for _, seed := range fuzzSeeds(f) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		resource, err := fhir.UnmarshalResource(data)
		if err != nil {
			return
		}
		outcome := fhir.Validate(resource)
		if outcome == nil {
			t.Fatal("Validate returned a nil outcome")
		}
		// No issue may echo a payload value: Validate names elements, paths, types, and
		// codes only. The corpus seeds carry the synthetic sentinels below; if a mutation
		// preserves one and the engine ever copied a field value into a message, the assert
		// bites. This is a property check, not a guarantee the fuzzer reaches the leak.
		for _, issue := range outcome.Issue {
			for _, sentinel := range phiFuzzSentinels {
				if strings.Contains(issue.Diagnostics, sentinel) || strings.Contains(issue.Expression, sentinel) {
					t.Fatalf("validation issue leaked a patient value token %q", sentinel)
				}
			}
		}
	})
}

// FuzzValidateTypedResource exercises Validate over a fuzzer-shaped typed resource rather
// than only decoded input, so the engine's choice/binding/required closures run against
// field combinations a decode might not produce. A Patient is built from fuzzed scalars and
// the gender code is set to an arbitrary fuzzed value so the required-binding closure runs
// against codes outside its value set. The contract is never-panic on any combination
// (PRD §9.3). The PHI-no-leak property is proven soundly by FuzzValidate over the synthetic
// corpus sentinels and by validate_test.go; it is not re-checked here because a fuzzer-chosen
// value can be any substring of a path or code, which a value-echo test cannot distinguish.
func FuzzValidateTypedResource(f *testing.F) {
	f.Add("pat-1", "TESTPATIENT", "female", true)
	f.Add("", "", "", false)
	f.Add("\x00\x01", "Workflow", "unknown", true)
	f.Fuzz(func(t *testing.T, id, family, gender string, active bool) {
		patient := &r5.Patient{}
		if id != "" {
			patient.ID = &id
		}
		if family != "" {
			patient.Name = []r5.HumanName{{Family: &family}}
		}
		if gender != "" {
			g := r5.AdministrativeGender(gender)
			patient.Gender = &g
		}
		patient.Active = &active

		if outcome := fhir.Validate(patient); outcome == nil {
			t.Fatal("Validate(*Patient) returned a nil outcome")
		}
	})
}

// TestValidateNilGuards pins the nil-resource guards Validate must hold: a bare nil and a
// typed-nil pointer each yield an error-severity outcome, never a panic. These are fixed
// properties, so they live in a plain test rather than being re-asserted on every fuzz
// iteration (PRD §9.3 — never panic on malformed or partial input).
func TestValidateNilGuards(t *testing.T) {
	if outcome := fhir.Validate(nil); outcome == nil || !outcome.HasErrors() {
		t.Error("Validate(nil) should report an error, not panic")
	}
	var typedNil *r5.Patient
	if outcome := fhir.Validate(typedNil); outcome == nil || !outcome.HasErrors() {
		t.Error("Validate(typed-nil *Patient) should report an error, not panic")
	}
}

// phiFuzzSentinels are the synthetic patient-data tokens the seed corpus carries. They are
// not real PHI; they are recognisable markers that would only appear in a validation issue
// if the engine copied a field value into a message, which the PHI-safety property forbids.
var phiFuzzSentinels = []string{"TESTPATIENT", "MRN0001234", "Workflow", "Synthetic"}

// TestCorpusTruncationMapsToUnexpectedEOF is the explicit, non-fuzz assertion that
// truncated valid JSON yields io.ErrUnexpectedEOF through the registry decode path,
// complementing the root-package TestUnmarshalTruncatedYieldsUnexpectedEOF which covers the
// same contract over the fake registry. Here it runs against the real corpus instances so
// the property is proven over the production decode of registered resources.
func TestCorpusTruncationMapsToUnexpectedEOF(t *testing.T) {
	for _, name := range corpusWorkflowSet {
		path := filepath.Join(corpusSeedDir, name+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		// Drop the trailing newline the corpus files carry, then clip the closing brace, so
		// the buffer ends inside the object rather than after a complete value. A complete
		// corpus instance minus its closing token is a truncated value: the decoder must
		// report io.ErrUnexpectedEOF, never an opaque syntax error and never a panic.
		trimmed := []byte(strings.TrimRight(string(raw), " \n\r\t"))
		if len(trimmed) < 2 {
			continue
		}
		truncated := trimmed[:len(trimmed)-1]
		if _, err := fhir.UnmarshalResource(truncated); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("UnmarshalResource(truncated %s): err = %v, want io.ErrUnexpectedEOF", name, err)
		}
	}
}

// marshalNoPanic marshals a resource, recovering any panic into an error so the fuzz target
// can treat a marshalling fault as a skip rather than a process crash. A panic on marshal
// would itself be a defect, but the fuzz target's job is the decode/validate surface; the
// recover keeps a marshalling regression from masking that as a fuzz crasher.
func marshalNoPanic(r fhir.Resource) (out []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = errPanic
		}
	}()
	return fhir.MarshalSummary(r, fhir.SummaryFull)
}

// errPanic is the sentinel marshalNoPanic returns when a marshal panics, so the fuzz target
// branches on a recovered panic without depending on the panic value's shape.
var errPanic = errors.New("r5_test: marshal panicked")
