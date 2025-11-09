package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/codeninja55/go-radx/fhir"
)

// SearchPatientsExample demonstrates searching for patients and processing results
func SearchPatientsExample() {
	// Simulate a FHIR search result bundle
	bundleJSON := `{
		"resourceType": "Bundle",
		"type": "searchset",
		"total": 2,
		"entry": [
			{
				"fullUrl": "http://example.org/Patient/1",
				"resource": {
					"resourceType": "Patient",
					"id": "1",
					"active": true,
					"name": [{"family": "Doe", "given": ["John"]}],
					"gender": "male"
				}
			},
			{
				"fullUrl": "http://example.org/Patient/2",
				"resource": {
					"resourceType": "Patient",
					"id": "2",
					"active": true,
					"name": [{"family": "Smith", "given": ["Jane"]}],
					"gender": "female"
				}
			}
		],
		"link": [
			{"relation": "self", "url": "http://example.org/Patient?name=doe"},
			{"relation": "next", "url": "http://example.org/Patient?name=doe&page=2"}
		]
	}`

	// Parse bundle
	var bundle fhir.Bundle
	if err := json.Unmarshal([]byte(bundleJSON), &bundle); err != nil {
		panic(err)
	}

	// Create helper
	helper := fhir.NewBundleHelper(&bundle)

	// Get summary
	fmt.Printf("Search returned %d total results\n", *bundle.Total)
	fmt.Printf("This page contains %d results\n", helper.Count())

	// Get all patients
	patients, _ := helper.GetPatients()
	fmt.Printf("\nPatients:\n")
	for i, patientJSON := range patients {
		var patient map[string]interface{}
		json.Unmarshal(patientJSON, &patient)
		fmt.Printf("  %d. ID: %s, Gender: %s\n",
			i+1,
			patient["id"],
			patient["gender"])
	}

	// Check for next page
	if nextLink := helper.GetNextLink(); nextLink != nil {
		fmt.Printf("\nNext page available at: %s\n", *nextLink)
	}
}

// PaginatedSearchExample demonstrates handling paginated search results
func PaginatedSearchExample(baseURL string) error {
	var allPatients []map[string]interface{}
	nextURL := baseURL + "/Patient?name=smith"

	for nextURL != "" {
		// Fetch page
		resp, err := http.Get(nextURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Parse bundle
		var bundle fhir.Bundle
		if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
			return err
		}

		helper := fhir.NewBundleHelper(&bundle)

		// Extract patients from this page
		patientJSONs, _ := helper.GetPatients()
		for _, pJSON := range patientJSONs {
			var patient map[string]interface{}
			json.Unmarshal(pJSON, &patient)
			allPatients = append(allPatients, patient)
		}

		fmt.Printf("Fetched page with %d patients (total so far: %d)\n",
			len(patientJSONs), len(allPatients))

		// Get next page URL
		if nextLink := helper.GetNextLink(); nextLink != nil {
			nextURL = *nextLink
		} else {
			nextURL = "" // No more pages
		}
	}

	fmt.Printf("\nFetched %d total patients across all pages\n", len(allPatients))
	return nil
}

// FilterBundleExample demonstrates filtering resources in a bundle
func FilterBundleExample() {
	bundleJSON := `{
		"resourceType": "Bundle",
		"type": "collection",
		"entry": [
			{
				"resource": {
					"resourceType": "Patient",
					"id": "1",
					"active": true,
					"gender": "male"
				}
			},
			{
				"resource": {
					"resourceType": "Patient",
					"id": "2",
					"active": false,
					"gender": "female"
				}
			},
			{
				"resource": {
					"resourceType": "Observation",
					"id": "obs-1",
					"status": "final"
				}
			}
		]
	}`

	var bundle fhir.Bundle
	json.Unmarshal([]byte(bundleJSON), &bundle)

	helper := fhir.NewBundleHelper(&bundle)

	// Get resource type summary
	types, _ := helper.GetResourceTypes()
	fmt.Println("Resource types in bundle:")
	for _, resourceType := range types {
		count, _ := helper.CountByType(resourceType)
		fmt.Printf("  - %s: %d\n", resourceType, count)
	}

	// Filter active patients
	fmt.Println("\nActive patients:")
	patients, _ := helper.GetPatients()
	for _, pJSON := range patients {
		var patient map[string]interface{}
		json.Unmarshal(pJSON, &patient)

		if active, ok := patient["active"].(bool); ok && active {
			fmt.Printf("  - Patient %s is active\n", patient["id"])
		}
	}
}

// ResolveReferencesExample demonstrates resolving references within a bundle
func ResolveReferencesExample() {
	bundleJSON := `{
		"resourceType": "Bundle",
		"type": "collection",
		"entry": [
			{
				"fullUrl": "http://example.org/Patient/123",
				"resource": {
					"resourceType": "Patient",
					"id": "123",
					"name": [{"family": "Doe"}]
				}
			},
			{
				"fullUrl": "http://example.org/Observation/obs-1",
				"resource": {
					"resourceType": "Observation",
					"id": "obs-1",
					"status": "final",
					"subject": {
						"reference": "Patient/123"
					}
				}
			}
		]
	}`

	var bundle fhir.Bundle
	json.Unmarshal([]byte(bundleJSON), &bundle)

	helper := fhir.NewBundleHelper(&bundle)

	// Get observations
	observations, _ := helper.GetObservations()

	for _, obsJSON := range observations {
		var obs map[string]interface{}
		json.Unmarshal(obsJSON, &obs)

		// Get subject reference
		if subject, ok := obs["subject"].(map[string]interface{}); ok {
			if ref, ok := subject["reference"].(string); ok {
				// Resolve reference
				patientJSON, _ := helper.ResolveReference(ref)
				if patientJSON != nil {
					var patient map[string]interface{}
					json.Unmarshal(patientJSON, &patient)

					fmt.Printf("Observation %s is for patient: %s\n",
						obs["id"],
						patient["id"])
				}
			}
		}
	}
}
