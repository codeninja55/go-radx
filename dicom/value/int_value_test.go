package value_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntValue_NewIntValue tests creating int values with various VRs
func TestIntValue_NewIntValue(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		values   []int64
		wantErr  bool
		wantInts []int64
	}{
		{
			name:     "SS with single value",
			vr:       vr.SignedShort,
			values:   []int64{123},
			wantErr:  false,
			wantInts: []int64{123},
		},
		{
			name:     "US with single value",
			vr:       vr.UnsignedShort,
			values:   []int64{65535},
			wantErr:  false,
			wantInts: []int64{65535},
		},
		{
			name:     "SL with single value",
			vr:       vr.SignedLong,
			values:   []int64{-123456},
			wantErr:  false,
			wantInts: []int64{-123456},
		},
		{
			name:     "UL with single value",
			vr:       vr.UnsignedLong,
			values:   []int64{4294967295},
			wantErr:  false,
			wantInts: []int64{4294967295},
		},
		{
			name:     "SV with single value",
			vr:       vr.SignedVeryLong,
			values:   []int64{-9223372036854775808},
			wantErr:  false,
			wantInts: []int64{-9223372036854775808},
		},
		{
			name:     "UV with single value",
			vr:       vr.UnsignedVeryLong,
			values:   []int64{9223372036854775807},
			wantErr:  false,
			wantInts: []int64{9223372036854775807},
		},
		{
			name:     "AT with tag value",
			vr:       vr.AttributeTag,
			values:   []int64{0x00080018}, // (0008,0018)
			wantErr:  false,
			wantInts: []int64{0x00080018},
		},
		{
			name:     "multi-value SS",
			vr:       vr.SignedShort,
			values:   []int64{1, 2, 3, 4},
			wantErr:  false,
			wantInts: []int64{1, 2, 3, 4},
		},
		{
			name:     "empty value",
			vr:       vr.SignedShort,
			values:   []int64{},
			wantErr:  false,
			wantInts: []int64{},
		},
		{
			name:     "zero value",
			vr:       vr.UnsignedLong,
			values:   []int64{0},
			wantErr:  false,
			wantInts: []int64{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewIntValue(tt.vr, tt.values)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.vr, val.VR())
				assert.Equal(t, tt.wantInts, val.Ints())
			}
		})
	}
}

// TestIntValue_String tests string representation
func TestIntValue_String(t *testing.T) {
	tests := []struct {
		name   string
		vr     vr.VR
		values []int64
		want   string
	}{
		{
			name:   "single positive value",
			vr:     vr.SignedShort,
			values: []int64{123},
			want:   "123",
		},
		{
			name:   "single negative value",
			vr:     vr.SignedLong,
			values: []int64{-456},
			want:   "-456",
		},
		{
			name:   "multi-value",
			vr:     vr.UnsignedShort,
			values: []int64{1, 2, 3},
			want:   "1\\2\\3",
		},
		{
			name:   "empty value",
			vr:     vr.SignedShort,
			values: []int64{},
			want:   "",
		},
		{
			name:   "zero value",
			vr:     vr.UnsignedLong,
			values: []int64{0},
			want:   "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewIntValue(tt.vr, tt.values)
			require.NoError(t, err)
			assert.Equal(t, tt.want, val.String())
		})
	}
}

// TestIntValue_Bytes tests byte encoding
func TestIntValue_Bytes(t *testing.T) {
	tests := []struct {
		name   string
		vr     vr.VR
		values []int64
		want   []byte
	}{
		{
			name:   "SS single value little-endian",
			vr:     vr.SignedShort,
			values: []int64{256}, // 0x0100
			want:   []byte{0x00, 0x01},
		},
		{
			name:   "US single value",
			vr:     vr.UnsignedShort,
			values: []int64{1},
			want:   []byte{0x01, 0x00},
		},
		{
			name:   "SL single value",
			vr:     vr.SignedLong,
			values: []int64{16909060}, // 0x01020304
			want:   []byte{0x04, 0x03, 0x02, 0x01},
		},
		{
			name:   "UL single value",
			vr:     vr.UnsignedLong,
			values: []int64{1},
			want:   []byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			name:   "SV single value",
			vr:     vr.SignedVeryLong,
			values: []int64{72623859790382856}, // 0x0102030405060708
			want:   []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
		},
		{
			name:   "AT tag value",
			vr:     vr.AttributeTag,
			values: []int64{0x00080018}, // (0008,0018)
			want:   []byte{0x08, 0x00, 0x18, 0x00},
		},
		{
			name:   "multi-value SS",
			vr:     vr.SignedShort,
			values: []int64{1, 2},
			want:   []byte{0x01, 0x00, 0x02, 0x00},
		},
		{
			name:   "empty value",
			vr:     vr.SignedShort,
			values: []int64{},
			want:   []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewIntValue(tt.vr, tt.values)
			require.NoError(t, err)
			assert.Equal(t, tt.want, val.Bytes())
		})
	}
}

// TestIntValue_Equals tests value equality
func TestIntValue_Equals(t *testing.T) {
	tests := []struct {
		name   string
		vr1    vr.VR
		vals1  []int64
		vr2    vr.VR
		vals2  []int64
		wantEq bool
	}{
		{
			name:   "equal single values",
			vr1:    vr.SignedShort,
			vals1:  []int64{123},
			vr2:    vr.SignedShort,
			vals2:  []int64{123},
			wantEq: true,
		},
		{
			name:   "equal multi values",
			vr1:    vr.UnsignedShort,
			vals1:  []int64{1, 2, 3},
			vr2:    vr.UnsignedShort,
			vals2:  []int64{1, 2, 3},
			wantEq: true,
		},
		{
			name:   "different values",
			vr1:    vr.SignedShort,
			vals1:  []int64{123},
			vr2:    vr.SignedShort,
			vals2:  []int64{456},
			wantEq: false,
		},
		{
			name:   "different VRs same value",
			vr1:    vr.SignedShort,
			vals1:  []int64{123},
			vr2:    vr.UnsignedShort,
			vals2:  []int64{123},
			wantEq: false,
		},
		{
			name:   "different lengths",
			vr1:    vr.SignedShort,
			vals1:  []int64{123},
			vr2:    vr.SignedShort,
			vals2:  []int64{123, 456},
			wantEq: false,
		},
		{
			name:   "both empty",
			vr1:    vr.SignedShort,
			vals1:  []int64{},
			vr2:    vr.SignedShort,
			vals2:  []int64{},
			wantEq: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val1, err := value.NewIntValue(tt.vr1, tt.vals1)
			require.NoError(t, err)
			val2, err := value.NewIntValue(tt.vr2, tt.vals2)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEq, val1.Equals(val2))
		})
	}
}

// TestIntValue_RangeValidation tests that values are within valid ranges for their VR
func TestIntValue_RangeValidation(t *testing.T) {
	tests := []struct {
		name    string
		vr      vr.VR
		value   int64
		wantErr bool
	}{
		{
			name:    "SS within range positive",
			vr:      vr.SignedShort,
			value:   32767, // max int16
			wantErr: false,
		},
		{
			name:    "SS within range negative",
			vr:      vr.SignedShort,
			value:   -32768, // min int16
			wantErr: false,
		},
		{
			name:    "SS exceeds max",
			vr:      vr.SignedShort,
			value:   32768,
			wantErr: true,
		},
		{
			name:    "SS exceeds min",
			vr:      vr.SignedShort,
			value:   -32769,
			wantErr: true,
		},
		{
			name:    "US within range",
			vr:      vr.UnsignedShort,
			value:   65535, // max uint16
			wantErr: false,
		},
		{
			name:    "US negative not allowed",
			vr:      vr.UnsignedShort,
			value:   -1,
			wantErr: true,
		},
		{
			name:    "US exceeds max",
			vr:      vr.UnsignedShort,
			value:   65536,
			wantErr: true,
		},
		{
			name:    "SL within range",
			vr:      vr.SignedLong,
			value:   2147483647, // max int32
			wantErr: false,
		},
		{
			name:    "SL within range negative",
			vr:      vr.SignedLong,
			value:   -2147483648, // min int32
			wantErr: false,
		},
		{
			name:    "UL within range",
			vr:      vr.UnsignedLong,
			value:   4294967295, // max uint32
			wantErr: false,
		},
		{
			name:    "UL negative not allowed",
			vr:      vr.UnsignedLong,
			value:   -1,
			wantErr: true,
		},
		{
			name:    "AT within range",
			vr:      vr.AttributeTag,
			value:   0xFFFFFFFF, // max uint32
			wantErr: false,
		},
		{
			name:    "AT negative not allowed",
			vr:      vr.AttributeTag,
			value:   -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value.NewIntValue(tt.vr, []int64{tt.value})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestIntValue_InvalidVR tests that non-int VRs are rejected
func TestIntValue_InvalidVR(t *testing.T) {
	tests := []struct {
		name string
		vr   vr.VR
	}{
		{
			name: "reject CS (code string)",
			vr:   vr.CodeString,
		},
		{
			name: "reject FD (float double)",
			vr:   vr.FloatingPointDouble,
		},
		{
			name: "reject SQ (sequence)",
			vr:   vr.SequenceOfItems,
		},
		{
			name: "reject OB (other byte)",
			vr:   vr.OtherByte,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value.NewIntValue(tt.vr, []int64{123})
			require.Error(t, err)
		})
	}
}
