package primitives

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDate_ValidFormats(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		precision string
	}{
		{
			name:      "year only",
			input:     "2024",
			wantErr:   false,
			precision: "year",
		},
		{
			name:      "year and month",
			input:     "2024-01",
			wantErr:   false,
			precision: "month",
		},
		{
			name:      "full date",
			input:     "2024-01-15",
			wantErr:   false,
			precision: "day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDate(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, d.String())
				assert.Equal(t, tt.precision, d.Precision())
			}
		})
	}
}

func TestNewDate_InvalidFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "invalid year",
			input: "202",
		},
		{
			name:  "invalid month",
			input: "2024-1",
		},
		{
			name:  "invalid day format",
			input: "2024-01-1",
		},
		{
			name:  "with time",
			input: "2024-01-15T10:30:00",
		},
		{
			name:  "invalid separator",
			input: "2024/01/15",
		},
		{
			name:  "extra precision",
			input: "2024-01-15-16",
		},
		{
			name:  "month only",
			input: "01",
		},
		{
			name:  "day only",
			input: "15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDate(tt.input)
			require.Error(t, err)
		})
	}
}

func TestMustDate_ValidInput(t *testing.T) {
	assert.NotPanics(t, func() {
		d := MustDate("2024-01-15")
		assert.Equal(t, "2024-01-15", d.String())
	})
}

func TestMustDate_InvalidInput(t *testing.T) {
	assert.Panics(t, func() {
		MustDate("invalid")
	})
}

func TestDate_Time(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantYear int
		wantMon  time.Month
		wantDay  int
	}{
		{
			name:     "full date",
			input:    "2024-01-15",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  15,
		},
		{
			name:     "year and month",
			input:    "2024-03",
			wantYear: 2024,
			wantMon:  time.March,
			wantDay:  1, // defaults to first day of month
		},
		{
			name:     "year only",
			input:    "2024",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  1, // defaults to January 1st
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := MustDate(tt.input)
			tm, err := d.Time()
			require.NoError(t, err)
			assert.Equal(t, tt.wantYear, tm.Year())
			assert.Equal(t, tt.wantMon, tm.Month())
			assert.Equal(t, tt.wantDay, tm.Day())
		})
	}
}

func TestDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON string
	}{
		{
			name:     "full date",
			input:    "2024-01-15",
			wantJSON: `"2024-01-15"`,
		},
		{
			name:     "year and month",
			input:    "2024-01",
			wantJSON: `"2024-01"`,
		},
		{
			name:     "year only",
			input:    "2024",
			wantJSON: `"2024"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := MustDate(tt.input)
			data, err := json.Marshal(d)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(data))
		})
	}
}

func TestDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "full date",
			input:   `"2024-01-15"`,
			want:    "2024-01-15",
			wantErr: false,
		},
		{
			name:    "year and month",
			input:   `"2024-01"`,
			want:    "2024-01",
			wantErr: false,
		},
		{
			name:    "year only",
			input:   `"2024"`,
			want:    "2024",
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   `"invalid"`,
			wantErr: true,
		},
		{
			name:    "not a string",
			input:   `123`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Date
			err := json.Unmarshal([]byte(tt.input), &d)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, d.String())
			}
		})
	}
}

func TestDate_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"full date", "2024-01-15"},
		{"year and month", "2024-01"},
		{"year only", "2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1 := MustDate(tt.input)

			// Marshal
			data, err := json.Marshal(d1)
			require.NoError(t, err)

			// Unmarshal
			var d2 Date
			err = json.Unmarshal(data, &d2)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, d1.String(), d2.String())
			assert.True(t, d1.Equal(d2))
		})
	}
}

func TestDate_IsZero(t *testing.T) {
	tests := []struct {
		name string
		date Date
		want bool
	}{
		{
			name: "zero value",
			date: Date{},
			want: true,
		},
		{
			name: "non-zero value",
			date: MustDate("2024"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.date.IsZero())
		})
	}
}

func TestDate_Equal(t *testing.T) {
	tests := []struct {
		name  string
		d1    Date
		d2    Date
		equal bool
	}{
		{
			name:  "same full dates",
			d1:    MustDate("2024-01-15"),
			d2:    MustDate("2024-01-15"),
			equal: true,
		},
		{
			name:  "different full dates",
			d1:    MustDate("2024-01-15"),
			d2:    MustDate("2024-01-16"),
			equal: false,
		},
		{
			name:  "same year-month",
			d1:    MustDate("2024-01"),
			d2:    MustDate("2024-01"),
			equal: true,
		},
		{
			name:  "different precision same data",
			d1:    MustDate("2024-01"),
			d2:    MustDate("2024-01-01"),
			equal: false,
		},
		{
			name:  "both zero",
			d1:    Date{},
			d2:    Date{},
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, tt.d1.Equal(tt.d2))
		})
	}
}

func TestFromTime(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("FromTime", func(t *testing.T) {
		d := FromTime(tm)
		assert.Equal(t, "2024-01-15", d.String())
		assert.Equal(t, "day", d.Precision())
	})

	t.Run("FromTimeYear", func(t *testing.T) {
		d := FromTimeYear(tm)
		assert.Equal(t, "2024", d.String())
		assert.Equal(t, "year", d.Precision())
	})

	t.Run("FromTimeMonth", func(t *testing.T) {
		d := FromTimeMonth(tm)
		assert.Equal(t, "2024-01", d.String())
		assert.Equal(t, "month", d.Precision())
	})
}

func TestDate_Precision(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		precision string
	}{
		{
			name:      "year precision",
			input:     "2024",
			precision: "year",
		},
		{
			name:      "month precision",
			input:     "2024-01",
			precision: "month",
		},
		{
			name:      "day precision",
			input:     "2024-01-15",
			precision: "day",
		},
		{
			name:      "zero value",
			input:     "",
			precision: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Date
			if tt.input != "" {
				d = MustDate(tt.input)
			}
			assert.Equal(t, tt.precision, d.Precision())
		})
	}
}
