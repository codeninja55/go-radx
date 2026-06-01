package pdu

import (
	"encoding/binary"
	"fmt"
	"io"
)

// PDUType is the one-byte PDU type discriminator (PS3.8 §9.3.1, Table 9-11).
type PDUType byte

const (
	PDUTypeAssociateRQ PDUType = 0x01
	PDUTypeAssociateAC PDUType = 0x02
	PDUTypeAssociateRJ PDUType = 0x03
	PDUTypeData        PDUType = 0x04 // P-DATA-TF
	PDUTypeReleaseRQ   PDUType = 0x05
	PDUTypeReleaseRP   PDUType = 0x06
	PDUTypeAbort       PDUType = 0x07
)

// MaxPDULength is the absolute ceiling the codec accepts for a PDU body length,
// independent of the association-negotiated Maximum Length. It bounds allocation
// against a hostile declared length (PRD §9.3): because the bounded reader is seeded
// from the declared length read off the wire, that length is not proof that bytes
// are present, so the remaining-bytes check alone cannot cap a buffer — a declared
// length and any single PDV payload above this ceiling are rejected before
// allocation. The association layer enforces the tighter negotiated maximum on top.
const MaxPDULength uint32 = 0x00FFFFFF // 16 MiB - 1

var pduTypeNames = map[PDUType]string{
	PDUTypeAssociateRQ: "A-ASSOCIATE-RQ",
	PDUTypeAssociateAC: "A-ASSOCIATE-AC",
	PDUTypeAssociateRJ: "A-ASSOCIATE-RJ",
	PDUTypeData:        "P-DATA-TF",
	PDUTypeReleaseRQ:   "A-RELEASE-RQ",
	PDUTypeReleaseRP:   "A-RELEASE-RP",
	PDUTypeAbort:       "A-ABORT",
}

// String renders the registered PDU name, never bare hex (PRD §8.2).
func (pt PDUType) String() string {
	if name, ok := pduTypeNames[pt]; ok {
		return name
	}
	return fmt.Sprintf("unknown-PDU(0x%02X)", byte(pt))
}

func (pt PDUType) valid() bool {
	_, ok := pduTypeNames[pt]
	return ok
}

// writeHeader writes the PDU type, the reserved byte (0x00), and the big-endian
// 4-byte body length (PS3.8 §9.3.1). The body follows; this writes only the header.
func writeHeader(w io.Writer, pt PDUType, length uint32) error {
	var h [6]byte
	h[0] = byte(pt)
	h[1] = 0x00
	binary.BigEndian.PutUint32(h[2:], length)
	_, err := w.Write(h[:])
	if err != nil {
		return fmt.Errorf("pdu: write %s header: %w", pt, err)
	}
	return nil
}

// readHeader reads and validates a 6-byte PDU header, returning the type and the
// declared body length. An unknown type is rejected; a short read surfaces as
// io.ErrUnexpectedEOF (truncation is failure, PRD §9.2).
func readHeader(r io.Reader) (PDUType, uint32, error) {
	var h [6]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		if err == io.EOF {
			return 0, 0, err // clean EOF at a PDU boundary
		}
		return 0, 0, fmt.Errorf("pdu: read header: %w", err)
	}
	pt := PDUType(h[0])
	if !pt.valid() {
		return 0, 0, fmt.Errorf("pdu: unrecognised PDU type 0x%02X", h[0])
	}
	length := binary.BigEndian.Uint32(h[2:])
	if length > MaxPDULength {
		return 0, 0, fmt.Errorf("pdu: %s declared body length %d exceeds maximum %d", pt, length, MaxPDULength)
	}
	return pt, length, nil
}
