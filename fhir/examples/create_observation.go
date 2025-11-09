//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/r4/resources"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
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
		ID:     testutil.StringPtr("blood-pressure"),
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
			Text: testutil.StringPtr("Blood pressure systolic & diastolic"),
		},
		Subject: resources.Reference{
			Reference: testutil.StringPtr("Patient/example"),
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
		Interpretation: []resources.CodeableConcept{
			{
				Coding: []resources.Coding{
					{
						System:  testutil.StringPtr("http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation"),
						Code:    testutil.StringPtr("N"),
						Display: testutil.StringPtr("Normal"),
					},
				},
			},
		},
	}
}
