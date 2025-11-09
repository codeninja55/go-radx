package primitives

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTime_ValidFormats(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "time without fractional seconds",
			input:   "10:30:00",
			wantErr: false,
		},
		{
			name:    "time with fractional seconds",
			input:   "10:30:00.123",
			wantErr: false,
		},
		{
			name:    "time with many fractional digits",
			input:   "10:30:00.123456789",
			wantErr: false,
		},
		{
			name:    "midnight",
			input:   "00:00:00",
			wantErr: false,
		},
		{
			name:    "end of day",
			input:   "23:59:59",
			wantErr: false,
		},
		{
			name:    "noon",
			input:   "12:00:00",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := NewTime(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, tm.String())
			}
		})
	}
}

func TestNewTime_InvalidFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "invalid hour",
			input: "24:00:00",
		},
		{
			name:  "invalid hour format",
			input: "1:00:00",
		},
		{
			name:  "invalid minute",
			input: "10:60:00",
		},
		{
			name:  "invalid minute format",
			input: "10:5:00",
		},
		{
			name:  "invalid second",
			input: "10:30:60",
		},
		{
			name:  "invalid second format",
			input: "10:30:5",
		},
		{
			name:  "missing seconds",
			input: "10:30",
		},
		{
			name:  "with timezone",
			input: "10:30:00Z",
		},
		{
			name:  "with date",
			input: "2024-01-15T10:30:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTime(tt.input)
			require.Error(t, err)
		})
	}
}

func TestMustTime_ValidInput(t *testing.T) {
	assert.NotPanics(t, func() {
		tm := MustTime("10:30:00")
		assert.Equal(t, "10:30:00", tm.String())
	})
}

func TestMustTime_InvalidInput(t *testing.T) {
	assert.Panics(t, func() {
		MustTime("invalid")
	})
}

func TestTime_Duration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHour int
		wantMin  int
		wantSec  int
	}{
		{
			name:     "morning time",
			input:    "10:30:45",
			wantHour: 10,
			wantMin:  30,
			wantSec:  45,
		},
		{
			name:     "midnight",
			input:    "00:00:00",
			wantHour: 0,
			wantMin:  0,
			wantSec:  0,
		},
		{
			name:     "noon",
			input:    "12:00:00",
			wantHour: 12,
			wantMin:  0,
			wantSec:  0,
		},
		{
			name:     "end of day",
			input:    "23:59:59",
			wantHour: 23,
			wantMin:  59,
			wantSec:  59,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := MustTime(tt.input)
			d, err := tm.Duration()
			require.NoError(t, err)

			wantDuration := time.Duration(tt.wantHour)*time.Hour +
				time.Duration(tt.wantMin)*time.Minute +
				time.Duration(tt.wantSec)*time.Second

			assert.Equal(t, wantDuration, d)
		})
	}
}

func TestTime_TimeOfDay(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		wantHour int
		wantMin  int
		wantSec  int
	}{
		{
			name:     "morning time",
			input:    "10:30:45",
			wantHour: 10,
			wantMin:  30,
			wantSec:  45,
		},
		{
			name:     "midnight",
			input:    "00:00:00",
			wantHour: 0,
			wantMin:  0,
			wantSec:  0,
		},
		{
			name:     "noon",
			input:    "12:00:00",
			wantHour: 12,
			wantMin:  0,
			wantSec:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := MustTime(tt.input)
			result, err := tm.TimeOfDay(date)
			require.NoError(t, err)

			assert.Equal(t, 2024, result.Year())
			assert.Equal(t, time.January, result.Month())
			assert.Equal(t, 15, result.Day())
			assert.Equal(t, tt.wantHour, result.Hour())
			assert.Equal(t, tt.wantMin, result.Minute())
			assert.Equal(t, tt.wantSec, result.Second())
		})
	}
}

func TestTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON string
	}{
		{
			name:     "time without fractional seconds",
			input:    "10:30:00",
			wantJSON: `"10:30:00"`,
		},
		{
			name:     "time with fractional seconds",
			input:    "10:30:00.123",
			wantJSON: `"10:30:00.123"`,
		},
		{
			name:     "midnight",
			input:    "00:00:00",
			wantJSON: `"00:00:00"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := MustTime(tt.input)
			data, err := json.Marshal(tm)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(data))
		})
	}
}

func TestTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "time without fractional seconds",
			input:   `"10:30:00"`,
			want:    "10:30:00",
			wantErr: false,
		},
		{
			name:    "time with fractional seconds",
			input:   `"10:30:00.123"`,
			want:    "10:30:00.123",
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
			var tm Time
			err := json.Unmarshal([]byte(tt.input), &tm)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, tm.String())
			}
		})
	}
}

func TestTime_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"time without fractional seconds", "10:30:00"},
		{"time with fractional seconds", "10:30:00.123456"},
		{"midnight", "00:00:00"},
		{"end of day", "23:59:59"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm1 := MustTime(tt.input)

			// Marshal
			data, err := json.Marshal(tm1)
			require.NoError(t, err)

			// Unmarshal
			var tm2 Time
			err = json.Unmarshal(data, &tm2)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, tm1.String(), tm2.String())
			assert.True(t, tm1.Equal(tm2))
		})
	}
}

func TestTime_IsZero(t *testing.T) {
	tests := []struct {
		name string
		time Time
		want bool
	}{
		{
			name: "zero value",
			time: Time{},
			want: true,
		},
		{
			name: "non-zero value",
			time: MustTime("10:30:00"),
			want: false,
		},
		{
			name: "midnight is not zero",
			time: MustTime("00:00:00"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.time.IsZero())
		})
	}
}

func TestTime_Equal(t *testing.T) {
	tests := []struct {
		name  string
		t1    Time
		t2    Time
		equal bool
	}{
		{
			name:  "same time",
			t1:    MustTime("10:30:00"),
			t2:    MustTime("10:30:00"),
			equal: true,
		},
		{
			name:  "different time",
			t1:    MustTime("10:30:00"),
			t2:    MustTime("10:30:01"),
			equal: false,
		},
		{
			name:  "same with fractional seconds",
			t1:    MustTime("10:30:00.123"),
			t2:    MustTime("10:30:00.123"),
			equal: true,
		},
		{
			name:  "different fractional seconds",
			t1:    MustTime("10:30:00.123"),
			t2:    MustTime("10:30:00.124"),
			equal: false,
		},
		{
			name:  "both zero",
			t1:    Time{},
			t2:    Time{},
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, tt.t1.Equal(tt.t2))
		})
	}
}

func TestFromDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "morning time",
			duration: 10*time.Hour + 30*time.Minute + 45*time.Second,
			want:     "10:30:45",
		},
		{
			name:     "midnight",
			duration: 0,
			want:     "00:00:00",
		},
		{
			name:     "with milliseconds",
			duration: 10*time.Hour + 30*time.Minute + 45*time.Second + 123*time.Millisecond,
			want:     "10:30:45.123000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := FromDuration(tt.duration)
			assert.Equal(t, tt.want, tm.String())
		})
	}
}

func TestFromTimeTime(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "without fractional seconds",
			time: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			want: "10:30:45",
		},
		{
			name: "with nanoseconds",
			time: time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC),
			want: "10:30:45.123",
		},
		{
			name: "midnight",
			time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			want: "00:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := FromTimeTime(tt.time)
			assert.Equal(t, tt.want, tm.String())
		})
	}
}
