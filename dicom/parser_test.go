// Package dicom provides DICOM file parsing and manipulation.
package dicom

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParser_ReadPreamble_Valid tests reading a valid DICOM preamble.
func TestParser_ReadPreamble_Valid(t *testing.T) {
	// Setup: Create a buffer with valid DICOM preamble
	buf := new(bytes.Buffer)

	// Write 128-byte preamble (null bytes)
	preamble := make([]byte, 128)
	buf.Write(preamble)

	// Write "DICM" prefix
	buf.WriteString("DICM")

	// Create parser
	reader := NewReader(buf, binary.LittleEndian)
	parser := &Parser{reader: reader}

	// Read preamble - should succeed
	err := parser.readPreamble()
	require.NoError(t, err)
}

// TestParser_ReadPreamble_ValidWithNonNullPreamble tests preamble with non-null bytes.
func TestParser_ReadPreamble_ValidWithNonNullPreamble(t *testing.T) {
	// Setup: Create a buffer with valid DICOM preamble containing application data
	buf := new(bytes.Buffer)

	// Write 128-byte preamble with some application-specific data
	preamble := make([]byte, 128)
	copy(preamble, []byte("APPLICATION DATA"))
	buf.Write(preamble)

	// Write "DICM" prefix
	buf.WriteString("DICM")

	// Create parser
	reader := NewReader(buf, binary.LittleEndian)
	parser := &Parser{reader: reader}

	// Read preamble - should succeed (preamble content doesn't matter)
	err := parser.readPreamble()
	require.NoError(t, err)
}

// TestParser_ReadPreamble_InvalidPrefix tests reading with invalid DICM prefix.
func TestParser_ReadPreamble_InvalidPrefix(t *testing.T) {
	testCases := []struct {
		name   string
		prefix string
	}{
		{
			name:   "wrong prefix DICOM",
			prefix: "DICOM", // 5 chars instead of 4
		},
		{
			name:   "wrong prefix ABCD",
			prefix: "ABCD",
		},
		{
			name:   "lowercase dicm",
			prefix: "dicm",
		},
		{
			name:   "empty prefix",
			prefix: "",
		},
		{
			name:   "partial prefix DIC",
			prefix: "DIC",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup: Create buffer with invalid prefix
			buf := new(bytes.Buffer)

			// Write 128-byte preamble
			preamble := make([]byte, 128)
			buf.Write(preamble)

			// Write invalid prefix
			if tc.prefix != "" {
				buf.WriteString(tc.prefix)
			}

			// Create parser
			reader := NewReader(buf, binary.LittleEndian)
			parser := &Parser{reader: reader}

			// Read preamble - should fail
			err := parser.readPreamble()
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidPreamble)
		})
	}
}

// TestParser_ReadPreamble_Truncated tests reading truncated preamble.
func TestParser_ReadPreamble_Truncated(t *testing.T) {
	testCases := []struct {
		name       string
		dataLength int // total bytes to write (should be 132 for valid)
	}{
		{
			name:       "no data",
			dataLength: 0,
		},
		{
			name:       "only 64 bytes",
			dataLength: 64,
		},
		{
			name:       "preamble only (128 bytes)",
			dataLength: 128,
		},
		{
			name:       "preamble + 1 byte",
			dataLength: 129,
		},
		{
			name:       "preamble + 2 bytes",
			dataLength: 130,
		},
		{
			name:       "preamble + 3 bytes",
			dataLength: 131,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup: Create buffer with truncated data
			buf := new(bytes.Buffer)
			data := make([]byte, tc.dataLength)
			if tc.dataLength > 128 {
				// Add partial "DICM" at the end
				copy(data[128:], "DICM"[:tc.dataLength-128])
			}
			buf.Write(data)

			// Create parser
			reader := NewReader(buf, binary.LittleEndian)
			parser := &Parser{reader: reader}

			// Read preamble - should fail
			err := parser.readPreamble()
			assert.Error(t, err)
		})
	}
}

// TestParseFile_RealDICOM tests parsing a real DICOM file from testdata.
func TestParseFile_RealDICOM(t *testing.T) {
	// Find a test DICOM file
	testFile := filepath.Join("../../testdata", "1.2.36.1.2001.1005.78.60.127832058365991103", "1.2.36.1.2001.1005.78.60.127832058365991103.1", "1.2.36.1.2001.1005.78.60.127832058365991103.1.1.dcm")

	// Check if file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test DICOM file not found, skipping real file test")
	}

	// Parse file
	ds, err := ParseFile(testFile)

	// This will fail initially (RED phase) but should pass once implemented
	require.NoError(t, err, "Failed to parse DICOM file")
	require.NotNil(t, ds)
	assert.Greater(t, ds.Len(), 0, "Dataset should not be empty")
}

// TestParseFile_NonExistent tests parsing a non-existent file.
func TestParseFile_NonExistent(t *testing.T) {
	_, err := ParseFile("/nonexistent/file.dcm")
	assert.Error(t, err)
}

// TestParseFile_NotDICOM tests parsing a non-DICOM file.
func TestParseFile_NotDICOM(t *testing.T) {
	// Create a temporary non-DICOM file
	tmpFile := filepath.Join(t.TempDir(), "not_dicom.txt")
	err := os.WriteFile(tmpFile, []byte("This is not a DICOM file"), 0644)
	require.NoError(t, err)

	// Try to parse it
	_, err = ParseFile(tmpFile)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPreamble)
}

// TestParser_Integration tests full parser workflow with minimal DICOM structure.
func TestParser_Integration(t *testing.T) {
	// Create a minimal valid DICOM file structure in memory
	buf := new(bytes.Buffer)

	// 1. Write preamble (128 bytes)
	preamble := make([]byte, 128)
	buf.Write(preamble)

	// 2. Write "DICM" prefix
	buf.WriteString("DICM")

	// 3. Write File Meta Information Group Length (0002,0000) UL = 4
	// Tag: (0002,0000)
	binary.Write(buf, binary.LittleEndian, uint16(0x0002)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // element
	buf.WriteString("UL")                                  // VR
	binary.Write(buf, binary.LittleEndian, uint16(4))      // length
	binary.Write(buf, binary.LittleEndian, uint32(0))      // value (placeholder)

	// For now, we'll test that the parser at least reads the preamble correctly
	reader := NewReader(buf, binary.LittleEndian)
	parser := &Parser{reader: reader}

	err := parser.readPreamble()
	require.NoError(t, err)

	// Further integration testing will be added as we implement more functionality
}
