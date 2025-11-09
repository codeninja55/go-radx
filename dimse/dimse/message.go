package dimse

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// Message represents a complete DIMSE message with command and optional dataset
type Message struct {
	CommandSet            *CommandSet
	DataSet               *dicom.DataSet
	PresentationContextID uint8
}

// Encode encodes the message into P-DATA-TF PDUs
func (m *Message) Encode(maxPDULength uint32) ([]*pdu.DataTF, error) {
	var pdus []*pdu.DataTF

	// Encode command set
	cmdDS, err := m.CommandSet.ToDataSet()
	if err != nil {
		return nil, fmt.Errorf("encode command set: %w", err)
	}

	// Serialize command dataset (always Implicit VR Little Endian)
	var cmdBuf bytes.Buffer
	if err := encodeDataSetImplicitVR(cmdDS, &cmdBuf); err != nil {
		return nil, fmt.Errorf("serialize command: %w", err)
	}

	// Check if there's a dataset to encode
	hasDataset := m.DataSet != nil

	// Fragment command into P-DATA PDUs
	// Don't mark as last if dataset follows
	cmdPDUs, err := fragmentData(cmdBuf.Bytes(), m.PresentationContextID, true, !hasDataset, maxPDULength)
	if err != nil {
		return nil, fmt.Errorf("fragment command: %w", err)
	}
	pdus = append(pdus, cmdPDUs...)

	// Encode dataset if present
	if hasDataset {
		var dsBuf bytes.Buffer
		if err := encodeDataSetImplicitVR(m.DataSet, &dsBuf); err != nil {
			return nil, fmt.Errorf("serialize dataset: %w", err)
		}

		// Fragment dataset into P-DATA PDUs
		// Dataset fragments are always last
		dsPDUs, err := fragmentData(dsBuf.Bytes(), m.PresentationContextID, false, true, maxPDULength)
		if err != nil {
			return nil, fmt.Errorf("fragment dataset: %w", err)
		}
		pdus = append(pdus, dsPDUs...)
	}

	return pdus, nil
}

// Decode reassembles a message from P-DATA-TF PDUs
func Decode(pdus []*pdu.DataTF) (*Message, error) {
	var commandData []byte
	var datasetData []byte
	var pcID uint8

	for _, dataPDU := range pdus {
		for _, item := range dataPDU.Items {
			if pcID == 0 {
				pcID = item.PresentationContextID
			}

			if item.IsCommand() {
				commandData = append(commandData, item.Data...)
			} else {
				datasetData = append(datasetData, item.Data...)
			}
		}
	}

	// Decode command set
	if len(commandData) == 0 {
		return nil, fmt.Errorf("no command data found in PDUs")
	}

	cmdDS, err := decodeDataSetImplicitVR(bytes.NewReader(commandData))
	if err != nil {
		return nil, fmt.Errorf("decode command dataset: %w", err)
	}

	cmdSet, err := FromDataSet(cmdDS)
	if err != nil {
		return nil, fmt.Errorf("parse command set: %w", err)
	}

	msg := &Message{
		CommandSet:            cmdSet,
		PresentationContextID: pcID,
	}

	// Decode dataset if present in PDUs
	// Check if there's any dataset fragment (not just command)
	hasDataset := false
	for _, dataPDU := range pdus {
		for _, item := range dataPDU.Items {
			if !item.IsCommand() {
				hasDataset = true
				break
			}
		}
		if hasDataset {
			break
		}
	}

	// If dataset fragments exist, decode them (even if zero bytes - creates empty DataSet)
	if hasDataset {
		ds, err := decodeDataSetImplicitVR(bytes.NewReader(datasetData))
		if err != nil {
			return nil, fmt.Errorf("decode dataset: %w", err)
		}
		msg.DataSet = ds
	}

	return msg, nil
}

// fragmentData fragments data into P-DATA-TF PDUs
// isCommand: true if this is command data, false if dataset data
// markAsLast: true if this is the last fragment of the entire DIMSE message
func fragmentData(data []byte, pcID uint8, isCommand, markAsLast bool, maxPDULength uint32) ([]*pdu.DataTF, error) {
	var pdus []*pdu.DataTF

	// Handle empty data - must still create a PDU with zero-length data
	// to indicate presence (per DICOM Part 7, a dataset marked as present
	// must have at least one P-DATA-TF PDU, even if empty)
	if len(data) == 0 {
		var controlHeader uint8
		if isCommand {
			controlHeader = pdu.MessageControlCommand
			if markAsLast {
				controlHeader |= pdu.MessageControlLastFragment
			}
		} else {
			// Dataset with zero length - only mark as last if this is truly the last fragment
			if markAsLast {
				controlHeader = pdu.MessageControlDatasetLast
			} else {
				controlHeader = pdu.MessageControlDataset
			}
		}

		pdv := pdu.PresentationDataValue{
			PresentationContextID: pcID,
			MessageControlHeader:  controlHeader,
			Data:                  []byte{}, // Empty data
		}

		dataPDU := &pdu.DataTF{
			Items: []pdu.PresentationDataValue{pdv},
		}

		return []*pdu.DataTF{dataPDU}, nil
	}

	// Calculate max data per PDU (subtract PDU header overhead)
	// P-DATA-TF header: 6 bytes (type + reserved + length)
	// PDV item header: 4 bytes (length) + 1 byte (context ID) + 1 byte (control header)
	maxDataPerPDU := int(maxPDULength) - 6 - 6

	offset := 0
	for offset < len(data) {
		remaining := len(data) - offset
		chunkSize := remaining
		if chunkSize > maxDataPerPDU {
			chunkSize = maxDataPerPDU
		}

		chunk := data[offset : offset+chunkSize]
		isLastChunk := offset+chunkSize >= len(data)

		// Set message control header
		var controlHeader uint8
		if isCommand {
			controlHeader = pdu.MessageControlCommand
			// Only mark as last if this is the last chunk AND markAsLast is true
			if isLastChunk && markAsLast {
				controlHeader |= pdu.MessageControlLastFragment
			}
		} else {
			// For dataset, mark as last only if this is the last chunk AND markAsLast is true
			if isLastChunk && markAsLast {
				controlHeader = pdu.MessageControlDatasetLast
			} else {
				controlHeader = pdu.MessageControlDataset
			}
		}

		pdv := pdu.PresentationDataValue{
			PresentationContextID: pcID,
			MessageControlHeader:  controlHeader,
			Data:                  chunk,
		}

		dataPDU := &pdu.DataTF{
			Items: []pdu.PresentationDataValue{pdv},
		}

		pdus = append(pdus, dataPDU)
		offset += chunkSize
	}

	return pdus, nil
}

// encodeDataSetImplicitVR encodes a dataset using Implicit VR Little Endian.
// This is used for DIMSE command sets which always use Implicit VR.
func encodeDataSetImplicitVR(ds *dicom.DataSet, w io.Writer) error {
	// DIMSE commands always use Implicit VR Little Endian per DICOM Part 7
	elements := ds.Elements()

	for _, elem := range elements {
		// Write tag (group, element) - 4 bytes
		if err := binary.Write(w, binary.LittleEndian, elem.Tag().Group); err != nil {
			return fmt.Errorf("failed to write tag group: %w", err)
		}
		if err := binary.Write(w, binary.LittleEndian, elem.Tag().Element); err != nil {
			return fmt.Errorf("failed to write tag element: %w", err)
		}

		// Write length - 4 bytes (Implicit VR always uses 4-byte length)
		valueBytes := elem.Value().Bytes()
		valueLength := uint32(len(valueBytes))
		if err := binary.Write(w, binary.LittleEndian, valueLength); err != nil {
			return fmt.Errorf("failed to write value length: %w", err)
		}

		// Write value bytes
		if len(valueBytes) > 0 {
			if _, err := w.Write(valueBytes); err != nil {
				return fmt.Errorf("failed to write value bytes: %w", err)
			}
		}
	}

	return nil
}

// decodeDataSetImplicitVR decodes a dataset using Implicit VR Little Endian.
// This is used for DIMSE command sets which always use Implicit VR.
func decodeDataSetImplicitVR(r io.Reader) (*dicom.DataSet, error) {
	// Create a DICOM reader with Little Endian byte order
	reader := dicom.NewReader(r, binary.LittleEndian)

	// Create transfer syntax for Implicit VR Little Endian
	ts := &dicom.TransferSyntax{
		UID:        "1.2.840.10008.1.2", // Implicit VR Little Endian
		ExplicitVR: false,
		ByteOrder:  binary.LittleEndian,
		Compressed: false,
		Deflated:   false,
	}

	// Create element parser
	parser := dicom.NewElementParser(reader, ts)

	// Create dataset to store elements
	ds := dicom.NewDataSet()

	// Read elements until EOF
	for {
		elem, err := parser.ReadElement()
		if err != nil {
			// Check for EOF (which may be wrapped in other errors)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			// Check if EOF is wrapped in the error chain
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, fmt.Errorf("failed to read element: %w", err)
		}

		if err := ds.Add(elem); err != nil {
			return nil, fmt.Errorf("failed to add element to dataset: %w", err)
		}
	}

	return ds, nil
}

// MessageReassembler helps reassemble fragmented DIMSE messages
type MessageReassembler struct {
	fragments map[uint8][]*pdu.DataTF // Keyed by presentation context ID
}

// NewMessageReassembler creates a new message reassembler
func NewMessageReassembler() *MessageReassembler {
	return &MessageReassembler{
		fragments: make(map[uint8][]*pdu.DataTF),
	}
}

// AddPDU adds a P-DATA-TF PDU to the reassembler
func (r *MessageReassembler) AddPDU(dataPDU *pdu.DataTF) (*Message, error) {
	if len(dataPDU.Items) == 0 {
		return nil, nil
	}

	// Get presentation context ID from first item
	pcID := dataPDU.Items[0].PresentationContextID

	// Add to fragments
	r.fragments[pcID] = append(r.fragments[pcID], dataPDU)

	// Check if we have a complete message (last fragment received)
	lastItem := dataPDU.Items[len(dataPDU.Items)-1]
	if lastItem.IsLastFragment() {
		// Reassemble message
		msg, err := Decode(r.fragments[pcID])
		if err != nil {
			return nil, err
		}

		// Clear fragments
		delete(r.fragments, pcID)

		return msg, nil
	}

	return nil, nil // Message not yet complete
}
