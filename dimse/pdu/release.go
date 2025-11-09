package pdu

import (
	"io"
)

// ReleaseRQ represents an A-RELEASE-RQ PDU
type ReleaseRQ struct {
	// No fields - just reserved bytes
}

// ReleaseRP represents an A-RELEASE-RP PDU
type ReleaseRP struct {
	// No fields - just reserved bytes
}

// Type returns the PDU type
func (p *ReleaseRQ) Type() byte {
	return PDUTypeReleaseRQ
}

// Encode writes the PDU to the writer
func (p *ReleaseRQ) Encode(w io.Writer) error {
	// Reserved bytes (4 bytes total for the body)
	reserved := make([]byte, 4)

	// Write PDU header
	if err := writePDUHeader(w, PDUTypeReleaseRQ, 4); err != nil {
		return err
	}

	// Write reserved bytes
	_, err := w.Write(reserved)
	return err
}

// Decode reads the PDU from the reader
func (p *ReleaseRQ) Decode(r io.Reader) error {
	// Skip 4 reserved bytes
	_, err := io.CopyN(io.Discard, r, 4)
	return err
}

// Type returns the PDU type
func (p *ReleaseRP) Type() byte {
	return PDUTypeReleaseRP
}

// Encode writes the PDU to the writer
func (p *ReleaseRP) Encode(w io.Writer) error {
	// Reserved bytes (4 bytes total for the body)
	reserved := make([]byte, 4)

	// Write PDU header
	if err := writePDUHeader(w, PDUTypeReleaseRP, 4); err != nil {
		return err
	}

	// Write reserved bytes
	_, err := w.Write(reserved)
	return err
}

// Decode reads the PDU from the reader
func (p *ReleaseRP) Decode(r io.Reader) error {
	// Skip 4 reserved bytes
	_, err := io.CopyN(io.Discard, r, 4)
	return err
}
