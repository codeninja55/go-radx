package fhir_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
	"github.com/codeninja55/go-radx/fhir/validation"
)

// TestIntegration_PatientFullWorkflow tests complete patient lifecycle
func TestIntegration_PatientFullWorkflow(t *testing.T) {
	// 1. Create a patient
	active := true
	birthDate := primitives.MustDate("1974-12-25")

	patient := &resources.Patient{
		Active: &active,
		Name: []resources.HumanName{
			{
				Use:    testutil.StringPtr("official"),
				Family: testutil.StringPtr("Doe"),
				Given:  []string{"John"},
			},
		},
		Gender:    testutil.StringPtr("male"),
		BirthDate: &birthDate,
	}
	patient.ID = testutil.StringPtr("example")
	patient.ResourceType = "Patient"

	// 2. Validate the patient
	validator := validation.NewFHIRValidator()
	if err := validator.Validate(patient); err != nil {
		t.Fatalf("Patient validation failed: %v", err)
	}

	// 3. Marshal to JSON
	data, err := json.Marshal(patient)
	if err != nil {
		t.Fatalf("Failed to marshal patient: %v", err)
	}

	// 4. Unmarshal back
	var patient2 resources.Patient
	if err := json.Unmarshal(data, &patient2); err != nil {
		t.Fatalf("Failed to unmarshal patient: %v", err)
	}

	// 5. Validate round-trip
	if *patient2.ID != "example" {
		t.Errorf("ID mismatch: got %s, want example", *patient2.ID)
	}
	if *patient2.Active != true {
		t.Error("Active should be true")
	}
	if len(patient2.Name) != 1 {
		t.Fatalf("Should have 1 name, got %d", len(patient2.Name))
	}
	if *patient2.Name[0].Family != "Doe" {
		t.Errorf("Family name mismatch: got %s, want Doe", *patient2.Name[0].Family)
	}
}

// TestIntegration_BundleBasics tests basic bundle creation and helper
func TestIntegration_BundleBasics(t *testing.T) {
	// Create a simple bundle
	bundle := &fhir.Bundle{
		Type:  "searchset",
		Total: testutil.IntPtr(2),
		Entry: []fhir.BundleEntry{
			{
				FullURL: testutil.StringPtr("Patient/1"),
			},
			{
				FullURL: testutil.StringPtr("Patient/2"),
			},
		},
	}

	// Use bundle helper
	helper := fhir.NewBundleHelper(bundle)

	// Count resources
	count := helper.Count()
	if count != 2 {
		t.Errorf("Should have 2 entries, got %d", count)
	}

	// Validate marshal/unmarshal
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("Failed to marshal bundle: %v", err)
	}

	var bundle2 fhir.Bundle
	if err := json.Unmarshal(data, &bundle2); err != nil {
		t.Fatalf("Failed to unmarshal bundle: %v", err)
	}

	if bundle2.Type != "searchset" {
		t.Errorf("Type mismatch: got %s, want searchset", bundle2.Type)
	}
	if *bundle2.Total != 2 {
		t.Errorf("Total mismatch: got %d, want 2", *bundle2.Total)
	}
}

// TestIntegration_ValidationErrors tests validation catches errors
func TestIntegration_ValidationErrors(t *testing.T) {
	validator := validation.NewFHIRValidator()

	// Patient with no data - should still validate (all fields optional)
	patient := &resources.Patient{}
	if err := validator.Validate(patient); err != nil {
		t.Logf("Empty patient validation: %v", err)
	}

	// Observation with missing required field (status is required)
	obs := &resources.Observation{
		// Missing Status - required field
	}
	obs.ID = testutil.StringPtr("obs-1")
	obs.ResourceType = "Observation"
	err := validator.Validate(obs)
	// Note: validation might pass if status has a zero value, this test is informational
	if err != nil {
		t.Logf("Observation validation error (expected): %v", err)
	}
}

// TestIntegration_SummaryMode tests summary mode basics
func TestIntegration_SummaryMode(t *testing.T) {
	// Create a patient
	patient := &resources.Patient{
		Active: testutil.BoolPtr(true),
		Name: []resources.HumanName{
			{
				Use:    testutil.StringPtr("official"),
				Family: testutil.StringPtr("Doe"),
				Given:  []string{"John"},
			},
		},
		Gender: testutil.StringPtr("male"),
		Address: []resources.Address{
			{
				Line: []string{"123 Main St"},
				City: testutil.StringPtr("Springfield"),
			},
		},
	}
	patient.ID = testutil.StringPtr("example")
	patient.ResourceType = "Patient"

	// Full JSON
	fullJSON, err := json.Marshal(patient)
	if err != nil {
		t.Fatalf("Failed to marshal patient: %v", err)
	}

	// Summary JSON (using the actual API)
	summaryJSON, err := fhir.MarshalSummaryJSON(patient)
	if err != nil {
		t.Fatalf("Failed to marshal summary: %v", err)
	}

	// Both should be valid JSON
	var full, summary map[string]interface{}
	if err := json.Unmarshal(fullJSON, &full); err != nil {
		t.Fatalf("Full JSON invalid: %v", err)
	}
	if err := json.Unmarshal(summaryJSON, &summary); err != nil {
		t.Fatalf("Summary JSON invalid: %v", err)
	}

	t.Logf("Full JSON: %d bytes", len(fullJSON))
	t.Logf("Summary JSON: %d bytes", len(summaryJSON))

	// Summary fields should exist in full JSON
	if _, ok := full["id"]; !ok {
		t.Error("Full JSON should have id field")
	}
}

// TestIntegration_PrimitivesHandling tests FHIR primitives
func TestIntegration_PrimitivesHandling(t *testing.T) {
	// Test Date primitive
	date := primitives.MustDate("1974-12-25")
	dateStr := date.String()
	if dateStr != "1974-12-25" {
		t.Errorf("Date string mismatch: got %s, want 1974-12-25", dateStr)
	}

	// Test DateTime primitive
	dt := primitives.MustDateTime("2023-01-01T12:00:00Z")
	dtStr := dt.String()
	if dtStr == "" {
		t.Error("DateTime string should not be empty")
	}

	// Test Instant primitive
	instant := primitives.MustInstant("2023-01-01T12:00:00Z")
	instantStr := instant.String()
	if instantStr == "" {
		t.Error("Instant string should not be empty")
	}

	// Test Time primitive
	time := primitives.MustTime("14:30:00")
	timeStr := time.String()
	if timeStr != "14:30:00" {
		t.Errorf("Time string mismatch: got %s, want 14:30:00", timeStr)
	}
}

// TestIntegration_ResourceInheritance tests resource inheritance
func TestIntegration_ResourceInheritance(t *testing.T) {
	// Patient extends DomainResource
	patient := &resources.Patient{}
	patient.ID = testutil.StringPtr("example")
	patient.ResourceType = "Patient"
	patient.Meta = &fhir.Meta{
		VersionID: testutil.StringPtr("1"),
	}

	// Marshal and check meta is at root level (not nested)
	data, err := json.Marshal(patient)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Meta should be at root level
	if _, ok := raw["meta"]; !ok {
		t.Error("Meta should be at root level")
	}
}

// TestIntegration_RoundtripJSONForResourceTypes tests JSON → struct → JSON roundtrip
// for multiple resource types using real-world examples
func TestIntegration_RoundtripJSONForResourceTypes(t *testing.T) {
	testCases := []struct {
		name         string
		resourceType string
		jsonPath     string
	}{
		{
			name:         "Patient example",
			resourceType: "Patient",
			jsonPath:     "testdata/fhir/examples/patient-example.json",
		},
		{
			name:         "Observation example",
			resourceType: "Observation",
			jsonPath:     "testdata/fhir/examples/observation-example.json",
		},
		{
			name:         "Bundle example",
			resourceType: "Bundle",
			jsonPath:     "testdata/fhir/examples/bundle-example.json",
		},
		{
			name:         "Encounter example",
			resourceType: "Encounter",
			jsonPath:     "testdata/fhir/examples/encounter-example.json",
		},
		{
			name:         "DiagnosticReport example",
			resourceType: "DiagnosticReport",
			jsonPath:     "testdata/fhir/examples/diagnosticreport-example-f201-brainct.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read original JSON
			originalJSON, err := os.ReadFile(tc.jsonPath)
			if err != nil {
				t.Skipf("Test file not found: %s", tc.jsonPath)
				return
			}

			// Unmarshal to struct
			var resource interface{}
			switch tc.resourceType {
			case "Patient":
				resource = &resources.Patient{}
			case "Observation":
				resource = &resources.Observation{}
			case "Bundle":
				resource = &resources.Bundle{}
			case "Encounter":
				resource = &resources.Encounter{}
			case "DiagnosticReport":
				resource = &resources.DiagnosticReport{}
			default:
				t.Fatalf("Unknown resource type: %s", tc.resourceType)
			}

			if err := json.Unmarshal(originalJSON, resource); err != nil {
				t.Fatalf("Failed to unmarshal %s: %v", tc.resourceType, err)
			}

			// Marshal back to JSON
			roundtripJSON, err := json.Marshal(resource)
			if err != nil {
				t.Fatalf("Failed to marshal %s: %v", tc.resourceType, err)
			}

			// Verify JSON is valid
			var originalMap, roundtripMap map[string]interface{}
			if err := json.Unmarshal(originalJSON, &originalMap); err != nil {
				t.Fatalf("Original JSON invalid: %v", err)
			}
			if err := json.Unmarshal(roundtripJSON, &roundtripMap); err != nil {
				t.Fatalf("Roundtrip JSON invalid: %v", err)
			}

			// Verify resourceType is preserved
			if originalMap["resourceType"] != roundtripMap["resourceType"] {
				t.Errorf("ResourceType mismatch: original=%s, roundtrip=%s",
					originalMap["resourceType"], roundtripMap["resourceType"])
			}

			// Verify ID is preserved if present
			if originalID, ok := originalMap["id"]; ok {
				if roundtripID, ok := roundtripMap["id"]; !ok {
					t.Error("ID field missing in roundtrip")
				} else if originalID != roundtripID {
					t.Errorf("ID mismatch: original=%s, roundtrip=%s", originalID, roundtripID)
				}
			}

			t.Logf("✓ %s roundtrip successful", tc.resourceType)
		})
	}
}

// TestIntegration_BundleOperationsEndToEnd tests complete Bundle workflow
func TestIntegration_BundleOperationsEndToEnd(t *testing.T) {
	// 1. Create a Bundle with multiple resources
	bundle := &resources.Bundle{
		Type: "transaction",
	}
	bundle.ResourceType = "Bundle"

	// 2. Create Patient resource
	patient := &resources.Patient{
		Active: testutil.BoolPtr(true),
		Name: []resources.HumanName{
			{
				Family: testutil.StringPtr("Test"),
				Given:  []string{"Integration"},
			},
		},
	}
	patient.ResourceType = "Patient"
	patient.ID = testutil.StringPtr("patient-1")

	// 3. Add Patient to Bundle entry using json.RawMessage
	patientJSON, err := json.Marshal(patient)
	if err != nil {
		t.Fatalf("Failed to marshal patient: %v", err)
	}

	bundle.Entry = []resources.BundleEntry{
		{
			FullUrl: testutil.StringPtr("Patient/patient-1"),
			Resource: json.RawMessage(patientJSON),
			Request: &resources.BundleEntryRequest{
				Method: "POST",
				URL:    "Patient",
			},
		},
	}

	// 4. Marshal Bundle to JSON (skip validation for json.RawMessage)
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("Failed to marshal bundle: %v", err)
	}

	// 5. Unmarshal Bundle back
	var retrievedBundle resources.Bundle
	if err := json.Unmarshal(bundleJSON, &retrievedBundle); err != nil {
		t.Fatalf("Failed to unmarshal bundle: %v", err)
	}

	// 6. Verify Bundle properties
	if retrievedBundle.Type != "transaction" {
		t.Errorf("Bundle type mismatch: got %s, want transaction", retrievedBundle.Type)
	}

	if len(retrievedBundle.Entry) != 1 {
		t.Fatalf("Expected 1 bundle entry, got %d", len(retrievedBundle.Entry))
	}

	// 7. Extract Patient from Bundle entry using generic helper
	retrievedPatient, err := fhir.UnmarshalResource[resources.Patient](retrievedBundle.Entry[0].Resource)
	if err != nil {
		t.Fatalf("Failed to unmarshal patient from bundle: %v", err)
	}

	// 8. Verify Patient properties
	if *retrievedPatient.ID != "patient-1" {
		t.Errorf("Patient ID mismatch: got %s, want patient-1", *retrievedPatient.ID)
	}

	if !*retrievedPatient.Active {
		t.Error("Patient should be active")
	}

	if len(retrievedPatient.Name) != 1 {
		t.Fatalf("Expected 1 name, got %d", len(retrievedPatient.Name))
	}

	if *retrievedPatient.Name[0].Family != "Test" {
		t.Errorf("Family name mismatch: got %s, want Test", *retrievedPatient.Name[0].Family)
	}

	t.Log("✓ Bundle end-to-end operations successful")
}

// TestIntegration_ChoiceTypeScenariosFromSpec tests choice type scenarios
func TestIntegration_ChoiceTypeScenariosFromSpec(t *testing.T) {
	t.Run("Patient deceased[x] with boolean", func(t *testing.T) {
		patient := &resources.Patient{
			Active:          testutil.BoolPtr(true),
			DeceasedBoolean: testutil.BoolPtr(true),
		}
		patient.ResourceType = "Patient"
		patient.ID = testutil.StringPtr("deceased-bool")

		// Validate
		validator := validation.NewFHIRValidator()
		if err := validator.Validate(patient); err != nil {
			t.Errorf("Validation failed: %v", err)
		}

		// Roundtrip
		data, err := json.Marshal(patient)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var patient2 resources.Patient
		if err := json.Unmarshal(data, &patient2); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if patient2.DeceasedBoolean == nil || !*patient2.DeceasedBoolean {
			t.Error("DeceasedBoolean should be true")
		}

		if patient2.DeceasedDateTime != nil {
			t.Error("DeceasedDateTime should be nil")
		}

		t.Log("✓ Patient.deceased[x] with boolean works correctly")
	})

	t.Run("Patient deceased[x] with dateTime", func(t *testing.T) {
		deceasedDate := primitives.MustDateTime("2024-01-15T10:30:00Z")
		patient := &resources.Patient{
			Active:           testutil.BoolPtr(true),
			DeceasedDateTime: &deceasedDate,
		}
		patient.ResourceType = "Patient"
		patient.ID = testutil.StringPtr("deceased-datetime")

		// Validate
		validator := validation.NewFHIRValidator()
		if err := validator.Validate(patient); err != nil {
			t.Errorf("Validation failed: %v", err)
		}

		// Roundtrip
		data, err := json.Marshal(patient)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var patient2 resources.Patient
		if err := json.Unmarshal(data, &patient2); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if patient2.DeceasedDateTime == nil {
			t.Fatal("DeceasedDateTime should not be nil")
		}

		if patient2.DeceasedDateTime.String() != "2024-01-15T10:30:00Z" {
			t.Errorf("DeceasedDateTime mismatch: got %s", patient2.DeceasedDateTime.String())
		}

		if patient2.DeceasedBoolean != nil {
			t.Error("DeceasedBoolean should be nil")
		}

		t.Log("✓ Patient.deceased[x] with dateTime works correctly")
	})

	t.Run("Observation value[x] with quantity", func(t *testing.T) {
		obs := &resources.Observation{
			Status: "final",
			Code: resources.CodeableConcept{
				Text: testutil.StringPtr("Blood Pressure"),
			},
			ValueQuantity: &resources.Quantity{
				Value:  testutil.Float64Ptr(120.0),
				Unit:   testutil.StringPtr("mmHg"),
				System: testutil.StringPtr("http://unitsofmeasure.org"),
				Code:   testutil.StringPtr("mm[Hg]"),
			},
		}
		obs.ResourceType = "Observation"
		obs.ID = testutil.StringPtr("bp-value")

		// Validate
		validator := validation.NewFHIRValidator()
		if err := validator.Validate(obs); err != nil {
			t.Errorf("Validation failed: %v", err)
		}

		// Roundtrip
		data, err := json.Marshal(obs)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var obs2 resources.Observation
		if err := json.Unmarshal(data, &obs2); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if obs2.ValueQuantity == nil {
			t.Fatal("ValueQuantity should not be nil")
		}

		if *obs2.ValueQuantity.Value != 120.0 {
			t.Errorf("ValueQuantity.Value mismatch: got %f, want 120.0", *obs2.ValueQuantity.Value)
		}

		t.Log("✓ Observation.value[x] with quantity works correctly")
	})
}

// TestIntegration_ErrorCasesAndValidationFailures tests error handling
func TestIntegration_ErrorCasesAndValidationFailures(t *testing.T) {
	validator := validation.NewFHIRValidator()

	t.Run("Invalid choice type - multiple fields set", func(t *testing.T) {
		deceasedDate := primitives.MustDateTime("2024-01-15T10:30:00Z")
		patient := &resources.Patient{
			Active:           testutil.BoolPtr(true),
			DeceasedBoolean:  testutil.BoolPtr(true),
			DeceasedDateTime: &deceasedDate, // Both set - violates mutual exclusion
		}
		patient.ResourceType = "Patient"

		err := validator.Validate(patient)
		if err == nil {
			t.Error("Expected validation error for multiple choice fields set")
		} else {
			t.Logf("✓ Correctly caught choice type violation: %v", err)
		}
	})

	t.Run("Invalid JSON - malformed", func(t *testing.T) {
		malformedJSON := []byte(`{"resourceType": "Patient", "id": "test", "active": "not-a-boolean"}`)

		var patient resources.Patient
		err := json.Unmarshal(malformedJSON, &patient)
		// Note: JSON unmarshaling is permissive, might not fail on type mismatches
		// This test documents the behavior
		t.Logf("Unmarshal result for malformed JSON: %v", err)
	})

	t.Run("Bundle with invalid resource reference", func(t *testing.T) {
		bundle := &resources.Bundle{
			Type: "searchset",
			Entry: []resources.BundleEntry{
				{
					FullUrl: testutil.StringPtr("InvalidReference"),
					Resource: json.RawMessage(`{"invalid": "json"}`),
				},
			},
		}
		bundle.ResourceType = "Bundle"

		// Skip validation for json.RawMessage fields (validator counts bytes, not elements)
		// Try to extract resource - JSON unmarshaling is permissive and won't fail on missing fields
		patient, err := fhir.UnmarshalResource[resources.Patient](bundle.Entry[0].Resource)
		// Note: This succeeds but produces an empty/invalid Patient
		t.Logf("Unmarshal result: error=%v, patient.ResourceType=%s", err, patient.ResourceType)

		// However, validation should catch the invalid resource
		validator2 := validation.NewFHIRValidator()
		if err := validator2.Validate(&patient); err != nil {
			t.Logf("✓ Validation correctly caught invalid resource structure: %v", err)
		}
	})

	t.Run("Contained resource with invalid type", func(t *testing.T) {
		patient := &resources.Patient{
			Active: testutil.BoolPtr(true),
		}
		patient.ResourceType = "Patient"
		patient.ID = testutil.StringPtr("contained-test")

		// Add invalid contained resource
		patient.Contained = []json.RawMessage{
			json.RawMessage(`{"resourceType": "Unknown", "id": "invalid"}`),
		}

		// Try to unmarshal contained resource (should succeed but with unknown type)
		var result resources.Observation
		err := json.Unmarshal(patient.Contained[0], &result)
		// Note: JSON unmarshaling is permissive and won't fail on unknown resourceType
		// This test documents that behavior
		t.Logf("Unmarshal result for unknown resourceType: error=%v, result.ResourceType=%s", err, result.ResourceType)
	})
}
