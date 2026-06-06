package convert

import (
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// codeSeqItem builds a single-item code-sequence dataset (the standard DICOM coded
// entry triplet CodeValue, CodingSchemeDesignator, CodeMeaning) for driving the
// procedure/reason code-sequence mappings.
func codeSeqItem(value, scheme, meaning string) *dicom.DataSet {
	item := dicom.NewDataSet()
	item.SetString(dicom.TagCodeValue, value)
	item.SetString(dicom.TagCodingSchemeDesignator, scheme)
	item.SetString(dicom.TagCodeMeaning, meaning)
	return item
}

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
	if study.Status == nil || *study.Status != r5.ImagingStudyStatusAvailable {
		t.Errorf("Status = %v, want available", study.Status)
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
	modality := study.Series[0].Modality
	if modality == nil || len(modality.Coding) == 0 || modality.Coding[0].Code == nil ||
		*modality.Coding[0].Code != "MR" {
		t.Errorf("series.modality = %+v, want repaired to MR", modality)
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

// TestDICOMToImagingStudyR5Referrer confirms ReferringPhysicianName becomes the
// referrer Reference carrying only the rendered display name — never a fabricated
// Reference URL (the identity rule).
func TestDICOMToImagingStudyR5Referrer(t *testing.T) {
	ds := instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR")
	ds.SetString(dicom.TagReferringPhysicianName, "Welby^Marcus^J^Dr^MD")

	study, _, err := DICOMToImagingStudyR5([]*dicom.DataSet{ds})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if study.Referrer == nil {
		t.Fatal("Referrer is nil; ReferringPhysicianName should map to referrer")
	}
	if study.Referrer.Reference != nil {
		t.Errorf("Referrer.Reference = %q, want nil (identity rule: never a URL)", *study.Referrer.Reference)
	}
	if study.Referrer.Display == nil || *study.Referrer.Display != "Welby^Marcus^J^Dr^MD" {
		t.Errorf("Referrer.Display = %v, want the rendered PN", study.Referrer.Display)
	}
}

// TestDICOMToImagingStudyR5SeriesLateralityAndStarted confirms the series-level
// Laterality and SeriesDate/Time map to series.laterality and series.started.
func TestDICOMToImagingStudyR5SeriesLateralityAndStarted(t *testing.T) {
	ds := instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR")
	ds.SetString(dicom.TagLaterality, "L")
	ds.SetString(dicom.TagSeriesDate, "19950501")
	ds.SetString(dicom.TagSeriesTime, "162253")
	ds.SetString(dicom.TagTimezoneOffsetFromUTC, "-0400")

	study, _, err := DICOMToImagingStudyR5([]*dicom.DataSet{ds})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if len(study.Series) != 1 {
		t.Fatalf("len(Series) = %d, want 1", len(study.Series))
	}
	s := study.Series[0]
	if s.Laterality == nil || len(s.Laterality.Coding) == 0 || s.Laterality.Coding[0].Code == nil ||
		*s.Laterality.Coding[0].Code != "L" {
		t.Errorf("series.laterality = %+v, want code L", s.Laterality)
	}
	if s.Started == nil || *s.Started != "1995-05-01T16:22:53-04:00" {
		t.Errorf("series.started = %v, want 1995-05-01T16:22:53-04:00", s.Started)
	}
}

// TestDICOMToImagingStudyR5ProcedureAndReason confirms ProcedureCodeSequence and
// ReasonForRequestedProcedureCodeSequence map to procedure and reason as
// CodeableReference values carrying the coded entry.
func TestDICOMToImagingStudyR5ProcedureAndReason(t *testing.T) {
	ds := instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "MR")
	ds.Set(dicom.Element{
		Tag:   dicom.TagProcedureCodeSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(codeSeqItem("XR-CHEST", "DCM", "Chest X-ray"))),
	})
	ds.Set(dicom.Element{
		Tag:   dicom.TagReasonForRequestedProcedureCodeSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(codeSeqItem("R07.9", "SCT", "Chest pain"))),
	})

	study, _, err := DICOMToImagingStudyR5([]*dicom.DataSet{ds})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	if len(study.Procedure) != 1 || study.Procedure[0].Concept == nil ||
		len(study.Procedure[0].Concept.Coding) == 0 || study.Procedure[0].Concept.Coding[0].Code == nil ||
		*study.Procedure[0].Concept.Coding[0].Code != "XR-CHEST" {
		t.Errorf("Procedure = %+v, want one concept coded XR-CHEST", study.Procedure)
	}
	if len(study.Reason) != 1 || study.Reason[0].Concept == nil ||
		len(study.Reason[0].Concept.Coding) == 0 || study.Reason[0].Concept.Coding[0].Code == nil ||
		*study.Reason[0].Concept.Coding[0].Code != "R07.9" {
		t.Errorf("Reason = %+v, want one concept coded R07.9", study.Reason)
	}
	// The SCT designator resolves to the SNOMED system URI.
	if sys := study.Reason[0].Concept.Coding[0].System; sys == nil || *sys != "http://snomed.info/sct" {
		t.Errorf("Reason coding system = %v, want SNOMED URI", sys)
	}
}

// TestDICOMToImagingStudyR5StudyStartedFixture confirms the vendored fixture's
// StudyDate/Time with its dataset timezone offset combine into a full FHIR
// dateTime, and the produced resource validates by construction.
func TestDICOMToImagingStudyR5StudyStartedFixture(t *testing.T) {
	f, err := dicom.ReadFile(filepath.Join("..", "testdata", "dicom", "MR2_UNCI.dcm"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	study, _, err := DICOMToImagingStudyR5([]*dicom.DataSet{f.DataSet})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR5: %v", err)
	}
	// StudyDate 20040826 + StudyTime 185059 + TZ -0400 -> full dateTime.
	if study.Started == nil || *study.Started != "2004-08-26T18:50:59-04:00" {
		t.Errorf("study.started = %v, want 2004-08-26T18:50:59-04:00", study.Started)
	}
	// SeriesDate 19950501 + SeriesTime 162253.7700 + TZ -0400 -> series.started.
	if len(study.Series) != 1 || study.Series[0].Started == nil ||
		*study.Series[0].Started != "1995-05-01T16:22:53.7700-04:00" {
		t.Errorf("series.started = %+v, want 1995-05-01T16:22:53.7700-04:00", study.Series)
	}
	if oo := fhir.Validate(study); oo.HasErrors() {
		t.Errorf("ImagingStudy fails validation: %+v", oo.Issue)
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
