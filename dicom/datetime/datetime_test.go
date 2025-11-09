package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseDateTime_FullDateTime tests parsing complete datetime with timezone.
func TestParseDateTime_FullDateTime(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantYear   int
		wantMonth  time.Month
		wantDay    int
		wantHour   int
		wantMin    int
		wantSec    int
		wantMicro  int
		wantOffset int // offset in seconds
		wantPrec   PrecisionLevel
		wantNoOff  bool
	}{
		{
			name:       "full datetime with positive timezone",
			input:      "20231015143025.123456+1000",
			wantYear:   2023,
			wantMonth:  time.October,
			wantDay:    15,
			wantHour:   14,
			wantMin:    30,
			wantSec:    25,
			wantMicro:  123456,
			wantOffset: 10 * 3600, // +10 hours in seconds
			wantPrec:   PrecisionMS6,
			wantNoOff:  false,
		},
		{
			name:       "full datetime with negative timezone",
			input:      "20231015143025.123456-0500",
			wantYear:   2023,
			wantMonth:  time.October,
			wantDay:    15,
			wantHour:   14,
			wantMin:    30,
			wantSec:    25,
			wantMicro:  123456,
			wantOffset: -5 * 3600, // -5 hours in seconds
			wantPrec:   PrecisionMS6,
			wantNoOff:  false,
		},
		{
			name:       "full datetime with UTC (+0000)",
			input:      "20231015143025+0000",
			wantYear:   2023,
			wantMonth:  time.October,
			wantDay:    15,
			wantHour:   14,
			wantMin:    30,
			wantSec:    25,
			wantMicro:  0,
			wantOffset: 0,
			wantPrec:   PrecisionSeconds,
			wantNoOff:  false,
		},
		{
			name:       "full datetime with UTC (-0000)",
			input:      "20231015143025-0000",
			wantYear:   2023,
			wantMonth:  time.October,
			wantDay:    15,
			wantHour:   14,
			wantMin:    30,
			wantSec:    25,
			wantMicro:  0,
			wantOffset: 0,
			wantPrec:   PrecisionSeconds,
			wantNoOff:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := ParseDateTime(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantYear, dt.Time.Year())
			assert.Equal(t, tt.wantMonth, dt.Time.Month())
			assert.Equal(t, tt.wantDay, dt.Time.Day())
			assert.Equal(t, tt.wantHour, dt.Time.Hour())
			assert.Equal(t, tt.wantMin, dt.Time.Minute())
			assert.Equal(t, tt.wantSec, dt.Time.Second())
			assert.Equal(t, tt.wantMicro, dt.Time.Nanosecond()/1000)

			_, offset := dt.Time.Zone()
			assert.Equal(t, tt.wantOffset, offset)
			assert.Equal(t, tt.wantPrec, dt.Precision)
			assert.Equal(t, tt.wantNoOff, dt.NoOffset)
		})
	}
}

// TestParseDateTime_WithoutTimezone tests parsing datetime without timezone.
func TestParseDateTime_WithoutTimezone(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantHour  int
		wantMin   int
		wantSec   int
		wantPrec  PrecisionLevel
	}{
		{
			name:      "full datetime without timezone",
			input:     "20231015143025",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   15,
			wantHour:  14,
			wantMin:   30,
			wantSec:   25,
			wantPrec:  PrecisionSeconds,
		},
		{
			name:      "datetime with microseconds, no timezone",
			input:     "20231015143025.123456",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   15,
			wantHour:  14,
			wantMin:   30,
			wantSec:   25,
			wantPrec:  PrecisionMS6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := ParseDateTime(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantYear, dt.Time.Year())
			assert.Equal(t, tt.wantMonth, dt.Time.Month())
			assert.Equal(t, tt.wantDay, dt.Time.Day())
			assert.Equal(t, tt.wantHour, dt.Time.Hour())
			assert.Equal(t, tt.wantMin, dt.Time.Minute())
			assert.Equal(t, tt.wantSec, dt.Time.Second())
			assert.Equal(t, tt.wantPrec, dt.Precision)
			assert.True(t, dt.NoOffset, "NoOffset flag should be set")
		})
	}
}

// TestParseDateTime_PartialDateTime tests parsing datetime with reduced precision.
func TestParseDateTime_PartialDateTime(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantHour  int
		wantMin   int
		wantSec   int
		wantPrec  PrecisionLevel
	}{
		{
			name:      "date and time without seconds",
			input:     "202310151430",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   15,
			wantHour:  14,
			wantMin:   30,
			wantSec:   0, // default
			wantPrec:  PrecisionMinutes,
		},
		{
			name:      "date and hours only",
			input:     "2023101514",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   15,
			wantHour:  14,
			wantMin:   0, // default
			wantSec:   0, // default
			wantPrec:  PrecisionHours,
		},
		{
			name:      "date only",
			input:     "20231015",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   15,
			wantHour:  0, // default
			wantMin:   0, // default
			wantSec:   0, // default
			wantPrec:  PrecisionDay,
		},
		{
			name:      "year and month only",
			input:     "202310",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   1, // default
			wantHour:  0,
			wantMin:   0,
			wantSec:   0,
			wantPrec:  PrecisionMonth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := ParseDateTime(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantYear, dt.Time.Year())
			assert.Equal(t, tt.wantMonth, dt.Time.Month())
			assert.Equal(t, tt.wantDay, dt.Time.Day())
			assert.Equal(t, tt.wantHour, dt.Time.Hour())
			assert.Equal(t, tt.wantMin, dt.Time.Minute())
			assert.Equal(t, tt.wantSec, dt.Time.Second())
			assert.Equal(t, tt.wantPrec, dt.Precision)
			assert.True(t, dt.NoOffset, "NoOffset should be true for partial datetime")
		})
	}
}

// TestParseDateTime_TimezoneEdgeCases tests edge case timezone values.
func TestParseDateTime_TimezoneEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantOffset int // offset in seconds
	}{
		{
			name:       "max positive timezone +1400",
			input:      "20231015143025+1400",
			wantOffset: 14 * 3600,
		},
		{
			name:       "max negative timezone -1200",
			input:      "20231015143025-1200",
			wantOffset: -12 * 3600,
		},
		{
			name:       "half-hour offset +0530",
			input:      "20231015143025+0530",
			wantOffset: (5*3600 + 30*60),
		},
		{
			name:       "half-hour negative -0930",
			input:      "20231015143025-0930",
			wantOffset: -(9*3600 + 30*60),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := ParseDateTime(tt.input)
			require.NoError(t, err)

			_, offset := dt.Time.Zone()
			assert.Equal(t, tt.wantOffset, offset)
			assert.False(t, dt.NoOffset)
		})
	}
}

// TestParseDateTime_Invalid tests error handling for invalid datetime strings.
func TestParseDateTime_Invalid(t *testing.T) {
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
			name:    "too short",
			input:   "202",
			wantErr: "invalid format",
		},
		{
			name:    "invalid date component",
			input:   "20231315143025",
			wantErr: "month",
		},
		{
			name:    "invalid time component",
			input:   "20231015256025",
			wantErr: "hour",
		},
		{
			name:    "invalid timezone format",
			input:   "20231015143025+99",
			wantErr: "invalid format",
		},
		{
			name:    "invalid timezone hour",
			input:   "20231015143025+2500",
			wantErr: "timezone",
		},
		{
			name:    "invalid timezone minute",
			input:   "20231015143025+1070",
			wantErr: "timezone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDateTime(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestDateTime_DCM tests formatting DateTime back to DICOM string format.
func TestDateTime_DCM(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full datetime with timezone",
			input: "20231015143025+1000",
			want:  "20231015143025+1000",
		},
		{
			name:  "full datetime without timezone",
			input: "20231015143025",
			want:  "20231015143025",
		},
		{
			name:  "datetime with microseconds and timezone",
			input: "20231015143025.123456-0500",
			want:  "20231015143025.123456-0500",
		},
		{
			name:  "partial datetime (minutes)",
			input: "202310151430",
			want:  "202310151430",
		},
		{
			name:  "partial datetime (hours)",
			input: "2023101514",
			want:  "2023101514",
		},
		{
			name:  "date only",
			input: "20231015",
			want:  "20231015",
		},
		{
			name:  "year-month only",
			input: "202310",
			want:  "202310",
		},
		{
			name:  "year only",
			input: "2023",
			want:  "2023",
		},
		{
			name:  "with 1 decimal place",
			input: "20231015143025.1",
			want:  "20231015143025.1",
		},
		{
			name:  "with 2 decimal places",
			input: "20231015143025.12",
			want:  "20231015143025.12",
		},
		{
			name:  "with 3 decimal places",
			input: "20231015143025.123",
			want:  "20231015143025.123",
		},
		{
			name:  "with 4 decimal places",
			input: "20231015143025.1234",
			want:  "20231015143025.1234",
		},
		{
			name:  "with 5 decimal places",
			input: "20231015143025.12345",
			want:  "20231015143025.12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := ParseDateTime(tt.input)
			require.NoError(t, err)

			got := dt.DCM()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDateTime_String tests human-readable string formatting.
func TestDateTime_String(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full datetime with timezone",
			input: "20231015143025+1000",
			want:  "2023-10-15 14:30:25 +1000",
		},
		{
			name:  "full datetime without timezone",
			input: "20231015143025",
			want:  "2023-10-15 14:30:25 UTC",
		},
		{
			name:  "datetime with microseconds",
			input: "20231015143025.123456",
			want:  "2023-10-15 14:30:25.123456 UTC",
		},
		{
			name:  "date only",
			input: "20231015",
			want:  "2023-10-15 UTC",
		},
		{
			name:  "year-month only",
			input: "202310",
			want:  "2023-10 UTC",
		},
		{
			name:  "year only",
			input: "2023",
			want:  "2023 UTC",
		},
		{
			name:  "hours only",
			input: "2023101514",
			want:  "2023-10-15 14 UTC",
		},
		{
			name:  "minutes precision",
			input: "202310151430",
			want:  "2023-10-15 14:30 UTC",
		},
		{
			name:  "1 decimal place",
			input: "20231015143025.1",
			want:  "2023-10-15 14:30:25.1 UTC",
		},
		{
			name:  "2 decimal places",
			input: "20231015143025.12",
			want:  "2023-10-15 14:30:25.12 UTC",
		},
		{
			name:  "3 decimal places",
			input: "20231015143025.123",
			want:  "2023-10-15 14:30:25.123 UTC",
		},
		{
			name:  "4 decimal places",
			input: "20231015143025.1234",
			want:  "2023-10-15 14:30:25.1234 UTC",
		},
		{
			name:  "5 decimal places",
			input: "20231015143025.12345",
			want:  "2023-10-15 14:30:25.12345 UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := ParseDateTime(tt.input)
			require.NoError(t, err)

			got := dt.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDateTime_RoundTrip tests that parse → format → parse yields identical results.
func TestDateTime_RoundTrip(t *testing.T) {
	inputs := []string{
		"20231015143025+1000",
		"20231015143025-0500",
		"20231015143025",
		"20231015143025.123456+1000",
		"202310151430",
		"20231015",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			dt1, err := ParseDateTime(input)
			require.NoError(t, err)

			formatted := dt1.DCM()
			dt2, err := ParseDateTime(formatted)
			require.NoError(t, err)

			assert.Equal(t, dt1.Time.Year(), dt2.Time.Year())
			assert.Equal(t, dt1.Time.Month(), dt2.Time.Month())
			assert.Equal(t, dt1.Time.Day(), dt2.Time.Day())
			assert.Equal(t, dt1.Time.Hour(), dt2.Time.Hour())
			assert.Equal(t, dt1.Time.Minute(), dt2.Time.Minute())
			assert.Equal(t, dt1.Time.Second(), dt2.Time.Second())
			assert.Equal(t, dt1.Time.Nanosecond(), dt2.Time.Nanosecond())
			assert.Equal(t, dt1.Precision, dt2.Precision)
			assert.Equal(t, dt1.NoOffset, dt2.NoOffset)

			if !dt1.NoOffset {
				_, off1 := dt1.Time.Zone()
				_, off2 := dt2.Time.Zone()
				assert.Equal(t, off1, off2)
			}
		})
	}
}
