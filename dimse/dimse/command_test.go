package dimse_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommandSet_CEcho tests C-ECHO command encoding/decoding
func TestCommandSet_CEcho(t *testing.T) {
	// C-ECHO-RQ
	original := &dimse.CommandSet{
		CommandField:        dimse.CommandCEchoRQ,
		MessageID:           123,
		CommandDataSetType:  dimse.DataSetNotPresent,
		AffectedSOPClassUID: "1.2.840.10008.1.1",
	}

	// Convert to dataset
	ds, err := original.ToDataSet()
	require.NoError(t, err)

	// Convert back from dataset
	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, original.CommandField, decoded.CommandField)
	assert.Equal(t, original.MessageID, decoded.MessageID)
	assert.Equal(t, original.CommandDataSetType, decoded.CommandDataSetType)
	assert.Equal(t, original.AffectedSOPClassUID, decoded.AffectedSOPClassUID)
}

// TestCommandSet_CStore tests C-STORE command encoding/decoding
func TestCommandSet_CStore(t *testing.T) {
	// C-STORE-RQ
	original := &dimse.CommandSet{
		CommandField:           dimse.CommandCStoreRQ,
		MessageID:              456,
		Priority:               dimse.PriorityMedium,
		CommandDataSetType:     dimse.DataSetPresent,
		AffectedSOPClassUID:    "1.2.840.10008.5.1.4.1.1.2",
		AffectedSOPInstanceUID: "1.2.840.12345.1.1.1.1",
	}

	ds, err := original.ToDataSet()
	require.NoError(t, err)

	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	assert.Equal(t, original.CommandField, decoded.CommandField)
	assert.Equal(t, original.MessageID, decoded.MessageID)
	assert.Equal(t, original.Priority, decoded.Priority)
	assert.Equal(t, original.CommandDataSetType, decoded.CommandDataSetType)
	assert.Equal(t, original.AffectedSOPClassUID, decoded.AffectedSOPClassUID)
	assert.Equal(t, original.AffectedSOPInstanceUID, decoded.AffectedSOPInstanceUID)
}

// TestCommandSet_CStoreResponse tests C-STORE-RSP encoding/decoding
func TestCommandSet_CStoreResponse(t *testing.T) {
	// C-STORE-RSP
	original := &dimse.CommandSet{
		CommandField:              dimse.CommandCStoreRSP,
		MessageIDBeingRespondedTo: 456,
		CommandDataSetType:        dimse.DataSetNotPresent,
		Status:                    dimse.StatusSuccess,
		AffectedSOPClassUID:       "1.2.840.10008.5.1.4.1.1.2",
		AffectedSOPInstanceUID:    "1.2.840.12345.1.1.1.1",
	}

	ds, err := original.ToDataSet()
	require.NoError(t, err)

	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	assert.Equal(t, original.CommandField, decoded.CommandField)
	assert.Equal(t, original.MessageIDBeingRespondedTo, decoded.MessageIDBeingRespondedTo)
	assert.Equal(t, original.Status, decoded.Status)
}

// TestCommandSet_CFind tests C-FIND command encoding/decoding
func TestCommandSet_CFind(t *testing.T) {
	// C-FIND-RQ
	original := &dimse.CommandSet{
		CommandField:        dimse.CommandCFindRQ,
		MessageID:           789,
		Priority:            dimse.PriorityHigh,
		CommandDataSetType:  dimse.DataSetPresent,
		AffectedSOPClassUID: "1.2.840.10008.5.1.4.1.2.1.1",
	}

	ds, err := original.ToDataSet()
	require.NoError(t, err)

	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	assert.Equal(t, original.CommandField, decoded.CommandField)
	assert.Equal(t, original.MessageID, decoded.MessageID)
	assert.Equal(t, original.Priority, decoded.Priority)
	assert.Equal(t, original.AffectedSOPClassUID, decoded.AffectedSOPClassUID)
}

// TestCommandSet_CFindPendingResponse tests C-FIND pending response
func TestCommandSet_CFindPendingResponse(t *testing.T) {
	// C-FIND-RSP (pending)
	original := &dimse.CommandSet{
		CommandField:              dimse.CommandCFindRSP,
		MessageIDBeingRespondedTo: 789,
		CommandDataSetType:        dimse.DataSetPresent,
		Status:                    dimse.StatusPending,
		AffectedSOPClassUID:       "1.2.840.10008.5.1.4.1.2.1.1",
	}

	ds, err := original.ToDataSet()
	require.NoError(t, err)

	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	assert.Equal(t, dimse.StatusPending, decoded.Status)
	assert.Equal(t, dimse.DataSetPresent, decoded.CommandDataSetType)
}

// TestCommandSet_CGet tests C-GET command encoding/decoding
func TestCommandSet_CGet(t *testing.T) {
	// C-GET-RQ
	original := &dimse.CommandSet{
		CommandField:        dimse.CommandCGetRQ,
		MessageID:           101,
		Priority:            dimse.PriorityLow,
		CommandDataSetType:  dimse.DataSetPresent,
		AffectedSOPClassUID: "1.2.840.10008.5.1.4.1.2.1.3",
	}

	ds, err := original.ToDataSet()
	require.NoError(t, err)

	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	assert.Equal(t, original.CommandField, decoded.CommandField)
	assert.Equal(t, original.MessageID, decoded.MessageID)
	assert.Equal(t, original.Priority, decoded.Priority)
}

// TestCommandSet_CGetWithSubOps tests C-GET response with sub-operation counts
func TestCommandSet_CGetWithSubOps(t *testing.T) {
	// C-GET-RSP with sub-operation counts
	original := &dimse.CommandSet{
		CommandField:              dimse.CommandCGetRSP,
		MessageIDBeingRespondedTo: 101,
		CommandDataSetType:        dimse.DataSetNotPresent,
		Status:                    dimse.StatusPending,
		NumberOfRemainingSubOps:   10,
		NumberOfCompletedSubOps:   5,
		NumberOfFailedSubOps:      1,
		NumberOfWarningSubOps:     0,
	}

	ds, err := original.ToDataSet()
	require.NoError(t, err)

	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	assert.Equal(t, original.NumberOfRemainingSubOps, decoded.NumberOfRemainingSubOps)
	assert.Equal(t, original.NumberOfCompletedSubOps, decoded.NumberOfCompletedSubOps)
	assert.Equal(t, original.NumberOfFailedSubOps, decoded.NumberOfFailedSubOps)
	assert.Equal(t, original.NumberOfWarningSubOps, decoded.NumberOfWarningSubOps)
}

// TestCommandSet_CMove tests C-MOVE command encoding/decoding
func TestCommandSet_CMove(t *testing.T) {
	// C-MOVE-RQ
	original := &dimse.CommandSet{
		CommandField:        dimse.CommandCMoveRQ,
		MessageID:           202,
		Priority:            dimse.PriorityMedium,
		CommandDataSetType:  dimse.DataSetPresent,
		AffectedSOPClassUID: "1.2.840.10008.5.1.4.1.2.1.2",
		MoveDestination:     "DEST_AE",
	}

	ds, err := original.ToDataSet()
	require.NoError(t, err)

	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	assert.Equal(t, original.CommandField, decoded.CommandField)
	assert.Equal(t, original.MessageID, decoded.MessageID)
	assert.Equal(t, original.MoveDestination, decoded.MoveDestination)
}

// TestCommandSet_StatusCodes tests various status codes
func TestCommandSet_StatusCodes(t *testing.T) {
	statuses := []struct {
		name   string
		status uint16
	}{
		{"Success", dimse.StatusSuccess},
		{"Pending", dimse.StatusPending},
		{"Cancel", dimse.StatusCancel},
		{"Processing Failure", dimse.StatusProcessingFailure},
		{"SOP Class Not Supported", dimse.StatusSOPClassNotSupported},
	}

	for _, tt := range statuses {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &dimse.CommandSet{
				CommandField:              dimse.CommandCEchoRSP,
				MessageIDBeingRespondedTo: 1,
				CommandDataSetType:        dimse.DataSetNotPresent,
				Status:                    tt.status,
			}

			ds, err := cmd.ToDataSet()
			require.NoError(t, err)

			decoded, err := dimse.FromDataSet(ds)
			require.NoError(t, err)

			assert.Equal(t, tt.status, decoded.Status)
		})
	}
}

// TestCommandSet_PriorityValues tests priority encoding
func TestCommandSet_PriorityValues(t *testing.T) {
	priorities := []uint16{
		dimse.PriorityLow,
		dimse.PriorityMedium,
		dimse.PriorityHigh,
	}

	for _, priority := range priorities {
		cmd := &dimse.CommandSet{
			CommandField:       dimse.CommandCStoreRQ,
			MessageID:          1,
			Priority:           priority,
			CommandDataSetType: dimse.DataSetPresent,
		}

		ds, err := cmd.ToDataSet()
		require.NoError(t, err)

		decoded, err := dimse.FromDataSet(ds)
		require.NoError(t, err)

		assert.Equal(t, priority, decoded.Priority)
	}
}

// TestCommandSet_EmptyFields tests encoding with minimal fields
func TestCommandSet_EmptyFields(t *testing.T) {
	// Minimal C-ECHO-RQ
	cmd := &dimse.CommandSet{
		CommandField:       dimse.CommandCEchoRQ,
		MessageID:          1,
		CommandDataSetType: dimse.DataSetNotPresent,
	}

	ds, err := cmd.ToDataSet()
	require.NoError(t, err)

	decoded, err := dimse.FromDataSet(ds)
	require.NoError(t, err)

	assert.Equal(t, cmd.CommandField, decoded.CommandField)
	assert.Equal(t, cmd.MessageID, decoded.MessageID)

	// Optional fields should be zero values
	assert.Empty(t, decoded.AffectedSOPClassUID)
	assert.Empty(t, decoded.AffectedSOPInstanceUID)
	assert.Zero(t, decoded.Priority)
}
