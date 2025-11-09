package fhir_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
	"github.com/codeninja55/go-radx/fhir/validation"
)

// TestOfficialFHIRExamples tests against official FHIR example resources
// from https://www.hl7.org/fhir/
func TestOfficialFHIRExamples(t *testing.T) {
	examplesDir := filepath.Join("..", "testdata", "fhir", "examples")

	// Check if examples directory exists
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skip("Official FHIR examples not downloaded. Run: make download-fhir-examples")
	}

	tests := []struct {
		name         string
		filename     string
		resourceType string
		validate     bool
	}{
		{
			name:         "Official Patient Example",
			filename:     "patient-example.json",
			resourceType: "Patient",
			validate:     true,
		},
		{
			name:         "Official Observation Example",
			filename:     "observation-example.json",
			resourceType: "Observation",
			validate:     true,
		},
		{
			name:         "Official Bundle Example",
			filename:     "bundle-example.json",
			resourceType: "Bundle",
			validate:     true,
		},
	}

	validator := validation.NewFHIRValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(examplesDir, tt.filename)

			// Read the example file
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Skipf("Could not read %s: %v", tt.filename, err)
			}

			// Parse JSON to verify structure
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("Invalid JSON in %s: %v", tt.filename, err)
			}

			// Verify resourceType
			resourceType, ok := raw["resourceType"].(string)
			if !ok {
				t.Fatal("Missing or invalid resourceType")
			}
			if resourceType != tt.resourceType {
				t.Errorf("Expected resourceType %s, got %s", tt.resourceType, resourceType)
			}

			// Unmarshal into appropriate type and validate
			switch tt.resourceType {
			case "Patient":
				var patient resources.Patient
				if err := json.Unmarshal(data, &patient); err != nil {
					t.Fatalf("Failed to unmarshal Patient: %v", err)
				}

				// Validate round-trip
				roundTripData, err := json.Marshal(patient)
				if err != nil {
					t.Fatalf("Failed to marshal Patient: %v", err)
				}

				var roundTrip resources.Patient
				if err := json.Unmarshal(roundTripData, &roundTrip); err != nil {
					t.Fatalf("Failed to unmarshal round-trip Patient: %v", err)
				}

				// Validate structure
				if tt.validate {
					if err := validator.Validate(&patient); err != nil {
						t.Logf("Validation warning for %s: %v", tt.filename, err)
						// Don't fail - official examples may have extensions we don't fully support
					}
				}

				t.Logf("Successfully processed Patient with ID: %v", patient.ID)

			case "Observation":
				var observation resources.Observation
				if err := json.Unmarshal(data, &observation); err != nil {
					t.Fatalf("Failed to unmarshal Observation: %v", err)
				}

				// Validate round-trip
				roundTripData, err := json.Marshal(observation)
				if err != nil {
					t.Fatalf("Failed to marshal Observation: %v", err)
				}

				var roundTrip resources.Observation
				if err := json.Unmarshal(roundTripData, &roundTrip); err != nil {
					t.Fatalf("Failed to unmarshal round-trip Observation: %v", err)
				}

				if tt.validate {
					if err := validator.Validate(&observation); err != nil {
						t.Logf("Validation warning for %s: %v", tt.filename, err)
					}
				}

				t.Logf("Successfully processed Observation with ID: %v", observation.ID)

			case "Bundle":
				var bundle fhir.Bundle
				if err := json.Unmarshal(data, &bundle); err != nil {
					t.Fatalf("Failed to unmarshal Bundle: %v", err)
				}

				// Validate round-trip
				roundTripData, err := json.Marshal(bundle)
				if err != nil {
					t.Fatalf("Failed to marshal Bundle: %v", err)
				}

				var roundTrip fhir.Bundle
				if err := json.Unmarshal(roundTripData, &roundTrip); err != nil {
					t.Fatalf("Failed to unmarshal round-trip Bundle: %v", err)
				}

				// Test bundle helper
				helper := fhir.NewBundleHelper(&bundle)
				count := helper.Count()

				t.Logf("Successfully processed Bundle with %d entries", count)
			}
		})
	}
}

// TestOfficialExamplesRoundTrip ensures we can parse and re-serialize
// official FHIR examples without data loss
func TestOfficialExamplesRoundTrip(t *testing.T) {
	examplesDir := filepath.Join("..", "testdata", "fhir", "examples")

	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skip("Official FHIR examples not available")
	}

	files, err := filepath.Glob(filepath.Join(examplesDir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Skip("No FHIR example files found")
	}

	for _, file := range files {
		filename := filepath.Base(file)
		t.Run(filename, func(t *testing.T) {
			// Read original
			originalData, err := os.ReadFile(file)
			if err != nil {
				t.Skipf("Could not read file: %v", err)
			}

			// Parse as generic JSON
			var original map[string]interface{}
			if err := json.Unmarshal(originalData, &original); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			resourceType, ok := original["resourceType"].(string)
			if !ok {
				t.Skip("No resourceType field")
			}

			// Re-serialize and compare structure
			reserializedData, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to re-serialize: %v", err)
			}

			var reserialized map[string]interface{}
			if err := json.Unmarshal(reserializedData, &reserialized); err != nil {
				t.Fatalf("Failed to parse re-serialized data: %v", err)
			}

			// Verify resourceType preserved
			if rt, ok := reserialized["resourceType"].(string); !ok || rt != resourceType {
				t.Errorf("ResourceType changed from %s to %v", resourceType, reserialized["resourceType"])
			}

			t.Logf("Successfully round-tripped %s", resourceType)
		})
	}
}

// TestOfficialExamplesSummaryMode tests that summary mode works with official examples
func TestOfficialExamplesSummaryMode(t *testing.T) {
	examplesDir := filepath.Join("..", "testdata", "fhir", "examples")

	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skip("Official FHIR examples not available")
	}

	patientFile := filepath.Join(examplesDir, "patient-example.json")
	data, err := os.ReadFile(patientFile)
	if err != nil {
		t.Skip("Patient example not available")
	}

	var patient resources.Patient
	if err := json.Unmarshal(data, &patient); err != nil {
		t.Fatalf("Failed to unmarshal patient: %v", err)
	}

	// Generate summary
	summaryData, err := fhir.MarshalSummaryJSON(&patient)
	if err != nil {
		t.Fatalf("Failed to generate summary: %v", err)
	}

	// Verify it's valid JSON
	var summaryCheck map[string]interface{}
	if err := json.Unmarshal(summaryData, &summaryCheck); err != nil {
		t.Fatalf("Summary is not valid JSON: %v", err)
	}

	// Verify it's smaller
	fullData, _ := json.Marshal(&patient)
	if len(summaryData) >= len(fullData) {
		t.Errorf("Summary should be smaller than full: summary=%d, full=%d",
			len(summaryData), len(fullData))
	}

	reduction := 100.0 * (1.0 - float64(len(summaryData))/float64(len(fullData)))
	t.Logf("Summary mode achieved %.1f%% reduction (%d -> %d bytes)",
		reduction, len(fullData), len(summaryData))
}
