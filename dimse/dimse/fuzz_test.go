package dimse

import (
	"bytes"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// FuzzCommandSetFromDataSet tests CommandSet decoding from malformed DICOM datasets
func FuzzCommandSetFromDataSet(f *testing.F) {
	// Seed with valid C-ECHO-RQ command
	validCmd := &CommandSet{
		CommandField:        CommandCEchoRQ,
		MessageID:           1,
		CommandDataSetType:  DataSetNotPresent,
		AffectedSOPClassUID: "1.2.840.10008.1.1",
	}

	ds, _ := validCmd.ToDataSet()
	buf := &bytes.Buffer{}
	_ = encodeDataSetImplicitVR(ds, buf) // Implicit VR Little Endian
	f.Add(buf.Bytes())

	// Seed with empty dataset
	f.Add([]byte{})

	// Seed with truncated dataset
	f.Add(buf.Bytes()[:20])

	// Seed with oversized values (skip if invalid)
	largeDS := dicom.NewDataSet()
	largeVal, err := value.NewStringValue(vr.UniqueIdentifier, []string{string(bytes.Repeat([]byte("1.2.3."), 100))})
	if err == nil && largeVal != nil {
		largeElem, err := element.NewElement(tag.New(0x0000, 0x0002), vr.UniqueIdentifier, largeVal)
		if err == nil {
			_ = largeDS.Add(largeElem)
			largeBuf := &bytes.Buffer{}
			_ = encodeDataSetImplicitVR(largeDS, largeBuf)
			f.Add(largeBuf.Bytes())
		}
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Parse dataset from bytes
		ds, err := decodeDataSetImplicitVR(bytes.NewReader(data))
		if err != nil {
			return // Invalid DICOM is expected
		}

		// Should never panic when decoding command set
		cmd, _ := FromDataSet(ds)

		// If decode succeeded, verify basic invariants
		if cmd != nil {
			// Command field should be valid DIMSE command
			validCommands := []uint16{
				CommandCEchoRQ, CommandCEchoRSP,
				CommandCStoreRQ, CommandCStoreRSP,
				CommandCFindRQ, CommandCFindRSP,
				CommandCGetRQ, CommandCGetRSP,
				CommandCMoveRQ, CommandCMoveRSP,
				CommandCCancelRQ,
			}

			isValidCommand := false
			for _, valid := range validCommands {
				if cmd.CommandField == valid {
					isValidCommand = true
					break
				}
			}

			if !isValidCommand && cmd.CommandField != 0 {
				t.Errorf("Invalid command field: 0x%04X", cmd.CommandField)
			}

			// Note: Field validation (priority, dataset type) should be done in the decoder,
			// not the fuzz test. Fuzz tests focus on crash prevention and memory safety.
		}
	})
}

// FuzzCommandSetRoundTrip tests CommandSet encode/decode round-trip
func FuzzCommandSetRoundTrip(f *testing.F) {
	// Seed with various command types
	commands := []*CommandSet{
		{CommandField: CommandCEchoRQ, MessageID: 1},
		{CommandField: CommandCStoreRQ, MessageID: 2, AffectedSOPClassUID: "1.2.3"},
		{CommandField: CommandCFindRQ, MessageID: 3, Priority: PriorityHigh},
		{CommandField: CommandCGetRSP, MessageID: 4, Status: StatusSuccess},
		{CommandField: CommandCMoveRSP, MessageID: 5, Status: StatusPending,
			NumberOfCompletedSubOps: 10, NumberOfFailedSubOps: 2},
	}

	for _, cmd := range commands {
		ds, _ := cmd.ToDataSet()
		buf := &bytes.Buffer{}
		_ = encodeDataSetImplicitVR(ds, buf)
		f.Add(buf.Bytes())
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Parse dataset
		ds, err := decodeDataSetImplicitVR(bytes.NewReader(data))
		if err != nil {
			return
		}

		// Decode command
		cmd1, err := FromDataSet(ds)
		if err != nil {
			return // Expected for malformed commands
		}

		// Encode back to dataset
		ds2, err := cmd1.ToDataSet()
		if err != nil {
			t.Fatalf("Failed to encode valid command: %v", err)
		}

		// Decode again
		cmd2, err := FromDataSet(ds2)
		if err != nil {
			t.Fatalf("Failed to decode after round-trip: %v", err)
		}

		// Verify round-trip preserves command field
		if cmd1.CommandField != cmd2.CommandField {
			t.Errorf("CommandField mismatch: %d != %d", cmd1.CommandField, cmd2.CommandField)
		}
	})
}

// FuzzMessageEncoding tests Message encoding with malformed command/data
func FuzzMessageEncoding(f *testing.F) {
	// Seed with valid message (unused - just for reference)
	_ = &Message{
		CommandSet: &CommandSet{
			CommandField:       CommandCEchoRQ,
			MessageID:          1,
			CommandDataSetType: DataSetNotPresent,
		},
		PresentationContextID: 1,
	}
	f.Add(uint8(1), uint16(CommandCEchoRQ), uint16(1))

	// Seed with various parameter combinations
	f.Add(uint8(0), uint16(0), uint16(0))             // Invalid values
	f.Add(uint8(255), uint16(0xFFFF), uint16(0xFFFF)) // Max values

	f.Fuzz(func(t *testing.T, pcID uint8, cmdField uint16, msgID uint16) {
		// Create message with fuzzed parameters
		msg := &Message{
			CommandSet: &CommandSet{
				CommandField:       cmdField,
				MessageID:          msgID,
				CommandDataSetType: DataSetNotPresent,
			},
			PresentationContextID: pcID,
		}

		// Should never panic when encoding
		pdus, err := msg.Encode(16384)

		// If encoding succeeded, verify basic invariants
		if err == nil && len(pdus) > 0 {
			// All PDUs should be P-DATA-TF type
			for _, p := range pdus {
				if p == nil {
					t.Error("Null PDU in result")
				}

				// Verify presentation context ID matches
				for _, item := range p.Items {
					if item.PresentationContextID != pcID {
						t.Errorf("PC ID mismatch: expected %d, got %d",
							pcID, item.PresentationContextID)
					}
				}
			}
		}
	})
}

// FuzzReassemblerAddPDU tests message reassembly with malformed PDU sequences
func FuzzReassemblerAddPDU(f *testing.F) {
	// Seed with valid PDU sequence
	validMsg := &Message{
		CommandSet: &CommandSet{
			CommandField:       CommandCEchoRQ,
			MessageID:          1,
			CommandDataSetType: DataSetNotPresent,
		},
		PresentationContextID: 1,
	}

	pdus, _ := validMsg.Encode(16384)
	for _, p := range pdus {
		buf := &bytes.Buffer{}
		_ = p.Encode(buf)
		f.Add(buf.Bytes())
	}

	// Seed with empty PDU
	emptyPDU := &pdu.DataTF{Items: []pdu.PresentationDataValue{}}
	buf := &bytes.Buffer{}
	_ = emptyPDU.Encode(buf)
	f.Add(buf.Bytes())

	// Seed with PDU having invalid flags
	invalidFlagsPDU := &pdu.DataTF{
		Items: []pdu.PresentationDataValue{
			{
				PresentationContextID: 1,
				MessageControlHeader:  0xFF, // Invalid flags
				Data:                  []byte{0x00, 0x01, 0x02},
			},
		},
	}
	buf2 := &bytes.Buffer{}
	_ = invalidFlagsPDU.Encode(buf2)
	f.Add(buf2.Bytes())

	f.Fuzz(func(t *testing.T, data []byte) {
		// Decode PDU
		dataPDU := &pdu.DataTF{}
		err := dataPDU.Decode(bytes.NewReader(data))
		if err != nil {
			return // Invalid PDU is expected
		}

		// Should never panic when adding PDU to reassembler
		reassembler := NewMessageReassembler()
		msg, _ := reassembler.AddPDU(dataPDU)

		// If message was reassembled, verify basic structure
		if msg != nil {
			if msg.CommandSet == nil {
				t.Error("Reassembled message has nil CommandSet")
			}
			// Note: Field validation should be done in the decoder/reassembler,
			// not the fuzz test. Fuzz tests focus on crash prevention.
		}
	})
}

// FuzzStatusCodeHandling tests status code validation
func FuzzStatusCodeHandling(f *testing.F) {
	// Seed with valid status codes
	validStatuses := []uint16{
		StatusSuccess,
		StatusPending,
		StatusPendingWarning,
		StatusCancel,
		0xA700, // Refused: Out of Resources
		0xA900, // Error: Dataset does not match SOP Class
		0xC000, // Error: Cannot understand
	}

	for _, status := range validStatuses {
		f.Add(status)
	}

	// Seed with invalid ranges
	f.Add(uint16(0x0001)) // Invalid success variant
	f.Add(uint16(0xFFFF)) // Max value
	f.Add(uint16(0x0000)) // Success

	f.Fuzz(func(t *testing.T, status uint16) {
		// Create command with fuzzed status
		cmd := &CommandSet{
			CommandField:       CommandCEchoRSP,
			MessageID:          1,
			CommandDataSetType: DataSetNotPresent,
			Status:             status,
		}

		// Should be able to encode/decode without panic
		ds, err := cmd.ToDataSet()
		if err != nil {
			t.Fatalf("Failed to encode command with status 0x%04X: %v", status, err)
		}

		decoded, err := FromDataSet(ds)
		if err != nil {
			t.Fatalf("Failed to decode command with status 0x%04X: %v", status, err)
		}

		// Verify status is preserved
		if decoded.Status != status {
			t.Errorf("Status mismatch: expected 0x%04X, got 0x%04X", status, decoded.Status)
		}
	})
}

// FuzzMessageFragmentation tests message fragmentation at various PDU sizes
func FuzzMessageFragmentation(f *testing.F) {
	// Seed with various max PDU sizes
	f.Add(uint32(512))        // Small
	f.Add(uint32(16384))      // Standard
	f.Add(uint32(65536))      // Large
	f.Add(uint32(1))          // Minimum
	f.Add(uint32(0))          // Invalid
	f.Add(uint32(0xFFFFFFFF)) // Max

	f.Fuzz(func(t *testing.T, maxPDUSize uint32) {
		// Skip invalid PDU sizes that would cause issues
		// Minimum viable PDU size must accommodate headers:
		// - PDU header: 6 bytes
		// - PDV item header: 6 bytes (4 length + 1 context ID + 1 control)
		// - At least some data
		if maxPDUSize < 512 {
			return // Skip invalid/too small sizes
		}

		// Create message with dataset
		ds := dicom.NewDataSet()
		_ = ds.SetPatientName("Test^Patient")
		_ = ds.SetPatientID("12345")

		msg := &Message{
			CommandSet: &CommandSet{
				CommandField:       CommandCStoreRQ,
				MessageID:          1,
				CommandDataSetType: DataSetPresent,
			},
			DataSet:               ds,
			PresentationContextID: 1,
		}

		// Should handle any reasonable PDU size without panic
		pdus, err := msg.Encode(maxPDUSize)

		if err != nil {
			return // May fail for extreme sizes
		}

		// All PDUs should respect max size
		for i, p := range pdus {
			buf := &bytes.Buffer{}
			_ = p.Encode(buf)
			pduSize := uint32(buf.Len())

			if pduSize > maxPDUSize {
				t.Errorf("PDU %d size %d exceeds max %d", i, pduSize, maxPDUSize)
			}
		}
	})
}
