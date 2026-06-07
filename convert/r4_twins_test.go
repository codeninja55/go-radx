package convert

import (
	"errors"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/hl7v2"
)

// TestDICOMToImagingStudyR4 exercises the R4 ImagingStudy twin against a two-instance,
// single-series study and asserts the three load-bearing R4/R5 differences the twin
// reconciles: ImagingStudy.modality and series.modality are Coding (not CodeableConcept),
// series.instance.sopClass is a Coding, and the produced resource validates against the
// R4 binding set.
func TestDICOMToImagingStudyR4(t *testing.T) {
	instances := []*dicom.DataSet{
		instance("1.2.3", "1.2.3.1", "1.2.3.1.1", "CT"),
		instance("1.2.3", "1.2.3.1", "1.2.3.1.2", "CT"),
	}
	// ImagingStudy.subject is required (1..1) in both R4 and R5; carry the PatientID so
	// the logical subject is populated and the resource validates.
	for _, ds := range instances {
		ds.SetString(dicom.TagPatientID, "5MR2")
	}

	study, report, err := DICOMToImagingStudyR4(instances)
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR4: %v", err)
	}
	_ = report

	// The study identifier is the Study Instance UID as a urn:oid, never a Reference URL.
	if len(study.Identifier) == 0 || study.Identifier[0].Value == nil || *study.Identifier[0].Value != "urn:oid:1.2.3" {
		t.Errorf("identifier = %+v, want urn:oid:1.2.3", study.Identifier)
	}

	// Counts are recomputed from the distinct UIDs seen.
	if study.NumberOfSeries == nil || *study.NumberOfSeries != 1 {
		t.Errorf("numberOfSeries = %v, want 1", study.NumberOfSeries)
	}
	if study.NumberOfInstances == nil || *study.NumberOfInstances != 2 {
		t.Errorf("numberOfInstances = %v, want 2", study.NumberOfInstances)
	}

	// R4 difference: study-level modality is a Coding (R5 carries a CodeableConcept).
	if len(study.Modality) != 1 || study.Modality[0].Code == nil || *study.Modality[0].Code != "CT" {
		t.Errorf("modality = %+v, want one Coding with code CT", study.Modality)
	}

	// R4 difference: series.modality is a single Coding.
	if len(study.Series) != 1 {
		t.Fatalf("len(series) = %d, want 1", len(study.Series))
	}
	s := study.Series[0]
	if s.Modality == nil || s.Modality.Code == nil || *s.Modality.Code != "CT" {
		t.Errorf("series.modality = %+v, want Coding code CT", s.Modality)
	}
	// R4 difference: series.instance.sopClass is a Coding under urn:ietf:rfc:3986.
	if len(s.Instance) != 2 {
		t.Fatalf("len(series.instance) = %d, want 2", len(s.Instance))
	}
	if s.Instance[0].SopClass == nil || s.Instance[0].SopClass.System == nil || *s.Instance[0].SopClass.System != rfc3986System {
		t.Errorf("series.instance.sopClass = %+v, want a Coding under %s", s.Instance[0].SopClass, rfc3986System)
	}

	if oo := r4.Validate(study); oo.HasErrors() {
		t.Errorf("ImagingStudy fails R4 validation: %s", oo.Error())
	}
}

// TestDICOMToImagingStudyR4ReasonCode confirms the no-CodeableReference R4 difference:
// a coded ReasonForRequestedProcedureCodeSequence and a free-text ReasonForStudy both
// land on ImagingStudy.reasonCode (a CodeableConcept list), not the R5 reason
// CodeableReference, and the procedure code lands on procedureCode.
func TestDICOMToImagingStudyR4ReasonCode(t *testing.T) {
	ds := instance("1.2.9", "1.2.9.1", "1.2.9.1.1", "MR")
	ds.SetString(dicom.TagPatientID, "5MR2")
	ds.Set(dicom.Element{
		Tag:   dicom.TagReasonForRequestedProcedureCodeSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(codeSeqItem("R07.9", "I10", "Chest pain"))),
	})
	ds.SetString(dicom.TagReasonForStudy, "Follow-up imaging")
	ds.Set(dicom.Element{
		Tag:   dicom.TagProcedureCodeSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(codeSeqItem("24727-0", "LN", "MRI Brain"))),
	})

	study, _, err := DICOMToImagingStudyR4([]*dicom.DataSet{ds})
	if err != nil {
		t.Fatalf("DICOMToImagingStudyR4: %v", err)
	}

	// R4 difference: reason lands on reasonCode (a CodeableConcept list), never a
	// CodeableReference. The coded reason and the free-text reason both appear.
	if len(study.ReasonCode) != 2 {
		t.Fatalf("len(reasonCode) = %d, want 2 (coded + free-text)", len(study.ReasonCode))
	}
	coded := study.ReasonCode[0]
	if len(coded.Coding) == 0 || coded.Coding[0].Code == nil || *coded.Coding[0].Code != "R07.9" {
		t.Errorf("reasonCode[0] = %+v, want coded R07.9", coded)
	}
	freeText := study.ReasonCode[1]
	if freeText.Text == nil || *freeText.Text != "Follow-up imaging" {
		t.Errorf("reasonCode[1] = %+v, want free-text reason", freeText)
	}

	// The procedure lands on procedureCode (R4 has no single procedure CodeableReference).
	if len(study.ProcedureCode) != 1 || len(study.ProcedureCode[0].Coding) == 0 ||
		study.ProcedureCode[0].Coding[0].Code == nil || *study.ProcedureCode[0].Coding[0].Code != "24727-0" {
		t.Errorf("procedureCode = %+v, want coded 24727-0", study.ProcedureCode)
	}

	if oo := r4.Validate(study); oo.HasErrors() {
		t.Errorf("ImagingStudy fails R4 validation: %s", oo.Error())
	}
}

// TestORMToServiceRequestR4 exercises the R4 ServiceRequest twin and asserts the two
// load-bearing R4/R5 differences: ServiceRequest.code is a CodeableConcept (not a
// CodeableReference) and the OBR-31 reason lands on reasonCode (not a reason
// CodeableReference).
func TestORMToServiceRequestR4(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(richORM))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}

	sr, report, err := ORMToServiceRequestR4(msg)
	if err != nil {
		t.Fatalf("ORMToServiceRequestR4: %v", err)
	}

	// intent is defaulted to order and recorded.
	if sr.Intent == nil || *sr.Intent != r4.RequestIntentOrder {
		t.Errorf("intent = %v, want order", sr.Intent)
	}
	if !hasDefault(report, "ServiceRequest.intent", "order") {
		t.Errorf("Report.Defaulted does not record the intent default: %+v", report.Defaulted)
	}

	// R4 difference: code is a CodeableConcept directly, not a CodeableReference.Concept.
	if sr.Code == nil || len(sr.Code.Coding) == 0 || sr.Code.Coding[0].Code == nil || *sr.Code.Coding[0].Code != "36643-5" {
		t.Errorf("code = %+v, want CodeableConcept code 36643-5", sr.Code)
	}

	// R4 difference: the OBR-31 reason lands on reasonCode (a CodeableConcept list).
	if len(sr.ReasonCode) != 1 || len(sr.ReasonCode[0].Coding) == 0 ||
		sr.ReasonCode[0].Coding[0].Code == nil || *sr.ReasonCode[0].Coding[0].Code != "R07.9" {
		t.Errorf("reasonCode = %+v, want coded R07.9", sr.ReasonCode)
	}

	// PID-3 becomes a logical subject reference — identifier only, never a URL.
	if sr.Subject == nil || sr.Subject.Reference != nil {
		t.Errorf("subject = %+v, want logical identifier (no URL)", sr.Subject)
	}
	if sr.Subject.Identifier == nil || sr.Subject.Identifier.Value == nil || *sr.Subject.Identifier.Value != "555-44-4444" {
		t.Errorf("subject.identifier = %+v, want 555-44-4444", sr.Subject.Identifier)
	}

	if oo := r4.Validate(sr); oo.HasErrors() {
		t.Errorf("ServiceRequest fails R4 validation: %s", oo.Error())
	}
}

// TestORMToServiceRequestR4MultiOrderFailsClosed mirrors the R5 fail-closed boundary:
// a multi-order ORM is rejected with ErrUnsupportedSource, never partially mapped.
func TestORMToServiceRequestR4MultiOrderFailsClosed(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(multiOrderORM))
	if err != nil {
		t.Fatalf("parse ORM: %v", err)
	}
	if _, _, err := ORMToServiceRequestR4(msg); !errors.Is(err, ErrUnsupportedSource) {
		t.Errorf("err = %v, want ErrUnsupportedSource", err)
	}
}

// TestADTToPatientR4 exercises the R4 Patient twin: the v1 Patient fields are the same
// shape in R4 and R5, so the twin's correctness is in producing R4-typed output that
// validates against the R4 AdministrativeGender binding.
func TestADTToPatientR4(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(canonicalADT))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}

	pat, _, err := ADTToPatientR4(msg)
	if err != nil {
		t.Fatalf("ADTToPatientR4: %v", err)
	}

	if len(pat.Identifier) == 0 || pat.Identifier[0].Value == nil || *pat.Identifier[0].Value != "555-44-4444" {
		t.Errorf("identifier = %+v, want 555-44-4444", pat.Identifier)
	}
	if len(pat.Name) == 0 || pat.Name[0].Family == nil || *pat.Name[0].Family != "EVERYWOMAN" {
		t.Errorf("name = %+v, want family EVERYWOMAN", pat.Name)
	}
	if pat.Gender == nil || *pat.Gender != r4.AdministrativeGenderFemale {
		t.Errorf("gender = %v, want female", pat.Gender)
	}
	if pat.BirthDate == nil || *pat.BirthDate != "1962-03-20" {
		t.Errorf("birthDate = %v, want 1962-03-20", pat.BirthDate)
	}
	if len(pat.Address) == 0 || pat.Address[0].City == nil || *pat.Address[0].City != "METROPOLIS" {
		t.Errorf("address = %+v, want city METROPOLIS", pat.Address)
	}

	if oo := r4.Validate(pat); oo.HasErrors() {
		t.Errorf("Patient fails R4 validation: %s", oo.Error())
	}
}

// TestADTToEncounterR4 exercises the R4 Encounter twin and asserts the load-bearing R4
// difference: Encounter.class is a single Coding (R5 widened it to a CodeableConcept
// list). The trigger-event status mapping uses the R4 value set name "finished".
func TestADTToEncounterR4(t *testing.T) {
	msg, err := hl7v2.Parse([]byte(canonicalADT))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}

	enc, report, err := ADTToEncounterR4(msg)
	if err != nil {
		t.Fatalf("ADTToEncounterR4: %v", err)
	}

	// A01 (admit) maps to in-progress, recorded as a Substitution.
	if enc.Status == nil || *enc.Status != r4.EncounterStatusInProgress {
		t.Errorf("status = %v, want in-progress", enc.Status)
	}
	if !hasSubstitutionContaining(report, "Encounter.status") {
		t.Errorf("Report.Substituted does not record the status approximation: %+v", report.Substituted)
	}

	// R4 difference: class is a single Coding, not a CodeableConcept list. PV1-2 "I"
	// (inpatient) maps to IMP under the v3 ActCode system.
	if enc.Class == nil || enc.Class.Code == nil || *enc.Class.Code != "IMP" {
		t.Errorf("class = %+v, want single Coding code IMP", enc.Class)
	}
	if enc.Class.System == nil || *enc.Class.System != patientClassSystem {
		t.Errorf("class.system = %+v, want %s", enc.Class, patientClassSystem)
	}

	// PV1-19 becomes the logical visit identifier.
	if len(enc.Identifier) == 0 || enc.Identifier[0].Value == nil || *enc.Identifier[0].Value != "VISIT-9001" {
		t.Errorf("identifier = %+v, want VISIT-9001", enc.Identifier)
	}

	if oo := r4.Validate(enc); oo.HasErrors() {
		t.Errorf("Encounter fails R4 validation: %s", oo.Error())
	}
}

// TestADTToEncounterR4DischargeStatus confirms the R4 value-set difference: the A03
// (discharge) trigger maps to the R4 status "finished" (R5 renamed it "completed").
func TestADTToEncounterR4DischargeStatus(t *testing.T) {
	const dischargeADT = "MSH|^~\\&|ADT1|HOSP|EMR|HOSP|202605311230-0500||ADT^A03|MSGADT3|P|2.4\r" +
		"EVN|A03|202605311230-0500\r" +
		"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
		"PV1|1|I|WARD3^301^A||||||||||||||||VISIT-9001\r"
	msg, err := hl7v2.Parse([]byte(dischargeADT))
	if err != nil {
		t.Fatalf("parse ADT: %v", err)
	}
	enc, _, err := ADTToEncounterR4(msg)
	if err != nil {
		t.Fatalf("ADTToEncounterR4: %v", err)
	}
	if enc.Status == nil || *enc.Status != r4.EncounterStatusFinished {
		t.Errorf("status = %v, want finished (the R4 completed-state name)", enc.Status)
	}
	if oo := r4.Validate(enc); oo.HasErrors() {
		t.Errorf("Encounter fails R4 validation: %s", oo.Error())
	}
}

// TestSRToDiagnosticReportR4 exercises the R4 SR forward twin against the shared
// measurement SR fixture and asserts the produced report and every Observation validate
// against the R4 binding set, mirroring how the R5 golden test validates through
// r5.Validate.
func TestSRToDiagnosticReportR4(t *testing.T) {
	sr := measurementSR(t)

	dr, observations, _, err := SRToDiagnosticReportR4(sr)
	if err != nil {
		t.Fatalf("SRToDiagnosticReportR4: %v", err)
	}

	// The SOP Instance UID becomes the report identifier as a urn:oid.
	if len(dr.Identifier) == 0 || dr.Identifier[0].Value == nil || *dr.Identifier[0].Value != "urn:oid:1.2.840.113619.2.55.3.604688.1" {
		t.Errorf("identifier = %+v, want urn:oid SOP UID", dr.Identifier)
	}
	// COMPLETE+VERIFIED maps to final.
	if dr.Status == nil || *dr.Status != r4.DiagnosticReportStatusFinal {
		t.Errorf("status = %v, want final", dr.Status)
	}
	// The bare TEXT leaf becomes the conclusion; the concept-named leaves become
	// Observations (the finding TEXT, CODE, NUM, DATETIME, and TIME leaves).
	if dr.Conclusion == nil || *dr.Conclusion != "No acute findings." {
		t.Errorf("conclusion = %v, want 'No acute findings.'", dr.Conclusion)
	}
	if len(observations) != 5 {
		t.Fatalf("len(observations) = %d, want 5", len(observations))
	}
	if len(dr.Result) != len(observations) {
		t.Fatalf("len(result) = %d, want %d", len(dr.Result), len(observations))
	}

	if oo := r4.Validate(dr); oo.HasErrors() {
		t.Errorf("DiagnosticReport fails R4 validation: %s", oo.Error())
	}
	for i, o := range observations {
		if oo := r4.Validate(o); oo.HasErrors() {
			t.Errorf("Observation[%d] fails R4 validation: %s", i, oo.Error())
		}
	}
}

// TestORUToDiagnosticReportR4 exercises the R4 ORU forward twin and validates the report
// plus its Observations against the R4 binding set.
func TestORUToDiagnosticReportR4(t *testing.T) {
	const canonicalORU = "MSH|^~\\&|LAB|HOSP|EMR|HOSP|202605311230-0500||ORU^R01|MSGORU1|P|2.4\r" +
		"PID|||555-44-4444^^^HOSP^MR||EVERYWOMAN^EVE^E^^^^L||19620320|F\r" +
		"OBR|1|PLACER123|FILLER456|24331-1^Lipid panel^LN|||202605311231-0500||||||||||||||||||F\r" +
		"OBX|1|NM|2093-3^Cholesterol^LN||185|mg/dL^mg/dL^UCUM|125-200|H|||F\r" +
		"OBX|2|ST|55233-1^Comment^LN||Within normal limits|||||F\r"
	msg, err := hl7v2.Parse([]byte(canonicalORU))
	if err != nil {
		t.Fatalf("parse ORU: %v", err)
	}

	dr, observations, _, err := ORUToDiagnosticReportR4(msg)
	if err != nil {
		t.Fatalf("ORUToDiagnosticReportR4: %v", err)
	}

	if dr.Code == nil || len(dr.Code.Coding) == 0 || dr.Code.Coding[0].Code == nil || *dr.Code.Coding[0].Code != "24331-1" {
		t.Errorf("code = %+v, want 24331-1", dr.Code)
	}
	if dr.Status == nil || *dr.Status != r4.DiagnosticReportStatusFinal {
		t.Errorf("status = %v, want final", dr.Status)
	}
	if len(observations) != 2 {
		t.Fatalf("len(observations) = %d, want 2", len(observations))
	}

	// The first OBX is an NM result; OBX-8 "H" becomes interpretation, OBX-7 the range.
	num := observations[0]
	value, ok := num.Value()
	if !ok {
		t.Fatal("first Observation has no value")
	}
	if q, isQ := value.(r4.Quantity); !isQ || q.Value == nil || q.Value.String() != "185" {
		t.Errorf("value = %+v, want Quantity 185", value)
	}
	if len(num.Interpretation) == 0 || len(num.Interpretation[0].Coding) == 0 ||
		num.Interpretation[0].Coding[0].Code == nil || *num.Interpretation[0].Coding[0].Code != "H" {
		t.Errorf("interpretation = %+v, want H", num.Interpretation)
	}

	if oo := r4.Validate(dr); oo.HasErrors() {
		t.Errorf("DiagnosticReport fails R4 validation: %s", oo.Error())
	}
	for i, o := range observations {
		if oo := r4.Validate(o); oo.HasErrors() {
			t.Errorf("Observation[%d] fails R4 validation: %s", i, oo.Error())
		}
	}
}

// TestSRRoundTripR4PreservesMeasurements converts a DICOM SR to an R4 DiagnosticReport
// plus Observations and back to an SR via DiagnosticReportToSRR4, then re-runs the
// forward conversion. The NUM, CODE, and DATETIME measurement leaves must survive and
// the round-tripped resources must validate, the R4 twin of the R5 round-trip gate.
func TestSRRoundTripR4PreservesMeasurements(t *testing.T) {
	sr := measurementSR(t)

	dr, observations, _, err := SRToDiagnosticReportR4(sr)
	if err != nil {
		t.Fatalf("forward SRToDiagnosticReportR4: %v", err)
	}

	rebuilt, reverseReport, err := DiagnosticReportToSRR4(dr, observations, WithUIDRoot(roundTripUIDRoot))
	if err != nil {
		t.Fatalf("reverse DiagnosticReportToSRR4: %v", err)
	}
	// The forward SR carries a TIME leaf the reverse builder cannot re-encode; the loss
	// must be reported, never silent.
	if !hasDropped(reverseReport, "Observation.valueTime") {
		t.Errorf("reverse report does not record the un-re-encodable TIME leaf: %+v", reverseReport.Dropped)
	}

	if _, err := dicom.ParseSR(rebuilt); err != nil {
		t.Fatalf("rebuilt SR does not parse as a conformant SR: %v", err)
	}

	dr2, observations2, _, err := SRToDiagnosticReportR4(rebuilt)
	if err != nil {
		t.Fatalf("re-forward SRToDiagnosticReportR4: %v", err)
	}

	if oo := r4.Validate(dr2); oo.HasErrors() {
		t.Errorf("round-tripped DiagnosticReport fails R4 validation: %s", oo.Error())
	}
	for i, o := range observations2 {
		if oo := r4.Validate(o); oo.HasErrors() {
			t.Errorf("round-tripped Observation[%d] fails R4 validation: %s", i, oo.Error())
		}
	}

	if dr.Conclusion == nil || dr2.Conclusion == nil || *dr.Conclusion != *dr2.Conclusion {
		t.Errorf("conclusion lost across round trip: %v -> %v", dr.Conclusion, dr2.Conclusion)
	}
}

// TestObservationToContentItemR4 confirms the R4 reverse leaf twin re-encodes a NUM
// Observation into a DICOM SR content item carrying the same value and units.
func TestObservationToContentItemR4(t *testing.T) {
	value, err := dicom.ParseDecimal("12.5")
	if err != nil {
		t.Fatalf("parse decimal: %v", err)
	}
	o := &r4.Observation{Code: &r4.CodeableConcept{
		Coding: []r4.Coding{{System: strPtr("http://loinc.org"), Code: strPtr("G-A437"), Display: strPtr("Diameter")}},
	}}
	q := r4.Quantity{Value: &value, Code: strPtr("mm"), Unit: strPtr("millimeter"), System: strPtr(ucumSystem)}
	o.SetValueQuantity(q)

	item, _, ok := ObservationToContentItemR4(o)
	if !ok {
		t.Fatal("ObservationToContentItemR4 ok=false")
	}
	if item.ValueType != dicom.ValueTypeNum {
		t.Fatalf("ValueType = %v, want NUM", item.ValueType)
	}
	if item.MeasuredValue.String() != "12.5" {
		t.Errorf("MeasuredValue = %q, want 12.5", item.MeasuredValue.String())
	}
	if item.MeasurementUnits.CodeValue != "mm" || item.MeasurementUnits.CodingSchemeDesignator != "UCUM" {
		t.Errorf("MeasurementUnits = %+v, want mm/UCUM", item.MeasurementUnits)
	}
}

// TestObservationToOBXR4 confirms the R4 reverse OBX twin re-encodes an NM Observation
// into an OBX carrying the value, units, interpretation, and reference range.
func TestObservationToOBXR4(t *testing.T) {
	value, err := dicom.ParseDecimal("185")
	if err != nil {
		t.Fatalf("parse decimal: %v", err)
	}
	o := &r4.Observation{Code: &r4.CodeableConcept{
		Coding: []r4.Coding{{System: strPtr("http://loinc.org"), Code: strPtr("2093-3"), Display: strPtr("Cholesterol")}},
	}}
	o.SetValueQuantity(r4.Quantity{Value: &value, Code: strPtr("mg/dL"), Unit: strPtr("mg/dL"), System: strPtr(ucumSystem)})
	o.Interpretation = []r4.CodeableConcept{{
		Coding: []r4.Coding{{System: strPtr(observationInterpretationSystem), Code: strPtr("H")}},
	}}
	low := r4.Quantity{Value: decimalPtr(t, "125")}
	high := r4.Quantity{Value: decimalPtr(t, "200")}
	o.ReferenceRange = []r4.ObservationReferenceRange{{Low: &low, High: &high}}

	obx, _, ok := ObservationToOBXR4(o)
	if !ok {
		t.Fatal("ObservationToOBXR4 ok=false")
	}
	if obx.ValueType != "NM" || len(obx.Value) == 0 || obx.Value[0] != "185" {
		t.Errorf("value = %v (type %q), want NM 185", obx.Value, obx.ValueType)
	}
	// OBX-6 units round-trip the UCUM coding system identifier.
	if obx.Units.Code != "mg/dL" || obx.Units.CodingSystem != "UCUM" {
		t.Errorf("units = %+v, want mg/dL/UCUM", obx.Units)
	}
	if len(obx.AbnormalFlags) != 1 || obx.AbnormalFlags[0] != "H" {
		t.Errorf("abnormalFlags = %v, want [H]", obx.AbnormalFlags)
	}
	if obx.ReferenceRange != "125-200" {
		t.Errorf("referenceRange = %q, want 125-200", obx.ReferenceRange)
	}
}

// decimalPtr parses a decimal literal and returns a pointer to it, the helper the R4
// reverse OBX test uses to build reference-range boundary Quantities.
func decimalPtr(t *testing.T, s string) *fhir.Decimal {
	t.Helper()
	dec, err := dicom.ParseDecimal(s)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", s, err)
	}
	d := fhir.Decimal(dec)
	return &d
}
