package r5_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

func TestDiagnosticReportResourceType(t *testing.T) {
	dr := &r5.DiagnosticReport{}
	if got := dr.ResourceType(); got != "DiagnosticReport" {
		t.Errorf("ResourceType() = %q, want DiagnosticReport", got)
	}
	var _ fhir.Resource = dr
}

func TestDiagnosticReportMarshal(t *testing.T) {
	dr := &r5.DiagnosticReport{
		Identifier:        []r5.Identifier{{System: strptr("urn:dicom:uid"), Value: strptr("urn:oid:1.2.3")}},
		Status:            "final",
		Code:              &r5.CodeableConcept{Text: strptr("Radiology Report")},
		Category:          []r5.CodeableConcept{{Coding: []r5.Coding{{System: strptr("http://terminology.hl7.org/CodeSystem/v2-0074"), Code: strptr("RAD")}}}},
		Subject:           &r5.Reference{Type: strptr("Patient"), Reference: strptr("Patient/pat-1")},
		EffectiveDateTime: strptr("2026-06-01"),
		Conclusion:        strptr("No acute findings."),
		Result:            []r5.Reference{{Reference: strptr("urn:uuid:obs-1")}},
	}
	b, err := json.Marshal(dr)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.HasPrefix(got, `{"resourceType":"DiagnosticReport"`) {
		t.Errorf("DiagnosticReport JSON does not start with resourceType: %s", got)
	}
	want := `{"resourceType":"DiagnosticReport",` +
		`"identifier":[{"system":"urn:dicom:uid","value":"urn:oid:1.2.3"}],"status":"final",` +
		`"category":[{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/v2-0074","code":"RAD"}]}],` +
		`"code":{"text":"Radiology Report"},` +
		`"subject":{"reference":"Patient/pat-1","type":"Patient"},` +
		`"effectiveDateTime":"2026-06-01","conclusion":"No acute findings.",` +
		`"result":[{"reference":"urn:uuid:obs-1"}]}`
	if got != want {
		t.Errorf("DiagnosticReport JSON\n got = %s\nwant = %s", got, want)
	}
}
