package datetime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ParseError
		want string
	}{
		{
			name: "with input",
			err: &ParseError{
				VR:     "DA",
				Input:  "invalid",
				Reason: "wrong format",
			},
			want: `parse DA "invalid": wrong format`,
		},
		{
			name: "empty input",
			err: &ParseError{
				VR:     "TM",
				Input:  "",
				Reason: "empty",
			},
			want: "parse TM: empty input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatError_Error(t *testing.T) {
	err := &FormatError{
		VR:     "DA",
		Reason: "invalid precision",
	}
	want := "format DA: invalid precision"
	got := err.Error()
	assert.Equal(t, want, got)
}

func TestRangeError_Error(t *testing.T) {
	err := &RangeError{
		Component: "month",
		Value:     13,
		Min:       1,
		Max:       12,
	}
	want := "month 13 out of range [1, 12]"
	got := err.Error()
	assert.Equal(t, want, got)
}

func TestNewParseError(t *testing.T) {
	err := newParseError("DA", "20231301", "invalid month")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DA")
	assert.Contains(t, err.Error(), "20231301")
	assert.Contains(t, err.Error(), "invalid month")
}

func TestNewFormatError(t *testing.T) {
	err := newFormatError("TM", "unsupported precision")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TM")
	assert.Contains(t, err.Error(), "unsupported precision")
}

func TestNewRangeError(t *testing.T) {
	err := newRangeError("day", 32, 1, 31)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "day")
	assert.Contains(t, err.Error(), "32")
	assert.Contains(t, err.Error(), "[1, 31]")
}
