package convert

import (
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// srFixture reads the vendored Basic Text SR document.
func srFixture(t *testing.T) *dicom.DataSet {
	t.Helper()
	f, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "basic-text-sr.dcm"))
	if err != nil {
		t.Fatalf("read SR fixture: %v", err)
	}
	return f.DataSet
}

func TestSRToDiagnosticReportR5(t *testing.T) {
	sr := srFixture(t)

	dr, report, err := SRToDiagnosticReportR5(sr)
	if err != nil {
		t.Fatalf("SRToDiagnosticReportR5: %v", err)
	}
	if dr == nil {
		t.Fatal("DiagnosticReport is nil")
	}

	// The SOP Instance UID becomes the identifier via UIDIdentifierR5, never a URL.
	wantSOP, _ := sr.GetString(dicom.TagSOPInstanceUID)
	if len(dr.Identifier) != 1 {
		t.Fatalf("len(Identifier) = %d, want 1", len(dr.Identifier))
	}
	if dr.Identifier[0].System == nil || *dr.Identifier[0].System != "urn:dicom:uid" {
		t.Errorf("Identifier.System = %v, want urn:dicom:uid", dr.Identifier[0].System)
	}
	if dr.Identifier[0].Value == nil || *dr.Identifier[0].Value != "urn:oid:"+wantSOP {
		t.Errorf("Identifier.Value = %v, want urn:oid:%s", dr.Identifier[0].Value, wantSOP)
	}

	// CompletionFlag PARTIAL + VerificationFlag UNVERIFIED maps to preliminary.
	if dr.Status == nil || *dr.Status != r5.DiagnosticReportStatusPreliminary {
		t.Errorf("Status = %v, want preliminary", dr.Status)
	}

	// The root CONTAINER concept name becomes the code.
	if dr.Code == nil || len(dr.Code.Coding) == 0 {
		t.Fatalf("Code not populated: %+v", dr.Code)
	}

	// Category is the imaging service section.
	if len(dr.Category) == 0 || len(dr.Category[0].Coding) == 0 ||
		dr.Category[0].Coding[0].Code == nil || *dr.Category[0].Coding[0].Code != "IMG" {
		t.Errorf("Category = %+v, want an IMG coding", dr.Category)
	}

	// ContentDate/Time becomes the effective dateTime via the effective[x] choice
	// setter. The fixture's ContentTime carries no timezone offset, and FHIR forbids
	// a timezone-less time, so the effective value is the date and the dropped time
	// is recorded.
	if dr.EffectiveDateTime == nil || string(*dr.EffectiveDateTime) != "2005-05-30" {
		t.Errorf("EffectiveDateTime = %v, want 2005-05-30 (time dropped for lack of offset)", dr.EffectiveDateTime)
	}

	// The TEXT items concatenate into the conclusion.
	if dr.Conclusion == nil || *dr.Conclusion == "" {
		t.Errorf("Conclusion not populated from TEXT items: %+v", dr.Conclusion)
	}

	// The fixture has no PatientID, so subject is left unset and recorded.
	if dr.Subject != nil {
		t.Errorf("Subject = %+v, want nil (fixture has no PatientID)", dr.Subject)
	}
	if !hasDefaultTarget(report, "DiagnosticReport.subject") {
		t.Errorf("Report.Defaulted does not record the absent subject: %+v", report.Defaulted)
	}
}

// TestSRToDiagnosticReportR5RejectsNonSR confirms a non-SR dataset fails closed.
func TestSRToDiagnosticReportR5RejectsNonSR(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.4") // MR Image, not an SR
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.3")

	if _, _, err := SRToDiagnosticReportR5(ds); err == nil {
		t.Fatal("SRToDiagnosticReportR5 of a non-SR dataset returned nil error, want a fail-closed error")
	}
}

// hasDefaultTarget reports whether the report records any default for target.
func hasDefaultTarget(r *Report, target string) bool {
	for _, d := range r.Defaulted {
		if d.Target == target {
			return true
		}
	}
	return false
}
