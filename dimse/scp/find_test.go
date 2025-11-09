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

// TestCFindSCP tests C-FIND SCP functionality
func TestCFindSCP(t *testing.T) {
	var mu sync.Mutex
	var queryReceived *dicom.DataSet

	// Create mock results
	mockResults := []*dicom.DataSet{
		dicom.NewDataSet(),
		dicom.NewDataSet(),
		dicom.NewDataSet(),
	}

	findHandler := scp.FindHandlerFunc(func(ctx context.Context, req *scp.FindRequest) *scp.FindResponse {
		mu.Lock()
		queryReceived = req.Query
		mu.Unlock()

		return &scp.FindResponse{
			Results: mockResults,
			Status:  dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "FIND_SCP",
		ListenAddr: "127.0.0.1:11119",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":           {"1.2.840.10008.1.2"}, // Verification
			"1.2.840.10008.5.1.4.1.2.1.1": {"1.2.840.10008.1.2"}, // Patient Root Q/R - FIND
			"1.2.840.10008.5.1.4.1.2.2.1": {"1.2.840.10008.1.2"}, // Study Root Q/R - FIND
		},
		EchoHandler: scp.NewDefaultEchoHandler(),
		FindHandler: findHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.2.1.1" // Patient Root Q/R - FIND

	clientConfig := scu.Config{
		CallingAETitle: "FIND_SCU",
		CalledAETitle:  "FIND_SCP",
		RemoteAddr:     "127.0.0.1:11119",
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

	// Perform C-FIND
	var resultsReceived []*dicom.DataSet
	var resultMu sync.Mutex

	err = client.Find(ctx, "PATIENT", sopClassUID, query, func(ds *dicom.DataSet) error {
		resultMu.Lock()
		resultsReceived = append(resultsReceived, ds)
		resultMu.Unlock()
		return nil
	})
	assert.NoError(t, err)

	// Verify query was received
	mu.Lock()
	assert.NotNil(t, queryReceived)
	mu.Unlock()

	// Verify results were received
	resultMu.Lock()
	assert.Len(t, resultsReceived, len(mockResults))
	resultMu.Unlock()
}

// TestCFindSCP_NoResults tests C-FIND with no matching results
func TestCFindSCP_NoResults(t *testing.T) {
	findHandler := scp.FindHandlerFunc(func(ctx context.Context, req *scp.FindRequest) *scp.FindResponse {
		return &scp.FindResponse{
			Results: nil, // No results
			Status:  dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "FIND_SCP",
		ListenAddr: "127.0.0.1:11120",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":           {"1.2.840.10008.1.2"},
			"1.2.840.10008.5.1.4.1.2.1.1": {"1.2.840.10008.1.2"},
		},
		FindHandler: findHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.2.1.1"

	clientConfig := scu.Config{
		CallingAETitle: "FIND_SCU",
		CalledAETitle:  "FIND_SCP",
		RemoteAddr:     "127.0.0.1:11120",
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

	var resultsReceived []*dicom.DataSet
	var resultMu sync.Mutex

	err = client.Find(ctx, "PATIENT", sopClassUID, query, func(ds *dicom.DataSet) error {
		resultMu.Lock()
		resultsReceived = append(resultsReceived, ds)
		resultMu.Unlock()
		return nil
	})
	assert.NoError(t, err)

	// Verify no results were received
	resultMu.Lock()
	assert.Len(t, resultsReceived, 0)
	resultMu.Unlock()
}

// TestCFindSCP_LargeResultSet tests C-FIND with many results
func TestCFindSCP_LargeResultSet(t *testing.T) {
	// Create large result set
	numResults := 100
	mockResults := make([]*dicom.DataSet, numResults)
	for i := 0; i < numResults; i++ {
		mockResults[i] = dicom.NewDataSet()
	}

	findHandler := scp.FindHandlerFunc(func(ctx context.Context, req *scp.FindRequest) *scp.FindResponse {
		return &scp.FindResponse{
			Results: mockResults,
			Status:  dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "FIND_SCP",
		ListenAddr: "127.0.0.1:11121",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":           {"1.2.840.10008.1.2"},
			"1.2.840.10008.5.1.4.1.2.1.1": {"1.2.840.10008.1.2"},
		},
		FindHandler: findHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.2.1.1"

	clientConfig := scu.Config{
		CallingAETitle: "FIND_SCU",
		CalledAETitle:  "FIND_SCP",
		RemoteAddr:     "127.0.0.1:11121",
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

	var resultsReceived []*dicom.DataSet
	var resultMu sync.Mutex

	err = client.Find(ctx, "PATIENT", sopClassUID, query, func(ds *dicom.DataSet) error {
		resultMu.Lock()
		resultsReceived = append(resultsReceived, ds)
		resultMu.Unlock()
		return nil
	})
	assert.NoError(t, err)

	// Verify all results were received
	resultMu.Lock()
	assert.Len(t, resultsReceived, numResults)
	resultMu.Unlock()
}

// TestCFindSCP_CallbackError tests C-FIND when callback returns error
func TestCFindSCP_CallbackError(t *testing.T) {
	mockResults := []*dicom.DataSet{
		dicom.NewDataSet(),
		dicom.NewDataSet(),
	}

	findHandler := scp.FindHandlerFunc(func(ctx context.Context, req *scp.FindRequest) *scp.FindResponse {
		return &scp.FindResponse{
			Results: mockResults,
			Status:  dimse.StatusSuccess,
		}
	})

	serverConfig := scp.Config{
		AETitle:    "FIND_SCP",
		ListenAddr: "127.0.0.1:11122",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":           {"1.2.840.10008.1.2"},
			"1.2.840.10008.5.1.4.1.2.1.1": {"1.2.840.10008.1.2"},
		},
		FindHandler: findHandler,
	}

	server, err := scp.NewServer(serverConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = server.Listen(ctx)
	require.NoError(t, err)
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond)

	sopClassUID := "1.2.840.10008.5.1.4.1.2.1.1"

	clientConfig := scu.Config{
		CallingAETitle: "FIND_SCU",
		CalledAETitle:  "FIND_SCP",
		RemoteAddr:     "127.0.0.1:11122",
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

	// Callback returns error on first result
	err = client.Find(ctx, "PATIENT", sopClassUID, query, func(ds *dicom.DataSet) error {
		return assert.AnError // Return error
	})

	// Should receive error from callback
	assert.Error(t, err)
}
