package validation

import (
	"github.com/codeninja55/go-radx/fhir/internal/testutil"
	"strings"
	"testing"
)

func TestChoiceTypeValidation(t *testing.T) {
	validator := NewFHIRValidator()

	// Test struct simulating a FHIR choice type
	type TestResource struct {
		DeceasedBoolean  *bool   `json:"deceasedBoolean,omitempty" fhir:"cardinality=0..1,choice=deceased"`
		DeceasedDateTime *string `json:"deceasedDateTime,omitempty" fhir:"cardinality=0..1,choice=deceased"`
	}

	tests := []struct {
		name      string
		resource  *TestResource
		wantError bool
		errorMsg  string
	}{
		{
			name:      "no fields set - valid",
			resource:  &TestResource{},
			wantError: false,
		},
		{
			name: "only boolean set - valid",
			resource: &TestResource{
				DeceasedBoolean: testutil.BoolPtr(true),
			},
			wantError: false,
		},
		{
			name: "only dateTime set - valid",
			resource: &TestResource{
				DeceasedDateTime: testutil.StringPtr("2024-01-01"),
			},
			wantError: false,
		},
		{
			name: "both fields set - invalid",
			resource: &TestResource{
				DeceasedBoolean:  testutil.BoolPtr(true),
				DeceasedDateTime: testutil.StringPtr("2024-01-01"),
			},
			wantError: true,
			errorMsg:  "deceased",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.resource)
			if tt.wantError {
				if err == nil {
					t.Error("expected validation error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
			}
		})
	}
}

func TestChoiceTypeValidation_MultipleGroups(t *testing.T) {
	validator := NewFHIRValidator()

	type TestResource struct {
		// First choice group
		DeceasedBoolean  *bool   `json:"deceasedBoolean,omitempty" fhir:"choice=deceased"`
		DeceasedDateTime *string `json:"deceasedDateTime,omitempty" fhir:"choice=deceased"`

		// Second choice group
		MultipleBirthBoolean *bool `json:"multipleBirthBoolean,omitempty" fhir:"choice=multipleBirth"`
		MultipleBirthInteger *int  `json:"multipleBirthInteger,omitempty" fhir:"choice=multipleBirth"`
	}

	tests := []struct {
		name      string
		resource  *TestResource
		wantError bool
		errorMsg  string
	}{
		{
			name: "one from each group - valid",
			resource: &TestResource{
				DeceasedBoolean:      testutil.BoolPtr(true),
				MultipleBirthBoolean: testutil.BoolPtr(false),
			},
			wantError: false,
		},
		{
			name: "multiple from first group - invalid",
			resource: &TestResource{
				DeceasedBoolean:      testutil.BoolPtr(true),
				DeceasedDateTime:     testutil.StringPtr("2024-01-01"),
				MultipleBirthBoolean: testutil.BoolPtr(false),
			},
			wantError: true,
			errorMsg:  "deceased",
		},
		{
			name: "multiple from second group - invalid",
			resource: &TestResource{
				DeceasedBoolean:      testutil.BoolPtr(true),
				MultipleBirthBoolean: testutil.BoolPtr(false),
				MultipleBirthInteger: testutil.IntPtr(2),
			},
			wantError: true,
			errorMsg:  "multipleBirth",
		},
		{
			name: "multiple from both groups - invalid",
			resource: &TestResource{
				DeceasedBoolean:      testutil.BoolPtr(true),
				DeceasedDateTime:     testutil.StringPtr("2024-01-01"),
				MultipleBirthBoolean: testutil.BoolPtr(false),
				MultipleBirthInteger: testutil.IntPtr(2),
			},
			wantError: true,
			// Should report errors for both groups
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.resource)
			if tt.wantError {
				if err == nil {
					t.Error("expected validation error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
			}
		})
	}
}

func TestChoiceTypeValidation_NestedStructs(t *testing.T) {
	validator := NewFHIRValidator()

	type Inner struct {
		ValueBoolean *bool   `json:"valueBoolean,omitempty" fhir:"choice=value"`
		ValueString  *string `json:"valueString,omitempty" fhir:"choice=value"`
	}

	type Outer struct {
		Name  string `json:"name"`
		Inner *Inner `json:"inner,omitempty"`
	}

	tests := []struct {
		name      string
		resource  *Outer
		wantError bool
	}{
		{
			name: "nested valid - one choice",
			resource: &Outer{
				Name: "test",
				Inner: &Inner{
					ValueBoolean: testutil.BoolPtr(true),
				},
			},
			wantError: false,
		},
		{
			name: "nested invalid - multiple choices",
			resource: &Outer{
				Name: "test",
				Inner: &Inner{
					ValueBoolean: testutil.BoolPtr(true),
					ValueString:  testutil.StringPtr("test"),
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.resource)
			if tt.wantError {
				if err == nil {
					t.Error("expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
			}
		})
	}
}

func TestChoiceTypeValidation_ZeroValues(t *testing.T) {
	validator := NewFHIRValidator()

	type TestResource struct {
		ValueBoolean *bool `json:"valueBoolean,omitempty" fhir:"choice=value"`
		ValueInteger *int  `json:"valueInteger,omitempty" fhir:"choice=value"`
	}

	// Zero values should be considered "set" if they're explicit pointers
	falseVal := false
	zeroVal := 0

	tests := []struct {
		name      string
		resource  *TestResource
		wantError bool
	}{
		{
			name: "explicit false - set",
			resource: &TestResource{
				ValueBoolean: &falseVal,
			},
			wantError: false,
		},
		{
			name: "explicit zero - set",
			resource: &TestResource{
				ValueInteger: &zeroVal,
			},
			wantError: false,
		},
		{
			name: "both zero values - invalid",
			resource: &TestResource{
				ValueBoolean: &falseVal,
				ValueInteger: &zeroVal,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.resource)
			if tt.wantError {
				if err == nil {
					t.Error("expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected validation error: %v", err)
				}
			}
		})
	}
}
