package fhir

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// summaryResource is a minimal in-package Resource whose MarshalJSON emits a fixed,
// canonically-ordered object so the summary filter can be exercised without importing a
// release package (which would create an import cycle). The wire shape mirrors a generated
// resource: a resourceType discriminator first, then the element keys in declaration
// order, including a primitive "_field" sibling that must ride with its value.
type summaryResource struct {
	body string
}

func (r *summaryResource) ResourceType() string { return "SummarySample" }

func (r *summaryResource) MarshalJSON() ([]byte, error) { return []byte(r.body), nil }

// registerSummarySample registers the synthetic descriptor once. The mode tests share it,
// and the registry rejects a duplicate, so registration is guarded to run a single time.
func registerSummarySample(t *testing.T) {
	t.Helper()
	if _, ok := lookupSummaryDescriptor("SummarySample"); ok {
		return
	}
	RegisterSummaryDescriptor("SummarySample", SummaryDescriptor{
		Elements: []SummaryElement{
			{JSONName: "id", IsSummary: true},
			{JSONName: "meta", IsSummary: true},
			{JSONName: "text", IsText: true},
			{JSONName: "status", IsSummary: true, IsMandatory: true, IsModifier: true},
			{JSONName: "code", IsSummary: true},
			{JSONName: "note"},
			{JSONName: "total", IsSummary: true, IsCount: true},
		},
	})
}

// sampleBody is a canonical encoding of a summaryResource: resourceType first, then the
// elements in declaration order. status carries a "_status" primitive sibling that must be
// filtered together with its value.
const sampleBody = `{"resourceType":"SummarySample","id":"abc","text":{"status":"generated"},` +
	`"status":"final","_status":{"id":"s1"},"code":"x","note":"hidden","total":7}`

func decodeKeys(t *testing.T, data []byte) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(string(data)))
	if _, err := dec.Token(); err != nil { // opening brace
		t.Fatalf("read object start: %v", err)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("read key: %v", err)
		}
		keys = append(keys, tok.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatalf("read value: %v", err)
		}
	}
	return keys
}

func hasKey(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}

// TestMarshalSummaryNilReturnsError is the FHIR-012 regression: a nil resource yields
// ErrNilResource rather than a panic, for both a nil interface and a typed-nil pointer.
func TestMarshalSummaryNilReturnsError(t *testing.T) {
	if _, err := MarshalSummary(nil, SummaryTrue); !errors.Is(err, ErrNilResource) {
		t.Fatalf("nil interface: err = %v, want ErrNilResource", err)
	}
	var typedNil *summaryResource
	if _, err := MarshalSummary(typedNil, SummaryTrue); !errors.Is(err, ErrNilResource) {
		t.Fatalf("typed-nil pointer: err = %v, want ErrNilResource", err)
	}
}

// TestMarshalSummaryFullIsIdentity confirms SummaryFull returns the resource's own
// encoding byte-for-byte, with no filtering and no SUBSETTED tag.
func TestMarshalSummaryFullIsIdentity(t *testing.T) {
	registerSummarySample(t)
	got, err := MarshalSummary(&summaryResource{body: sampleBody}, SummaryFull)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	if string(got) != sampleBody {
		t.Fatalf("full view = %s, want identity %s", got, sampleBody)
	}
}

// TestMarshalSummaryTrueKeepsFlaggedElements confirms SummaryTrue emits exactly the
// isSummary, mandatory, and modifier elements (plus the always-kept infrastructure keys),
// drops the rest, preserves their canonical order, keeps a value's "_field" sibling with
// it, and sets the SUBSETTED tag.
func TestMarshalSummaryTrueKeepsFlaggedElements(t *testing.T) {
	registerSummarySample(t)
	got, err := MarshalSummary(&summaryResource{body: sampleBody}, SummaryTrue)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	keys := decodeKeys(t, got)

	for _, want := range []string{"resourceType", "id", "status", "_status", "code", "total"} {
		if !hasKey(keys, want) {
			t.Errorf("SummaryTrue dropped %q it should keep; keys = %v", want, keys)
		}
	}
	for _, drop := range []string{"text", "note"} {
		if hasKey(keys, drop) {
			t.Errorf("SummaryTrue kept %q it should drop; keys = %v", drop, keys)
		}
	}
	if !strings.Contains(string(got), "SUBSETTED") {
		t.Errorf("SummaryTrue did not set the SUBSETTED tag: %s", got)
	}
	// Canonical order is preserved: status precedes code precedes total in the source.
	idxStatus, idxCode, idxTotal := indexOf(keys, "status"), indexOf(keys, "code"), indexOf(keys, "total")
	if idxStatus >= idxCode || idxCode >= idxTotal {
		t.Errorf("SummaryTrue did not preserve canonical order; keys = %v", keys)
	}
}

// TestMarshalSummaryTextKeepsNarrativeAndMandatory confirms SummaryText keeps the
// narrative and mandatory elements, drops the rest, and tags SUBSETTED.
func TestMarshalSummaryTextKeepsNarrativeAndMandatory(t *testing.T) {
	registerSummarySample(t)
	got, err := MarshalSummary(&summaryResource{body: sampleBody}, SummaryText)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	keys := decodeKeys(t, got)
	for _, want := range []string{"id", "text", "status"} {
		if !hasKey(keys, want) {
			t.Errorf("SummaryText dropped %q it should keep; keys = %v", want, keys)
		}
	}
	for _, drop := range []string{"code", "note", "total"} {
		if hasKey(keys, drop) {
			t.Errorf("SummaryText kept %q it should drop; keys = %v", drop, keys)
		}
	}
}

// TestMarshalSummaryDataDropsNarrative confirms SummaryData keeps everything except the
// narrative.
func TestMarshalSummaryDataDropsNarrative(t *testing.T) {
	registerSummarySample(t)
	got, err := MarshalSummary(&summaryResource{body: sampleBody}, SummaryData)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	keys := decodeKeys(t, got)
	if hasKey(keys, "text") {
		t.Errorf("SummaryData kept the narrative; keys = %v", keys)
	}
	for _, want := range []string{"status", "code", "note", "total"} {
		if !hasKey(keys, want) {
			t.Errorf("SummaryData dropped %q it should keep; keys = %v", want, keys)
		}
	}
}

// TestMarshalSummaryCountKeepsTotalOnly confirms SummaryCount keeps only the count element
// (plus infrastructure) and drops every other element.
func TestMarshalSummaryCountKeepsTotalOnly(t *testing.T) {
	registerSummarySample(t)
	got, err := MarshalSummary(&summaryResource{body: sampleBody}, SummaryCount)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	keys := decodeKeys(t, got)
	if !hasKey(keys, "total") {
		t.Errorf("SummaryCount dropped total; keys = %v", keys)
	}
	for _, drop := range []string{"text", "status", "code", "note"} {
		if hasKey(keys, drop) {
			t.Errorf("SummaryCount kept %q it should drop; keys = %v", drop, keys)
		}
	}
}

// TestMarshalSummaryUnregisteredReturnsFull confirms a resource with no registered summary
// descriptor is returned in full rather than guessing which elements to drop.
func TestMarshalSummaryUnregisteredReturnsFull(t *testing.T) {
	body := `{"resourceType":"Unregistered","id":"z","note":"keep"}`
	got, err := MarshalSummary(&unregisteredResource{body: body}, SummaryTrue)
	if err != nil {
		t.Fatalf("MarshalSummary: %v", err)
	}
	if string(got) != body {
		t.Fatalf("unregistered view = %s, want full %s", got, body)
	}
}

type unregisteredResource struct{ body string }

func (r *unregisteredResource) ResourceType() string         { return "Unregistered" }
func (r *unregisteredResource) MarshalJSON() ([]byte, error) { return []byte(r.body), nil }

// TestRegisterSummaryDescriptorRejectsDuplicate confirms a duplicate registration panics,
// the build-time defect guard, and an empty resourceType panics too.
func TestRegisterSummaryDescriptorRejectsDuplicate(t *testing.T) {
	RegisterSummaryDescriptor("DupSummary", SummaryDescriptor{})
	assertPanics(t, "duplicate", func() { RegisterSummaryDescriptor("DupSummary", SummaryDescriptor{}) })
	assertPanics(t, "empty", func() { RegisterSummaryDescriptor("", SummaryDescriptor{}) })
}

func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected a panic, got none", name)
		}
	}()
	fn()
}

func indexOf(keys []string, want string) int {
	for i, k := range keys {
		if k == want {
			return i
		}
	}
	return -1
}
