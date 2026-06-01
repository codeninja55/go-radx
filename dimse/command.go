package dimse

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/codeninja55/go-radx/dicom"
)

// CommandField is the DIMSE command type (0000,0100), a US value (PS3.7 §9.3, §10.3). The
// response bit (0x8000) distinguishes a response (-RSP) from a request (-RQ).
type CommandField uint16

const (
	// CommandCStoreRQ is the C-STORE request command field (PS3.7 §9.1.1, verified against
	// pynetdicom dimse_messages.py C_STORE_RQ).
	CommandCStoreRQ CommandField = 0x0001
	// CommandCStoreRSP is the C-STORE response command field (PS3.7 §9.1.1).
	CommandCStoreRSP CommandField = 0x8001
	// CommandCEchoRQ is the C-ECHO request command field (PS3.7 §9.3.5).
	CommandCEchoRQ CommandField = 0x0030
	// CommandCEchoRSP is the C-ECHO response command field (PS3.7 §9.3.6).
	CommandCEchoRSP CommandField = 0x8030
	// CommandCMoveRQ is the C-MOVE request command field (PS3.7 §9.1.4). The C-MOVE service is
	// M3 scope; the constant exists so the command-set encoder can carry Move Destination, the
	// VR-AE element the DIMSE-007 regression exercises.
	CommandCMoveRQ CommandField = 0x0021
)

// Priority is the DIMSE operation priority (0000,0700), a US value (PS3.7 §10.3.1). The wire
// values are not ordered numerically: medium is 0x0000, high 0x0001, low 0x0002 (verified against
// pynetdicom DIMSE priority constants).
type Priority uint16

const (
	// PriorityMedium is the default operation priority (0x0000).
	PriorityMedium Priority = 0x0000
	// PriorityHigh requests expedited handling (0x0001).
	PriorityHigh Priority = 0x0001
	// PriorityLow requests deferred handling (0x0002).
	PriorityLow Priority = 0x0002
)

// commandResponseBit marks a command field as a response (PS3.7 §9.1: the high bit set).
const commandResponseBit uint16 = 0x8000

// Command Data Set Type values (0000,0800), a US (PS3.7 §10.3.6). A value other than
// CommandDataSetNotPresent means a data set follows the command set in the message.
const (
	// CommandDataSetPresent indicates a data set follows the command (any value != 0x0101).
	CommandDataSetPresent uint16 = 0x0000
	// CommandDataSetNotPresent (0x0101) indicates no data set follows; a C-ECHO carries none.
	CommandDataSetNotPresent uint16 = 0x0101
)

// Command-set element tags (PS3.7 §10.3, group 0000). The command set is always encoded in
// Implicit VR Little Endian, so the VR is implied by the command dictionary, not the wire. The
// command dictionary is separate from the PS3.6 data dictionary the dicom package ships (which
// has no group-0000 entries), so it is held here as commandVR.
var (
	tagCommandGroupLength        = dicom.NewTag(0x0000, 0x0000) // UL
	tagAffectedSOPClassUID       = dicom.NewTag(0x0000, 0x0002) // UI
	tagCommandField              = dicom.NewTag(0x0000, 0x0100) // US
	tagMessageID                 = dicom.NewTag(0x0000, 0x0110) // US
	tagMessageIDBeingRespondedTo = dicom.NewTag(0x0000, 0x0120) // US
	tagMoveDestination           = dicom.NewTag(0x0000, 0x0600) // AE
	tagPriority                  = dicom.NewTag(0x0000, 0x0700) // US
	tagCommandDataSetType        = dicom.NewTag(0x0000, 0x0800) // US
	tagStatus                    = dicom.NewTag(0x0000, 0x0900) // US
	tagAffectedSOPInstanceUID    = dicom.NewTag(0x0000, 0x1000) // UI
)

// commandVR is the Value Representation each group-0000 command element carries in the DICOM
// Command Dictionary (PS3.7 §E.1, verified against pydicom _dicom_dict.py). Although the command
// set is encoded Implicit VR LE — so the VR is not written on the wire — the encoder selects the
// padding and width by VR: a UI value NUL-pads to even length, an AE value space-pads, a US is two
// bytes, a UL four. Giving Move Destination (0000,0600) its dictionary VR AE rather than UI was
// the DIMSE-007 fix (the prototype NUL-padded an AE field).
var commandVR = map[dicom.Tag]dicom.VR{
	tagCommandGroupLength:        dicom.VRUL,
	tagAffectedSOPClassUID:       dicom.VRUI,
	tagCommandField:              dicom.VRUS,
	tagMessageID:                 dicom.VRUS,
	tagMessageIDBeingRespondedTo: dicom.VRUS,
	tagMoveDestination:           dicom.VRAE,
	tagPriority:                  dicom.VRUS,
	tagCommandDataSetType:        dicom.VRUS,
	tagStatus:                    dicom.VRUS,
	tagAffectedSOPInstanceUID:    dicom.VRUI,
}

// CommandSet is a decoded DIMSE command (PS3.7 §6.3.1): the elements of group 0000 that carry
// the command type, message identifiers, the affected SOP Class, the data-set-present flag, and
// (on a response) the status. It is always encoded as Implicit VR Little Endian. This minimal
// model carries the fields a C-ECHO exercises; later services extend it (Increment 5).
type CommandSet struct {
	CommandField              CommandField
	MessageID                 uint16
	MessageIDBeingRespondedTo uint16
	AffectedSOPClassUID       dicom.UID
	AffectedSOPInstanceUID    dicom.UID
	CommandDataSetType        uint16
	// Priority is the operation priority (0000,0700); present on data-bearing requests (C-STORE,
	// C-FIND, C-GET, C-MOVE). HasPriority distinguishes a present medium priority (0x0000) from an
	// absent element, since PriorityMedium is the zero value.
	HasPriority bool
	Priority    Priority
	// MoveDestination is the C-MOVE destination AE Title (0000,0600), VR AE. It is empty on a
	// C-STORE-RQ (which does not carry it); the field exists so the command-set encoder exercises
	// a VR-AE element (the DIMSE-007 regression).
	MoveDestination AETitle
	// HasStatus distinguishes a present zero status (0x0000 Success) from an absent one; the
	// status element is present only on responses.
	HasStatus bool
	Status    uint16
}

// IsResponse reports whether the command field has its response bit (0x8000) set.
func (cs CommandSet) IsResponse() bool { return uint16(cs.CommandField)&commandResponseBit != 0 }

// HasDataSet reports whether the command declares a data set follows (CommandDataSetType is any
// value other than 0x0101). A C-ECHO never carries one.
func (cs CommandSet) HasDataSet() bool { return cs.CommandDataSetType != CommandDataSetNotPresent }

// commandElement is one group-0000 element awaiting encoding: its tag and its value bytes. The
// VR is implicit (the command set is always Implicit VR LE), so only a 4-byte length precedes
// the value on the wire.
type commandElement struct {
	tag   dicom.Tag
	value []byte
}

// Encode serialises the command set as Implicit VR Little Endian (PS3.7 §6.3.1). Elements are
// emitted in strictly increasing tag order, and Command Group Length (0000,0000) is computed
// over the bytes of every following element and written first — so its value is computed last
// even though it is encoded at the front (Codex DIMSE-006/007 build discipline). A C-ECHO
// command set carries no data set, so no fragmentation is exercised here.
func (cs CommandSet) Encode() ([]byte, error) {
	elements := cs.elements()
	sort.Slice(elements, func(i, j int) bool { return elements[i].tag < elements[j].tag })

	var body bytes.Buffer
	for _, e := range elements {
		if err := writeImplicitVRElement(&body, e.tag, e.value); err != nil {
			return nil, err
		}
	}

	// Command Group Length is the byte length of the encoded command set EXCLUDING the
	// group-length element itself (PS3.7 §10.3.1). Compute it over body, then prepend it.
	var out bytes.Buffer
	groupLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(groupLength, uint32(body.Len()))
	if err := writeImplicitVRElement(&out, tagCommandGroupLength, groupLength); err != nil {
		return nil, err
	}
	if _, err := out.Write(body.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// elements builds the non-group-length elements this command set carries, each with its
// dictionary VR's wire encoding. Only the elements relevant to the present command type are
// emitted: the message-id on requests, the message-id-being-responded-to and status on
// responses.
func (cs CommandSet) elements() []commandElement {
	var es []commandElement
	if cs.AffectedSOPClassUID != "" {
		es = append(es, encodeCommandString(tagAffectedSOPClassUID, string(cs.AffectedSOPClassUID)))
	}
	es = append(es, encodeCommandUS(tagCommandField, uint16(cs.CommandField)))
	if cs.IsResponse() {
		es = append(es, encodeCommandUS(tagMessageIDBeingRespondedTo, cs.MessageIDBeingRespondedTo))
	} else {
		es = append(es, encodeCommandUS(tagMessageID, cs.MessageID))
	}
	if cs.MoveDestination != "" {
		es = append(es, encodeCommandString(tagMoveDestination, string(cs.MoveDestination)))
	}
	if cs.HasPriority {
		es = append(es, encodeCommandUS(tagPriority, uint16(cs.Priority)))
	}
	es = append(es, encodeCommandUS(tagCommandDataSetType, cs.CommandDataSetType))
	if cs.HasStatus {
		es = append(es, encodeCommandUS(tagStatus, cs.Status))
	}
	if cs.AffectedSOPInstanceUID != "" {
		es = append(es, encodeCommandString(tagAffectedSOPInstanceUID, string(cs.AffectedSOPInstanceUID)))
	}
	return es
}

// encodeCommandUS encodes a US (or UL, for the group-length element) command value at tag, taking
// the width from the command dictionary VR so a UL is four bytes and a US two.
func encodeCommandUS(tag dicom.Tag, v uint16) commandElement {
	if commandVR[tag] == dicom.VRUL {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(v))
		return commandElement{tag, b}
	}
	return commandElement{tag, encodeUS(v)}
}

// encodeCommandString encodes a string command value at tag, padding per its command dictionary
// VR: a UI value NUL-pads to even length, an AE value space-pads (the DIMSE-007 distinction).
func encodeCommandString(tag dicom.Tag, s string) commandElement {
	if commandVR[tag] == dicom.VRAE {
		return commandElement{tag, encodeAE(s)}
	}
	return commandElement{tag, encodeUI(s)}
}

// DecodeCommandSet parses an Implicit VR Little Endian command set. It tolerates command
// elements it does not model (skipping their values) so a peer's optional elements (e.g.
// ErrorComment) do not fail the decode. A value length exceeding the bytes remaining is a
// truncation, surfaced as an error (PRD §9.2/§9.3).
func DecodeCommandSet(data []byte) (CommandSet, error) {
	var cs CommandSet
	r := bytes.NewReader(data)
	for r.Len() > 0 {
		var hdr [8]byte
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			return CommandSet{}, &ProtocolError{
				State:  Sta6,
				Detail: "command set truncated reading element header: " + err.Error(),
			}
		}
		tag := dicom.NewTag(binary.LittleEndian.Uint16(hdr[0:2]), binary.LittleEndian.Uint16(hdr[2:4]))
		vlen := binary.LittleEndian.Uint32(hdr[4:8])
		if int64(vlen) > int64(r.Len()) {
			return CommandSet{}, &ProtocolError{
				State:  Sta6,
				Detail: fmt.Sprintf("command element %s value length %d exceeds %d bytes remaining", tag, vlen, r.Len()),
			}
		}
		value := make([]byte, vlen)
		if _, err := io.ReadFull(r, value); err != nil {
			return CommandSet{}, &ProtocolError{
				State:  Sta6,
				Detail: "command set truncated reading element value: " + err.Error(),
			}
		}
		cs.applyElement(tag, value)
	}
	return cs, nil
}

// applyElement folds one decoded element into the command set. Unmodelled tags are ignored.
func (cs *CommandSet) applyElement(tag dicom.Tag, value []byte) {
	switch tag {
	case tagAffectedSOPClassUID:
		cs.AffectedSOPClassUID = dicom.UID(decodeUI(value))
	case tagCommandField:
		cs.CommandField = CommandField(decodeUS(value))
	case tagMessageID:
		cs.MessageID = decodeUS(value)
	case tagMessageIDBeingRespondedTo:
		cs.MessageIDBeingRespondedTo = decodeUS(value)
	case tagMoveDestination:
		cs.MoveDestination = AETitle(decodeAE(value))
	case tagPriority:
		cs.HasPriority = true
		cs.Priority = Priority(decodeUS(value))
	case tagCommandDataSetType:
		cs.CommandDataSetType = decodeUS(value)
	case tagStatus:
		cs.HasStatus = true
		cs.Status = decodeUS(value)
	case tagAffectedSOPInstanceUID:
		cs.AffectedSOPInstanceUID = dicom.UID(decodeUI(value))
	}
}

// writeImplicitVRElement writes one element in Implicit VR Little Endian: group(2) element(2)
// length(4) value (PS3.5 §7.1.3).
func writeImplicitVRElement(w io.Writer, tag dicom.Tag, value []byte) error {
	var hdr [8]byte
	binary.LittleEndian.PutUint16(hdr[0:2], tag.Group())
	binary.LittleEndian.PutUint16(hdr[2:4], tag.Element())
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(value)))
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("dimse: write command element %s header: %w", tag, err)
	}
	if _, err := w.Write(value); err != nil {
		return fmt.Errorf("dimse: write command element %s value: %w", tag, err)
	}
	return nil
}

// encodeUS encodes a US (unsigned short) value as little-endian.
func encodeUS(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

// decodeUS decodes a US value, tolerating a short or absent value as zero.
func decodeUS(b []byte) uint16 {
	if len(b) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

// encodeUI encodes a UI (UID) value, padding to an even length with a trailing NUL as PS3.5
// §6.2 requires for UI values.
func encodeUI(s string) []byte {
	b := []byte(s)
	if len(b)%2 != 0 {
		b = append(b, 0x00)
	}
	return b
}

// decodeUI decodes a UI value, trimming the PS3.5 NUL/space padding.
func decodeUI(b []byte) string {
	return string(bytes.TrimRight(b, "\x00 "))
}

// encodeAE encodes an AE (Application Entity title), padding to an even length with a trailing
// space as PS3.5 §6.2 requires for AE values — never a NUL, which is the UI/UID pad (the DIMSE-007
// distinction: Move Destination is VR AE, so it is space-padded).
func encodeAE(s string) []byte {
	b := []byte(s)
	if len(b)%2 != 0 {
		b = append(b, ' ')
	}
	return b
}

// decodeAE decodes an AE value, trimming the PS3.5 leading/trailing space padding.
func decodeAE(b []byte) string {
	return string(bytes.TrimSpace(b))
}
