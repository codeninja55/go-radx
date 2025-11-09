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
	// Create a Patient resource programmatically
	patient := createPatient()

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(patient, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling patient: %v\n", err)
		os.Exit(1)
	}

	// Print the JSON
	fmt.Println(string(data))
}

func createPatient() resources.Patient {
	active := true
	birthDate := primitives.MustDate("1974-12-25")

	return resources.Patient{
		ID:     stringPtr("example"),
		Active: &active,
		Name: []resources.HumanName{
			{
				Use:    stringPtr("official"),
				Family: stringPtr("Chalmers"),
				Given:  []string{"Peter", "James"},
			},
			{
				Use:   stringPtr("usual"),
				Given: []string{"Jim"},
			},
		},
		Gender:    stringPtr("male"),
		BirthDate: &birthDate,
		Telecom: []resources.ContactPoint{
			{
				System: stringPtr("phone"),
				Value:  stringPtr("(03) 5555 6473"),
				Use:    stringPtr("work"),
				Rank:   intPtr(1),
			},
			{
				System: stringPtr("phone"),
				Value:  stringPtr("(03) 3410 5613"),
				Use:    stringPtr("mobile"),
				Rank:   intPtr(2),
			},
		},
		Address: []resources.Address{
			{
				Use:        stringPtr("home"),
				Type:       stringPtr("both"),
				Line:       []string{"534 Erewhon St"},
				City:       stringPtr("PleasantVille"),
				State:      stringPtr("Vic"),
				PostalCode: stringPtr("3999"),
				Period: &resources.Period{
					Start: datetimePtr("1974-12-25"),
				},
			},
		},
		MaritalStatus: &resources.CodeableConcept{
			Coding: []resources.Coding{
				{
					System:  stringPtr("http://terminology.hl7.org/CodeSystem/v3-MaritalStatus"),
					Code:    stringPtr("M"),
					Display: stringPtr("Married"),
				},
			},
		},
		Contact: []resources.PatientContact{
			{
				Relationship: []resources.CodeableConcept{
					{
						Coding: []resources.Coding{
							{
								System: stringPtr("http://terminology.hl7.org/CodeSystem/v2-0131"),
								Code:   stringPtr("N"),
							},
						},
					},
				},
				Name: &resources.HumanName{
					Family: stringPtr("du Marché"),
					Given:  []string{"Bénédicte"},
				},
				Telecom: []resources.ContactPoint{
					{
						System: stringPtr("phone"),
						Value:  stringPtr("+33 (237) 998327"),
					},
				},
				Address: &resources.Address{
					Use:        stringPtr("home"),
					Type:       stringPtr("both"),
					Line:       []string{"534 Erewhon St"},
					City:       stringPtr("PleasantVille"),
					State:      stringPtr("Vic"),
					PostalCode: stringPtr("3999"),
					Period: &resources.Period{
						Start: datetimePtr("1974-12-25"),
					},
				},
				Gender: stringPtr("female"),
				Period: &resources.Period{
					Start: datetimePtr("2012"),
				},
			},
		},
	}
}

// Helper functions for pointer types
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func datetimePtr(s string) *primitives.DateTime {
	dt := primitives.MustDateTime(s)
	return &dt
}
