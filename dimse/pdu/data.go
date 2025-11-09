package pdu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrPDUTooLarge is returned when a PDU or PDU item exceeds the maximum allowed size
	ErrPDUTooLarge = errors.New("PDU or item length exceeds maximum allowed size")
)

// DataTF represents a P-DATA-TF PDU
type DataTF struct {
	Items []PresentationDataValue
}

// PresentationDataValue represents a presentation data value item
type PresentationDataValue struct {
	PresentationContextID uint8
	MessageControlHeader  uint8
	Data                  []byte
}

// Message control header flags
const (
	MessageControlCommand      uint8 = 0x01 // Message is command
	MessageControlLastFragment uint8 = 0x02 // Last fragment
	MessageControlDataset      uint8 = 0x00 // Message is dataset, more fragments
	MessageControlDatasetLast  uint8 = 0x02 // Dataset, last fragment
)

// Type returns the PDU type
func (p *DataTF) Type() byte {
	return PDUTypeData
}

// Encode writes the PDU to the writer
func (p *DataTF) Encode(w io.Writer) error {
	var buf bytes.Buffer

	// Encode presentation data value items
	for _, item := range p.Items {
		if err := encodePresentationDataValue(&buf, item); err != nil {
			return err
		}
	}

	// Write PDU header and body
	if err := writePDUHeader(w, PDUTypeData, uint32(buf.Len())); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Decode reads the PDU from the reader
func (p *DataTF) Decode(r io.Reader) error {
	// Read all presentation data value items until EOF
	for {
		item, err := decodePresentationDataValue(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		p.Items = append(p.Items, item)
	}
	return nil
}

func encodePresentationDataValue(w io.Writer, pdv PresentationDataValue) error {
	// Item length (4 bytes) = 1 (context ID) + 1 (control header) + data length
	itemLength := uint32(1 + 1 + len(pdv.Data))
	if err := binary.Write(w, binary.BigEndian, itemLength); err != nil {
		return err
	}

	// Presentation Context ID
	if err := binary.Write(w, binary.BigEndian, pdv.PresentationContextID); err != nil {
		return err
	}

	// Message Control Header
	if err := binary.Write(w, binary.BigEndian, pdv.MessageControlHeader); err != nil {
		return err
	}

	// Data
	_, err := w.Write(pdv.Data)
	return err
}

func decodePresentationDataValue(r io.Reader) (PresentationDataValue, error) {
	var pdv PresentationDataValue

	// Item length
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return pdv, err
	}

	// Validate item length to prevent DoS via memory exhaustion
	if length > MaxPDULength {
		return pdv, fmt.Errorf("PDV item length %d exceeds maximum %d: %w", length, MaxPDULength, ErrPDUTooLarge)
	}

	// Presentation Context ID
	if err := binary.Read(r, binary.BigEndian, &pdv.PresentationContextID); err != nil {
		return pdv, err
	}

	// Message Control Header
	if err := binary.Read(r, binary.BigEndian, &pdv.MessageControlHeader); err != nil {
		return pdv, err
	}

	// Data (length - 2 bytes for context ID and control header)
	dataLength := length - 2
	pdv.Data = make([]byte, dataLength)
	if _, err := io.ReadFull(r, pdv.Data); err != nil {
		return pdv, err
	}

	return pdv, nil
}

// IsCommand returns true if the PDV contains a command
func (pdv *PresentationDataValue) IsCommand() bool {
	return pdv.MessageControlHeader&MessageControlCommand != 0
}

// IsLastFragment returns true if this is the last fragment
func (pdv *PresentationDataValue) IsLastFragment() bool {
	return pdv.MessageControlHeader&MessageControlLastFragment != 0
}
