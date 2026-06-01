package pdu

import "io"

// ReleaseRQ is an A-RELEASE-RQ PDU (PS3.8 §9.3.6). Its body is four reserved bytes.
type ReleaseRQ struct{}

// ReleaseRP is an A-RELEASE-RP PDU (PS3.8 §9.3.7). Its body is four reserved bytes.
type ReleaseRP struct{}

var releaseReservedBody = [4]byte{}

// Encode writes the A-RELEASE-RQ PDU (header + four reserved bytes).
func (p *ReleaseRQ) Encode(w io.Writer) error {
	return encodeReserved4(w, PDUTypeReleaseRQ)
}

// Encode writes the A-RELEASE-RP PDU (header + four reserved bytes).
func (p *ReleaseRP) Encode(w io.Writer) error {
	return encodeReserved4(w, PDUTypeReleaseRP)
}

func encodeReserved4(w io.Writer, pt PDUType) error {
	if err := writeHeader(w, pt, 4); err != nil {
		return err
	}
	_, err := w.Write(releaseReservedBody[:])
	return err
}

// DecodeReleaseRQ reads an A-RELEASE-RQ PDU (header included), discarding the reserved
// body bounded by the declared length (PRD §9.3).
func DecodeReleaseRQ(r io.Reader) (*ReleaseRQ, error) {
	if err := discardReserved(r, PDUTypeReleaseRQ); err != nil {
		return nil, err
	}
	return &ReleaseRQ{}, nil
}

// DecodeReleaseRP reads an A-RELEASE-RP PDU (header included).
func DecodeReleaseRP(r io.Reader) (*ReleaseRP, error) {
	if err := discardReserved(r, PDUTypeReleaseRP); err != nil {
		return nil, err
	}
	return &ReleaseRP{}, nil
}

func discardReserved(r io.Reader, want PDUType) error {
	br, err := openPDUBody(r, want)
	if err != nil {
		return err
	}
	return discardReservedBody(br)
}

// discardReservedBody drains the reserved body from a seeded bounded reader (used by
// ReadPDU once it has already consumed the header).
func discardReservedBody(br *boundedReader) error {
	_, err := io.Copy(io.Discard, br)
	return err
}
