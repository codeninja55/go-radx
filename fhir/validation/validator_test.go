package validation

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir/internal/testutil"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   any
		wantErr bool
		message string
	}{
		{
			name:    "non-nil value",
			field:   "Patient.name",
			value:   "John Doe",
			wantErr: false,
		},
		{
			name:    "nil value",
			field:   "Patient.name",
			value:   nil,
			wantErr: true,
			message: "required field is missing",
		},
		{
			name:    "empty string",
			field:   "Patient.name",
			value:   "",
			wantErr: true,
			message: "required field cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.field, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Field != tt.field {
					t.Errorf("expected field %q, got %q", tt.field, err.Field)
				}
				if err.Message != tt.message {
					t.Errorf("expected message %q, got %q", tt.message, err.Message)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateReference(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		ref     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative reference",
			field:   "Observation.subject",
			ref:     "Patient/123",
			wantErr: false,
		},
		{
			name:    "valid absolute URL",
			field:   "Observation.subject",
			ref:     "https://example.com/fhir/Patient/123",
			wantErr: false,
		},
		{
			name:    "empty reference",
			field:   "Observation.subject",
			ref:     "",
			wantErr: false,
		},
		{
			name:    "invalid format - no slash",
			field:   "Observation.subject",
			ref:     "Patient123",
			wantErr: true,
			errMsg:  "invalid reference format",
		},
		{
			name:    "invalid format - lowercase resource type",
			field:   "Observation.subject",
			ref:     "patient/123",
			wantErr: true,
			errMsg:  "invalid resource type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReference(tt.field, tt.ref)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				validErr, ok := err.(*Error)
				if !ok {
					t.Fatalf("expected *Error, got %T", err)
				}
				if validErr.Field != tt.field {
					t.Errorf("expected field %q, got %q", tt.field, validErr.Field)
				}
				if !contains(validErr.Message, tt.errMsg) {
					t.Errorf("expected message to contain %q, got %q", tt.errMsg, validErr.Message)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestErrors_List tests the List method of Errors
func TestErrors_List(t *testing.T) {
	errors := &Errors{}

	// Empty errors
	if len(errors.List()) != 0 {
		t.Error("List() should return empty slice for no errors")
	}

	// Add some errors
	errors.Add("field1", "error1")
	errors.Add("field2", "error2")
	errors.Addf("field3", "error %d", 3)

	list := errors.List()
	if len(list) != 3 {
		t.Errorf("List() = %d errors, want 3", len(list))
	}

	// Check error messages
	expectedFields := []string{"field1", "field2", "field3"}
	for i, err := range list {
		if err.Field != expectedFields[i] {
			t.Errorf("List()[%d].Field = %q, want %q", i, err.Field, expectedFields[i])
		}
	}
}

// TestValidateCardinality tests the ValidateCardinality function
func TestValidateCardinality(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		count   int
		min     int
		max     int
		wantErr bool
	}{
		{
			name:    "within bounds",
			field:   "field1",
			count:   2,
			min:     1,
			max:     3,
			wantErr: false,
		},
		{
			name:    "below minimum",
			field:   "field1",
			count:   0,
			min:     1,
			max:     3,
			wantErr: true,
		},
		{
			name:    "above maximum",
			field:   "field1",
			count:   5,
			min:     1,
			max:     3,
			wantErr: true,
		},
		{
			name:    "unlimited maximum",
			field:   "field1",
			count:   100,
			min:     0,
			max:     -1,
			wantErr: false,
		},
		{
			name:    "exact minimum",
			field:   "field1",
			count:   1,
			min:     1,
			max:     3,
			wantErr: false,
		},
		{
			name:    "exact maximum",
			field:   "field1",
			count:   3,
			min:     1,
			max:     3,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCardinality(tt.field, tt.count, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCardinality() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCheckRequired tests the checkRequired validator function
func TestCheckRequired(t *testing.T) {
	type TestStruct struct {
		RequiredField    *string `json:"requiredField" fhir:"required"`
		OptionalField    *string `json:"optionalField"`
		RequiredNonNil   *string `json:"requiredNonNil" fhir:"required"`
		RequiredWithCard *string `json:"requiredWithCard" fhir:"cardinality=1..1,required"`
	}

	validator := NewFHIRValidator()

	tests := []struct {
		name    string
		input   *TestStruct
		wantErr bool
		errMsg  string
	}{
		{
			name: "all required fields present",
			input: &TestStruct{
				RequiredField:    testutil.StringPtr("value"),
				RequiredNonNil:   testutil.StringPtr("value"),
				RequiredWithCard: testutil.StringPtr("value"),
			},
			wantErr: false,
		},
		{
			name: "missing required field",
			input: &TestStruct{
				RequiredField:    nil,
				RequiredNonNil:   testutil.StringPtr("value"),
				RequiredWithCard: testutil.StringPtr("value"),
			},
			wantErr: true,
			errMsg:  "RequiredField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestCheckEnum tests enum validation
func TestCheckEnum(t *testing.T) {
	type TestStruct struct {
		Status *string `json:"status" fhir:"enum=active|inactive|pending"`
		Type   *string `json:"type" fhir:"enum=typeA|typeB"`
	}

	validator := NewFHIRValidator()

	tests := []struct {
		name    string
		input   *TestStruct
		wantErr bool
	}{
		{
			name: "valid enum values",
			input: &TestStruct{
				Status: testutil.StringPtr("active"),
				Type:   testutil.StringPtr("typeA"),
			},
			wantErr: false,
		},
		{
			name: "invalid enum value",
			input: &TestStruct{
				Status: testutil.StringPtr("invalid"),
				Type:   testutil.StringPtr("typeA"),
			},
			wantErr: true,
		},
		{
			name: "nil enum field",
			input: &TestStruct{
				Status: nil,
				Type:   testutil.StringPtr("typeB"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCardinalityValidation tests cardinality constraints
func TestCardinalityValidation(t *testing.T) {
	type TestStruct struct {
		OneToOne   *string  `json:"oneToOne" fhir:"cardinality=1..1"`
		ZeroToMany []string `json:"zeroToMany" fhir:"cardinality=0..*"`
		OneToThree []string `json:"oneToThree" fhir:"cardinality=1..3"`
		ZeroToOne  *string  `json:"zeroToOne" fhir:"cardinality=0..1"`
	}

	validator := NewFHIRValidator()

	tests := []struct {
		name    string
		input   *TestStruct
		wantErr bool
		errMsg  string
	}{
		{
			name: "all cardinalities satisfied",
			input: &TestStruct{
				OneToOne:   testutil.StringPtr("value"),
				ZeroToMany: []string{"a", "b"},
				OneToThree: []string{"x"},
				ZeroToOne:  testutil.StringPtr("z"),
			},
			wantErr: false,
		},
		{
			name: "missing required 1..1 field",
			input: &TestStruct{
				OneToOne:   nil,
				ZeroToMany: []string{},
				OneToThree: []string{"x"},
			},
			wantErr: true,
			errMsg:  "OneToOne",
		},
		{
			name: "too many items in 1..3 field",
			input: &TestStruct{
				OneToOne:   testutil.StringPtr("value"),
				OneToThree: []string{"a", "b", "c", "d"},
			},
			wantErr: true,
			errMsg:  "OneToThree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestFHIRValidator_ChoiceType tests choice type mutual exclusion validation.
func TestFHIRValidator_ChoiceType(t *testing.T) {
	// Define test structures with choice types
	type PatientDeceased struct {
		DeceasedBoolean  *bool   `json:"deceasedBoolean,omitempty" fhir:"cardinality=0..1,summary,choice=deceased"`
		DeceasedDateTime *string `json:"deceasedDateTime,omitempty" fhir:"cardinality=0..1,summary,choice=deceased"`
	}

	type ObservationValue struct {
		ValueQuantity        *string `json:"valueQuantity,omitempty" fhir:"cardinality=0..1,choice=value"`
		ValueCodeableConcept *string `json:"valueCodeableConcept,omitempty" fhir:"cardinality=0..1,choice=value"`
		ValueString          *string `json:"valueString,omitempty" fhir:"cardinality=0..1,choice=value"`
	}

	validator := NewFHIRValidator()

	trueBool := true
	dateTime := "2023-01-15"
	quantity := "10 mg"
	codeableConcept := "test-code"
	stringValue := "test-string"

	tests := []struct {
		name    string
		input   any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid - only deceasedBoolean set",
			input: &PatientDeceased{
				DeceasedBoolean: &trueBool,
			},
			wantErr: false,
		},
		{
			name: "valid - only deceasedDateTime set",
			input: &PatientDeceased{
				DeceasedDateTime: &dateTime,
			},
			wantErr: false,
		},
		{
			name:    "valid - no choice fields set",
			input:   &PatientDeceased{},
			wantErr: false,
		},
		{
			name: "invalid - both deceased fields set",
			input: &PatientDeceased{
				DeceasedBoolean:  &trueBool,
				DeceasedDateTime: &dateTime,
			},
			wantErr: true,
			errMsg:  "choice type 'deceased' has multiple fields set",
		},
		{
			name: "valid - only valueQuantity set",
			input: &ObservationValue{
				ValueQuantity: &quantity,
			},
			wantErr: false,
		},
		{
			name: "valid - only valueCodeableConcept set",
			input: &ObservationValue{
				ValueCodeableConcept: &codeableConcept,
			},
			wantErr: false,
		},
		{
			name: "invalid - two value fields set",
			input: &ObservationValue{
				ValueQuantity: &quantity,
				ValueString:   &stringValue,
			},
			wantErr: true,
			errMsg:  "choice type 'value' has multiple fields set",
		},
		{
			name: "invalid - all three value fields set",
			input: &ObservationValue{
				ValueQuantity:        &quantity,
				ValueCodeableConcept: &codeableConcept,
				ValueString:          &stringValue,
			},
			wantErr: true,
			errMsg:  "choice type 'value' has multiple fields set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestFHIRValidator_ChoiceTypeNested tests choice type validation in nested structs.
func TestFHIRValidator_ChoiceTypeNested(t *testing.T) {
	type Component struct {
		ValueQuantity *string `json:"valueQuantity,omitempty" fhir:"cardinality=0..1,choice=value"`
		ValueString   *string `json:"valueString,omitempty" fhir:"cardinality=0..1,choice=value"`
	}

	type Observation struct {
		Component []Component `json:"component,omitempty" fhir:"cardinality=0..*"`
	}

	validator := NewFHIRValidator()

	quantity := "10 mg"
	stringValue := "test"

	tests := []struct {
		name    string
		input   *Observation
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid - single component with valueQuantity",
			input: &Observation{
				Component: []Component{
					{ValueQuantity: &quantity},
				},
			},
			wantErr: false,
		},
		{
			name: "valid - multiple components each with one value",
			input: &Observation{
				Component: []Component{
					{ValueQuantity: &quantity},
					{ValueString: &stringValue},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - component with multiple values set",
			input: &Observation{
				Component: []Component{
					{
						ValueQuantity: &quantity,
						ValueString:   &stringValue,
					},
				},
			},
			wantErr: true,
			errMsg:  "choice type 'value' has multiple fields set",
		},
		{
			name: "invalid - second component violates choice constraint",
			input: &Observation{
				Component: []Component{
					{ValueQuantity: &quantity}, // valid
					{
						ValueQuantity: &quantity,
						ValueString:   &stringValue, // invalid - both set
					},
				},
			},
			wantErr: true,
			errMsg:  "choice type 'value' has multiple fields set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestFHIRValidator_ChoiceTypeWithOtherValidation tests choice validation combined with other rules.
func TestFHIRValidator_ChoiceTypeWithOtherValidation(t *testing.T) {
	type MedicationRequest struct {
		Status              *string `json:"status,omitempty" fhir:"cardinality=1..1,required,enum=active|completed|canceled"`
		DosageInstruction   *string `json:"dosageInstruction,omitempty" fhir:"cardinality=0..1"`
		MedicationReference *string `json:"medicationReference,omitempty" fhir:"cardinality=0..1,choice=medication"`
		MedicationCodeable  *string `json:"medicationCodeableConcept,omitempty" fhir:"cardinality=0..1,choice=medication"`
	}

	validator := NewFHIRValidator()

	activeStatus := "active"
	invalidStatus := "pending"
	reference := "Medication/123"
	codeable := "code-123"

	tests := []struct {
		name    string
		input   *MedicationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid - required field and one choice field set",
			input: &MedicationRequest{
				Status:              &activeStatus,
				MedicationReference: &reference,
			},
			wantErr: false,
		},
		{
			name: "invalid - missing required status",
			input: &MedicationRequest{
				MedicationReference: &reference,
			},
			wantErr: true,
			errMsg:  "required field is missing",
		},
		{
			name: "invalid - invalid enum value",
			input: &MedicationRequest{
				Status:              &invalidStatus,
				MedicationReference: &reference,
			},
			wantErr: true,
			errMsg:  "invalid enum value",
		},
		{
			name: "invalid - both choice fields set",
			input: &MedicationRequest{
				Status:              &activeStatus,
				MedicationReference: &reference,
				MedicationCodeable:  &codeable,
			},
			wantErr: true,
			errMsg:  "choice type 'medication' has multiple fields set",
		},
		{
			name: "invalid - multiple errors (missing required and choice violation)",
			input: &MedicationRequest{
				MedicationReference: &reference,
				MedicationCodeable:  &codeable,
			},
			wantErr: true,
			// Either error message is acceptable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}
