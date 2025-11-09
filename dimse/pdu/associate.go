package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

// AssociateRQ represents an A-ASSOCIATE-RQ PDU
type AssociateRQ struct {
	ProtocolVersion      uint16
	CalledAETitle        [16]byte
	CallingAETitle       [16]byte
	ApplicationContext   string
	PresentationContexts []PresentationContextRQ
	UserInfo             UserInformation
}

// PresentationContextRQ represents a presentation context in A-ASSOCIATE-RQ
type PresentationContextRQ struct {
	ID               uint8
	AbstractSyntax   string
	TransferSyntaxes []string
}

// AssociateAC represents an A-ASSOCIATE-AC PDU
type AssociateAC struct {
	ProtocolVersion      uint16
	CalledAETitle        [16]byte
	CallingAETitle       [16]byte
	ApplicationContext   string
	PresentationContexts []PresentationContextAC
	UserInfo             UserInformation
}

// PresentationContextAC represents a presentation context in A-ASSOCIATE-AC
type PresentationContextAC struct {
	ID             uint8
	Result         uint8
	TransferSyntax string
}

// Presentation context results
const (
	PresentationContextAcceptance                   uint8 = 0
	PresentationContextUserRejection                uint8 = 1
	PresentationContextProviderRejection            uint8 = 2
	PresentationContextAbstractSyntaxNotSupported   uint8 = 3
	PresentationContextTransferSyntaxesNotSupported uint8 = 4
)

// AssociateRJ represents an A-ASSOCIATE-RJ PDU
type AssociateRJ struct {
	Result uint8
	Source uint8
	Reason uint8
}

// Rejection results
const (
	AssociateRJResultPermanent uint8 = 1
	AssociateRJResultTransient uint8 = 2
)

// Rejection sources
const (
	AssociateRJSourceServiceUser                 uint8 = 1
	AssociateRJSourceServiceProvider             uint8 = 2
	AssociateRJSourceServiceProviderACSE         uint8 = 2
	AssociateRJSourceServiceProviderPresentation uint8 = 3
)

// UserInformation contains user information items
type UserInformation struct {
	MaxPDULength           uint32
	ImplementationClassUID string
	ImplementationVersion  string
}

// Type returns the PDU type
func (p *AssociateRQ) Type() byte {
	return PDUTypeAssociateRQ
}

// Encode writes the PDU to the writer
func (p *AssociateRQ) Encode(w io.Writer) error {
	var buf bytes.Buffer

	// Protocol version (2 bytes)
	if err := binary.Write(&buf, binary.BigEndian, p.ProtocolVersion); err != nil {
		return err
	}

	// Reserved (2 bytes)
	if err := binary.Write(&buf, binary.BigEndian, uint16(0)); err != nil {
		return err
	}

	// Called AE Title (16 bytes)
	if _, err := buf.Write(p.CalledAETitle[:]); err != nil {
		return err
	}

	// Calling AE Title (16 bytes)
	if _, err := buf.Write(p.CallingAETitle[:]); err != nil {
		return err
	}

	// Reserved (32 bytes)
	reserved := make([]byte, 32)
	if _, err := buf.Write(reserved); err != nil {
		return err
	}

	// Application Context Item
	if err := encodeItem(&buf, ItemTypeApplicationContext, []byte(p.ApplicationContext)); err != nil {
		return err
	}

	// Presentation Context Items
	for _, pc := range p.PresentationContexts {
		if err := encodePresentationContextRQ(&buf, pc); err != nil {
			return err
		}
	}

	// User Information Item
	if err := encodeUserInformation(&buf, p.UserInfo); err != nil {
		return err
	}

	// Write PDU header and body
	if err := writePDUHeader(w, PDUTypeAssociateRQ, uint32(buf.Len())); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Decode reads the PDU from the reader
func (p *AssociateRQ) Decode(r io.Reader) error {
	// Protocol version
	if err := binary.Read(r, binary.BigEndian, &p.ProtocolVersion); err != nil {
		return err
	}

	// Skip reserved bytes (2 bytes)
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return err
	}

	// Called AE Title
	if _, err := io.ReadFull(r, p.CalledAETitle[:]); err != nil {
		return err
	}

	// Calling AE Title
	if _, err := io.ReadFull(r, p.CallingAETitle[:]); err != nil {
		return err
	}

	// Skip reserved bytes (32 bytes)
	if _, err := io.CopyN(io.Discard, r, 32); err != nil {
		return err
	}

	// Read items
	for {
		itemType, itemData, err := readItem(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch itemType {
		case ItemTypeApplicationContext:
			p.ApplicationContext = string(itemData)
		case ItemTypePresentationContextRQ:
			pc, err := decodePresentationContextRQ(itemData)
			if err != nil {
				return err
			}
			p.PresentationContexts = append(p.PresentationContexts, pc)
		case ItemTypeUserInformation:
			ui, err := decodeUserInformation(itemData)
			if err != nil {
				return err
			}
			p.UserInfo = ui
		}
	}

	return nil
}

// Type returns the PDU type
func (p *AssociateAC) Type() byte {
	return PDUTypeAssociateAC
}

// Encode writes the PDU to the writer
func (p *AssociateAC) Encode(w io.Writer) error {
	var buf bytes.Buffer

	// Protocol version (2 bytes)
	if err := binary.Write(&buf, binary.BigEndian, p.ProtocolVersion); err != nil {
		return err
	}

	// Reserved (2 bytes)
	if err := binary.Write(&buf, binary.BigEndian, uint16(0)); err != nil {
		return err
	}

	// Called AE Title (16 bytes)
	if _, err := buf.Write(p.CalledAETitle[:]); err != nil {
		return err
	}

	// Calling AE Title (16 bytes)
	if _, err := buf.Write(p.CallingAETitle[:]); err != nil {
		return err
	}

	// Reserved (32 bytes)
	reserved := make([]byte, 32)
	if _, err := buf.Write(reserved); err != nil {
		return err
	}

	// Application Context Item
	if err := encodeItem(&buf, ItemTypeApplicationContext, []byte(p.ApplicationContext)); err != nil {
		return err
	}

	// Presentation Context Items
	for _, pc := range p.PresentationContexts {
		if err := encodePresentationContextAC(&buf, pc); err != nil {
			return err
		}
	}

	// User Information Item
	if err := encodeUserInformation(&buf, p.UserInfo); err != nil {
		return err
	}

	// Write PDU header and body
	if err := writePDUHeader(w, PDUTypeAssociateAC, uint32(buf.Len())); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Decode reads the PDU from the reader
func (p *AssociateAC) Decode(r io.Reader) error {
	// Same structure as AssociateRQ
	if err := binary.Read(r, binary.BigEndian, &p.ProtocolVersion); err != nil {
		return err
	}

	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return err
	}

	if _, err := io.ReadFull(r, p.CalledAETitle[:]); err != nil {
		return err
	}

	if _, err := io.ReadFull(r, p.CallingAETitle[:]); err != nil {
		return err
	}

	if _, err := io.CopyN(io.Discard, r, 32); err != nil {
		return err
	}

	for {
		itemType, itemData, err := readItem(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch itemType {
		case ItemTypeApplicationContext:
			p.ApplicationContext = string(itemData)
		case ItemTypePresentationContextAC:
			pc, err := decodePresentationContextAC(itemData)
			if err != nil {
				return err
			}
			p.PresentationContexts = append(p.PresentationContexts, pc)
		case ItemTypeUserInformation:
			ui, err := decodeUserInformation(itemData)
			if err != nil {
				return err
			}
			p.UserInfo = ui
		}
	}

	return nil
}

// Type returns the PDU type
func (p *AssociateRJ) Type() byte {
	return PDUTypeAssociateRJ
}

// Encode writes the PDU to the writer
func (p *AssociateRJ) Encode(w io.Writer) error {
	var buf bytes.Buffer

	// Reserved byte
	if err := binary.Write(&buf, binary.BigEndian, byte(0)); err != nil {
		return err
	}

	// Result
	if err := binary.Write(&buf, binary.BigEndian, p.Result); err != nil {
		return err
	}

	// Source
	if err := binary.Write(&buf, binary.BigEndian, p.Source); err != nil {
		return err
	}

	// Reason
	if err := binary.Write(&buf, binary.BigEndian, p.Reason); err != nil {
		return err
	}

	// Write PDU header and body
	if err := writePDUHeader(w, PDUTypeAssociateRJ, uint32(buf.Len())); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Decode reads the PDU from the reader
func (p *AssociateRJ) Decode(r io.Reader) error {
	// Skip reserved byte
	if _, err := io.CopyN(io.Discard, r, 1); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &p.Result); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &p.Source); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &p.Reason); err != nil {
		return err
	}

	return nil
}

// Helper functions for encoding/decoding items

func encodeItem(w io.Writer, itemType byte, data []byte) error {
	if err := binary.Write(w, binary.BigEndian, itemType); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, byte(0)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint16(len(data))); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func readItem(r io.Reader) (byte, []byte, error) {
	var itemType byte
	if err := binary.Read(r, binary.BigEndian, &itemType); err != nil {
		return 0, nil, err
	}

	var reserved byte
	if err := binary.Read(r, binary.BigEndian, &reserved); err != nil {
		return 0, nil, err
	}

	var length uint16
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return 0, nil, err
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return 0, nil, err
	}

	return itemType, data, nil
}

func encodePresentationContextRQ(w io.Writer, pc PresentationContextRQ) error {
	var buf bytes.Buffer

	// Presentation Context ID
	if err := binary.Write(&buf, binary.BigEndian, pc.ID); err != nil {
		return err
	}

	// Reserved (3 bytes)
	for i := 0; i < 3; i++ {
		if err := binary.Write(&buf, binary.BigEndian, byte(0)); err != nil {
			return err
		}
	}

	// Abstract Syntax
	if err := encodeItem(&buf, ItemTypeAbstractSyntax, []byte(pc.AbstractSyntax)); err != nil {
		return err
	}

	// Transfer Syntaxes
	for _, ts := range pc.TransferSyntaxes {
		if err := encodeItem(&buf, ItemTypeTransferSyntax, []byte(ts)); err != nil {
			return err
		}
	}

	return encodeItem(w, ItemTypePresentationContextRQ, buf.Bytes())
}

func decodePresentationContextRQ(data []byte) (PresentationContextRQ, error) {
	r := bytes.NewReader(data)
	var pc PresentationContextRQ

	if err := binary.Read(r, binary.BigEndian, &pc.ID); err != nil {
		return pc, err
	}

	// Skip reserved bytes
	if _, err := io.CopyN(io.Discard, r, 3); err != nil {
		return pc, err
	}

	for {
		itemType, itemData, err := readItem(r)
		if err == io.EOF {
			break
		}
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

func encodePresentationContextAC(w io.Writer, pc PresentationContextAC) error {
	var buf bytes.Buffer

	// Presentation Context ID
	if err := binary.Write(&buf, binary.BigEndian, pc.ID); err != nil {
		return err
	}

	// Reserved byte
	if err := binary.Write(&buf, binary.BigEndian, byte(0)); err != nil {
		return err
	}

	// Result
	if err := binary.Write(&buf, binary.BigEndian, pc.Result); err != nil {
		return err
	}

	// Reserved byte
	if err := binary.Write(&buf, binary.BigEndian, byte(0)); err != nil {
		return err
	}

	// Transfer Syntax (only if accepted)
	if pc.Result == PresentationContextAcceptance {
		if err := encodeItem(&buf, ItemTypeTransferSyntax, []byte(pc.TransferSyntax)); err != nil {
			return err
		}
	}

	return encodeItem(w, ItemTypePresentationContextAC, buf.Bytes())
}

func decodePresentationContextAC(data []byte) (PresentationContextAC, error) {
	r := bytes.NewReader(data)
	var pc PresentationContextAC

	if err := binary.Read(r, binary.BigEndian, &pc.ID); err != nil {
		return pc, err
	}

	// Skip reserved byte
	if _, err := io.CopyN(io.Discard, r, 1); err != nil {
		return pc, err
	}

	if err := binary.Read(r, binary.BigEndian, &pc.Result); err != nil {
		return pc, err
	}

	// Skip reserved byte
	if _, err := io.CopyN(io.Discard, r, 1); err != nil {
		return pc, err
	}

	// Transfer syntax (if present)
	for {
		itemType, itemData, err := readItem(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return pc, err
		}

		if itemType == ItemTypeTransferSyntax {
			pc.TransferSyntax = string(itemData)
		}
	}

	return pc, nil
}

func encodeUserInformation(w io.Writer, ui UserInformation) error {
	var buf bytes.Buffer

	// Max PDU Length
	if ui.MaxPDULength > 0 {
		var lengthBuf bytes.Buffer
		if err := binary.Write(&lengthBuf, binary.BigEndian, ui.MaxPDULength); err != nil {
			return err
		}
		if err := encodeItem(&buf, ItemTypeMaxLength, lengthBuf.Bytes()); err != nil {
			return err
		}
	}

	// Implementation Class UID
	if ui.ImplementationClassUID != "" {
		if err := encodeItem(&buf, ItemTypeImplementationClassUID, []byte(ui.ImplementationClassUID)); err != nil {
			return err
		}
	}

	// Implementation Version
	if ui.ImplementationVersion != "" {
		if err := encodeItem(&buf, ItemTypeImplementationVersion, []byte(ui.ImplementationVersion)); err != nil {
			return err
		}
	}

	return encodeItem(w, ItemTypeUserInformation, buf.Bytes())
}

func decodeUserInformation(data []byte) (UserInformation, error) {
	r := bytes.NewReader(data)
	var ui UserInformation

	for {
		itemType, itemData, err := readItem(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return ui, err
		}

		switch itemType {
		case ItemTypeMaxLength:
			if err := binary.Read(bytes.NewReader(itemData), binary.BigEndian, &ui.MaxPDULength); err != nil {
				return ui, err
			}
		case ItemTypeImplementationClassUID:
			ui.ImplementationClassUID = string(itemData)
		case ItemTypeImplementationVersion:
			ui.ImplementationVersion = string(itemData)
		}
	}

	return ui, nil
}
