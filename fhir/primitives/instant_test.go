package primitives

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstant_ValidFormats(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "instant with Z timezone",
			input:   "2024-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "instant with positive timezone",
			input:   "2024-01-15T10:30:00+10:00",
			wantErr: false,
		},
		{
			name:    "instant with negative timezone",
			input:   "2024-01-15T10:30:00-05:00",
			wantErr: false,
		},
		{
			name:    "instant with fractional seconds",
			input:   "2024-01-15T10:30:00.123Z",
			wantErr: false,
		},
		{
			name:    "instant with many fractional digits",
			input:   "2024-01-15T10:30:00.123456789Z",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInstant(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, i.String())
			}
		})
	}
}

func TestNewInstant_InvalidFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "missing timezone",
			input: "2024-01-15T10:30:00",
		},
		{
			name:  "date only",
			input: "2024-01-15",
		},
		{
			name:  "invalid timezone format",
			input: "2024-01-15T10:30:00+10",
		},
		{
			name:  "invalid separator",
			input: "2024-01-15 10:30:00Z",
		},
		{
			name:  "missing time",
			input: "2024-01-15TZ",
		},
		{
			name:  "invalid time format",
			input: "2024-01-15T10:30Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewInstant(tt.input)
			require.Error(t, err)
		})
	}
}

func TestMustInstant_ValidInput(t *testing.T) {
	assert.NotPanics(t, func() {
		i := MustInstant("2024-01-15T10:30:00Z")
		assert.Equal(t, "2024-01-15T10:30:00Z", i.String())
	})
}

func TestMustInstant_InvalidInput(t *testing.T) {
	assert.Panics(t, func() {
		MustInstant("invalid")
	})
}

func TestInstant_Time(t *testing.T) {
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
			name:     "instant with Z timezone",
			input:    "2024-01-15T10:30:45Z",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  15,
			wantHour: 10,
			wantMin:  30,
			wantSec:  45,
		},
		{
			name:     "instant with positive timezone",
			input:    "2024-01-15T10:30:45+10:00",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  15,
			wantHour: 10,
			wantMin:  30,
			wantSec:  45,
		},
		{
			name:     "instant with negative timezone",
			input:    "2024-01-15T10:30:45-05:00",
			wantYear: 2024,
			wantMon:  time.January,
			wantDay:  15,
			wantHour: 10,
			wantMin:  30,
			wantSec:  45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := MustInstant(tt.input)
			tm, err := i.Time()
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

func TestInstant_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON string
	}{
		{
			name:     "instant with Z timezone",
			input:    "2024-01-15T10:30:00Z",
			wantJSON: `"2024-01-15T10:30:00Z"`,
		},
		{
			name:     "instant with fractional seconds",
			input:    "2024-01-15T10:30:00.123Z",
			wantJSON: `"2024-01-15T10:30:00.123Z"`,
		},
		{
			name:     "instant with positive timezone",
			input:    "2024-01-15T10:30:00+10:00",
			wantJSON: `"2024-01-15T10:30:00+10:00"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := MustInstant(tt.input)
			data, err := json.Marshal(i)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(data))
		})
	}
}

func TestInstant_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "instant with Z timezone",
			input:   `"2024-01-15T10:30:00Z"`,
			want:    "2024-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "instant with fractional seconds",
			input:   `"2024-01-15T10:30:00.123Z"`,
			want:    "2024-01-15T10:30:00.123Z",
			wantErr: false,
		},
		{
			name:    "instant with positive timezone",
			input:   `"2024-01-15T10:30:00+10:00"`,
			want:    "2024-01-15T10:30:00+10:00",
			wantErr: false,
		},
		{
			name:    "invalid format - missing timezone",
			input:   `"2024-01-15T10:30:00"`,
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
			var i Instant
			err := json.Unmarshal([]byte(tt.input), &i)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, i.String())
			}
		})
	}
}

func TestInstant_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"instant with Z timezone", "2024-01-15T10:30:00Z"},
		{"instant with fractional seconds", "2024-01-15T10:30:00.123456Z"},
		{"instant with positive timezone", "2024-01-15T10:30:00+10:00"},
		{"instant with negative timezone", "2024-01-15T10:30:00-05:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i1 := MustInstant(tt.input)

			// Marshal
			data, err := json.Marshal(i1)
			require.NoError(t, err)

			// Unmarshal
			var i2 Instant
			err = json.Unmarshal(data, &i2)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, i1.String(), i2.String())
			assert.True(t, i1.Equal(i2))
		})
	}
}

func TestInstant_IsZero(t *testing.T) {
	tests := []struct {
		name    string
		instant Instant
		want    bool
	}{
		{
			name:    "zero value",
			instant: Instant{},
			want:    true,
		},
		{
			name:    "non-zero value",
			instant: MustInstant("2024-01-15T10:30:00Z"),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.instant.IsZero())
		})
	}
}

func TestInstant_Equal(t *testing.T) {
	tests := []struct {
		name  string
		i1    Instant
		i2    Instant
		equal bool
	}{
		{
			name:  "same instant",
			i1:    MustInstant("2024-01-15T10:30:00Z"),
			i2:    MustInstant("2024-01-15T10:30:00Z"),
			equal: true,
		},
		{
			name:  "different instant",
			i1:    MustInstant("2024-01-15T10:30:00Z"),
			i2:    MustInstant("2024-01-15T10:30:01Z"),
			equal: false,
		},
		{
			name:  "same instant different timezone representation",
			i1:    MustInstant("2024-01-15T10:30:00Z"),
			i2:    MustInstant("2024-01-15T10:30:00+00:00"),
			equal: false, // String comparison, not time comparison
		},
		{
			name:  "both zero",
			i1:    Instant{},
			i2:    Instant{},
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, tt.i1.Equal(tt.i2))
		})
	}
}

func TestFromTimeInstant(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	i := FromTimeInstant(tm)
	assert.Equal(t, "2024-01-15T10:30:45Z", i.String())

	// Verify it can be converted back
	result, err := i.Time()
	require.NoError(t, err)
	assert.True(t, tm.Equal(result))
}

func TestFromTimeInstantNano(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{
			name: "without nanoseconds",
			time: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			want: "2024-01-15T10:30:45Z",
		},
		{
			name: "with nanoseconds",
			time: time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC),
			want: "2024-01-15T10:30:45.123Z",
		},
		{
			name: "with many nanosecond digits",
			time: time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC),
			want: "2024-01-15T10:30:45.123456789Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := FromTimeInstantNano(tt.time)
			assert.Equal(t, tt.want, i.String())
		})
	}
}

func TestInstant_TimezonePreservation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "UTC timezone",
			input: "2024-01-15T10:30:00Z",
		},
		{
			name:  "positive timezone",
			input: "2024-01-15T10:30:00+10:00",
		},
		{
			name:  "negative timezone",
			input: "2024-01-15T10:30:00-05:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := MustInstant(tt.input)
			tm, err := i.Time()
			require.NoError(t, err)

			// Verify the time can be parsed and represents a valid instant
			assert.NotZero(t, tm)

			// Verify string representation is preserved
			assert.Equal(t, tt.input, i.String())
		})
	}
}
