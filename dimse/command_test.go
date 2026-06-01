package dimse

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// TestCommandFieldValues pins the C-ECHO command field constants to their PS3.7 wire values.
func TestCommandFieldValues(t *testing.T) {
	if CommandCEchoRQ != 0x0030 {
		t.Errorf("CommandCEchoRQ = %#04x, want 0x0030", uint16(CommandCEchoRQ))
	}
	if CommandCEchoRSP != 0x8030 {
		t.Errorf("CommandCEchoRSP = %#04x, want 0x8030", uint16(CommandCEchoRSP))
	}
}

// TestCommandSetEncodeCEchoRQ checks the encoded C-ECHO-RQ command set: Implicit VR LE, every
// element in increasing tag order, and Command Group Length (0000,0000) encoded first carrying
// the byte length of everything that follows it.
func TestCommandSetEncodeCEchoRQ(t *testing.T) {
	cs := CommandSet{
		CommandField:        CommandCEchoRQ,
		MessageID:           7,
		AffectedSOPClassUID: dicom.UID(verificationSOPClass),
		CommandDataSetType:  CommandDataSetNotPresent,
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// The first element must be (0000,0000) UL group length, Implicit VR LE: group(2) element(2)
	// length(4) = 12 followed by a 4-byte value.
	if len(encoded) < 12 {
		t.Fatalf("encoded command set too short: %d bytes", len(encoded))
	}
	gotGroup := binary.LittleEndian.Uint16(encoded[0:2])
	gotElem := binary.LittleEndian.Uint16(encoded[2:4])
	gotLen := binary.LittleEndian.Uint32(encoded[4:8])
	if gotGroup != 0x0000 || gotElem != 0x0000 {
		t.Errorf("first element = (%04X,%04X), want (0000,0000) group length", gotGroup, gotElem)
	}
	if gotLen != 4 {
		t.Errorf("group-length value length = %d, want 4 (a UL)", gotLen)
	}
	groupLength := binary.LittleEndian.Uint32(encoded[8:12])
	// The group length must equal the number of bytes after the 12-byte group-length element.
	if int(groupLength) != len(encoded)-12 {
		t.Errorf("group length = %d, want %d (bytes following the group-length element)",
			groupLength, len(encoded)-12)
	}

	// Tags must appear in strictly increasing order.
	assertIncreasingTagOrder(t, encoded)
}

// TestCommandSetRoundTripCEchoRQ encodes and decodes a C-ECHO-RQ command set, recovering every
// field.
func TestCommandSetRoundTripCEchoRQ(t *testing.T) {
	cs := CommandSet{
		CommandField:        CommandCEchoRQ,
		MessageID:           42,
		AffectedSOPClassUID: dicom.UID(verificationSOPClass),
		CommandDataSetType:  CommandDataSetNotPresent,
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if got.CommandField != CommandCEchoRQ {
		t.Errorf("CommandField = %#04x, want C-ECHO-RQ", uint16(got.CommandField))
	}
	if got.MessageID != 42 {
		t.Errorf("MessageID = %d, want 42", got.MessageID)
	}
	if got.AffectedSOPClassUID != dicom.UID(verificationSOPClass) {
		t.Errorf("AffectedSOPClassUID = %q, want Verification", got.AffectedSOPClassUID)
	}
	if got.CommandDataSetType != CommandDataSetNotPresent {
		t.Errorf("CommandDataSetType = %#04x, want 0x0101", got.CommandDataSetType)
	}
}

// TestCommandSetRoundTripCEchoRSP round-trips a C-ECHO-RSP carrying a status and the
// message-id-being-responded-to.
func TestCommandSetRoundTripCEchoRSP(t *testing.T) {
	cs := CommandSet{
		CommandField:              CommandCEchoRSP,
		MessageIDBeingRespondedTo: 42,
		AffectedSOPClassUID:       dicom.UID(verificationSOPClass),
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    0x0000,
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)

	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if got.CommandField != CommandCEchoRSP {
		t.Errorf("CommandField = %#04x, want C-ECHO-RSP", uint16(got.CommandField))
	}
	if got.MessageIDBeingRespondedTo != 42 {
		t.Errorf("MessageIDBeingRespondedTo = %d, want 42", got.MessageIDBeingRespondedTo)
	}
	if !got.HasStatus || got.Status != 0x0000 {
		t.Errorf("status = (has=%v, %#04x), want (true, 0x0000)", got.HasStatus, got.Status)
	}
}

// TestCommandSetIsResponse reports whether the command field has its response bit (0x8000) set.
func TestCommandSetIsResponse(t *testing.T) {
	if (CommandSet{CommandField: CommandCEchoRQ}).IsResponse() {
		t.Error("C-ECHO-RQ.IsResponse() = true, want false")
	}
	if !(CommandSet{CommandField: CommandCEchoRSP}).IsResponse() {
		t.Error("C-ECHO-RSP.IsResponse() = false, want true")
	}
}

// TestDecodeCommandSetRejectsTruncated guards against a command set that ends mid-element.
func TestDecodeCommandSetRejectsTruncated(t *testing.T) {
	// (0000,0100) US length 2, but only 1 value byte present.
	raw := []byte{0x00, 0x00, 0x00, 0x01, 0x02, 0x00, 0x00, 0x00, 0x30}
	if _, err := DecodeCommandSet(raw); err == nil {
		t.Error("DecodeCommandSet should reject a command set truncated mid-value")
	}
}

// assertIncreasingTagOrder walks an Implicit VR LE command set and fails if the tags are not in
// strictly increasing order.
func assertIncreasingTagOrder(t *testing.T, encoded []byte) {
	t.Helper()
	r := bytes.NewReader(encoded)
	var prev dicom.Tag
	first := true
	for r.Len() > 0 {
		var hdr [8]byte
		if _, err := r.Read(hdr[:]); err != nil {
			t.Fatalf("reading element header: %v", err)
		}
		group := binary.LittleEndian.Uint16(hdr[0:2])
		elem := binary.LittleEndian.Uint16(hdr[2:4])
		vlen := binary.LittleEndian.Uint32(hdr[4:8])
		tag := dicom.NewTag(group, elem)
		if !first && tag <= prev {
			t.Errorf("tag %s is not greater than previous %s — command elements must be in increasing order",
				tag, prev)
		}
		prev = tag
		first = false
		if _, err := r.Seek(int64(vlen), 1); err != nil {
			t.Fatalf("skipping value: %v", err)
		}
	}
}
