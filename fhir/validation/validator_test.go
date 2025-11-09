package validation

import (
	"testing"
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
