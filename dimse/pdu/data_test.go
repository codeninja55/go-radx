package pdu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestPDVControlBits(t *testing.T) {
	cases := []struct {
		header            byte
		isCommand, isLast bool
	}{
		{0x00, false, false}, // dataset, more fragments
		{0x01, true, false},  // command, more fragments
		{0x02, false, true},  // dataset, last
		{0x03, true, true},   // command, last (the DIMSE-001 case)
	}
	for _, c := range cases {
		pdv := PresentationDataValue{MessageControlHeader: c.header}
		if pdv.IsCommand() != c.isCommand {
			t.Errorf("header %#02x IsCommand() = %v, want %v", c.header, pdv.IsCommand(), c.isCommand)
		}
		if pdv.IsLastFragment() != c.isLast {
			t.Errorf("header %#02x IsLastFragment() = %v, want %v", c.header, pdv.IsLastFragment(), c.isLast)
		}
	}
}

func TestMakeControlHeader(t *testing.T) {
	// A final command fragment is 0x03 regardless of whether a dataset follows.
	if got := MakeControlHeader(true, true); got != 0x03 {
		t.Errorf("MakeControlHeader(command=true, last=true) = %#02x, want 0x03", got)
	}
	if got := MakeControlHeader(false, true); got != 0x02 {
		t.Errorf("MakeControlHeader(command=false, last=true) = %#02x, want 0x02", got)
	}
	if got := MakeControlHeader(true, false); got != 0x01 {
		t.Errorf("MakeControlHeader(command=true, last=false) = %#02x, want 0x01", got)
	}
}

func TestPDVEncodeDecodeRoundTrip(t *testing.T) {
	pdv := PresentationDataValue{
		PresentationContextID: 1,
		MessageControlHeader:  MakeControlHeader(true, true), // 0x03
		Data:                  []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	var buf bytes.Buffer
	if err := encodePDV(&buf, pdv); err != nil {
		t.Fatalf("encodePDV: %v", err)
	}
	// item length = 2 header bytes + 4 payload = 6, big-endian.
	want := []byte{0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0xDE, 0xAD, 0xBE, 0xEF}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("encodePDV bytes = % x, want % x", buf.Bytes(), want)
	}
	br := newBoundedReader(&buf, int64(buf.Len()))
	got, err := decodePDV(br)
	if err != nil {
		t.Fatalf("decodePDV: %v", err)
	}
	if got.PresentationContextID != 1 || got.MessageControlHeader != 0x03 ||
		!bytes.Equal(got.Data, pdv.Data) {
		t.Errorf("decodePDV = %+v, want round-trip of %+v", got, pdv)
	}
}

// TestPDVDecodeRejectsUnderflowLength guards Codex DIMSE-004: an item length below
// the 2-byte header must be rejected before the length-2 subtraction, never
// underflow into a giant allocation.
func TestPDVDecodeRejectsUnderflowLength(t *testing.T) {
	for _, badLen := range []uint32{0, 1} {
		var raw bytes.Buffer
		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], badLen)
		raw.Write(hdr[:])
		br := newBoundedReader(&raw, int64(raw.Len()))
		_, err := decodePDV(br)
		if err == nil {
			t.Fatalf("decodePDV(item length %d) = nil error, want rejection", badLen)
		}
		if _, ok := errors.AsType[*PDUError](err); !ok {
			t.Errorf("decodePDV(item length %d) error = %T, want *PDUError", badLen, err)
		}
	}
}

// TestPDVDecodeRejectsLengthBeyondBody guards against a PDV item length larger than
// the bytes remaining in the PDU body.
func TestPDVDecodeRejectsLengthBeyondBody(t *testing.T) {
	var raw bytes.Buffer
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], 1000) // claims 1000 bytes...
	raw.Write(hdr[:])
	raw.Write([]byte{0x01, 0x03, 0x00}) // ...but only 3 follow
	br := newBoundedReader(&raw, int64(raw.Len()))
	if _, err := decodePDV(br); err == nil {
		t.Error("decodePDV should reject an item length exceeding the bytes remaining")
	}
}

// TestPDVDecodeRejectsOversizedPayloadBeforeAlloc guards the allocation against a
// hostile declared length: the bounded reader is seeded with a huge remaining count
// (as it would be from an attacker-declared PDU length), so the remaining-bytes
// guard alone would pass — only the absolute MaxPDULength cap stops the make. The
// payload one byte over the ceiling must be rejected before any buffer is allocated.
func TestPDVDecodeRejectsOversizedPayloadBeforeAlloc(t *testing.T) {
	var raw bytes.Buffer
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], MaxPDULength+pdvHeaderLen+1) // payload = MaxPDULength + 1
	raw.Write(hdr[:])
	br := newBoundedReader(&raw, int64(1)<<40) // remaining far exceeds the bytes present
	_, err := decodePDV(br)
	if err == nil {
		t.Fatal("decodePDV should reject a payload exceeding MaxPDULength before allocation")
	}
	if _, ok := errors.AsType[*PDUError](err); !ok {
		t.Errorf("error = %T, want *PDUError", err)
	}
}

// TestDataTFRejectsEmptyBody asserts a P-DATA-TF whose body carries no PDV items is a
// protocol error, not an empty success (PS3.8 §9.3.5 requires one or more PDVs).
func TestDataTFRejectsEmptyBody(t *testing.T) {
	out := &DataTF{}
	err := out.Decode(newBoundedReader(bytes.NewReader(nil), 0))
	if err == nil {
		t.Fatal("Decode of an empty P-DATA-TF body should be a protocol error")
	}
	if _, ok := errors.AsType[*PDUError](err); !ok {
		t.Errorf("error = %T, want *PDUError", err)
	}
}

func TestDataTFRoundTrip(t *testing.T) {
	in := &DataTF{Items: []PresentationDataValue{
		{PresentationContextID: 1, MessageControlHeader: 0x01, Data: []byte("cmd")},
		{PresentationContextID: 1, MessageControlHeader: 0x03, Data: []byte("end")},
	}}
	var buf bytes.Buffer
	if err := in.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if PDUType(buf.Bytes()[0]) != PDUTypeData {
		t.Fatalf("first byte = %#02x, want P-DATA-TF type 0x04", buf.Bytes()[0])
	}
	pt, length, err := readHeader(&buf)
	if err != nil || pt != PDUTypeData {
		t.Fatalf("readHeader = (%s, %d), %v", pt, length, err)
	}
	out := &DataTF{}
	if err := out.Decode(newBoundedReader(&buf, int64(length))); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(out.Items) != 2 || string(out.Items[0].Data) != "cmd" || string(out.Items[1].Data) != "end" {
		t.Errorf("Decode items = %+v, want round-trip", out.Items)
	}
	if !out.Items[1].IsLastFragment() {
		t.Error("last item should be marked last-fragment")
	}
}
