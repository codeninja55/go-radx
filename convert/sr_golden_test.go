package convert

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// updateGolden regenerates the golden files instead of asserting against them.
var updateGolden = flag.Bool("update", false, "regenerate golden test files")

// measurementSR builds an in-memory Comprehensive SR document with a concept-named TEXT
// finding (a string-valued Observation), a coded CODE item, a NUM measurement, a
// DATETIME with no inline offset, a TIME item, and a bare (un-coded) TEXT conclusion,
// plus the document-level identity, status, modality, and timezone offset a conformant
// SR carries. The concept-named TEXT exercises the string-Observation classification and
// the bare TEXT exercises the narrative conclusion; the DATETIME exercises the
// document-level TimezoneOffsetFromUTC fallback and the TIME exercises valueTime. It is
// the fixture for the SR -> DiagnosticReport + Observations golden conversion.
func measurementSR(t *testing.T) *dicom.DataSet {
	t.Helper()
	value, err := dicom.ParseDecimal("12.5")
	if err != nil {
		t.Fatalf("parse decimal: %v", err)
	}
	root := &dicom.ContentItem{
		ValueType:   dicom.ValueTypeContainer,
		ConceptName: dicom.ConceptNameCode{CodeValue: "11528-7", CodingSchemeDesignator: "LN", CodeMeaning: "Radiology Report"},
		Children: []dicom.ContentItem{
			{
				ValueType:    dicom.ValueTypeText,
				Relationship: dicom.RelationshipContains,
				ConceptName:  dicom.ConceptNameCode{CodeValue: "121071", CodingSchemeDesignator: "DCM", CodeMeaning: "Finding"},
				Text:         "Solitary pulmonary nodule.",
			},
			{
				ValueType:    dicom.ValueTypeCode,
				Relationship: dicom.RelationshipContains,
				ConceptName:  dicom.ConceptNameCode{CodeValue: "121072", CodingSchemeDesignator: "DCM", CodeMeaning: "Diagnosis"},
				Code:         dicom.ConceptNameCode{CodeValue: "168537006", CodingSchemeDesignator: "SCT", CodeMeaning: "Normal"},
			},
			{
				ValueType:        dicom.ValueTypeNum,
				Relationship:     dicom.RelationshipContains,
				ConceptName:      dicom.ConceptNameCode{CodeValue: "G-A437", CodingSchemeDesignator: "SRT", CodeMeaning: "Diameter"},
				MeasuredValue:    value,
				MeasurementUnits: dicom.ConceptNameCode{CodeValue: "mm", CodingSchemeDesignator: "UCUM", CodeMeaning: "millimeter"},
			},
			{
				ValueType:    dicom.ValueTypeDateTime,
				Relationship: dicom.RelationshipContains,
				ConceptName:  dicom.ConceptNameCode{CodeValue: "111526", CodingSchemeDesignator: "DCM", CodeMeaning: "DateTime Started"},
				DateTime:     mustDT(t, "20050530081530"),
			},
			{
				// A bare (un-coded) TEXT leaf is the document narrative, mapped to
				// DiagnosticReport.conclusion rather than a string Observation.
				ValueType:    dicom.ValueTypeText,
				Relationship: dicom.RelationshipContains,
				Text:         "No acute findings.",
			},
		},
	}

	ds := dicom.NewDataSet()
	if err := dicom.BuildSR(ds, root); err != nil {
		t.Fatalf("BuildSR: %v", err)
	}
	appendTimeContentItem(t, ds, "081545")
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.88.33") // Comprehensive SR
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.840.113619.2.55.3.604688.1")
	ds.SetString(dicom.TagModality, "SR")
	ds.SetString(dicom.TagContentDate, "20050530")
	ds.SetString(dicom.TagContentTime, "080000")
	ds.SetString(dicom.TagTimezoneOffsetFromUTC, "-0500")
	ds.SetString(dicom.TagCompletionFlag, "COMPLETE")
	ds.SetString(dicom.TagVerificationFlag, "VERIFIED")
	return ds
}

// appendTimeContentItem appends a TIME content item carrying the lexical TM value to
// the root Content Sequence. A TIME value is VR TM, not a constructible DT, so it is
// written directly to the (0040,A730) sequence rather than through the ContentItem
// tree, matching how a real SR carries a (0040,A122) Time leaf.
func appendTimeContentItem(t *testing.T, ds *dicom.DataSet, lexical string) {
	t.Helper()
	seq, ok := ds.GetSequence(dicom.TagContentSequence)
	if !ok {
		t.Fatal("SR has no Content Sequence to append the TIME item to")
	}
	child := dicom.NewDataSet()
	child.SetString(dicom.TagValueType, "TIME")
	child.SetString(dicom.TagRelationshipType, "CONTAINS")
	child.SetString(dicom.TagTime, lexical)
	concept := dicom.NewDataSet()
	concept.SetString(dicom.TagCodeValue, "111526")
	concept.SetString(dicom.TagCodingSchemeDesignator, "DCM")
	concept.SetString(dicom.TagCodeMeaning, "Time")
	child.Set(dicom.Element{Tag: dicom.TagConceptNameCodeSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence(concept))})
	seq.Append(child)
}

// goldenEnvelope is the deterministic marshalling envelope: the DiagnosticReport
// followed by its linked Observations, so the golden file captures both the report's
// result references and the resources they point at in one stable document.
type goldenEnvelope struct {
	DiagnosticReport *r5.DiagnosticReport `json:"diagnosticReport"`
	Observations     []*r5.Observation    `json:"observations"`
}

func TestSRToDiagnosticReportR5Golden(t *testing.T) {
	sr := measurementSR(t)

	dr, observations, report, err := SRToDiagnosticReportR5(sr)
	if err != nil {
		t.Fatalf("SRToDiagnosticReportR5: %v", err)
	}
	_ = report

	// Every produced resource must pass the in-process structural gate, which mirrors
	// the merge-blocking HL7 validator gate that runs over convert output in CI.
	if outcome := fhir.Validate(dr); outcome.HasErrors() {
		t.Errorf("DiagnosticReport is not structurally valid: %s", outcome.Error())
	}
	for i, o := range observations {
		if outcome := fhir.Validate(o); outcome.HasErrors() {
			t.Errorf("Observation[%d] is not structurally valid: %s", i, outcome.Error())
		}
	}

	got, err := json.MarshalIndent(goldenEnvelope{DiagnosticReport: dr, Observations: observations}, "", "  ")
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	got = append(got, '\n')

	goldenPath := filepath.Join("testdata", "sr_diagnosticreport.golden.json")
	if *updateGolden {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with -update to generate): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("conversion output does not match golden file %s\n--- got ---\n%s\n--- want ---\n%s", goldenPath, got, want)
	}
}

// TestSRToDiagnosticReportR5Deterministic confirms two conversions of the same input
// produce byte-identical output, so the urn:uuid result links never drift.
func TestSRToDiagnosticReportR5Deterministic(t *testing.T) {
	first := mustMarshalEnvelope(t, measurementSR(t))
	second := mustMarshalEnvelope(t, measurementSR(t))
	if !bytes.Equal(first, second) {
		t.Error("two conversions of the same SR produced different output; the urn:uuid links are not deterministic")
	}
}

func mustMarshalEnvelope(t *testing.T, sr *dicom.DataSet) []byte {
	t.Helper()
	dr, observations, _, err := SRToDiagnosticReportR5(sr)
	if err != nil {
		t.Fatalf("SRToDiagnosticReportR5: %v", err)
	}
	out, err := json.Marshal(goldenEnvelope{DiagnosticReport: dr, Observations: observations})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}

// TestSRToDiagnosticReportR5LinksObservations confirms the report.result references
// match the urn:uuid logical id of each produced Observation, one per measurement leaf.
func TestSRToDiagnosticReportR5LinksObservations(t *testing.T) {
	dr, observations, _, err := SRToDiagnosticReportR5(measurementSR(t))
	if err != nil {
		t.Fatalf("SRToDiagnosticReportR5: %v", err)
	}
	if len(observations) != 5 {
		t.Fatalf("len(observations) = %d, want 5 (concept-named TEXT finding, CODE, NUM, DATETIME, TIME leaves; the bare TEXT is the conclusion)", len(observations))
	}
	if len(dr.Result) != len(observations) {
		t.Fatalf("len(Result) = %d, want %d", len(dr.Result), len(observations))
	}
	for i, o := range observations {
		if o.ID == nil || *o.ID == "" {
			t.Fatalf("Observation[%d] has no id", i)
		}
		wantRef := "urn:uuid:" + *o.ID
		ref := dr.Result[i]
		if ref.Reference == nil || *ref.Reference != wantRef {
			t.Errorf("Result[%d].reference = %v, want %s", i, ref.Reference, wantRef)
		}
	}
}
