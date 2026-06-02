package fhir_test

import (
	"testing"

	"github.com/codeninja55/go-radx/fhir"
)

func TestReleaseConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  fhir.Release
		want string
	}{
		{"R4 is FHIR 4.0.1", fhir.R4, "4.0.1"},
		{"R5 is FHIR 5.0.0", fhir.R5, "5.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.got) != tt.want {
				t.Errorf("Release = %q, want %q", string(tt.got), tt.want)
			}
			if tt.got.String() != tt.want {
				t.Errorf("Release.String() = %q, want %q", tt.got.String(), tt.want)
			}
		})
	}
}
