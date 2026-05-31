package dicom

import (
	"slices"
	"testing"
)

func TestDataSetGetSetDelete(t *testing.T) {
	ds := NewDataSet()
	if _, ok := ds.Get(TagPatientName); ok {
		t.Error("empty dataset should return ok == false")
	}
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "Doe^Jane")})
	e, ok := ds.Get(TagPatientName)
	if !ok || e.VR != VRPN {
		t.Errorf("Get(PatientName) = (%+v,%v)", e, ok)
	}
	if ds.Len() != 1 {
		t.Errorf("Len() = %d, want 1", ds.Len())
	}
	ds.Delete(TagPatientName)
	if _, ok := ds.Get(TagPatientName); ok {
		t.Error("Delete should remove the element")
	}
	ds.Delete(TagPatientName) // not an error when absent
}

func TestDataSetAllAscendingTagOrder(t *testing.T) {
	ds := NewDataSet()
	// Insert out of order; All must yield ascending.
	ds.Set(Element{Tag: TagPixelData, VR: VROW, Value: NewBytes(VROW, []byte{0, 0})})
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "a")})
	ds.Set(Element{Tag: TagStudyInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2")})
	var got []Tag
	for e := range ds.All() {
		got = append(got, e.Tag)
	}
	want := []Tag{TagPatientName, TagStudyInstanceUID, TagPixelData}
	if !slices.Equal(got, want) {
		t.Errorf("All() order = %v, want ascending %v", got, want)
	}
}

func TestDataSetSetReplaces(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "a")})
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "b")})
	if ds.Len() != 1 {
		t.Errorf("Set on same tag should replace, Len() = %d", ds.Len())
	}
	e, _ := ds.Get(TagPatientName)
	if s, _ := ds.GetString(TagPatientName); s != "b" {
		t.Errorf("replaced value = %q, want b (element VR %s)", s, e.VR)
	}
}

func TestSetStringUsesDictionaryVR(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(TagPatientName, "Doe^Jane")
	e, ok := ds.Get(TagPatientName)
	if !ok || e.VR != VRPN { // dictionary VR for PatientName is PN
		t.Errorf("SetString should use dictionary VR PN, got %s", e.VR)
	}
}

func TestSetEmptyDeclaresZeroLengthReturnKey(t *testing.T) {
	// The C-FIND universal-match idiom.
	ds := NewDataSet()
	ds.SetEmpty(TagStudyDescription)
	e, ok := ds.Get(TagStudyDescription)
	if !ok {
		t.Fatal("SetEmpty should insert the element")
	}
	if e.Value.EncodedLen(nil) != 0 {
		t.Errorf("SetEmpty value length = %d, want 0", e.Value.EncodedLen(nil))
	}
}

func TestCloneIsDeep(t *testing.T) {
	src := NewDataSet()
	src.Set(Element{Tag: TagPixelData, VR: VROB, Value: NewBytes(VROB, []byte{1, 2, 3, 4})})

	clone := src.Clone()
	// Mutate the clone's element; the source must be untouched.
	clone.Set(Element{Tag: TagPixelData, VR: VROB, Value: NewBytes(VROB, []byte{9, 9})})

	se, _ := src.Get(TagPixelData)
	sb := se.Value.(*Bytes).Bytes()
	if len(sb) != 4 || sb[0] != 1 {
		t.Errorf("Clone aliased source value: source bytes = %v (Codex DCM-016)", sb)
	}
}

func TestCloneLengthMatches(t *testing.T) {
	src := NewDataSet()
	src.SetString(TagPatientName, "a")
	src.SetString(TagStudyInstanceUID, "1.2")
	if src.Clone().Len() != src.Len() {
		t.Error("Clone should preserve element count")
	}
}

func TestTypedGetters(t *testing.T) {
	ds := NewDataSet()
	ds.Set(Element{Tag: TagStudyInstanceUID, VR: VRUI, Value: NewStrings(VRUI, "1.2.840.10008.1.2.1")})
	ds.Set(Element{Tag: TagPatientName, VR: VRPN, Value: NewStrings(VRPN, "Doe^Jane")})
	d, _ := ParseDecimal("1.5")
	ds.Set(Element{Tag: LookupKeywordTag("SliceThickness"), VR: VRDS, Value: NewDecimals(VRDS, d)})
	ds.Set(Element{Tag: LookupKeywordTag("Rows"), VR: VRUS, Value: NewInts(VRUS, 512)})

	if u, ok := ds.GetUID(TagStudyInstanceUID); !ok || u != "1.2.840.10008.1.2.1" {
		t.Errorf("GetUID = (%q,%v)", u, ok)
	}
	if pn, ok := ds.GetPersonName(TagPatientName); !ok || pn.Alphabetic.FamilyName != "Doe" {
		t.Errorf("GetPersonName = (%+v,%v)", pn, ok)
	}
	if dec, ok := ds.GetDecimal(LookupKeywordTag("SliceThickness")); !ok || dec.String() != "1.5" {
		t.Errorf("GetDecimal = (%v,%v)", dec, ok)
	}
	if n, ok := ds.GetInt(LookupKeywordTag("Rows")); !ok || n != 512 {
		t.Errorf("GetInt = (%d,%v)", n, ok)
	}
	if _, ok := ds.GetString(TagPatientID); ok {
		t.Error("absent tag should return ok == false")
	}
}
