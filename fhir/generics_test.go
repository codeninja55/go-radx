package fhir_test

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
)

// TestUnmarshalResource tests the generic UnmarshalResource function
func TestUnmarshalResource(t *testing.T) {
	t.Run("unmarshal Patient resource", func(t *testing.T) {
		// Create a sample Patient JSON
		patientJSON := json.RawMessage(`{
			"resourceType": "Patient",
			"id": "example",
			"active": true,
			"name": [{
				"use": "official",
				"family": "Chalmers",
				"given": ["Peter", "James"]
			}],
			"gender": "male"
		}`)

		// Unmarshal using the generic function
		patient, err := fhir.UnmarshalResource[resources.Patient](patientJSON)
		if err != nil {
			t.Fatalf("UnmarshalResource failed: %v", err)
		}

		// Verify the patient was unmarshaled correctly
		if patient.ResourceType != "Patient" {
			t.Errorf("Expected ResourceType Patient, got %s", patient.ResourceType)
		}

		if patient.ID == nil || *patient.ID != "example" {
			t.Errorf("Expected ID 'example', got %v", patient.ID)
		}

		if patient.Active == nil || *patient.Active != true {
			t.Errorf("Expected Active true, got %v", patient.Active)
		}

		if patient.Gender == nil || *patient.Gender != "male" {
			t.Errorf("Expected Gender 'male', got %v", patient.Gender)
		}

		if len(patient.Name) != 1 {
			t.Errorf("Expected 1 name, got %d", len(patient.Name))
		}
	})

	t.Run("unmarshal Observation resource", func(t *testing.T) {
		observationJSON := json.RawMessage(`{
			"resourceType": "Observation",
			"id": "example-observation",
			"status": "final",
			"code": {
				"coding": [{
					"system": "http://loinc.org",
					"code": "85354-9",
					"display": "Blood pressure"
				}]
			}
		}`)

		observation, err := fhir.UnmarshalResource[resources.Observation](observationJSON)
		if err != nil {
			t.Fatalf("UnmarshalResource failed: %v", err)
		}

		if observation.ResourceType != "Observation" {
			t.Errorf("Expected ResourceType Observation, got %s", observation.ResourceType)
		}

		if observation.Status != "final" {
			t.Errorf("Expected Status 'final', got %s", observation.Status)
		}
	})

	t.Run("unmarshal invalid JSON returns error", func(t *testing.T) {
		invalidJSON := json.RawMessage(`{invalid json`)

		_, err := fhir.UnmarshalResource[resources.Patient](invalidJSON)
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})
}

// TestUnmarshalContainedResource tests the generic UnmarshalContainedResource function
func TestUnmarshalContainedResource(t *testing.T) {
	t.Run("unmarshal contained Patient at valid index", func(t *testing.T) {
		// Create contained resources
		patientJSON := json.RawMessage(`{
			"resourceType": "Patient",
			"id": "contained-patient",
			"name": [{
				"family": "Smith"
			}]
		}`)

		organizationJSON := json.RawMessage(`{
			"resourceType": "Organization",
			"id": "contained-org",
			"name": "Example Org"
		}`)

		contained := []json.RawMessage{patientJSON, organizationJSON}

		// Unmarshal the first contained resource (Patient)
		patient, err := fhir.UnmarshalContainedResource[resources.Patient](contained, 0)
		if err != nil {
			t.Fatalf("UnmarshalContainedResource failed: %v", err)
		}

		if patient.ID == nil || *patient.ID != "contained-patient" {
			t.Errorf("Expected ID 'contained-patient', got %v", patient.ID)
		}

		// Unmarshal the second contained resource (Organization)
		org, err := fhir.UnmarshalContainedResource[resources.Organization](contained, 1)
		if err != nil {
			t.Fatalf("UnmarshalContainedResource failed: %v", err)
		}

		if org.ID == nil || *org.ID != "contained-org" {
			t.Errorf("Expected ID 'contained-org', got %v", org.ID)
		}
	})

	t.Run("out of range index returns error", func(t *testing.T) {
		contained := []json.RawMessage{
			json.RawMessage(`{"resourceType": "Patient", "id": "test"}`),
		}

		_, err := fhir.UnmarshalContainedResource[resources.Patient](contained, 5)
		if err == nil {
			t.Error("Expected error for out of range index, got nil")
		}
	})

	t.Run("negative index returns error", func(t *testing.T) {
		contained := []json.RawMessage{
			json.RawMessage(`{"resourceType": "Patient", "id": "test"}`),
		}

		_, err := fhir.UnmarshalContainedResource[resources.Patient](contained, -1)
		if err == nil {
			t.Error("Expected error for negative index, got nil")
		}
	})
}

// TestAddContainedResource tests the generic AddContainedResource function
func TestAddContainedResource(t *testing.T) {
	t.Run("add Patient to contained resources", func(t *testing.T) {
		var contained []json.RawMessage

		// Create a Patient
		patient := resources.Patient{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Patient",
					ID:           testutil.StringPtr("patient-1"),
				},
			},
		}
		familyName := "Doe"
		patient.Name = []resources.HumanName{
			{
				Family: &familyName,
			},
		}

		// Add patient to contained resources
		var err error
		contained, err = fhir.AddContainedResource(contained, patient)
		if err != nil {
			t.Fatalf("AddContainedResource failed: %v", err)
		}

		if len(contained) != 1 {
			t.Errorf("Expected 1 contained resource, got %d", len(contained))
		}

		// Verify we can unmarshal it back
		retrieved, err := fhir.UnmarshalContainedResource[resources.Patient](contained, 0)
		if err != nil {
			t.Fatalf("Failed to unmarshal added resource: %v", err)
		}

		if retrieved.ID == nil || *retrieved.ID != "patient-1" {
			t.Errorf("Expected ID 'patient-1', got %v", retrieved.ID)
		}
	})

	t.Run("add multiple resources maintains order", func(t *testing.T) {
		var contained []json.RawMessage

		// Add Patient
		patient := resources.Patient{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Patient",
					ID:           testutil.StringPtr("patient-1"),
				},
			},
		}

		var err error
		contained, err = fhir.AddContainedResource(contained, patient)
		if err != nil {
			t.Fatalf("Failed to add patient: %v", err)
		}

		// Add Organization
		org := resources.Organization{
			DomainResource: fhir.DomainResource{
				Resource: fhir.Resource{
					ResourceType: "Organization",
					ID:           testutil.StringPtr("org-1"),
				},
			},
		}

		contained, err = fhir.AddContainedResource(contained, org)
		if err != nil {
			t.Fatalf("Failed to add organization: %v", err)
		}

		if len(contained) != 2 {
			t.Errorf("Expected 2 contained resources, got %d", len(contained))
		}

		// Verify order is maintained
		retrievedPatient, _ := fhir.UnmarshalContainedResource[resources.Patient](contained, 0)
		if retrievedPatient.ID == nil || *retrievedPatient.ID != "patient-1" {
			t.Error("First resource should be Patient")
		}

		retrievedOrg, _ := fhir.UnmarshalContainedResource[resources.Organization](contained, 1)
		if retrievedOrg.ID == nil || *retrievedOrg.ID != "org-1" {
			t.Error("Second resource should be Organization")
		}
	})
}

// TestGetContainedResourceByID tests the GetContainedResourceByID function
func TestGetContainedResourceByID(t *testing.T) {
	t.Run("find contained resource by ID", func(t *testing.T) {
		// Create contained resources with IDs
		patientJSON := json.RawMessage(`{
			"resourceType": "Patient",
			"id": "patient-123",
			"name": [{"family": "Smith"}]
		}`)

		organizationJSON := json.RawMessage(`{
			"resourceType": "Organization",
			"id": "org-456",
			"name": "Example Org"
		}`)

		contained := []json.RawMessage{patientJSON, organizationJSON}

		// Find the patient by ID
		raw, err := fhir.GetContainedResourceByID(contained, "patient-123")
		if err != nil {
			t.Fatalf("GetContainedResourceByID failed: %v", err)
		}

		// Verify we got the right resource
		patient, err := fhir.UnmarshalResource[resources.Patient](raw)
		if err != nil {
			t.Fatalf("Failed to unmarshal found resource: %v", err)
		}

		if patient.ID == nil || *patient.ID != "patient-123" {
			t.Errorf("Expected ID 'patient-123', got %v", patient.ID)
		}

		// Find the organization by ID
		raw, err = fhir.GetContainedResourceByID(contained, "org-456")
		if err != nil {
			t.Fatalf("GetContainedResourceByID failed: %v", err)
		}

		org, err := fhir.UnmarshalResource[resources.Organization](raw)
		if err != nil {
			t.Fatalf("Failed to unmarshal found resource: %v", err)
		}

		if org.ID == nil || *org.ID != "org-456" {
			t.Errorf("Expected ID 'org-456', got %v", org.ID)
		}
	})

	t.Run("ID not found returns error", func(t *testing.T) {
		patientJSON := json.RawMessage(`{
			"resourceType": "Patient",
			"id": "patient-123"
		}`)

		contained := []json.RawMessage{patientJSON}

		_, err := fhir.GetContainedResourceByID(contained, "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent ID, got nil")
		}
	})

	t.Run("resource without ID is skipped", func(t *testing.T) {
		// Resource without ID
		noIDJSON := json.RawMessage(`{
			"resourceType": "Patient"
		}`)

		// Resource with ID
		withIDJSON := json.RawMessage(`{
			"resourceType": "Organization",
			"id": "org-456"
		}`)

		contained := []json.RawMessage{noIDJSON, withIDJSON}

		// Should find the one with ID
		raw, err := fhir.GetContainedResourceByID(contained, "org-456")
		if err != nil {
			t.Fatalf("GetContainedResourceByID failed: %v", err)
		}

		org, _ := fhir.UnmarshalResource[resources.Organization](raw)
		if org.ID == nil || *org.ID != "org-456" {
			t.Error("Should have found organization with ID")
		}
	})
}
