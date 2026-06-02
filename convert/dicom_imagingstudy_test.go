package convert

import (
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// instance builds a minimal DICOM instance with the study/series/SOP identity and
// modality the ImagingStudy converter reads.
func instance(study, series, sop, modality string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.4") // MR Image Storage
	ds.SetString(dicom.TagSOPInstanceUID, sop)
	ds.SetString(dicom.TagStudyInstanceUID, study)
	ds.SetString(dicom.TagSeriesInstanceUID, series)
	if modality != "" {
		ds.SetString(dicom.TagModality, modality)
	}
	return ds
}

// TestDICOMToImagingStudyR5RecomputesCounts is the key regression: two instances
// of one study (one series, two SOP instances) recompute numberOfSeries=1 and
// numberOfInstances=2, and the subject is NEVER a fabricated Reference URL.
func TestDICOMToImagingStudyR5RecomputesCounts(t *testing.T) {
	insts := []*dicom.DataSet{
		instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR"),
		instance("1.2.3", "1.2.3.1", "1.2.3.1.2", "MR"),
	}

	study, report, err := DICOMToImagingStudyR5(insts)
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}

	if study.NumberOfSeries == nil || *study.NumberOfSeries != 1 {
		t.Errorf("NumberOfSeries = %v, want 1", study.NumberOfSeries)
	}
	if study.NumberOfInstances == nil || *study.NumberOfInstances != 2 {
		t.Errorf("NumberOfInstances = %v, want 2", study.NumberOfInstances)
	}
	if len(study.Series) != 1 {
		t.Fatalf("len(Series) = %d, want 1", len(study.Series))
	}
	if len(study.Series[0].Instance) != 2 {
		t.Errorf("len(Series[0].Instance) = %d, want 2", len(study.Series[0].Instance))
	}

	// The study identifier is the Study Instance UID as an Identifier, not a URL.
	if len(study.Identifier) != 1 || study.Identifier[0].System == nil ||
		*study.Identifier[0].System != "urn:dicom:uid" {
		t.Errorf("Identifier = %+v, want one urn:dicom:uid identifier", study.Identifier)
	}

	// The identity rule: no subject was supplied and no PatientID is present, so
	// the subject is left unset and never fabricated as a reference URL.
	if study.Subject != nil {
		t.Errorf("Subject = %+v, want nil (no PatientID, no WithSubjectR5)", study.Subject)
	}
	if !hasDefaultTarget(report, "ImagingStudy.subject") {
		t.Errorf("Report.Defaulted does not record the absent subject: %+v", report.Defaulted)
	}

	// status is defaulted to available and recorded.
	if study.Status != "available" {
		t.Errorf("Status = %q, want available", study.Status)
	}
	if !hasDefault(report, "ImagingStudy.status", "available") {
		t.Errorf("Report.Defaulted does not record the status default: %+v", report.Defaulted)
	}
}

// TestDICOMToImagingStudyR5MultiSeries recomputes counts across two series.
func TestDICOMToImagingStudyR5MultiSeries(t *testing.T) {
	insts := []*dicom.DataSet{
		instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR"),
		instance("1.2.3", "1.2.3.2", "1.2.3.2.1", "CT"),
		instance("1.2.3", "1.2.3.2", "1.2.3.2.2", "CT"),
	}

	study, _, err := DICOMToImagingStudyR5(insts)
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if study.NumberOfSeries == nil || *study.NumberOfSeries != 2 {
		t.Errorf("NumberOfSeries = %v, want 2", study.NumberOfSeries)
	}
	if study.NumberOfInstances == nil || *study.NumberOfInstances != 3 {
		t.Errorf("NumberOfInstances = %v, want 3", study.NumberOfInstances)
	}
	// The study-level modality is the union {MR, CT}.
	if len(study.Modality) != 2 {
		t.Errorf("len(Modality) = %d, want 2 (union of MR and CT)", len(study.Modality))
	}
}

// TestDICOMToImagingStudyR5DeDuplicatesInstances drops a repeated SOP Instance UID.
func TestDICOMToImagingStudyR5DeDuplicatesInstances(t *testing.T) {
	insts := []*dicom.DataSet{
		instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR"),
		instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR"), // duplicate
	}
	study, _, err := DICOMToImagingStudyR5(insts)
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if study.NumberOfInstances == nil || *study.NumberOfInstances != 1 {
		t.Errorf("NumberOfInstances = %v, want 1 after de-duplication", study.NumberOfInstances)
	}
}

// TestDICOMToImagingStudyR5RepairsModalityFromLaterInstance confirms a series
// whose first instance lacks Modality takes it from a later instance, rather than
// emitting an empty required series.modality.
func TestDICOMToImagingStudyR5RepairsModalityFromLaterInstance(t *testing.T) {
	noModality := instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "")
	withModality := instance("1.2.3", "1.2.3.1", "1.2.3.1.2", "MR")

	study, _, err := DICOMToImagingStudyR5([]*dicom.DataSet{noModality, withModality})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if len(study.Series) != 1 {
		t.Fatalf("len(Series) = %d, want 1", len(study.Series))
	}
	if len(study.Series[0].Modality.Coding) == 0 || study.Series[0].Modality.Coding[0].Code == nil ||
		*study.Series[0].Modality.Coding[0].Code != "MR" {
		t.Errorf("series.modality = %+v, want repaired to MR", study.Series[0].Modality)
	}
}

// TestDICOMToImagingStudyR5DropsSeriesWithoutModality confirms a series with no
// modality on any instance is dropped (recorded), never emitting an invalid one.
func TestDICOMToImagingStudyR5DropsSeriesWithoutModality(t *testing.T) {
	good := instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR")
	noModality := instance("1.2.3", "1.2.3.2", "1.2.3.2.1", "")

	study, report, err := DICOMToImagingStudyR5([]*dicom.DataSet{good, noModality})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if study.NumberOfSeries == nil || *study.NumberOfSeries != 1 {
		t.Errorf("NumberOfSeries = %v, want 1 (the modality-less series is dropped)", study.NumberOfSeries)
	}
	if !hasDropped(report, "DICOM (0008,0060) Modality") {
		t.Errorf("Report.Dropped does not record the modality-less series: %+v", report.Dropped)
	}
}

// TestDICOMToImagingStudyR5MissingStudyUID fails closed with ErrMissingIdentifier.
func TestDICOMToImagingStudyR5MissingStudyUID(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.1")
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.3.1.1")

	if _, _, err := DICOMToImagingStudyR5([]*dicom.DataSet{ds}); err == nil {
		t.Fatal("DICOMToImagingStudyR5 with no Study Instance UID returned nil error, want fail-closed")
	}
}

// TestDICOMToImagingStudyR5FixtureSubject confirms the vendored fixture's
// PatientID is carried as a logical Reference.identifier, never a URL.
func TestDICOMToImagingStudyR5FixtureSubject(t *testing.T) {
	f, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "MR2_UNCI.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	study, _, err := DICOMToImagingStudyR5([]*dicom.DataSet{f.DataSet})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if study.Subject == nil {
		t.Fatal("Subject is nil; the fixture's PatientID should be carried logically")
	}
	if study.Subject.Reference != nil {
		t.Errorf("Subject.Reference = %q, want nil (identity rule: never a URL)", *study.Subject.Reference)
	}
	if study.Subject.Identifier == nil || study.Subject.Identifier.Value == nil ||
		*study.Subject.Identifier.Value != "5MR2" {
		t.Errorf("Subject.Identifier.Value = %v, want 5MR2 (the fixture PatientID)", study.Subject.Identifier)
	}
	// StudyDescription "SHOULDER" maps across.
	if study.Description == nil || *study.Description != "SHOULDER" {
		t.Errorf("Description = %v, want SHOULDER", study.Description)
	}
}

// TestDICOMToImagingStudyR5WithSubject confirms WithSubjectR5 overrides the
// logical identity.
func TestDICOMToImagingStudyR5WithSubject(t *testing.T) {
	want := "Patient/pat-9"
	study, _, err := DICOMToImagingStudyR5(
		[]*dicom.DataSet{instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR")},
		WithSubjectR5(r5.Reference{Reference: &want}),
	)
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if study.Subject == nil || study.Subject.Reference == nil || *study.Subject.Reference != want {
		t.Errorf("Subject.Reference = %v, want %q", study.Subject, want)
	}
}
