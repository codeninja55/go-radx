//go:build ignore

package main

import (
	"fmt"
	"log"

	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
	"github.com/codeninja55/go-radx/fhir/validation"
)

// Example: Various validation scenarios
// Demonstrates FHIR validation including cardinality, required fields, and choice types
func main() {
	validator := validation.NewFHIRValidator()

	fmt.Println("=== FHIR Validation Examples ===\n")

	// Example 1: Valid patient (all required fields present)
	fmt.Println("1. Valid Patient:")
	validPatient := &resources.Patient{
		ID:     stringPtr("valid-patient"),
		Active: boolPtr(true),
	}
	if err := validator.Validate(validPatient); err != nil {
		fmt.Printf("   ❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Patient is valid")
	}

	// Example 2: Patient with invalid gender enum
	fmt.Println("\n2. Patient with Invalid Gender:")
	invalidGender := &resources.Patient{
		ID:     stringPtr("invalid-gender"),
		Gender: stringPtr("invalid-value"), // Should be: male, female, other, unknown
	}
	if err := validator.Validate(invalidGender); err != nil {
		fmt.Printf("   ❌ Expected validation error: %v\n", err)
	} else {
		fmt.Println("   ✓ Patient is valid")
	}

	// Example 3: Observation with required fields
	fmt.Println("\n3. Valid Observation:")
	effectiveDateTime := primitives.MustDateTime("2024-01-15T10:30:00Z")
	validObservation := &resources.Observation{
		ID:     stringPtr("obs-1"),
		Status: "final", // Required field
		Code: resources.CodeableConcept{ // Required field
			Coding: []resources.Coding{
				{
					System:  stringPtr("http://loinc.org"),
					Code:    stringPtr("8867-4"),
					Display: stringPtr("Heart rate"),
				},
			},
		},
		EffectiveDateTime: &effectiveDateTime,
		ValueQuantity: &resources.Quantity{
			Value:  float64Ptr(72),
			Unit:   stringPtr("beats/minute"),
			System: stringPtr("http://unitsofmeasure.org"),
			Code:   stringPtr("/min"),
		},
	}
	if err := validator.Validate(validObservation); err != nil {
		fmt.Printf("   ❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Observation is valid")
	}

	// Example 4: Observation missing required status
	fmt.Println("\n4. Observation Missing Required Status:")
	invalidObservation := &resources.Observation{
		ID: stringPtr("obs-2"),
		// Missing Status (required field)
		Code: resources.CodeableConcept{
			Text: stringPtr("Heart rate"),
		},
	}
	if err := validator.Validate(invalidObservation); err != nil {
		fmt.Printf("   ❌ Expected validation error: %v\n", err)
	} else {
		fmt.Println("   ✓ Observation is valid")
	}

	// Example 5: Choice type validation (deceased[x])
	fmt.Println("\n5. Valid Choice Type (deceasedBoolean):")
	patientDeceased := &resources.Patient{
		ID:              stringPtr("patient-deceased"),
		DeceasedBoolean: boolPtr(false),
		// Only one deceased[x] field should be set
	}
	if err := validator.Validate(patientDeceased); err != nil {
		fmt.Printf("   ❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Patient with deceasedBoolean is valid")
	}

	// Example 6: Invalid choice type (multiple fields from same choice group)
	fmt.Println("\n6. Invalid Choice Type (multiple deceased[x] fields):")
	deceasedDateTime := primitives.MustDateTime("2024-01-01T00:00:00Z")
	invalidChoice := &resources.Patient{
		ID:               stringPtr("patient-invalid-choice"),
		DeceasedBoolean:  boolPtr(false),    // First choice
		DeceasedDateTime: &deceasedDateTime, // Second choice - INVALID!
	}
	if err := validator.Validate(invalidChoice); err != nil {
		fmt.Printf("   ❌ Expected validation error: %v\n", err)
	} else {
		fmt.Println("   ✓ Patient is valid (unexpected)")
	}

	// Example 7: Reference validation
	fmt.Println("\n7. Valid Reference:")
	validReference := &resources.Reference{
		Reference: stringPtr("Patient/123"),
		Display:   stringPtr("John Doe"),
	}
	if err := validation.ValidateReference(validReference); err != nil {
		fmt.Printf("   ❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Reference is valid")
	}

	// Example 8: Invalid reference format
	fmt.Println("\n8. Invalid Reference Format:")
	invalidReference := &resources.Reference{
		Reference: stringPtr("invalid-format"), // Should be ResourceType/id
	}
	if err := validation.ValidateReference(invalidReference); err != nil {
		fmt.Printf("   ❌ Expected validation error: %v\n", err)
	} else {
		fmt.Println("   ✓ Reference is valid (unexpected)")
	}

	// Example 9: Cardinality validation
	fmt.Println("\n9. Cardinality Validation:")
	// Testing cardinality for identifiers (0..*)
	patientWithIdentifiers := &resources.Patient{
		ID: stringPtr("patient-identifiers"),
		Identifier: []resources.Identifier{
			{
				System: stringPtr("http://hospital.example.org"),
				Value:  stringPtr("MRN12345"),
			},
			{
				System: stringPtr("http://hl7.org/fhir/sid/us-ssn"),
				Value:  stringPtr("123-45-6789"),
			},
		},
	}
	if err := validator.Validate(patientWithIdentifiers); err != nil {
		fmt.Printf("   ❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Patient with multiple identifiers is valid")
	}

	// Example 10: Custom validation function
	fmt.Println("\n10. Custom Business Rules:")
	patient := &resources.Patient{
		ID:     stringPtr("patient-business-rules"),
		Active: boolPtr(true),
	}

	// Add custom business logic validation
	if err := validateBusinessRules(patient); err != nil {
		fmt.Printf("   ❌ Business rule validation failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Patient passes business rule validation")
	}

	fmt.Println("\n=== Validation Examples Complete ===")
}

// Custom business rule validation example
func validateBusinessRules(patient *resources.Patient) error {
	// Example rule: Active patients must have contact information
	if patient.Active != nil && *patient.Active {
		if len(patient.Telecom) == 0 && len(patient.Address) == 0 {
			return fmt.Errorf("active patients must have contact information")
		}
	}

	// Example rule: Patients over 18 must have their own contact info
	if patient.BirthDate != nil {
		// In real implementation, calculate age from birthDate
		// and validate accordingly
	}

	return nil
}

// Example: Batch validation
func validateMultipleResources() {
	validator := validation.NewFHIRValidator()

	patients := []*resources.Patient{
		{ID: stringPtr("p1"), Active: boolPtr(true)},
		{ID: stringPtr("p2"), Gender: stringPtr("male")},
		{ID: stringPtr("p3"), Gender: stringPtr("invalid")}, // This will fail
	}

	fmt.Println("\n=== Batch Validation ===")
	validCount := 0
	invalidCount := 0

	for i, patient := range patients {
		if err := validator.Validate(patient); err != nil {
			fmt.Printf("Patient %d: ❌ %v\n", i+1, err)
			invalidCount++
		} else {
			fmt.Printf("Patient %d: ✓ Valid\n", i+1)
			validCount++
		}
	}

	fmt.Printf("\nResults: %d valid, %d invalid\n", validCount, invalidCount)
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func float64Ptr(f float64) *float64 {
	return &f
}

func init() {
	// You can also run batch validation
	// Uncomment to see batch validation in action:
	// validateMultipleResources()
}
