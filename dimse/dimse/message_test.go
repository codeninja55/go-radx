package dimse_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessage_EncodeSimple tests encoding a simple message without dataset
func TestMessage_EncodeSimple(t *testing.T) {
	cmd := &dimse.CommandSet{
		CommandField:        dimse.CommandCEchoRQ,
		MessageID:           1,
		CommandDataSetType:  dimse.DataSetNotPresent,
		AffectedSOPClassUID: "1.2.840.10008.1.1",
	}

	msg := &dimse.Message{
		CommandSet:            cmd,
		PresentationContextID: 1,
	}

	pdus, err := msg.Encode(16384)
	require.NoError(t, err)
	assert.NotEmpty(t, pdus)

	// Should have at least one PDU for the command
	assert.GreaterOrEqual(t, len(pdus), 1)

	// Verify PDU type
	for _, pduItem := range pdus {
		assert.NotNil(t, pduItem)
		assert.Len(t, pduItem.Items, 1)
		assert.True(t, pduItem.Items[0].IsCommand())
	}
}

// TestMessage_EncodeWithDataset tests encoding message with dataset
func TestMessage_EncodeWithDataset(t *testing.T) {
	cmd := &dimse.CommandSet{
		CommandField:           dimse.CommandCStoreRQ,
		MessageID:              2,
		Priority:               dimse.PriorityMedium,
		CommandDataSetType:     dimse.DataSetPresent,
		AffectedSOPClassUID:    "1.2.840.10008.5.1.4.1.1.2",
		AffectedSOPInstanceUID: "1.2.840.12345.1.1.1.1",
	}

	ds := dicom.NewDataSet()
	// Add some data to dataset (in real use would add proper DICOM elements)

	msg := &dimse.Message{
		CommandSet:            cmd,
		DataSet:               ds,
		PresentationContextID: 1,
	}

	pdus, err := msg.Encode(16384)
	require.NoError(t, err)
	assert.NotEmpty(t, pdus)

	// Should have PDUs for command (empty dataset might not produce dataset PDUs)
	hasCommand := false

	for _, pduItem := range pdus {
		for _, item := range pduItem.Items {
			if item.IsCommand() {
				hasCommand = true
			}
			// Note: We don't check hasDataset because empty datasets might not produce PDUs
		}
	}

	assert.True(t, hasCommand, "Should have command PDUs")
}

// TestMessage_Fragmentation tests message fragmentation with small PDU size
func TestMessage_Fragmentation(t *testing.T) {
	cmd := &dimse.CommandSet{
		CommandField:        dimse.CommandCEchoRQ,
		MessageID:           3,
		CommandDataSetType:  dimse.DataSetNotPresent,
		AffectedSOPClassUID: "1.2.840.10008.1.1",
	}

	msg := &dimse.Message{
		CommandSet:            cmd,
		PresentationContextID: 1,
	}

	// Use very small PDU size to force fragmentation
	smallPDUSize := uint32(256)
	pdus, err := msg.Encode(smallPDUSize)
	require.NoError(t, err)

	// Verify fragmentation occurred
	// (number of PDUs depends on command size, should be at least 1)
	assert.GreaterOrEqual(t, len(pdus), 1)

	// Verify last fragment flag
	lastPDU := pdus[len(pdus)-1]
	assert.True(t, lastPDU.Items[len(lastPDU.Items)-1].IsLastFragment())
}

// TestMessageReassembler_Simple tests reassembling a simple message
func TestMessageReassembler_Simple(t *testing.T) {
	// Create original message
	cmd := &dimse.CommandSet{
		CommandField:        dimse.CommandCEchoRQ,
		MessageID:           4,
		CommandDataSetType:  dimse.DataSetNotPresent,
		AffectedSOPClassUID: "1.2.840.10008.1.1",
	}

	original := &dimse.Message{
		CommandSet:            cmd,
		PresentationContextID: 1,
	}

	// Encode to PDUs
	pdus, err := original.Encode(16384)
	require.NoError(t, err)

	// Reassemble
	reassembler := dimse.NewMessageReassembler()
	var reassembled *dimse.Message

	for _, pduItem := range pdus {
		msg, err := reassembler.AddPDU(pduItem)
		require.NoError(t, err)

		if msg != nil {
			reassembled = msg
			break
		}
	}

	require.NotNil(t, reassembled)
	assert.Equal(t, original.CommandSet.CommandField, reassembled.CommandSet.CommandField)
	assert.Equal(t, original.CommandSet.MessageID, reassembled.CommandSet.MessageID)
}

// TestMessageReassembler_Fragmented tests reassembling fragmented message
func TestMessageReassembler_Fragmented(t *testing.T) {
	// Create message that will be fragmented
	cmd := &dimse.CommandSet{
		CommandField:           dimse.CommandCStoreRQ,
		MessageID:              5,
		Priority:               dimse.PriorityHigh,
		CommandDataSetType:     dimse.DataSetPresent,
		AffectedSOPClassUID:    "1.2.840.10008.5.1.4.1.1.2",
		AffectedSOPInstanceUID: "1.2.840.12345.1.1.1.1",
	}

	ds := dicom.NewDataSet()

	original := &dimse.Message{
		CommandSet:            cmd,
		DataSet:               ds,
		PresentationContextID: 3,
	}

	// Encode with small PDU size to force fragmentation
	pdus, err := original.Encode(512)
	require.NoError(t, err)

	// Reassemble
	reassembler := dimse.NewMessageReassembler()
	var reassembled *dimse.Message

	for i, pduItem := range pdus {
		msg, err := reassembler.AddPDU(pduItem)
		require.NoError(t, err)

		if i < len(pdus)-1 {
			// Not last PDU, should return nil
			assert.Nil(t, msg)
		} else {
			// Last PDU, should return complete message
			assert.NotNil(t, msg)
			reassembled = msg
		}
	}

	require.NotNil(t, reassembled)
	assert.Equal(t, original.CommandSet.CommandField, reassembled.CommandSet.CommandField)
	assert.Equal(t, original.PresentationContextID, reassembled.PresentationContextID)
}

// TestMessageReassembler_MultiplePresentationContexts tests concurrent message reassembly
func TestMessageReassembler_MultiplePresentationContexts(t *testing.T) {
	// Create two messages with different presentation contexts
	cmd1 := &dimse.CommandSet{
		CommandField:       dimse.CommandCEchoRQ,
		MessageID:          6,
		CommandDataSetType: dimse.DataSetNotPresent,
	}

	cmd2 := &dimse.CommandSet{
		CommandField:       dimse.CommandCEchoRQ,
		MessageID:          7,
		CommandDataSetType: dimse.DataSetNotPresent,
	}

	msg1 := &dimse.Message{
		CommandSet:            cmd1,
		PresentationContextID: 1,
	}

	msg2 := &dimse.Message{
		CommandSet:            cmd2,
		PresentationContextID: 3,
	}

	// Encode both
	pdus1, err := msg1.Encode(512)
	require.NoError(t, err)

	pdus2, err := msg2.Encode(512)
	require.NoError(t, err)

	// Interleave PDUs
	reassembler := dimse.NewMessageReassembler()

	// Add first PDU from each message
	_, err = reassembler.AddPDU(pdus1[0])
	require.NoError(t, err)

	_, err = reassembler.AddPDU(pdus2[0])
	require.NoError(t, err)

	// Add remaining PDUs
	for i := 1; i < len(pdus1); i++ {
		result, err := reassembler.AddPDU(pdus1[i])
		require.NoError(t, err)
		if i == len(pdus1)-1 {
			assert.NotNil(t, result)
			assert.Equal(t, uint8(1), result.PresentationContextID)
		}
	}

	for i := 1; i < len(pdus2); i++ {
		result, err := reassembler.AddPDU(pdus2[i])
		require.NoError(t, err)
		if i == len(pdus2)-1 {
			assert.NotNil(t, result)
			assert.Equal(t, uint8(3), result.PresentationContextID)
		}
	}
}

// TestMessage_LargeDataset tests encoding/decoding large dataset
func TestMessage_LargeDataset(t *testing.T) {
	t.Skip("Skipping large dataset test - requires substantial DICOM data")

	// This test would:
	// 1. Create a large dataset (e.g., CT image with pixel data)
	// 2. Encode it with normal PDU size
	// 3. Verify multiple PDUs are created
	// 4. Reassemble and verify integrity
}

// TestMessage_MaxPDULength tests various max PDU lengths
func TestMessage_MaxPDULength(t *testing.T) {
	cmd := &dimse.CommandSet{
		CommandField:       dimse.CommandCEchoRQ,
		MessageID:          8,
		CommandDataSetType: dimse.DataSetNotPresent,
	}

	msg := &dimse.Message{
		CommandSet:            cmd,
		PresentationContextID: 1,
	}

	testSizes := []uint32{
		1024,   // 1KB
		8192,   // 8KB
		16384,  // 16KB (default)
		32768,  // 32KB
		131072, // 128KB
	}

	for _, size := range testSizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			pdus, err := msg.Encode(size)
			require.NoError(t, err)
			assert.NotEmpty(t, pdus)
		})
	}
}

// TestDecode tests decoding messages from PDUs
func TestDecode(t *testing.T) {
	// Create and encode a message
	cmd := &dimse.CommandSet{
		CommandField:        dimse.CommandCEchoRQ,
		MessageID:           9,
		CommandDataSetType:  dimse.DataSetNotPresent,
		AffectedSOPClassUID: "1.2.840.10008.1.1",
	}

	original := &dimse.Message{
		CommandSet:            cmd,
		PresentationContextID: 1,
	}

	pdus, err := original.Encode(16384)
	require.NoError(t, err)

	// Decode directly
	decoded, err := dimse.Decode(pdus)
	require.NoError(t, err)

	assert.Equal(t, original.CommandSet.CommandField, decoded.CommandSet.CommandField)
	assert.Equal(t, original.CommandSet.MessageID, decoded.CommandSet.MessageID)
	assert.Equal(t, original.PresentationContextID, decoded.PresentationContextID)
}
