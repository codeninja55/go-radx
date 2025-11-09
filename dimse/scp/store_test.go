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

// TestCStoreSCP tests C-STORE SCP functionality
func TestCStoreSCP(t *testing.T) {
	var mu sync.Mutex
	var storedInstances []*scp.StoreRequest

	storeHandler := scp.StoreHandlerFunc(func(ctx context.Context, req *scp.StoreRequest) *scp.StoreResponse {
		mu.Lock()
		defer mu.Unlock()
		storedInstances = append(storedInstances, req)
		return &scp.StoreResponse{
			Status: dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "STORE_SCP",
		ListenAddr: "127.0.0.1:11116",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":             {"1.2.840.10008.1.2"}, // Verification
			"1.2.840.10008.5.1.4.1.1.2":     {"1.2.840.10008.1.2"}, // CT Image Storage
			"1.2.840.10008.5.1.4.1.1.4":     {"1.2.840.10008.1.2"}, // MR Image Storage
			"1.2.840.10008.5.1.4.1.1.88.22": {"1.2.840.10008.1.2"}, // Enhanced SR Storage
		},
		EchoHandler:  scp.NewDefaultEchoHandler(),
		StoreHandler: storeHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	// Create test dataset
	ds := dicom.NewDataSet()
	// Add required DICOM tags for storage
	// Note: In a real test, we would add proper tags using the tag/element packages

	sopClassUID := "1.2.840.10008.5.1.4.1.1.2" // CT Image Storage
	sopInstanceUID := "1.2.840.12345.1.1.1.1"

	// Create SCU client
	clientConfig := scu.Config{
		CallingAETitle: "STORE_SCU",
		CalledAETitle:  "STORE_SCP",
		RemoteAddr:     "127.0.0.1:11116",
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

	// Send C-STORE
	err = client.Store(ctx, ds, sopClassUID, sopInstanceUID)
	assert.NoError(t, err)

	// Verify handler was called
	mu.Lock()
	assert.Len(t, storedInstances, 1)
	if len(storedInstances) > 0 {
		assert.Equal(t, "STORE_SCU", storedInstances[0].CallingAE)
		assert.Equal(t, "STORE_SCP", storedInstances[0].CalledAE)
		assert.Equal(t, sopClassUID, storedInstances[0].SOPClassUID)
		assert.Equal(t, sopInstanceUID, storedInstances[0].SOPInstanceUID)
		assert.NotNil(t, storedInstances[0].DataSet)
	}
	mu.Unlock()
}

// TestCStoreSCP_MultipleInstances tests storing multiple instances
func TestCStoreSCP_MultipleInstances(t *testing.T) {
	var mu sync.Mutex
	var storedCount int

	storeHandler := scp.StoreHandlerFunc(func(ctx context.Context, req *scp.StoreRequest) *scp.StoreResponse {
		mu.Lock()
		storedCount++
		mu.Unlock()
		return &scp.StoreResponse{
			Status: dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "STORE_SCP",
		ListenAddr: "127.0.0.1:11117",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":         {"1.2.840.10008.1.2"},
			"1.2.840.10008.5.1.4.1.1.2": {"1.2.840.10008.1.2"},
		},
		StoreHandler: storeHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.1.2"

	clientConfig := scu.Config{
		CallingAETitle: "STORE_SCU",
		CalledAETitle:  "STORE_SCP",
		RemoteAddr:     "127.0.0.1:11117",
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

	// Store multiple instances
	numInstances := 5
	for i := 0; i < numInstances; i++ {
		ds := dicom.NewDataSet()
		sopInstanceUID := "1.2.840.12345.1.1.1." + string(rune(i+1))

		err = client.Store(ctx, ds, sopClassUID, sopInstanceUID)
		assert.NoError(t, err)
	}

	// Verify all instances were stored
	mu.Lock()
	assert.Equal(t, numInstances, storedCount)
	mu.Unlock()
}

// TestCStoreSCP_FailureResponse tests handling of storage failures
func TestCStoreSCP_FailureResponse(t *testing.T) {
	storeHandler := scp.StoreHandlerFunc(func(ctx context.Context, req *scp.StoreRequest) *scp.StoreResponse {
		// Simulate storage failure
		return &scp.StoreResponse{
			Status: dimse.StatusProcessingFailure,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "STORE_SCP",
		ListenAddr: "127.0.0.1:11118",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":         {"1.2.840.10008.1.2"},
			"1.2.840.10008.5.1.4.1.1.2": {"1.2.840.10008.1.2"},
		},
		StoreHandler: storeHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.1.2"
	sopInstanceUID := "1.2.840.12345.1.1.1.1"

	clientConfig := scu.Config{
		CallingAETitle: "STORE_SCU",
		CalledAETitle:  "STORE_SCP",
		RemoteAddr:     "127.0.0.1:11118",
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

	ds := dicom.NewDataSet()

	// Attempt to store - should fail
	err = client.Store(ctx, ds, sopClassUID, sopInstanceUID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "C-STORE failed")
}
