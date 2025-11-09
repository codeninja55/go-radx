package scp_test

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/scp"
	"github.com/codeninja55/go-radx/dimse/scu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCEchoSCP tests C-ECHO SCP functionality
func TestCEchoSCP(t *testing.T) {
	// Create server config
	serverConfig := scp.Config{
		AETitle:    "TEST_SCP",
		ListenAddr: "127.0.0.1:0", // Use random port
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1": {"1.2.840.10008.1.2"}, // Verification SOP Class, Implicit VR LE
		},
		EchoHandler: scp.NewDefaultEchoHandler(),
	}

	// Create and start server
	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	// Get the actual listening address
	// Note: We would need to expose the listener's address through the Server type
	// For now, using a fixed port for testing
	serverAddr := "127.0.0.1:11112"

	// Recreate server with fixed address for testing
	serverConfig.ListenAddr = serverAddr
	server, err = scp.NewServer(serverConfig)
	require.NoError(t, err)

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	// Small delay to ensure server is listening
	time.Sleep(100 * time.Millisecond)

	// Create SCU client
	clientConfig := scu.Config{
		CallingAETitle: "TEST_SCU",
		CalledAETitle:  "TEST_SCP",
		RemoteAddr:     serverAddr,
		PresentationContexts: []dul.PresentationContextRQ{
			{
				ID:             1,
				AbstractSyntax: "1.2.840.10008.1.1", // Verification SOP Class
				TransferSyntaxes: []string{
					"1.2.840.10008.1.2", // Implicit VR Little Endian
				},
			},
		},
	}

	client := scu.NewClient(clientConfig)

	// Connect to server
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(ctx)

	// Perform C-ECHO
	err = client.Echo(ctx)
	assert.NoError(t, err)
}

// TestCEchoSCP_CustomHandler tests C-ECHO with custom handler
func TestCEchoSCP_CustomHandler(t *testing.T) {
	handlerCalled := false

	customHandler := scp.EchoHandlerFunc(func(ctx context.Context, req *scp.EchoRequest) *scp.EchoResponse {
		handlerCalled = true
		assert.Equal(t, "TEST_SCU", req.CallingAE)
		assert.Equal(t, "TEST_SCP", req.CalledAE)
		return &scp.EchoResponse{
			Status: dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "TEST_SCP",
		ListenAddr: "127.0.0.1:11113",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1": {"1.2.840.10008.1.2"},
		},
		EchoHandler: customHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	// Create SCU client
	clientConfig := scu.Config{
		CallingAETitle: "TEST_SCU",
		CalledAETitle:  "TEST_SCP",
		RemoteAddr:     "127.0.0.1:11113",
		PresentationContexts: []dul.PresentationContextRQ{
			{
				ID:             1,
				AbstractSyntax: "1.2.840.10008.1.1",
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

	err = client.Echo(ctx)
	assert.NoError(t, err)
	assert.True(t, handlerCalled, "Custom echo handler should have been called")
}

// TestCEchoSCP_MultipleClients tests multiple concurrent C-ECHO operations
func TestCEchoSCP_MultipleClients(t *testing.T) {
	serverConfig := scp.Config{
		AETitle:         "TEST_SCP",
		ListenAddr:      "127.0.0.1:11114",
		MaxAssociations: 5,
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1": {"1.2.840.10008.1.2"},
		},
		EchoHandler: scp.NewDefaultEchoHandler(),
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	// Create multiple clients concurrently
	numClients := 3
	errChan := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientNum int) {
			clientConfig := scu.Config{
				CallingAETitle: "TEST_SCU",
				CalledAETitle:  "TEST_SCP",
				RemoteAddr:     "127.0.0.1:11114",
				PresentationContexts: []dul.PresentationContextRQ{
					{
						ID:             1,
						AbstractSyntax: "1.2.840.10008.1.1",
						TransferSyntaxes: []string{
							"1.2.840.10008.1.2",
						},
					},
				},
			}

			client := scu.NewClient(clientConfig)

			if err := client.Connect(ctx); err != nil {
				errChan <- err
				return
			}
			defer client.Close(ctx)

			if err := client.Echo(ctx); err != nil {
				errChan <- err
				return
			}

			errChan <- nil
		}(i)
	}

	// Wait for all clients to complete
	for i := 0; i < numClients; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
}

// TestServerShutdown tests graceful server shutdown
func TestServerShutdown(t *testing.T) {
	serverConfig := scp.Config{
		AETitle:    "TEST_SCP",
		ListenAddr: "127.0.0.1:11115",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1": {"1.2.840.10008.1.2"},
		},
		EchoHandler: scp.NewDefaultEchoHandler(),
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx := context.Background()

	err = server.Listen(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

// TestMaxAssociations tests that server enforces max associations limit
func TestMaxAssociations(t *testing.T) {
	t.Skip("Skipping max associations test - requires more complex setup")

	// This test would require:
	// 1. Setting MaxAssociations to a low number (e.g., 2)
	// 2. Creating more clients than the limit
	// 3. Verifying that excess connections are rejected
	// 4. Keeping connections open long enough to hit the limit
}
