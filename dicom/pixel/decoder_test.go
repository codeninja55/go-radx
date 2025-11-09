package pixel

import (
	"testing"
)

// Test decoder registry operations
func TestRegisterDecoder(t *testing.T) {
	// Test registering a new decoder
	testUID := "1.2.3.4.5.6.7.8.9"
	mockDecoder := &MockDecoder{uid: testUID}

	// Clean up after test
	defer UnregisterDecoder(testUID)

	RegisterDecoder(testUID, mockDecoder)

	// Verify decoder can be retrieved
	decoder, err := GetDecoder(testUID)
	if err != nil {
		t.Fatalf("expected decoder to be registered, got error: %v", err)
	}

	if decoder.TransferSyntaxUID() != testUID {
		t.Errorf("expected UID %s, got %s", testUID, decoder.TransferSyntaxUID())
	}
}

func TestGetDecoder(t *testing.T) {
	tests := []struct {
		name      string
		uid       string
		shouldErr bool
	}{
		{
			name:      "native decoder - implicit VR little endian",
			uid:       "1.2.840.10008.1.2",
			shouldErr: false,
		},
		{
			name:      "native decoder - explicit VR little endian",
			uid:       "1.2.840.10008.1.2.1",
			shouldErr: false,
		},
		{
			name:      "RLE decoder",
			uid:       "1.2.840.10008.1.2.5",
			shouldErr: false,
		},
		{
			name:      "unsupported UID",
			uid:       "1.2.3.4.5.6.7.8.9.10",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder, err := GetDecoder(tt.uid)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error for UID %s, got nil", tt.uid)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if decoder == nil {
				t.Fatal("expected decoder, got nil")
			}
		})
	}
}

func TestUnregisterDecoder(t *testing.T) {
	// Register a test decoder
	testUID := "1.2.3.4.5.6.7.8.9.11"
	mockDecoder := &MockDecoder{uid: testUID}

	RegisterDecoder(testUID, mockDecoder)

	// Verify it's registered
	_, err := GetDecoder(testUID)
	if err != nil {
		t.Fatalf("expected decoder to be registered, got error: %v", err)
	}

	// Unregister it
	UnregisterDecoder(testUID)

	// Verify it's no longer registered
	_, err = GetDecoder(testUID)
	if err == nil {
		t.Error("expected error after unregistering, got nil")
	}
}

func TestListDecoders(t *testing.T) {
	// Get initial list
	initialDecoders := ListDecoders()

	// Should at least have native decoders registered
	if len(initialDecoders) < 4 {
		t.Errorf("expected at least 4 decoders (native + RLE), got %d", len(initialDecoders))
	}

	// Verify native decoders are present
	expectedUIDs := map[string]bool{
		"1.2.840.10008.1.2":      false, // Implicit VR Little Endian
		"1.2.840.10008.1.2.1":    false, // Explicit VR Little Endian
		"1.2.840.10008.1.2.2":    false, // Explicit VR Big Endian
		"1.2.840.10008.1.2.1.99": false, // Deflated Explicit VR Little Endian
		"1.2.840.10008.1.2.5":    false, // RLE Lossless
	}

	for _, uid := range initialDecoders {
		if _, ok := expectedUIDs[uid]; ok {
			expectedUIDs[uid] = true
		}
	}

	for uid, found := range expectedUIDs {
		if !found {
			t.Errorf("expected UID %s in decoder list", uid)
		}
	}
}

func TestNativeDecoder(t *testing.T) {
	decoder := &NativeDecoder{}

	// Test TransferSyntaxUID
	uid := decoder.TransferSyntaxUID()
	if uid != "" {
		t.Errorf("expected empty UID for native decoder, got %s", uid)
	}

	// Test Decode (should return data unchanged)
	testData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	info := &PixelInfo{
		Rows:    1,
		Columns: 5,
	}

	result, err := decoder.Decode(testData, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != len(testData) {
		t.Errorf("expected result length %d, got %d", len(testData), len(result))
	}

	for i := range result {
		if result[i] != testData[i] {
			t.Errorf("expected byte %d at index %d, got %d", testData[i], i, result[i])
		}
	}
}

func TestCalculateExpectedSize(t *testing.T) {
	tests := []struct {
		name     string
		info     *PixelInfo
		expected int
	}{
		{
			name: "8-bit grayscale 512x512 single frame",
			info: &PixelInfo{
				Rows:            512,
				Columns:         512,
				BitsAllocated:   8,
				SamplesPerPixel: 1,
				NumberOfFrames:  1,
			},
			expected: 512 * 512 * 1 * 1 * 1, // 262144 bytes
		},
		{
			name: "16-bit grayscale 256x256 single frame",
			info: &PixelInfo{
				Rows:            256,
				Columns:         256,
				BitsAllocated:   16,
				SamplesPerPixel: 1,
				NumberOfFrames:  1,
			},
			expected: 256 * 256 * 1 * 1 * 2, // 131072 bytes
		},
		{
			name: "8-bit RGB 100x100 single frame",
			info: &PixelInfo{
				Rows:            100,
				Columns:         100,
				BitsAllocated:   8,
				SamplesPerPixel: 3,
				NumberOfFrames:  1,
			},
			expected: 100 * 100 * 3 * 1 * 1, // 30000 bytes
		},
		{
			name: "16-bit grayscale 512x512 10 frames",
			info: &PixelInfo{
				Rows:            512,
				Columns:         512,
				BitsAllocated:   16,
				SamplesPerPixel: 1,
				NumberOfFrames:  10,
			},
			expected: 512 * 512 * 1 * 10 * 2, // 5242880 bytes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateExpectedSize(tt.info)
			if result != tt.expected {
				t.Errorf("expected size %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestValidatePixelData(t *testing.T) {
	tests := []struct {
		name      string
		dataSize  int
		info      *PixelInfo
		shouldErr bool
	}{
		{
			name:     "valid 8-bit grayscale",
			dataSize: 512 * 512,
			info: &PixelInfo{
				Rows:            512,
				Columns:         512,
				BitsAllocated:   8,
				SamplesPerPixel: 1,
				NumberOfFrames:  1,
			},
			shouldErr: false,
		},
		{
			name:     "invalid size - too small",
			dataSize: 100,
			info: &PixelInfo{
				Rows:            512,
				Columns:         512,
				BitsAllocated:   8,
				SamplesPerPixel: 1,
				NumberOfFrames:  1,
			},
			shouldErr: true,
		},
		{
			name:     "invalid size - too large",
			dataSize: 512 * 512 * 2,
			info: &PixelInfo{
				Rows:            512,
				Columns:         512,
				BitsAllocated:   8,
				SamplesPerPixel: 1,
				NumberOfFrames:  1,
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.dataSize)
			err := ValidatePixelData(data, tt.info)

			if tt.shouldErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// MockDecoder is a simple mock for testing decoder registration
type MockDecoder struct {
	uid string
}

func (m *MockDecoder) Decode(encapsulated []byte, info *PixelInfo) ([]byte, error) {
	return encapsulated, nil
}

func (m *MockDecoder) TransferSyntaxUID() string {
	return m.uid
}
