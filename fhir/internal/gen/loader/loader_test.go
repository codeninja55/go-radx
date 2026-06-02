package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyChecksumsRejectsMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "a.json", []byte(`{"x":1}`))
	writeFile(t, dir, "b.json", []byte(`{"y":2}`))
	writeSums(t, dir, map[string]string{
		"a.json": sha256Hex([]byte(`{"x":1}`)),
		"b.json": sha256Hex([]byte(`{"y":2}`)),
	})
	if err := verifyChecksums(dir); err != nil {
		t.Fatalf("verifyChecksums on matching bundle: %v", err)
	}

	writeFile(t, dir, "a.json", []byte(`{"x":2}`)) // drift
	err := verifyChecksums(dir)
	if err == nil {
		t.Fatal("verifyChecksums should reject a drifted file")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "a.json") {
		t.Errorf("error %q should name the offending file", le.Error())
	}
	// The error must not leak the file's contents: neither the drifted bytes nor
	// the expected bytes should appear in the diagnostic.
	if strings.Contains(le.Error(), `{"x":2}`) || strings.Contains(le.Error(), `{"x":1}`) {
		t.Errorf("error %q should not contain file bytes", le.Error())
	}
}

func TestVerifyChecksumsRejectsMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "present.json", []byte(`{"x":1}`))
	writeSums(t, dir, map[string]string{
		"present.json": sha256Hex([]byte(`{"x":1}`)),
		"absent.json":  sha256Hex([]byte(`{"z":9}`)),
	})

	err := verifyChecksums(dir)
	if err == nil {
		t.Fatal("verifyChecksums should reject a missing listed file")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "absent.json") {
		t.Errorf("error %q should name the missing file", le.Error())
	}
}

func TestVerifyChecksumsRejectsMissingManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "a.json", []byte(`{"x":1}`))

	err := verifyChecksums(dir)
	if err == nil {
		t.Fatal("verifyChecksums should reject a directory without SHA256SUMS")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), sumsFile) {
		t.Errorf("error %q should name the manifest %q", le.Error(), sumsFile)
	}
}

// vendoredR5Dir is the committed R5 bundle, relative to this package directory
// (fhir/internal/gen/loader → fhir/internal/gen/testdata/definitions/r5).
const vendoredR5Dir = "../testdata/definitions/r5"

func TestLoadVendoredR5Bundle(t *testing.T) {
	t.Parallel()

	b, err := Load(vendoredR5Dir)
	if err != nil {
		t.Fatalf("Load(%q): %v", vendoredR5Dir, err)
	}

	wantResources := []string{
		"Patient", "Observation", "Bundle", "OperationOutcome",
		"ServiceRequest", "ImagingStudy", "DiagnosticReport",
	}
	for _, name := range wantResources {
		sd, ok := b.StructureDefinition(name)
		if !ok {
			t.Errorf("resource %q not indexed by name", name)
			continue
		}
		if sd.Kind != "resource" {
			t.Errorf("%q kind = %q, want resource", name, sd.Kind)
		}
	}

	wantDatatypes := []string{
		"Reference", "Identifier", "CodeableConcept", "Quantity", "HumanName", "Period",
	}
	for _, name := range wantDatatypes {
		sd, ok := b.StructureDefinition(name)
		if !ok {
			t.Errorf("datatype %q not indexed by name", name)
			continue
		}
		if sd.Kind != "complex-type" {
			t.Errorf("%q kind = %q, want complex-type", name, sd.Kind)
		}
	}

	// Resolve a StructureDefinition by its canonical URL as well as by name.
	const patientURL = "http://hl7.org/fhir/StructureDefinition/Patient"
	if _, ok := b.StructureDefinitionByURL(patientURL); !ok {
		t.Errorf("Patient not indexed by URL %q", patientURL)
	}

	// The administrative-gender value set must resolve (the required-binding enums
	// in Increment 8 enumerate its codes).
	const genderVS = "http://hl7.org/fhir/ValueSet/administrative-gender"
	if _, ok := b.ValueSet(genderVS); !ok {
		t.Errorf("administrative-gender value set not indexed by URL %q", genderVS)
	}

	// The resource count sits in a band rather than an exact number so a
	// definition-bundle patch release does not break the test; the exact count is
	// asserted in the conformance statement, not here.
	n := b.ResourceCount()
	if n < 140 || n > 170 {
		t.Errorf("resource StructureDefinition count = %d, want within [140,170]", n)
	}
	t.Logf("loaded %d resource and %d datatype StructureDefinitions, %d value sets, %d code systems",
		b.ResourceCount(), b.DatatypeCount(), b.ValueSetCount(), b.CodeSystemCount())
}

func TestLoadRejectsMissingValuesets(t *testing.T) {
	t.Parallel()

	// A temp bundle directory with only two of the three required files, plus a
	// SHA256SUMS listing all three: Load must fail closed naming valuesets.json.
	dir := t.TempDir()
	types := []byte(`{"resourceType":"Bundle","entry":[]}`)
	resources := []byte(`{"resourceType":"Bundle","entry":[]}`)
	valuesets := []byte(`{"resourceType":"Bundle","entry":[]}`)
	writeFile(t, dir, "profiles-types.json", types)
	writeFile(t, dir, "profiles-resources.json", resources)
	writeSums(t, dir, map[string]string{
		"profiles-types.json":     sha256Hex(types),
		"profiles-resources.json": sha256Hex(resources),
		"valuesets.json":          sha256Hex(valuesets),
	})

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load should fail when valuesets.json is missing")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "valuesets.json") {
		t.Errorf("error %q should name the missing file", le.Error())
	}
}

func TestLoadRejectsMalformedBundle(t *testing.T) {
	t.Parallel()

	// All three files present and checksum-correct, but one is not valid JSON:
	// Load must fail closed naming that file rather than silently skipping it.
	dir := t.TempDir()
	good := []byte(`{"resourceType":"Bundle","entry":[]}`)
	bad := []byte(`{"resourceType":"Bundle","entry":[`) // truncated
	writeFile(t, dir, "profiles-types.json", good)
	writeFile(t, dir, "profiles-resources.json", bad)
	writeFile(t, dir, "valuesets.json", good)
	writeSums(t, dir, map[string]string{
		"profiles-types.json":     sha256Hex(good),
		"profiles-resources.json": sha256Hex(bad),
		"valuesets.json":          sha256Hex(good),
	})

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load should fail on a malformed bundle file")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "profiles-resources.json") {
		t.Errorf("error %q should name the malformed file", le.Error())
	}
}

func TestLoadRejectsUnpinnedRequiredFile(t *testing.T) {
	t.Parallel()

	// All three required files exist and the listed ones hash correctly, but the
	// manifest omits valuesets.json. An incomplete manifest must fail closed
	// rather than letting Load parse the unpinned file.
	dir := t.TempDir()
	body := []byte(`{"resourceType":"Bundle","entry":[]}`)
	for _, name := range requiredFiles {
		writeFile(t, dir, name, body)
	}
	writeSums(t, dir, map[string]string{
		"profiles-types.json":     sha256Hex(body),
		"profiles-resources.json": sha256Hex(body),
		// valuesets.json deliberately omitted.
	})

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load should reject a required file missing from the manifest")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "valuesets.json") {
		t.Errorf("error %q should name the unpinned file", le.Error())
	}
}

func TestLoadRejectsTruncatedBundleObject(t *testing.T) {
	t.Parallel()

	// The entry array closes but the outer object is never closed. Without
	// consuming the final delimiter Load would accept this as an empty bundle;
	// it must fail closed instead.
	dir := t.TempDir()
	good := []byte(`{"resourceType":"Bundle","entry":[]}`)
	truncated := []byte(`{"resourceType":"Bundle","entry":[]`) // missing closing '}'
	writeFile(t, dir, "profiles-types.json", truncated)
	writeFile(t, dir, "profiles-resources.json", good)
	writeFile(t, dir, "valuesets.json", good)
	writeSums(t, dir, map[string]string{
		"profiles-types.json":     sha256Hex(truncated),
		"profiles-resources.json": sha256Hex(good),
		"valuesets.json":          sha256Hex(good),
	})

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load should fail on a truncated bundle object")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "profiles-types.json") {
		t.Errorf("error %q should name the truncated file", le.Error())
	}
}

func TestLoadRejectsTrailingContent(t *testing.T) {
	t.Parallel()

	// Garbage after the bundle object must be rejected, not silently ignored.
	dir := t.TempDir()
	good := []byte(`{"resourceType":"Bundle","entry":[]}`)
	trailing := []byte(`{"resourceType":"Bundle","entry":[]} {"junk":1}`)
	writeFile(t, dir, "profiles-types.json", good)
	writeFile(t, dir, "profiles-resources.json", trailing)
	writeFile(t, dir, "valuesets.json", good)
	writeSums(t, dir, map[string]string{
		"profiles-types.json":     sha256Hex(good),
		"profiles-resources.json": sha256Hex(trailing),
		"valuesets.json":          sha256Hex(good),
	})

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load should fail on trailing content after the bundle object")
	}
	var le *LoadError
	if !errors.As(err, &le) {
		t.Fatalf("error = %T, want *LoadError", err)
	}
	if !strings.Contains(le.Error(), "profiles-resources.json") {
		t.Errorf("error %q should name the offending file", le.Error())
	}
}

func writeFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func writeSums(t *testing.T, dir string, sums map[string]string) {
	t.Helper()
	var b strings.Builder
	for name, sum := range sums {
		b.WriteString(sum)
		b.WriteString("  ")
		b.WriteString(name)
		b.WriteString("\n")
	}
	writeFile(t, dir, sumsFile, []byte(b.String()))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
