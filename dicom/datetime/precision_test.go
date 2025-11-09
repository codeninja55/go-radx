package datetime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrecisionLevel_String(t *testing.T) {
	tests := []struct {
		precision PrecisionLevel
		want      string
	}{
		{PrecisionFull, "Full"},
		{PrecisionMS6, "Microsecond"},
		{PrecisionMS5, "MS5"},
		{PrecisionMS4, "MS4"},
		{PrecisionMS3, "Millisecond"},
		{PrecisionMS2, "Centisecond"},
		{PrecisionMS1, "Decisecond"},
		{PrecisionSeconds, "Second"},
		{PrecisionMinutes, "Minute"},
		{PrecisionHours, "Hour"},
		{PrecisionDay, "Day"},
		{PrecisionMonth, "Month"},
		{PrecisionYear, "Year"},
		{PrecisionLevel(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.precision.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAgeUnit_String(t *testing.T) {
	tests := []struct {
		unit AgeUnit
		want string
	}{
		{Days, "D"},
		{Weeks, "W"},
		{Months, "M"},
		{Years, "Y"},
		{AgeUnit(999), ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.unit.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAgeUnit_LongString(t *testing.T) {
	tests := []struct {
		unit AgeUnit
		want string
	}{
		{Days, "days"},
		{Weeks, "weeks"},
		{Months, "months"},
		{Years, "years"},
		{AgeUnit(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.unit.LongString()
			assert.Equal(t, tt.want, got)
		})
	}
}
