package primitives

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDateTime_ValidFormats(t *testing.T) {
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
		{
			name:      "datetime with seconds",
			input:     "2024-01-15T10:30:00",
			wantErr:   false,
			precision: "second",
		},
		{
			name:      "datetime with seconds and timezone Z",
			input:     "2024-01-15T10:30:00Z",
			wantErr:   false,
			precision: "second",
		},
		{
			name:      "datetime with seconds and positive timezone",
			input:     "2024-01-15T10:30:00+10:00",
			wantErr:   false,
			precision: "second",
		},
		{
			name:      "datetime with seconds and negative timezone",
			input:     "2024-01-15T10:30:00-05:00",
			wantErr:   false,
			precision: "second",
		},
		{
			name:      "datetime with fractional seconds",
			input:     "2024-01-15T10:30:00.123",
			wantErr:   false,
			precision: "second",
		},
		{
			name:      "datetime with fractional seconds and timezone",
			input:     "2024-01-15T10:30:00.123456Z",
			wantErr:   false,
			precision: "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := NewDateTime(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, dt.String())
				assert.Equal(t, tt.precision, dt.Precision())
			}
		})
	}
}

func TestNewDateTime_InvalidFormats(t *testing.T) {
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
			name:  "invalid separator",
			input: "2024/01/15",
		},
		{
			name:  "datetime without T separator",
			input: "2024-01-15 10:30:00",
		},
		{
			name:  "invalid time format",
			input: "2024-01-15T10:30",
		},
		{
			name:  "invalid timezone format",
			input: "2024-01-15T10:30:00+10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDateTime(tt.input)
			require.Error(t, err)
		})
	}
}

func TestMustDateTime_ValidInput(t *testing.T) {
	assert.NotPanics(t, func() {
		dt := MustDateTime("2024-01-15T10:30:00Z")
		assert.Equal(t, "2024-01-15T10:30:00Z", dt.String())
	})
}

func TestMustDateTime_InvalidInput(t *testing.T) {
	assert.Panics(t, func() {
		MustDateTime("invalid")
	})
}

func TestDateTime_Time(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantYear int
		wantMon  time.Month
		wantDay  int
		wantHour int
		wantMin  int
		wantSec  int
	}{
		{
			name:     "full datetime with timezone",
			input:    "2024-01-15T10:30:45Z",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  15,
			wantHour: 10,
			wantMin:  30,
			wantSec:  45,
		},
		{
			name:     "datetime without timezone",
			input:    "2024-01-15T10:30:45",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  15,
			wantHour: 10,
			wantMin:  30,
			wantSec:  45,
		},
		{
			name:     "full date only",
			input:    "2024-01-15",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  15,
			wantHour: 0,
			wantMin:  0,
			wantSec:  0,
		},
		{
			name:     "year and month",
			input:    "2024-03",
			wantYear: 2024,
			wantMon:  time.March,
			wantDay:  1,
			wantHour: 0,
			wantMin:  0,
			wantSec:  0,
		},
		{
			name:     "year only",
			input:    "2024",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  1,
			wantHour: 0,
			wantMin:  0,
			wantSec:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := MustDateTime(tt.input)
			tm, err := dt.Time()
			require.NoError(t, err)
			assert.Equal(t, tt.wantYear, tm.Year())
			assert.Equal(t, tt.wantMon, tm.Month())
			assert.Equal(t, tt.wantDay, tm.Day())
			assert.Equal(t, tt.wantHour, tm.Hour())
			assert.Equal(t, tt.wantMin, tm.Minute())
			assert.Equal(t, tt.wantSec, tm.Second())
		})
	}
}

func TestDateTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON string
	}{
		{
			name:     "full datetime",
			input:    "2024-01-15T10:30:00Z",
			wantJSON: `"2024-01-15T10:30:00Z"`,
		},
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
			dt := MustDateTime(tt.input)
			data, err := json.Marshal(dt)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(data))
		})
	}
}

func TestDateTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "full datetime",
			input:   `"2024-01-15T10:30:00Z"`,
			want:    "2024-01-15T10:30:00Z",
			wantErr: false,
		},
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
			var dt DateTime
			err := json.Unmarshal([]byte(tt.input), &dt)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, dt.String())
			}
		})
	}
}

func TestDateTime_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"full datetime", "2024-01-15T10:30:00Z"},
		{"datetime with fractional seconds", "2024-01-15T10:30:00.123Z"},
		{"full date", "2024-01-15"},
		{"year and month", "2024-01"},
		{"year only", "2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt1 := MustDateTime(tt.input)

			// Marshal
			data, err := json.Marshal(dt1)
			require.NoError(t, err)

			// Unmarshal
			var dt2 DateTime
			err = json.Unmarshal(data, &dt2)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, dt1.String(), dt2.String())
			assert.True(t, dt1.Equal(dt2))
		})
	}
}

func TestDateTime_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		datetime DateTime
		want     bool
	}{
		{
			name:     "zero value",
			datetime: DateTime{},
			want:     true,
		},
		{
			name:     "non-zero value",
			datetime: MustDateTime("2024"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.datetime.IsZero())
		})
	}
}

func TestDateTime_Equal(t *testing.T) {
	tests := []struct {
		name  string
		dt1   DateTime
		dt2   DateTime
		equal bool
	}{
		{
			name:  "same full datetime",
			dt1:   MustDateTime("2024-01-15T10:30:00Z"),
			dt2:   MustDateTime("2024-01-15T10:30:00Z"),
			equal: true,
		},
		{
			name:  "different datetime",
			dt1:   MustDateTime("2024-01-15T10:30:00Z"),
			dt2:   MustDateTime("2024-01-15T10:30:01Z"),
			equal: false,
		},
		{
			name:  "same year-month",
			dt1:   MustDateTime("2024-01"),
			dt2:   MustDateTime("2024-01"),
			equal: true,
		},
		{
			name:  "different precision same date",
			dt1:   MustDateTime("2024-01"),
			dt2:   MustDateTime("2024-01-01"),
			equal: false,
		},
		{
			name:  "both zero",
			dt1:   DateTime{},
			dt2:   DateTime{},
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, tt.dt1.Equal(tt.dt2))
		})
	}
}

func TestFromTimeDateTime(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	t.Run("FromTimeDateTime", func(t *testing.T) {
		dt := FromTimeDateTime(tm)
		assert.Equal(t, "2024-01-15T10:30:45Z", dt.String())
		assert.Equal(t, "second", dt.Precision())
	})

	t.Run("FromTimeDateTimeDate", func(t *testing.T) {
		dt := FromTimeDateTimeDate(tm)
		assert.Equal(t, "2024-01-15", dt.String())
		assert.Equal(t, "day", dt.Precision())
	})

	t.Run("FromTimeDateTimeYear", func(t *testing.T) {
		dt := FromTimeDateTimeYear(tm)
		assert.Equal(t, "2024", dt.String())
		assert.Equal(t, "year", dt.Precision())
	})

	t.Run("FromTimeDateTimeMonth", func(t *testing.T) {
		dt := FromTimeDateTimeMonth(tm)
		assert.Equal(t, "2024-01", dt.String())
		assert.Equal(t, "month", dt.Precision())
	})
}

func TestDateTime_Precision(t *testing.T) {
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
			name:      "second precision",
			input:     "2024-01-15T10:30:00Z",
			precision: "second",
		},
		{
			name:      "zero value",
			input:     "",
			precision: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dt DateTime
			if tt.input != "" {
				dt = MustDateTime(tt.input)
			}
			assert.Equal(t, tt.precision, dt.Precision())
		})
	}
}

func TestDateTime_TimezoneHandling(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "UTC timezone",
			input:     "2024-01-15T10:30:00Z",
			wantError: false,
		},
		{
			name:      "positive offset",
			input:     "2024-01-15T10:30:00+10:00",
			wantError: false,
		},
		{
			name:      "negative offset",
			input:     "2024-01-15T10:30:00-05:00",
			wantError: false,
		},
		{
			name:      "no timezone",
			input:     "2024-01-15T10:30:00",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := MustDateTime(tt.input)
			tm, err := dt.Time()
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, tm)
			}
		})
	}
}
