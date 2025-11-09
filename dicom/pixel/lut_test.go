package pixel

import (
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyWindowLevel_Basic(t *testing.T) {
	// Create 16-bit signed test data (simulating CT Hounsfield Units)
	data := make([]int16, 10*10)
	for i := range data {
		data[i] = int16(-1024 + i*10) // HU range from -1024 to -24
	}

	pixelData, err := NewPixelDataFromInt16(data, 10, 10)
	require.NoError(t, err)

	// Apply lung window: center=-600, width=1500
	windowed, err := ApplyWindowLevel(pixelData, -600, 1500, 8)
	require.NoError(t, err)
	require.NotNil(t, windowed)

	assert.Equal(t, uint16(8), windowed.BitsAllocated)
	assert.Equal(t, uint16(0), windowed.PixelRepresentation) // Output is unsigned
	assert.Len(t, windowed.data, 10*10)                      // 8-bit output

	// Verify window/level transformation
	// Lower bound = -600 - 1500/2 = -1350
	// Upper bound = -600 + 1500/2 = 600
	// Values below -1350 → 0
	// Values above 600 → 255
	// Values in between → linear mapping

	// First pixel: -1024, within window
	// (-1024 - (-1350)) / 1500 * 255 ≈ 55
	assert.InDelta(t, 55, windowed.data[0], 2)

	// Verify some pixel is clipped to 0 (below lower bound)
	// data[0] = -1024 is not below -1350, so won't be clipped
}

func TestApplyWindowLevel_ClampingBehavior(t *testing.T) {
	// Create test data with values below, within, and above window
	data := make([]int16, 5)
	data[0] = -2000 // Below window
	data[1] = -1350 // At lower bound
	data[2] = -600  // Center
	data[3] = 600   // At upper bound
	data[4] = 3000  // Above window

	pixelData, err := NewPixelDataFromInt16(data, 5, 1)
	require.NoError(t, err)

	// Window: center=-600, width=1500
	// Lower bound = -1350, Upper bound = 600
	windowed, err := ApplyWindowLevel(pixelData, -600, 1500, 8)
	require.NoError(t, err)

	// Below lower bound → 0
	assert.Equal(t, uint8(0), windowed.data[0])

	// At lower bound → 0
	assert.Equal(t, uint8(0), windowed.data[1])

	// At center → 127 (middle of 0-255)
	assert.InDelta(t, 127, windowed.data[2], 2)

	// At upper bound → 255
	assert.Equal(t, uint8(255), windowed.data[3])

	// Above upper bound → 255
	assert.Equal(t, uint8(255), windowed.data[4])
}

func TestApplyWindowLevel_8BitInput(t *testing.T) {
	// Test with 8-bit input data
	data := make([]uint8, 10*10)
	for i := range data {
		data[i] = uint8(i)
	}

	pixelData, err := NewPixelDataFromUint8(data, 10, 10)
	require.NoError(t, err)

	// Apply window/level
	windowed, err := ApplyWindowLevel(pixelData, 50, 100, 8)
	require.NoError(t, err)
	require.NotNil(t, windowed)

	assert.Equal(t, uint16(8), windowed.BitsAllocated)
}

func TestApplyWindowLevel_16BitOutput(t *testing.T) {
	data := make([]int16, 10*10)
	for i := range data {
		data[i] = int16(i * 10)
	}

	pixelData, err := NewPixelDataFromInt16(data, 10, 10)
	require.NoError(t, err)

	// Apply window/level with 16-bit output
	windowed, err := ApplyWindowLevel(pixelData, 500, 1000, 16)
	require.NoError(t, err)

	assert.Equal(t, uint16(16), windowed.BitsAllocated)
	assert.Len(t, windowed.data, 10*10*2) // 16-bit = 2 bytes per pixel
}

func TestApplyWindowLevel_InvalidInputs(t *testing.T) {
	data := make([]uint8, 10*10)
	pixelData, err := NewPixelDataFromUint8(data, 10, 10)
	require.NoError(t, err)

	// Negative window width
	_, err = ApplyWindowLevel(pixelData, 128, -100, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "window width must be positive")

	// Zero window width
	_, err = ApplyWindowLevel(pixelData, 128, 0, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "window width must be positive")

	// Invalid output bits
	_, err = ApplyWindowLevel(pixelData, 128, 100, 12)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "output bits must be 8 or 16")
}

func TestApplyWindowLevel_ColorImageError(t *testing.T) {
	data := make([]byte, 10*10*3)
	pixelData, err := NewPixelDataFromRGB(data, 10, 10)
	require.NoError(t, err)

	// Window/level only applies to grayscale
	_, err = ApplyWindowLevel(pixelData, 128, 100, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only applies to grayscale images")
}

func TestApplyModalityLUT_BasicRescale(t *testing.T) {
	// Create test data
	data := make([]uint16, 10*10)
	for i := range data {
		data[i] = uint16(i * 10)
	}

	pixelData, err := NewPixelDataFromUint16(data, 10, 10)
	require.NoError(t, err)

	// Apply modality LUT: HU = 1.0 * pixel_value + (-1024)
	rescaled, err := ApplyModalityLUT(pixelData, 1.0, -1024)
	require.NoError(t, err)
	require.NotNil(t, rescaled)

	// Output should be signed (negative values expected)
	assert.Equal(t, uint16(1), rescaled.PixelRepresentation)

	// Verify rescale transformation
	array := rescaled.Array().([]int16)
	for i := range data {
		expected := int16(data[i]) - 1024
		assert.Equal(t, expected, array[i], "pixel %d should be rescaled", i)
	}
}

func TestApplyModalityLUT_NoRescale(t *testing.T) {
	data := make([]uint16, 10*10)
	pixelData, err := NewPixelDataFromUint16(data, 10, 10)
	require.NoError(t, err)

	// Apply identity transformation: slope=1.0, intercept=0.0
	rescaled, err := ApplyModalityLUT(pixelData, 1.0, 0.0)
	require.NoError(t, err)

	// Output should be unsigned (no negative values)
	assert.Equal(t, uint16(0), rescaled.PixelRepresentation)
}

func TestApplyModalityLUT_8BitInput(t *testing.T) {
	data := make([]uint8, 10*10)
	for i := range data {
		data[i] = uint8(i)
	}

	pixelData, err := NewPixelDataFromUint8(data, 10, 10)
	require.NoError(t, err)

	// Apply rescale
	rescaled, err := ApplyModalityLUT(pixelData, 2.0, -50)
	require.NoError(t, err)

	// Output stays 8-bit since range fits
	assert.Equal(t, uint16(8), rescaled.BitsAllocated)
	assert.Equal(t, uint16(1), rescaled.PixelRepresentation) // Signed due to negative values

	// Verify transformation - output is int8 array
	array := rescaled.Array().([]int8)
	for i := range data {
		expected := int8(float64(data[i])*2.0 - 50)
		assert.Equal(t, expected, array[i])
	}
}

func TestApplyModalityLUT_SignedInput(t *testing.T) {
	// Create signed 16-bit test data
	data := make([]int16, 10*10)
	for i := range data {
		data[i] = int16(i - 50)
	}

	pixelData, err := NewPixelDataFromInt16(data, 10, 10)
	require.NoError(t, err)

	// Apply rescale
	rescaled, err := ApplyModalityLUT(pixelData, 1.0, 100)
	require.NoError(t, err)

	// Verify transformation
	array := rescaled.Array().([]int16)
	for i := range data {
		expected := data[i] + 100
		assert.Equal(t, expected, array[i])
	}
}

func TestApplyModalityLUT_ColorImageError(t *testing.T) {
	data := make([]byte, 10*10*3)
	pixelData, err := NewPixelDataFromRGB(data, 10, 10)
	require.NoError(t, err)

	// Modality LUT only applies to grayscale
	_, err = ApplyModalityLUT(pixelData, 1.0, 0.0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only applies to grayscale images")
}

func TestExtractWindowLevelFromDataSet(t *testing.T) {
	// Create a DataSet with window/level tags
	ds := dicom.NewDataSet()

	// Window Center (0028,1050) - DS VR
	centerVal, _ := value.NewStringValue(vr.DecimalString, []string{"128"})
	centerElem, err := element.NewElement(tag.New(0x0028, 0x1050), vr.DecimalString, centerVal)
	require.NoError(t, err)
	err = ds.Add(centerElem)
	require.NoError(t, err)

	// Window Width (0028,1051) - DS VR
	widthVal, _ := value.NewStringValue(vr.DecimalString, []string{"256"})
	widthElem, err := element.NewElement(tag.New(0x0028, 0x1051), vr.DecimalString, widthVal)
	require.NoError(t, err)
	err = ds.Add(widthElem)
	require.NoError(t, err)

	// Extract window/level
	wl, err := ExtractWindowLevelFromDataSet(ds)
	require.NoError(t, err)
	require.NotNil(t, wl)

	assert.Equal(t, 128.0, wl.WindowCenter)
	assert.Equal(t, 256.0, wl.WindowWidth)
}

func TestExtractWindowLevelFromDataSet_MissingTags(t *testing.T) {
	ds := dicom.NewDataSet()

	// Missing both tags
	_, err := ExtractWindowLevelFromDataSet(ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "window center not found")

	// Add only center
	centerVal, _ := value.NewStringValue(vr.DecimalString, []string{"128"})
	centerElem, err := element.NewElement(tag.New(0x0028, 0x1050), vr.DecimalString, centerVal)
	require.NoError(t, err)
	err = ds.Add(centerElem)
	require.NoError(t, err)

	// Missing width
	_, err = ExtractWindowLevelFromDataSet(ds)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "window width not found")
}

func TestExtractModalityLUTFromDataSet(t *testing.T) {
	// Create a DataSet with modality LUT tags
	ds := dicom.NewDataSet()

	// Rescale Intercept (0028,1052) - DS VR
	interceptVal, _ := value.NewStringValue(vr.DecimalString, []string{"-1024"})
	interceptElem, err := element.NewElement(tag.New(0x0028, 0x1052), vr.DecimalString, interceptVal)
	require.NoError(t, err)
	err = ds.Add(interceptElem)
	require.NoError(t, err)

	// Rescale Slope (0028,1053) - DS VR
	slopeVal, _ := value.NewStringValue(vr.DecimalString, []string{"1.0"})
	slopeElem, err := element.NewElement(tag.New(0x0028, 0x1053), vr.DecimalString, slopeVal)
	require.NoError(t, err)
	err = ds.Add(slopeElem)
	require.NoError(t, err)

	// Rescale Type (0028,1054) - LO VR
	typeVal, _ := value.NewStringValue(vr.LongString, []string{"HU"})
	typeElem, err := element.NewElement(tag.New(0x0028, 0x1054), vr.LongString, typeVal)
	require.NoError(t, err)
	err = ds.Add(typeElem)
	require.NoError(t, err)

	// Extract modality LUT
	lut, err := ExtractModalityLUTFromDataSet(ds)
	require.NoError(t, err)
	require.NotNil(t, lut)

	assert.Equal(t, 1.0, lut.RescaleSlope)
	assert.Equal(t, -1024.0, lut.RescaleIntercept)
	assert.Equal(t, "HU", lut.RescaleType)
}

func TestExtractModalityLUTFromDataSet_Defaults(t *testing.T) {
	// Create empty DataSet
	ds := dicom.NewDataSet()

	// Extract modality LUT (should return defaults)
	lut, err := ExtractModalityLUTFromDataSet(ds)
	require.NoError(t, err)
	require.NotNil(t, lut)

	// Default values
	assert.Equal(t, 1.0, lut.RescaleSlope)
	assert.Equal(t, 0.0, lut.RescaleIntercept)
	assert.Equal(t, "", lut.RescaleType)
}

func TestExtractModalityLUTFromDataSet_PartialTags(t *testing.T) {
	ds := dicom.NewDataSet()

	// Add only intercept
	interceptVal, _ := value.NewStringValue(vr.DecimalString, []string{"-500"})
	interceptElem, err := element.NewElement(tag.New(0x0028, 0x1052), vr.DecimalString, interceptVal)
	require.NoError(t, err)
	err = ds.Add(interceptElem)
	require.NoError(t, err)

	lut, err := ExtractModalityLUTFromDataSet(ds)
	require.NoError(t, err)

	// Slope should be default, intercept should be from tag
	assert.Equal(t, 1.0, lut.RescaleSlope)
	assert.Equal(t, -500.0, lut.RescaleIntercept)
}

func TestApplyFullImagePipeline_CompleteTransformation(t *testing.T) {
	// Create CT-like data
	data := make([]uint16, 10*10)
	for i := range data {
		data[i] = uint16(i * 50)
	}

	pixelData, err := NewPixelDataFromUint16(data, 10, 10)
	require.NoError(t, err)

	// Create DataSet with both modality LUT and window/level
	ds := dicom.NewDataSet()

	// Modality LUT tags
	interceptVal, _ := value.NewStringValue(vr.DecimalString, []string{"-1024"})
	interceptElem, _ := element.NewElement(tag.New(0x0028, 0x1052), vr.DecimalString, interceptVal)
	ds.Add(interceptElem)
	slopeVal, _ := value.NewStringValue(vr.DecimalString, []string{"1.0"})
	slopeElem, _ := element.NewElement(tag.New(0x0028, 0x1053), vr.DecimalString, slopeVal)
	ds.Add(slopeElem)
	typeVal, _ := value.NewStringValue(vr.LongString, []string{"HU"})
	typeElem, _ := element.NewElement(tag.New(0x0028, 0x1054), vr.LongString, typeVal)
	ds.Add(typeElem)

	// Window/Level tags
	centerVal, _ := value.NewStringValue(vr.DecimalString, []string{"-600"})
	centerElem, _ := element.NewElement(tag.New(0x0028, 0x1050), vr.DecimalString, centerVal)
	ds.Add(centerElem)
	widthVal, _ := value.NewStringValue(vr.DecimalString, []string{"1500"})
	widthElem, _ := element.NewElement(tag.New(0x0028, 0x1051), vr.DecimalString, widthVal)
	ds.Add(widthElem)

	// Apply complete pipeline
	display, err := ApplyFullImagePipeline(ds, pixelData, 8)
	require.NoError(t, err)
	require.NotNil(t, display)

	// Output should be 8-bit unsigned
	assert.Equal(t, uint16(8), display.BitsAllocated)
	assert.Equal(t, uint16(0), display.PixelRepresentation)
	assert.Len(t, display.data, 10*10)
}

func TestApplyFullImagePipeline_NoLUTs(t *testing.T) {
	data := make([]uint16, 10*10)
	pixelData, err := NewPixelDataFromUint16(data, 10, 10)
	require.NoError(t, err)

	// Empty DataSet
	ds := dicom.NewDataSet()

	// Apply pipeline (should return original data since no LUTs)
	display, err := ApplyFullImagePipeline(ds, pixelData, 8)
	require.NoError(t, err)

	// Should be same as input (no transformations applied)
	assert.Equal(t, pixelData, display)
}

func TestApplyFullImagePipeline_OnlyModalityLUT(t *testing.T) {
	data := make([]uint16, 10*10)
	pixelData, err := NewPixelDataFromUint16(data, 10, 10)
	require.NoError(t, err)

	ds := dicom.NewDataSet()
	interceptVal, _ := value.NewStringValue(vr.DecimalString, []string{"-1024"})
	interceptElem, _ := element.NewElement(tag.New(0x0028, 0x1052), vr.DecimalString, interceptVal)
	ds.Add(interceptElem)
	slopeVal, _ := value.NewStringValue(vr.DecimalString, []string{"1.0"})
	slopeElem, _ := element.NewElement(tag.New(0x0028, 0x1053), vr.DecimalString, slopeVal)
	ds.Add(slopeElem)

	// Apply pipeline (only modality LUT, no window/level)
	display, err := ApplyFullImagePipeline(ds, pixelData, 16)
	require.NoError(t, err)

	// Should have applied modality LUT
	assert.Equal(t, uint16(1), display.PixelRepresentation) // Signed due to negative intercept
}

func TestApplyFullImagePipeline_OnlyWindowLevel(t *testing.T) {
	data := make([]uint16, 10*10)
	pixelData, err := NewPixelDataFromUint16(data, 10, 10)
	require.NoError(t, err)

	ds := dicom.NewDataSet()
	centerVal, _ := value.NewStringValue(vr.DecimalString, []string{"128"})
	centerElem, _ := element.NewElement(tag.New(0x0028, 0x1050), vr.DecimalString, centerVal)
	ds.Add(centerElem)
	widthVal, _ := value.NewStringValue(vr.DecimalString, []string{"256"})
	widthElem, _ := element.NewElement(tag.New(0x0028, 0x1051), vr.DecimalString, widthVal)
	ds.Add(widthElem)

	// Apply pipeline (only window/level, no modality LUT)
	display, err := ApplyFullImagePipeline(ds, pixelData, 8)
	require.NoError(t, err)

	// Should have applied window/level
	assert.Equal(t, uint16(8), display.BitsAllocated)
	assert.Equal(t, uint16(0), display.PixelRepresentation)
}

func TestApplyFullImagePipeline_IdentityModalityLUT(t *testing.T) {
	data := make([]uint16, 10*10)
	pixelData, err := NewPixelDataFromUint16(data, 10, 10)
	require.NoError(t, err)

	ds := dicom.NewDataSet()
	// Identity transformation (slope=1.0, intercept=0.0) should be skipped
	interceptVal, _ := value.NewStringValue(vr.DecimalString, []string{"0.0"})
	interceptElem, _ := element.NewElement(tag.New(0x0028, 0x1052), vr.DecimalString, interceptVal)
	ds.Add(interceptElem)
	slopeVal, _ := value.NewStringValue(vr.DecimalString, []string{"1.0"})
	slopeElem, _ := element.NewElement(tag.New(0x0028, 0x1053), vr.DecimalString, slopeVal)
	ds.Add(slopeElem)
	centerVal, _ := value.NewStringValue(vr.DecimalString, []string{"128"})
	centerElem, _ := element.NewElement(tag.New(0x0028, 0x1050), vr.DecimalString, centerVal)
	ds.Add(centerElem)
	widthVal, _ := value.NewStringValue(vr.DecimalString, []string{"256"})
	widthElem, _ := element.NewElement(tag.New(0x0028, 0x1051), vr.DecimalString, widthVal)
	ds.Add(widthElem)

	// Apply pipeline
	display, err := ApplyFullImagePipeline(ds, pixelData, 8)
	require.NoError(t, err)

	// Modality LUT should be skipped, only window/level applied
	assert.NotNil(t, display)
}

func TestWindowLevel_CTPresets(t *testing.T) {
	// Test common CT window/level presets
	data := make([]int16, 512*512)
	for i := range data {
		// Simulate CT HU values from -1000 (air) to 1000 (bone)
		data[i] = int16(-1000 + (i % 2000))
	}

	pixelData, err := NewPixelDataFromInt16(data, 512, 512)
	require.NoError(t, err)

	testCases := []struct {
		name   string
		center float64
		width  float64
	}{
		{"Lung Window", -600, 1500},
		{"Mediastinum Window", 50, 350},
		{"Bone Window", 300, 1500},
		{"Brain Window", 40, 80},
		{"Liver Window", 60, 160},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			windowed, err := ApplyWindowLevel(pixelData, tc.center, tc.width, 8)
			require.NoError(t, err)
			assert.NotNil(t, windowed)
			assert.Equal(t, uint16(8), windowed.BitsAllocated)
		})
	}
}
