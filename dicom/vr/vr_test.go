package vr_test

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVR_String(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		expected string
	}{
		{"Application Entity", vr.ApplicationEntity, "AE"},
		{"Age String", vr.AgeString, "AS"},
		{"Code String", vr.CodeString, "CS"},
		{"Person Name", vr.PersonName, "PN"},
		{"Unique Identifier", vr.UniqueIdentifier, "UI"},
		{"Other Byte", vr.OtherByte, "OB"},
		{"Sequence", vr.SequenceOfItems, "SQ"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.vr.String())
		})
	}
}

func TestVR_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		vrString string
		expected bool
	}{
		{"valid AE", "AE", true},
		{"valid PN", "PN", true},
		{"valid SQ", "SQ", true},
		{"invalid XX", "XX", false},
		{"invalid ZZ", "ZZ", false},
		{"empty string", "", false},
		{"single character", "A", false},
		{"three characters", "ABC", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := vr.IsValid(tc.vrString)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVR_Parse(t *testing.T) {
	tests := []struct {
		name      string
		vrString  string
		expected  vr.VR
		wantError bool
	}{
		{"valid AE", "AE", vr.ApplicationEntity, false},
		{"valid PN", "PN", vr.PersonName, false},
		{"valid UI", "UI", vr.UniqueIdentifier, false},
		{"invalid XX", "XX", vr.VR(0), true},
		{"empty string", "", vr.VR(0), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vr.Parse(tc.vrString)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestVR_UsesExplicitLength32(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		expected bool
	}{
		{"OB uses 32-bit", vr.OtherByte, true},
		{"OD uses 32-bit", vr.OtherDouble, true},
		{"OF uses 32-bit", vr.OtherFloat, true},
		{"OL uses 32-bit", vr.OtherLong, true},
		{"OW uses 32-bit", vr.OtherWord, true},
		{"SQ uses 32-bit", vr.SequenceOfItems, true},
		{"UC uses 32-bit", vr.UnlimitedCharacters, true},
		{"UN uses 32-bit", vr.Unknown, true},
		{"UR uses 32-bit", vr.UniversalResourceIdentifier, true},
		{"UT uses 32-bit", vr.UnlimitedText, true},
		{"AE uses 16-bit", vr.ApplicationEntity, false},
		{"CS uses 16-bit", vr.CodeString, false},
		{"PN uses 16-bit", vr.PersonName, false},
		{"UI uses 16-bit", vr.UniqueIdentifier, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.vr.UsesExplicitLength32()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVR_PaddingByte(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		expected byte
	}{
		{"AE pads with space", vr.ApplicationEntity, ' '},
		{"CS pads with space", vr.CodeString, ' '},
		{"PN pads with space", vr.PersonName, ' '},
		{"UI pads with null", vr.UniqueIdentifier, 0x00},
		{"OB pads with null", vr.OtherByte, 0x00},
		{"OW pads with null", vr.OtherWord, 0x00},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.vr.PaddingByte()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVR_MaxLength(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		expected int
	}{
		{"AE max 16", vr.ApplicationEntity, 16},
		{"AS max 4", vr.AgeString, 4},
		{"CS max 16", vr.CodeString, 16},
		{"UI max 64", vr.UniqueIdentifier, 64},
		{"PN max 324", vr.PersonName, 324},
		{"LO max 64", vr.LongString, 64},
		{"SH max 16", vr.ShortString, 16},
		{"OB unlimited", vr.OtherByte, 0},
		{"SQ unlimited", vr.SequenceOfItems, 0},
		{"UN unlimited", vr.Unknown, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.vr.MaxLength()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVR_AllowsBackslash(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		expected bool
	}{
		{"PN allows backslash", vr.PersonName, true},
		{"AE does not allow", vr.ApplicationEntity, false},
		{"CS does not allow", vr.CodeString, false},
		{"UI does not allow", vr.UniqueIdentifier, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.vr.AllowsBackslash()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVR_IsStringType(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		expected bool
	}{
		{"AE is string", vr.ApplicationEntity, true},
		{"CS is string", vr.CodeString, true},
		{"PN is string", vr.PersonName, true},
		{"UI is string", vr.UniqueIdentifier, true},
		{"LO is string", vr.LongString, true},
		{"OB is not string", vr.OtherByte, false},
		{"OW is not string", vr.OtherWord, false},
		{"SQ is not string", vr.SequenceOfItems, false},
		{"US is not string", vr.UnsignedShort, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.vr.IsStringType()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVR_IsBinaryType(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		expected bool
	}{
		{"OB is binary", vr.OtherByte, true},
		{"OW is binary", vr.OtherWord, true},
		{"OD is binary", vr.OtherDouble, true},
		{"OF is binary", vr.OtherFloat, true},
		{"OL is binary", vr.OtherLong, true},
		{"OV is binary", vr.OtherVeryLong, true},
		{"AE is not binary", vr.ApplicationEntity, false},
		{"PN is not binary", vr.PersonName, false},
		{"US is not binary", vr.UnsignedShort, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.vr.IsBinaryType()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVR_IsNumericType(t *testing.T) {
	tests := []struct {
		name     string
		vr       vr.VR
		expected bool
	}{
		{"US is numeric", vr.UnsignedShort, true},
		{"UL is numeric", vr.UnsignedLong, true},
		{"SS is numeric", vr.SignedShort, true},
		{"SL is numeric", vr.SignedLong, true},
		{"FL is numeric", vr.FloatingPointSingle, true},
		{"FD is numeric", vr.FloatingPointDouble, true},
		{"AE is not numeric", vr.ApplicationEntity, false},
		{"OB is not numeric", vr.OtherByte, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.vr.IsNumericType()
			assert.Equal(t, tc.expected, result)
		})
	}
}
