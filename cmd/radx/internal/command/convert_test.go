package command

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
)

// Synthetic, non-PHI HL7 v2 messages for the convert tests. The names and identifiers are
// deliberately fictional test tokens (PRD §9.1).
const (
	convertADT = "MSH|^~\\&|ADT|TEST|EMR|TEST|20260101120000||ADT^A01|CVTADT1|P|2.4\r" +
		"EVN|A01|20260101120000\r" +
		"PID|||TEST-1^^^TEST^MR||FIXTURE^TESTONE^^^^^L||20000101|O\r" +
		"PV1|1|I|WARD^1^A||||||||||||||||VISIT-1\r"

	convertORU = "MSH|^~\\&|LIS|TEST|EMR|TEST|20260101120000||ORU^R01|CVTORU1|P|2.4\r" +
		"PID|||TEST-1^^^TEST^MR||FIXTURE^TESTONE^^^^^L||20000101|O\r" +
		"OBR|1|PLC-1|FIL-1|TESTPANEL^TEST PANEL^LOCAL|||20260101120100\r" +
		"OBX|1|NM|TESTAN^TEST ANALYTE^LOCAL||42|u^u^UCUM|0-100|N|||F\r"

	convertORM = "MSH|^~\\&|RADX|TEST|PACS|TEST|20260101120000||ORM^O01|CVTORM1|P|2.4\r" +
		"PID|||TEST-1^^^TEST^MR||FIXTURE^TESTONE^^^^^L||20000101|O\r" +
		"ORC|NW|PLC-1|FIL-1||||||20260101120000\r" +
		"OBR|1|PLC-1|FIL-1|TESTCODE^TEST PROCEDURE^LOCAL|||20260101120100\r"
)

// hasResourceType asserts that the JSON document on stdout carries a resourceType field equal to
// want, the marker every FHIR resource serialises with.
func assertResourceType(t *testing.T, stdout, want string) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if rt, _ := doc["resourceType"].(string); rt != want {
		t.Errorf("resourceType = %q, want %q\nstdout=%q", rt, want, stdout)
	}
}

// TestConvertADTToPatient confirms an ADT maps to a Patient under the default R5 release.
func TestConvertADTToPatient(t *testing.T) {
	stdout, stderr, code := runRadxStdin(t, convertADT, "convert", "adt-to-fhir", "-")
	if code != exitcode.Success {
		t.Fatalf("convert adt-to-fhir exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	assertResourceType(t, stdout, "Patient")
}

// TestConvertADTToEncounter confirms --as encounter maps an ADT to an Encounter.
func TestConvertADTToEncounter(t *testing.T) {
	stdout, _, code := runRadxStdin(t, convertADT, "convert", "adt-to-fhir", "--as", "encounter", "-")
	if code != exitcode.Success {
		t.Fatalf("convert adt-to-fhir --as encounter exit = %d, want %d", code, exitcode.Success)
	}
	assertResourceType(t, stdout, "Encounter")
}

// TestConvertADTToPatientR4 confirms --release R4 selects the R4 converter twin (a different code
// path producing an R4 Patient).
func TestConvertADTToPatientR4(t *testing.T) {
	stdout, _, code := runRadxStdin(t, convertADT, "convert", "adt-to-fhir", "--release", "R4", "-")
	if code != exitcode.Success {
		t.Fatalf("convert adt-to-fhir --release R4 exit = %d, want %d", code, exitcode.Success)
	}
	assertResourceType(t, stdout, "Patient")
}

// TestConvertORUToDiagnosticReport confirms an ORU maps to a DiagnosticReport bundle.
func TestConvertORUToDiagnosticReport(t *testing.T) {
	stdout, stderr, code := runRadxStdin(t, convertORU, "convert", "oru-to-fhir", "-")
	if code != exitcode.Success {
		t.Fatalf("convert oru-to-fhir exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	var bundle map[string]any
	if err := json.Unmarshal([]byte(stdout), &bundle); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	dr, _ := bundle["diagnosticReport"].(map[string]any)
	if rt, _ := dr["resourceType"].(string); rt != "DiagnosticReport" {
		t.Errorf("diagnosticReport.resourceType = %q, want DiagnosticReport", rt)
	}
}

// TestConvertORMToServiceRequest confirms an ORM maps to a ServiceRequest.
func TestConvertORMToServiceRequest(t *testing.T) {
	stdout, stderr, code := runRadxStdin(t, convertORM, "convert", "orm-to-fhir", "-")
	if code != exitcode.Success {
		t.Fatalf("convert orm-to-fhir exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	assertResourceType(t, stdout, "ServiceRequest")
}

// TestConvertDICOMToImagingStudy confirms a DICOM instance maps to an ImagingStudy.
func TestConvertDICOMToImagingStudy(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.950.1")
	stdout, stderr, code := runRadx(t, "convert", "dicom-to-fhir", src)
	if code != exitcode.Success {
		t.Fatalf("convert dicom-to-fhir exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	assertResourceType(t, stdout, "ImagingStudy")
}

// TestConvertSRToDiagnosticReport confirms a DICOM SR maps to a DiagnosticReport bundle, using the
// validated basic-text-SR fixture from the root module's testdata.
func TestConvertSRToDiagnosticReport(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "..", "testdata", "dicom", "basic-text-sr.dcm")
	stdout, stderr, code := runRadx(t, "convert", "sr-to-fhir", fixture)
	if code != exitcode.Success {
		t.Fatalf("convert sr-to-fhir exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	var bundle map[string]any
	if err := json.Unmarshal([]byte(stdout), &bundle); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	dr, _ := bundle["diagnosticReport"].(map[string]any)
	if rt, _ := dr["resourceType"].(string); rt != "DiagnosticReport" {
		t.Errorf("diagnosticReport.resourceType = %q, want DiagnosticReport", rt)
	}
}

// TestConvertMalformedSourceFailsClosed confirms a malformed HL7 source makes the conversion exit
// non-zero rather than emitting a lossy or empty resource (fail-closed, PRD §9.2).
func TestConvertMalformedSourceFailsClosed(t *testing.T) {
	_, _, code := runRadxStdin(t, "this is not an HL7 message", "convert", "oru-to-fhir", "-")
	if code == exitcode.Success {
		t.Fatalf("convert of a malformed ORU exited 0; want non-zero (fail-closed)")
	}
}

// TestConvertCSVIsUsageError confirms --format csv is rejected for the conversions, which emit FHIR
// resources rather than tables.
func TestConvertCSVIsUsageError(t *testing.T) {
	_, _, code := runRadxStdin(t, convertADT, "convert", "adt-to-fhir", "--format", "csv", "-")
	if code != exitcode.UsageError {
		t.Fatalf("convert --format csv exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}
