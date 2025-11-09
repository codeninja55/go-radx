package fhir_test

import (
	"encoding/json"
	"fmt"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
	"github.com/codeninja55/go-radx/fhir/validation"
)

// ExamplePatient demonstrates creating a basic FHIR Patient resource.
func ExamplePatient() {
	birthDate := primitives.MustDate("1974-12-25")

	patient := &resources.Patient{
		ID:     stringPtr("example-patient"),
		Active: boolPtr(true),
		Name: []resources.HumanName{
			{
				Use:    stringPtr("official"),
				Family: stringPtr("Doe"),
				Given:  []string{"John"},
			},
		},
		Gender:    stringPtr("male"),
		BirthDate: &birthDate,
		Telecom: []resources.ContactPoint{
			{
				System: stringPtr("phone"),
				Value:  stringPtr("+1-555-1234"),
				Use:    stringPtr("home"),
			},
		},
	}

	// Marshal to JSON
	data, _ := json.MarshalIndent(patient, "", "  ")
	fmt.Println(string(data))
}

// ExampleObservation demonstrates creating a FHIR Observation resource
// for vital signs (heart rate).
func ExampleObservation() {
	obs := &resources.Observation{
		ID:     stringPtr("heart-rate-example"),
		Status: "final",
		Code: resources.CodeableConcept{
			Coding: []resources.Coding{
				{
					System:  stringPtr("http://loinc.org"),
					Code:    stringPtr("8867-4"),
					Display: stringPtr("Heart rate"),
				},
			},
			Text: stringPtr("Heart rate"),
		},
		Subject: &resources.Reference{
			Reference: stringPtr("Patient/example-patient"),
			Display:   stringPtr("John Doe"),
		},
	}

	data, _ := json.MarshalIndent(obs, "", "  ")
	fmt.Printf("Created observation: %s\n", *obs.ID)
	_ = data // data contains full JSON
	// Output: Created observation: heart-rate-example
}

// ExampleBundle_searchset demonstrates creating a Bundle for search results.
func ExampleBundle_searchset() {
	bundle := &fhir.Bundle{
		Type: "searchset",
	}

	helper := fhir.NewBundleHelper(bundle)

	// Add a patient
	patient := &resources.Patient{
		ID:     stringPtr("patient-1"),
		Active: boolPtr(true),
		Name: []resources.HumanName{
			{
				Family: stringPtr("Smith"),
				Given:  []string{"Jane"},
			},
		},
	}

	_ = helper.AddEntry(patient, stringPtr("Patient/patient-1"))

	fmt.Printf("Bundle type: %s, Total entries: %d\n", bundle.Type, len(bundle.Entry))
	// Output: Bundle type: searchset, Total entries: 1
}

// ExampleBundleHelper_GetPatients demonstrates filtering resources by type.
func ExampleBundleHelper_GetPatients() {
	bundle := &fhir.Bundle{Type: "searchset"}
	helper := fhir.NewBundleHelper(bundle)

	// Add multiple resources (using maps with resourceType until struct support is added)
	patient1 := map[string]interface{}{
		"resourceType": "Patient",
		"id":           "patient-1",
	}
	patient2 := map[string]interface{}{
		"resourceType": "Patient",
		"id":           "patient-2",
	}

	_ = helper.AddEntry(patient1, stringPtr("Patient/patient-1"))
	_ = helper.AddEntry(patient2, stringPtr("Patient/patient-2"))

	// Get all patients
	patients, _ := helper.GetPatients()
	fmt.Printf("Found %d patients\n", len(patients))
	// Output: Found 2 patients
}

// ExampleMarshalSummaryJSON demonstrates FHIR summary mode serialization.
func ExampleMarshalSummaryJSON() {
	patient := &resources.Patient{
		ID:     stringPtr("example"),
		Active: boolPtr(true),
		Name: []resources.HumanName{
			{
				Family: stringPtr("Doe"),
				Given:  []string{"John"},
			},
		},
	}

	// Marshal with summary mode (only summary elements)
	summaryData, _ := fhir.MarshalSummaryJSON(patient)

	fmt.Printf("Summary JSON size: %d bytes\n", len(summaryData))
	// The summary will be smaller than full JSON
}

// ExampleFHIRValidator demonstrates resource validation.
func ExampleFHIRValidator() {
	validator := validation.NewFHIRValidator()

	patient := &resources.Patient{
		ID:     stringPtr("valid-patient"),
		Active: boolPtr(true),
		Name: []resources.HumanName{
			{
				Family: stringPtr("Doe"),
			},
		},
	}

	err := validator.Validate(patient)
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("Validation passed")
	}
	// Output: Validation passed
}

// ExampleValidateReference demonstrates FHIR reference validation.
func ExampleValidateReference() {
	// Valid relative reference
	err := validation.ValidateReference("subject", "Patient/123")
	if err == nil {
		fmt.Println("Valid reference")
	}

	// Invalid reference format
	err = validation.ValidateReference("subject", "patient123")
	if err != nil {
		fmt.Printf("Invalid reference: %v\n", err)
	}
	// Output:
	// Valid reference
	// Invalid reference: subject: invalid reference format: patient123 (expected 'ResourceType/id')
}

// Helper functions for examples
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
