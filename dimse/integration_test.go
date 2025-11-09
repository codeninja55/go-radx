package dimse_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/scp"
	"github.com/codeninja55/go-radx/dimse/scu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_CStoreWorkflow tests complete C-STORE workflow from SCU to SCP
func TestIntegration_CStoreWorkflow(t *testing.T) {
	// Setup: Create test dataset to store
	testDS := createTestDataSet(t)
	sopClass := "1.2.840.10008.5.1.4.1.1.2" // CT Image Storage
	sopInstance := "1.2.3.4.5.6.7.8.9"

	// Setup: Create SCP with store handler
	storedDS := &sync.Map{} // Thread-safe map to store received datasets
	storeHandler := scp.StoreHandlerFunc(func(ctx context.Context, req *scp.StoreRequest) *scp.StoreResponse {
		// Store the received dataset
		storedDS.Store(req.SOPInstanceUID, req.DataSet)

		// Verify request fields
		assert.Equal(t, sopClass, req.SOPClassUID)
		assert.Equal(t, sopInstance, req.SOPInstanceUID)

		return &scp.StoreResponse{Status: dimse.StatusSuccess}
	})

	// Start SCP server
	serverAddr := "127.0.0.1:11200"
	server, err := startIntegrationSCP(t, serverAddr, &integrationHandlers{
		store: storeHandler,
	})
	require.NoError(t, err)
	defer server.Shutdown(context.Background())

	// Create SCU client
	client := createIntegrationSCU(t, serverAddr, []string{sopClass})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test: Establish association
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(ctx)

	// Test: Perform C-STORE
	err = client.Store(ctx, testDS, sopClass, sopInstance)
	require.NoError(t, err)

	// Verify: Dataset was received by SCP
	stored, ok := storedDS.Load(sopInstance)
	require.True(t, ok, "Dataset should have been stored")

	storedDataSet := stored.(*dicom.DataSet)
	verifyDataSetsMatch(t, testDS, storedDataSet)
}

// TestIntegration_CFindWorkflow tests complete C-FIND workflow
func TestIntegration_CFindWorkflow(t *testing.T) {
	// Setup: Create test results
	expectedResults := []*dicom.DataSet{
		createPatientDataSet(t, "Smith^John", "PAT001", "19800101"),
		createPatientDataSet(t, "Doe^Jane", "PAT002", "19850615"),
		createPatientDataSet(t, "Johnson^Bob", "PAT003", "19901225"),
	}

	// Setup: Create SCP with find handler
	findHandler := scp.FindHandlerFunc(func(ctx context.Context, req *scp.FindRequest) *scp.FindResponse {
		// Verify query parameters
		assert.NotNil(t, req.Query)
		assert.NotEmpty(t, req.SOPClassUID)

		return &scp.FindResponse{
			Results: expectedResults,
			Status:  dimse.StatusSuccess,
		}
	})

	// Start SCP server
	serverAddr := "127.0.0.1:11201"
	server, err := startIntegrationSCP(t, serverAddr, &integrationHandlers{
		find: findHandler,
	})
	require.NoError(t, err)
	defer server.Shutdown(context.Background())

	// Create SCU client
	sopClass := "1.2.840.10008.5.1.4.1.2.1.1" // Patient Root Query/Retrieve - FIND
	client := createIntegrationSCU(t, serverAddr, []string{sopClass})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test: Establish association
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(ctx)

	// Test: Perform C-FIND
	query := dicom.NewDataSet()
	_ = query.SetPatientName("*") // Wildcard search

	receivedResults := make([]*dicom.DataSet, 0)
	err = client.Find(ctx, "PATIENT", sopClass, query, func(ds *dicom.DataSet) error {
		receivedResults = append(receivedResults, ds)
		return nil
	})
	require.NoError(t, err)

	// Verify: Correct number of results
	assert.Equal(t, len(expectedResults), len(receivedResults))

	// Verify: Result contents match
	for i, expected := range expectedResults {
		verifyDataSetsMatch(t, expected, receivedResults[i])
	}
}

// TestIntegration_AssociationLifecycle tests complete association establish/release cycle
func TestIntegration_AssociationLifecycle(t *testing.T) {
	// Setup: Track association events
	var associationEstablished bool
	var mu sync.Mutex

	// Create custom echo handler that tracks calls
	echoHandler := scp.EchoHandlerFunc(func(ctx context.Context, req *scp.EchoRequest) *scp.EchoResponse {
		mu.Lock()
		associationEstablished = true
		mu.Unlock()
		return &scp.EchoResponse{Status: dimse.StatusSuccess}
	})

	// Start SCP server
	serverAddr := "127.0.0.1:11202"
	server, err := startIntegrationSCP(t, serverAddr, &integrationHandlers{
		echo: echoHandler,
	})
	require.NoError(t, err)
	defer server.Shutdown(context.Background())

	// Create SCU client
	client := createIntegrationSCU(t, serverAddr, []string{"1.2.840.10008.1.1"}) // Verification SOP Class
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test: Establish association
	err = client.Connect(ctx)
	require.NoError(t, err)
	assert.NotNil(t, client, "Client should be connected")

	// Test: Perform operation (C-ECHO)
	err = client.Echo(ctx)
	require.NoError(t, err)

	// Verify: Association was established
	mu.Lock()
	assert.True(t, associationEstablished, "Association should have been established")
	mu.Unlock()

	// Test: Release association
	err = client.Close(ctx)
	require.NoError(t, err)

	// Give server time to process release
	time.Sleep(100 * time.Millisecond)

	// Note: In a real implementation, we'd track association release in the SCP
	// For now, we just verify that Close() succeeded without error
}

// TestIntegration_MultipleOperations tests multiple operations in a single association
func TestIntegration_MultipleOperations(t *testing.T) {
	// Setup: Create handlers
	echoCallCount := 0
	storeCallCount := 0
	var mu sync.Mutex

	echoHandler := scp.EchoHandlerFunc(func(ctx context.Context, req *scp.EchoRequest) *scp.EchoResponse {
		mu.Lock()
		echoCallCount++
		mu.Unlock()
		return &scp.EchoResponse{Status: dimse.StatusSuccess}
	})

	storeHandler := scp.StoreHandlerFunc(func(ctx context.Context, req *scp.StoreRequest) *scp.StoreResponse {
		mu.Lock()
		storeCallCount++
		mu.Unlock()
		return &scp.StoreResponse{Status: dimse.StatusSuccess}
	})

	// Start SCP server
	serverAddr := "127.0.0.1:11203"
	server, err := startIntegrationSCP(t, serverAddr, &integrationHandlers{
		echo:  echoHandler,
		store: storeHandler,
	})
	require.NoError(t, err)
	defer server.Shutdown(context.Background())

	// Create SCU client with multiple SOP classes
	client := createIntegrationSCU(t, serverAddr, []string{
		"1.2.840.10008.1.1",         // Verification SOP Class
		"1.2.840.10008.5.1.4.1.1.2", // CT Image Storage
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test: Establish association
	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(ctx)

	// Test: Perform multiple C-ECHO operations
	for i := 0; i < 3; i++ {
		err = client.Echo(ctx)
		require.NoError(t, err)
	}

	// Test: Perform multiple C-STORE operations
	for i := 0; i < 2; i++ {
		testDS := createTestDataSet(t)
		sopClass := "1.2.840.10008.5.1.4.1.1.2"
		sopInstance := fmt.Sprintf("1.2.3.4.5.6.7.8.%d", i)
		err = client.Store(ctx, testDS, sopClass, sopInstance)
		require.NoError(t, err)
	}

	// Verify: Correct number of calls
	mu.Lock()
	assert.Equal(t, 3, echoCallCount, "Should have received 3 C-ECHO requests")
	assert.Equal(t, 2, storeCallCount, "Should have received 2 C-STORE requests")
	mu.Unlock()
}

// Helper types and functions

type integrationHandlers struct {
	echo  scp.EchoHandler
	store scp.StoreHandler
	find  scp.FindHandler
	get   scp.GetHandler
	move  scp.MoveHandler
}

func startIntegrationSCP(t *testing.T, addr string, handlers *integrationHandlers) (*scp.Server, error) {
	t.Helper()

	config := scp.Config{
		AETitle:      "INTEGRATION_SCP",
		ListenAddr:   addr,
		MaxPDULength: 16384,
		EchoHandler:  handlers.echo,
		StoreHandler: handlers.store,
		FindHandler:  handlers.find,
		GetHandler:   handlers.get,
		MoveHandler:  handlers.move,
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":           {"1.2.840.10008.1.2"}, // Verification SOP Class
			"1.2.840.10008.5.1.4.1.1.2":   {"1.2.840.10008.1.2"}, // CT Image Storage
			"1.2.840.10008.5.1.4.1.2.1.1": {"1.2.840.10008.1.2"}, // Patient Root Query/Retrieve - FIND
			"1.2.840.10008.5.1.4.1.2.1.2": {"1.2.840.10008.1.2"}, // Patient Root Query/Retrieve - MOVE
			"1.2.840.10008.5.1.4.1.2.1.3": {"1.2.840.10008.1.2"}, // Patient Root Query/Retrieve - GET
		},
	}

	// Use default handlers if none provided
	if config.EchoHandler == nil {
		config.EchoHandler = scp.NewDefaultEchoHandler()
	}

	server, err := scp.NewServer(config)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	err = server.Listen(ctx)
	if err != nil {
		return nil, err
	}

	// Give server time to start listening
	time.Sleep(100 * time.Millisecond)

	return server, nil
}

func createIntegrationSCU(t *testing.T, addr string, abstractSyntaxes []string) *scu.Client {
	t.Helper()

	var contexts []dul.PresentationContextRQ
	for i, as := range abstractSyntaxes {
		contexts = append(contexts, dul.PresentationContextRQ{
			ID:               uint8((i * 2) + 1),
			AbstractSyntax:   as,
			TransferSyntaxes: []string{"1.2.840.10008.1.2"}, // Implicit VR Little Endian
		})
	}

	return scu.NewClient(scu.Config{
		CallingAETitle:       "INTEGRATION_SCU",
		CalledAETitle:        "INTEGRATION_SCP",
		RemoteAddr:           addr,
		MaxPDULength:         16384,
		PresentationContexts: contexts,
	})
}

func createTestDataSet(t *testing.T) *dicom.DataSet {
	t.Helper()

	ds := dicom.NewDataSet()
	_ = ds.SetPatientName("Test^Patient^Middle^^Dr")
	_ = ds.SetPatientID("TEST12345")
	_ = ds.SetPatientBirthDate("19800101")
	_ = ds.SetPatientSex("M")
	_ = ds.SetStudyInstanceUID("1.2.3.4.5")
	_ = ds.SetSeriesInstanceUID("1.2.3.4.5.6")

	return ds
}

func createPatientDataSet(t *testing.T, name, id, birthDate string) *dicom.DataSet {
	t.Helper()

	ds := dicom.NewDataSet()
	_ = ds.SetPatientName(name)
	_ = ds.SetPatientID(id)
	_ = ds.SetPatientBirthDate(birthDate)

	return ds
}

func verifyDataSetsMatch(t *testing.T, expected, actual *dicom.DataSet) {
	t.Helper()

	// Helper function to get string value from dataset
	getString := func(ds *dicom.DataSet, tag tag.Tag) string {
		elem, err := ds.Get(tag)
		if err != nil {
			return ""
		}
		return elem.Value().String()
	}

	// Compare common fields
	expectedName := getString(expected, tag.PatientName)
	actualName := getString(actual, tag.PatientName)
	if expectedName != "" {
		assert.Equal(t, expectedName, actualName, "PatientName should match")
	}

	expectedID := getString(expected, tag.PatientID)
	actualID := getString(actual, tag.PatientID)
	if expectedID != "" {
		assert.Equal(t, expectedID, actualID, "PatientID should match")
	}

	expectedBirthDate := getString(expected, tag.PatientBirthDate)
	actualBirthDate := getString(actual, tag.PatientBirthDate)
	if expectedBirthDate != "" {
		assert.Equal(t, expectedBirthDate, actualBirthDate, "PatientBirthDate should match")
	}
}
