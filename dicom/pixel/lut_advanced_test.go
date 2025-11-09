package pixel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyPresentationLUTShape tests shape-based presentation LUT
func TestApplyPresentationLUTShape(t *testing.T) {
	tests := []struct {
		name  string
		shape string
	}{
		{"Identity", "IDENTITY"},
		{"Inverse", "INVERSE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixelData := setupGrayscaleTestData(t, 64, 64, 8)

			presentationLUT := &PresentationLUT{
				PresentationLUTShape: tt.shape,
			}

			result, err := ApplyPresentationLUT(pixelData, presentationLUT)
			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.shape == "IDENTITY" {
				// Identity should preserve pixel values
				assert.Equal(t, len(pixelData.data), len(result.data))
			} else if tt.shape == "INVERSE" {
				// Inverse should invert pixel values
				assert.Equal(t, len(pixelData.data), len(result.data))
				// Check first pixel is inverted
				if len(pixelData.data) > 0 && len(result.data) > 0 {
					expected := uint8(255) - pixelData.data[0]
					assert.Equal(t, expected, result.data[0])
				}
			}
		})
	}
}

// TestApplyPresentationLUTInvalidShape tests invalid shape handling
func TestApplyPresentationLUTInvalidShape(t *testing.T) {
	pixelData := setupGrayscaleTestData(t, 64, 64, 8)

	presentationLUT := &PresentationLUT{
		PresentationLUTShape: "INVALID_SHAPE",
	}

	_, err := ApplyPresentationLUT(pixelData, presentationLUT)
	assert.Error(t, err)
}

// TestApplyPresentationLUTTable tests table-based presentation LUT
func TestApplyPresentationLUTTable(t *testing.T) {
	pixelData := setupGrayscaleTestData(t, 64, 64, 8)

	// Create a simple linear LUT
	lutData := make([]uint16, 256)
	for i := 0; i < 256; i++ {
		lutData[i] = uint16(i)
	}

	presentationLUT := &PresentationLUT{
		LUTData:       lutData,
		LUTDescriptor: [3]uint16{256, 0, 8},
	}

	result, err := ApplyPresentationLUT(pixelData, presentationLUT)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, pixelData.Rows, result.Rows)
	assert.Equal(t, pixelData.Columns, result.Columns)
}

// TestInvertPixelData tests pixel inversion
func TestInvertPixelData(t *testing.T) {
	t.Run("8-bit", func(t *testing.T) {
		pixelData := setupGrayscaleTestData(t, 8, 8, 8)

		result, err := invertPixelData(pixelData)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify inversion
		for i := 0; i < len(pixelData.data); i++ {
			expected := 255 - pixelData.data[i]
			assert.Equal(t, expected, result.data[i])
		}
	})

	t.Run("16-bit", func(t *testing.T) {
		pixelData := setupGrayscaleTestData(t, 8, 8, 16)

		result, err := invertPixelData(pixelData)
		require.NoError(t, err)
		assert.NotNil(t, result)

		maxVal := uint16((1 << 16) - 1)

		// Verify inversion (check first few pixels)
		for i := 0; i < 4 && i*2+1 < len(pixelData.data); i++ {
			origVal := uint16(pixelData.data[i*2]) | uint16(pixelData.data[i*2+1])<<8
			resultVal := uint16(result.data[i*2]) | uint16(result.data[i*2+1])<<8
			expected := maxVal - origVal
			assert.Equal(t, expected, resultVal)
		}
	})
}

// TestApplyPaletteColorLUT tests palette color transformation
func TestApplyPaletteColorLUT(t *testing.T) {
	pixelData := setupPaletteColorTestData(t, 64, 64)

	// Create grayscale palette
	palette := setupSimplePalette(t, 256)

	result, err := ApplyPaletteColorLUT(pixelData, palette)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify output is RGB
	assert.Equal(t, uint16(3), result.SamplesPerPixel)
	assert.Equal(t, "RGB", result.PhotometricInterpretation)
	assert.Equal(t, pixelData.Rows, result.Rows)
	assert.Equal(t, pixelData.Columns, result.Columns)

	// Verify data size (RGB = 3 bytes per pixel)
	expectedSize := int(pixelData.Rows) * int(pixelData.Columns) * 3
	assert.Equal(t, expectedSize, len(result.data))
}

// TestApplyPaletteColorLUTInvalidPhotometric tests error on non-palette image
func TestApplyPaletteColorLUTInvalidPhotometric(t *testing.T) {
	pixelData := setupGrayscaleTestData(t, 64, 64, 8)
	palette := setupSimplePalette(t, 256)

	_, err := ApplyPaletteColorLUT(pixelData, palette)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PALETTE COLOR")
}

// TestSegmentedLUTExpand tests segmented LUT expansion
func TestSegmentedLUTExpand(t *testing.T) {
	t.Run("DiscreteSegment", func(t *testing.T) {
		// Create segmented LUT with discrete segment
		segmentedLUT := &SegmentedLUT{
			Data: []uint16{
				0x0005, 100, 200, 300, 400, 500, // Discrete segment: 5 values
			},
		}

		result, err := segmentedLUT.Expand(5)
		require.NoError(t, err)
		require.Len(t, result, 5)

		assert.Equal(t, uint16(100), result[0])
		assert.Equal(t, uint16(200), result[1])
		assert.Equal(t, uint16(300), result[2])
		assert.Equal(t, uint16(400), result[3])
		assert.Equal(t, uint16(500), result[4])
	})

	t.Run("LinearSegment", func(t *testing.T) {
		// Create segmented LUT with discrete + linear segment
		segmentedLUT := &SegmentedLUT{
			Data: []uint16{
				0x0001, 0, // Discrete: 1 value (0)
				0x0104, 1000, // Linear: 4 values from 0 to 1000
			},
		}

		result, err := segmentedLUT.Expand(5)
		require.NoError(t, err)
		require.Len(t, result, 5)

		assert.Equal(t, uint16(0), result[0]) // First discrete value
		// Linear interpolation from 0 to 1000 over 4 steps
		assert.True(t, result[1] < result[2])
		assert.True(t, result[2] < result[3])
		assert.True(t, result[3] < result[4])
	})

	t.Run("IndirectSegment", func(t *testing.T) {
		// Create segmented LUT with indirect segment
		segmentedLUT := &SegmentedLUT{
			Data: []uint16{
				0x0003, 100, 200, 300, // Discrete: 3 values
				0x0202, 0, // Indirect: copy 2 values from position 0
			},
		}

		result, err := segmentedLUT.Expand(5)
		require.NoError(t, err)
		require.Len(t, result, 5)

		assert.Equal(t, uint16(100), result[0])
		assert.Equal(t, uint16(200), result[1])
		assert.Equal(t, uint16(300), result[2])
		assert.Equal(t, uint16(100), result[3]) // Copied from position 0
		assert.Equal(t, uint16(200), result[4]) // Copied from position 1
	})
}

// TestSegmentedLUTExpandError tests error conditions
func TestSegmentedLUTExpandError(t *testing.T) {
	t.Run("InvalidSegmentType", func(t *testing.T) {
		segmentedLUT := &SegmentedLUT{
			Data: []uint16{
				0x0305, 100, // Invalid segment type (3)
			},
		}

		_, err := segmentedLUT.Expand(10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown segment type")
	})

	t.Run("DiscreteSegmentExceedsData", func(t *testing.T) {
		segmentedLUT := &SegmentedLUT{
			Data: []uint16{
				0x0010, 100, // Claims 16 values but only provides 1
			},
		}

		_, err := segmentedLUT.Expand(20)
		assert.Error(t, err)
	})
}

// TestConvertColorSpace tests color space conversions
func TestConvertColorSpace(t *testing.T) {
	t.Run("ToSRGB", func(t *testing.T) {
		pixelData := setupRGBTestData(t, 32, 32)

		result, err := ConvertColorSpace(pixelData, "sRGB")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "sRGB", result.PhotometricInterpretation)
		assert.Equal(t, pixelData.Rows, result.Rows)
		assert.Equal(t, pixelData.Columns, result.Columns)
	})

	t.Run("ToLinearRGB", func(t *testing.T) {
		pixelData := setupRGBTestData(t, 32, 32)

		result, err := ConvertColorSpace(pixelData, "Linear RGB")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "RGB", result.PhotometricInterpretation)
		assert.Equal(t, pixelData.Rows, result.Rows)
		assert.Equal(t, pixelData.Columns, result.Columns)
	})

	t.Run("UnsupportedColorSpace", func(t *testing.T) {
		pixelData := setupRGBTestData(t, 32, 32)

		_, err := ConvertColorSpace(pixelData, "CMYK")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported target color space")
	})

	t.Run("NonRGBImage", func(t *testing.T) {
		pixelData := setupGrayscaleTestData(t, 32, 32, 8)

		_, err := ConvertColorSpace(pixelData, "sRGB")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SamplesPerPixel=3")
	})
}

// TestLinearToSRGB tests linear to sRGB conversion
func TestLinearToSRGB(t *testing.T) {
	tests := []struct {
		linear float64
		min    float64
		max    float64
	}{
		{0.0, 0.0, 0.01},
		{0.5, 0.7, 0.8},
		{1.0, 0.99, 1.0},
	}

	for _, tt := range tests {
		result := linearToSRGB(tt.linear)
		assert.GreaterOrEqual(t, result, tt.min)
		assert.LessOrEqual(t, result, tt.max)
	}
}

// TestSRGBToLinear tests sRGB to linear conversion
func TestSRGBToLinear(t *testing.T) {
	tests := []struct {
		srgb float64
		min  float64
		max  float64
	}{
		{0.0, 0.0, 0.01},
		{0.5, 0.2, 0.3},
		{1.0, 0.99, 1.0},
	}

	for _, tt := range tests {
		result := srgbToLinear(tt.srgb)
		assert.GreaterOrEqual(t, result, tt.min)
		assert.LessOrEqual(t, result, tt.max)
	}
}

// TestPaletteColorLUTLookup tests palette lookup functions
func TestPaletteColorLUTLookup(t *testing.T) {
	palette := setupSimplePalette(t, 256)

	t.Run("ValidIndex", func(t *testing.T) {
		r := palette.lookupRed(128)
		g := palette.lookupGreen(128)
		b := palette.lookupBlue(128)

		assert.NotZero(t, r)
		assert.NotZero(t, g)
		assert.NotZero(t, b)

		// For grayscale palette, R=G=B
		assert.Equal(t, r, g)
		assert.Equal(t, g, b)
	})

	t.Run("OutOfBoundsNegative", func(t *testing.T) {
		r := palette.lookupRed(-1)
		assert.Equal(t, uint16(0), r)
	})

	t.Run("OutOfBoundsTooHigh", func(t *testing.T) {
		r := palette.lookupRed(1000)
		assert.Equal(t, uint16(0), r)
	})
}

// Helper functions

func setupGrayscaleTestData(t *testing.T, rows, cols, bits uint16) *PixelData {
	t.Helper()

	numPixels := int(rows) * int(cols)
	bytesPerPixel := int(bits / 8)
	data := make([]byte, numPixels*bytesPerPixel)

	// Fill with gradient
	for i := 0; i < numPixels; i++ {
		val := byte((i * 255) / numPixels)
		if bytesPerPixel == 1 {
			data[i] = val
		} else {
			val16 := uint16(val) * 256
			data[i*2] = byte(val16)
			data[i*2+1] = byte(val16 >> 8)
		}
	}

	pixelData := &PixelData{
		Rows:                      rows,
		Columns:                   cols,
		BitsAllocated:             bits,
		BitsStored:                bits,
		HighBit:                   bits - 1,
		PixelRepresentation:       0,
		SamplesPerPixel:           1,
		PhotometricInterpretation: "MONOCHROME2",
		PlanarConfiguration:       0,
		NumberOfFrames:            1,
		data:                      data,
	}

	return pixelData
}

func setupRGBTestData(t *testing.T, rows, cols uint16) *PixelData {
	t.Helper()

	numPixels := int(rows) * int(cols)
	data := make([]byte, numPixels*3) // RGB

	// Fill with color gradient
	for i := 0; i < numPixels; i++ {
		data[i*3] = byte((i * 255) / numPixels)           // R
		data[i*3+1] = byte(((i % 256) * 255) / 256)       // G
		data[i*3+2] = byte(((i / 256) % 256) * 255 / 256) // B
	}

	pixelData := &PixelData{
		Rows:                      rows,
		Columns:                   cols,
		BitsAllocated:             8,
		BitsStored:                8,
		HighBit:                   7,
		PixelRepresentation:       0,
		SamplesPerPixel:           3,
		PhotometricInterpretation: "RGB",
		PlanarConfiguration:       0,
		NumberOfFrames:            1,
		data:                      data,
	}

	return pixelData
}

func setupPaletteColorTestData(t *testing.T, rows, cols uint16) *PixelData {
	t.Helper()

	numPixels := int(rows) * int(cols)
	data := make([]byte, numPixels)

	// Fill with indices
	for i := 0; i < numPixels; i++ {
		data[i] = byte(i % 256)
	}

	pixelData := &PixelData{
		Rows:                      rows,
		Columns:                   cols,
		BitsAllocated:             8,
		BitsStored:                8,
		HighBit:                   7,
		PixelRepresentation:       0,
		SamplesPerPixel:           1,
		PhotometricInterpretation: "PALETTE COLOR",
		PlanarConfiguration:       0,
		NumberOfFrames:            1,
		data:                      data,
	}

	return pixelData
}

func setupSimplePalette(t *testing.T, numEntries int) *PaletteColorLUT {
	t.Helper()

	redData := make([]uint16, numEntries)
	greenData := make([]uint16, numEntries)
	blueData := make([]uint16, numEntries)

	// Create grayscale palette
	for i := 0; i < numEntries; i++ {
		val := uint16((i * 65535) / (numEntries - 1))
		redData[i] = val
		greenData[i] = val
		blueData[i] = val
	}

	return &PaletteColorLUT{
		RedDescriptor:   [3]uint16{uint16(numEntries), 0, 16},
		GreenDescriptor: [3]uint16{uint16(numEntries), 0, 16},
		BlueDescriptor:  [3]uint16{uint16(numEntries), 0, 16},
		RedData:         redData,
		GreenData:       greenData,
		BlueData:        blueData,
	}
}
