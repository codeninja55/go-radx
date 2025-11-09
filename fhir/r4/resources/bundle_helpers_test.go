package resources

import (
	"encoding/json"
	"testing"

	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSearchSetBundle(t *testing.T) {
	bundle := NewSearchSetBundle()
	assert.Equal(t, "searchset", bundle.Type)
}

func TestNewTransactionBundle(t *testing.T) {
	bundle := NewTransactionBundle()
	assert.Equal(t, "transaction", bundle.Type)
}

func TestBundle_AddEntry(t *testing.T) {
	bundle := NewSearchSetBundle()

	patient := &Patient{
		ID:     stringPtr("patient-1"),
		Gender: stringPtr("male"),
	}

	err := bundle.AddEntry(patient, "Patient/patient-1")
	require.NoError(t, err)

	assert.Len(t, bundle.Entry, 1)
	assert.Equal(t, uint(1), *bundle.Total)
	assert.Equal(t, "Patient/patient-1", *bundle.Entry[0].FullUrl)
}

func TestBundle_AddMultipleEntries(t *testing.T) {
	bundle := NewSearchSetBundle()

	// Add patient
	patient := &Patient{
		ID: stringPtr("patient-1"),
	}
	err := bundle.AddEntry(patient, "Patient/patient-1")
	require.NoError(t, err)

	// Add observation
	observation := &Observation{
		ID:     stringPtr("obs-1"),
		Status: "final",
	}
	err = bundle.AddEntry(observation, "Observation/obs-1")
	require.NoError(t, err)

	assert.Len(t, bundle.Entry, 2)
	assert.Equal(t, uint(2), *bundle.Total)
}

func TestBundle_GetEntry(t *testing.T) {
	bundle := NewSearchSetBundle()

	patient := &Patient{
		ID:     stringPtr("patient-1"),
		Gender: stringPtr("female"),
	}
	err := bundle.AddEntry(patient, "Patient/patient-1")
	require.NoError(t, err)

	// Get the entry back
	resource, err := bundle.GetEntry(0)
	require.NoError(t, err)

	retrievedPatient, ok := resource.(*Patient)
	require.True(t, ok)
	assert.Equal(t, "patient-1", *retrievedPatient.ID)
	assert.Equal(t, "female", *retrievedPatient.Gender)
}

func TestBundle_GetEntry_OutOfRange(t *testing.T) {
	bundle := NewSearchSetBundle()

	_, err := bundle.GetEntry(0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestBundle_GetAllEntries(t *testing.T) {
	bundle := NewSearchSetBundle()

	birthDate := primitives.MustDate("1980-01-01")
	patient := &Patient{
		ID:        stringPtr("patient-1"),
		BirthDate: &birthDate,
	}
	observation := &Observation{
		ID:     stringPtr("obs-1"),
		Status: "final",
	}

	err := bundle.AddEntry(patient, "Patient/patient-1")
	require.NoError(t, err)
	err = bundle.AddEntry(observation, "Observation/obs-1")
	require.NoError(t, err)

	// Get all entries
	resources, err := bundle.GetAllEntries()
	require.NoError(t, err)
	assert.Len(t, resources, 2)

	// Verify types
	_, ok1 := resources[0].(*Patient)
	_, ok2 := resources[1].(*Observation)
	assert.True(t, ok1)
	assert.True(t, ok2)
}

func TestBundle_FindResourceByID(t *testing.T) {
	bundle := NewSearchSetBundle()

	patient1 := &Patient{ID: stringPtr("patient-1")}
	patient2 := &Patient{ID: stringPtr("patient-2")}
	observation := &Observation{ID: stringPtr("obs-1"), Status: "final"}

	bundle.AddEntry(patient1, "Patient/patient-1")
	bundle.AddEntry(patient2, "Patient/patient-2")
	bundle.AddEntry(observation, "Observation/obs-1")

	// Find patient-2
	resource, index, err := bundle.FindResourceByID("patient-2")
	require.NoError(t, err)
	assert.Equal(t, 1, index)

	patient, ok := resource.(*Patient)
	require.True(t, ok)
	assert.Equal(t, "patient-2", *patient.ID)
}

func TestBundle_FindResourceByID_NotFound(t *testing.T) {
	bundle := NewSearchSetBundle()

	patient := &Patient{ID: stringPtr("patient-1")}
	bundle.AddEntry(patient, "Patient/patient-1")

	_, _, err := bundle.FindResourceByID("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBundle_FilterByResourceType(t *testing.T) {
	bundle := NewSearchSetBundle()

	patient1 := &Patient{ID: stringPtr("patient-1")}
	patient2 := &Patient{ID: stringPtr("patient-2")}
	observation := &Observation{ID: stringPtr("obs-1"), Status: "final"}
	practitioner := &Practitioner{ID: stringPtr("prac-1")}

	bundle.AddEntry(patient1, "Patient/patient-1")
	bundle.AddEntry(observation, "Observation/obs-1")
	bundle.AddEntry(patient2, "Patient/patient-2")
	bundle.AddEntry(practitioner, "Practitioner/prac-1")

	// Filter for patients only
	patients, err := bundle.FilterByResourceType("Patient")
	require.NoError(t, err)
	assert.Len(t, patients, 2)

	// Verify they're all patients
	for _, res := range patients {
		_, ok := res.(*Patient)
		assert.True(t, ok)
	}

	// Filter for observations
	observations, err := bundle.FilterByResourceType("Observation")
	require.NoError(t, err)
	assert.Len(t, observations, 1)
}

func TestBundle_RoundTrip(t *testing.T) {
	// Create a bundle with resources
	bundle := NewSearchSetBundle()
	bundle.ID = stringPtr("bundle-example")

	patient := &Patient{
		ID:     stringPtr("patient-1"),
		Gender: stringPtr("male"),
		Name: []HumanName{
			{Family: stringPtr("Smith")},
		},
	}

	err := bundle.AddEntry(patient, "Patient/patient-1")
	require.NoError(t, err)

	// Marshal to JSON
	jsonData, err := json.Marshal(bundle)
	require.NoError(t, err)

	// Unmarshal back
	var bundle2 Bundle
	err = json.Unmarshal(jsonData, &bundle2)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "bundle-example", *bundle2.ID)
	assert.Equal(t, "searchset", bundle2.Type)
	assert.Len(t, bundle2.Entry, 1)

	// Get the patient from the unmarshaled bundle
	resource, err := bundle2.GetEntry(0)
	require.NoError(t, err)

	patient2, ok := resource.(*Patient)
	require.True(t, ok)
	assert.Equal(t, "patient-1", *patient2.ID)
	assert.Equal(t, "male", *patient2.Gender)
}
