//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r5/resources"
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
)

// Example: Processing FHIR search results from a server
// Demonstrates bundle navigation, filtering, and data extraction
func main() {
	// Simulate loading a search bundle from a FHIR server response
	// In real usage, this would come from an HTTP response
	bundle := loadSearchBundle()

	// Create bundle helper for easy navigation
	helper := fhir.NewBundleHelper(bundle)

	// Example 1: Count resources by type
	fmt.Println("=== Resource Count ===")
	fmt.Printf("Total resources: %d\n", helper.Count())

	resourceTypes := helper.GetResourceTypes()
	for _, rt := range resourceTypes {
		count := helper.CountByType(rt)
		fmt.Printf("  %s: %d\n", rt, count)
	}

	// Example 2: Extract all patients
	fmt.Println("\n=== Patients ===")
	patients, err := helper.GetPatients()
	if err != nil {
		log.Fatal(err)
	}

	for _, patientJSON := range patients {
		var patient resources.Patient
		if err := json.Unmarshal(patientJSON, &patient); err != nil {
			log.Printf("Error parsing patient: %v", err)
			continue
		}

		if patient.ID != nil && len(patient.Name) > 0 && patient.Name[0].Family != nil {
			fmt.Printf("  Patient %s: %s", *patient.ID, *patient.Name[0].Family)
			if len(patient.Name[0].Given) > 0 {
				fmt.Printf(", %s", patient.Name[0].Given[0])
			}
			fmt.Println()
		}
	}

	// Example 3: Extract and analyze observations
	fmt.Println("\n=== Observations ===")
	observations, err := helper.GetObservations()
	if err != nil {
		log.Fatal(err)
	}

	bloodPressures := 0
	heartRates := 0
	temperatures := 0

	for _, obsJSON := range observations {
		var obs resources.Observation
		if err := json.Unmarshal(obsJSON, &obs); err != nil {
			continue
		}

		// Classify by LOINC code
		if len(obs.Code.Coding) > 0 {
			code := obs.Code.Coding[0].Code
			if code == nil {
				continue
			}

			switch *code {
			case "85354-9": // Blood pressure panel
				bloodPressures++
			case "8867-4": // Heart rate
				heartRates++
			case "8310-5": // Body temperature
				temperatures++
			}
		}
	}

	fmt.Printf("  Blood Pressure readings: %d\n", bloodPressures)
	fmt.Printf("  Heart Rate readings: %d\n", heartRates)
	fmt.Printf("  Temperature readings: %d\n", temperatures)

	// Example 4: Follow references
	fmt.Println("\n=== Reference Resolution ===")
	if len(observations) > 0 {
		var firstObs resources.Observation
		if err := json.Unmarshal(observations[0], &firstObs); err == nil {
			if firstObs.Subject != nil && firstObs.Subject.Reference != nil {
				fmt.Printf("First observation references: %s\n", *firstObs.Subject.Reference)

				// Resolve the reference
				subject, err := helper.ResolveReference(*firstObs.Subject.Reference)
				if err == nil && subject != nil {
					var patient resources.Patient
					if err := json.Unmarshal(subject, &patient); err == nil {
						if len(patient.Name) > 0 && patient.Name[0].Family != nil {
							fmt.Printf("  Resolved to patient: %s\n", *patient.Name[0].Family)
						}
					}
				}
			}
		}
	}

	// Example 5: Pagination handling
	fmt.Println("\n=== Pagination ===")
	if nextLink := helper.GetNextLink(); nextLink != "" {
		fmt.Printf("Next page URL: %s\n", nextLink)
		fmt.Println("  (In production, fetch this URL for more results)")
	} else {
		fmt.Println("No more pages (this is the last page)")
	}

	// Example 6: Export to summary JSON for bandwidth reduction
	fmt.Println("\n=== Summary Mode ===")
	summaryJSON, err := fhir.MarshalSummaryJSON(bundle)
	if err == nil {
		fullJSON, _ := json.Marshal(bundle)
		reduction := 100.0 * (1.0 - float64(len(summaryJSON))/float64(len(fullJSON)))
		fmt.Printf("Full bundle: %d bytes\n", len(fullJSON))
		fmt.Printf("Summary bundle: %d bytes\n", len(summaryJSON))
		fmt.Printf("Reduction: %.1f%%\n", reduction)
	}
}

func loadSearchBundle() *fhir.Bundle {
	// In real usage, this would be from an HTTP response
	// For this example, we create a mock bundle
	bundle := &fhir.Bundle{
		Type:  "searchset",
		Total: testutil.IntPtr(5),
		Link: []fhir.BundleLink{
			{
				Relation: "self",
				URL:      "https://server.example.org/fhir/Observation?patient=123&_count=10",
			},
			{
				Relation: "next",
				URL:      "https://server.example.org/fhir/Observation?patient=123&_count=10&page=2",
			},
		},
	}

	// Add sample patient
	patient := map[string]interface{}{
		"resourceType": "Patient",
		"id":           "patient-123",
		"name": []map[string]interface{}{
			{
				"family": "Doe",
				"given":  []string{"John"},
			},
		},
	}
	patientJSON, _ := json.Marshal(patient)

	// Add sample observations
	obs1 := map[string]interface{}{
		"resourceType": "Observation",
		"id":           "obs-1",
		"status":       "final",
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://loinc.org",
					"code":    "85354-9",
					"display": "Blood pressure panel",
				},
			},
		},
		"subject": map[string]interface{}{
			"reference": "Patient/patient-123",
		},
	}
	obs1JSON, _ := json.Marshal(obs1)

	obs2 := map[string]interface{}{
		"resourceType": "Observation",
		"id":           "obs-2",
		"status":       "final",
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://loinc.org",
					"code":    "8867-4",
					"display": "Heart rate",
				},
			},
		},
		"subject": map[string]interface{}{
			"reference": "Patient/patient-123",
		},
	}
	obs2JSON, _ := json.Marshal(obs2)

	bundle.Entry = []fhir.BundleEntry{
		{
			FullURL:  testutil.StringPtr("Patient/patient-123"),
			Resource: patientJSON,
		},
		{
			FullURL:  testutil.StringPtr("Observation/obs-1"),
			Resource: obs1JSON,
		},
		{
			FullURL:  testutil.StringPtr("Observation/obs-2"),
			Resource: obs2JSON,
		},
	}

	return bundle
}

// Helper to read bundle from file (alternative to loadSearchBundle)
func readBundleFromFile(filename string) (*fhir.Bundle, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var bundle fhir.Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, err
	}

	return &bundle, nil
}

func testutil.StringPtr(s string) *string {
	return &s
}

func testutil.IntPtr(i int) *int {
	return &i
}
