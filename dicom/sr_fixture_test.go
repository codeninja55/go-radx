package dicom

import (
	"path/filepath"
	"testing"
)

// TestParseSRBasicTextFixture parses a vendored Basic Text SR fixture (pydicom
// reportsi.dcm, MIT) end to end through the Part 10 reader and ParseSR, asserting the
// real content tree is read with the correct root, relationships, value types, and
// values.
func TestParseSRBasicTextFixture(t *testing.T) {
	path := filepath.Join("..", "testdata", "dicom", "basic-text-sr.dcm")
	f, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}

	sop, _ := f.DataSet.GetUID(TagSOPClassUID)
	if sop != "1.2.840.10008.5.1.4.1.1.88.11" {
		t.Fatalf("fixture SOP Class = %s (%s), want Basic Text SR Storage", sop, sop.Name())
	}

	root, err := ParseSR(f.DataSet)
	if err != nil {
		t.Fatalf("ParseSR: %v", err)
	}

	if root.ValueType != ValueTypeContainer {
		t.Errorf("root ValueType = %v, want CONTAINER", root.ValueType)
	}
	if root.ConceptName.CodeMeaning != "Document Title" {
		t.Errorf("root concept meaning = %q, want Document Title", root.ConceptName.CodeMeaning)
	}
	if len(root.Children) != 5 {
		t.Fatalf("root has %d children, want 5", len(root.Children))
	}

	// Spot-check the documented structure: a PNAME observer, a TEXT observer
	// organisation, and a nested CONTAINS container.
	pname := root.Children[1]
	if pname.ValueType != ValueTypePName || pname.Relationship != RelationshipHasObsContext {
		t.Errorf("child[1] = vt %v rel %v, want PNAME / HAS OBS CONTEXT", pname.ValueType, pname.Relationship)
	}
	text := root.Children[2]
	if text.ValueType != ValueTypeText || text.Text == "" {
		t.Errorf("child[2] = vt %v text %q, want a non-empty TEXT", text.ValueType, text.Text)
	}
	section := root.Children[4]
	if section.ValueType != ValueTypeContainer || section.Relationship != RelationshipContains {
		t.Errorf("child[4] = vt %v rel %v, want CONTAINER / CONTAINS", section.ValueType, section.Relationship)
	}
}

// TestSRFixtureRebuildRoundTrip parses the vendored SR fixture, rebuilds the parsed
// tree into a fresh dataset with BuildSR, and re-parses it, asserting the rebuilt tree
// equals the original parse. This proves ParseSR and BuildSR are mutual inverses over
// a real content tree, not just a synthetic one.
func TestSRFixtureRebuildRoundTrip(t *testing.T) {
	path := filepath.Join("..", "testdata", "dicom", "basic-text-sr.dcm")
	f, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	parsed, err := ParseSR(f.DataSet)
	if err != nil {
		t.Fatalf("ParseSR: %v", err)
	}

	rebuilt := NewDataSet()
	rebuilt.SetString(TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.88.11")
	if err := BuildSR(rebuilt, parsed); err != nil {
		t.Fatalf("BuildSR: %v", err)
	}
	reparsed, err := ParseSR(rebuilt)
	if err != nil {
		t.Fatalf("ParseSR (rebuilt): %v", err)
	}

	if !contentItemsEqual(parsed, reparsed) {
		t.Fatalf("rebuild round-trip changed the tree:\n parsed   %d children\n reparsed %d children",
			len(parsed.Children), len(reparsed.Children))
	}
}

// contentItemsEqual compares two content items structurally. It compares the
// preserved lexical forms of value fields (Decimal, DT) rather than their internal
// representation so a re-encoded value that round-trips byte-identically compares
// equal.
func contentItemsEqual(a, b *ContentItem) bool {
	if a.ValueType != b.ValueType || a.Relationship != b.Relationship {
		return false
	}
	if a.ConceptName != b.ConceptName || a.Code != b.Code || a.MeasurementUnits != b.MeasurementUnits {
		return false
	}
	if a.Text != b.Text || a.UID != b.UID {
		return false
	}
	if a.MeasuredValue.String() != b.MeasuredValue.String() {
		return false
	}
	if a.DateTime.String() != b.DateTime.String() {
		return false
	}
	if a.PersonName.String() != b.PersonName.String() {
		return false
	}
	if len(a.Referenced) != len(b.Referenced) {
		return false
	}
	for i := range a.Referenced {
		if a.Referenced[i] != b.Referenced[i] {
			return false
		}
	}
	if len(a.Children) != len(b.Children) {
		return false
	}
	for i := range a.Children {
		if !contentItemsEqual(&a.Children[i], &b.Children[i]) {
			return false
		}
	}
	return true
}
