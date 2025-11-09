package pdu

import (
	"encoding/binary"
	"io"
)

// Abort represents an A-ABORT PDU
type Abort struct {
	Source uint8
	Reason uint8
}

// Abort sources
const (
	AbortSourceServiceUser     uint8 = 0
	AbortSourceServiceProvider uint8 = 2
)

// Abort reasons (service-user)
const (
	AbortReasonNotSpecified uint8 = 0
)

// Abort reasons (service-provider)
const (
	AbortReasonUnrecognizedPDU        uint8 = 1
	AbortReasonUnexpectedPDU          uint8 = 2
	AbortReasonUnexpectedPDUParameter uint8 = 4
	AbortReasonInvalidPDUParameter    uint8 = 5
)

// Type returns the PDU type
func (p *Abort) Type() byte {
	return PDUTypeAbort
}

// Encode writes the PDU to the writer
func (p *Abort) Encode(w io.Writer) error {
	// Write PDU header (length is always 4 bytes)
	if err := writePDUHeader(w, PDUTypeAbort, 4); err != nil {
		return err
	}

	// Reserved byte (1 byte)
	if err := binary.Write(w, binary.BigEndian, byte(0)); err != nil {
		return err
	}

	// Reserved byte (1 byte)
	if err := binary.Write(w, binary.BigEndian, byte(0)); err != nil {
		return err
	}

	// Source
	if err := binary.Write(w, binary.BigEndian, p.Source); err != nil {
		return err
	}

	// Reason
	if err := binary.Write(w, binary.BigEndian, p.Reason); err != nil {
		return err
	}

	return nil
}

// Decode reads the PDU from the reader
func (p *Abort) Decode(r io.Reader) error {
	// Skip 2 reserved bytes
	if _, err := io.CopyN(io.Discard, r, 2); err != nil {
		return err
	}

	// Source
	if err := binary.Read(r, binary.BigEndian, &p.Source); err != nil {
		return err
	}

	// Reason
	if err := binary.Read(r, binary.BigEndian, &p.Reason); err != nil {
		return err
	}

	return nil
}
