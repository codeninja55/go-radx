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
