package dicom

import (
	"errors"
	"testing"
)

// srDataSet builds a minimal SR document dataset around a root content item
// container: the supported SOP Class UID plus the root's ValueType/ConceptName and
// its Content Sequence children.
func srDataSet(t *testing.T, sopClass UID) *DataSet {
	t.Helper()
	ds := NewDataSet()
	ds.SetString(TagSOPClassUID, string(sopClass))
	return ds
}

func TestParseSRRejectsNonSR(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	ds.SetString(TagValueType, ValueTypeContainer.String())
	if _, err := ParseSR(ds); err == nil {
		t.Fatal("ParseSR of a non-SR SOP class should return an error")
	} else {
		if _, ok := errors.AsType[*ValueError](err); !ok {
			t.Errorf("want *ValueError for non-SR IOD, got %T: %v", err, err)
		}
	}
}

func TestParseSRRejectsMissingSOPClass(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagValueType, ValueTypeContainer.String())
	if _, err := ParseSR(ds); err == nil {
		t.Fatal("ParseSR with no SOP Class UID should error")
	}
}

func TestParseSRRejectsNonContainerRoot(t *testing.T) {
	ds := srDataSet(t, "1.2.840.10008.5.1.4.1.1.88.11")
	ds.SetString(TagValueType, ValueTypeText.String())
	if _, err := ParseSR(ds); err == nil {
		t.Fatal("ParseSR with a non-CONTAINER root should error")
	}
}

func TestParseSRRootAndChildren(t *testing.T) {
	ds := srDataSet(t, "1.2.840.10008.5.1.4.1.1.88.11") // Basic Text SR
	ds.SetString(TagValueType, ValueTypeContainer.String())
	writeCodeSeq(ds, TagConceptNameCodeSequence, ConceptNameCode{
		CodeValue: "11528-7", CodingSchemeDesignator: "LN", CodeMeaning: "Radiology Report",
	})

	textChild := NewDataSet()
	textChild.SetString(TagRelationshipType, RelationshipContains.String())
	textChild.SetString(TagValueType, ValueTypeText.String())
	writeCodeSeq(textChild, TagConceptNameCodeSequence, ConceptNameCode{
		CodeValue: "121071", CodingSchemeDesignator: "DCM", CodeMeaning: "Finding",
	})
	textChild.SetString(TagTextValue, "No acute findings.")

	ds.Set(Element{Tag: TagContentSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(textChild))})

	root, err := ParseSR(ds)
	if err != nil {
		t.Fatalf("ParseSR: %v", err)
	}
	if root.ValueType != ValueTypeContainer {
		t.Errorf("root ValueType = %v, want CONTAINER", root.ValueType)
	}
	if root.ConceptName.CodeMeaning != "Radiology Report" {
		t.Errorf("root concept = %+v", root.ConceptName)
	}
	if len(root.Children) != 1 {
		t.Fatalf("root has %d children, want 1", len(root.Children))
	}
	child := root.Children[0]
	if child.Relationship != RelationshipContains {
		t.Errorf("child relationship = %v, want CONTAINS", child.Relationship)
	}
	if child.ValueType != ValueTypeText {
		t.Errorf("child ValueType = %v, want TEXT", child.ValueType)
	}
	if child.Text != "No acute findings." {
		t.Errorf("child Text = %q", child.Text)
	}
	if child.ConceptName.CodeValue != "121071" {
		t.Errorf("child concept = %+v", child.ConceptName)
	}
}

func TestParseSRValueExtractionPerType(t *testing.T) {
	ds := srDataSet(t, "1.2.840.10008.5.1.4.1.1.88.33") // Comprehensive SR
	ds.SetString(TagValueType, ValueTypeContainer.String())

	num := NewDataSet()
	num.SetString(TagRelationshipType, RelationshipContains.String())
	num.SetString(TagValueType, ValueTypeNum.String())
	measured := NewDataSet()
	measured.Set(Element{Tag: TagNumericValue, VR: VRDS, Value: NewDecimals(VRDS, mustDecimal(t, "12.5"))})
	writeCodeSeq(measured, TagMeasurementUnitsCodeSequence, ConceptNameCode{
		CodeValue: "mm", CodingSchemeDesignator: "UCUM", CodeMeaning: "millimeter",
	})
	num.Set(Element{Tag: TagMeasuredValueSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(measured))})

	code := NewDataSet()
	code.SetString(TagRelationshipType, RelationshipContains.String())
	code.SetString(TagValueType, ValueTypeCode.String())
	writeCodeSeq(code, TagConceptCodeSequence, ConceptNameCode{
		CodeValue: "R-00339", CodingSchemeDesignator: "SRT", CodeMeaning: "Normal",
	})

	uidref := NewDataSet()
	uidref.SetString(TagRelationshipType, RelationshipContains.String())
	uidref.SetString(TagValueType, ValueTypeUIDRef.String())
	uidref.SetString(TagUID, "1.2.3.4.5")

	dt := NewDataSet()
	dt.SetString(TagRelationshipType, RelationshipContains.String())
	dt.SetString(TagValueType, ValueTypeDateTime.String())
	dt.SetString(TagDateTime, "20240115103000")

	pname := NewDataSet()
	pname.SetString(TagRelationshipType, RelationshipContains.String())
	pname.SetString(TagValueType, ValueTypePName.String())
	pname.SetString(TagPersonName, "Smith^Jane")

	img := NewDataSet()
	img.SetString(TagRelationshipType, RelationshipContains.String())
	img.SetString(TagValueType, ValueTypeImage.String())
	writeReferencedSOPSeq(img, []ReferencedSOPInstance{
		{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SOPInstanceUID: "9.9.9"},
	})

	ds.Set(Element{
		Tag: TagContentSequence, VR: VRSQ,
		Value: NewSequenceValue(NewSequence(num, code, uidref, dt, pname, img)),
	})

	root, err := ParseSR(ds)
	if err != nil {
		t.Fatalf("ParseSR: %v", err)
	}
	if len(root.Children) != 6 {
		t.Fatalf("got %d children, want 6", len(root.Children))
	}
	c := root.Children
	if c[0].MeasuredValue.String() != "12.5" || c[0].MeasurementUnits.CodeValue != "mm" {
		t.Errorf("NUM child = value %q units %+v", c[0].MeasuredValue.String(), c[0].MeasurementUnits)
	}
	if c[1].Code.CodeValue != "R-00339" {
		t.Errorf("CODE child = %+v", c[1].Code)
	}
	if c[2].UID != "1.2.3.4.5" {
		t.Errorf("UIDREF child = %q", c[2].UID)
	}
	if c[3].DateTime.String() != "20240115103000" {
		t.Errorf("DATETIME child = %q", c[3].DateTime.String())
	}
	if c[4].PersonName.String() != "Smith^Jane" {
		t.Errorf("PNAME child = %q", c[4].PersonName.String())
	}
	if len(c[5].Referenced) != 1 || c[5].Referenced[0].SOPInstanceUID != "9.9.9" {
		t.Errorf("IMAGE child = %+v", c[5].Referenced)
	}
}

func TestParseSRDepthCap(t *testing.T) {
	// A pathologically deep CONTAINS chain returns LimitExceededError, never a stack
	// overflow. The cap is the package default sequence depth (64).
	ds := srDataSet(t, "1.2.840.10008.5.1.4.1.1.88.11")
	ds.SetString(TagValueType, ValueTypeContainer.String())

	// Build a nested chain of CONTAINER items 200 deep, each holding the next.
	makeChain := func(depth int) *DataSet {
		var child *DataSet
		for i := 0; i < depth; i++ {
			node := NewDataSet()
			node.SetString(TagRelationshipType, RelationshipContains.String())
			node.SetString(TagValueType, ValueTypeContainer.String())
			if child != nil {
				node.Set(Element{Tag: TagContentSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(child))})
			}
			child = node
		}
		return child
	}
	ds.Set(Element{Tag: TagContentSequence, VR: VRSQ, Value: NewSequenceValue(NewSequence(makeChain(200)))})

	_, err := ParseSR(ds)
	if err == nil {
		t.Fatal("ParseSR of a 200-deep SR should fail the depth cap")
	}
	le, ok := errors.AsType[*LimitExceededError](err)
	if !ok {
		t.Fatalf("ParseSR deep error = %T %v, want *LimitExceededError", err, err)
	}
	if le.Kind != "sequence-depth" {
		t.Errorf("LimitExceededError.Kind = %q, want sequence-depth", le.Kind)
	}
}

func mustDecimal(t *testing.T, s string) Decimal {
	t.Helper()
	d, err := ParseDecimal(s)
	if err != nil {
		t.Fatalf("ParseDecimal(%q): %v", s, err)
	}
	return d
}
