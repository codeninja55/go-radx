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

// TestCommandFieldCStoreValues pins the C-STORE command field constants to their PS3.7 wire
// values (verified against pynetdicom dimse_messages.py: C_STORE_RQ 0x0001, C_STORE_RSP 0x8001).
func TestCommandFieldCStoreValues(t *testing.T) {
	if CommandCStoreRQ != 0x0001 {
		t.Errorf("CommandCStoreRQ = %#04x, want 0x0001", uint16(CommandCStoreRQ))
	}
	if CommandCStoreRSP != 0x8001 {
		t.Errorf("CommandCStoreRSP = %#04x, want 0x8001", uint16(CommandCStoreRSP))
	}
}

// TestCommandSetRoundTripCStoreRQ round-trips a C-STORE-RQ carrying the SOP Class, the SOP
// Instance, the priority, and the data-set-present flag (the elements pynetdicom lists for
// C-STORE-RQ: AffectedSOPClassUID, CommandField, MessageID, Priority, CommandDataSetType,
// AffectedSOPInstanceUID).
func TestCommandSetRoundTripCStoreRQ(t *testing.T) {
	cs := CommandSet{
		CommandField:           CommandCStoreRQ,
		MessageID:              9,
		AffectedSOPClassUID:    dicom.UID("1.2.840.10008.5.1.4.1.1.4"),
		AffectedSOPInstanceUID: dicom.UID("1.2.3.4.5.6.7.8"),
		Priority:               PriorityMedium,
		CommandDataSetType:     CommandDataSetPresent,
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
	if got.CommandField != CommandCStoreRQ {
		t.Errorf("CommandField = %#04x, want C-STORE-RQ", uint16(got.CommandField))
	}
	if got.MessageID != 9 {
		t.Errorf("MessageID = %d, want 9", got.MessageID)
	}
	if got.AffectedSOPClassUID != dicom.UID("1.2.840.10008.5.1.4.1.1.4") {
		t.Errorf("AffectedSOPClassUID = %q", got.AffectedSOPClassUID)
	}
	if got.AffectedSOPInstanceUID != dicom.UID("1.2.3.4.5.6.7.8") {
		t.Errorf("AffectedSOPInstanceUID = %q, want round-trip", got.AffectedSOPInstanceUID)
	}
	if got.Priority != PriorityMedium {
		t.Errorf("Priority = %#04x, want medium (0x0000)", got.Priority)
	}
	if !got.HasDataSet() {
		t.Error("C-STORE-RQ HasDataSet() = false, want true (a dataset follows)")
	}
}

// TestCommandSetTagOrderAndGroupLength is the named DIMSE-006/007 regression. It encodes a command
// set carrying Move Destination (0000,0600) and asserts: command elements appear in strictly
// increasing tag order, Command Group Length (0000,0000) is computed last over the bytes that
// follow it, and Move Destination is encoded with VR AE (an even-length, space-padded value), not
// VR UI (which NUL-pads). The prototype gave Move Destination VR UI and computed group length
// before all elements existed.
func TestCommandSetTagOrderAndGroupLength(t *testing.T) {
	cs := CommandSet{
		CommandField:        CommandCMoveRQ,
		MessageID:           3,
		AffectedSOPClassUID: dicom.UID("1.2.840.10008.5.1.4.1.2.2.2"),
		MoveDestination:     AETitle("REMOTE-AE"), // odd length (9): AE pads with a trailing space
		Priority:            PriorityMedium,
		CommandDataSetType:  CommandDataSetPresent,
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	assertIncreasingTagOrder(t, encoded)

	// Group length must be computed over everything after the 12-byte group-length element.
	groupLength := binary.LittleEndian.Uint32(encoded[8:12])
	if int(groupLength) != len(encoded)-12 {
		t.Errorf("group length = %d, want %d (computed last over the following bytes)",
			groupLength, len(encoded)-12)
	}

	// Locate the Move Destination (0000,0600) element and check its value is AE-padded (a trailing
	// 0x20 space to even length), never UI-padded (a trailing 0x00 NUL).
	val := findCommandElementValue(t, encoded, 0x0000, 0x0600)
	if len(val) == 0 {
		t.Fatal("Move Destination (0000,0600) absent from the encoded command set")
	}
	if len(val)%2 != 0 {
		t.Errorf("Move Destination value length %d is odd; AE values pad to even length", len(val))
	}
	if val[len(val)-1] == 0x00 {
		t.Errorf("Move Destination padded with NUL (0x00) — that is UI padding; VR AE pads with space (0x20)")
	}
	if got := string(bytes.TrimRight(val, " ")); got != "REMOTE-AE" {
		t.Errorf("Move Destination = %q, want REMOTE-AE", got)
	}
}

// TestCommandFieldCFindValues pins the C-FIND and C-CANCEL command field constants to their PS3.7
// wire values (verified against pynetdicom dimse_messages.py: C_FIND_RQ 0x0020, C_FIND_RSP 0x8020,
// C_CANCEL_RQ 0x0FFF).
func TestCommandFieldCFindValues(t *testing.T) {
	if CommandCFindRQ != 0x0020 {
		t.Errorf("CommandCFindRQ = %#04x, want 0x0020", uint16(CommandCFindRQ))
	}
	if CommandCFindRSP != 0x8020 {
		t.Errorf("CommandCFindRSP = %#04x, want 0x8020", uint16(CommandCFindRSP))
	}
	if CommandCCancelRQ != 0x0FFF {
		t.Errorf("CommandCCancelRQ = %#04x, want 0x0FFF", uint16(CommandCCancelRQ))
	}
}

// TestCFindRQRoundTrip round-trips a C-FIND-RQ carrying the Affected SOP Class, the Message ID, the
// priority, and the data-set-present flag (an identifier follows). pynetdicom lists C_FIND_RQ's
// command-set keywords as AffectedSOPClassUID, CommandField, MessageID, Priority,
// CommandDataSetType. The encoding must place (0000,0000) group length first, every element in
// strictly increasing tag order, and Command Group Length last over the following bytes.
func TestCFindRQRoundTrip(t *testing.T) {
	cs := CommandSet{
		CommandField:        CommandCFindRQ,
		MessageID:           11,
		AffectedSOPClassUID: dicom.UID("1.2.840.10008.5.1.4.1.2.2.1"), // Study Root Q/R - FIND
		Priority:            PriorityMedium,
		HasPriority:         true,
		CommandDataSetType:  CommandDataSetPresent,
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)

	// Group length first, over the following bytes.
	groupLength := binary.LittleEndian.Uint32(encoded[8:12])
	if int(groupLength) != len(encoded)-12 {
		t.Errorf("group length = %d, want %d (bytes following the group-length element)",
			groupLength, len(encoded)-12)
	}

	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if got.CommandField != CommandCFindRQ {
		t.Errorf("CommandField = %#04x, want C-FIND-RQ", uint16(got.CommandField))
	}
	if got.MessageID != 11 {
		t.Errorf("MessageID = %d, want 11", got.MessageID)
	}
	if got.AffectedSOPClassUID != dicom.UID("1.2.840.10008.5.1.4.1.2.2.1") {
		t.Errorf("AffectedSOPClassUID = %q, want Study Root Q/R FIND", got.AffectedSOPClassUID)
	}
	if !got.HasPriority || got.Priority != PriorityMedium {
		t.Errorf("priority = (has=%v, %#04x), want (true, medium)", got.HasPriority, got.Priority)
	}
	if !got.HasDataSet() {
		t.Error("C-FIND-RQ HasDataSet() = false, want true (an identifier follows)")
	}

	// Re-encode the decoded set; the bytes must be identical (byte-stable round-trip).
	reencoded, err := got.Encode()
	if err != nil {
		t.Fatalf("re-Encode: %v", err)
	}
	if !bytes.Equal(encoded, reencoded) {
		t.Errorf("encode->decode->encode is not byte-stable:\n first = %x\n second = %x", encoded, reencoded)
	}
}

// TestCFindRSPPendingHasDataset checks the multi-response RSP shape: a Pending C-FIND-RSP (0xFF00)
// declares CommandDataSetType present (a matching identifier follows) and carries the status and the
// message-id-being-responded-to, while a terminal Success RSP declares the data set not-present.
func TestCFindRSPPendingHasDataset(t *testing.T) {
	pending := CommandSet{
		CommandField:              CommandCFindRSP,
		MessageIDBeingRespondedTo: 11,
		AffectedSOPClassUID:       dicom.UID("1.2.840.10008.5.1.4.1.2.2.1"),
		CommandDataSetType:        CommandDataSetPresent,
		HasStatus:                 true,
		Status:                    0xFF00,
	}
	encoded, err := pending.Encode()
	if err != nil {
		t.Fatalf("Encode pending: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)
	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet pending: %v", err)
	}
	if got.CommandField != CommandCFindRSP {
		t.Errorf("CommandField = %#04x, want C-FIND-RSP", uint16(got.CommandField))
	}
	if got.MessageIDBeingRespondedTo != 11 {
		t.Errorf("MessageIDBeingRespondedTo = %d, want 11", got.MessageIDBeingRespondedTo)
	}
	if !got.HasStatus || got.Status != 0xFF00 {
		t.Errorf("status = (has=%v, %#04x), want (true, 0xFF00 pending)", got.HasStatus, got.Status)
	}
	if !got.HasDataSet() {
		t.Error("Pending C-FIND-RSP HasDataSet() = false, want true (a matching identifier follows)")
	}

	terminal := CommandSet{
		CommandField:              CommandCFindRSP,
		MessageIDBeingRespondedTo: 11,
		AffectedSOPClassUID:       dicom.UID("1.2.840.10008.5.1.4.1.2.2.1"),
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    0x0000,
	}
	tEncoded, err := terminal.Encode()
	if err != nil {
		t.Fatalf("Encode terminal: %v", err)
	}
	tGot, err := DecodeCommandSet(tEncoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet terminal: %v", err)
	}
	if tGot.HasDataSet() {
		t.Error("terminal Success C-FIND-RSP HasDataSet() = true, want false (no dataset on the terminal response)")
	}
}

// TestCCancelRQEncodes checks the C-CANCEL-RQ command set (PS3.7 §9.3.2.3): it carries the cancel
// command field (0x0FFF, no response bit) and the Message ID Being Responded To (0000,0120) — the
// Message ID of the operation being cancelled — and declares no data set. pynetdicom's C_CANCEL_RQ
// command-set keywords are CommandGroupLength, CommandField, MessageIDBeingRespondedTo,
// CommandDataSetType, and it carries no AffectedSOPClassUID. Because the cancel command field has no
// response bit set, the codec must still emit (0000,0120), not (0000,0110) MessageID.
func TestCCancelRQEncodes(t *testing.T) {
	cs := CommandSet{
		CommandField:              CommandCCancelRQ,
		MessageIDBeingRespondedTo: 11,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	if cs.IsResponse() {
		t.Error("C-CANCEL-RQ.IsResponse() = true, want false (0x0FFF has no response bit)")
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)

	// Message ID Being Responded To (0000,0120) must be present; Message ID (0000,0110) must not.
	if val := findCommandElementValue(t, encoded, 0x0000, 0x0120); len(val) == 0 {
		t.Error("C-CANCEL-RQ missing Message ID Being Responded To (0000,0120)")
	}
	if val := findCommandElementValue(t, encoded, 0x0000, 0x0110); val != nil {
		t.Error("C-CANCEL-RQ encoded Message ID (0000,0110); it must use Message ID Being Responded To instead")
	}
	// No Affected SOP Class UID on a C-CANCEL-RQ.
	if val := findCommandElementValue(t, encoded, 0x0000, 0x0002); val != nil {
		t.Error("C-CANCEL-RQ encoded an Affected SOP Class UID (0000,0002); it carries none")
	}

	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if got.CommandField != CommandCCancelRQ {
		t.Errorf("CommandField = %#04x, want C-CANCEL-RQ", uint16(got.CommandField))
	}
	if got.MessageIDBeingRespondedTo != 11 {
		t.Errorf("MessageIDBeingRespondedTo = %d, want 11", got.MessageIDBeingRespondedTo)
	}
	if got.HasDataSet() {
		t.Error("C-CANCEL-RQ HasDataSet() = true, want false")
	}
}

// TestCommandFieldCMoveValues pins the C-MOVE command field constants to their PS3.7 wire values
// (verified against pynetdicom dimse_messages.py: C_MOVE_RQ 0x0021, C_MOVE_RSP 0x8021). The RQ
// constant already exists from the M2 DIMSE-007 regression; the RSP is added in M3.
func TestCommandFieldCMoveValues(t *testing.T) {
	if CommandCMoveRQ != 0x0021 {
		t.Errorf("CommandCMoveRQ = %#04x, want 0x0021", uint16(CommandCMoveRQ))
	}
	if CommandCMoveRSP != 0x8021 {
		t.Errorf("CommandCMoveRSP = %#04x, want 0x8021", uint16(CommandCMoveRSP))
	}
}

// TestCMoveRQRoundTrip round-trips a C-MOVE-RQ carrying the Affected SOP Class, the Message ID, the
// priority, the Move Destination (0000,0600, VR AE), and the data-set-present flag (the identifier
// follows). pynetdicom lists C_MOVE_RQ's command-set keywords as AffectedSOPClassUID, CommandField,
// MessageID, Priority, CommandDataSetType, MoveDestination. A C-MOVE-RQ never carries the
// sub-operation counts: they are present only on the RSP.
func TestCMoveRQRoundTrip(t *testing.T) {
	cs := CommandSet{
		CommandField:        CommandCMoveRQ,
		MessageID:           13,
		AffectedSOPClassUID: dicom.UID("1.2.840.10008.5.1.4.1.2.2.2"), // Study Root Q/R - MOVE
		Priority:            PriorityMedium,
		HasPriority:         true,
		MoveDestination:     AETitle("DEST-AE"),
		CommandDataSetType:  CommandDataSetPresent,
	}
	encoded, err := cs.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)

	groupLength := binary.LittleEndian.Uint32(encoded[8:12])
	if int(groupLength) != len(encoded)-12 {
		t.Errorf("group length = %d, want %d (bytes following the group-length element)",
			groupLength, len(encoded)-12)
	}

	// A C-MOVE-RQ must NOT carry any of the four sub-operation count elements.
	for _, tag := range []uint16{0x1020, 0x1021, 0x1022, 0x1023} {
		if val := findCommandElementValue(t, encoded, 0x0000, tag); val != nil {
			t.Errorf("C-MOVE-RQ encoded sub-operation count (0000,%04X); it carries none", tag)
		}
	}

	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if got.CommandField != CommandCMoveRQ {
		t.Errorf("CommandField = %#04x, want C-MOVE-RQ", uint16(got.CommandField))
	}
	if got.MessageID != 13 {
		t.Errorf("MessageID = %d, want 13", got.MessageID)
	}
	if got.MoveDestination != AETitle("DEST-AE") {
		t.Errorf("MoveDestination = %q, want DEST-AE", got.MoveDestination)
	}
	if !got.HasPriority || got.Priority != PriorityMedium {
		t.Errorf("priority = (has=%v, %#04x), want (true, medium)", got.HasPriority, got.Priority)
	}
	if !got.HasDataSet() {
		t.Error("C-MOVE-RQ HasDataSet() = false, want true (an identifier follows)")
	}
	if got.HasSubOpCounts {
		t.Error("C-MOVE-RQ decoded HasSubOpCounts = true; the RQ carries no sub-operation counts")
	}

	reencoded, err := got.Encode()
	if err != nil {
		t.Fatalf("re-Encode: %v", err)
	}
	if !bytes.Equal(encoded, reencoded) {
		t.Errorf("encode->decode->encode is not byte-stable:\n first = %x\n second = %x", encoded, reencoded)
	}
}

// TestCMoveRSPCarriesSubOperationCounts checks the C-MOVE-RSP shape: a Pending response (0xFF00)
// carries the four sub-operation counts (Remaining 0000,1020 / Completed 0000,1021 / Failed
// 0000,1022 / Warning 0000,1023, each VR US) gated by HasSubOpCounts, the status, and the
// message-id-being-responded-to, and declares no data set. A terminal RSP equally carries the final
// counts. The counts round-trip and the count elements appear in strictly increasing tag order.
func TestCMoveRSPCarriesSubOperationCounts(t *testing.T) {
	pending := CommandSet{
		CommandField:              CommandCMoveRSP,
		MessageIDBeingRespondedTo: 13,
		AffectedSOPClassUID:       dicom.UID("1.2.840.10008.5.1.4.1.2.2.2"),
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    0xFF00,
		HasSubOpCounts:            true,
		RemainingSubOperations:    2,
		CompletedSubOperations:    1,
		FailedSubOperations:       0,
		WarningSubOperations:      0,
	}
	encoded, err := pending.Encode()
	if err != nil {
		t.Fatalf("Encode pending: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)

	// All four count elements must be present and 2 bytes (US) wide.
	for _, tag := range []uint16{0x1020, 0x1021, 0x1022, 0x1023} {
		val := findCommandElementValue(t, encoded, 0x0000, tag)
		if val == nil {
			t.Errorf("Pending C-MOVE-RSP missing sub-operation count (0000,%04X)", tag)
			continue
		}
		if len(val) != 2 {
			t.Errorf("sub-operation count (0000,%04X) value length %d, want 2 (a US)", tag, len(val))
		}
	}

	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet pending: %v", err)
	}
	if got.CommandField != CommandCMoveRSP {
		t.Errorf("CommandField = %#04x, want C-MOVE-RSP", uint16(got.CommandField))
	}
	if !got.HasStatus || got.Status != 0xFF00 {
		t.Errorf("status = (has=%v, %#04x), want (true, 0xFF00 pending)", got.HasStatus, got.Status)
	}
	if !got.HasSubOpCounts {
		t.Fatal("decoded HasSubOpCounts = false, want true (the RSP carried the counts)")
	}
	if got.RemainingSubOperations != 2 || got.CompletedSubOperations != 1 ||
		got.FailedSubOperations != 0 || got.WarningSubOperations != 0 {
		t.Errorf("counts = (rem=%d comp=%d fail=%d warn=%d), want (2 1 0 0)",
			got.RemainingSubOperations, got.CompletedSubOperations,
			got.FailedSubOperations, got.WarningSubOperations)
	}
	if got.HasDataSet() {
		t.Error("C-MOVE-RSP HasDataSet() = true, want false (a C-MOVE-RSP carries no dataset)")
	}

	reencoded, err := got.Encode()
	if err != nil {
		t.Fatalf("re-Encode: %v", err)
	}
	if !bytes.Equal(encoded, reencoded) {
		t.Errorf("encode->decode->encode is not byte-stable:\n first = %x\n second = %x", encoded, reencoded)
	}
}

// TestCMoveRSPOmitsRemainingSubOp checks that OmitRemainingSubOp suppresses only the conditional
// NumberOfRemainingSubOperations (0000,1020) element while the Completed/Failed/Warning counts are
// still emitted and round-trip — the standards-valid shape a streaming C-MOVE SCP sends when it does
// not know the outstanding count (PS3.4 C.4.2.1.5).
func TestCMoveRSPOmitsRemainingSubOp(t *testing.T) {
	rsp := CommandSet{
		CommandField:              CommandCMoveRSP,
		MessageIDBeingRespondedTo: 13,
		AffectedSOPClassUID:       dicom.UID("1.2.840.10008.5.1.4.1.2.2.2"),
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    0xFF00,
		HasSubOpCounts:            true,
		OmitRemainingSubOp:        true,
		CompletedSubOperations:    3,
		FailedSubOperations:       1,
		WarningSubOperations:      0,
	}
	encoded, err := rsp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	assertIncreasingTagOrder(t, encoded)

	// NumberOfRemainingSubOperations (0000,1020) must be ABSENT; the other three must be present.
	if val := findCommandElementValue(t, encoded, 0x0000, 0x1020); val != nil {
		t.Error("OmitRemainingSubOp did not suppress NumberOfRemainingSubOperations (0000,1020)")
	}
	for _, tag := range []uint16{0x1021, 0x1022, 0x1023} {
		if val := findCommandElementValue(t, encoded, 0x0000, tag); val == nil {
			t.Errorf("count (0000,%04X) absent; only Remaining should be omitted", tag)
		}
	}

	got, err := DecodeCommandSet(encoded)
	if err != nil {
		t.Fatalf("DecodeCommandSet: %v", err)
	}
	if !got.HasSubOpCounts {
		t.Fatal("decoded HasSubOpCounts = false, want true (Completed/Failed/Warning were present)")
	}
	if got.RemainingSubOperations != 0 {
		t.Errorf("decoded Remaining = %d, want 0 (the element was omitted)", got.RemainingSubOperations)
	}
	if got.CompletedSubOperations != 3 || got.FailedSubOperations != 1 || got.WarningSubOperations != 0 {
		t.Errorf("decoded counts = (comp=%d fail=%d warn=%d), want (3 1 0)",
			got.CompletedSubOperations, got.FailedSubOperations, got.WarningSubOperations)
	}
}

// findCommandElementValue walks an Implicit VR LE command set and returns the value bytes of the
// element at (group,element), or nil if absent.
func findCommandElementValue(t *testing.T, encoded []byte, group, element uint16) []byte {
	t.Helper()
	r := bytes.NewReader(encoded)
	for r.Len() > 0 {
		var hdr [8]byte
		if _, err := r.Read(hdr[:]); err != nil {
			t.Fatalf("reading element header: %v", err)
		}
		g := binary.LittleEndian.Uint16(hdr[0:2])
		e := binary.LittleEndian.Uint16(hdr[2:4])
		vlen := binary.LittleEndian.Uint32(hdr[4:8])
		val := make([]byte, vlen)
		if _, err := r.Read(val); err != nil && vlen > 0 {
			t.Fatalf("reading element value: %v", err)
		}
		if g == group && e == element {
			return val
		}
	}
	return nil
}
