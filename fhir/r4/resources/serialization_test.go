package resources

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPatientMarshaling tests basic JSON marshaling of Patient resource.
func TestPatientMarshaling(t *testing.T) {
	// Create a simple Patient resource
	active := true
	patient := Patient{
		ID:     stringPtr("example"),
		Active: &active,
		Name: []HumanName{
			{
				Use:    stringPtr("official"),
				Family: stringPtr("Chalmers"),
				Given:  []string{"Peter", "James"},
			},
		},
		Gender:    stringPtr("male"),
		BirthDate: datePtr("1974-12-25"),
		Telecom: []ContactPoint{
			{
				System: stringPtr("phone"),
				Value:  stringPtr("(03) 5555 6473"),
				Use:    stringPtr("work"),
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(patient)
	require.NoError(t, err, "marshaling should not error")

	// Verify JSON is not empty
	assert.NotEmpty(t, data, "marshaled data should not be empty")

	t.Logf("Marshaled Patient:\n%s", string(data))
}

// TestPatientUnmarshaling tests basic JSON unmarshaling of Patient resource.
func TestPatientUnmarshaling(t *testing.T) {
	// Sample FHIR Patient JSON (simplified)
	patientJSON := `{
		"resourceType": "Patient",
		"id": "example",
		"active": true,
		"name": [{
			"use": "official",
			"family": "Chalmers",
			"given": ["Peter", "James"]
		}],
		"gender": "male",
		"birthDate": "1974-12-25",
		"telecom": [{
			"system": "phone",
			"value": "(03) 5555 6473",
			"use": "work"
		}]
	}`

	// Unmarshal from JSON
	var patient Patient
	err := json.Unmarshal([]byte(patientJSON), &patient)
	require.NoError(t, err, "unmarshaling should not error")

	// Verify fields
	assert.Equal(t, "example", *patient.ID)
	assert.True(t, *patient.Active)
	assert.Equal(t, "male", *patient.Gender)
	assert.Equal(t, "1974-12-25", patient.BirthDate.String())

	require.Len(t, patient.Name, 1)
	assert.Equal(t, "official", *patient.Name[0].Use)
	assert.Equal(t, "Chalmers", *patient.Name[0].Family)
	assert.Equal(t, []string{"Peter", "James"}, patient.Name[0].Given)

	require.Len(t, patient.Telecom, 1)
	assert.Equal(t, "phone", *patient.Telecom[0].System)
}

// TestPatientRoundTrip tests marshaling and unmarshaling produces identical results.
func TestPatientRoundTrip(t *testing.T) {
	// Original JSON
	originalJSON := `{
		"resourceType": "Patient",
		"id": "example",
		"active": true,
		"name": [{
			"use": "official",
			"family": "Chalmers",
			"given": ["Peter", "James"]
		}],
		"gender": "male",
		"birthDate": "1974-12-25"
	}`

	// Unmarshal
	var patient Patient
	err := json.Unmarshal([]byte(originalJSON), &patient)
	require.NoError(t, err)

	// Marshal back
	data, err := json.Marshal(patient)
	require.NoError(t, err)

	// Unmarshal again to compare
	var patient2 Patient
	err = json.Unmarshal(data, &patient2)
	require.NoError(t, err)

	// Compare key fields
	assert.Equal(t, patient.ID, patient2.ID)
	assert.Equal(t, patient.Active, patient2.Active)
	assert.Equal(t, patient.Gender, patient2.Gender)
	assert.Equal(t, patient.BirthDate, patient2.BirthDate)
}

// TestObservationMarshaling tests Observation resource with nested BackboneElements.
func TestObservationMarshaling(t *testing.T) {
	// Create an Observation with components
	obs := Observation{
		ID:     stringPtr("blood-pressure"),
		Status: "final", // Required field, not a pointer
		Code: CodeableConcept{ // Required field, not a pointer
			Coding: []Coding{
				{
					System:  stringPtr("http://loinc.org"),
					Code:    stringPtr("85354-9"),
					Display: stringPtr("Blood pressure panel"),
				},
			},
		},
	}

	// Marshal
	data, err := json.Marshal(obs)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	t.Logf("Marshaled Observation:\n%s", string(data))

	// Unmarshal back
	var obs2 Observation
	err = json.Unmarshal(data, &obs2)
	require.NoError(t, err)
	assert.Equal(t, "final", obs2.Status)
}

// TestBundleMarshaling tests Bundle resource with entries.
func TestBundleMarshaling(t *testing.T) {
	bundle := Bundle{
		ID:    stringPtr("bundle-example"),
		Type:  "searchset", // Required field, not a pointer
		Total: uintPtr(1),
	}

	// Marshal
	data, err := json.Marshal(bundle)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	t.Logf("Marshaled Bundle:\n%s", string(data))
}

// Helper functions
func uintPtr(i uint) *uint {
	return &i
}

func datePtr(s string) *primitives.Date {
	d := primitives.MustDate(s)
	return &d
}
