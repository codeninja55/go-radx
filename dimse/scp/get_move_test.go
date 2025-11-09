package scp_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/scp"
	"github.com/codeninja55/go-radx/dimse/scu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCGetSCP tests basic C-GET SCP functionality
func TestCGetSCP(t *testing.T) {
	t.Skip("Skipping C-GET test - requires complex sub-operation handling")

	// C-GET is complex because:
	// 1. Server must initiate C-STORE sub-operations
	// 2. Client must act as SCP to receive stores
	// 3. Bidirectional message flow
	//
	// This test would require:
	// - Mock instances to retrieve
	// - Client-side store handler
	// - Sub-operation tracking
}

// TestCMoveSCP tests basic C-MOVE SCP functionality
func TestCMoveSCP(t *testing.T) {
	var mu sync.Mutex
	var moveRequest *scp.MoveRequest

	moveHandler := scp.MoveHandlerFunc(func(ctx context.Context, req *scp.MoveRequest) *scp.MoveResponse {
		mu.Lock()
		moveRequest = req
		mu.Unlock()

		// Simulate successful move of 3 instances
		return &scp.MoveResponse{
			NumberOfCompletedSubOps: 3,
			NumberOfFailedSubOps:    0,
			NumberOfWarningSubOps:   0,
			Status:                  dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "MOVE_SCP",
		ListenAddr: "127.0.0.1:11123",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":           {"1.2.840.10008.1.2"}, // Verification
			"1.2.840.10008.5.1.4.1.2.1.2": {"1.2.840.10008.1.2"}, // Patient Root Q/R - MOVE
		},
		EchoHandler: scp.NewDefaultEchoHandler(),
		MoveHandler: moveHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.2.1.2" // Patient Root Q/R - MOVE

	clientConfig := scu.Config{
		CallingAETitle: "MOVE_SCU",
		CalledAETitle:  "MOVE_SCP",
		RemoteAddr:     "127.0.0.1:11123",
		PresentationContexts: []dul.PresentationContextRQ{
			{
				ID:             1,
				AbstractSyntax: "1.2.840.10008.1.1",
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2",
				},
			},
			{
				ID:             3,
				AbstractSyntax: sopClassUID,
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2",
				},
			},
		},
	}

	client := scu.NewClient(clientConfig)

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(ctx)

	// Create query dataset
	query := dicom.NewDataSet()

	// Perform C-MOVE
	destination := "DESTINATION_AE"
	err = client.Move(ctx, sopClassUID, destination, query)
	assert.NoError(t, err)

	// Verify move request was received
	mu.Lock()
	assert.NotNil(t, moveRequest)
	if moveRequest != nil {
		assert.Equal(t, "MOVE_SCU", moveRequest.CallingAE)
		assert.Equal(t, "MOVE_SCP", moveRequest.CalledAE)
		assert.Equal(t, destination, moveRequest.Destination)
		assert.NotNil(t, moveRequest.Query)
	}
	mu.Unlock()
}

// TestCMoveSCP_WithFailures tests C-MOVE with partial failures
func TestCMoveSCP_WithFailures(t *testing.T) {
	moveHandler := scp.MoveHandlerFunc(func(ctx context.Context, req *scp.MoveRequest) *scp.MoveResponse {
		// Simulate move with some failures
		return &scp.MoveResponse{
			NumberOfCompletedSubOps: 5,
			NumberOfFailedSubOps:    2,
			NumberOfWarningSubOps:   1,
			Status:                  dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "MOVE_SCP",
		ListenAddr: "127.0.0.1:11124",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":           {"1.2.840.10008.1.2"},
			"1.2.840.10008.5.1.4.1.2.1.2": {"1.2.840.10008.1.2"},
		},
		MoveHandler: moveHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.2.1.2"

	clientConfig := scu.Config{
		CallingAETitle: "MOVE_SCU",
		CalledAETitle:  "MOVE_SCP",
		RemoteAddr:     "127.0.0.1:11124",
		PresentationContexts: []dul.PresentationContextRQ{
			{
				ID:             1,
				AbstractSyntax: "1.2.840.10008.1.1",
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2",
				},
			},
			{
				ID:             3,
				AbstractSyntax: sopClassUID,
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2",
				},
			},
		},
	}

	client := scu.NewClient(clientConfig)

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(ctx)

	query := dicom.NewDataSet()

	err = client.Move(ctx, sopClassUID, "DEST_AE", query)
	assert.NoError(t, err)
}

// TestCMoveSCP_UnknownDestination tests C-MOVE to unknown destination
func TestCMoveSCP_UnknownDestination(t *testing.T) {
	moveHandler := scp.MoveHandlerFunc(func(ctx context.Context, req *scp.MoveRequest) *scp.MoveResponse {
		// Simulate unknown destination
		return &scp.MoveResponse{
			NumberOfCompletedSubOps: 0,
			NumberOfFailedSubOps:    0,
			NumberOfWarningSubOps:   0,
			Status:                  dimse.StatusMoveDestinationUnknown,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "MOVE_SCP",
		ListenAddr: "127.0.0.1:11125",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":           {"1.2.840.10008.1.2"},
			"1.2.840.10008.5.1.4.1.2.1.2": {"1.2.840.10008.1.2"},
		},
		MoveHandler: moveHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.2.1.2"

	clientConfig := scu.Config{
		CallingAETitle: "MOVE_SCU",
		CalledAETitle:  "MOVE_SCP",
		RemoteAddr:     "127.0.0.1:11125",
		PresentationContexts: []dul.PresentationContextRQ{
			{
				ID:             1,
				AbstractSyntax: "1.2.840.10008.1.1",
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2",
				},
			},
			{
				ID:             3,
				AbstractSyntax: sopClassUID,
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2",
				},
			},
		},
	}

	client := scu.NewClient(clientConfig)

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(ctx)

	query := dicom.NewDataSet()

	err = client.Move(ctx, sopClassUID, "UNKNOWN_AE", query)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "C-MOVE failed")
}
