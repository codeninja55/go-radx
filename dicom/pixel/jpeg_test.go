package pixel

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestJPEGBaselineDecoder_TransferSyntaxUID(t *testing.T) {
	decoder := NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.50")
	expected := "1.2.840.10008.1.2.4.50"
	if decoder.TransferSyntaxUID() != expected {
		t.Errorf("expected UID %s, got %s", expected, decoder.TransferSyntaxUID())
	}
}

func TestJPEGBaselineDecoder_Decode_Grayscale(t *testing.T) {
	// Create a simple 8x8 grayscale test image
	width := 8
	height := 8
	img := image.NewGray(image.Rect(0, 0, width, height))

	// Fill with gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gray := uint8((x + y) * 16)
			img.SetGray(x, y, color.Gray{Y: gray})
		}
	}

	// Encode as JPEG
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	if err != nil {
		t.Fatalf("failed to encode test JPEG: %v", err)
	}

	// Create decoder
	decoder := NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.50")

	// Create PixelInfo
	info := &PixelInfo{
		Rows:            uint16(height),
		Columns:         uint16(width),
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	// Decode
	result, err := decoder.Decode(buf.Bytes(), info)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Verify size
	expectedSize := width * height
	if len(result) != expectedSize {
		t.Errorf("expected %d bytes, got %d", expectedSize, len(result))
	}

	// Note: We can't do exact pixel comparison due to JPEG lossy compression,
	// but we can verify the result is reasonable
	if result[0] == 0 && result[len(result)-1] == 0 {
		t.Error("decoded pixels appear to be all zeros (unexpected)")
	}
}

func TestJPEGBaselineDecoder_Decode_RGB(t *testing.T) {
	// Create a simple 8x8 RGB test image
	width := 8
	height := 8
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with color pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8(x * 32)
			g := uint8(y * 32)
			b := uint8((x + y) * 16)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// Encode as JPEG
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	if err != nil {
		t.Fatalf("failed to encode test JPEG: %v", err)
	}

	// Create decoder
	decoder := NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.50")

	// Create PixelInfo
	info := &PixelInfo{
		Rows:            uint16(height),
		Columns:         uint16(width),
		BitsAllocated:   8,
		SamplesPerPixel: 3,
		NumberOfFrames:  1,
	}

	// Decode
	result, err := decoder.Decode(buf.Bytes(), info)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	// Verify size
	expectedSize := width * height * 3
	if len(result) != expectedSize {
		t.Errorf("expected %d bytes, got %d", expectedSize, len(result))
	}

	// Verify we have RGB data (not all zeros)
	hasNonZero := false
	for _, b := range result {
		if b != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("decoded RGB pixels appear to be all zeros (unexpected)")
	}
}

func TestJPEGBaselineDecoder_Decode_EmptyData(t *testing.T) {
	decoder := NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.50")

	info := &PixelInfo{
		Rows:            8,
		Columns:         8,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	_, err := decoder.Decode([]byte{}, info)
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestJPEGBaselineDecoder_Decode_InvalidJPEG(t *testing.T) {
	decoder := NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.50")

	info := &PixelInfo{
		Rows:            8,
		Columns:         8,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	// Try to decode garbage data as JPEG
	invalidData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	_, err := decoder.Decode(invalidData, info)
	if err == nil {
		t.Error("expected error for invalid JPEG data, got nil")
	}
}

func TestJPEGBaselineDecoder_Decode_DimensionMismatch(t *testing.T) {
	// Create 8x8 image
	width := 8
	height := 8
	img := image.NewGray(image.Rect(0, 0, width, height))

	// Encode as JPEG
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	if err != nil {
		t.Fatalf("failed to encode test JPEG: %v", err)
	}

	decoder := NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.50")

	// Create PixelInfo with wrong dimensions
	info := &PixelInfo{
		Rows:            16, // Wrong!
		Columns:         16, // Wrong!
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	_, err = decoder.Decode(buf.Bytes(), info)
	if err == nil {
		t.Error("expected error for dimension mismatch, got nil")
	}
}

func TestJPEGBaselineDecoder_Decode_SamplesPerPixelMismatch(t *testing.T) {
	// Create grayscale image
	width := 8
	height := 8
	img := image.NewGray(image.Rect(0, 0, width, height))

	// Encode as JPEG
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	if err != nil {
		t.Fatalf("failed to encode test JPEG: %v", err)
	}

	decoder := NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.50")

	// Create PixelInfo expecting RGB (SamplesPerPixel=3) but image is grayscale
	info := &PixelInfo{
		Rows:            uint16(height),
		Columns:         uint16(width),
		BitsAllocated:   8,
		SamplesPerPixel: 3, // Wrong!
		NumberOfFrames:  1,
	}

	_, err = decoder.Decode(buf.Bytes(), info)
	if err == nil {
		t.Error("expected error for SamplesPerPixel mismatch, got nil")
	}
}

func TestYcbcrToRGB(t *testing.T) {
	// Create a small YCbCr test image
	width := 4
	height := 4
	ycbcr := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio444)

	// Set some known YCbCr values
	// Pure white: Y=255, Cb=128, Cr=128
	for i := 0; i < width*height; i++ {
		ycbcr.Y[i] = 255
		ycbcr.Cb[i] = 128
		ycbcr.Cr[i] = 128
	}

	// Convert to RGB
	rgb := ycbcrToRGB(ycbcr)

	// Verify output size
	expectedSize := width * height * 3
	if len(rgb) != expectedSize {
		t.Errorf("expected %d bytes, got %d", expectedSize, len(rgb))
	}

	// For white (Y=255, Cb=128, Cr=128), RGB should be close to (255, 255, 255)
	// Due to conversion rounding, we allow small tolerance
	for i := 0; i < len(rgb); i++ {
		if rgb[i] < 250 {
			t.Errorf("expected white pixel value ~255, got %d at index %d", rgb[i], i)
			break
		}
	}
}

func TestRgbaToRGB(t *testing.T) {
	// Create RGBA image
	width := 4
	height := 4
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with known RGB values
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			rgba.SetRGBA(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}

	// Convert to RGB
	rgb := rgbaToRGB(rgba)

	// Verify output size
	expectedSize := width * height * 3
	if len(rgb) != expectedSize {
		t.Errorf("expected %d bytes, got %d", expectedSize, len(rgb))
	}

	// Verify RGB values (alpha should be stripped)
	for i := 0; i < width*height; i++ {
		idx := i * 3
		if rgb[idx] != 100 {
			t.Errorf("expected R=100, got %d at pixel %d", rgb[idx], i)
		}
		if rgb[idx+1] != 150 {
			t.Errorf("expected G=150, got %d at pixel %d", rgb[idx+1], i)
		}
		if rgb[idx+2] != 200 {
			t.Errorf("expected B=200, got %d at pixel %d", rgb[idx+2], i)
		}
	}
}

func TestNrgbaToRGB(t *testing.T) {
	// Create NRGBA image
	width := 4
	height := 4
	nrgba := image.NewNRGBA(image.Rect(0, 0, width, height))

	// Fill with known RGB values
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			nrgba.SetNRGBA(x, y, color.NRGBA{R: 50, G: 100, B: 150, A: 255})
		}
	}

	// Convert to RGB
	rgb := nrgbaToRGB(nrgba)

	// Verify output size
	expectedSize := width * height * 3
	if len(rgb) != expectedSize {
		t.Errorf("expected %d bytes, got %d", expectedSize, len(rgb))
	}

	// Verify RGB values (alpha should be stripped)
	for i := 0; i < width*height; i++ {
		idx := i * 3
		if rgb[idx] != 50 {
			t.Errorf("expected R=50, got %d at pixel %d", rgb[idx], i)
		}
		if rgb[idx+1] != 100 {
			t.Errorf("expected G=100, got %d at pixel %d", rgb[idx+1], i)
		}
		if rgb[idx+2] != 150 {
			t.Errorf("expected B=150, got %d at pixel %d", rgb[idx+2], i)
		}
	}
}

func TestClampUint8(t *testing.T) {
	tests := []struct {
		input    int32
		expected uint8
	}{
		{-100, 0},
		{-1, 0},
		{0, 0},
		{128, 128},
		{255, 255},
		{256, 255},
		{1000, 255},
	}

	for _, tt := range tests {
		result := clampUint8(tt.input)
		if result != tt.expected {
			t.Errorf("clampUint8(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestJPEGBaselineDecoder_RegisteredInInit(t *testing.T) {
	// Verify JPEG Baseline decoders are registered
	uids := []string{
		"1.2.840.10008.1.2.4.50", // JPEG Baseline Process 1
		"1.2.840.10008.1.2.4.51", // JPEG Baseline Processes 2 & 4
	}

	for _, uid := range uids {
		decoder, err := GetDecoder(uid)
		if err != nil {
			t.Errorf("expected decoder to be registered for %s, got error: %v", uid, err)
		}
		if decoder.TransferSyntaxUID() != uid {
			t.Errorf("expected UID %s, got %s", uid, decoder.TransferSyntaxUID())
		}
	}
}
