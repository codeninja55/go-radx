package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalResource_Patient(t *testing.T) {
	jsonData := `{
		"resourceType": "Patient",
		"id": "example",
		"name": [{
			"family": "Chalmers",
			"given": ["Peter", "James"]
		}],
		"gender": "male"
	}`

	resource, err := UnmarshalResource([]byte(jsonData))
	require.NoError(t, err)

	patient, ok := resource.(*Patient)
	require.True(t, ok, "should be a *Patient")
	assert.Equal(t, "example", *patient.ID)
	assert.Equal(t, "male", *patient.Gender)
	assert.Len(t, patient.Name, 1)
	assert.Equal(t, "Chalmers", *patient.Name[0].Family)
}

func TestUnmarshalResource_Observation(t *testing.T) {
	jsonData := `{
		"resourceType": "Observation",
		"id": "example",
		"status": "final",
		"code": {
			"coding": [{
				"system": "http://loinc.org",
				"code": "15074-8",
				"display": "Glucose"
			}]
		}
	}`

	resource, err := UnmarshalResource([]byte(jsonData))
	require.NoError(t, err)

	observation, ok := resource.(*Observation)
	require.True(t, ok, "should be an *Observation")
	assert.Equal(t, "example", *observation.ID)
	assert.Equal(t, "final", observation.Status)
}

func TestUnmarshalResource_Bundle(t *testing.T) {
	jsonData := `{
		"resourceType": "Bundle",
		"id": "bundle-example",
		"type": "searchset",
		"total": 1
	}`

	resource, err := UnmarshalResource([]byte(jsonData))
	require.NoError(t, err)

	bundle, ok := resource.(*Bundle)
	require.True(t, ok, "should be a *Bundle")
	assert.Equal(t, "bundle-example", *bundle.ID)
	assert.Equal(t, "searchset", bundle.Type)
}

func TestUnmarshalResource_MissingResourceType(t *testing.T) {
	jsonData := `{
		"id": "example",
		"name": "Test"
	}`

	_, err := UnmarshalResource([]byte(jsonData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing resourceType")
}

func TestUnmarshalResource_UnknownResourceType(t *testing.T) {
	jsonData := `{
		"resourceType": "UnknownResource",
		"id": "example"
	}`

	_, err := UnmarshalResource([]byte(jsonData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestUnmarshalResource_InvalidJSON(t *testing.T) {
	jsonData := `{
		"resourceType": "Patient",
		"id": "example",
		invalid json here
	}`

	_, err := UnmarshalResource([]byte(jsonData))
	require.Error(t, err)
}

func TestUnmarshalResource_RoundTrip(t *testing.T) {
	// Create JSON with resourceType field
	jsonData := []byte(`{
		"resourceType": "Patient",
		"id": "round-trip-test",
		"gender": "female",
		"name": [{
			"family": "Smith",
			"given": ["Jane"]
		}]
	}`)

	// Unmarshal using factory
	resource, err := UnmarshalResource(jsonData)
	require.NoError(t, err)

	// Verify it's correct
	patient, ok := resource.(*Patient)
	require.True(t, ok)
	assert.Equal(t, "round-trip-test", *patient.ID)
	assert.Equal(t, "female", *patient.Gender)
	assert.Len(t, patient.Name, 1)
	assert.Equal(t, "Smith", *patient.Name[0].Family)
	assert.Len(t, patient.Name[0].Given, 1)
	assert.Equal(t, "Jane", patient.Name[0].Given[0])
}
