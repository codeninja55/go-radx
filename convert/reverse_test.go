package convert

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// strPtr returns a pointer to s, the helper the reverse tests use to build FHIR
// primitive value fields.
func strPtr(s string) *string { return &s }

func TestObservationToContentItem(t *testing.T) {
	value, err := dicom.ParseDecimal("12.5")
	if err != nil {
		t.Fatalf("parse decimal: %v", err)
	}

	tests := []struct {
		name  string
		build func() *r5.Observation
		want  func(t *testing.T, item dicom.ContentItem)
	}{
		{
			name: "valueQuantity maps to NUM with units",
			build: func() *r5.Observation {
				o := &r5.Observation{Code: numConcept()}
				q := r5.Quantity{Value: &value, Code: strPtr("mm"), Unit: strPtr("millimeter"), System: strPtr(ucumSystem)}
				o.SetValueQuantity(q)
				return o
			},
			want: func(t *testing.T, item dicom.ContentItem) {
				if item.ValueType != dicom.ValueTypeNum {
					t.Fatalf("ValueType = %v, want NUM", item.ValueType)
				}
				if item.MeasuredValue.String() != "12.5" {
					t.Errorf("MeasuredValue = %q, want 12.5", item.MeasuredValue.String())
				}
				if item.MeasurementUnits.CodeValue != "mm" || item.MeasurementUnits.CodingSchemeDesignator != "UCUM" {
					t.Errorf("MeasurementUnits = %+v, want mm/UCUM", item.MeasurementUnits)
				}
			},
		},
		{
			name: "valueCodeableConcept maps to CODE",
			build: func() *r5.Observation {
				o := &r5.Observation{Code: numConcept()}
				o.SetValueCodeableConcept(r5.CodeableConcept{
					Coding: []r5.Coding{{System: strPtr("http://snomed.info/sct"), Code: strPtr("168537006"), Display: strPtr("Normal")}},
				})
				return o
			},
			want: func(t *testing.T, item dicom.ContentItem) {
				if item.ValueType != dicom.ValueTypeCode {
					t.Fatalf("ValueType = %v, want CODE", item.ValueType)
				}
				if item.Code.CodeValue != "168537006" || item.Code.CodingSchemeDesignator != "SCT" {
					t.Errorf("Code = %+v, want 168537006/SCT", item.Code)
				}
			},
		},
		{
			name: "valueString maps to TEXT",
			build: func() *r5.Observation {
				o := &r5.Observation{Code: numConcept()}
				o.SetValueString(r5.FHIRString("Solitary pulmonary nodule."))
				return o
			},
			want: func(t *testing.T, item dicom.ContentItem) {
				if item.ValueType != dicom.ValueTypeText || item.Text != "Solitary pulmonary nodule." {
					t.Errorf("item = %+v, want TEXT with the finding text", item)
				}
			},
		},
		{
			name: "valueDateTime with offset maps to DATETIME",
			build: func() *r5.Observation {
				o := &r5.Observation{Code: numConcept()}
				o.SetValueDateTime(r5.FHIRDateTime("2005-05-30T08:00:00-05:00"))
				return o
			},
			want: func(t *testing.T, item dicom.ContentItem) {
				if item.ValueType != dicom.ValueTypeDateTime {
					t.Fatalf("ValueType = %v, want DATETIME", item.ValueType)
				}
				if item.DateTime.String() != "20050530080000-0500" {
					t.Errorf("DateTime = %q, want 20050530080000-0500", item.DateTime.String())
				}
			},
		},
		{
			name: "date-only valueDateTime maps to DATE",
			build: func() *r5.Observation {
				o := &r5.Observation{Code: numConcept()}
				o.SetValueDateTime(r5.FHIRDateTime("2005-05-30"))
				return o
			},
			want: func(t *testing.T, item dicom.ContentItem) {
				if item.ValueType != dicom.ValueTypeDate || item.DateTime.String() != "20050530" {
					t.Errorf("item = %+v, want DATE 20050530", item)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, _, ok := ObservationToContentItem(tt.build())
			if !ok {
				t.Fatalf("ObservationToContentItem ok=false for %s", tt.name)
			}
			if item.Relationship != dicom.RelationshipContains {
				t.Errorf("Relationship = %v, want CONTAINS", item.Relationship)
			}
			if item.ConceptName.IsZero() {
				t.Error("ConceptName is zero; the leaf has no Concept Name Code Sequence")
			}
			tt.want(t, item)
		})
	}
}

// TestObservationToContentItemNoCodeDropped confirms an Observation with no code is not
// re-encoded (an SR content item requires a Concept Name Code Sequence) and the loss is
// recorded by element path.
func TestObservationToContentItemNoCodeDropped(t *testing.T) {
	o := &r5.Observation{}
	o.SetValueString(r5.FHIRString("orphan"))
	item, report, ok := ObservationToContentItem(o)
	if ok {
		t.Fatalf("ObservationToContentItem ok=true, want false for a code-less Observation: %+v", item)
	}
	if !hasDropped(report, "Observation.code") {
		t.Errorf("Report.Dropped does not record the code-less Observation: %+v", report.Dropped)
	}
}

// TestObservationToContentItemTimeReported confirms a valueTime is reported as loss
// rather than emitted on a wrong-typed leaf, since the reverse content-item builder
// cannot carry a VR TM value from the convert package.
func TestObservationToContentItemTimeReported(t *testing.T) {
	o := &r5.Observation{Code: numConcept()}
	o.SetValueTime(r5.FHIRTime("08:00:00"))
	if _, report, ok := ObservationToContentItem(o); ok {
		t.Fatal("ObservationToContentItem ok=true, want false for a valueTime leaf")
	} else if !hasDropped(report, "Observation.valueTime") {
		t.Errorf("Report.Dropped does not record the valueTime: %+v", report.Dropped)
	}
}

func TestObservationToOBX(t *testing.T) {
	value, err := dicom.ParseDecimal("9.4")
	if err != nil {
		t.Fatalf("parse decimal: %v", err)
	}

	tests := []struct {
		name  string
		build func() *r5.Observation
		want  func(t *testing.T, obx hl7v2.OBX)
	}{
		{
			name: "valueQuantity maps to NM with units",
			build: func() *r5.Observation {
				o := &r5.Observation{Code: cweConceptValue("718-7", "Hemoglobin", "http://loinc.org")}
				o.SetValueQuantity(r5.Quantity{Value: &value, Code: strPtr("g/dL"), Unit: strPtr("g/dL"), System: strPtr(ucumSystem)})
				return o
			},
			want: func(t *testing.T, obx hl7v2.OBX) {
				if obx.ValueType != "NM" || len(obx.Value) != 1 || obx.Value[0] != "9.4" {
					t.Errorf("obx value = %v/%v, want NM/9.4", obx.ValueType, obx.Value)
				}
				if obx.Units.Code != "g/dL" || obx.Units.CodingSystem != "UCUM" {
					t.Errorf("OBX-6 units = %+v, want g/dL/UCUM", obx.Units)
				}
			},
		},
		{
			name: "valueString maps to ST",
			build: func() *r5.Observation {
				o := &r5.Observation{Code: cweConceptValue("NOTE", "Comment", "L")}
				o.SetValueString(r5.FHIRString("within normal limits"))
				return o
			},
			want: func(t *testing.T, obx hl7v2.OBX) {
				if obx.ValueType != "ST" || obx.Value[0] != "within normal limits" {
					t.Errorf("obx = %v/%v, want ST/text", obx.ValueType, obx.Value)
				}
			},
		},
		{
			name: "valueCodeableConcept maps to CWE",
			build: func() *r5.Observation {
				o := &r5.Observation{Code: cweConceptValue("X", "result", "L")}
				o.SetValueCodeableConcept(r5.CodeableConcept{
					Coding: []r5.Coding{{Code: strPtr("260385009"), Display: strPtr("Negative"), System: strPtr("SCT")}},
				})
				return o
			},
			want: func(t *testing.T, obx hl7v2.OBX) {
				if obx.ValueType != "CWE" || obx.Value[0] != "260385009^Negative^SCT" {
					t.Errorf("obx = %v/%v, want CWE/coded form", obx.ValueType, obx.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obx, _, ok := ObservationToOBX(tt.build())
			if !ok {
				t.Fatalf("ObservationToOBX ok=false for %s", tt.name)
			}
			if obx.ObservationID.Code == "" {
				t.Error("OBX-3 has no code")
			}
			if obx.ResultStatus != "F" {
				t.Errorf("OBX-11 = %q, want F", obx.ResultStatus)
			}
			tt.want(t, obx)
		})
	}
}

// TestObservationToOBXInterpretationAndRange confirms the interpretation flags and a
// numeric reference range round-trip back to OBX-8 and OBX-7.
func TestObservationToOBXInterpretationAndRange(t *testing.T) {
	low, _ := dicom.ParseDecimal("12")
	high, _ := dicom.ParseDecimal("16")
	o := &r5.Observation{Code: cweConceptValue("718-7", "Hemoglobin", "http://loinc.org")}
	val, _ := dicom.ParseDecimal("9.4")
	o.SetValueQuantity(r5.Quantity{Value: &val})
	o.Interpretation = []r5.CodeableConcept{
		{Coding: []r5.Coding{{System: strPtr(observationInterpretationSystem), Code: strPtr("L")}}},
	}
	o.ReferenceRange = []r5.ObservationReferenceRange{{Low: &r5.Quantity{Value: &low}, High: &r5.Quantity{Value: &high}}}

	obx, _, ok := ObservationToOBX(o)
	if !ok {
		t.Fatal("ObservationToOBX ok=false")
	}
	if len(obx.AbnormalFlags) != 1 || obx.AbnormalFlags[0] != "L" {
		t.Errorf("OBX-8 = %v, want [L]", obx.AbnormalFlags)
	}
	if obx.ReferenceRange != "12-16" {
		t.Errorf("OBX-7 = %q, want 12-16", obx.ReferenceRange)
	}
}

// numConcept builds the NUM concept name shared by the reverse SR tests.
func numConcept() *r5.CodeableConcept {
	return &r5.CodeableConcept{
		Coding: []r5.Coding{{System: strPtr("urn:dicom:scheme:SRT"), Code: strPtr("G-A437"), Display: strPtr("Diameter")}},
	}
}

// cweConceptValue builds a CodeableConcept carrying one Coding for the OBX reverse tests.
func cweConceptValue(code, display, system string) *r5.CodeableConcept {
	return &r5.CodeableConcept{
		Coding: []r5.Coding{{Code: strPtr(code), Display: strPtr(display), System: strPtr(system)}},
	}
}
