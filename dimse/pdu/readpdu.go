package pdu

import "io"

// PDU is any DICOM Upper Layer Protocol Data Unit that can frame itself onto the wire.
// The dul package reads and writes whole PDUs through ReadPDU/WritePDU without touching
// the unexported header codec, keeping the layering acyclic.
type PDU interface {
	// Type reports the PDU type discriminator.
	Type() PDUType
	// Encode writes the complete PDU (6-byte header plus body) to w.
	Encode(w io.Writer) error
}

// Type methods make each PDU satisfy the PDU interface.
func (p *AssociateRQ) Type() PDUType { return PDUTypeAssociateRQ }
func (p *AssociateAC) Type() PDUType { return PDUTypeAssociateAC }
func (p *AssociateRJ) Type() PDUType { return PDUTypeAssociateRJ }
func (p *DataTF) Type() PDUType      { return PDUTypeData }
func (p *ReleaseRQ) Type() PDUType   { return PDUTypeReleaseRQ }
func (p *ReleaseRP) Type() PDUType   { return PDUTypeReleaseRP }
func (p *Abort) Type() PDUType       { return PDUTypeAbort }

// WritePDU encodes a single PDU onto w. It is the dul layer's only write entry point.
func WritePDU(w io.Writer, p PDU) error {
	return p.Encode(w)
}

// ReadPDU reads one complete PDU from r: it consumes the 6-byte header (validating the
// type and capping the declared length at MaxPDULength), seeds a bounded reader with the
// declared body length, and decodes the matching concrete PDU. Every body decode reads
// only within the bound, so a hostile declared length cannot over-read or over-allocate
// (PRD §9.3). A clean io.EOF at a PDU boundary is returned verbatim so the caller can
// detect an orderly transport close.
func ReadPDU(r io.Reader) (PDU, error) {
	pt, length, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	br := newBoundedReader(r, int64(length))
	switch pt {
	case PDUTypeAssociateRQ:
		return decodeAssociateRQBody(br)
	case PDUTypeAssociateAC:
		return decodeAssociateACBody(br)
	case PDUTypeAssociateRJ:
		return decodeAssociateRJBody(br)
	case PDUTypeData:
		d := &DataTF{}
		if err := d.Decode(br); err != nil {
			return nil, err
		}
		return d, nil
	case PDUTypeReleaseRQ:
		if err := discardReservedBody(br, PDUTypeReleaseRQ); err != nil {
			return nil, err
		}
		return &ReleaseRQ{}, nil
	case PDUTypeReleaseRP:
		if err := discardReservedBody(br, PDUTypeReleaseRP); err != nil {
			return nil, err
		}
		return &ReleaseRP{}, nil
	case PDUTypeAbort:
		return decodeAbortBody(br)
	default:
		// readHeader already rejects unknown types; this is a defensive fall-through.
		return nil, &PDUError{Detail: "unhandled PDU type " + pt.String()}
	}
}
