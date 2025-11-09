package value

import (
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom/datetime"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStringValue_AsDate tests parsing StringValue as Date (DA).
func TestStringValue_AsDate(t *testing.T) {
	tests := []struct {
		name      string
		vr        vr.VR
		values    []string
		wantYear  int
		wantMonth int
		wantDay   int
		wantErr   string
	}{
		{
			name:      "valid full date",
			vr:        vr.Date,
			values:    []string{"20231015"},
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   15,
		},
		{
			name:      "valid year-month",
			vr:        vr.Date,
			values:    []string{"202310"},
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   1, // defaults to 1
		},
		{
			name:      "valid year only",
			vr:        vr.Date,
			values:    []string{"2023"},
			wantYear:  2023,
			wantMonth: 1, // defaults to 1
			wantDay:   1, // defaults to 1
		},
		{
			name:    "wrong VR (TM)",
			vr:      vr.Time,
			values:  []string{"20231015"},
			wantErr: "expected DA",
		},
		{
			name:    "wrong VR (DT)",
			vr:      vr.DateTime,
			values:  []string{"20231015"},
			wantErr: "expected DA",
		},
		{
			name:    "empty value",
			vr:      vr.Date,
			values:  []string{},
			wantErr: "empty Date value",
		},
		{
			name:    "multiple values",
			vr:      vr.Date,
			values:  []string{"20231015", "20231016"},
			wantErr: "multiple values",
		},
		{
			name:    "invalid date format",
			vr:      vr.Date,
			values:  []string{"invalid"},
			wantErr: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := NewStringValue(tt.vr, tt.values)
			require.NoError(t, err)

			date, err := sv.AsDate()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantYear, date.Time.Year())
			assert.Equal(t, tt.wantMonth, int(date.Time.Month()))
			assert.Equal(t, tt.wantDay, date.Time.Day())
		})
	}
}

// TestStringValue_AsTime tests parsing StringValue as Time (TM).
func TestStringValue_AsTime(t *testing.T) {
	tests := []struct {
		name      string
		vr        vr.VR
		values    []string
		wantHour  int
		wantMin   int
		wantSec   int
		wantMicro int
		wantErr   string
	}{
		{
			name:      "valid full time with microseconds",
			vr:        vr.Time,
			values:    []string{"143025.123456"},
			wantHour:  14,
			wantMin:   30,
			wantSec:   25,
			wantMicro: 123456,
		},
		{
			name:      "valid time without fractional seconds",
			vr:        vr.Time,
			values:    []string{"143025"},
			wantHour:  14,
			wantMin:   30,
			wantSec:   25,
			wantMicro: 0,
		},
		{
			name:      "valid hours and minutes",
			vr:        vr.Time,
			values:    []string{"1430"},
			wantHour:  14,
			wantMin:   30,
			wantSec:   0,
			wantMicro: 0,
		},
		{
			name:      "valid hours only",
			vr:        vr.Time,
			values:    []string{"14"},
			wantHour:  14,
			wantMin:   0,
			wantSec:   0,
			wantMicro: 0,
		},
		{
			name:    "wrong VR (DA)",
			vr:      vr.Date,
			values:  []string{"143025"},
			wantErr: "expected TM",
		},
		{
			name:    "wrong VR (DT)",
			vr:      vr.DateTime,
			values:  []string{"143025"},
			wantErr: "expected TM",
		},
		{
			name:    "empty value",
			vr:      vr.Time,
			values:  []string{},
			wantErr: "empty Time value",
		},
		{
			name:    "multiple values",
			vr:      vr.Time,
			values:  []string{"143025", "150030"},
			wantErr: "multiple values",
		},
		{
			name:    "invalid time format",
			vr:      vr.Time,
			values:  []string{"invalid"},
			wantErr: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := NewStringValue(tt.vr, tt.values)
			require.NoError(t, err)

			tim, err := sv.AsTime()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantHour, tim.Time.Hour())
			assert.Equal(t, tt.wantMin, tim.Time.Minute())
			assert.Equal(t, tt.wantSec, tim.Time.Second())
			assert.Equal(t, tt.wantMicro, tim.Time.Nanosecond()/1000)
		})
	}
}

// TestStringValue_AsDateTime tests parsing StringValue as DateTime (DT).
func TestStringValue_AsDateTime(t *testing.T) {
	tests := []struct {
		name       string
		vr         vr.VR
		values     []string
		wantYear   int
		wantMonth  int
		wantDay    int
		wantHour   int
		wantOffset int // timezone offset in seconds
		wantErr    string
	}{
		{
			name:       "valid full datetime with timezone",
			vr:         vr.DateTime,
			values:     []string{"20231015143025+1000"},
			wantYear:   2023,
			wantMonth:  10,
			wantDay:    15,
			wantHour:   14,
			wantOffset: 10 * 3600, // +10 hours
		},
		{
			name:       "valid datetime without timezone",
			vr:         vr.DateTime,
			values:     []string{"20231015143025"},
			wantYear:   2023,
			wantMonth:  10,
			wantDay:    15,
			wantHour:   14,
			wantOffset: 0, // UTC
		},
		{
			name:      "valid date only",
			vr:        vr.DateTime,
			values:    []string{"20231015"},
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   15,
			wantHour:  0,
		},
		{
			name:      "valid year only",
			vr:        vr.DateTime,
			values:    []string{"2023"},
			wantYear:  2023,
			wantMonth: 1,
			wantDay:   1,
			wantHour:  0,
		},
		{
			name:    "wrong VR (DA)",
			vr:      vr.Date,
			values:  []string{"20231015"},
			wantErr: "expected DT",
		},
		{
			name:    "wrong VR (TM)",
			vr:      vr.Time,
			values:  []string{"143025"},
			wantErr: "expected DT",
		},
		{
			name:    "empty value",
			vr:      vr.DateTime,
			values:  []string{},
			wantErr: "empty DateTime value",
		},
		{
			name:    "multiple values",
			vr:      vr.DateTime,
			values:  []string{"20231015143025", "20231016143025"},
			wantErr: "multiple values",
		},
		{
			name:    "invalid datetime format",
			vr:      vr.DateTime,
			values:  []string{"invalid"},
			wantErr: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := NewStringValue(tt.vr, tt.values)
			require.NoError(t, err)

			dt, err := sv.AsDateTime()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantYear, dt.Time.Year())
			assert.Equal(t, tt.wantMonth, int(dt.Time.Month()))
			assert.Equal(t, tt.wantDay, dt.Time.Day())
			assert.Equal(t, tt.wantHour, dt.Time.Hour())

			if tt.wantOffset != 0 {
				_, offset := dt.Time.Zone()
				assert.Equal(t, tt.wantOffset, offset)
			}
		})
	}
}

// TestStringValue_AsAge tests parsing StringValue as Age String (AS).
func TestStringValue_AsAge(t *testing.T) {
	tests := []struct {
		name         string
		vr           vr.VR
		values       []string
		wantValue    int
		wantUnit     datetime.AgeUnit
		wantDuration time.Duration
		wantErr      string
	}{
		{
			name:         "valid days",
			vr:           vr.AgeString,
			values:       []string{"007D"},
			wantValue:    7,
			wantUnit:     datetime.Days,
			wantDuration: 7 * 24 * time.Hour,
		},
		{
			name:         "valid weeks",
			vr:           vr.AgeString,
			values:       []string{"004W"},
			wantValue:    4,
			wantUnit:     datetime.Weeks,
			wantDuration: 4 * 7 * 24 * time.Hour,
		},
		{
			name:         "valid months",
			vr:           vr.AgeString,
			values:       []string{"006M"},
			wantValue:    6,
			wantUnit:     datetime.Months,
			wantDuration: time.Duration(6 * 30.4375 * 24 * float64(time.Hour)),
		},
		{
			name:         "valid years",
			vr:           vr.AgeString,
			values:       []string{"042Y"},
			wantValue:    42,
			wantUnit:     datetime.Years,
			wantDuration: time.Duration(42 * 365.25 * 24 * float64(time.Hour)),
		},
		{
			name:      "zero age",
			vr:        vr.AgeString,
			values:    []string{"000D"},
			wantValue: 0,
			wantUnit:  datetime.Days,
		},
		{
			name:    "wrong VR (DA)",
			vr:      vr.Date,
			values:  []string{"042Y"},
			wantErr: "expected AS",
		},
		{
			name:    "wrong VR (TM)",
			vr:      vr.Time,
			values:  []string{"042Y"},
			wantErr: "expected AS",
		},
		{
			name:    "empty value",
			vr:      vr.AgeString,
			values:  []string{},
			wantErr: "empty Age value",
		},
		{
			name:    "multiple values",
			vr:      vr.AgeString,
			values:  []string{"042Y", "043Y"},
			wantErr: "multiple values",
		},
		{
			name:    "invalid age format",
			vr:      vr.AgeString,
			values:  []string{"ABCD"},
			wantErr: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := NewStringValue(tt.vr, tt.values)
			require.NoError(t, err)

			age, err := sv.AsAge()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, age.Value)
			assert.Equal(t, tt.wantUnit, age.Unit)

			if tt.wantDuration != 0 {
				assert.Equal(t, tt.wantDuration, age.Duration())
			}
		})
	}
}

// TestStringValue_TemporalRoundTrip tests that temporal values can be
// parsed from StringValue, converted back to DICOM format, and parsed again.
func TestStringValue_TemporalRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		vr     vr.VR
		value  string
		parser func(*StringValue) (string, error) // returns DCM() string
	}{
		{
			name:  "date round-trip",
			vr:    vr.Date,
			value: "20231015",
			parser: func(sv *StringValue) (string, error) {
				d, err := sv.AsDate()
				if err != nil {
					return "", err
				}
				return d.DCM(), nil
			},
		},
		{
			name:  "time round-trip",
			vr:    vr.Time,
			value: "143025.123456",
			parser: func(sv *StringValue) (string, error) {
				t, err := sv.AsTime()
				if err != nil {
					return "", err
				}
				return t.DCM(), nil
			},
		},
		{
			name:  "datetime round-trip",
			vr:    vr.DateTime,
			value: "20231015143025+1000",
			parser: func(sv *StringValue) (string, error) {
				dt, err := sv.AsDateTime()
				if err != nil {
					return "", err
				}
				return dt.DCM(), nil
			},
		},
		{
			name:  "age round-trip",
			vr:    vr.AgeString,
			value: "042Y",
			parser: func(sv *StringValue) (string, error) {
				a, err := sv.AsAge()
				if err != nil {
					return "", err
				}
				return a.DCM(), nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse original value
			sv1, err := NewStringValue(tt.vr, []string{tt.value})
			require.NoError(t, err)

			// Convert to temporal type and back to DCM format
			dcmStr, err := tt.parser(sv1)
			require.NoError(t, err)

			// Parse the DCM string again
			sv2, err := NewStringValue(tt.vr, []string{dcmStr})
			require.NoError(t, err)

			// Convert again and verify it matches
			dcmStr2, err := tt.parser(sv2)
			require.NoError(t, err)

			assert.Equal(t, tt.value, dcmStr, "first parse should match original")
			assert.Equal(t, tt.value, dcmStr2, "second parse should match original")
		})
	}
}
