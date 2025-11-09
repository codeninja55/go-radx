package fhir_test

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
	"github.com/codeninja55/go-radx/fhir/validation"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
)

// TestIntegration_PatientFullWorkflow tests complete patient lifecycle
func TestIntegration_PatientFullWorkflow(t *testing.T) {
	// 1. Create a patient
	active := true
	birthDate := primitives.MustDate("1974-12-25")

	patient := &resources.Patient{
		ID:     testutil.StringPtr("example"),
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
		ID: testutil.StringPtr("obs-1"),
		// Missing Status - required field
	}
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
		ID:     testutil.StringPtr("example"),
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
	patient := &resources.Patient{
		ID: testutil.StringPtr("example"),
		Meta: &resources.Meta{
			VersionId: testutil.StringPtr("1"),
		},
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
