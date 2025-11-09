//go:build cgo
// +build cgo

package pixel

import (
	"testing"
)

func TestJPEGLosslessDecoder_CGoBuild(t *testing.T) {
	// This test verifies that the CGo build works and decoders are registered
	decoder := NewJPEGLosslessDecoder("1.2.840.10008.1.2.4.70")

	if decoder == nil {
		t.Fatal("expected decoder, got nil")
	}

	if decoder.TransferSyntaxUID() != "1.2.840.10008.1.2.4.70" {
		t.Errorf("expected UID 1.2.840.10008.1.2.4.70, got %s", decoder.TransferSyntaxUID())
	}
}

func TestJPEGLosslessDecoder_RegisteredInInit(t *testing.T) {
	// Verify JPEG Lossless decoders are registered when CGo is enabled
	uids := []string{
		"1.2.840.10008.1.2.4.57", // JPEG Lossless First-Order Prediction
		"1.2.840.10008.1.2.4.70", // JPEG Lossless Process 14
	}

	for _, uid := range uids {
		decoder, err := GetDecoder(uid)
		if err != nil {
			t.Errorf("expected decoder to be registered for %s (CGo build), got error: %v", uid, err)
		}
		if decoder.TransferSyntaxUID() != uid {
			t.Errorf("expected UID %s, got %s", uid, decoder.TransferSyntaxUID())
		}
	}
}

func TestJPEGLosslessDecoder_Decode_EmptyData(t *testing.T) {
	decoder := NewJPEGLosslessDecoder("1.2.840.10008.1.2.4.70")

	info := &PixelInfo{
		Rows:            512,
		Columns:         512,
		BitsAllocated:   16,
		BitsStored:      12,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	_, err := decoder.Decode([]byte{}, info)
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestJPEGLosslessDecoder_Decode_InvalidData(t *testing.T) {
	decoder := NewJPEGLosslessDecoder("1.2.840.10008.1.2.4.70")

	info := &PixelInfo{
		Rows:            512,
		Columns:         512,
		BitsAllocated:   16,
		BitsStored:      12,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	// Try to decode garbage data
	invalidData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	_, err := decoder.Decode(invalidData, info)
	if err == nil {
		t.Error("expected error for invalid JPEG data, got nil")
	}

	// Verify error is a DecompressionError
	var decompressionErr *DecompressionError
	if !isDecompressionError(err) {
		t.Errorf("expected DecompressionError, got %T", err)
	}
	_ = decompressionErr // Use variable to avoid unused warning
}

func TestJPEGLosslessDecoder_MemorySafety(t *testing.T) {
	// Test that CGo memory management doesn't leak
	// Run multiple decompression attempts with invalid data
	decoder := NewJPEGLosslessDecoder("1.2.840.10008.1.2.4.70")

	info := &PixelInfo{
		Rows:            512,
		Columns:         512,
		BitsAllocated:   16,
		BitsStored:      12,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	// Run multiple times to check for memory leaks
	for i := 0; i < 100; i++ {
		invalidData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
		_, err := decoder.Decode(invalidData, info)
		if err == nil {
			t.Error("expected error for invalid data")
		}
	}

	// If this test completes without crashing or hanging, memory management is likely correct
}

// Helper function to check if error is DecompressionError
func isDecompressionError(err error) bool {
	_, ok := err.(*DecompressionError)
	return ok
}
