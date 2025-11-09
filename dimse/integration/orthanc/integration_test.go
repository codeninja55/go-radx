package orthanc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/codeninja55/go-radx/dimse/dimse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/scp"
	"github.com/codeninja55/go-radx/dimse/scu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to set SOP Class UID using element API
func setSOPClassUID(ds *dicom.DataSet, uid string) error {
	val, err := value.NewStringValue(vr.UniqueIdentifier, []string{uid})
	if err != nil {
		return err
	}
	elem, err := element.NewElement(tag.New(0x0008, 0x0016), vr.UniqueIdentifier, val)
	if err != nil {
		return err
	}
	return ds.Add(elem)
}

// Helper to get SOP Class UID
func getSOPClassUID(ds *dicom.DataSet) (string, error) {
	elem, err := ds.Get(tag.New(0x0008, 0x0016))
	if err != nil {
		return "", err
	}
	return elem.Value().String(), nil
}

// Helper to get SOP Instance UID
func getSOPInstanceUID(ds *dicom.DataSet) (string, error) {
	elem, err := ds.Get(tag.SOPInstanceUID)
	if err != nil {
		return "", err
	}
	return elem.Value().String(), nil
}

// Helper to get Patient Name
func getPatientName(ds *dicom.DataSet) (string, error) {
	elem, err := ds.Get(tag.PatientName)
	if err != nil {
		return "", err
	}
	return elem.Value().String(), nil
}

// Test helper to create standard presentation contexts
func standardPresentationContexts() []dul.PresentationContextRQ {
	return []dul.PresentationContextRQ{
		{
			ID:             1,
			AbstractSyntax: "1.2.840.10008.1.1", // Verification SOP Class
			TransferSyntaxes: []string{
				"1.2.840.10008.1.2",   // Implicit VR Little Endian
				"1.2.840.10008.1.2.1", // Explicit VR Little Endian
			},
		},
		{
			ID:             3,
			AbstractSyntax: "1.2.840.10008.5.1.4.1.1.2", // CT Image Storage
			TransferSyntaxes: []string{
				"1.2.840.10008.1.2",
				"1.2.840.10008.1.2.1",
			},
		},
		{
			ID:             5,
			AbstractSyntax: "1.2.840.10008.5.1.4.1.2.1.1", // Patient Root Query/Retrieve - FIND
			TransferSyntaxes: []string{
				"1.2.840.10008.1.2",
			},
		},
		{
			ID:             7,
			AbstractSyntax: "1.2.840.10008.5.1.4.1.2.1.3", // Patient Root Query/Retrieve - GET
			TransferSyntaxes: []string{
				"1.2.840.10008.1.2",
			},
		},
	}
}

// TestOrthancIntegration_CEcho tests C-ECHO operation against Orthanc
func TestOrthancIntegration_CEcho(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start Orthanc container
	orth, err := StartOrthanc(ctx)
	require.NoError(t, err, "failed to start Orthanc")
	defer orth.Stop(context.Background())

	// Create SCU client
	client := scu.NewClient(scu.Config{
		CallingAETitle:       "TEST_SCU",
		CalledAETitle:        "ORTHANC",
		RemoteAddr:           orth.DICOMAddress(),
		MaxPDULength:         16384,
		PresentationContexts: standardPresentationContexts(),
	})

	// Connect
	err = client.Connect(ctx)
	require.NoError(t, err, "failed to connect")
	defer client.Close(context.Background())

	// Perform C-ECHO
	err = client.Echo(ctx)
	assert.NoError(t, err, "C-ECHO should succeed")
}

// TestOrthancIntegration_CStore tests C-STORE operation with REST verification
func TestOrthancIntegration_CStore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start Orthanc container
	orth, err := StartOrthanc(ctx)
	require.NoError(t, err, "failed to start Orthanc")
	defer orth.Stop(context.Background())

	// Create test dataset
	ds := dicom.NewDataSet()
	err = setSOPClassUID(ds, "1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	require.NoError(t, err)
	err = ds.SetSOPInstanceUID("1.2.840.113619.2.55.3.123456789.1")
	require.NoError(t, err)
	err = ds.SetPatientName("TestPatient^Integration")
	require.NoError(t, err)
	err = ds.SetPatientID("TEST001")
	require.NoError(t, err)
	err = ds.SetStudyInstanceUID("1.2.840.113619.2.55.3.123456789.100")
	require.NoError(t, err)
	err = ds.SetSeriesInstanceUID("1.2.840.113619.2.55.3.123456789.200")
	require.NoError(t, err)

	// Create SCU client
	client := scu.NewClient(scu.Config{
		CallingAETitle:       "TEST_SCU",
		CalledAETitle:        "ORTHANC",
		RemoteAddr:           orth.DICOMAddress(),
		MaxPDULength:         16384,
		PresentationContexts: standardPresentationContexts(),
	})

	// Connect
	err = client.Connect(ctx)
	require.NoError(t, err, "failed to connect")
	defer client.Close(context.Background())

	// Perform C-STORE
	sopClassUID, _ := getSOPClassUID(ds)
	sopInstanceUID, _ := getSOPInstanceUID(ds)
	err = client.Store(ctx, ds, sopClassUID, sopInstanceUID)
	assert.NoError(t, err, "C-STORE should succeed")

	// Verify via REST API
	// Give Orthanc a moment to process
	time.Sleep(500 * time.Millisecond)

	instances, err := orth.GetInstances(ctx)
	require.NoError(t, err, "failed to get instances from Orthanc")
	assert.NotEmpty(t, instances, "Orthanc should contain the stored instance")
}

// TestOrthancIntegration_CFind tests C-FIND operation
func TestOrthancIntegration_CFind(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start Orthanc container
	orth, err := StartOrthanc(ctx)
	require.NoError(t, err, "failed to start Orthanc")
	defer orth.Stop(context.Background())

	// First, store a test dataset
	ds := dicom.NewDataSet()
	_ = setSOPClassUID(ds, "1.2.840.10008.5.1.4.1.1.2")
	_ = ds.SetSOPInstanceUID("1.2.840.113619.2.55.3.987654321.1")
	_ = ds.SetPatientName("FindTest^Patient")
	_ = ds.SetPatientID("FIND001")
	_ = ds.SetStudyInstanceUID("1.2.840.113619.2.55.3.987654321.100")
	_ = ds.SetSeriesInstanceUID("1.2.840.113619.2.55.3.987654321.200")

	client := scu.NewClient(scu.Config{
		CallingAETitle:       "TEST_SCU",
		CalledAETitle:        "ORTHANC",
		RemoteAddr:           orth.DICOMAddress(),
		MaxPDULength:         16384,
		PresentationContexts: standardPresentationContexts(),
	})

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(context.Background())

	sopClassUID, _ := getSOPClassUID(ds)
	sopInstanceUID, _ := getSOPInstanceUID(ds)
	_ = client.Store(ctx, ds, sopClassUID, sopInstanceUID)

	// Wait for indexing
	time.Sleep(time.Second)

	// Perform C-FIND (Patient Root Query/Retrieve)
	query := dicom.NewDataSet()
	_ = query.SetPatientName("FindTest^Patient")
	_ = query.SetPatientID("") // Return all patient IDs

	results := 0
	err = client.Find(ctx, "PATIENT", "1.2.840.10008.5.1.4.1.2.1.1", query, func(result *dicom.DataSet) error {
		results++
		// Verify result contains expected data
		patientName, _ := getPatientName(result)
		assert.Contains(t, patientName, "FindTest", "Result should contain query patient name")
		return nil
	})

	assert.NoError(t, err, "C-FIND should succeed")
	assert.Greater(t, results, 0, "Should find at least one result")
}

// TestOrthancIntegration_CGet tests C-GET operation
func TestOrthancIntegration_CGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start Orthanc container
	orth, err := StartOrthanc(ctx)
	require.NoError(t, err, "failed to start Orthanc")
	defer orth.Stop(context.Background())

	// First, store a test dataset
	ds := dicom.NewDataSet()
	_ = setSOPClassUID(ds, "1.2.840.10008.5.1.4.1.1.2")
	_ = ds.SetSOPInstanceUID("1.2.840.113619.2.55.3.111222333.1")
	_ = ds.SetPatientName("GetTest^Patient")
	_ = ds.SetPatientID("GET001")
	studyUID := "1.2.840.113619.2.55.3.111222333.100"
	_ = ds.SetStudyInstanceUID(studyUID)
	_ = ds.SetSeriesInstanceUID("1.2.840.113619.2.55.3.111222333.200")

	client := scu.NewClient(scu.Config{
		CallingAETitle:       "TEST_SCU",
		CalledAETitle:        "ORTHANC",
		RemoteAddr:           orth.DICOMAddress(),
		MaxPDULength:         16384,
		PresentationContexts: standardPresentationContexts(),
	})

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(context.Background())

	sopClassUID, _ := getSOPClassUID(ds)
	sopInstanceUID, _ := getSOPInstanceUID(ds)
	_ = client.Store(ctx, ds, sopClassUID, sopInstanceUID)

	// Wait for indexing
	time.Sleep(time.Second)

	// Perform C-GET (retrieve the study we just stored)
	query := dicom.NewDataSet()
	_ = query.SetStudyInstanceUID(studyUID)

	retrievedCount := 0
	err = client.Get(ctx, "1.2.840.10008.5.1.4.1.2.1.3", query, func(retrieved *dicom.DataSet) error {
		retrievedCount++
		// Verify retrieved dataset
		patientName, _ := getPatientName(retrieved)
		assert.Contains(t, patientName, "GetTest", "Retrieved dataset should match")
		return nil
	})

	assert.NoError(t, err, "C-GET should succeed")
	assert.Equal(t, 1, retrievedCount, "Should retrieve exactly one instance")
}

// TestOrthancIntegration_SCPReceive tests receiving C-STORE from Orthanc as SCP
func TestOrthancIntegration_SCPReceive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start Orthanc container
	orth, err := StartOrthanc(ctx)
	require.NoError(t, err, "failed to start Orthanc")
	defer orth.Stop(context.Background())

	// Track received instances
	var mu sync.Mutex
	var receivedInstances []string

	// Configure and start SCP server
	scpServer, err := scp.NewServer(scp.Config{
		AETitle:    "TEST_SCP",
		ListenAddr: "0.0.0.0:11119",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":         {"1.2.840.10008.1.2"}, // Verification
			"1.2.840.10008.5.1.4.1.1.2": {"1.2.840.10008.1.2"}, // CT Image Storage
		},
		StoreHandler: scp.StoreHandlerFunc(func(ctx context.Context, req *scp.StoreRequest) *scp.StoreResponse {
			mu.Lock()
			defer mu.Unlock()
			receivedInstances = append(receivedInstances, req.SOPInstanceUID)
			return &scp.StoreResponse{Status: dimse.StatusSuccess}
		}),
	})
	require.NoError(t, err, "failed to create SCP server")

	err = scpServer.Listen(ctx)
	require.NoError(t, err, "failed to start SCP server")
	defer scpServer.Shutdown(context.Background())

	// Wait for server to be ready
	time.Sleep(500 * time.Millisecond)

	// First, store a test dataset to Orthanc
	ds := dicom.NewDataSet()
	_ = setSOPClassUID(ds, "1.2.840.10008.5.1.4.1.1.2")
	sopInstanceUID := "1.2.840.113619.2.55.3.444555666.1"
	_ = ds.SetSOPInstanceUID(sopInstanceUID)
	_ = ds.SetPatientName("SCPTest^Patient")
	_ = ds.SetPatientID("SCP001")
	_ = ds.SetStudyInstanceUID("1.2.840.113619.2.55.3.444555666.100")
	_ = ds.SetSeriesInstanceUID("1.2.840.113619.2.55.3.444555666.200")

	client := scu.NewClient(scu.Config{
		CallingAETitle:       "TEST_SCU",
		CalledAETitle:        "ORTHANC",
		RemoteAddr:           orth.DICOMAddress(),
		MaxPDULength:         16384,
		PresentationContexts: standardPresentationContexts(),
	})

	err = client.Connect(ctx)
	require.NoError(t, err)
	defer client.Close(context.Background())

	sopClassUID, _ := getSOPClassUID(ds)
	err = client.Store(ctx, ds, sopClassUID, sopInstanceUID)
	require.NoError(t, err, "failed to store to Orthanc")

	// Wait for Orthanc to index
	time.Sleep(time.Second)

	// Configure Orthanc to send to our SCP using REST API
	err = orth.ConfigureModality(ctx, "TEST_SCP", "localhost", 11119)
	require.NoError(t, err, "failed to configure modality")

	// Trigger C-STORE from Orthanc to our SCP
	err = orth.SendToModality(ctx, "TEST_SCP", sopInstanceUID)
	require.NoError(t, err, "failed to trigger C-STORE")

	// Wait for transmission
	time.Sleep(2 * time.Second)

	// Verify our SCP received the instance
	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, receivedInstances, "SCP should have received instances")
	assert.Contains(t, receivedInstances, sopInstanceUID, "Should receive the correct instance")
}

// TestOrthancIntegration_CMove tests C-MOVE operation with SCP destination
func TestOrthancIntegration_CMove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start Orthanc container
	orth, err := StartOrthanc(ctx)
	require.NoError(t, err, "failed to start Orthanc")
	defer orth.Stop(context.Background())

	// Track received instances in our SCP
	var mu sync.Mutex
	var movedInstances []string

	// Configure and start SCP server to receive moved data
	scpServer, err := scp.NewServer(scp.Config{
		AETitle:    "MOVE_DEST_SCP",
		ListenAddr: "0.0.0.0:11120",
		SupportedContexts: map[string][]string{
			"1.2.840.10008.1.1":         {"1.2.840.10008.1.2"}, // Verification
			"1.2.840.10008.5.1.4.1.1.2": {"1.2.840.10008.1.2"}, // CT Image Storage
		},
		StoreHandler: scp.StoreHandlerFunc(func(ctx context.Context, req *scp.StoreRequest) *scp.StoreResponse {
			mu.Lock()
			defer mu.Unlock()
			movedInstances = append(movedInstances, req.SOPInstanceUID)
			return &scp.StoreResponse{Status: dimse.StatusSuccess}
		}),
	})
	require.NoError(t, err, "failed to create SCP server")

	err = scpServer.Listen(ctx)
	require.NoError(t, err, "failed to start SCP server")
	defer scpServer.Shutdown(context.Background())

	// Wait for SCP server to be ready
	time.Sleep(500 * time.Millisecond)

	// Store test dataset to Orthanc
	ds := dicom.NewDataSet()
	_ = setSOPClassUID(ds, "1.2.840.10008.5.1.4.1.1.2")
	sopInstanceUID := "1.2.840.113619.2.55.3.777888999.1"
	_ = ds.SetSOPInstanceUID(sopInstanceUID)
	_ = ds.SetPatientName("MoveTest^Patient")
	_ = ds.SetPatientID("MOVE001")
	studyUID := "1.2.840.113619.2.55.3.777888999.100"
	_ = ds.SetStudyInstanceUID(studyUID)
	_ = ds.SetSeriesInstanceUID("1.2.840.113619.2.55.3.777888999.200")

	// Store to Orthanc
	storeClient := scu.NewClient(scu.Config{
		CallingAETitle:       "TEST_SCU",
		CalledAETitle:        "ORTHANC",
		RemoteAddr:           orth.DICOMAddress(),
		MaxPDULength:         16384,
		PresentationContexts: standardPresentationContexts(),
	})

	err = storeClient.Connect(ctx)
	require.NoError(t, err)
	defer storeClient.Close(context.Background())

	sopClassUID, _ := getSOPClassUID(ds)
	err = storeClient.Store(ctx, ds, sopClassUID, sopInstanceUID)
	require.NoError(t, err, "failed to store to Orthanc")

	// Wait for Orthanc to index
	time.Sleep(time.Second)

	// Configure our SCP as a destination modality in Orthanc
	err = orth.ConfigureModality(ctx, "MOVE_DEST_SCP", "host.docker.internal", 11120)
	require.NoError(t, err, "failed to configure destination modality")

	// Create Move client with C-MOVE presentation context
	moveContexts := append(standardPresentationContexts(), dul.PresentationContextRQ{
		ID:             9,
		AbstractSyntax: "1.2.840.10008.5.1.4.1.2.1.2", // Patient Root Query/Retrieve - MOVE
		TransferSyntaxes: []string{
			"1.2.840.10008.1.2",
		},
	})

	moveClient := scu.NewClient(scu.Config{
		CallingAETitle:       "TEST_SCU",
		CalledAETitle:        "ORTHANC",
		RemoteAddr:           orth.DICOMAddress(),
		MaxPDULength:         16384,
		PresentationContexts: moveContexts,
	})

	err = moveClient.Connect(ctx)
	require.NoError(t, err, "failed to connect move client")
	defer moveClient.Close(context.Background())

	// Perform C-MOVE to send study to our SCP
	query := dicom.NewDataSet()
	_ = query.SetStudyInstanceUID(studyUID)

	err = moveClient.Move(ctx, "1.2.840.10008.5.1.4.1.2.1.2", "MOVE_DEST_SCP", query)
	require.NoError(t, err, "C-MOVE should succeed")

	// Wait for data transfer
	time.Sleep(2 * time.Second)

	// Verify our SCP received the moved instance
	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, movedInstances, "Destination SCP should have received moved instances")
	assert.Contains(t, movedInstances, sopInstanceUID, "Should receive the correct moved instance")
}
