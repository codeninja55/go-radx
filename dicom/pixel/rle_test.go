package pixel

import (
	"encoding/binary"
	"testing"
)

func TestRLEDecoder_TransferSyntaxUID(t *testing.T) {
	decoder := &RLEDecoder{}
	expected := "1.2.840.10008.1.2.5"
	if decoder.TransferSyntaxUID() != expected {
		t.Errorf("expected UID %s, got %s", expected, decoder.TransferSyntaxUID())
	}
}

func TestRLEDecoder_Decode_InvalidHeader(t *testing.T) {
	decoder := &RLEDecoder{}
	info := &PixelInfo{
		Rows:            10,
		Columns:         10,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	// Test with data too small (< 64 bytes)
	smallData := make([]byte, 32)
	_, err := decoder.Decode(smallData, info)
	if err == nil {
		t.Error("expected error for data < 64 bytes, got nil")
	}
}

func TestRLEDecoder_Decode_InvalidNumSegments(t *testing.T) {
	decoder := &RLEDecoder{}
	info := &PixelInfo{
		Rows:            10,
		Columns:         10,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	tests := []struct {
		name        string
		numSegments uint32
	}{
		{"zero segments", 0},
		{"too many segments", 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create RLE header with invalid number of segments
			header := make([]byte, 64)
			binary.LittleEndian.PutUint32(header[0:4], tt.numSegments)

			_, err := decoder.Decode(header, info)
			if err == nil {
				t.Errorf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestDecodePackBits(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
		wantErr  bool
	}{
		{
			name:     "literal run - single byte",
			input:    []byte{0x00, 0xAA}, // Copy 1 byte (0+1)
			expected: []byte{0xAA},
			wantErr:  false,
		},
		{
			name:     "literal run - multiple bytes",
			input:    []byte{0x02, 0x01, 0x02, 0x03}, // Copy 3 bytes (2+1)
			expected: []byte{0x01, 0x02, 0x03},
			wantErr:  false,
		},
		{
			name:     "repeat run - single byte",
			input:    []byte{0xFF, 0xAA}, // Repeat 0xAA 2 times (1 - (-1))
			expected: []byte{0xAA, 0xAA},
			wantErr:  false,
		},
		{
			name:     "repeat run - multiple bytes",
			input:    []byte{0xFD, 0xBB}, // Repeat 0xBB 4 times (1 - (-3))
			expected: []byte{0xBB, 0xBB, 0xBB, 0xBB},
			wantErr:  false,
		},
		{
			name:     "no-op control byte",
			input:    []byte{0x80}, // No operation
			expected: []byte{},
			wantErr:  false,
		},
		{
			name:     "mixed operations",
			input:    []byte{0x01, 0x01, 0x02, 0xFE, 0xFF, 0x00, 0xAA}, // Literal(2) + Repeat(3) + Literal(1)
			expected: []byte{0x01, 0x02, 0xFF, 0xFF, 0xFF, 0xAA},
			wantErr:  false,
		},
		{
			name:     "literal run beyond data",
			input:    []byte{0x05, 0x01, 0x02}, // Wants 6 bytes but only 2 available
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "repeat run missing data",
			input:    []byte{0xFD}, // Wants repeat byte but none available
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodePackBits(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("at index %d: expected 0x%02X, got 0x%02X", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestRLEDecoder_Decode_8BitGrayscale(t *testing.T) {
	decoder := &RLEDecoder{}

	// Create a simple 4x4 8-bit grayscale image
	// Each pixel is one byte, so we have 16 bytes total
	// Use a simple pattern: 0x01, 0x02, 0x03, ... 0x10
	expectedData := []byte{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C,
		0x0D, 0x0E, 0x0F, 0x10,
	}

	// Encode this data as RLE (using literal runs for simplicity)
	// Segment 0: All 16 bytes
	segmentData := append([]byte{0x0F}, expectedData...) // Literal run of 16 bytes

	// Create RLE header
	header := make([]byte, 64)
	binary.LittleEndian.PutUint32(header[0:4], 1)  // 1 segment
	binary.LittleEndian.PutUint32(header[4:8], 64) // Segment 0 starts at byte 64

	// Combine header and segment data
	rleData := append(header, segmentData...)

	info := &PixelInfo{
		Rows:            4,
		Columns:         4,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	result, err := decoder.Decode(rleData, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != len(expectedData) {
		t.Errorf("expected length %d, got %d", len(expectedData), len(result))
		return
	}

	for i := range result {
		if result[i] != expectedData[i] {
			t.Errorf("at index %d: expected 0x%02X, got 0x%02X", i, expectedData[i], result[i])
		}
	}
}

func TestRLEDecoder_Decode_16BitGrayscale(t *testing.T) {
	decoder := &RLEDecoder{}

	// Create a simple 2x2 16-bit grayscale image
	// Each pixel is 2 bytes (little-endian), so we have 8 bytes total
	// Pixels: 0x0100, 0x0200, 0x0300, 0x0400
	// Raw bytes: 00 01 00 02 00 03 00 04
	expectedData := []byte{
		0x00, 0x01, // Pixel 0: 0x0100
		0x00, 0x02, // Pixel 1: 0x0200
		0x00, 0x03, // Pixel 2: 0x0300
		0x00, 0x04, // Pixel 3: 0x0400
	}

	// For 16-bit data, RLE uses 2 segments:
	// Segment 0: LSB (least significant byte): 00 00 00 00
	// Segment 1: MSB (most significant byte): 01 02 03 04

	// Segment 0 data (LSB - all zeros, use repeat run)
	segment0 := []byte{0xFC, 0x00} // Repeat 0x00 4 times (1 - (-3))

	// Segment 1 data (MSB - sequential, use literal run)
	segment1 := []byte{0x03, 0x01, 0x02, 0x03, 0x04} // Literal run of 4 bytes

	// Create RLE header
	header := make([]byte, 64)
	binary.LittleEndian.PutUint32(header[0:4], 2)                         // 2 segments
	binary.LittleEndian.PutUint32(header[4:8], 64)                        // Segment 0 starts at byte 64
	binary.LittleEndian.PutUint32(header[8:12], 64+uint32(len(segment0))) // Segment 1 starts after segment 0

	// Combine header and segment data
	rleData := append(header, segment0...)
	rleData = append(rleData, segment1...)

	info := &PixelInfo{
		Rows:            2,
		Columns:         2,
		BitsAllocated:   16,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	result, err := decoder.Decode(rleData, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != len(expectedData) {
		t.Errorf("expected length %d, got %d", len(expectedData), len(result))
		return
	}

	for i := range result {
		if result[i] != expectedData[i] {
			t.Errorf("at index %d: expected 0x%02X, got 0x%02X", i, expectedData[i], result[i])
		}
	}
}

func TestRLEDecoder_Decode_SegmentOffsetValidation(t *testing.T) {
	decoder := &RLEDecoder{}

	// Create header with invalid segment offsets
	header := make([]byte, 64)
	binary.LittleEndian.PutUint32(header[0:4], 2)      // 2 segments
	binary.LittleEndian.PutUint32(header[4:8], 10000)  // Invalid offset (beyond data)
	binary.LittleEndian.PutUint32(header[8:12], 20000) // Invalid offset

	info := &PixelInfo{
		Rows:            2,
		Columns:         2,
		BitsAllocated:   8,
		SamplesPerPixel: 1,
		NumberOfFrames:  1,
	}

	_, err := decoder.Decode(header, info)
	if err == nil {
		t.Error("expected error for invalid segment offsets, got nil")
	}
}
