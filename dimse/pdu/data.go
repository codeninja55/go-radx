package pdu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Message-control header bits (PS3.8 §9.3.5.1). Bit 0 is the command/dataset bit;
// bit 1 is the last-fragment bit. They are independent: a final command fragment
// is 0x03 whether or not a dataset follows (Codex DIMSE-001).
const (
	controlCommandBit byte = 0x01 // bit 0: 1 = command, 0 = dataset
	controlLastBit    byte = 0x02 // bit 1: 1 = last fragment of this command/dataset
)

// PresentationDataValue is one PDV inside a P-DATA-TF PDU: a presentation context
// ID, a one-byte message-control header, and the fragment payload.
type PresentationDataValue struct {
	PresentationContextID uint8
	MessageControlHeader  byte
	Data                  []byte
}

// IsCommand reports whether the PDV carries command-set bytes (bit 0 set).
func (p PresentationDataValue) IsCommand() bool { return p.MessageControlHeader&controlCommandBit != 0 }

// IsLastFragment reports whether this is the last fragment of its command or
// dataset (bit 1 set).
func (p PresentationDataValue) IsLastFragment() bool {
	return p.MessageControlHeader&controlLastBit != 0
}

// MakeControlHeader composes a message-control header from the two independent
// bits. The DIMSE message layer (Increment 5) uses this so the final command
// fragment is always 0x03 and the final dataset fragment 0x02.
func MakeControlHeader(command, last bool) byte {
	var h byte
	if command {
		h |= controlCommandBit
	}
	if last {
		h |= controlLastBit
	}
	return h
}

// pdvHeaderLen is the PDV item-header size counted inside the item length: the
// 1-byte presentation-context ID plus the 1-byte message-control header.
const pdvHeaderLen = 2

// encodePDV writes one PDV item: a 4-byte big-endian item length (header + payload),
// the context ID, the message-control header, then the payload. The dimse sender
// fragments payloads to the negotiated send cap, but Encode is public, so an
// over-length payload from a direct caller is refused rather than emitted with a
// wrapped 32-bit item length.
func encodePDV(w io.Writer, pdv PresentationDataValue) error {
	if int64(len(pdv.Data)) > int64(math.MaxUint32)-pdvHeaderLen {
		return &EncodeError{Detail: fmt.Sprintf("PDV payload is %d bytes, exceeds the 32-bit item length field", len(pdv.Data))}
	}
	itemLen := uint32(pdvHeaderLen + len(pdv.Data)) // #nosec G115 -- bounded by the check above
	var hdr [6]byte
	binary.BigEndian.PutUint32(hdr[0:4], itemLen)
	hdr[4] = pdv.PresentationContextID
	hdr[5] = pdv.MessageControlHeader
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("pdu: write PDV header: %w", err)
	}
	if _, err := w.Write(pdv.Data); err != nil {
		return fmt.Errorf("pdu: write PDV payload: %w", err)
	}
	return nil
}

// decodePDV reads one PDV item from a bounded reader. It rejects an item length
// below the 2-byte header BEFORE subtracting (Codex DIMSE-004) and validates the
// payload length against the bytes remaining before allocation (PRD §9.3).
func decodePDV(br *boundedReader) (PresentationDataValue, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(br, lenBuf[:]); err != nil {
		return PresentationDataValue{}, err
	}
	itemLen := binary.BigEndian.Uint32(lenBuf[:])
	if itemLen < pdvHeaderLen {
		return PresentationDataValue{}, &PDUError{
			Detail: fmt.Sprintf("PDV item length %d below header size %d", itemLen, pdvHeaderLen),
		}
	}
	payloadLen := int64(itemLen - pdvHeaderLen)
	// Cap the payload against the absolute maximum before allocating: the bounded
	// reader's remaining count is seeded from the declared (attacker-controlled) PDU
	// length, so CanRead alone does not bound the allocation (the remaining-bytes
	// check and the absolute-length check are distinct guards). PRD §9.3.
	if payloadLen > int64(MaxPDULength) {
		return PresentationDataValue{}, &PDUError{
			Detail: fmt.Sprintf("PDV payload length %d exceeds maximum PDU length %d", payloadLen, MaxPDULength),
		}
	}
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return PresentationDataValue{}, err
	}
	if !br.CanRead(payloadLen) {
		return PresentationDataValue{}, &PDUError{
			Detail: fmt.Sprintf("PDV payload length %d exceeds %d bytes remaining in PDU body",
				payloadLen, br.Remaining()),
		}
	}
	data := make([]byte, payloadLen)
	if _, err := io.ReadFull(br, data); err != nil {
		return PresentationDataValue{}, err
	}
	return PresentationDataValue{
		PresentationContextID: hdr[0],
		MessageControlHeader:  hdr[1],
		Data:                  data,
	}, nil
}

// DataTF is a P-DATA-TF PDU: one or more PDV items carrying command and dataset
// fragments (PS3.8 §9.3.5).
type DataTF struct {
	Items []PresentationDataValue
}

// Encode writes the P-DATA-TF PDU: the 6-byte header (with the summed item length)
// followed by each PDV item.
func (p *DataTF) Encode(w io.Writer) error {
	var body bytes.Buffer
	for _, item := range p.Items {
		if err := encodePDV(&body, item); err != nil {
			return err
		}
	}
	if int64(body.Len()) > int64(math.MaxUint32) {
		return &EncodeError{Detail: fmt.Sprintf("p-data-tf body is %d bytes, exceeds the 32-bit PDU length field", body.Len())}
	}
	if err := writeHeader(w, PDUTypeData, uint32(body.Len())); err != nil { // #nosec G115 -- bounded by the check above
		return err
	}
	_, err := w.Write(body.Bytes())
	return err
}

// Decode reads PDV items from a bounded reader seeded with the PDU body length, so
// the items cannot collectively exceed the declared body (PRD §9.3).
func (p *DataTF) Decode(br *boundedReader) error {
	for br.Remaining() > 0 {
		item, err := decodePDV(br)
		if err != nil {
			return err
		}
		p.Items = append(p.Items, item)
	}
	// A P-DATA-TF PDU body must carry one or more PDV items (PS3.8 §9.3.5); an empty
	// body is a malformed PDU, not an empty success.
	if len(p.Items) == 0 {
		return &PDUError{Detail: "P-DATA-TF PDU must contain at least one PDV item"}
	}
	return nil
}
