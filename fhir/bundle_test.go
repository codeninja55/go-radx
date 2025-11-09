package fhir

import (
	"encoding/json"
	"testing"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
)

func createTestBundle() *Bundle {
	patient1 := map[string]interface{}{
		"resourceType": "Patient",
		"id":           "patient-1",
		"active":       true,
	}
	patient1JSON, _ := json.Marshal(patient1)

	patient2 := map[string]interface{}{
		"resourceType": "Patient",
		"id":           "patient-2",
		"active":       false,
	}
	patient2JSON, _ := json.Marshal(patient2)

	obs1 := map[string]interface{}{
		"resourceType": "Observation",
		"id":           "obs-1",
		"status":       "final",
	}
	obs1JSON, _ := json.Marshal(obs1)

	total := 3
	bundle := &Bundle{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType: "Bundle",
			},
		},
		Type:  "searchset",
		Total: &total,
		Entry: []BundleEntry{
			{
				FullURL:  testutil.StringPtr("http://example.org/Patient/patient-1"),
				Resource: patient1JSON,
			},
			{
				FullURL:  testutil.StringPtr("http://example.org/Patient/patient-2"),
				Resource: patient2JSON,
			},
			{
				FullURL:  testutil.StringPtr("http://example.org/Observation/obs-1"),
				Resource: obs1JSON,
			},
		},
		Link: []BundleLink{
			{Relation: "self", URL: "http://example.org/Patient"},
			{Relation: "next", URL: "http://example.org/Patient?page=2"},
		},
	}

	return bundle
}

func TestBundleHelper_FindResourcesByType(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	// Find patients
	patients, err := helper.FindResourcesByType("Patient")
	if err != nil {
		t.Fatalf("FindResourcesByType() error = %v", err)
	}
	if len(patients) != 2 {
		t.Errorf("Expected 2 patients, got %d", len(patients))
	}

	// Find observations
	observations, err := helper.FindResourcesByType("Observation")
	if err != nil {
		t.Fatalf("FindResourcesByType() error = %v", err)
	}
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation, got %d", len(observations))
	}

	// Find non-existent type
	conditions, err := helper.FindResourcesByType("Condition")
	if err != nil {
		t.Fatalf("FindResourcesByType() error = %v", err)
	}
	if len(conditions) != 0 {
		t.Errorf("Expected 0 conditions, got %d", len(conditions))
	}
}

func TestBundleHelper_GetResourceByID(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	// Find existing patient
	resource, err := helper.GetResourceByID("Patient", "patient-1")
	if err != nil {
		t.Fatalf("GetResourceByID() error = %v", err)
	}
	if resource == nil {
		t.Fatal("Expected resource, got nil")
	}

	var patient map[string]interface{}
	_ = json.Unmarshal(resource, &patient)
	if patient["id"] != "patient-1" {
		t.Errorf("Expected id 'patient-1', got %v", patient["id"])
	}

	// Find non-existent resource
	resource, err = helper.GetResourceByID("Patient", "non-existent")
	if err != nil {
		t.Fatalf("GetResourceByID() error = %v", err)
	}
	if resource != nil {
		t.Error("Expected nil for non-existent resource")
	}
}

func TestBundleHelper_ResolveReference(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	tests := []struct {
		name      string
		reference string
		wantFound bool
	}{
		{
			name:      "by fullUrl",
			reference: "http://example.org/Patient/patient-1",
			wantFound: true,
		},
		{
			name:      "by relative reference",
			reference: "Patient/patient-1",
			wantFound: true,
		},
		{
			name:      "observation by relative reference",
			reference: "Observation/obs-1",
			wantFound: true,
		},
		{
			name:      "non-existent reference",
			reference: "Patient/non-existent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := helper.ResolveReference(tt.reference)
			if tt.wantFound {
				if err != nil {
					t.Errorf("ResolveReference() error = %v", err)
				}
				if resource == nil {
					t.Error("Expected resource, got nil")
				}
			} else {
				// Not found is ok
				if resource != nil && err == nil {
					t.Error("Expected nil resource for non-existent reference")
				}
			}
		})
	}
}

func TestBundleHelper_AddEntry(t *testing.T) {
	bundle := &Bundle{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType: "Bundle",
			},
		},
		Type:  "collection",
		Entry: []BundleEntry{},
	}
	helper := NewBundleHelper(bundle)

	initialCount := helper.Count()

	// Add a patient
	patient := map[string]interface{}{
		"resourceType": "Patient",
		"id":           "new-patient",
		"active":       true,
	}

	err := helper.AddEntry(patient, testutil.StringPtr("http://example.org/Patient/new-patient"))
	if err != nil {
		t.Fatalf("AddEntry() error = %v", err)
	}

	// Check count increased
	if helper.Count() != initialCount+1 {
		t.Errorf("Expected count %d, got %d", initialCount+1, helper.Count())
	}

	// Check total was updated
	if bundle.Total == nil {
		t.Error("Expected total to be set")
	} else if *bundle.Total != 1 {
		t.Errorf("Expected total 1, got %d", *bundle.Total)
	}

	// Verify the resource was added
	resource, err := helper.GetResourceByID("Patient", "new-patient")
	if err != nil {
		t.Fatalf("GetResourceByID() error = %v", err)
	}
	if resource == nil {
		t.Error("Expected to find newly added patient")
	}
}

func TestBundleHelper_TypeSpecificGetters(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	// Test GetPatients
	patients, err := helper.GetPatients()
	if err != nil {
		t.Fatalf("GetPatients() error = %v", err)
	}
	if len(patients) != 2 {
		t.Errorf("GetPatients() = %d, want 2", len(patients))
	}

	// Test GetObservations
	observations, err := helper.GetObservations()
	if err != nil {
		t.Fatalf("GetObservations() error = %v", err)
	}
	if len(observations) != 1 {
		t.Errorf("GetObservations() = %d, want 1", len(observations))
	}

	// Test getter for non-existent type
	conditions, err := helper.GetConditions()
	if err != nil {
		t.Fatalf("GetConditions() error = %v", err)
	}
	if len(conditions) != 0 {
		t.Errorf("GetConditions() = %d, want 0", len(conditions))
	}
}

func TestBundleHelper_GetAllResources(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	resources := helper.GetAllResources()
	if len(resources) != 3 {
		t.Errorf("GetAllResources() = %d, want 3", len(resources))
	}
}

func TestBundleHelper_GetResourceTypes(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	types, err := helper.GetResourceTypes()
	if err != nil {
		t.Fatalf("GetResourceTypes() error = %v", err)
	}

	if len(types) != 2 {
		t.Errorf("GetResourceTypes() = %d types, want 2", len(types))
	}

	// Check that Patient and Observation are in the list
	hasPatient := false
	hasObservation := false
	for _, t := range types {
		if t == "Patient" {
			hasPatient = true
		}
		if t == "Observation" {
			hasObservation = true
		}
	}

	if !hasPatient {
		t.Error("Expected Patient in resource types")
	}
	if !hasObservation {
		t.Error("Expected Observation in resource types")
	}
}

func TestBundleHelper_Count(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	count := helper.Count()
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestBundleHelper_CountByType(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	// Count patients
	count, err := helper.CountByType("Patient")
	if err != nil {
		t.Fatalf("CountByType() error = %v", err)
	}
	if count != 2 {
		t.Errorf("CountByType(Patient) = %d, want 2", count)
	}

	// Count observations
	count, err = helper.CountByType("Observation")
	if err != nil {
		t.Fatalf("CountByType() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CountByType(Observation) = %d, want 1", count)
	}

	// Count non-existent type
	count, err = helper.CountByType("Condition")
	if err != nil {
		t.Fatalf("CountByType() error = %v", err)
	}
	if count != 0 {
		t.Errorf("CountByType(Condition) = %d, want 0", count)
	}
}

func TestBundleHelper_GetLinks(t *testing.T) {
	bundle := createTestBundle()
	helper := NewBundleHelper(bundle)

	// Test GetNextLink
	nextLink := helper.GetNextLink()
	if nextLink == nil {
		t.Fatal("Expected next link, got nil")
	}
	if *nextLink != "http://example.org/Patient?page=2" {
		t.Errorf("GetNextLink() = %v, want http://example.org/Patient?page=2", *nextLink)
	}

	// Test GetSelfLink
	selfLink := helper.GetSelfLink()
	if selfLink == nil {
		t.Fatal("Expected self link, got nil")
	}
	if *selfLink != "http://example.org/Patient" {
		t.Errorf("GetSelfLink() = %v, want http://example.org/Patient", *selfLink)
	}

	// Test GetPreviousLink (not in test bundle)
	prevLink := helper.GetPreviousLink()
	if prevLink != nil {
		t.Errorf("GetPreviousLink() = %v, want nil", *prevLink)
	}
}

func TestBundleHelper_EmptyBundle(t *testing.T) {
	bundle := &Bundle{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType: "Bundle",
			},
		},
		Type:  "searchset",
		Entry: []BundleEntry{},
	}
	helper := NewBundleHelper(bundle)

	// Test with empty bundle
	if helper.Count() != 0 {
		t.Errorf("Count() = %d, want 0", helper.Count())
	}

	patients, _ := helper.GetPatients()
	if len(patients) != 0 {
		t.Errorf("GetPatients() on empty bundle = %d, want 0", len(patients))
	}

	types, _ := helper.GetResourceTypes()
	if len(types) != 0 {
		t.Errorf("GetResourceTypes() on empty bundle = %d, want 0", len(types))
	}
}

func TestBundleHelper_RoundTrip(t *testing.T) {
	// Create bundle, add entry, marshal, unmarshal, verify
	bundle := &Bundle{
		DomainResource: DomainResource{
			Resource: Resource{
				ResourceType: "Bundle",
			},
		},
		Type:  "collection",
		Entry: []BundleEntry{},
	}

	helper := NewBundleHelper(bundle)

	// Add multiple resources
	patient := map[string]interface{}{
		"resourceType": "Patient",
		"id":           "test-1",
	}
	_ = helper.AddEntry(patient, testutil.StringPtr("Patient/test-1"))

	observation := map[string]interface{}{
		"resourceType": "Observation",
		"id":           "obs-1",
	}
	_ = helper.AddEntry(observation, testutil.StringPtr("Observation/obs-1"))

	// Marshal bundle
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Unmarshal bundle
	var loaded Bundle
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Create helper for loaded bundle
	loadedHelper := NewBundleHelper(&loaded)

	// Verify contents
	if loadedHelper.Count() != 2 {
		t.Errorf("Loaded bundle count = %d, want 2", loadedHelper.Count())
	}

	patients, _ := loadedHelper.GetPatients()
	if len(patients) != 1 {
		t.Errorf("Loaded bundle patients = %d, want 1", len(patients))
	}

	observations, _ := loadedHelper.GetObservations()
	if len(observations) != 1 {
		t.Errorf("Loaded bundle observations = %d, want 1", len(observations))
	}
}
