package pixel

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewICCProfile tests creating ICC profile from data
func TestNewICCProfile(t *testing.T) {
	t.Run("ValidProfile", func(t *testing.T) {
		profileData := createMockICCProfile(t)

		profile, err := NewICCProfile(profileData)
		require.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, uint32(len(profileData)), profile.ProfileSize)
		assert.Equal(t, "acsp", string(profile.ProfileSignature[:]))
	})

	t.Run("TooShort", func(t *testing.T) {
		shortData := make([]byte, 64) // Less than required 128 bytes

		_, err := NewICCProfile(shortData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		profileData := createMockICCProfile(t)
		// Corrupt signature
		copy(profileData[36:40], []byte("badd"))

		_, err := NewICCProfile(profileData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ICC profile signature")
	})
}

// TestICCProfileParsing tests header parsing
func TestICCProfileParsing(t *testing.T) {
	profileData := createMockICCProfile(t)
	profile, err := NewICCProfile(profileData)
	require.NoError(t, err)

	// Verify header fields were parsed
	assert.Equal(t, uint32(len(profileData)), profile.ProfileSize)
	assert.NotEmpty(t, profile.ColorSpace())
	assert.NotEmpty(t, profile.PCS())
	assert.NotEmpty(t, profile.Class())
	assert.NotEmpty(t, profile.Version())
}

// TestICCProfileColorSpace tests color space extraction
func TestICCProfileColorSpace(t *testing.T) {
	profileData := createMockICCProfile(t)
	profile, err := NewICCProfile(profileData)
	require.NoError(t, err)

	colorSpace := profile.ColorSpace()
	assert.NotEmpty(t, colorSpace)
	assert.Len(t, colorSpace, 4) // Color space is 4 bytes
}

// TestICCProfilePCS tests PCS extraction
func TestICCProfilePCS(t *testing.T) {
	profileData := createMockICCProfile(t)
	profile, err := NewICCProfile(profileData)
	require.NoError(t, err)

	pcs := profile.PCS()
	assert.NotEmpty(t, pcs)
	assert.Len(t, pcs, 4) // PCS is 4 bytes
}

// TestICCProfileClass tests profile class extraction
func TestICCProfileClass(t *testing.T) {
	profileData := createMockICCProfile(t)
	profile, err := NewICCProfile(profileData)
	require.NoError(t, err)

	class := profile.Class()
	assert.NotEmpty(t, class)
	assert.Len(t, class, 4) // Class is 4 bytes
}

// TestICCProfileVersion tests version extraction
func TestICCProfileVersion(t *testing.T) {
	profileData := createMockICCProfile(t)
	profile, err := NewICCProfile(profileData)
	require.NoError(t, err)

	version := profile.Version()
	assert.NotEmpty(t, version)
	assert.Contains(t, version, ".") // Should be formatted like "4.3.0"
}

// TestICCProfileString tests string representation
func TestICCProfileString(t *testing.T) {
	profileData := createMockICCProfile(t)
	profile, err := NewICCProfile(profileData)
	require.NoError(t, err)

	str := profile.String()
	assert.NotEmpty(t, str)
	assert.Contains(t, str, "ICC Profile")
	assert.Contains(t, str, "Class")
	assert.Contains(t, str, "Color Space")
	assert.Contains(t, str, "PCS")
	assert.Contains(t, str, "Size")
}

// TestLinearToSRGBGammaCorrection tests sRGB gamma curve
func TestLinearToSRGBGammaCorrection(t *testing.T) {
	tests := []struct {
		name   string
		linear float64
	}{
		{"Black", 0.0},
		{"Low", 0.001},
		{"Threshold", 0.0031308},
		{"Mid", 0.5},
		{"High", 0.9},
		{"White", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srgb := linearToSRGB(tt.linear)

			// Verify output is in valid range
			assert.GreaterOrEqual(t, srgb, 0.0)
			assert.LessOrEqual(t, srgb, 1.0)

			// Verify monotonicity: higher linear should give higher sRGB
			if tt.linear > 0 {
				lowerSRGB := linearToSRGB(tt.linear * 0.9)
				assert.Less(t, lowerSRGB, srgb)
			}
		})
	}
}

// TestSRGBToLinearGammaCorrection tests inverse sRGB gamma
func TestSRGBToLinearGammaCorrection(t *testing.T) {
	tests := []struct {
		name string
		srgb float64
	}{
		{"Black", 0.0},
		{"Low", 0.001},
		{"Threshold", 0.04045},
		{"Mid", 0.5},
		{"High", 0.9},
		{"White", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linear := srgbToLinear(tt.srgb)

			// Verify output is in valid range
			assert.GreaterOrEqual(t, linear, 0.0)
			assert.LessOrEqual(t, linear, 1.0)

			// Verify monotonicity
			if tt.srgb > 0 {
				lowerLinear := srgbToLinear(tt.srgb * 0.9)
				assert.Less(t, lowerLinear, linear)
			}
		})
	}
}

// TestGammaRoundTrip tests sRGB gamma round-trip
func TestGammaRoundTrip(t *testing.T) {
	tests := []float64{0.0, 0.1, 0.3, 0.5, 0.7, 0.9, 1.0}

	for _, linear := range tests {
		t.Run("Linear_"+string(rune(int(linear*10))), func(t *testing.T) {
			// Linear -> sRGB -> Linear
			srgb := linearToSRGB(linear)
			backToLinear := srgbToLinear(srgb)

			// Should be approximately equal (allowing for floating point errors)
			assert.InDelta(t, linear, backToLinear, 0.01)
		})
	}
}

// TestToColor tests color conversion
func TestToColor(t *testing.T) {
	tests := []struct {
		name    string
		r, g, b uint8
	}{
		{"Black", 0, 0, 0},
		{"White", 255, 255, 255},
		{"Red", 255, 0, 0},
		{"Green", 0, 255, 0},
		{"Blue", 0, 0, 255},
		{"Gray", 128, 128, 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := ToColor(tt.r, tt.g, tt.b)
			assert.NotNil(t, color)

			r, g, b, a := color.RGBA()

			// RGBA returns 16-bit values, so convert back to 8-bit
			assert.Equal(t, uint32(tt.r)*257, r)
			assert.Equal(t, uint32(tt.g)*257, g)
			assert.Equal(t, uint32(tt.b)*257, b)
			assert.Equal(t, uint32(255)*257, a) // Alpha should be 255
		})
	}
}

// TestRenderingIntent tests rendering intent enum
func TestRenderingIntent(t *testing.T) {
	tests := []struct {
		name   string
		intent RenderingIntent
		value  int
	}{
		{"Perceptual", RenderingIntentPerceptual, 0},
		{"RelativeColorimetric", RenderingIntentRelativeColorimetric, 1},
		{"Saturation", RenderingIntentSaturation, 2},
		{"AbsoluteColorimetric", RenderingIntentAbsoluteColorimetric, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, RenderingIntent(tt.value), tt.intent)
		})
	}
}

// TestPowFunction tests the power function (using math.Pow)
func TestPowFunction(t *testing.T) {
	tests := []struct {
		name     string
		x, y     float64
		expected float64
		delta    float64
	}{
		{"Square", 2.0, 2.0, 4.0, 0.001},
		{"Cube", 2.0, 3.0, 8.0, 0.001},
		{"SquareRoot", 4.0, 0.5, 2.0, 0.001},
		{"Identity", 5.0, 1.0, 5.0, 0.001},
		{"Zero", 0.0, 2.0, 0.0, 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := math.Pow(tt.x, tt.y)
			assert.InDelta(t, tt.expected, result, tt.delta)
		})
	}
}

// Helper functions

func createMockICCProfile(t *testing.T) []byte {
	t.Helper()

	// Create minimal valid ICC profile header (128 bytes)
	profile := make([]byte, 128)

	// Profile size (bytes 0-3, big-endian)
	profile[0] = 0x00
	profile[1] = 0x00
	profile[2] = 0x00
	profile[3] = 0x80 // 128 bytes

	// CMM Type (bytes 4-7)
	copy(profile[4:8], []byte("ADBE"))

	// Version (bytes 8-11) - version 4.3.0.0
	profile[8] = 0x04 // Major version
	profile[9] = 0x30 // Minor version (3.0)
	profile[10] = 0x00
	profile[11] = 0x00

	// Profile Class (bytes 12-15) - "scnr" for input device
	copy(profile[12:16], []byte("scnr"))

	// Color Space Type (bytes 16-19) - "RGB "
	copy(profile[16:20], []byte("RGB "))

	// PCS Type (bytes 20-23) - "XYZ "
	copy(profile[20:24], []byte("XYZ "))

	// Creation DateTime (bytes 24-35) - 12 bytes
	// Format: YYYY MM DD HH MM SS (2 bytes each)
	copy(profile[24:36], []byte{
		0x07, 0xE8, // 2024
		0x00, 0x01, // January
		0x00, 0x0F, // 15th
		0x00, 0x0C, // 12
		0x00, 0x00, // 00
		0x00, 0x00, // 00
	})

	// Profile Signature (bytes 36-39) - "acsp"
	copy(profile[36:40], []byte("acsp"))

	// Platform Signature (bytes 40-43) - "APPL"
	copy(profile[40:44], []byte("APPL"))

	// Profile Flags (bytes 44-47)
	profile[44] = 0x00
	profile[45] = 0x00
	profile[46] = 0x00
	profile[47] = 0x00

	// Device Manufacturer (bytes 48-51)
	copy(profile[48:52], []byte("ADBE"))

	// Device Model (bytes 52-55)
	copy(profile[52:56], []byte("TEST"))

	// Rendering Intent (bytes 64-67)
	profile[64] = 0x00
	profile[65] = 0x00
	profile[66] = 0x00
	profile[67] = 0x00 // Perceptual

	return profile
}
