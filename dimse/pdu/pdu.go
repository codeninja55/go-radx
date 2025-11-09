package pdu

import (
	"encoding/binary"
	"fmt"
	"io"
)

// PDU types as defined in DICOM Part 8
const (
	PDUTypeAssociateRQ byte = 0x01
	PDUTypeAssociateAC byte = 0x02
	PDUTypeAssociateRJ byte = 0x03
	PDUTypeData        byte = 0x04
	PDUTypeReleaseRQ   byte = 0x05
	PDUTypeReleaseRP   byte = 0x06
	PDUTypeAbort       byte = 0x07
)

// Maximum PDU size constants
const (
	DefaultMaxPDULength = 16384    // 16KB default
	MaxPDULength        = 16777215 // 16MB maximum per DICOM standard
)

// Item types for sub-items in PDUs
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

// PDU represents a Protocol Data Unit
type PDU interface {
	Type() byte
	Encode(w io.Writer) error
	Decode(r io.Reader) error
}

// ReadPDU reads a PDU from the reader and returns the appropriate PDU type
func ReadPDU(r io.Reader) (PDU, error) {
	// Read PDU type (1 byte)
	var pduType byte
	if err := binary.Read(r, binary.BigEndian, &pduType); err != nil {
		return nil, fmt.Errorf("read PDU type: %w", err)
	}

	// Read reserved byte
	var reserved byte
	if err := binary.Read(r, binary.BigEndian, &reserved); err != nil {
		return nil, fmt.Errorf("read reserved byte: %w", err)
	}

	// Read PDU length (4 bytes)
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("read PDU length: %w", err)
	}

	if length > MaxPDULength {
		return nil, fmt.Errorf("PDU length %d exceeds maximum %d", length, MaxPDULength)
	}

	// Create appropriate PDU based on type
	var pdu PDU
	switch pduType {
	case PDUTypeAssociateRQ:
		pdu = &AssociateRQ{}
	case PDUTypeAssociateAC:
		pdu = &AssociateAC{}
	case PDUTypeAssociateRJ:
		pdu = &AssociateRJ{}
	case PDUTypeData:
		pdu = &DataTF{}
	case PDUTypeReleaseRQ:
		pdu = &ReleaseRQ{}
	case PDUTypeReleaseRP:
		pdu = &ReleaseRP{}
	case PDUTypeAbort:
		pdu = &Abort{}
	default:
		return nil, fmt.Errorf("unknown PDU type: 0x%02X", pduType)
	}

	// Read PDU body
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("read PDU body: %w", err)
	}

	// Decode PDU from body
	if err := pdu.Decode(io.NopCloser(newBytesReader(body))); err != nil {
		return nil, fmt.Errorf("decode PDU: %w", err)
	}

	return pdu, nil
}

// writePDUHeader writes the PDU type, reserved byte, and length
func writePDUHeader(w io.Writer, pduType byte, length uint32) error {
	if err := binary.Write(w, binary.BigEndian, pduType); err != nil {
		return fmt.Errorf("write PDU type: %w", err)
	}
	if err := binary.Write(w, binary.BigEndian, byte(0)); err != nil {
		return fmt.Errorf("write reserved byte: %w", err)
	}
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return fmt.Errorf("write PDU length: %w", err)
	}
	return nil
}

// Helper type for bytes.Reader that implements io.ReadCloser
type bytesReader struct {
	*io.LimitedReader
}

func newBytesReader(b []byte) *bytesReader {
	return &bytesReader{&io.LimitedReader{R: newReader(b), N: int64(len(b))}}
}

func (b *bytesReader) Close() error {
	return nil
}

// Simple bytes reader implementation
type simpleReader struct {
	data []byte
	pos  int
}

func newReader(data []byte) *simpleReader {
	return &simpleReader{data: data}
}

func (r *simpleReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// PadAETitle pads an AE title to 16 bytes with trailing spaces
func PadAETitle(title string) [16]byte {
	var result [16]byte
	copy(result[:], title)
	// Pad with spaces
	for i := len(title); i < 16; i++ {
		result[i] = ' '
	}
	return result
}

// TrimAETitle trims trailing spaces from an AE title
func TrimAETitle(title [16]byte) string {
	s := string(title[:])
	// Trim trailing spaces
	for s != "" && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
