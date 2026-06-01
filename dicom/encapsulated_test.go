package dicom

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

// itemHeader writes an (FFFE,E000) item header with the given length in little endian.
func itemHeader(length uint32) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint16(b[0:2], 0xFFFE)
	binary.LittleEndian.PutUint16(b[2:4], 0xE000)
	binary.LittleEndian.PutUint32(b[4:8], length)
	return b
}

// seqDelim writes the (FFFE,E0DD) Sequence Delimitation Item.
func seqDelim() []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint16(b[0:2], 0xFFFE)
	binary.LittleEndian.PutUint16(b[2:4], 0xE0DD)
	binary.LittleEndian.PutUint32(b[4:8], 0)
	return b
}

// bot writes a Basic Offset Table item carrying the given 32-bit offsets.
func bot(offsets ...uint32) []byte {
	var buf bytes.Buffer
	buf.Write(itemHeader(uint32(len(offsets) * 4)))
	for _, o := range offsets {
		var v [4]byte
		binary.LittleEndian.PutUint32(v[:], o)
		buf.Write(v[:])
	}
	return buf.Bytes()
}

func TestParseEncapsulatedEmptyBOTOneFragmentPerFrame(t *testing.T) {
	// Empty BOT, two fragments, two frames -> one fragment per frame.
	var s bytes.Buffer
	s.Write(itemHeader(0)) // empty BOT
	frag0 := []byte{1, 2, 3, 4}
	frag1 := []byte{5, 6, 7, 8}
	s.Write(itemHeader(uint32(len(frag0))))
	s.Write(frag0)
	s.Write(itemHeader(uint32(len(frag1))))
	s.Write(frag1)
	s.Write(seqDelim())

	enc, err := parseEncapsulated(s.Bytes(), 2)
	if err != nil {
		t.Fatalf("parseEncapsulated: %v", err)
	}
	if len(enc.fragments) != 2 {
		t.Fatalf("parsed %d fragments, want 2", len(enc.fragments))
	}
	if bt, ok := enc.basicOffsetTable(); ok && len(bt) != 0 {
		t.Errorf("expected empty BOT, got %v", bt)
	}
	frames := enc.frameRanges()
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	if !bytes.Equal(enc.frameBytes(frames[0]), frag0) {
		t.Errorf("frame 0 = %v, want %v", enc.frameBytes(frames[0]), frag0)
	}
	if !bytes.Equal(enc.frameBytes(frames[1]), frag1) {
		t.Errorf("frame 1 = %v, want %v", enc.frameBytes(frames[1]), frag1)
	}
}

func TestParseEncapsulatedWithBasicOffsetTable(t *testing.T) {
	frag0 := []byte{1, 2, 3, 4, 5, 6}
	frag1 := []byte{7, 8}
	// Offsets are relative to the first byte after the BOT item.
	// frag0 item header (8) + frag0 (6) = 14 is where frag1's item header begins.
	var s bytes.Buffer
	s.Write(bot(0, 14))
	s.Write(itemHeader(uint32(len(frag0))))
	s.Write(frag0)
	s.Write(itemHeader(uint32(len(frag1))))
	s.Write(frag1)
	s.Write(seqDelim())

	enc, err := parseEncapsulated(s.Bytes(), 2)
	if err != nil {
		t.Fatalf("parseEncapsulated: %v", err)
	}
	bt, ok := enc.basicOffsetTable()
	if !ok || len(bt) != 2 || bt[0] != 0 || bt[1] != 14 {
		t.Fatalf("BOT = %v ok=%v, want [0 14]", bt, ok)
	}
	frames := enc.frameRanges()
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	if !bytes.Equal(enc.frameBytes(frames[0]), frag0) {
		t.Errorf("frame 0 = %v, want %v", enc.frameBytes(frames[0]), frag0)
	}
	if !bytes.Equal(enc.frameBytes(frames[1]), frag1) {
		t.Errorf("frame 1 = %v, want %v", enc.frameBytes(frames[1]), frag1)
	}
}

func TestParseEncapsulatedMultiFragmentSingleFrame(t *testing.T) {
	// One frame split across two fragments, signalled by a single-entry BOT.
	frag0 := []byte{1, 2, 3, 4}
	frag1 := []byte{5, 6, 7, 8}
	var s bytes.Buffer
	s.Write(bot(0))
	s.Write(itemHeader(uint32(len(frag0))))
	s.Write(frag0)
	s.Write(itemHeader(uint32(len(frag1))))
	s.Write(frag1)
	s.Write(seqDelim())

	enc, err := parseEncapsulated(s.Bytes(), 1)
	if err != nil {
		t.Fatalf("parseEncapsulated: %v", err)
	}
	frames := enc.frameRanges()
	if len(frames) != 1 {
		t.Fatalf("got %d frames, want 1", len(frames))
	}
	want := append(append([]byte{}, frag0...), frag1...)
	if !bytes.Equal(enc.frameBytes(frames[0]), want) {
		t.Errorf("frame 0 = %v, want %v", enc.frameBytes(frames[0]), want)
	}
}

// DCM-006: a truncated trailing item header is an error, not a silent break that
// produces a partial image.
func TestParseEncapsulatedTruncatedHeaderIsError(t *testing.T) {
	var s bytes.Buffer
	s.Write(itemHeader(0)) // empty BOT
	s.Write(itemHeader(4))
	s.Write([]byte{1, 2, 3, 4})
	// A trailing partial item header (only 5 of 8 bytes) instead of a delimiter.
	s.Write([]byte{0xFE, 0xFF, 0x00, 0xE0, 0x10})

	_, err := parseEncapsulated(s.Bytes(), 1)
	if err == nil {
		t.Fatal("expected an error for a truncated trailing item header")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("error = %v, want it to wrap io.ErrUnexpectedEOF", err)
	}
}

// DCM-006: an item whose declared length runs past the bytes remaining is rejected
// before any read, not accepted as a short fragment.
func TestParseEncapsulatedOverlongFragmentIsError(t *testing.T) {
	var s bytes.Buffer
	s.Write(itemHeader(0))          // empty BOT
	s.Write(itemHeader(0x7FFFFFFF)) // claims 2 GiB but no data follows
	s.Write([]byte{1, 2})

	_, err := parseEncapsulated(s.Bytes(), 1)
	if err == nil {
		t.Fatal("expected an error for a fragment length past the stream end")
	}
}

// DCM-006: an odd fragment length is invalid (PS3.5 items are even-length).
func TestParseEncapsulatedOddFragmentLengthIsError(t *testing.T) {
	var s bytes.Buffer
	s.Write(itemHeader(0))
	s.Write(itemHeader(3)) // odd
	s.Write([]byte{1, 2, 3})
	s.Write(seqDelim())

	_, err := parseEncapsulated(s.Bytes(), 1)
	if err == nil {
		t.Fatal("expected an error for an odd-length fragment item")
	}
}

// DCM-006: the stream must begin with a Basic Offset Table item (FFFE,E000); a
// different first tag is malformed.
func TestParseEncapsulatedMissingBOTItemIsError(t *testing.T) {
	var s bytes.Buffer
	s.Write(seqDelim()) // delimiter where the BOT item must be
	_, err := parseEncapsulated(s.Bytes(), 1)
	if err == nil {
		t.Fatal("expected an error for a stream not starting with a BOT item")
	}
}

// DCM-006: a stream that ends without a Sequence Delimitation Item is truncated.
func TestParseEncapsulatedMissingDelimiterIsError(t *testing.T) {
	var s bytes.Buffer
	s.Write(itemHeader(0))
	s.Write(itemHeader(4))
	s.Write([]byte{1, 2, 3, 4})
	// no seqDelim

	_, err := parseEncapsulated(s.Bytes(), 1)
	if err == nil {
		t.Fatal("expected an error for a stream missing its Sequence Delimitation Item")
	}
}

func TestParseEncapsulatedBOTOffsetOutOfRangeIsError(t *testing.T) {
	frag0 := []byte{1, 2, 3, 4}
	var s bytes.Buffer
	s.Write(bot(0, 9999)) // second offset points past every fragment
	s.Write(itemHeader(uint32(len(frag0))))
	s.Write(frag0)
	s.Write(seqDelim())

	// The frame mapping is validated as the stream closes, so a Basic Offset Table
	// entry that matches no fragment fails closed during parse (Codex DCM-006).
	if _, err := parseEncapsulated(s.Bytes(), 2); err == nil {
		t.Fatal("expected an error for a BOT offset that matches no fragment")
	}
}
