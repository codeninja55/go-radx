package pixel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPhotometricInterpretation_RGBToYBR(t *testing.T) {
	// Create RGB test data
	data := make([]byte, 3*3*3) // 3x3 RGB image
	// Red pixel
	data[0], data[1], data[2] = 255, 0, 0
	// Green pixel
	data[3], data[4], data[5] = 0, 255, 0
	// Blue pixel
	data[6], data[7], data[8] = 0, 0, 255

	pixelData, err := NewPixelDataFromRGB(data, 3, 3)
	require.NoError(t, err)

	// Convert RGB → YBR_FULL
	ybr, err := ConvertPhotometricInterpretation(pixelData, "YBR_FULL")
	require.NoError(t, err)
	require.NotNil(t, ybr)

	assert.Equal(t, "YBR_FULL", ybr.PhotometricInterpretation)
	assert.Equal(t, uint16(3), ybr.SamplesPerPixel)
	assert.Equal(t, uint16(0), ybr.PlanarConfiguration)

	// Verify conversion values (ITU-R BT.601)
	// Red (255,0,0) → Y≈76, Cb≈85, Cr≈255
	assert.InDelta(t, 76, ybr.data[0], 2)
	assert.InDelta(t, 85, ybr.data[1], 2)
	assert.InDelta(t, 255, ybr.data[2], 2)

	// Green (0,255,0) → Y≈150, Cb≈44, Cr≈21
	assert.InDelta(t, 150, ybr.data[3], 2)
	assert.InDelta(t, 44, ybr.data[4], 2)
	assert.InDelta(t, 21, ybr.data[5], 2)

	// Blue (0,0,255) → Y≈29, Cb≈255, Cr≈107
	assert.InDelta(t, 29, ybr.data[6], 2)
	assert.InDelta(t, 255, ybr.data[7], 2)
	assert.InDelta(t, 107, ybr.data[8], 2)
}

func TestConvertPhotometricInterpretation_YBRToRGB(t *testing.T) {
	// Create YBR test data (known values)
	data := make([]byte, 3*3*3) // 3x3 YBR image
	// Red: Y≈76, Cb≈85, Cr≈255
	data[0], data[1], data[2] = 76, 85, 255
	// Green: Y≈150, Cb≈44, Cr≈21
	data[3], data[4], data[5] = 150, 44, 21
	// Blue: Y≈29, Cb≈255, Cr≈107
	data[6], data[7], data[8] = 29, 255, 107

	pixelData, err := NewPixelDataBuilder().
		WithDimensions(3, 3).
		WithBitsAllocated(8).
		WithSamplesPerPixel(3).
		WithPhotometricInterpretation("YBR_FULL").
		WithPlanarConfiguration(0).
		WithPixelData(data).
		Build()
	require.NoError(t, err)

	// Convert YBR_FULL → RGB
	rgb, err := ConvertPhotometricInterpretation(pixelData, "RGB")
	require.NoError(t, err)
	require.NotNil(t, rgb)

	assert.Equal(t, "RGB", rgb.PhotometricInterpretation)

	// Verify conversion back to RGB (with some tolerance for rounding)
	// Red
	assert.InDelta(t, 255, rgb.data[0], 5)
	assert.InDelta(t, 0, rgb.data[1], 5)
	assert.InDelta(t, 0, rgb.data[2], 5)

	// Green
	assert.InDelta(t, 0, rgb.data[3], 5)
	assert.InDelta(t, 255, rgb.data[4], 5)
	assert.InDelta(t, 0, rgb.data[5], 5)

	// Blue
	assert.InDelta(t, 0, rgb.data[6], 5)
	assert.InDelta(t, 0, rgb.data[7], 5)
	assert.InDelta(t, 255, rgb.data[8], 5)
}

func TestConvertPhotometricInterpretation_RGBToYBR422(t *testing.T) {
	// Create 4x2 RGB image (width must be even for 4:2:2)
	data := make([]byte, 4*2*3) // 4x2 RGB
	for i := 0; i < len(data); i += 3 {
		data[i] = 255   // R
		data[i+1] = 128 // G
		data[i+2] = 64  // B
	}

	pixelData, err := NewPixelDataFromRGB(data, 4, 2)
	require.NoError(t, err)

	// Convert RGB → YBR_FULL_422
	ybr422, err := ConvertPhotometricInterpretation(pixelData, "YBR_FULL_422")
	require.NoError(t, err)
	require.NotNil(t, ybr422)

	assert.Equal(t, "YBR_FULL_422", ybr422.PhotometricInterpretation)
	assert.Equal(t, uint16(3), ybr422.SamplesPerPixel)

	// YBR_FULL_422 uses 4:2:2 subsampling
	// Cb and Cr are averaged horizontally (every 2 pixels)
	// Data size should be same (Y values for all, subsampled Cb/Cr)
	assert.Len(t, ybr422.data, len(data))
}

func TestConvertPhotometricInterpretation_YBR422ToRGB(t *testing.T) {
	// Create YBR_FULL_422 test data
	data := make([]byte, 4*2*3) // 4x2 YBR_FULL_422

	pixelData, err := NewPixelDataBuilder().
		WithDimensions(4, 2).
		WithBitsAllocated(8).
		WithSamplesPerPixel(3).
		WithPhotometricInterpretation("YBR_FULL_422").
		WithPlanarConfiguration(0).
		WithPixelData(data).
		Build()
	require.NoError(t, err)

	// Convert YBR_FULL_422 → RGB
	rgb, err := ConvertPhotometricInterpretation(pixelData, "RGB")
	require.NoError(t, err)
	require.NotNil(t, rgb)

	assert.Equal(t, "RGB", rgb.PhotometricInterpretation)
	assert.Len(t, rgb.data, len(data))
}

func TestConvertPhotometricInterpretation_MonochromeInversion(t *testing.T) {
	// Create MONOCHROME2 test data
	data := make([]uint8, 10*10)
	for i := range data {
		data[i] = uint8(i % 256)
	}

	pixelData, err := NewPixelDataFromUint8(data, 10, 10)
	require.NoError(t, err)
	assert.Equal(t, "MONOCHROME2", pixelData.PhotometricInterpretation)

	// Convert MONOCHROME2 → MONOCHROME1
	mono1, err := ConvertPhotometricInterpretation(pixelData, "MONOCHROME1")
	require.NoError(t, err)
	require.NotNil(t, mono1)

	assert.Equal(t, "MONOCHROME1", mono1.PhotometricInterpretation)

	// Verify inversion: output = 255 - input
	for i := range data {
		expected := 255 - data[i]
		assert.Equal(t, expected, mono1.data[i], "pixel %d should be inverted", i)
	}

	// Convert back MONOCHROME1 → MONOCHROME2
	mono2, err := ConvertPhotometricInterpretation(mono1, "MONOCHROME2")
	require.NoError(t, err)
	require.NotNil(t, mono2)

	assert.Equal(t, "MONOCHROME2", mono2.PhotometricInterpretation)

	// Should match original
	for i := range data {
		assert.Equal(t, data[i], mono2.data[i], "pixel %d should match original after double inversion", i)
	}
}

func TestConvertPhotometricInterpretation_Monochrome16bit(t *testing.T) {
	// Create 16-bit MONOCHROME2 test data
	data := make([]uint16, 10*10)
	for i := range data {
		data[i] = uint16(i * 100)
	}

	pixelData, err := NewPixelDataFromUint16(data, 10, 10)
	require.NoError(t, err)

	// Convert MONOCHROME2 → MONOCHROME1
	mono1, err := ConvertPhotometricInterpretation(pixelData, "MONOCHROME1")
	require.NoError(t, err)

	// Verify 16-bit inversion
	array := mono1.Array().([]uint16)
	maxVal := uint16((1 << pixelData.BitsStored) - 1)
	for i := range data {
		expected := maxVal - data[i]
		assert.Equal(t, expected, array[i], "16-bit pixel %d should be inverted", i)
	}
}

func TestConvertPhotometricInterpretation_RoundTrip(t *testing.T) {
	// Test RGB → YBR_FULL → RGB round trip
	original := make([]byte, 100*100*3)
	for i := 0; i < len(original); i++ {
		original[i] = byte(i % 256)
	}

	pixelData, err := NewPixelDataFromRGB(original, 100, 100)
	require.NoError(t, err)

	// Convert to YBR_FULL
	ybr, err := ConvertPhotometricInterpretation(pixelData, "YBR_FULL")
	require.NoError(t, err)

	// Convert back to RGB
	rgb, err := ConvertPhotometricInterpretation(ybr, "RGB")
	require.NoError(t, err)

	// Should be close to original (within rounding tolerance)
	for i := 0; i < len(original); i++ {
		assert.InDelta(t, original[i], rgb.data[i], 2,
			"pixel %d should match after round trip", i)
	}
}

func TestConvertPhotometricInterpretation_InvalidConversion(t *testing.T) {
	data := make([]uint8, 10*10)
	pixelData, err := NewPixelDataFromUint8(data, 10, 10)
	require.NoError(t, err)

	// Try to convert MONOCHROME2 to RGB (invalid)
	_, err = ConvertPhotometricInterpretation(pixelData, "RGB")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported photometric interpretation conversion")
}

func TestConvertPhotometricInterpretation_NoConversionNeeded(t *testing.T) {
	data := make([]uint8, 10*10)
	pixelData, err := NewPixelDataFromUint8(data, 10, 10)
	require.NoError(t, err)

	// Convert MONOCHROME2 → MONOCHROME2 (no-op)
	result, err := ConvertPhotometricInterpretation(pixelData, "MONOCHROME2")
	require.NoError(t, err)
	assert.Equal(t, pixelData, result, "should return same object when no conversion needed")
}

func TestConvertPlanarConfiguration_InterleavedToPlanar(t *testing.T) {
	// Create interleaved RGB data: RGBRGBRGB...
	data := make([]byte, 4*4*3) // 4x4 RGB
	for i := 0; i < len(data); i += 3 {
		data[i] = 100   // R
		data[i+1] = 150 // G
		data[i+2] = 200 // B
	}

	pixelData, err := NewPixelDataFromRGB(data, 4, 4)
	require.NoError(t, err)
	assert.Equal(t, uint16(0), pixelData.PlanarConfiguration)

	// Convert interleaved → planar
	planar, err := ConvertPlanarConfiguration(pixelData, 1)
	require.NoError(t, err)
	require.NotNil(t, planar)

	assert.Equal(t, uint16(1), planar.PlanarConfiguration)
	assert.Len(t, planar.data, len(data))

	// Verify planar organization: RRR...GGG...BBB...
	pixelCount := 4 * 4

	// All R values should be in first third
	for i := 0; i < pixelCount; i++ {
		assert.Equal(t, byte(100), planar.data[i], "R plane pixel %d", i)
	}

	// All G values should be in second third
	for i := 0; i < pixelCount; i++ {
		assert.Equal(t, byte(150), planar.data[pixelCount+i], "G plane pixel %d", i)
	}

	// All B values should be in last third
	for i := 0; i < pixelCount; i++ {
		assert.Equal(t, byte(200), planar.data[2*pixelCount+i], "B plane pixel %d", i)
	}
}

func TestConvertPlanarConfiguration_PlanarToInterleaved(t *testing.T) {
	// Create planar RGB data: RRR...GGG...BBB...
	pixelCount := 4 * 4
	data := make([]byte, pixelCount*3)

	// R plane
	for i := 0; i < pixelCount; i++ {
		data[i] = 100
	}
	// G plane
	for i := 0; i < pixelCount; i++ {
		data[pixelCount+i] = 150
	}
	// B plane
	for i := 0; i < pixelCount; i++ {
		data[2*pixelCount+i] = 200
	}

	pixelData, err := NewPixelDataBuilder().
		WithDimensions(4, 4).
		WithBitsAllocated(8).
		WithSamplesPerPixel(3).
		WithPhotometricInterpretation("RGB").
		WithPlanarConfiguration(1). // Planar
		WithPixelData(data).
		Build()
	require.NoError(t, err)

	// Convert planar → interleaved
	interleaved, err := ConvertPlanarConfiguration(pixelData, 0)
	require.NoError(t, err)
	require.NotNil(t, interleaved)

	assert.Equal(t, uint16(0), interleaved.PlanarConfiguration)
	assert.Len(t, interleaved.data, len(data))

	// Verify interleaved organization: RGBRGBRGB...
	for i := 0; i < pixelCount; i++ {
		assert.Equal(t, byte(100), interleaved.data[i*3], "R value at pixel %d", i)
		assert.Equal(t, byte(150), interleaved.data[i*3+1], "G value at pixel %d", i)
		assert.Equal(t, byte(200), interleaved.data[i*3+2], "B value at pixel %d", i)
	}
}

func TestConvertPlanarConfiguration_RoundTrip(t *testing.T) {
	// Create interleaved RGB data
	data := make([]byte, 100*100*3)
	for i := 0; i < len(data); i++ {
		data[i] = byte(i % 256)
	}

	pixelData, err := NewPixelDataFromRGB(data, 100, 100)
	require.NoError(t, err)

	// Convert to planar
	planar, err := ConvertPlanarConfiguration(pixelData, 1)
	require.NoError(t, err)

	// Convert back to interleaved
	interleaved, err := ConvertPlanarConfiguration(planar, 0)
	require.NoError(t, err)

	// Should match original exactly
	assert.Equal(t, data, interleaved.data)
}

func TestConvertPlanarConfiguration_NoConversionNeeded(t *testing.T) {
	data := make([]byte, 10*10*3)
	pixelData, err := NewPixelDataFromRGB(data, 10, 10)
	require.NoError(t, err)

	// Already interleaved, convert to interleaved (no-op)
	result, err := ConvertPlanarConfiguration(pixelData, 0)
	require.NoError(t, err)
	assert.Equal(t, pixelData, result)
}

func TestConvertPlanarConfiguration_GrayscaleError(t *testing.T) {
	data := make([]uint8, 10*10)
	pixelData, err := NewPixelDataFromUint8(data, 10, 10)
	require.NoError(t, err)

	// Grayscale images don't have planar configuration
	_, err = ConvertPlanarConfiguration(pixelData, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "planar configuration conversion requires SamplesPerPixel > 1")
}

func TestConvertPlanarConfiguration_InvalidTarget(t *testing.T) {
	data := make([]byte, 10*10*3)
	pixelData, err := NewPixelDataFromRGB(data, 10, 10)
	require.NoError(t, err)

	// Invalid target configuration
	_, err = ConvertPlanarConfiguration(pixelData, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid planar configuration")
}

func TestConvertPhotometricInterpretation_RGBPlanar(t *testing.T) {
	// Test RGB → YBR conversion with planar configuration
	pixelCount := 10 * 10
	r := make([]byte, pixelCount)
	g := make([]byte, pixelCount)
	b := make([]byte, pixelCount)

	for i := 0; i < pixelCount; i++ {
		r[i] = 255
		g[i] = 0
		b[i] = 0
	}

	pixelData, err := NewPixelDataFromRGBPlanar(r, g, b, 10, 10)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), pixelData.PlanarConfiguration)

	// Convert RGB (planar) → YBR_FULL
	ybr, err := ConvertPhotometricInterpretation(pixelData, "YBR_FULL")
	require.NoError(t, err)

	// Should maintain planar configuration
	assert.Equal(t, uint16(1), ybr.PlanarConfiguration)
	assert.Equal(t, "YBR_FULL", ybr.PhotometricInterpretation)
}
