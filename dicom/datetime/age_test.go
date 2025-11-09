package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseAge_Valid tests parsing valid DICOM Age String (AS) values.
func TestParseAge_Valid(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue int
		wantUnit  AgeUnit
	}{
		{
			name:      "days",
			input:     "007D",
			wantValue: 7,
			wantUnit:  Days,
		},
		{
			name:      "weeks",
			input:     "004W",
			wantValue: 4,
			wantUnit:  Weeks,
		},
		{
			name:      "months",
			input:     "006M",
			wantValue: 6,
			wantUnit:  Months,
		},
		{
			name:      "years",
			input:     "042Y",
			wantValue: 42,
			wantUnit:  Years,
		},
		{
			name:      "zero age",
			input:     "000D",
			wantValue: 0,
			wantUnit:  Days,
		},
		{
			name:      "maximum age",
			input:     "999Y",
			wantValue: 999,
			wantUnit:  Years,
		},
		{
			name:      "single digit with leading zeros",
			input:     "001M",
			wantValue: 1,
			wantUnit:  Months,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age, err := ParseAge(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantValue, age.Value)
			assert.Equal(t, tt.wantUnit, age.Unit)
		})
	}
}

// TestParseAge_Invalid tests error handling for invalid Age String values.
func TestParseAge_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: "empty",
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: "empty",
		},
		{
			name:    "too short",
			input:   "42Y",
			wantErr: "invalid format",
		},
		{
			name:    "too long",
			input:   "0042Y",
			wantErr: "invalid format",
		},
		{
			name:    "non-numeric value",
			input:   "ABCD",
			wantErr: "invalid format",
		},
		{
			name:    "invalid unit",
			input:   "042X",
			wantErr: "invalid format",
		},
		{
			name:    "lowercase unit",
			input:   "042y",
			wantErr: "invalid format",
		},
		{
			name:    "negative value",
			input:   "-042Y",
			wantErr: "invalid format",
		},
		{
			name:    "decimal value",
			input:   "4.2Y",
			wantErr: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAge(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestAge_Duration tests conversion from Age to time.Duration.
func TestAge_Duration(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDuration time.Duration
	}{
		{
			name:         "1 day",
			input:        "001D",
			wantDuration: 24 * time.Hour,
		},
		{
			name:         "7 days",
			input:        "007D",
			wantDuration: 7 * 24 * time.Hour,
		},
		{
			name:         "1 week",
			input:        "001W",
			wantDuration: 7 * 24 * time.Hour,
		},
		{
			name:         "4 weeks",
			input:        "004W",
			wantDuration: 4 * 7 * 24 * time.Hour,
		},
		{
			name:         "1 month (30.4375 days)",
			input:        "001M",
			wantDuration: time.Duration(30.4375 * 24 * float64(time.Hour)),
		},
		{
			name:         "6 months",
			input:        "006M",
			wantDuration: time.Duration(6 * 30.4375 * 24 * float64(time.Hour)),
		},
		{
			name:         "1 year (365.25 days)",
			input:        "001Y",
			wantDuration: time.Duration(365.25 * 24 * float64(time.Hour)),
		},
		{
			name:         "42 years",
			input:        "042Y",
			wantDuration: time.Duration(42 * 365.25 * 24 * float64(time.Hour)),
		},
		{
			name:         "zero age",
			input:        "000D",
			wantDuration: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age, err := ParseAge(tt.input)
			require.NoError(t, err)

			got := age.Duration()
			assert.Equal(t, tt.wantDuration, got)
		})
	}
}

// TestAge_DCM tests formatting Age back to DICOM AS format.
func TestAge_DCM(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "days",
			input: "007D",
			want:  "007D",
		},
		{
			name:  "weeks",
			input: "004W",
			want:  "004W",
		},
		{
			name:  "months",
			input: "006M",
			want:  "006M",
		},
		{
			name:  "years",
			input: "042Y",
			want:  "042Y",
		},
		{
			name:  "zero age",
			input: "000D",
			want:  "000D",
		},
		{
			name:  "maximum age",
			input: "999Y",
			want:  "999Y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age, err := ParseAge(tt.input)
			require.NoError(t, err)

			got := age.DCM()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestAge_String tests human-readable string formatting.
func TestAge_String(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "days",
			input: "007D",
			want:  "7 days",
		},
		{
			name:  "single day",
			input: "001D",
			want:  "1 day",
		},
		{
			name:  "weeks",
			input: "004W",
			want:  "4 weeks",
		},
		{
			name:  "single week",
			input: "001W",
			want:  "1 week",
		},
		{
			name:  "months",
			input: "006M",
			want:  "6 months",
		},
		{
			name:  "single month",
			input: "001M",
			want:  "1 month",
		},
		{
			name:  "years",
			input: "042Y",
			want:  "42 years",
		},
		{
			name:  "single year",
			input: "001Y",
			want:  "1 year",
		},
		{
			name:  "zero age",
			input: "000D",
			want:  "0 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age, err := ParseAge(tt.input)
			require.NoError(t, err)

			got := age.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestAge_RoundTrip tests that parse → format → parse yields identical results.
func TestAge_RoundTrip(t *testing.T) {
	inputs := []string{
		"007D",
		"004W",
		"006M",
		"042Y",
		"000D",
		"999Y",
		"001M",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			age1, err := ParseAge(input)
			require.NoError(t, err)

			formatted := age1.DCM()
			age2, err := ParseAge(formatted)
			require.NoError(t, err)

			assert.Equal(t, age1.Value, age2.Value)
			assert.Equal(t, age1.Unit, age2.Unit)
		})
	}
}
