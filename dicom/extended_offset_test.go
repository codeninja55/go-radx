package dicom

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// ovBytes encodes uint64 values as a little-endian OV value field.
func ovBytes(vals ...uint64) []byte {
	b := make([]byte, len(vals)*8)
	for i, v := range vals {
		binary.LittleEndian.PutUint64(b[i*8:i*8+8], v)
	}
	return b
}

func TestExtendedOffsetTableFromDataSet(t *testing.T) {
	ds := NewDataSet()
	// Two frames: offsets {0, 8}, lengths {8, 4}. OV is 64-bit little endian; the
	// value codec stores OV as a Bytes value, so encode the raw bytes directly.
	ds.Set(Element{Tag: TagExtendedOffsetTable, VR: VROV, Value: NewBytes(VROV, ovBytes(0, 8))})
	ds.Set(Element{Tag: TagExtendedOffsetTableLengths, VR: VROV, Value: NewBytes(VROV, ovBytes(8, 4))})

	eot, ok := extendedOffsetTable(ds)
	if !ok {
		t.Fatal("expected an Extended Offset Table")
	}
	if len(eot.offsets) != 2 || eot.offsets[0] != 0 || eot.offsets[1] != 8 {
		t.Errorf("offsets = %v, want [0 8]", eot.offsets)
	}
	if len(eot.lengths) != 2 || eot.lengths[0] != 8 || eot.lengths[1] != 4 {
		t.Errorf("lengths = %v, want [8 4]", eot.lengths)
	}
}

func TestExtendedOffsetTableMapsFrames(t *testing.T) {
	frag0 := []byte{1, 2, 3, 4, 5, 6} // frame 0 spans fragment 0 fully (6 bytes)
	frag1 := []byte{7, 8}             // frame 1 is fragment 1 (2 bytes)
	var s bytes.Buffer
	s.Write(itemHeader(0)) // empty BOT (Extended Offset Table supersedes it)
	s.Write(itemHeader(uint32(len(frag0))))
	s.Write(frag0)
	s.Write(itemHeader(uint32(len(frag1))))
	s.Write(frag1)
	s.Write(seqDelim())

	enc, err := parseEncapsulated(s.Bytes(), 2)
	if err != nil {
		t.Fatalf("parseEncapsulated: %v", err)
	}

	// Extended Offset Table addresses the concatenated fragment value stream: frame 0
	// at byte 0 length 6, frame 1 at byte 6 length 2.
	eot := &extendedOffsets{offsets: []uint64{0, 6}, lengths: []uint64{6, 2}}
	got, err := enc.framesViaExtendedOffsets(eot)
	if err != nil {
		t.Fatalf("framesViaExtendedOffsets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d frames, want 2", len(got))
	}
	if !bytes.Equal(got[0], frag0) {
		t.Errorf("frame 0 = %v, want %v", got[0], frag0)
	}
	if !bytes.Equal(got[1], frag1) {
		t.Errorf("frame 1 = %v, want %v", got[1], frag1)
	}
}

func TestExtendedOffsetTableRejectsOutOfRange(t *testing.T) {
	frag0 := []byte{1, 2, 3, 4}
	var s bytes.Buffer
	s.Write(itemHeader(0))
	s.Write(itemHeader(uint32(len(frag0))))
	s.Write(frag0)
	s.Write(seqDelim())

	enc, err := parseEncapsulated(s.Bytes(), 1)
	if err != nil {
		t.Fatalf("parseEncapsulated: %v", err)
	}
	// Offset+length runs past the 4-byte concatenated stream.
	eot := &extendedOffsets{offsets: []uint64{0}, lengths: []uint64{99}}
	if _, err := enc.framesViaExtendedOffsets(eot); err == nil {
		t.Fatal("expected an error for an Extended Offset Table entry past the stream")
	}
}
