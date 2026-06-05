package convert

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// numItem builds a NUM measurement leaf.
func numItem(t *testing.T) dicom.ContentItem {
	t.Helper()
	value, err := dicom.ParseDecimal("12.5")
	if err != nil {
		t.Fatalf("parse decimal: %v", err)
	}
	return dicom.ContentItem{
		ValueType:        dicom.ValueTypeNum,
		Relationship:     dicom.RelationshipContains,
		ConceptName:      dicom.ConceptNameCode{CodeValue: "G-A437", CodingSchemeDesignator: "SRT", CodeMeaning: "Diameter"},
		MeasuredValue:    value,
		MeasurementUnits: dicom.ConceptNameCode{CodeValue: "mm", CodingSchemeDesignator: "UCUM", CodeMeaning: "millimeter"},
	}
}

func TestContentItemToObservationR5(t *testing.T) {
	tests := []struct {
		name string
		item dicom.ContentItem
		want func(t *testing.T, o *r5.Observation)
	}{
		{
			name: "NUM maps to valueQuantity with units",
			item: numItem(t),
			want: func(t *testing.T, o *r5.Observation) {
				if o.ValueQuantity == nil {
					t.Fatalf("ValueQuantity not set: %+v", o)
				}
				if o.ValueQuantity.Value == nil || o.ValueQuantity.Value.String() != "12.5" {
					t.Errorf("Quantity.Value = %v, want 12.5", o.ValueQuantity.Value)
				}
				if o.ValueQuantity.Code == nil || *o.ValueQuantity.Code != "mm" {
					t.Errorf("Quantity.Code = %v, want mm", o.ValueQuantity.Code)
				}
				if o.ValueQuantity.Unit == nil || *o.ValueQuantity.Unit != "millimeter" {
					t.Errorf("Quantity.Unit = %v, want millimeter", o.ValueQuantity.Unit)
				}
				if o.ValueQuantity.System == nil || *o.ValueQuantity.System != "http://unitsofmeasure.org" {
					t.Errorf("Quantity.System = %v, want UCUM system", o.ValueQuantity.System)
				}
			},
		},
		{
			name: "CODE maps to valueCodeableConcept",
			item: dicom.ContentItem{
				ValueType:    dicom.ValueTypeCode,
				Relationship: dicom.RelationshipContains,
				ConceptName:  dicom.ConceptNameCode{CodeValue: "121072", CodingSchemeDesignator: "DCM", CodeMeaning: "Diagnosis"},
				Code:         dicom.ConceptNameCode{CodeValue: "168537006", CodingSchemeDesignator: "SCT", CodeMeaning: "Normal"},
			},
			want: func(t *testing.T, o *r5.Observation) {
				if o.ValueCodeableConcept == nil || len(o.ValueCodeableConcept.Coding) != 1 {
					t.Fatalf("ValueCodeableConcept not set: %+v", o.ValueCodeableConcept)
				}
				c := o.ValueCodeableConcept.Coding[0]
				if c.Code == nil || *c.Code != "168537006" {
					t.Errorf("value Coding.Code = %v, want 168537006", c.Code)
				}
				if c.System == nil || *c.System != "http://snomed.info/sct" {
					t.Errorf("value Coding.System = %v, want SNOMED", c.System)
				}
			},
		},
		{
			name: "TEXT maps to valueString",
			item: dicom.ContentItem{
				ValueType:    dicom.ValueTypeText,
				Relationship: dicom.RelationshipContains,
				ConceptName:  dicom.ConceptNameCode{CodeValue: "121071", CodingSchemeDesignator: "DCM", CodeMeaning: "Finding"},
				Text:         "Solitary pulmonary nodule.",
			},
			want: func(t *testing.T, o *r5.Observation) {
				if o.ValueString == nil || string(*o.ValueString) != "Solitary pulmonary nodule." {
					t.Errorf("ValueString = %v, want the finding text", o.ValueString)
				}
			},
		},
		{
			name: "DATE maps to valueDateTime",
			item: dicom.ContentItem{
				ValueType:    dicom.ValueTypeDate,
				Relationship: dicom.RelationshipContains,
				ConceptName:  dicom.ConceptNameCode{CodeValue: "111060", CodingSchemeDesignator: "DCM", CodeMeaning: "Study Date"},
				DateTime:     mustDT(t, "20050530"),
			},
			want: func(t *testing.T, o *r5.Observation) {
				if o.ValueDateTime == nil || string(*o.ValueDateTime) != "2005-05-30" {
					t.Errorf("ValueDateTime = %v, want 2005-05-30", o.ValueDateTime)
				}
			},
		},
		{
			name: "DATETIME with offset maps to full valueDateTime",
			item: dicom.ContentItem{
				ValueType:    dicom.ValueTypeDateTime,
				Relationship: dicom.RelationshipContains,
				ConceptName:  dicom.ConceptNameCode{CodeValue: "111526", CodingSchemeDesignator: "DCM", CodeMeaning: "DateTime Started"},
				DateTime:     mustDT(t, "20050530080000-0500"),
			},
			want: func(t *testing.T, o *r5.Observation) {
				if o.ValueDateTime == nil || string(*o.ValueDateTime) != "2005-05-30T08:00:00-05:00" {
					t.Errorf("ValueDateTime = %v, want 2005-05-30T08:00:00-05:00", o.ValueDateTime)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, ok := ContentItemToObservationR5(tt.item)
			if !ok {
				t.Fatalf("ContentItemToObservationR5 returned ok=false for %s", tt.name)
			}
			if o == nil {
				t.Fatal("Observation is nil")
			}
			// Every leaf carries its concept name as the required Observation.code and a
			// status, so the resource is structurally conformant.
			if o.Status == nil {
				t.Error("Observation.status is nil")
			}
			if o.Code == nil || len(o.Code.Coding) == 0 {
				t.Errorf("Observation.code not populated: %+v", o.Code)
			}
			tt.want(t, o)
		})
	}
}

// TestContentItemToObservationR5SkipsStructure confirms a CONTAINER and other
// non-leaf value types are not observations.
func TestContentItemToObservationR5SkipsStructure(t *testing.T) {
	for _, vt := range []dicom.ValueType{
		dicom.ValueTypeContainer,
		dicom.ValueTypeSCoord,
		dicom.ValueTypeSCoord3D,
		dicom.ValueTypeImage,
	} {
		item := dicom.ContentItem{ValueType: vt}
		if o, ok := ContentItemToObservationR5(item); ok || o != nil {
			t.Errorf("ContentItemToObservationR5(%v) = (%v, %v), want (nil, false)", vt, o, ok)
		}
	}
}

// mustDT parses a DICOM DT lexical form or fails the test.
func mustDT(t *testing.T, s string) dicom.DT {
	t.Helper()
	dt, err := dicom.ParseDT(s)
	if err != nil {
		t.Fatalf("parse DT %q: %v", s, err)
	}
	return dt
}
