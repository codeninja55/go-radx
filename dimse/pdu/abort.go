package pdu

import "io"

// Abort is an A-ABORT PDU (PS3.8 §9.3.8): two reserved bytes, an abort source, and a
// reason/diagnostic.
type Abort struct {
	Source uint8
	Reason uint8
}

// Abort source values (PS3.8 §9.3.8).
const (
	AbortSourceServiceUser     uint8 = 0
	AbortSourceServiceProvider uint8 = 2
)

// Abort reason/diagnostic values. The not-specified reason applies to a service-user
// abort; the remaining values are the provider reasons the AA-8 path reports for an
// invalid or unexpected PDU (Codex DIMSE-011).
const (
	AbortReasonNotSpecified           uint8 = 0
	AbortReasonUnrecognizedPDU        uint8 = 1
	AbortReasonUnexpectedPDU          uint8 = 2
	AbortReasonUnexpectedPDUParameter uint8 = 4
	AbortReasonInvalidPDUParameter    uint8 = 5
)

// Encode writes the A-ABORT PDU (header + four body bytes: reserved, reserved, source,
// reason).
func (p *Abort) Encode(w io.Writer) error {
	body := []byte{0x00, 0x00, p.Source, p.Reason}
	if err := writeHeader(w, PDUTypeAbort, uint32(len(body))); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}

// DecodeAbort reads an A-ABORT PDU (header included) from r.
func DecodeAbort(r io.Reader) (*Abort, error) {
	br, err := openPDUBody(r, PDUTypeAbort)
	if err != nil {
		return nil, err
	}
	var b [4]byte
	if _, err := io.ReadFull(br, b[:]); err != nil {
		return nil, err
	}
	// b[0], b[1] reserved.
	return &Abort{Source: b[2], Reason: b[3]}, nil
}
