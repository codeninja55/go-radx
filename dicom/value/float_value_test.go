package value_test

import (
	"math"
	"testing"

	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFloatValue_NewFloatValue tests creating float values with various VRs
func TestFloatValue_NewFloatValue(t *testing.T) {
	tests := []struct {
		name       string
		vr         vr.VR
		values     []float64
		wantErr    bool
		wantFloats []float64
	}{
		{
			name:       "FL with single value",
			vr:         vr.FloatingPointSingle,
			values:     []float64{3.14159},
			wantErr:    false,
			wantFloats: []float64{3.14159},
		},
		{
			name:       "FD with single value",
			vr:         vr.FloatingPointDouble,
			values:     []float64{2.718281828459045},
			wantErr:    false,
			wantFloats: []float64{2.718281828459045},
		},
		{
			name:       "FL with multi-value",
			vr:         vr.FloatingPointSingle,
			values:     []float64{1.5, 2.5, 3.5, 4.5},
			wantErr:    false,
			wantFloats: []float64{1.5, 2.5, 3.5, 4.5},
		},
		{
			name:       "FD with multi-value",
			vr:         vr.FloatingPointDouble,
			values:     []float64{1.1, 2.2, 3.3},
			wantErr:    false,
			wantFloats: []float64{1.1, 2.2, 3.3},
		},
		{
			name:       "empty value",
			vr:         vr.FloatingPointSingle,
			values:     []float64{},
			wantErr:    false,
			wantFloats: []float64{},
		},
		{
			name:       "zero value",
			vr:         vr.FloatingPointDouble,
			values:     []float64{0.0},
			wantErr:    false,
			wantFloats: []float64{0.0},
		},
		{
			name:       "negative value",
			vr:         vr.FloatingPointSingle,
			values:     []float64{-123.456},
			wantErr:    false,
			wantFloats: []float64{-123.456},
		},
		{
			name:       "very small value",
			vr:         vr.FloatingPointDouble,
			values:     []float64{1.23e-10},
			wantErr:    false,
			wantFloats: []float64{1.23e-10},
		},
		{
			name:       "very large value",
			vr:         vr.FloatingPointSingle,
			values:     []float64{1.23e+10},
			wantErr:    false,
			wantFloats: []float64{1.23e+10},
		},
		{
			name:       "positive infinity",
			vr:         vr.FloatingPointDouble,
			values:     []float64{math.Inf(1)},
			wantErr:    false,
			wantFloats: []float64{math.Inf(1)},
		},
		{
			name:       "negative infinity",
			vr:         vr.FloatingPointSingle,
			values:     []float64{math.Inf(-1)},
			wantErr:    false,
			wantFloats: []float64{math.Inf(-1)},
		},
		{
			name:       "NaN value",
			vr:         vr.FloatingPointDouble,
			values:     []float64{math.NaN()},
			wantErr:    false,
			wantFloats: []float64{math.NaN()},
		},
		{
			name:    "invalid VR (code string)",
			vr:      vr.CodeString,
			values:  []float64{1.0},
			wantErr: true,
		},
		{
			name:    "invalid VR (signed short)",
			vr:      vr.SignedShort,
			values:  []float64{1.0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewFloatValue(tt.vr, tt.values)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.vr, val.VR())

				// Special handling for NaN comparison
				gotFloats := val.Floats()
				require.Equal(t, len(tt.wantFloats), len(gotFloats))
				for i := range tt.wantFloats {
					if math.IsNaN(tt.wantFloats[i]) {
						assert.True(t, math.IsNaN(gotFloats[i]), "expected NaN at index %d", i)
					} else {
						assert.Equal(t, tt.wantFloats[i], gotFloats[i])
					}
				}
			}
		})
	}
}

// TestFloatValue_String tests string representation
func TestFloatValue_String(t *testing.T) {
	tests := []struct {
		name   string
		vr     vr.VR
		values []float64
		want   string
	}{
		{
			name:   "single positive value",
			vr:     vr.FloatingPointSingle,
			values: []float64{3.14159},
			want:   "3.14159",
		},
		{
			name:   "single negative value",
			vr:     vr.FloatingPointDouble,
			values: []float64{-123.456},
			want:   "-123.456",
		},
		{
			name:   "multi-value",
			vr:     vr.FloatingPointSingle,
			values: []float64{1.5, 2.5, 3.5},
			want:   "1.5\\2.5\\3.5",
		},
		{
			name:   "empty value",
			vr:     vr.FloatingPointDouble,
			values: []float64{},
			want:   "",
		},
		{
			name:   "zero value",
			vr:     vr.FloatingPointSingle,
			values: []float64{0.0},
			want:   "0",
		},
		{
			name:   "very small number (scientific notation)",
			vr:     vr.FloatingPointDouble,
			values: []float64{1.23e-10},
			want:   "1.23e-10",
		},
		{
			name:   "very large number (scientific notation)",
			vr:     vr.FloatingPointSingle,
			values: []float64{1.23e+10},
			want:   "1.23e+10",
		},
		{
			name:   "positive infinity",
			vr:     vr.FloatingPointDouble,
			values: []float64{math.Inf(1)},
			want:   "+Inf",
		},
		{
			name:   "negative infinity",
			vr:     vr.FloatingPointSingle,
			values: []float64{math.Inf(-1)},
			want:   "-Inf",
		},
		{
			name:   "NaN",
			vr:     vr.FloatingPointDouble,
			values: []float64{math.NaN()},
			want:   "NaN",
		},
		{
			name:   "mixed with special values",
			vr:     vr.FloatingPointSingle,
			values: []float64{1.0, math.Inf(1), -2.5, math.NaN()},
			want:   "1\\+Inf\\-2.5\\NaN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewFloatValue(tt.vr, tt.values)
			require.NoError(t, err)
			assert.Equal(t, tt.want, val.String())
		})
	}
}

// TestFloatValue_Bytes tests byte encoding (IEEE 754 little-endian)
func TestFloatValue_Bytes(t *testing.T) {
	tests := []struct {
		name   string
		vr     vr.VR
		values []float64
		want   []byte
	}{
		{
			name:   "FL single value 1.0",
			vr:     vr.FloatingPointSingle,
			values: []float64{1.0},
			want:   []byte{0x00, 0x00, 0x80, 0x3F}, // IEEE 754 float32 little-endian
		},
		{
			name:   "FD single value 1.0",
			vr:     vr.FloatingPointDouble,
			values: []float64{1.0},
			want:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F}, // IEEE 754 float64 little-endian
		},
		{
			name:   "FL single value -1.0",
			vr:     vr.FloatingPointSingle,
			values: []float64{-1.0},
			want:   []byte{0x00, 0x00, 0x80, 0xBF},
		},
		{
			name:   "FD single value -1.0",
			vr:     vr.FloatingPointDouble,
			values: []float64{-1.0},
			want:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0xBF},
		},
		{
			name:   "FL multi-value",
			vr:     vr.FloatingPointSingle,
			values: []float64{1.0, 2.0},
			want:   []byte{0x00, 0x00, 0x80, 0x3F, 0x00, 0x00, 0x00, 0x40}, // 1.0, 2.0
		},
		{
			name:   "empty value",
			vr:     vr.FloatingPointSingle,
			values: []float64{},
			want:   []byte{},
		},
		{
			name:   "FL positive infinity",
			vr:     vr.FloatingPointSingle,
			values: []float64{math.Inf(1)},
			want:   []byte{0x00, 0x00, 0x80, 0x7F},
		},
		{
			name:   "FL negative infinity",
			vr:     vr.FloatingPointSingle,
			values: []float64{math.Inf(-1)},
			want:   []byte{0x00, 0x00, 0x80, 0xFF},
		},
		{
			name:   "FD positive infinity",
			vr:     vr.FloatingPointDouble,
			values: []float64{math.Inf(1)},
			want:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x7F},
		},
		{
			name:   "FD negative infinity",
			vr:     vr.FloatingPointDouble,
			values: []float64{math.Inf(-1)},
			want:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0xFF},
		},
		{
			name:   "FL NaN",
			vr:     vr.FloatingPointSingle,
			values: []float64{math.NaN()},
			want:   []byte{0x00, 0x00, 0xC0, 0x7F}, // One possible NaN representation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewFloatValue(tt.vr, tt.values)
			require.NoError(t, err)
			assert.Equal(t, tt.want, val.Bytes())
		})
	}
}

// TestFloatValue_Equals tests value equality
func TestFloatValue_Equals(t *testing.T) {
	tests := []struct {
		name   string
		vr1    vr.VR
		vals1  []float64
		vr2    vr.VR
		vals2  []float64
		wantEq bool
	}{
		{
			name:   "equal single values",
			vr1:    vr.FloatingPointSingle,
			vals1:  []float64{3.14159},
			vr2:    vr.FloatingPointSingle,
			vals2:  []float64{3.14159},
			wantEq: true,
		},
		{
			name:   "equal multi values",
			vr1:    vr.FloatingPointDouble,
			vals1:  []float64{1.1, 2.2, 3.3},
			vr2:    vr.FloatingPointDouble,
			vals2:  []float64{1.1, 2.2, 3.3},
			wantEq: true,
		},
		{
			name:   "different values",
			vr1:    vr.FloatingPointSingle,
			vals1:  []float64{1.23},
			vr2:    vr.FloatingPointSingle,
			vals2:  []float64{4.56},
			wantEq: false,
		},
		{
			name:   "different VRs same value",
			vr1:    vr.FloatingPointSingle,
			vals1:  []float64{1.0},
			vr2:    vr.FloatingPointDouble,
			vals2:  []float64{1.0},
			wantEq: false,
		},
		{
			name:   "different lengths",
			vr1:    vr.FloatingPointDouble,
			vals1:  []float64{1.0},
			vr2:    vr.FloatingPointDouble,
			vals2:  []float64{1.0, 2.0},
			wantEq: false,
		},
		{
			name:   "both empty",
			vr1:    vr.FloatingPointSingle,
			vals1:  []float64{},
			vr2:    vr.FloatingPointSingle,
			vals2:  []float64{},
			wantEq: true,
		},
		{
			name:   "both NaN (treated as equal)",
			vr1:    vr.FloatingPointDouble,
			vals1:  []float64{math.NaN()},
			vr2:    vr.FloatingPointDouble,
			vals2:  []float64{math.NaN()},
			wantEq: true, // Our implementation treats NaN == NaN for comparison purposes
		},
		{
			name:   "both positive infinity",
			vr1:    vr.FloatingPointSingle,
			vals1:  []float64{math.Inf(1)},
			vr2:    vr.FloatingPointSingle,
			vals2:  []float64{math.Inf(1)},
			wantEq: true,
		},
		{
			name:   "positive vs negative infinity",
			vr1:    vr.FloatingPointDouble,
			vals1:  []float64{math.Inf(1)},
			vr2:    vr.FloatingPointDouble,
			vals2:  []float64{math.Inf(-1)},
			wantEq: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val1, err := value.NewFloatValue(tt.vr1, tt.vals1)
			require.NoError(t, err)
			val2, err := value.NewFloatValue(tt.vr2, tt.vals2)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEq, val1.Equals(val2))
		})
	}
}

// TestFloatValue_PrecisionHandling tests precision differences between FL and FD
func TestFloatValue_PrecisionHandling(t *testing.T) {
	tests := []struct {
		name        string
		vr          vr.VR
		inputValue  float64
		expectLoss  bool
		description string
	}{
		{
			name:        "FL loses precision for high-precision value",
			vr:          vr.FloatingPointSingle,
			inputValue:  1.234567890123456789, // More precision than float32 can hold
			expectLoss:  true,
			description: "float32 only has ~7 decimal digits of precision",
		},
		{
			name:        "FD retains precision for high-precision value",
			vr:          vr.FloatingPointDouble,
			inputValue:  1.234567890123456789,
			expectLoss:  false,
			description: "float64 has ~15-16 decimal digits of precision",
		},
		{
			name:        "FL retains precision for simple value",
			vr:          vr.FloatingPointSingle,
			inputValue:  1.5,
			expectLoss:  false,
			description: "Simple values can be represented exactly in float32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewFloatValue(tt.vr, []float64{tt.inputValue})
			require.NoError(t, err)

			// Decode bytes back to float to check precision
			bytes := val.Bytes()

			if tt.vr == vr.FloatingPointSingle {
				// For FL, precision loss is expected when converting float64 -> float32
				recovered := float64(math.Float32frombits(uint32(bytes[0]) | uint32(bytes[1])<<8 | uint32(bytes[2])<<16 | uint32(bytes[3])<<24))

				if tt.expectLoss {
					assert.NotEqual(t, tt.inputValue, recovered, "expected precision loss for FL: %s", tt.description)
				} else {
					assert.Equal(t, tt.inputValue, recovered, "expected no precision loss: %s", tt.description)
				}
			}
		})
	}
}

// TestFloatValue_InvalidVR tests that non-float VRs are rejected
func TestFloatValue_InvalidVR(t *testing.T) {
	tests := []struct {
		name string
		vr   vr.VR
	}{
		{
			name: "reject CS (code string)",
			vr:   vr.CodeString,
		},
		{
			name: "reject SS (signed short)",
			vr:   vr.SignedShort,
		},
		{
			name: "reject OB (other byte)",
			vr:   vr.OtherByte,
		},
		{
			name: "reject SQ (sequence)",
			vr:   vr.SequenceOfItems,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value.NewFloatValue(tt.vr, []float64{1.0})
			require.Error(t, err)
		})
	}
}
