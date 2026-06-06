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

// srTimeContentItem builds a TIME content item carrying the lexical TM value. A TIME
// item's (0040,A122) value is a VR TM, which is not a valid DT, so the content-tree
// parser preserves it on the DT's lexical string. Round-tripping through the real
// BuildSR/ParseSR path reproduces exactly what the converter sees at runtime.
func srTimeContentItem(t *testing.T, lexical string) dicom.ContentItem {
	t.Helper()
	ds := dicom.NewDataSet()
	root := &dicom.ContentItem{
		ValueType:   dicom.ValueTypeContainer,
		ConceptName: dicom.ConceptNameCode{CodeValue: "11528-7", CodingSchemeDesignator: "LN", CodeMeaning: "Radiology Report"},
	}
	if err := dicom.BuildSR(ds, root); err != nil {
		t.Fatalf("BuildSR: %v", err)
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
	seq := dicom.NewSequence(child)
	ds.Set(dicom.Element{Tag: dicom.TagContentSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(seq)})
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.88.33")
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.3")

	parsed, err := dicom.ParseSR(ds)
	if err != nil {
		t.Fatalf("ParseSR: %v", err)
	}
	if len(parsed.Children) != 1 {
		t.Fatalf("expected one TIME child, got %d", len(parsed.Children))
	}
	return parsed.Children[0]
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
		{
			name: "DATETIME with inline offset preserves fractional seconds",
			item: dicom.ContentItem{
				ValueType:    dicom.ValueTypeDateTime,
				Relationship: dicom.RelationshipContains,
				ConceptName:  dicom.ConceptNameCode{CodeValue: "111526", CodingSchemeDesignator: "DCM", CodeMeaning: "DateTime Started"},
				DateTime:     mustDT(t, "20050530080000.123456-0500"),
			},
			want: func(t *testing.T, o *r5.Observation) {
				if o.ValueDateTime == nil || string(*o.ValueDateTime) != "2005-05-30T08:00:00.123456-05:00" {
					t.Errorf("ValueDateTime = %v, want 2005-05-30T08:00:00.123456-05:00", o.ValueDateTime)
				}
			},
		},
		{
			name: "TIME maps to valueTime",
			item: srTimeContentItem(t, "080000"),
			want: func(t *testing.T, o *r5.Observation) {
				if o.ValueTime == nil || string(*o.ValueTime) != "08:00:00" {
					t.Errorf("ValueTime = %v, want 08:00:00", o.ValueTime)
				}
			},
		},
		{
			name: "TIME preserves fractional seconds as valueTime",
			item: srTimeContentItem(t, "080000.123456"),
			want: func(t *testing.T, o *r5.Observation) {
				if o.ValueTime == nil || string(*o.ValueTime) != "08:00:00.123456" {
					t.Errorf("ValueTime = %v, want 08:00:00.123456", o.ValueTime)
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

// TestContentItemDATETIMEAppliesDatasetTimezone confirms a DATETIME leaf carrying a
// time but no inline offset borrows the dataset's document-level
// TimezoneOffsetFromUTC (0008,0201) to produce a valid FHIR dateTime, rather than
// degrading to date-only. The fractional second is preserved through the dataset-zone
// path too.
func TestContentItemDATETIMEAppliesDatasetTimezone(t *testing.T) {
	item := dicom.ContentItem{
		ValueType:    dicom.ValueTypeDateTime,
		Relationship: dicom.RelationshipContains,
		ConceptName:  dicom.ConceptNameCode{CodeValue: "111526", CodingSchemeDesignator: "DCM", CodeMeaning: "DateTime Started"},
		DateTime:     mustDT(t, "20050530080000.123456"),
	}

	o, ok := contentItemToObservationR5(item, "-05:00", true, &Report{})
	if !ok || o == nil {
		t.Fatalf("contentItemToObservationR5 ok=%v o=%v, want a tz-corrected Observation", ok, o)
	}
	if o.ValueDateTime == nil || string(*o.ValueDateTime) != "2005-05-30T08:00:00.123456-05:00" {
		t.Errorf("ValueDateTime = %v, want 2005-05-30T08:00:00.123456-05:00 (dataset timezone applied)", o.ValueDateTime)
	}

	// With no offset available at all, the time is dropped to date-only rather than
	// emitting a FHIR-invalid timezone-less time.
	degraded, ok := contentItemToObservationR5(item, "", false, &Report{})
	if !ok || degraded == nil {
		t.Fatalf("contentItemToObservationR5 ok=%v o=%v, want a date-only Observation", ok, degraded)
	}
	if degraded.ValueDateTime == nil || string(*degraded.ValueDateTime) != "2005-05-30" {
		t.Errorf("ValueDateTime = %v, want 2005-05-30 (no offset: degraded to date)", degraded.ValueDateTime)
	}
}

// TestContentItemDATETIMEInlineOffsetWinsOverDataset confirms an inline &ZZXX offset
// takes precedence over the dataset-level offset, so a content item that states its
// own zone is never overridden by the document default.
func TestContentItemDATETIMEInlineOffsetWinsOverDataset(t *testing.T) {
	item := dicom.ContentItem{
		ValueType:    dicom.ValueTypeDateTime,
		Relationship: dicom.RelationshipContains,
		ConceptName:  dicom.ConceptNameCode{CodeValue: "111526", CodingSchemeDesignator: "DCM", CodeMeaning: "DateTime Started"},
		DateTime:     mustDT(t, "20050530080000+0100"),
	}
	o, ok := contentItemToObservationR5(item, "-05:00", true, &Report{})
	if !ok || o == nil {
		t.Fatalf("contentItemToObservationR5 ok=%v o=%v", ok, o)
	}
	if o.ValueDateTime == nil || string(*o.ValueDateTime) != "2005-05-30T08:00:00+01:00" {
		t.Errorf("ValueDateTime = %v, want 2005-05-30T08:00:00+01:00 (inline offset wins)", o.ValueDateTime)
	}
}

// TestContentItemNUMWithoutValueIsDropped confirms a NUM leaf with no numeric value is
// dropped and recorded rather than emitted as a Quantity with a null value, which
// would be a non-conformant Observation.
func TestContentItemNUMWithoutValueIsDropped(t *testing.T) {
	item := dicom.ContentItem{
		ValueType:        dicom.ValueTypeNum,
		Relationship:     dicom.RelationshipContains,
		ConceptName:      dicom.ConceptNameCode{CodeValue: "G-A437", CodingSchemeDesignator: "SRT", CodeMeaning: "Diameter"},
		MeasurementUnits: dicom.ConceptNameCode{CodeValue: "mm", CodingSchemeDesignator: "UCUM", CodeMeaning: "millimeter"},
		// MeasuredValue left as the zero Decimal: no numeric value parsed.
	}
	report := &Report{}
	o, ok := contentItemToObservationR5(item, "", false, report)
	if ok || o != nil {
		t.Fatalf("contentItemToObservationR5 = (%v, %v), want (nil, false) for a value-less NUM", o, ok)
	}
	if !hasDropped(report, "DICOM (0040,A30A) NumericValue") {
		t.Errorf("Report.Dropped does not record the value-less NUM: %+v", report.Dropped)
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
