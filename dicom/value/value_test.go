package value_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStringValue_NewStringValue tests creating string values with various VRs
func TestStringValue_NewStringValue(t *testing.T) {
	tests := []struct {
		name      string
		vr        vr.VR
		values    []string
		wantErr   bool
		wantValue []string
	}{
		{
			name:      "single AE value",
			vr:        vr.ApplicationEntity,
			values:    []string{"MYAETITLE"},
			wantErr:   false,
			wantValue: []string{"MYAETITLE"},
		},
		{
			name:      "single CS value",
			vr:        vr.CodeString,
			values:    []string{"ORIGINAL"},
			wantErr:   false,
			wantValue: []string{"ORIGINAL"},
		},
		{
			name:      "multi-value CS",
			vr:        vr.CodeString,
			values:    []string{"ORIGINAL", "PRIMARY", "AXIAL"},
			wantErr:   false,
			wantValue: []string{"ORIGINAL", "PRIMARY", "AXIAL"},
		},
		{
			name:      "single LO value",
			vr:        vr.LongString,
			values:    []string{"Patient Name"},
			wantErr:   false,
			wantValue: []string{"Patient Name"},
		},
		{
			name:      "single PN value",
			vr:        vr.PersonName,
			values:    []string{"Doe^John"},
			wantErr:   false,
			wantValue: []string{"Doe^John"},
		},
		{
			name:      "single UI value",
			vr:        vr.UniqueIdentifier,
			values:    []string{"1.2.840.10008.5.1.4.1.1.2"},
			wantErr:   false,
			wantValue: []string{"1.2.840.10008.5.1.4.1.1.2"},
		},
		{
			name:      "single DA value",
			vr:        vr.Date,
			values:    []string{"20230515"},
			wantErr:   false,
			wantValue: []string{"20230515"},
		},
		{
			name:      "single TM value",
			vr:        vr.Time,
			values:    []string{"143025.123"},
			wantErr:   false,
			wantValue: []string{"143025.123"},
		},
		{
			name:      "single DT value",
			vr:        vr.DateTime,
			values:    []string{"20230515143025.123456"},
			wantErr:   false,
			wantValue: []string{"20230515143025.123456"},
		},
		{
			name:      "empty value",
			vr:        vr.CodeString,
			values:    []string{},
			wantErr:   false,
			wantValue: []string{},
		},
		{
			name:      "empty string in value",
			vr:        vr.CodeString,
			values:    []string{""},
			wantErr:   false,
			wantValue: []string{""},
		},
		{
			name:      "single IS value",
			vr:        vr.IntegerString,
			values:    []string{"123"},
			wantErr:   false,
			wantValue: []string{"123"},
		},
		{
			name:      "single DS value",
			vr:        vr.DecimalString,
			values:    []string{"1.23456"},
			wantErr:   false,
			wantValue: []string{"1.23456"},
		},
		{
			name:      "single AS value",
			vr:        vr.AgeString,
			values:    []string{"025Y"},
			wantErr:   false,
			wantValue: []string{"025Y"},
		},
		{
			name:      "single SH value",
			vr:        vr.ShortString,
			values:    []string{"Short Text"},
			wantErr:   false,
			wantValue: []string{"Short Text"},
		},
		{
			name:      "single LT value",
			vr:        vr.LongText,
			values:    []string{"This is a long text field that can contain multiple sentences."},
			wantErr:   false,
			wantValue: []string{"This is a long text field that can contain multiple sentences."},
		},
		{
			name:      "single ST value",
			vr:        vr.ShortText,
			values:    []string{"Short text description"},
			wantErr:   false,
			wantValue: []string{"Short text description"},
		},
		{
			name:      "single UC value",
			vr:        vr.UnlimitedCharacters,
			values:    []string{"Unlimited characters can be very long"},
			wantErr:   false,
			wantValue: []string{"Unlimited characters can be very long"},
		},
		{
			name:      "single UR value",
			vr:        vr.UniversalResourceIdentifier,
			values:    []string{"http://example.com/path/to/resource"},
			wantErr:   false,
			wantValue: []string{"http://example.com/path/to/resource"},
		},
		{
			name:      "single UT value",
			vr:        vr.UnlimitedText,
			values:    []string{"Unlimited text can contain very long narrative content without length restrictions."},
			wantErr:   false,
			wantValue: []string{"Unlimited text can contain very long narrative content without length restrictions."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewStringValue(tt.vr, tt.values)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.vr, val.VR())
				assert.Equal(t, tt.wantValue, val.Strings())
			}
		})
	}
}

// TestStringValue_String tests string representation
func TestStringValue_String(t *testing.T) {
	tests := []struct {
		name   string
		vr     vr.VR
		values []string
		want   string
	}{
		{
			name:   "single value",
			vr:     vr.CodeString,
			values: []string{"ORIGINAL"},
			want:   "ORIGINAL",
		},
		{
			name:   "multi-value",
			vr:     vr.CodeString,
			values: []string{"ORIGINAL", "PRIMARY", "AXIAL"},
			want:   "ORIGINAL\\PRIMARY\\AXIAL",
		},
		{
			name:   "empty value",
			vr:     vr.CodeString,
			values: []string{},
			want:   "",
		},
		{
			name:   "empty string",
			vr:     vr.CodeString,
			values: []string{""},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewStringValue(tt.vr, tt.values)
			require.NoError(t, err)
			assert.Equal(t, tt.want, val.String())
		})
	}
}

// TestStringValue_Bytes tests byte encoding
func TestStringValue_Bytes(t *testing.T) {
	tests := []struct {
		name   string
		vr     vr.VR
		values []string
		want   []byte
	}{
		{
			name:   "single value",
			vr:     vr.CodeString,
			values: []string{"ORIGINAL"},
			want:   []byte("ORIGINAL"),
		},
		{
			name:   "multi-value",
			vr:     vr.CodeString,
			values: []string{"ORIGINAL", "PRIMARY"},
			want:   []byte("ORIGINAL\\PRIMARY"),
		},
		{
			name:   "empty value",
			vr:     vr.CodeString,
			values: []string{},
			want:   []byte{},
		},
		{
			name:   "UI with odd length needs null padding",
			vr:     vr.UniqueIdentifier,
			values: []string{"1.2.3"},
			want:   []byte("1.2.3\x00"),
		},
		{
			name:   "UI with even length no padding",
			vr:     vr.UniqueIdentifier,
			values: []string{"1.23"},
			want:   []byte("1.23"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := value.NewStringValue(tt.vr, tt.values)
			require.NoError(t, err)
			assert.Equal(t, tt.want, val.Bytes())
		})
	}
}

// TestStringValue_Equals tests value equality
func TestStringValue_Equals(t *testing.T) {
	tests := []struct {
		name   string
		vr1    vr.VR
		vals1  []string
		vr2    vr.VR
		vals2  []string
		wantEq bool
	}{
		{
			name:   "equal single values",
			vr1:    vr.CodeString,
			vals1:  []string{"ORIGINAL"},
			vr2:    vr.CodeString,
			vals2:  []string{"ORIGINAL"},
			wantEq: true,
		},
		{
			name:   "equal multi values",
			vr1:    vr.CodeString,
			vals1:  []string{"ORIGINAL", "PRIMARY"},
			vr2:    vr.CodeString,
			vals2:  []string{"ORIGINAL", "PRIMARY"},
			wantEq: true,
		},
		{
			name:   "different values",
			vr1:    vr.CodeString,
			vals1:  []string{"ORIGINAL"},
			vr2:    vr.CodeString,
			vals2:  []string{"DERIVED"},
			wantEq: false,
		},
		{
			name:   "different VRs",
			vr1:    vr.CodeString,
			vals1:  []string{"TEST"},
			vr2:    vr.LongString,
			vals2:  []string{"TEST"},
			wantEq: false,
		},
		{
			name:   "different lengths",
			vr1:    vr.CodeString,
			vals1:  []string{"ORIGINAL"},
			vr2:    vr.CodeString,
			vals2:  []string{"ORIGINAL", "PRIMARY"},
			wantEq: false,
		},
		{
			name:   "both empty",
			vr1:    vr.CodeString,
			vals1:  []string{},
			vr2:    vr.CodeString,
			vals2:  []string{},
			wantEq: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val1, err := value.NewStringValue(tt.vr1, tt.vals1)
			require.NoError(t, err)
			val2, err := value.NewStringValue(tt.vr2, tt.vals2)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEq, val1.Equals(val2))
		})
	}
}

// TestStringValue_MaxLength tests length validation
func TestStringValue_MaxLength(t *testing.T) {
	tests := []struct {
		name    string
		vr      vr.VR
		value   string
		wantErr bool
	}{
		{
			name:    "AE within limit (16 chars)",
			vr:      vr.ApplicationEntity,
			value:   "MYAETITLE",
			wantErr: false,
		},
		{
			name:    "AE at limit (16 chars)",
			vr:      vr.ApplicationEntity,
			value:   "1234567890123456",
			wantErr: false,
		},
		{
			name:    "AE exceeds limit",
			vr:      vr.ApplicationEntity,
			value:   "12345678901234567",
			wantErr: true,
		},
		{
			name:    "CS within limit (16 chars)",
			vr:      vr.CodeString,
			value:   "ORIGINAL",
			wantErr: false,
		},
		{
			name:    "LO within limit (64 chars)",
			vr:      vr.LongString,
			value:   "This is a long string value within the limit",
			wantErr: false,
		},
		{
			name:    "UI within limit (64 chars)",
			vr:      vr.UniqueIdentifier,
			value:   "1.2.840.10008.5.1.4.1.1.2",
			wantErr: false,
		},
		{
			name:    "AS within limit (4 chars)",
			vr:      vr.AgeString,
			value:   "025Y",
			wantErr: false,
		},
		{
			name:    "AS at limit (4 chars)",
			vr:      vr.AgeString,
			value:   "999D",
			wantErr: false,
		},
		{
			name:    "AS exceeds limit",
			vr:      vr.AgeString,
			value:   "9999D",
			wantErr: true,
		},
		{
			name:    "SH within limit (16 chars)",
			vr:      vr.ShortString,
			value:   "Short String",
			wantErr: false,
		},
		{
			name:    "SH at limit (16 chars)",
			vr:      vr.ShortString,
			value:   "1234567890123456",
			wantErr: false,
		},
		{
			name:    "SH exceeds limit",
			vr:      vr.ShortString,
			value:   "12345678901234567",
			wantErr: true,
		},
		{
			name:    "LT within limit (10240 chars)",
			vr:      vr.LongText,
			value:   "This is a long text field.",
			wantErr: false,
		},
		{
			name:    "ST within limit (1024 chars)",
			vr:      vr.ShortText,
			value:   "This is a short text description.",
			wantErr: false,
		},
		{
			name:    "UC unlimited allowed",
			vr:      vr.UnlimitedCharacters,
			value:   string(make([]byte, 100000)), // 100KB
			wantErr: false,
		},
		{
			name:    "UT unlimited allowed",
			vr:      vr.UnlimitedText,
			value:   string(make([]byte, 100000)), // 100KB
			wantErr: false,
		},
		{
			name:    "PN within limit (64 chars per component group)",
			vr:      vr.PersonName,
			value:   "LastName^FirstName^MiddleName^Prefix^Suffix",
			wantErr: false,
		},
		{
			name:    "PN at limit (64 chars)",
			vr:      vr.PersonName,
			value:   "1234567890123456789012345678901234567890123456789012345678901234",
			wantErr: false,
		},
		{
			name:    "PN exceeds limit",
			vr:      vr.PersonName,
			value:   "12345678901234567890123456789012345678901234567890123456789012345",
			wantErr: true,
		},
		{
			name:    "IS within limit (12 chars)",
			vr:      vr.IntegerString,
			value:   "123456789012",
			wantErr: false,
		},
		{
			name:    "IS exceeds limit",
			vr:      vr.IntegerString,
			value:   "1234567890123",
			wantErr: true,
		},
		{
			name:    "DS within limit (16 chars)",
			vr:      vr.DecimalString,
			value:   "1234567890.12345",
			wantErr: false,
		},
		{
			name:    "DS exceeds limit",
			vr:      vr.DecimalString,
			value:   "12345678901234567",
			wantErr: true,
		},
		{
			name:    "DA within limit (8 chars)",
			vr:      vr.Date,
			value:   "20230515",
			wantErr: false,
		},
		{
			name:    "DA exceeds limit",
			vr:      vr.Date,
			value:   "202305151",
			wantErr: true,
		},
		{
			name:    "TM within limit (14 chars)",
			vr:      vr.Time,
			value:   "143025.1234567", // 14 chars exactly
			wantErr: false,
		},
		{
			name:    "TM exceeds limit",
			vr:      vr.Time,
			value:   "143025.12345678", // 15 chars - exceeds limit
			wantErr: true,
		},
		{
			name:    "DT within limit (26 chars)",
			vr:      vr.DateTime,
			value:   "20230515143025.123456+0000",
			wantErr: false,
		},
		{
			name:    "DT exceeds limit",
			vr:      vr.DateTime,
			value:   "20230515143025.123456+00001",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value.NewStringValue(tt.vr, []string{tt.value})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestStringValue_InvalidVR tests that non-string VRs are rejected
func TestStringValue_InvalidVR(t *testing.T) {
	tests := []struct {
		name string
		vr   vr.VR
	}{
		{
			name: "reject SL (signed long)",
			vr:   vr.SignedLong,
		},
		{
			name: "reject UL (unsigned long)",
			vr:   vr.UnsignedLong,
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
			_, err := value.NewStringValue(tt.vr, []string{"test"})
			require.Error(t, err)
		})
	}
}
