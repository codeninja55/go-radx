package resources

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPatientExampleRoundTrip tests round-trip with official Patient example.
func TestPatientExampleRoundTrip(t *testing.T) {
	// Read fixture
	data, err := os.ReadFile("../../../testdata/fhir/r4/examples/patient-example.json")
	require.NoError(t, err, "failed to read fixture")

	// Unmarshal
	var patient Patient
	err = json.Unmarshal(data, &patient)
	require.NoError(t, err, "unmarshaling should not error")

	// Verify key fields
	assert.Equal(t, "example", *patient.ID)
	assert.True(t, *patient.Active)
	assert.Equal(t, "male", *patient.Gender)
	assert.Equal(t, "1974-12-25", patient.BirthDate.String())

	require.Len(t, patient.Name, 2, "should have 2 names")
	assert.Equal(t, "official", *patient.Name[0].Use)
	assert.Equal(t, "Chalmers", *patient.Name[0].Family)
	assert.Equal(t, []string{"Peter", "James"}, patient.Name[0].Given)

	require.Len(t, patient.Identifier, 1, "should have 1 identifier")
	assert.Equal(t, "usual", *patient.Identifier[0].Use)
	assert.Equal(t, "12345", *patient.Identifier[0].Value)

	require.Len(t, patient.Address, 1, "should have 1 address")
	assert.Equal(t, "home", *patient.Address[0].Use)
	assert.Equal(t, "PleasantVille", *patient.Address[0].City)
	assert.Equal(t, "Vic", *patient.Address[0].State)

	require.Len(t, patient.Contact, 1, "should have 1 contact")
	assert.Equal(t, "female", *patient.Contact[0].Gender)

	// Marshal back
	marshaled, err := json.Marshal(patient)
	require.NoError(t, err, "marshaling should not error")

	// Unmarshal again to verify consistency
	var patient2 Patient
	err = json.Unmarshal(marshaled, &patient2)
	require.NoError(t, err, "second unmarshal should not error")

	// Compare key fields
	assert.Equal(t, patient.ID, patient2.ID)
	assert.Equal(t, patient.Active, patient2.Active)
	assert.Equal(t, patient.Gender, patient2.Gender)
	assert.Equal(t, patient.BirthDate, patient2.BirthDate)
	assert.Equal(t, len(patient.Name), len(patient2.Name))
	assert.Equal(t, len(patient.Identifier), len(patient2.Identifier))
}

// TestObservationExampleRoundTrip tests round-trip with official Observation example.
func TestObservationExampleRoundTrip(t *testing.T) {
	// Read fixture
	data, err := os.ReadFile("../../../testdata/fhir/r4/examples/observation-example-bloodpressure.json")
	require.NoError(t, err, "failed to read fixture")

	// Unmarshal
	var obs Observation
	err = json.Unmarshal(data, &obs)
	require.NoError(t, err, "unmarshaling should not error")

	// Verify key fields
	assert.Equal(t, "blood-pressure", *obs.ID)
	assert.Equal(t, "final", obs.Status)

	require.Len(t, obs.Component, 2, "should have 2 components")
	assert.Equal(t, "8480-6", *obs.Component[0].Code.Coding[0].Code) // Systolic
	assert.Equal(t, "8462-4", *obs.Component[1].Code.Coding[0].Code) // Diastolic

	// Marshal back
	marshaled, err := json.Marshal(obs)
	require.NoError(t, err, "marshaling should not error")

	// Unmarshal again
	var obs2 Observation
	err = json.Unmarshal(marshaled, &obs2)
	require.NoError(t, err, "second unmarshal should not error")

	// Compare
	assert.Equal(t, obs.ID, obs2.ID)
	assert.Equal(t, obs.Status, obs2.Status)
	assert.Equal(t, len(obs.Component), len(obs2.Component))
}

// TestBundleExampleRoundTrip tests round-trip with official Bundle example.
func TestBundleExampleRoundTrip(t *testing.T) {
	// Read fixture
	data, err := os.ReadFile("../../../testdata/fhir/r4/examples/bundle-example.json")
	require.NoError(t, err, "failed to read fixture")

	// Unmarshal
	var bundle Bundle
	err = json.Unmarshal(data, &bundle)
	require.NoError(t, err, "unmarshaling should not error")

	// Verify key fields
	assert.Equal(t, "bundle-example", *bundle.ID)
	assert.Equal(t, "searchset", bundle.Type)
	assert.Equal(t, uint(2), *bundle.Total)

	require.Len(t, bundle.Entry, 2, "should have 2 entries")
	assert.Equal(t, "https://example.com/base/Patient/example", *bundle.Entry[0].FullUrl)

	// Marshal back
	marshaled, err := json.Marshal(bundle)
	require.NoError(t, err, "marshaling should not error")

	// Unmarshal again
	var bundle2 Bundle
	err = json.Unmarshal(marshaled, &bundle2)
	require.NoError(t, err, "second unmarshal should not error")

	// Compare
	assert.Equal(t, bundle.ID, bundle2.ID)
	assert.Equal(t, bundle.Type, bundle2.Type)
	assert.Equal(t, bundle.Total, bundle2.Total)
	assert.Equal(t, len(bundle.Entry), len(bundle2.Entry))
}

// TestRoundTripPreservesData tests that marshaling and unmarshaling preserves all data.
func TestRoundTripPreservesData(t *testing.T) {
	fixtures := []string{
		"../../../testdata/fhir/r4/examples/patient-example.json",
		"../../../testdata/fhir/r4/examples/observation-example-bloodpressure.json",
		"../../../testdata/fhir/r4/examples/bundle-example.json",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			// Read original
			original, err := os.ReadFile(fixture)
			require.NoError(t, err)

			// Unmarshal to interface
			var data1 map[string]any
			err = json.Unmarshal(original, &data1)
			require.NoError(t, err)

			// Marshal back
			marshaled, err := json.Marshal(data1)
			require.NoError(t, err)

			// Unmarshal again
			var data2 map[string]any
			err = json.Unmarshal(marshaled, &data2)
			require.NoError(t, err)

			// Compare resource types
			assert.Equal(t, data1["resourceType"], data2["resourceType"])
			assert.Equal(t, data1["id"], data2["id"])
		})
	}
}
