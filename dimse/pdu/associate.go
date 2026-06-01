package pdu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Item types for the variable-length sub-items inside association PDUs (PS3.8 §9.3.2).
const (
	ItemTypeApplicationContext     byte = 0x10
	ItemTypePresentationContextRQ  byte = 0x20
	ItemTypePresentationContextAC  byte = 0x21
	ItemTypeAbstractSyntax         byte = 0x30
	ItemTypeTransferSyntax         byte = 0x40
	ItemTypeUserInformation        byte = 0x50
	ItemTypeMaxLength              byte = 0x51
	ItemTypeImplementationClassUID byte = 0x52
	ItemTypeImplementationVersion  byte = 0x55
)

// Presentation-context negotiation results (PS3.8 §9.3.3.2).
const (
	PresentationContextAcceptance                   uint8 = 0
	PresentationContextUserRejection                uint8 = 1
	PresentationContextProviderRejection            uint8 = 2
	PresentationContextAbstractSyntaxNotSupported   uint8 = 3
	PresentationContextTransferSyntaxesNotSupported uint8 = 4
)

// A-ASSOCIATE-RJ result, source, and reason values (PS3.8 §9.3.4).
const (
	AssociateRJResultPermanent uint8 = 1
	AssociateRJResultTransient uint8 = 2

	AssociateRJSourceServiceUser                 uint8 = 1
	AssociateRJSourceServiceProviderACSE         uint8 = 2
	AssociateRJSourceServiceProviderPresentation uint8 = 3
)

// insignificantTransferSyntax is the placeholder transfer syntax a rejected
// presentation context still carries in an A-ASSOCIATE-AC: PS3.8 §9.3.3.2 requires
// exactly one (insignificant) transfer-syntax sub-item even when Result != acceptance.
// Implicit VR Little Endian is the conventional placeholder (Codex DIMSE-008).
const insignificantTransferSyntax = "1.2.840.10008.1.2"

// UserInformation carries the negotiated user-information sub-items (PS3.7 Annex D): the
// maximum PDU length, the implementation class UID, and the implementation version name.
type UserInformation struct {
	MaxPDULength           uint32
	ImplementationClassUID string
	ImplementationVersion  string
}

// PresentationContextRQ is one proposed presentation context in an A-ASSOCIATE-RQ.
type PresentationContextRQ struct {
	ID               uint8
	AbstractSyntax   string
	TransferSyntaxes []string
}

// PresentationContextAC is one negotiated presentation context in an A-ASSOCIATE-AC: a
// result and the single accepted (or, for a rejection, insignificant) transfer syntax.
type PresentationContextAC struct {
	ID             uint8
	Result         uint8
	TransferSyntax string
}

// AssociateRQ is an A-ASSOCIATE-RQ PDU (PS3.8 §9.3.2).
type AssociateRQ struct {
	ProtocolVersion      uint16
	CalledAETitle        [16]byte
	CallingAETitle       [16]byte
	ApplicationContext   string
	PresentationContexts []PresentationContextRQ
	UserInfo             UserInformation
}

// AssociateAC is an A-ASSOCIATE-AC PDU (PS3.8 §9.3.3).
type AssociateAC struct {
	ProtocolVersion      uint16
	CalledAETitle        [16]byte
	CallingAETitle       [16]byte
	ApplicationContext   string
	PresentationContexts []PresentationContextAC
	UserInfo             UserInformation
}

// AssociateRJ is an A-ASSOCIATE-RJ PDU (PS3.8 §9.3.4).
type AssociateRJ struct {
	Result uint8
	Source uint8
	Reason uint8
}

// Encode writes the A-ASSOCIATE-RQ PDU: the 6-byte header with the summed body length,
// then the fixed fields and variable sub-items.
func (p *AssociateRQ) Encode(w io.Writer) error {
	body := p.encodeBody()
	if err := writeHeader(w, PDUTypeAssociateRQ, uint32(body.Len())); err != nil {
		return err
	}
	_, err := w.Write(body.Bytes())
	return err
}

func (p *AssociateRQ) encodeBody() *bytes.Buffer {
	var buf bytes.Buffer
	writeAssociateFixedFields(&buf, p.ProtocolVersion, p.CalledAETitle, p.CallingAETitle)
	encodeItem(&buf, ItemTypeApplicationContext, []byte(p.ApplicationContext))
	for _, pc := range p.PresentationContexts {
		encodePresentationContextRQ(&buf, pc)
	}
	encodeUserInformation(&buf, p.UserInfo)
	return &buf
}

// Encode writes the A-ASSOCIATE-AC PDU.
func (p *AssociateAC) Encode(w io.Writer) error {
	var buf bytes.Buffer
	writeAssociateFixedFields(&buf, p.ProtocolVersion, p.CalledAETitle, p.CallingAETitle)
	encodeItem(&buf, ItemTypeApplicationContext, []byte(p.ApplicationContext))
	for _, pc := range p.PresentationContexts {
		encodePresentationContextAC(&buf, pc)
	}
	encodeUserInformation(&buf, p.UserInfo)
	if err := writeHeader(w, PDUTypeAssociateAC, uint32(buf.Len())); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Encode writes the A-ASSOCIATE-RJ PDU (a 4-byte body: reserved, result, source, reason).
func (p *AssociateRJ) Encode(w io.Writer) error {
	body := []byte{0x00, p.Result, p.Source, p.Reason}
	if err := writeHeader(w, PDUTypeAssociateRJ, uint32(len(body))); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}

// DecodeAssociateRQ reads an A-ASSOCIATE-RQ PDU (header included) from r, validating all
// sub-item lengths against the declared body length via a bounded reader (PRD §9.3).
func DecodeAssociateRQ(r io.Reader) (*AssociateRQ, error) {
	br, err := openPDUBody(r, PDUTypeAssociateRQ)
	if err != nil {
		return nil, err
	}
	return decodeAssociateRQBody(br)
}

// decodeAssociateRQBody decodes the A-ASSOCIATE-RQ body from a bounded reader already
// seeded with the declared PDU-body length (used by both DecodeAssociateRQ and ReadPDU).
func decodeAssociateRQBody(br *boundedReader) (*AssociateRQ, error) {
	p := &AssociateRQ{}
	if err := readAssociateFixedFields(br, &p.ProtocolVersion, &p.CalledAETitle, &p.CallingAETitle); err != nil {
		return nil, err
	}
	for br.Remaining() > 0 {
		itemType, data, err := readItem(br)
		if err != nil {
			return nil, err
		}
		switch itemType {
		case ItemTypeApplicationContext:
			p.ApplicationContext = string(data)
		case ItemTypePresentationContextRQ:
			pc, err := decodePresentationContextRQ(data)
			if err != nil {
				return nil, err
			}
			p.PresentationContexts = append(p.PresentationContexts, pc)
		case ItemTypeUserInformation:
			ui, err := decodeUserInformation(data)
			if err != nil {
				return nil, err
			}
			p.UserInfo = ui
		}
	}
	return p, nil
}

// DecodeAssociateAC reads an A-ASSOCIATE-AC PDU (header included) from r.
func DecodeAssociateAC(r io.Reader) (*AssociateAC, error) {
	br, err := openPDUBody(r, PDUTypeAssociateAC)
	if err != nil {
		return nil, err
	}
	return decodeAssociateACBody(br)
}

// decodeAssociateACBody decodes the A-ASSOCIATE-AC body from a seeded bounded reader.
func decodeAssociateACBody(br *boundedReader) (*AssociateAC, error) {
	p := &AssociateAC{}
	if err := readAssociateFixedFields(br, &p.ProtocolVersion, &p.CalledAETitle, &p.CallingAETitle); err != nil {
		return nil, err
	}
	for br.Remaining() > 0 {
		itemType, data, err := readItem(br)
		if err != nil {
			return nil, err
		}
		switch itemType {
		case ItemTypeApplicationContext:
			p.ApplicationContext = string(data)
		case ItemTypePresentationContextAC:
			pc, err := decodePresentationContextAC(data)
			if err != nil {
				return nil, err
			}
			p.PresentationContexts = append(p.PresentationContexts, pc)
		case ItemTypeUserInformation:
			ui, err := decodeUserInformation(data)
			if err != nil {
				return nil, err
			}
			p.UserInfo = ui
		}
	}
	return p, nil
}

// DecodeAssociateRJ reads an A-ASSOCIATE-RJ PDU (header included) from r.
func DecodeAssociateRJ(r io.Reader) (*AssociateRJ, error) {
	br, err := openPDUBody(r, PDUTypeAssociateRJ)
	if err != nil {
		return nil, err
	}
	return decodeAssociateRJBody(br)
}

// decodeAssociateRJBody decodes the A-ASSOCIATE-RJ body from a seeded bounded reader.
func decodeAssociateRJBody(br *boundedReader) (*AssociateRJ, error) {
	var b [4]byte
	if _, err := io.ReadFull(br, b[:]); err != nil {
		return nil, err
	}
	// b[0] is reserved.
	return &AssociateRJ{Result: b[1], Source: b[2], Reason: b[3]}, nil
}

// openPDUBody reads the 6-byte header, checks the type matches, and returns a bounded
// reader seeded with the declared body length so every sub-item length is validated
// against bytes actually present (PRD §9.3).
func openPDUBody(r io.Reader, want PDUType) (*boundedReader, error) {
	pt, length, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	if pt != want {
		return nil, &PDUError{Detail: fmt.Sprintf("expected %s, got %s", want, pt)}
	}
	return newBoundedReader(r, int64(length)), nil
}

func writeAssociateFixedFields(buf *bytes.Buffer, version uint16, called, calling [16]byte) {
	var hdr [4]byte
	binary.BigEndian.PutUint16(hdr[0:2], version)
	// hdr[2:4] reserved (0x0000).
	buf.Write(hdr[:])
	buf.Write(called[:])
	buf.Write(calling[:])
	buf.Write(make([]byte, 32)) // reserved
}

func readAssociateFixedFields(br *boundedReader, version *uint16, called, calling *[16]byte) error {
	var hdr [4]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return err
	}
	*version = binary.BigEndian.Uint16(hdr[0:2])
	if _, err := io.ReadFull(br, called[:]); err != nil {
		return err
	}
	if _, err := io.ReadFull(br, calling[:]); err != nil {
		return err
	}
	var reserved [32]byte
	_, err := io.ReadFull(br, reserved[:])
	return err
}

// encodeItem writes a sub-item: type byte, reserved byte, 2-byte big-endian length,
// data. It writes to a bytes.Buffer (whose Write never fails), so the sub-item layout
// composes without error plumbing on infallible writes.
func encodeItem(buf *bytes.Buffer, itemType byte, data []byte) {
	var hdr [4]byte
	hdr[0] = itemType
	binary.BigEndian.PutUint16(hdr[2:4], uint16(len(data)))
	buf.Write(hdr[:])
	buf.Write(data)
}

// readItem reads one sub-item, validating its declared length against the bounded
// reader's remaining bytes before allocation (PRD §9.3).
func readItem(br *boundedReader) (byte, []byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return 0, nil, err
	}
	length := int64(binary.BigEndian.Uint16(hdr[2:4]))
	if !br.CanRead(length) {
		return 0, nil, &PDUError{
			Detail: fmt.Sprintf("sub-item type 0x%02X length %d exceeds %d bytes remaining",
				hdr[0], length, br.Remaining()),
		}
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(br, data); err != nil {
		return 0, nil, err
	}
	return hdr[0], data, nil
}

func encodePresentationContextRQ(out *bytes.Buffer, pc PresentationContextRQ) {
	var buf bytes.Buffer
	buf.WriteByte(pc.ID)
	buf.Write([]byte{0x00, 0x00, 0x00}) // reserved
	encodeItem(&buf, ItemTypeAbstractSyntax, []byte(pc.AbstractSyntax))
	for _, ts := range pc.TransferSyntaxes {
		encodeItem(&buf, ItemTypeTransferSyntax, []byte(ts))
	}
	encodeItem(out, ItemTypePresentationContextRQ, buf.Bytes())
}

func decodePresentationContextRQ(data []byte) (PresentationContextRQ, error) {
	br := newBoundedReader(bytes.NewReader(data), int64(len(data)))
	var pc PresentationContextRQ
	var hdr [4]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return pc, err
	}
	pc.ID = hdr[0] // hdr[1:4] reserved
	for br.Remaining() > 0 {
		itemType, itemData, err := readItem(br)
		if err != nil {
			return pc, err
		}
		switch itemType {
		case ItemTypeAbstractSyntax:
			pc.AbstractSyntax = string(itemData)
		case ItemTypeTransferSyntax:
			pc.TransferSyntaxes = append(pc.TransferSyntaxes, string(itemData))
		}
	}
	return pc, nil
}

// encodePresentationContextAC encodes one negotiated context. A rejected context still
// carries exactly one (insignificant) transfer-syntax sub-item, as PS3.8 §9.3.3.2
// requires; the prototype omitted it for non-acceptance results (Codex DIMSE-008).
func encodePresentationContextAC(out *bytes.Buffer, pc PresentationContextAC) {
	var buf bytes.Buffer
	buf.WriteByte(pc.ID)
	buf.WriteByte(0x00)      // reserved
	buf.WriteByte(pc.Result) // result/reason
	buf.WriteByte(0x00)      // reserved
	ts := pc.TransferSyntax
	if ts == "" {
		ts = insignificantTransferSyntax
	}
	encodeItem(&buf, ItemTypeTransferSyntax, []byte(ts))
	encodeItem(out, ItemTypePresentationContextAC, buf.Bytes())
}

func decodePresentationContextAC(data []byte) (PresentationContextAC, error) {
	br := newBoundedReader(bytes.NewReader(data), int64(len(data)))
	var pc PresentationContextAC
	var hdr [4]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return pc, err
	}
	pc.ID = hdr[0]     // hdr[1] reserved
	pc.Result = hdr[2] // hdr[3] reserved
	for br.Remaining() > 0 {
		itemType, itemData, err := readItem(br)
		if err != nil {
			return pc, err
		}
		if itemType == ItemTypeTransferSyntax {
			pc.TransferSyntax = string(itemData)
		}
	}
	return pc, nil
}

func encodeUserInformation(out *bytes.Buffer, ui UserInformation) {
	var buf bytes.Buffer
	if ui.MaxPDULength > 0 {
		var lb [4]byte
		binary.BigEndian.PutUint32(lb[:], ui.MaxPDULength)
		encodeItem(&buf, ItemTypeMaxLength, lb[:])
	}
	if ui.ImplementationClassUID != "" {
		encodeItem(&buf, ItemTypeImplementationClassUID, []byte(ui.ImplementationClassUID))
	}
	if ui.ImplementationVersion != "" {
		encodeItem(&buf, ItemTypeImplementationVersion, []byte(ui.ImplementationVersion))
	}
	encodeItem(out, ItemTypeUserInformation, buf.Bytes())
}

func decodeUserInformation(data []byte) (UserInformation, error) {
	br := newBoundedReader(bytes.NewReader(data), int64(len(data)))
	var ui UserInformation
	for br.Remaining() > 0 {
		itemType, itemData, err := readItem(br)
		if err != nil {
			return ui, err
		}
		switch itemType {
		case ItemTypeMaxLength:
			if len(itemData) >= 4 {
				ui.MaxPDULength = binary.BigEndian.Uint32(itemData[:4])
			}
		case ItemTypeImplementationClassUID:
			ui.ImplementationClassUID = string(itemData)
		case ItemTypeImplementationVersion:
			ui.ImplementationVersion = string(itemData)
		}
	}
	return ui, nil
}

// padAETitle pads an AE title to the 16-byte DICOM field with trailing spaces.
func padAETitle(title string) [16]byte {
	var result [16]byte
	n := copy(result[:], title)
	for i := n; i < 16; i++ {
		result[i] = ' '
	}
	return result
}

// trimAETitle trims the trailing-space padding from a 16-byte AE-title field.
func trimAETitle(field [16]byte) string {
	s := string(field[:])
	for s != "" && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
