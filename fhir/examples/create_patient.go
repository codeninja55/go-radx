//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codeninja55/go-radx/fhir/internal/testutil"
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
		ID:     testutil.StringPtr("example"),
		Active: &active,
		Name: []resources.HumanName{
			{
				Use:    testutil.StringPtr("official"),
				Family: testutil.StringPtr("Chalmers"),
				Given:  []string{"Peter", "James"},
			},
			{
				Use:   testutil.StringPtr("usual"),
				Given: []string{"Jim"},
			},
		},
		Gender:    testutil.StringPtr("male"),
		BirthDate: &birthDate,
		Telecom: []resources.ContactPoint{
			{
				System: testutil.StringPtr("phone"),
				Value:  testutil.StringPtr("(03) 5555 6473"),
				Use:    testutil.StringPtr("work"),
				Rank:   testutil.IntPtr(1),
			},
			{
				System: testutil.StringPtr("phone"),
				Value:  testutil.StringPtr("(03) 3410 5613"),
				Use:    testutil.StringPtr("mobile"),
				Rank:   testutil.IntPtr(2),
			},
		},
		Address: []resources.Address{
			{
				Use:        testutil.StringPtr("home"),
				Type:       testutil.StringPtr("both"),
				Line:       []string{"534 Erewhon St"},
				City:       testutil.StringPtr("PleasantVille"),
				State:      testutil.StringPtr("Vic"),
				PostalCode: testutil.StringPtr("3999"),
				Period: &resources.Period{
					Start: datetimePtr("1974-12-25"),
				},
			},
		},
		MaritalStatus: &resources.CodeableConcept{
			Coding: []resources.Coding{
				{
					System:  testutil.StringPtr("http://terminology.hl7.org/CodeSystem/v3-MaritalStatus"),
					Code:    testutil.StringPtr("M"),
					Display: testutil.StringPtr("Married"),
				},
			},
		},
		Contact: []resources.PatientContact{
			{
				Relationship: []resources.CodeableConcept{
					{
						Coding: []resources.Coding{
							{
								System: testutil.StringPtr("http://terminology.hl7.org/CodeSystem/v2-0131"),
								Code:   testutil.StringPtr("N"),
							},
						},
					},
				},
				Name: &resources.HumanName{
					Family: testutil.StringPtr("du Marché"),
					Given:  []string{"Bénédicte"},
				},
				Telecom: []resources.ContactPoint{
					{
						System: testutil.StringPtr("phone"),
						Value:  testutil.StringPtr("+33 (237) 998327"),
					},
				},
				Address: &resources.Address{
					Use:        testutil.StringPtr("home"),
					Type:       testutil.StringPtr("both"),
					Line:       []string{"534 Erewhon St"},
					City:       testutil.StringPtr("PleasantVille"),
					State:      testutil.StringPtr("Vic"),
					PostalCode: testutil.StringPtr("3999"),
					Period: &resources.Period{
						Start: datetimePtr("1974-12-25"),
					},
				},
				Gender: testutil.StringPtr("female"),
				Period: &resources.Period{
					Start: datetimePtr("2012"),
				},
			},
		},
	}
}

func datetimePtr(s string) *primitives.DateTime {
	dt := primitives.MustDateTime(s)
	return &dt
}
