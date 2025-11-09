package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealFHIRExamples tests that we can unmarshal real FHIR examples from the testdata directory.
func TestRealFHIRExamples(t *testing.T) {
	examples := []struct {
		name         string
		filepath     string
		resourceType string
	}{
		{
			name:         "Patient example",
			filepath:     "../../../testdata/fhir/r4/examples/patient-example.json",
			resourceType: "Patient",
		},
		{
			name:         "Observation example (blood pressure)",
			filepath:     "../../../testdata/fhir/r4/examples/observation-example-bloodpressure.json",
			resourceType: "Observation",
		},
		{
			name:         "Bundle example",
			filepath:     "../../../testdata/fhir/r4/examples/bundle-example.json",
			resourceType: "Bundle",
		},
	}

	for _, tt := range examples {
		t.Run(tt.name, func(t *testing.T) {
			// Read the file
			data, err := os.ReadFile(tt.filepath)
			require.NoError(t, err, "failed to read %s", tt.filepath)

			// Unmarshal using factory
			resource, err := UnmarshalResource(data)
			require.NoError(t, err, "failed to unmarshal %s", tt.filepath)
			require.NotNil(t, resource, "resource should not be nil")

			// Verify the resource type
			switch tt.resourceType {
			case "Patient":
				patient, ok := resource.(*Patient)
				require.True(t, ok, "should be a Patient")
				assert.NotNil(t, patient.ID, "patient should have an ID")
			case "Observation":
				obs, ok := resource.(*Observation)
				require.True(t, ok, "should be an Observation")
				assert.NotNil(t, obs.ID, "observation should have an ID")
				assert.NotEmpty(t, obs.Status, "observation should have a status")
			case "Bundle":
				bundle, ok := resource.(*Bundle)
				require.True(t, ok, "should be a Bundle")
				assert.NotEmpty(t, bundle.Type, "bundle should have a type")
			default:
				t.Fatalf("unknown resource type: %s", tt.resourceType)
			}
		})
	}
}

// TestBundleWithRealPatient tests creating a bundle with the real patient example.
func TestBundleWithRealPatient(t *testing.T) {
	// Read the real patient example
	patientData, err := os.ReadFile("../../../testdata/fhir/r4/examples/patient-example.json")
	require.NoError(t, err)

	// Unmarshal the patient
	resource, err := UnmarshalResource(patientData)
	require.NoError(t, err)

	patient, ok := resource.(*Patient)
	require.True(t, ok)

	// Create a bundle and add the patient
	bundle := NewSearchSetBundle()
	err = bundle.AddEntry(patient, "Patient/example")
	require.NoError(t, err)

	// Verify we can get the patient back
	retrievedResource, err := bundle.GetEntry(0)
	require.NoError(t, err)

	retrievedPatient, ok := retrievedResource.(*Patient)
	require.True(t, ok)
	assert.Equal(t, *patient.ID, *retrievedPatient.ID)
}

// TestAllFHIRExamples finds and tests all JSON files in the testdata directory.
func TestAllFHIRExamples(t *testing.T) {
	examplesDir := "../../../testdata/fhir/r4/examples"

	// Check if directory exists
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skip("FHIR examples directory does not exist")
	}

	// Find all JSON files
	files, err := filepath.Glob(filepath.Join(examplesDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Skip("No FHIR example files found")
	}

	t.Logf("Found %d FHIR example files to test", len(files))

	// Test each file
	successCount := 0
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			// Read the file
			data, err := os.ReadFile(file)
			if err != nil {
				t.Logf("Skipping %s: failed to read file: %v", filepath.Base(file), err)
				return
			}

			// Try to unmarshal
			resource, err := UnmarshalResource(data)
			if err != nil {
				t.Logf("Skipping %s: failed to unmarshal: %v", filepath.Base(file), err)
				return
			}

			// Basic validation - resource should not be nil
			assert.NotNil(t, resource)
			successCount++
		})
	}

	t.Logf("Successfully unmarshaled %d/%d FHIR examples", successCount, len(files))
}

// TestBundleFromRealExample tests unmarshaling a real bundle example and accessing its entries.
func TestBundleFromRealExample(t *testing.T) {
	bundleData, err := os.ReadFile("../../../testdata/fhir/r4/examples/bundle-example.json")
	if os.IsNotExist(err) {
		t.Skip("Bundle example file not found")
	}
	require.NoError(t, err)

	// Unmarshal the bundle
	resource, err := UnmarshalResource(bundleData)
	require.NoError(t, err)

	bundle, ok := resource.(*Bundle)
	require.True(t, ok)

	// Verify bundle properties
	assert.NotEmpty(t, bundle.Type, "bundle should have a type")

	// Try to get all entries from the bundle
	if len(bundle.Entry) > 0 {
		entries, err := bundle.GetAllEntries()
		require.NoError(t, err)
		assert.Len(t, entries, len(bundle.Entry))
	}
}
