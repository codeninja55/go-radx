package dicom

import (
	"encoding/binary"
	"testing"
)

// leTag and le32 build little-endian tag and length bytes for synthetic sequence
// fixtures.
func leTag(g, e uint16) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint16(b[0:2], g)
	binary.LittleEndian.PutUint16(b[2:4], e)
	return b
}

func le32(n uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

func TestNewSequenceAndItems(t *testing.T) {
	a := NewDataSet()
	a.SetString(NewTag(0x0010, 0x0010), "Doe^Jane")
	b := NewDataSet()
	b.SetString(NewTag(0x0010, 0x0020), "ID1")

	s := NewSequence(a, b)
	if s.Len() != 2 {
		t.Fatalf("Len = %d, want 2", s.Len())
	}

	var got []*DataSet
	for it := range s.Items() {
		got = append(got, it.DataSet)
	}
	if len(got) != 2 {
		t.Fatalf("Items yielded %d, want 2", len(got))
	}
	if v, ok := got[0].GetString(NewTag(0x0010, 0x0010)); !ok || v != "Doe^Jane" {
		t.Errorf("item 0 PatientName = %q,%v", v, ok)
	}
	if v, ok := got[1].GetString(NewTag(0x0010, 0x0020)); !ok || v != "ID1" {
		t.Errorf("item 1 PatientID = %q,%v", v, ok)
	}
}

func TestSequenceAppend(t *testing.T) {
	s := NewSequence()
	if s.Len() != 0 {
		t.Fatalf("empty Len = %d, want 0", s.Len())
	}
	s.Append(NewDataSet())
	s.Append(NewDataSet())
	if s.Len() != 2 {
		t.Fatalf("after Append Len = %d, want 2", s.Len())
	}
}

func TestNewSequenceValueVR(t *testing.T) {
	v := NewSequenceValue(NewSequence(NewDataSet()))
	if v.VR() != VRSQ {
		t.Errorf("VR = %s, want SQ", v.VR())
	}
}

// NewSequence does not alias the caller's datasets across mutation paths beyond the
// usual pointer sharing; Items() must hand back the same *DataSet pointers the
// caller appended so navigation works.
func TestSequenceItemsArePointers(t *testing.T) {
	ds := NewDataSet()
	s := NewSequence(ds)
	for it := range s.Items() {
		if it.DataSet != ds {
			t.Error("Items() did not return the appended dataset pointer")
		}
	}
}

// A programmatically built sequence reports an EncodedLen consistent with the bytes
// the encoder writes for the value field (delimiter-terminated undefined form by
// default). The exact value is verified in the codec tests; here we only assert it
// does not panic and is non-zero for a non-empty sequence.
func TestNewSequenceValueEncodedLenNonZero(t *testing.T) {
	ds := NewDataSet()
	ds.SetString(NewTag(0x0010, 0x0010), "Doe^Jane")
	v := NewSequenceValue(NewSequence(ds))
	if got := v.EncodedLen(binary.LittleEndian); got == 0 {
		t.Error("non-empty sequence EncodedLen is 0")
	}
}
