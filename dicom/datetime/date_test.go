package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseDate_FullDate tests parsing complete YYYYMMDD dates.
func TestParseDate_FullDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantPrec  PrecisionLevel
		wantNEMA  bool
	}{
		{
			name:      "standard date",
			input:     "20231015",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   15,
			wantPrec:  PrecisionDay,
			wantNEMA:  false,
		},
		{
			name:      "start of year",
			input:     "20230101",
			wantYear:  2023,
			wantMonth: time.January,
			wantDay:   1,
			wantPrec:  PrecisionDay,
			wantNEMA:  false,
		},
		{
			name:      "end of year",
			input:     "20231231",
			wantYear:  2023,
			wantMonth: time.December,
			wantDay:   31,
			wantPrec:  PrecisionDay,
			wantNEMA:  false,
		},
		{
			name:      "leap year date",
			input:     "20240229",
			wantYear:  2024,
			wantMonth: time.February,
			wantDay:   29,
			wantPrec:  PrecisionDay,
			wantNEMA:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantYear, date.Time.Year())
			assert.Equal(t, tt.wantMonth, date.Time.Month())
			assert.Equal(t, tt.wantDay, date.Time.Day())
			assert.Equal(t, tt.wantPrec, date.Precision)
			assert.Equal(t, tt.wantNEMA, date.IsNEMA)
		})
	}
}

// TestParseDate_PartialDate tests parsing dates with reduced precision.
func TestParseDate_PartialDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantPrec  PrecisionLevel
	}{
		{
			name:      "year and month only",
			input:     "202310",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   1, // default
			wantPrec:  PrecisionMonth,
		},
		{
			name:      "year only",
			input:     "2023",
			wantYear:  2023,
			wantMonth: time.January, // default
			wantDay:   1,            // default
			wantPrec:  PrecisionYear,
		},
		{
			name:      "year-month start of year",
			input:     "202301",
			wantYear:  2023,
			wantMonth: time.January,
			wantDay:   1,
			wantPrec:  PrecisionMonth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantYear, date.Time.Year())
			assert.Equal(t, tt.wantMonth, date.Time.Month())
			assert.Equal(t, tt.wantDay, date.Time.Day())
			assert.Equal(t, tt.wantPrec, date.Precision)
			assert.False(t, date.IsNEMA)
		})
	}
}

// TestParseDate_NEMAFormat tests parsing legacy NEMA-300 format (YYYY.MM.DD).
func TestParseDate_NEMAFormat(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
	}{
		{
			name:      "full NEMA date",
			input:     "2023.10.15",
			wantYear:  2023,
			wantMonth: time.October,
			wantDay:   15,
		},
		{
			name:      "NEMA with single digit month",
			input:     "2023.01.15",
			wantYear:  2023,
			wantMonth: time.January,
			wantDay:   15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantYear, date.Time.Year())
			assert.Equal(t, tt.wantMonth, date.Time.Month())
			assert.Equal(t, tt.wantDay, date.Time.Day())
			assert.Equal(t, PrecisionDay, date.Precision)
			assert.True(t, date.IsNEMA, "NEMA flag should be set")
		})
	}
}

// TestParseDate_Invalid tests error handling for invalid date strings.
func TestParseDate_Invalid(t *testing.T) {
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
			input:   "202",
			wantErr: "invalid format",
		},
		{
			name:    "non-numeric",
			input:   "20AB1015",
			wantErr: "invalid format",
		},
		{
			name:    "invalid month",
			input:   "20231315",
			wantErr: "month",
		},
		{
			name:    "invalid day for month",
			input:   "20230231", // February 31st
			wantErr: "day",
		},
		{
			name:    "invalid leap year",
			input:   "20230229", // 2023 is not a leap year
			wantErr: "day",
		},
		{
			name:    "month 00",
			input:   "20230015",
			wantErr: "month",
		},
		{
			name:    "day 00",
			input:   "20231000",
			wantErr: "day",
		},
		{
			name:    "NEMA invalid month",
			input:   "2023.13.15",
			wantErr: "month",
		},
		{
			name:    "NEMA invalid day",
			input:   "2023.02.31",
			wantErr: "day",
		},
		{
			name:    "NEMA month 00",
			input:   "2023.00.15",
			wantErr: "month",
		},
		{
			name:    "NEMA day 00",
			input:   "2023.10.00",
			wantErr: "day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDate(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestDate_DCM tests formatting Date back to DICOM string format.
func TestDate_DCM(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full date",
			input: "20231015",
			want:  "20231015",
		},
		{
			name:  "year-month",
			input: "202310",
			want:  "202310",
		},
		{
			name:  "year only",
			input: "2023",
			want:  "2023",
		},
		{
			name:  "NEMA format",
			input: "2023.10.15",
			want:  "2023.10.15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			require.NoError(t, err)

			got := date.DCM()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDate_String tests human-readable string formatting.
func TestDate_String(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full date",
			input: "20231015",
			want:  "2023-10-15",
		},
		{
			name:  "year-month",
			input: "202310",
			want:  "2023-10",
		},
		{
			name:  "year only",
			input: "2023",
			want:  "2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDate(tt.input)
			require.NoError(t, err)

			got := date.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDate_RoundTrip tests that parse → format → parse yields identical results.
func TestDate_RoundTrip(t *testing.T) {
	inputs := []string{
		"20231015",
		"202310",
		"2023",
		"2023.10.15",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			date1, err := ParseDate(input)
			require.NoError(t, err)

			formatted := date1.DCM()
			date2, err := ParseDate(formatted)
			require.NoError(t, err)

			assert.Equal(t, date1.Time.Year(), date2.Time.Year())
			assert.Equal(t, date1.Time.Month(), date2.Time.Month())
			assert.Equal(t, date1.Time.Day(), date2.Time.Day())
			assert.Equal(t, date1.Precision, date2.Precision)
			assert.Equal(t, date1.IsNEMA, date2.IsNEMA)
		})
	}
}
