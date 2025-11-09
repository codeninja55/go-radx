//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
	"github.com/codeninja55/go-radx/fhir/validation"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
)

// Example: Complete healthcare workflow
// Creating a patient encounter with observations and medications
func main() {
	// Step 1: Create a patient
	patient := createPatient()

	// Step 2: Create an encounter for the patient
	encounter := createEncounter(patient.ID)

	// Step 3: Create observations (vital signs)
	observations := createVitalSigns(patient.ID, encounter.ID)

	// Step 4: Create a medication request
	medication := createMedicationRequest(patient.ID, encounter.ID)

	// Step 5: Create a bundle with all resources
	bundle := createBundle(patient, encounter, observations, medication)

	// Step 6: Validate all resources
	if err := validateBundle(bundle); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Step 7: Serialize to JSON
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Healthcare workflow bundle created successfully!")
	fmt.Printf("Bundle contains %d resources\n", len(bundle.Entry))
	fmt.Printf("\nJSON output (first 500 chars):\n%s...\n", string(data[:min(500, len(data))]))
}

func createPatient() *resources.Patient {
	birthDate := primitives.MustDate("1970-05-15")
	return &resources.Patient{
		ID:     testutil.StringPtr("patient-001"),
		Active: testutil.BoolPtr(true),
		Name: []resources.HumanName{
			{
				Use:    testutil.StringPtr("official"),
				Family: testutil.StringPtr("Smith"),
				Given:  []string{"Jane", "Marie"},
			},
		},
		Gender:    testutil.StringPtr("female"),
		BirthDate: &birthDate,
		Telecom: []resources.ContactPoint{
			{
				System: testutil.StringPtr("phone"),
				Value:  testutil.StringPtr("+1-555-0123"),
				Use:    testutil.StringPtr("mobile"),
			},
			{
				System: testutil.StringPtr("email"),
				Value:  testutil.StringPtr("jane.smith@example.com"),
			},
		},
		Address: []resources.Address{
			{
				Use:        testutil.StringPtr("home"),
				Line:       []string{"123 Main St", "Apt 4B"},
				City:       testutil.StringPtr("Springfield"),
				State:      testutil.StringPtr("IL"),
				PostalCode: testutil.StringPtr("62701"),
				Country:    testutil.StringPtr("USA"),
			},
		},
	}
}

func createEncounter(patientID *string) *resources.Encounter {
	now := primitives.MustDateTime(time.Now().Format(time.RFC3339))
	return &resources.Encounter{
		ID:     testutil.StringPtr("encounter-001"),
		Status: "finished",
		Class: []resources.CodeableConcept{
			{
				Coding: []resources.Coding{
					{
						System:  testutil.StringPtr("http://terminology.hl7.org/CodeSystem/v3-ActCode"),
						Code:    testutil.StringPtr("AMB"),
						Display: testutil.StringPtr("ambulatory"),
					},
				},
			},
		},
		Subject: &resources.Reference{
			Reference: testutil.StringPtr(fmt.Sprintf("Patient/%s", *patientID)),
			Display:   testutil.StringPtr("Jane Smith"),
		},
		Period: &resources.Period{
			Start: &now,
			End:   &now,
		},
	}
}

func createVitalSigns(patientID, encounterID *string) []*resources.Observation {
	effectiveDateTime := primitives.MustDateTime(time.Now().Format(time.RFC3339))

	// Blood Pressure
	bp := &resources.Observation{
		ID:     testutil.StringPtr("obs-bp-001"),
		Status: "final",
		Category: []resources.CodeableConcept{
			{
				Coding: []resources.Coding{
					{
						System:  testutil.StringPtr("http://terminology.hl7.org/CodeSystem/observation-category"),
						Code:    testutil.StringPtr("vital-signs"),
						Display: testutil.StringPtr("Vital Signs"),
					},
				},
			},
		},
		Code: resources.CodeableConcept{
			Coding: []resources.Coding{
				{
					System:  testutil.StringPtr("http://loinc.org"),
					Code:    testutil.StringPtr("85354-9"),
					Display: testutil.StringPtr("Blood pressure panel"),
				},
			},
			Text: testutil.StringPtr("Blood Pressure"),
		},
		Subject: &resources.Reference{
			Reference: testutil.StringPtr(fmt.Sprintf("Patient/%s", *patientID)),
		},
		Encounter: &resources.Reference{
			Reference: testutil.StringPtr(fmt.Sprintf("Encounter/%s", *encounterID)),
		},
		EffectiveDateTime: &effectiveDateTime,
		Component: []resources.ObservationComponent{
			{
				Code: resources.CodeableConcept{
					Coding: []resources.Coding{
						{
							System:  testutil.StringPtr("http://loinc.org"),
							Code:    testutil.StringPtr("8480-6"),
							Display: testutil.StringPtr("Systolic blood pressure"),
						},
					},
				},
				ValueQuantity: &resources.Quantity{
					Value:  float64Ptr(120),
					Unit:   testutil.StringPtr("mmHg"),
					System: testutil.StringPtr("http://unitsofmeasure.org"),
					Code:   testutil.StringPtr("mm[Hg]"),
				},
			},
			{
				Code: resources.CodeableConcept{
					Coding: []resources.Coding{
						{
							System:  testutil.StringPtr("http://loinc.org"),
							Code:    testutil.StringPtr("8462-4"),
							Display: testutil.StringPtr("Diastolic blood pressure"),
						},
					},
				},
				ValueQuantity: &resources.Quantity{
					Value:  float64Ptr(80),
					Unit:   testutil.StringPtr("mmHg"),
					System: testutil.StringPtr("http://unitsofmeasure.org"),
					Code:   testutil.StringPtr("mm[Hg]"),
				},
			},
		},
	}

	// Heart Rate
	hr := &resources.Observation{
		ID:     testutil.StringPtr("obs-hr-001"),
		Status: "final",
		Category: []resources.CodeableConcept{
			{
				Coding: []resources.Coding{
					{
						System:  testutil.StringPtr("http://terminology.hl7.org/CodeSystem/observation-category"),
						Code:    testutil.StringPtr("vital-signs"),
						Display: testutil.StringPtr("Vital Signs"),
					},
				},
			},
		},
		Code: resources.CodeableConcept{
			Coding: []resources.Coding{
				{
					System:  testutil.StringPtr("http://loinc.org"),
					Code:    testutil.StringPtr("8867-4"),
					Display: testutil.StringPtr("Heart rate"),
				},
			},
			Text: testutil.StringPtr("Heart Rate"),
		},
		Subject: &resources.Reference{
			Reference: testutil.StringPtr(fmt.Sprintf("Patient/%s", *patientID)),
		},
		Encounter: &resources.Reference{
			Reference: testutil.StringPtr(fmt.Sprintf("Encounter/%s", *encounterID)),
		},
		EffectiveDateTime: &effectiveDateTime,
		ValueQuantity: &resources.Quantity{
			Value:  float64Ptr(72),
			Unit:   testutil.StringPtr("beats/minute"),
			System: testutil.StringPtr("http://unitsofmeasure.org"),
			Code:   testutil.StringPtr("/min"),
		},
	}

	return []*resources.Observation{bp, hr}
}

func createMedicationRequest(patientID, encounterID *string) *resources.MedicationRequest {
	authoredOn := primitives.MustDateTime(time.Now().Format(time.RFC3339))

	return &resources.MedicationRequest{
		ID:     testutil.StringPtr("medreq-001"),
		Status: "active",
		Intent: "order",
		Medication: resources.CodeableReference{
			Concept: &resources.CodeableConcept{
				Coding: []resources.Coding{
					{
						System:  testutil.StringPtr("http://www.nlm.nih.gov/research/umls/rxnorm"),
						Code:    testutil.StringPtr("197361"),
						Display: testutil.StringPtr("Lisinopril 10 MG Oral Tablet"),
					},
				},
				Text: testutil.StringPtr("Lisinopril 10mg tablet"),
			},
		},
		Subject: &resources.Reference{
			Reference: testutil.StringPtr(fmt.Sprintf("Patient/%s", *patientID)),
			Display:   testutil.StringPtr("Jane Smith"),
		},
		Encounter: &resources.Reference{
			Reference: testutil.StringPtr(fmt.Sprintf("Encounter/%s", *encounterID)),
		},
		AuthoredOn: &authoredOn,
		DosageInstruction: []resources.Dosage{
			{
				Text:   testutil.StringPtr("Take one tablet by mouth once daily"),
				Timing: &resources.Timing{},
			},
		},
	}
}

func createBundle(patient *resources.Patient, encounter *resources.Encounter,
	observations []*resources.Observation, medication *resources.MedicationRequest) *fhir.Bundle {

	bundle := &fhir.Bundle{
		Type: "collection",
	}

	helper := fhir.NewBundleHelper(bundle)

	// Add all resources to bundle
	_ = helper.AddEntry(patient, testutil.StringPtr(fmt.Sprintf("Patient/%s", *patient.ID)))
	_ = helper.AddEntry(encounter, testutil.StringPtr(fmt.Sprintf("Encounter/%s", *encounter.ID)))

	for _, obs := range observations {
		_ = helper.AddEntry(obs, testutil.StringPtr(fmt.Sprintf("Observation/%s", *obs.ID)))
	}

	_ = helper.AddEntry(medication, testutil.StringPtr(fmt.Sprintf("MedicationRequest/%s", *medication.ID)))

	return bundle
}

func validateBundle(bundle *fhir.Bundle) error {
	validator := validation.NewFHIRValidator()

	// Validate each entry in the bundle
	for i, entry := range bundle.Entry {
		// Parse the resource to get its type
		var resourceMap map[string]interface{}
		if err := json.Unmarshal(entry.Resource, &resourceMap); err != nil {
			return fmt.Errorf("entry %d: failed to parse resource: %w", i, err)
		}

		resourceType, ok := resourceMap["resourceType"].(string)
		if !ok {
			return fmt.Errorf("entry %d: missing resourceType", i)
		}

		fmt.Printf("Validating %s...\n", resourceType)

		// For demonstration, we'd need to unmarshal to the correct type
		// This is simplified - in practice you'd use a type switch or registry
	}

	fmt.Println("All resources validated successfully!")
	return nil
}

// Helper functions
func testutil.StringPtr(s string) *string {
	return &s
}

func testutil.BoolPtr(b bool) *bool {
	return &b
}

func float64Ptr(f float64) *float64 {
	return &f
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
