package resources

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir/primitives"
	"github.com/codeninja55/go-radx/fhir/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatient_Validate_Valid(t *testing.T) {
	// Create a valid patient
	birthDate := primitives.MustDate("1974-12-25")
	patient := Patient{
		ID:     stringPtr("example"),
		Active: boolPtr(true),
		Name: []HumanName{
			{
				Use:    stringPtr("official"),
				Family: stringPtr("Chalmers"),
				Given:  []string{"Peter", "James"},
			},
		},
		Gender:    stringPtr("male"),
		BirthDate: &birthDate,
		Telecom: []ContactPoint{
			{
				System: stringPtr("phone"),
				Value:  stringPtr("(03) 5555 6473"),
				Use:    stringPtr("work"),
			},
		},
	}

	err := patient.Validate()
	assert.NoError(t, err)
}

func TestPatient_Validate_InvalidGender(t *testing.T) {
	patient := Patient{
		ID:     stringPtr("example"),
		Gender: stringPtr("invalid"),
	}

	err := patient.Validate()
	require.Error(t, err)

	valErrs, ok := err.(*validation.Errors)
	require.True(t, ok)
	assert.True(t, valErrs.HasErrors())

	errList := valErrs.List()
	require.Len(t, errList, 1)
	assert.Contains(t, errList[0].Message, "invalid gender value")
}

func TestPatient_Validate_InvalidReference(t *testing.T) {
	patient := Patient{
		ID: stringPtr("example"),
		ManagingOrganization: &Reference{
			Reference: stringPtr("invalid-ref"),
		},
	}

	err := patient.Validate()
	require.Error(t, err)

	valErrs, ok := err.(*validation.Errors)
	require.True(t, ok)
	assert.True(t, valErrs.HasErrors())

	errList := valErrs.List()
	require.Len(t, errList, 1)
	assert.Contains(t, errList[0].Message, "invalid reference format")
}

func TestPatient_Validate_InvalidHumanNameUse(t *testing.T) {
	patient := Patient{
		ID: stringPtr("example"),
		Name: []HumanName{
			{
				Use:    stringPtr("invalid-use"),
				Family: stringPtr("Doe"),
			},
		},
	}

	err := patient.Validate()
	require.Error(t, err)

	valErrs, ok := err.(*validation.Errors)
	require.True(t, ok)
	assert.True(t, valErrs.HasErrors())

	errList := valErrs.List()
	require.Len(t, errList, 1)
	assert.Contains(t, errList[0].Field, "Patient.name")
	assert.Contains(t, errList[0].Message, "invalid use value")
}

func TestPatient_Validate_InvalidContactPointSystem(t *testing.T) {
	patient := Patient{
		ID: stringPtr("example"),
		Telecom: []ContactPoint{
			{
				System: stringPtr("invalid-system"),
				Value:  stringPtr("123"),
			},
		},
	}

	err := patient.Validate()
	require.Error(t, err)

	valErrs, ok := err.(*validation.Errors)
	require.True(t, ok)
	assert.True(t, valErrs.HasErrors())

	errList := valErrs.List()
	require.Len(t, errList, 1)
	assert.Contains(t, errList[0].Field, "Patient.telecom")
	assert.Contains(t, errList[0].Message, "invalid system value")
}

func TestPatient_Validate_InvalidLinkType(t *testing.T) {
	patient := Patient{
		ID: stringPtr("example"),
		Link: []PatientLink{
			{
				Other: Reference{
					Reference: stringPtr("Patient/123"),
				},
				Type: "invalid-type",
			},
		},
	}

	err := patient.Validate()
	require.Error(t, err)

	valErrs, ok := err.(*validation.Errors)
	require.True(t, ok)
	assert.True(t, valErrs.HasErrors())

	errList := valErrs.List()
	require.Len(t, errList, 1)
	assert.Contains(t, errList[0].Field, "Patient.link")
	assert.Contains(t, errList[0].Message, "invalid type value")
}

func TestPatient_Validate_ValidLink(t *testing.T) {
	patient := Patient{
		ID: stringPtr("example"),
		Link: []PatientLink{
			{
				Other: Reference{
					Reference: stringPtr("Patient/123"),
				},
				Type: "seealso",
			},
		},
	}

	err := patient.Validate()
	assert.NoError(t, err)
}

func TestPatient_Validate_MultipleErrors(t *testing.T) {
	patient := Patient{
		ID:     stringPtr("example"),
		Gender: stringPtr("invalid-gender"),
		Name: []HumanName{
			{
				Use:    stringPtr("invalid-use"),
				Family: stringPtr("Doe"),
			},
		},
		Telecom: []ContactPoint{
			{
				System: stringPtr("invalid-system"),
			},
		},
	}

	err := patient.Validate()
	require.Error(t, err)

	valErrs, ok := err.(*validation.Errors)
	require.True(t, ok)
	assert.True(t, valErrs.HasErrors())

	// Should have 3 errors: gender, name use, and telecom system
	errList := valErrs.List()
	assert.Len(t, errList, 3)
}

// Helper functions

func boolPtr(b bool) *bool {
	return &b
}
