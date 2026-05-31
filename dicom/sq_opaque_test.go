package dicom

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

// le helpers for building synthetic undefined-length sequence bytes.
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

// itemDelim, seqDelim, itemTag are the FFFE delimiter tags.
func TestScanUndefinedLengthSQDefinedItems(t *testing.T) {
	// SQ value: one defined-length item containing a single element, then the
	// sequence delimiter. The scanner captures everything up to and including the
	// delimiter trailer.
	var inner bytes.Buffer
	inner.Write(buildElement(t, ExplicitVRLittleEndian, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane")))

	var sq bytes.Buffer
	sq.Write(leTag(0xFFFE, 0xE000)) // item
	sq.Write(le32(uint32(inner.Len())))
	sq.Write(inner.Bytes())
	sq.Write(leTag(0xFFFE, 0xE0DD)) // sequence delimiter
	sq.Write(le32(0))
	// Trailing data after the SQ that must NOT be consumed.
	trailing := buildElement(t, ExplicitVRLittleEndian, NewTag(0x0020, 0x000D), VRUI, NewStrings(VRUI, "1.2.3"))

	full := append(append([]byte{}, sq.Bytes()...), trailing...)
	br := newBoundedReader(bytes.NewReader(full), defaultMaxElementLen)

	raw, err := scanUndefinedLengthValue(br, ExplicitVRLittleEndian, NewTag(0x0008, 0x1115))
	if err != nil {
		t.Fatalf("scanUndefinedLengthValue: %v", err)
	}
	if !bytes.Equal(raw, sq.Bytes()) {
		t.Errorf("captured SQ bytes not exact:\n got % x\nwant % x", raw, sq.Bytes())
	}
	// The trailing element is still readable.
	ds, err := readDataSet(br, ExplicitVRLittleEndian, newReadConfig())
	if err != nil {
		t.Fatalf("readDataSet after SQ: %v", err)
	}
	if v, ok := ds.GetUID(NewTag(0x0020, 0x000D)); !ok || v != "1.2.3" {
		t.Errorf("trailing element after SQ = %q,%v, want 1.2.3", v, ok)
	}
}

func TestScanUndefinedLengthSQUndefinedItems(t *testing.T) {
	// One undefined-length item (terminated by the item delimiter) with a nested
	// undefined-length SQ inside it.
	var nestedInner bytes.Buffer
	nestedInner.Write(buildElement(t, ExplicitVRLittleEndian, NewTag(0x0010, 0x0020), VRLO, NewStrings(VRLO, "ID1")))

	var nestedSQ bytes.Buffer
	// nested SQ header (undefined length) for (0040,A730)
	nestedSQ.Write(leTag(0x0040, 0xA730))
	nestedSQ.WriteString("SQ")
	nestedSQ.Write([]byte{0x00, 0x00}) // reserved
	nestedSQ.Write(le32(undefinedLength))
	nestedSQ.Write(leTag(0xFFFE, 0xE000)) // nested item
	nestedSQ.Write(le32(uint32(nestedInner.Len())))
	nestedSQ.Write(nestedInner.Bytes())
	nestedSQ.Write(leTag(0xFFFE, 0xE0DD)) // nested seq delim
	nestedSQ.Write(le32(0))

	var item bytes.Buffer
	item.Write(leTag(0xFFFE, 0xE000)) // outer item
	item.Write(le32(undefinedLength)) // undefined-length item
	item.Write(nestedSQ.Bytes())
	item.Write(leTag(0xFFFE, 0xE00D)) // item delimiter
	item.Write(le32(0))

	var sq bytes.Buffer
	sq.Write(item.Bytes())
	sq.Write(leTag(0xFFFE, 0xE0DD)) // outer sequence delim
	sq.Write(le32(0))

	br := newBoundedReader(bytes.NewReader(sq.Bytes()), defaultMaxElementLen)
	raw, err := scanUndefinedLengthValue(br, ExplicitVRLittleEndian, NewTag(0x0008, 0x1115))
	if err != nil {
		t.Fatalf("scanUndefinedLengthValue: %v", err)
	}
	if !bytes.Equal(raw, sq.Bytes()) {
		t.Errorf("nested SQ capture not exact:\n got % x\nwant % x", raw, sq.Bytes())
	}
}

func TestScanUndefinedLengthSQTruncatedFails(t *testing.T) {
	// An undefined-length SQ that ends before its delimiter is a truncation.
	var sq bytes.Buffer
	sq.Write(leTag(0xFFFE, 0xE000))
	sq.Write(le32(8))
	sq.Write([]byte{1, 2, 3, 4}) // only 4 of 8 declared item bytes
	br := newBoundedReader(bytes.NewReader(sq.Bytes()), defaultMaxElementLen)
	_, err := scanUndefinedLengthValue(br, ExplicitVRLittleEndian, NewTag(0x0008, 0x1115))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated SQ = %v, want io.ErrUnexpectedEOF", err)
	}
}

// The reader handles a real fixture with undefined-length sequences end to end.
func TestReadDataSetWithUndefinedLengthSQ(t *testing.T) {
	var inner bytes.Buffer
	inner.Write(buildElement(t, ExplicitVRLittleEndian, NewTag(0x0010, 0x0010), VRPN, NewStrings(VRPN, "Doe^Jane")))
	var sq bytes.Buffer
	sq.Write(leTag(0x0008, 0x1115)) // ReferencedSeriesSequence
	sq.WriteString("SQ")
	sq.Write([]byte{0x00, 0x00})
	sq.Write(le32(undefinedLength))
	sq.Write(leTag(0xFFFE, 0xE000))
	sq.Write(le32(uint32(inner.Len())))
	sq.Write(inner.Bytes())
	sq.Write(leTag(0xFFFE, 0xE0DD))
	sq.Write(le32(0))
	// a plain element after the SQ
	sq.Write(buildElement(t, ExplicitVRLittleEndian, NewTag(0x0020, 0x000D), VRUI, NewStrings(VRUI, "1.2.3")))

	br := newBoundedReader(bytes.NewReader(sq.Bytes()), defaultMaxElementLen)
	ds, err := readDataSet(br, ExplicitVRLittleEndian, newReadConfig())
	if err != nil {
		t.Fatalf("readDataSet: %v", err)
	}
	e, ok := ds.Get(NewTag(0x0008, 0x1115))
	if !ok {
		t.Fatal("SQ element dropped from dataset (DCM-005)")
	}
	if e.VR != VRSQ {
		t.Errorf("SQ element VR = %s, want SQ", e.VR)
	}
	if v, ok := ds.GetUID(NewTag(0x0020, 0x000D)); !ok || v != "1.2.3" {
		t.Errorf("element after SQ = %q,%v, want 1.2.3", v, ok)
	}

	// Re-encode and confirm the SQ bytes survive byte-identically.
	var out bytes.Buffer
	if err := writeDataSet(&out, ds, ExplicitVRLittleEndian); err != nil {
		t.Fatalf("writeDataSet: %v", err)
	}
	if !bytes.Equal(out.Bytes(), sq.Bytes()) {
		t.Errorf("SQ-bearing dataset not byte-identical on re-encode")
	}
}
