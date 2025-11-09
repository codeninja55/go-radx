//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/r4/resources"
)

func main() {
	// Create a blood pressure observation
	observation := createBloodPressureObservation()

	// Marshal to JSON
	data, err := json.MarshalIndent(observation, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling observation: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}

func createBloodPressureObservation() resources.Observation {
	effectiveDateTime := primitives.MustDateTime("2024-01-15T10:30:00Z")

	return resources.Observation{
		ID:     stringPtr("blood-pressure"),
		Status: "final",
		Category: []resources.CodeableConcept{
			{
				Coding: []resources.Coding{
					{
						System:  stringPtr("http://terminology.hl7.org/CodeSystem/observation-category"),
						Code:    stringPtr("vital-signs"),
						Display: stringPtr("Vital Signs"),
					},
				},
			},
		},
		Code: resources.CodeableConcept{
			Coding: []resources.Coding{
				{
					System:  stringPtr("http://loinc.org"),
					Code:    stringPtr("85354-9"),
					Display: stringPtr("Blood pressure panel"),
				},
			},
			Text: stringPtr("Blood pressure systolic & diastolic"),
		},
		Subject: resources.Reference{
			Reference: stringPtr("Patient/example"),
		},
		EffectiveDateTime: &effectiveDateTime,
		Component: []resources.ObservationComponent{
			{
				Code: resources.CodeableConcept{
					Coding: []resources.Coding{
						{
							System:  stringPtr("http://loinc.org"),
							Code:    stringPtr("8480-6"),
							Display: stringPtr("Systolic blood pressure"),
						},
					},
				},
				ValueQuantity: &resources.Quantity{
					Value:  float64Ptr(120),
					Unit:   stringPtr("mmHg"),
					System: stringPtr("http://unitsofmeasure.org"),
					Code:   stringPtr("mm[Hg]"),
				},
			},
			{
				Code: resources.CodeableConcept{
					Coding: []resources.Coding{
						{
							System:  stringPtr("http://loinc.org"),
							Code:    stringPtr("8462-4"),
							Display: stringPtr("Diastolic blood pressure"),
						},
					},
				},
				ValueQuantity: &resources.Quantity{
					Value:  float64Ptr(80),
					Unit:   stringPtr("mmHg"),
					System: stringPtr("http://unitsofmeasure.org"),
					Code:   stringPtr("mm[Hg]"),
				},
			},
		},
		Interpretation: []resources.CodeableConcept{
			{
				Coding: []resources.Coding{
					{
						System:  stringPtr("http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation"),
						Code:    stringPtr("N"),
						Display: stringPtr("Normal"),
					},
				},
			},
		},
	}
}

func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
