package value_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBytesValue_NewBytesValue tests creating bytes values with various VRs
func TestBytesValue_NewBytesValue(t *testing.T) {
	tests := []struct {
		name      string
		vr        vr.VR
		data      []byte
		wantErr   bool
		wantBytes []byte
	}{
		{
			name:      "OB with binary data",
			vr:        vr.OtherByte,
			data:      []byte{0x01, 0x02, 0x03, 0x04},
			wantErr:   false,
			wantBytes: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:      "OW with binary data",
			vr:        vr.OtherWord,
			data:      []byte{0xFF, 0xFE, 0x00, 0xE0},
			wantErr:   false,
			wantBytes: []byte{0xFF, 0xFE, 0x00, 0xE0},
		},
		{
			name:      "OD with double data",
			vr:        vr.OtherDouble,
			data:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F}, // 1.0 in IEEE 754
			wantErr:   false,
			wantBytes: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F},
		},
		{
			name:      "OF with float data",
			vr:        vr.OtherFloat,
			data:      []byte{0x00, 0x00, 0x80, 0x3F}, // 1.0 in IEEE 754 single
			wantErr:   false,
			wantBytes: []byte{0x00, 0x00, 0x80, 0x3F},
		},
		{
			name:      "OL with long data",
			vr:        vr.OtherLong,
			data:      []byte{0x01, 0x00, 0x00, 0x00}, // 1 as 32-bit int
			wantErr:   false,
			wantBytes: []byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			name:      "OV with very long data",
			vr:        vr.OtherVeryLong,
			data:      []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // 1 as 64-bit int
			wantErr:   false,
			wantBytes: []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:      "UN with unknown data",
			vr:        vr.Unknown,
			data:      []byte{0xDE, 0xAD, 0xBE, 0xEF},
			wantErr:   false,
			wantBytes: []byte{0xDE, 0xAD, 0xBE, 0xEF},
		},
		{
			name:      "empty bytes",
			vr:        vr.OtherByte,
			data:      []byte{},
			wantErr:   false,
			wantBytes: []byte{},
		},
		{
			name:      "nil bytes",
			vr:        vr.OtherByte,
			data:      nil,
			wantErr:   false,
			wantBytes: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewBytesValue(tt.vr, tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.vr, val.VR())
				assert.Equal(t, tt.wantBytes, val.Bytes())
			}
		})
	}
}

// TestBytesValue_String tests string representation of bytes
func TestBytesValue_String(t *testing.T) {
	tests := []struct {
		name string
		vr   vr.VR
		data []byte
		want string
	}{
		{
			name: "small byte array",
			vr:   vr.OtherByte,
			data: []byte{0x01, 0x02, 0x03},
			want: "[01 02 03]",
		},
		{
			name: "empty bytes",
			vr:   vr.OtherByte,
			data: []byte{},
			want: "[]",
		},
		{
			name: "single byte",
			vr:   vr.OtherByte,
			data: []byte{0xFF},
			want: "[FF]",
		},
		{
			name: "long byte array truncated",
			vr:   vr.OtherByte,
			data: make([]byte, 100),
			want: "[00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 ... (100 bytes)]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewBytesValue(tt.vr, tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.want, val.String())
		})
	}
}

// TestBytesValue_Equals tests bytes equality
func TestBytesValue_Equals(t *testing.T) {
	tests := []struct {
		name   string
		vr1    vr.VR
		data1  []byte
		vr2    vr.VR
		data2  []byte
		wantEq bool
	}{
		{
			name:   "equal bytes same VR",
			vr1:    vr.OtherByte,
			data1:  []byte{0x01, 0x02, 0x03},
			vr2:    vr.OtherByte,
			data2:  []byte{0x01, 0x02, 0x03},
			wantEq: true,
		},
		{
			name:   "different bytes same VR",
			vr1:    vr.OtherByte,
			data1:  []byte{0x01, 0x02, 0x03},
			vr2:    vr.OtherByte,
			data2:  []byte{0x04, 0x05, 0x06},
			wantEq: false,
		},
		{
			name:   "equal bytes different VR",
			vr1:    vr.OtherByte,
			data1:  []byte{0x01, 0x02, 0x03},
			vr2:    vr.OtherWord,
			data2:  []byte{0x01, 0x02, 0x03},
			wantEq: false,
		},
		{
			name:   "different lengths",
			vr1:    vr.OtherByte,
			data1:  []byte{0x01, 0x02},
			vr2:    vr.OtherByte,
			data2:  []byte{0x01, 0x02, 0x03},
			wantEq: false,
		},
		{
			name:   "both empty",
			vr1:    vr.OtherByte,
			data1:  []byte{},
			vr2:    vr.OtherByte,
			data2:  []byte{},
			wantEq: true,
		},
		{
			name:   "empty vs nil",
			vr1:    vr.OtherByte,
			data1:  []byte{},
			vr2:    vr.OtherByte,
			data2:  nil,
			wantEq: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val1, err := value.NewBytesValue(tt.vr1, tt.data1)
			require.NoError(t, err)
			val2, err := value.NewBytesValue(tt.vr2, tt.data2)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEq, val1.Equals(val2))
		})
	}
}

// TestBytesValue_Padding tests that odd-length bytes get null-padded
func TestBytesValue_Padding(t *testing.T) {
	tests := []struct {
		name      string
		vr        vr.VR
		data      []byte
		wantBytes []byte
	}{
		{
			name:      "odd length gets padded",
			vr:        vr.OtherByte,
			data:      []byte{0x01, 0x02, 0x03},
			wantBytes: []byte{0x01, 0x02, 0x03, 0x00},
		},
		{
			name:      "even length no padding",
			vr:        vr.OtherByte,
			data:      []byte{0x01, 0x02},
			wantBytes: []byte{0x01, 0x02},
		},
		{
			name:      "empty no padding",
			vr:        vr.OtherByte,
			data:      []byte{},
			wantBytes: []byte{},
		},
		{
			name:      "single byte gets padded",
			vr:        vr.OtherByte,
			data:      []byte{0xFF},
			wantBytes: []byte{0xFF, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewBytesValue(tt.vr, tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBytes, val.Bytes())
		})
	}
}

// TestBytesValue_InvalidVR tests that non-bytes VRs are rejected
func TestBytesValue_InvalidVR(t *testing.T) {
	tests := []struct {
		name string
		vr   vr.VR
	}{
		{
			name: "reject CS (code string)",
			vr:   vr.CodeString,
		},
		{
			name: "reject SL (signed long)",
			vr:   vr.SignedLong,
		},
		{
			name: "reject FD (float double)",
			vr:   vr.FloatingPointDouble,
		},
		// Note: SQ (SequenceOfItems) is now accepted as a binary VR because
		// sequences are skipped during parsing and represented as placeholder byte values
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value.NewBytesValue(tt.vr, []byte{0x01, 0x02})
			require.Error(t, err)
		})
	}
}
