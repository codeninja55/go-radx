package pixel

import (
	"image"
	"image/color"
	"testing"
)

func TestPixelData_Array_Unsigned8Bit(t *testing.T) {
	pd := &PixelData{
		Rows:                4,
		Columns:             4,
		BitsAllocated:       8,
		PixelRepresentation: 0, // unsigned
		SamplesPerPixel:     1,
		data:                []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}

	result := pd.Array()
	pixels, ok := result.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", result)
	}

	if len(pixels) != 16 {
		t.Errorf("expected 16 pixels, got %d", len(pixels))
	}

	// Check a few values
	if pixels[0] != 0x01 {
		t.Errorf("expected pixel[0] = 0x01, got 0x%02X", pixels[0])
	}
	if pixels[15] != 0x10 {
		t.Errorf("expected pixel[15] = 0x10, got 0x%02X", pixels[15])
	}
}

func TestPixelData_Array_Signed8Bit(t *testing.T) {
	pd := &PixelData{
		Rows:                2,
		Columns:             2,
		BitsAllocated:       8,
		PixelRepresentation: 1, // signed
		SamplesPerPixel:     1,
		data:                []byte{0xFF, 0x01, 0x80, 0x7F}, // -1, 1, -128, 127
	}

	result := pd.Array()
	pixels, ok := result.([]int8)
	if !ok {
		t.Fatalf("expected []int8, got %T", result)
	}

	if len(pixels) != 4 {
		t.Errorf("expected 4 pixels, got %d", len(pixels))
	}

	expected := []int8{-1, 1, -128, 127}
	for i, exp := range expected {
		if pixels[i] != exp {
			t.Errorf("pixel[%d]: expected %d, got %d", i, exp, pixels[i])
		}
	}
}

func TestPixelData_Array_Unsigned16Bit(t *testing.T) {
	pd := &PixelData{
		Rows:                2,
		Columns:             2,
		BitsAllocated:       16,
		PixelRepresentation: 0, // unsigned
		SamplesPerPixel:     1,
		// Little-endian 16-bit values: 0x0100, 0x0200, 0x0300, 0x0400
		data: []byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x04},
	}

	result := pd.Array()
	pixels, ok := result.([]uint16)
	if !ok {
		t.Fatalf("expected []uint16, got %T", result)
	}

	if len(pixels) != 4 {
		t.Errorf("expected 4 pixels, got %d", len(pixels))
	}

	expected := []uint16{0x0100, 0x0200, 0x0300, 0x0400}
	for i, exp := range expected {
		if pixels[i] != exp {
			t.Errorf("pixel[%d]: expected 0x%04X, got 0x%04X", i, exp, pixels[i])
		}
	}
}

func TestPixelData_Array_Signed16Bit(t *testing.T) {
	pd := &PixelData{
		Rows:                2,
		Columns:             2,
		BitsAllocated:       16,
		PixelRepresentation: 1, // signed
		SamplesPerPixel:     1,
		// Little-endian 16-bit signed values: -1, 1, -32768, 32767
		data: []byte{0xFF, 0xFF, 0x01, 0x00, 0x00, 0x80, 0xFF, 0x7F},
	}

	result := pd.Array()
	pixels, ok := result.([]int16)
	if !ok {
		t.Fatalf("expected []int16, got %T", result)
	}

	if len(pixels) != 4 {
		t.Errorf("expected 4 pixels, got %d", len(pixels))
	}

	expected := []int16{-1, 1, -32768, 32767}
	for i, exp := range expected {
		if pixels[i] != exp {
			t.Errorf("pixel[%d]: expected %d, got %d", i, exp, pixels[i])
		}
	}
}

func TestPixelData_Frames_SingleFrame(t *testing.T) {
	pd := &PixelData{
		Rows:            4,
		Columns:         4,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
		data:            make([]byte, 16),
	}

	frames := pd.Frames()

	if len(frames) != 1 {
		t.Errorf("expected 1 frame, got %d", len(frames))
	}

	if frames[0].Index != 0 {
		t.Errorf("expected frame index 0, got %d", frames[0].Index)
	}

	if frames[0].Rows != 4 || frames[0].Columns != 4 {
		t.Errorf("expected frame size 4x4, got %dx%d", frames[0].Rows, frames[0].Columns)
	}

	if len(frames[0].data) != 16 {
		t.Errorf("expected frame data length 16, got %d", len(frames[0].data))
	}
}

func TestPixelData_Frames_MultiFrame(t *testing.T) {
	// 3 frames of 2x2 8-bit grayscale
	pd := &PixelData{
		Rows:            2,
		Columns:         2,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  3,
		data: []byte{
			0x01, 0x02, 0x03, 0x04, // Frame 0
			0x05, 0x06, 0x07, 0x08, // Frame 1
			0x09, 0x0A, 0x0B, 0x0C, // Frame 2
		},
	}

	frames := pd.Frames()

	if len(frames) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(frames))
	}

	for i := 0; i < 3; i++ {
		if frames[i].Index != i {
			t.Errorf("frame %d: expected index %d, got %d", i, i, frames[i].Index)
		}

		if len(frames[i].data) != 4 {
			t.Errorf("frame %d: expected data length 4, got %d", i, len(frames[i].data))
		}

		// Check first byte of each frame
		expectedFirstByte := byte(i*4 + 1)
		if frames[i].data[0] != expectedFirstByte {
			t.Errorf("frame %d: expected first byte 0x%02X, got 0x%02X", i, expectedFirstByte, frames[i].data[0])
		}
	}
}

func TestPixelData_Image_Grayscale8Bit(t *testing.T) {
	pd := &PixelData{
		Rows:            2,
		Columns:         2,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		data:            []byte{0x10, 0x20, 0x30, 0x40},
	}

	img := pd.Image()
	grayImg, ok := img.(*image.Gray)
	if !ok {
		t.Fatalf("expected *image.Gray, got %T", img)
	}

	bounds := grayImg.Bounds()
	if bounds.Dx() != 2 || bounds.Dy() != 2 {
		t.Errorf("expected bounds 2x2, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Check pixel values
	expectedPixels := []struct {
		x, y  int
		value uint8
	}{
		{0, 0, 0x10},
		{1, 0, 0x20},
		{0, 1, 0x30},
		{1, 1, 0x40},
	}

	for _, ep := range expectedPixels {
		c := grayImg.GrayAt(ep.x, ep.y)
		if c.Y != ep.value {
			t.Errorf("pixel (%d,%d): expected 0x%02X, got 0x%02X", ep.x, ep.y, ep.value, c.Y)
		}
	}
}

func TestPixelData_Image_Grayscale16Bit(t *testing.T) {
	pd := &PixelData{
		Rows:            2,
		Columns:         2,
		BitsAllocated:   16,
		SamplesPerPixel: 1,
		// Little-endian values: 0x1000, 0x2000, 0x3000, 0x4000
		data: []byte{0x00, 0x10, 0x00, 0x20, 0x00, 0x30, 0x00, 0x40},
	}

	img := pd.Image()
	gray16Img, ok := img.(*image.Gray16)
	if !ok {
		t.Fatalf("expected *image.Gray16, got %T", img)
	}

	bounds := gray16Img.Bounds()
	if bounds.Dx() != 2 || bounds.Dy() != 2 {
		t.Errorf("expected bounds 2x2, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Check one pixel (note: image.Gray16 stores as big-endian)
	c := gray16Img.Gray16At(0, 0)
	if c.Y != 0x1000 {
		t.Errorf("pixel (0,0): expected 0x1000, got 0x%04X", c.Y)
	}
}

func TestPixelData_Image_RGB_Interleaved(t *testing.T) {
	pd := &PixelData{
		Rows:                2,
		Columns:             2,
		BitsAllocated:       8,
		SamplesPerPixel:     3,
		PlanarConfiguration: 0, // Interleaved (RGBRGBRGB...)
		// Pixels: (255,0,0) (0,255,0) (0,0,255) (128,128,128)
		data: []byte{
			255, 0, 0, // Red
			0, 255, 0, // Green
			0, 0, 255, // Blue
			128, 128, 128, // Gray
		},
	}

	img := pd.Image()
	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		t.Fatalf("expected *image.RGBA, got %T", img)
	}

	bounds := rgbaImg.Bounds()
	if bounds.Dx() != 2 || bounds.Dy() != 2 {
		t.Errorf("expected bounds 2x2, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Check pixel colors
	expectedColors := []struct {
		x, y    int
		r, g, b uint8
	}{
		{0, 0, 255, 0, 0},     // Red
		{1, 0, 0, 255, 0},     // Green
		{0, 1, 0, 0, 255},     // Blue
		{1, 1, 128, 128, 128}, // Gray
	}

	for _, ec := range expectedColors {
		c := rgbaImg.RGBAAt(ec.x, ec.y)
		if c.R != ec.r || c.G != ec.g || c.B != ec.b {
			t.Errorf("pixel (%d,%d): expected RGB(%d,%d,%d), got RGB(%d,%d,%d)",
				ec.x, ec.y, ec.r, ec.g, ec.b, c.R, c.G, c.B)
		}
		if c.A != 255 {
			t.Errorf("pixel (%d,%d): expected alpha 255, got %d", ec.x, ec.y, c.A)
		}
	}
}

func TestPixelData_Image_RGB_Planar(t *testing.T) {
	pd := &PixelData{
		Rows:                2,
		Columns:             2,
		BitsAllocated:       8,
		SamplesPerPixel:     3,
		PlanarConfiguration: 1, // Planar (RRR...GGG...BBB...)
		// Red plane: 255, 0, 0, 128
		// Green plane: 0, 255, 0, 128
		// Blue plane: 0, 0, 255, 128
		data: []byte{
			255, 0, 0, 128, // Red plane
			0, 255, 0, 128, // Green plane
			0, 0, 255, 128, // Blue plane
		},
	}

	img := pd.Image()
	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		t.Fatalf("expected *image.RGBA, got %T", img)
	}

	// Check first pixel (should be red)
	c := rgbaImg.RGBAAt(0, 0)
	if c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("pixel (0,0): expected RGB(255,0,0), got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}

	// Check last pixel (should be gray)
	c = rgbaImg.RGBAAt(1, 1)
	if c.R != 128 || c.G != 128 || c.B != 128 {
		t.Errorf("pixel (1,1): expected RGB(128,128,128), got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}
}

func TestFrame_Array(t *testing.T) {
	frame := Frame{
		Index:               0,
		Rows:                2,
		Columns:             2,
		BitsAllocated:       8,
		PixelRepresentation: 0, // unsigned
		SamplesPerPixel:     1,
		data:                []byte{0x01, 0x02, 0x03, 0x04},
	}

	result := frame.Array()
	pixels, ok := result.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", result)
	}

	if len(pixels) != 4 {
		t.Errorf("expected 4 pixels, got %d", len(pixels))
	}
}

func TestFrame_Image(t *testing.T) {
	frame := Frame{
		Index:           0,
		Rows:            2,
		Columns:         2,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		data:            []byte{0x10, 0x20, 0x30, 0x40},
	}

	img := frame.Image()
	grayImg, ok := img.(*image.Gray)
	if !ok {
		t.Fatalf("expected *image.Gray, got %T", img)
	}

	c := grayImg.GrayAt(0, 0)
	if c.Y != 0x10 {
		t.Errorf("expected pixel value 0x10, got 0x%02X", c.Y)
	}
}

func TestFrame_At(t *testing.T) {
	// Test grayscale frame
	grayFrame := Frame{
		Rows:            2,
		Columns:         2,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		data:            []byte{0x10, 0x20, 0x30, 0x40},
	}

	c := grayFrame.At(0, 0)
	expected := color.Gray{Y: 0x10}
	if c != expected {
		t.Errorf("expected %v, got %v", expected, c)
	}

	// Test out of bounds
	c = grayFrame.At(10, 10)
	if c != (color.RGBA{}) {
		t.Errorf("expected zero color for out of bounds, got %v", c)
	}

	// Test RGB frame
	rgbFrame := Frame{
		Rows:            2,
		Columns:         2,
		BitsAllocated:   8,
		SamplesPerPixel: 3,
		data:            []byte{255, 0, 0, 0, 255, 0, 0, 0, 255, 128, 128, 128},
	}

	c = rgbFrame.At(0, 0)
	expectedRGB := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	if c != expectedRGB {
		t.Errorf("expected %v, got %v", expectedRGB, c)
	}
}

func TestPixelData_String(t *testing.T) {
	pd := &PixelData{
		Rows:                      512,
		Columns:                   512,
		SamplesPerPixel:           1,
		BitsStored:                16,
		PhotometricInterpretation: "MONOCHROME2",
		NumberOfFrames:            1,
	}

	s := pd.String()
	expected := "PixelData{512x512x1, 16 bits, MONOCHROME2, 1 frames}"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestFrame_String(t *testing.T) {
	frame := Frame{
		Index:      5,
		Rows:       256,
		Columns:    256,
		BitsStored: 8,
	}

	s := frame.String()
	expected := "Frame{5: 256x256, 8 bits}"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}
