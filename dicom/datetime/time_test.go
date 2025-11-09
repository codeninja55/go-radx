package datetime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTime_FullTime tests parsing complete HHMMSS.FFFFFF times.
func TestParseTime_FullTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantHour  int
		wantMin   int
		wantSec   int
		wantMicro int
		wantPrec  PrecisionLevel
	}{
		{
			name:      "full time with microseconds",
			input:     "143025.123456",
			wantHour:  14,
			wantMin:   30,
			wantSec:   25,
			wantMicro: 123456,
			wantPrec:  PrecisionMS6,
		},
		{
			name:      "midnight",
			input:     "000000.000000",
			wantHour:  0,
			wantMin:   0,
			wantSec:   0,
			wantMicro: 0,
			wantPrec:  PrecisionMS6,
		},
		{
			name:      "end of day",
			input:     "235959.999999",
			wantHour:  23,
			wantMin:   59,
			wantSec:   59,
			wantMicro: 999999,
			wantPrec:  PrecisionMS6,
		},
		{
			name:      "noon",
			input:     "120000.000000",
			wantHour:  12,
			wantMin:   0,
			wantSec:   0,
			wantMicro: 0,
			wantPrec:  PrecisionMS6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tim, err := ParseTime(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantHour, tim.Time.Hour())
			assert.Equal(t, tt.wantMin, tim.Time.Minute())
			assert.Equal(t, tt.wantSec, tim.Time.Second())
			assert.Equal(t, tt.wantMicro, tim.Time.Nanosecond()/1000)
			assert.Equal(t, tt.wantPrec, tim.Precision)
		})
	}
}

// TestParseTime_PartialTime tests parsing times with reduced precision.
func TestParseTime_PartialTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantHour  int
		wantMin   int
		wantSec   int
		wantMicro int
		wantPrec  PrecisionLevel
	}{
		{
			name:      "hours, minutes, seconds",
			input:     "143025",
			wantHour:  14,
			wantMin:   30,
			wantSec:   25,
			wantMicro: 0,
			wantPrec:  PrecisionSeconds,
		},
		{
			name:      "hours and minutes only",
			input:     "1430",
			wantHour:  14,
			wantMin:   30,
			wantSec:   0, // default
			wantMicro: 0,
			wantPrec:  PrecisionMinutes,
		},
		{
			name:      "hours only",
			input:     "14",
			wantHour:  14,
			wantMin:   0, // default
			wantSec:   0, // default
			wantMicro: 0,
			wantPrec:  PrecisionHours,
		},
		{
			name:      "midnight hours only",
			input:     "00",
			wantHour:  0,
			wantMin:   0,
			wantSec:   0,
			wantMicro: 0,
			wantPrec:  PrecisionHours,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tim, err := ParseTime(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantHour, tim.Time.Hour())
			assert.Equal(t, tt.wantMin, tim.Time.Minute())
			assert.Equal(t, tt.wantSec, tim.Time.Second())
			assert.Equal(t, tt.wantMicro, tim.Time.Nanosecond()/1000)
			assert.Equal(t, tt.wantPrec, tim.Precision)
		})
	}
}

// TestParseTime_FractionalSeconds tests parsing fractional seconds with varying precision.
func TestParseTime_FractionalSeconds(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMicro int
		wantPrec  PrecisionLevel
	}{
		{
			name:      "1 decimal place (decisecond)",
			input:     "143025.1",
			wantMicro: 100000, // .1 = 100000 microseconds
			wantPrec:  PrecisionMS1,
		},
		{
			name:      "2 decimal places (centisecond)",
			input:     "143025.12",
			wantMicro: 120000, // .12 = 120000 microseconds
			wantPrec:  PrecisionMS2,
		},
		{
			name:      "3 decimal places (millisecond)",
			input:     "143025.123",
			wantMicro: 123000, // .123 = 123000 microseconds
			wantPrec:  PrecisionMS3,
		},
		{
			name:      "4 decimal places",
			input:     "143025.1234",
			wantMicro: 123400,
			wantPrec:  PrecisionMS4,
		},
		{
			name:      "5 decimal places",
			input:     "143025.12345",
			wantMicro: 123450,
			wantPrec:  PrecisionMS5,
		},
		{
			name:      "6 decimal places (microsecond)",
			input:     "143025.123456",
			wantMicro: 123456,
			wantPrec:  PrecisionMS6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tim, err := ParseTime(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantMicro, tim.Time.Nanosecond()/1000)
			assert.Equal(t, tt.wantPrec, tim.Precision)
		})
	}
}

// TestParseTime_Invalid tests error handling for invalid time strings.
func TestParseTime_Invalid(t *testing.T) {
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
			input:   "1",
			wantErr: "invalid format",
		},
		{
			name:    "non-numeric",
			input:   "14AB25",
			wantErr: "invalid format",
		},
		{
			name:    "invalid hour",
			input:   "245959",
			wantErr: "hour",
		},
		{
			name:    "invalid minute",
			input:   "146025",
			wantErr: "minute",
		},
		{
			name:    "invalid second",
			input:   "143060",
			wantErr: "second",
		},
		{
			name:    "hour 24",
			input:   "240000",
			wantErr: "hour",
		},
		{
			name:    "minute 60",
			input:   "146000",
			wantErr: "minute",
		},
		{
			name:    "second 60",
			input:   "143060",
			wantErr: "second",
		},
		{
			name:    "too many fractional digits",
			input:   "143025.1234567",
			wantErr: "invalid format",
		},
		{
			name:    "negative fractional",
			input:   "143025.-123456",
			wantErr: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTime(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestTime_DCM tests formatting Time back to DICOM string format.
func TestTime_DCM(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full time with microseconds",
			input: "143025.123456",
			want:  "143025.123456",
		},
		{
			name:  "time without fractional seconds",
			input: "143025",
			want:  "143025",
		},
		{
			name:  "hours and minutes",
			input: "1430",
			want:  "1430",
		},
		{
			name:  "hours only",
			input: "14",
			want:  "14",
		},
		{
			name:  "1 decimal place",
			input: "143025.1",
			want:  "143025.1",
		},
		{
			name:  "2 decimal places",
			input: "143025.12",
			want:  "143025.12",
		},
		{
			name:  "3 decimal places",
			input: "143025.123",
			want:  "143025.123",
		},
		{
			name:  "4 decimal places",
			input: "143025.1234",
			want:  "143025.1234",
		},
		{
			name:  "5 decimal places",
			input: "143025.12345",
			want:  "143025.12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tim, err := ParseTime(tt.input)
			require.NoError(t, err)

			got := tim.DCM()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestTime_String tests human-readable string formatting.
func TestTime_String(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full time with microseconds",
			input: "143025.123456",
			want:  "14:30:25.123456",
		},
		{
			name:  "time without fractional seconds",
			input: "143025",
			want:  "14:30:25",
		},
		{
			name:  "hours and minutes",
			input: "1430",
			want:  "14:30",
		},
		{
			name:  "hours only",
			input: "14",
			want:  "14",
		},
		{
			name:  "1 decimal place",
			input: "143025.1",
			want:  "14:30:25.1",
		},
		{
			name:  "2 decimal places",
			input: "143025.12",
			want:  "14:30:25.12",
		},
		{
			name:  "3 decimal places",
			input: "143025.123",
			want:  "14:30:25.123",
		},
		{
			name:  "4 decimal places",
			input: "143025.1234",
			want:  "14:30:25.1234",
		},
		{
			name:  "5 decimal places",
			input: "143025.12345",
			want:  "14:30:25.12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tim, err := ParseTime(tt.input)
			require.NoError(t, err)

			got := tim.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestTime_RoundTrip tests that parse → format → parse yields identical results.
func TestTime_RoundTrip(t *testing.T) {
	inputs := []string{
		"143025.123456",
		"143025",
		"1430",
		"14",
		"143025.1",
		"143025.12",
		"143025.123",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			tim1, err := ParseTime(input)
			require.NoError(t, err)

			formatted := tim1.DCM()
			tim2, err := ParseTime(formatted)
			require.NoError(t, err)

			assert.Equal(t, tim1.Time.Hour(), tim2.Time.Hour())
			assert.Equal(t, tim1.Time.Minute(), tim2.Time.Minute())
			assert.Equal(t, tim1.Time.Second(), tim2.Time.Second())
			assert.Equal(t, tim1.Time.Nanosecond(), tim2.Time.Nanosecond())
			assert.Equal(t, tim1.Precision, tim2.Precision)
		})
	}
}
