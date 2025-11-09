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
