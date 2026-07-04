package dicom

import (
	"errors"
	"reflect"
	"testing"
)

// namedSRTree builds the named-regression content tree: a CONTAINER root holding a
// TEXT, a CODE, a NUM, and a nested CONTAINER child that itself holds an IMAGE
// reference.
func namedSRTree(t *testing.T) *ContentItem {
	t.Helper()
	return &ContentItem{
		ValueType: ValueTypeContainer,
		ConceptName: ConceptNameCode{
			CodeValue: "11528-7", CodingSchemeDesignator: "LN", CodeMeaning: "Radiology Report",
		},
		Children: []ContentItem{
			{
				ValueType:    ValueTypeText,
				Relationship: RelationshipContains,
				ConceptName:  ConceptNameCode{CodeValue: "121071", CodingSchemeDesignator: "DCM", CodeMeaning: "Finding"},
				Text:         "Solitary pulmonary nodule.",
			},
			{
				ValueType:    ValueTypeCode,
				Relationship: RelationshipContains,
				ConceptName:  ConceptNameCode{CodeValue: "121072", CodingSchemeDesignator: "DCM", CodeMeaning: "Diagnosis"},
				Code:         ConceptNameCode{CodeValue: "R-00339", CodingSchemeDesignator: "SRT", CodeMeaning: "Normal"},
			},
			{
				ValueType:        ValueTypeNum,
				Relationship:     RelationshipContains,
				ConceptName:      ConceptNameCode{CodeValue: "G-A437", CodingSchemeDesignator: "SRT", CodeMeaning: "Diameter"},
				MeasuredValue:    mustDecimal(t, "12.5"),
				MeasurementUnits: ConceptNameCode{CodeValue: "mm", CodingSchemeDesignator: "UCUM", CodeMeaning: "millimeter"},
			},
			{
				ValueType:    ValueTypeContainer,
				Relationship: RelationshipContains,
				ConceptName:  ConceptNameCode{CodeValue: "121180", CodingSchemeDesignator: "DCM", CodeMeaning: "Key Images"},
				Children: []ContentItem{
					{
						ValueType:    ValueTypeImage,
						Relationship: RelationshipContains,
						ConceptName:  ConceptNameCode{CodeValue: "121191", CodingSchemeDesignator: "DCM", CodeMeaning: "Referenced Object"},
						Referenced: []ReferencedSOPInstance{
							{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SOPInstanceUID: "1.2.3.4.5.6.7.8.9"},
						},
					},
				},
			},
		},
	}
}

// TestSRTreeRoundTrip is the named Increment 8 regression: a content-item tree built
// in memory, encoded into a DataSet with BuildSR, then parsed back with ParseSR, must
// reconstruct an equal tree and remain navigable root -> children -> value.
func TestSRTreeRoundTrip(t *testing.T) {
	want := namedSRTree(t)

	ds := NewDataSet()
	ds.SetString(TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.88.11") // Basic Text SR
	if err := BuildSR(ds, want); err != nil {
		t.Fatalf("BuildSR: %v", err)
	}

	got, err := ParseSR(ds)
	if err != nil {
		t.Fatalf("ParseSR: %v", err)
	}

	if !reflect.DeepEqual(*want, *got) {
		t.Fatalf("round-trip mismatch:\n want %#v\n got  %#v", *want, *got)
	}

	// Navigation: root -> children -> ConceptName/value.
	if got.ConceptName.CodeMeaning != "Radiology Report" {
		t.Errorf("root concept = %+v", got.ConceptName)
	}
	if len(got.Children) != 4 {
		t.Fatalf("root has %d children, want 4", len(got.Children))
	}
	if got.Children[0].Text != "Solitary pulmonary nodule." {
		t.Errorf("TEXT child = %q", got.Children[0].Text)
	}
	if got.Children[2].MeasuredValue.String() != "12.5" {
		t.Errorf("NUM child value = %q", got.Children[2].MeasuredValue.String())
	}
	nested := got.Children[3]
	if nested.ValueType != ValueTypeContainer || len(nested.Children) != 1 {
		t.Fatalf("nested container = %+v", nested)
	}
	img := nested.Children[0]
	if img.ValueType != ValueTypeImage || img.Referenced[0].SOPInstanceUID != "1.2.3.4.5.6.7.8.9" {
		t.Errorf("IMAGE leaf = %+v", img)
	}
}

func TestBuildSRSetsRootAttributes(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.88.11")
	root := &ContentItem{
		ValueType:   ValueTypeContainer,
		ConceptName: ConceptNameCode{CodeValue: "1", CodingSchemeDesignator: "DCM", CodeMeaning: "Doc"},
	}
	if err := BuildSR(ds, root); err != nil {
		t.Fatalf("BuildSR: %v", err)
	}
	if vt, _ := ds.GetString(TagValueType); vt != "CONTAINER" {
		t.Errorf("root ValueType attr = %q, want CONTAINER", vt)
	}
	// The root must not carry a Relationship Type at the document top level.
	if _, ok := ds.Get(TagRelationshipType); ok {
		t.Error("root content item must not write a Relationship Type")
	}
	if _, ok := ds.GetSequence(TagConceptNameCodeSequence); !ok {
		t.Error("root must write a Concept Name Code Sequence")
	}
}

func TestBuildSRRejectsNilRoot(t *testing.T) {
	ds := NewDataSet()
	if err := BuildSR(ds, nil); err == nil {
		t.Error("BuildSR with a nil root should error")
	}
	if err := BuildSR(nil, &ContentItem{ValueType: ValueTypeContainer}); err == nil {
		t.Error("BuildSR with a nil dataset should error")
	}
}

func TestBuildSRRejectsNonContainerRoot(t *testing.T) {
	ds := NewDataSet()
	err := BuildSR(ds, &ContentItem{ValueType: ValueTypeText, Text: "x"})
	if err == nil {
		t.Fatal("BuildSR with a non-CONTAINER root should error")
	}
	if _, ok := errors.AsType[*ValueError](err); !ok {
		t.Errorf("want *ValueError, got %T", err)
	}
}

func TestBuildSRDepthCap(t *testing.T) {
	// Building a tree deeper than the cap returns LimitExceededError rather than
	// emitting an unbounded sequence nest.
	deepest := ContentItem{ValueType: ValueTypeContainer, Relationship: RelationshipContains}
	node := deepest
	for i := 0; i < 200; i++ {
		parent := ContentItem{
			ValueType:    ValueTypeContainer,
			Relationship: RelationshipContains,
			Children:     []ContentItem{node},
		}
		node = parent
	}
	root := &ContentItem{ValueType: ValueTypeContainer, Children: []ContentItem{node}}

	ds := NewDataSet()
	ds.SetString(TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.88.11")
	err := BuildSR(ds, root)
	if err == nil {
		t.Fatal("BuildSR of a 200-deep tree should fail the depth cap")
	}
	le, ok := errors.AsType[*LimitExceededError](err)
	if !ok {
		t.Fatalf("BuildSR deep error = %T %v, want *LimitExceededError", err, err)
	}
	if le.Kind != "sequence-depth" {
		t.Errorf("LimitExceededError.Kind = %q, want sequence-depth", le.Kind)
	}
}
