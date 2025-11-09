// Package dicom provides DICOM file parsing and manipulation.
package dicom

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReader_ReadUint16_LittleEndian tests reading 16-bit unsigned integers in little endian.
func TestReader_ReadUint16_LittleEndian(t *testing.T) {
	// Setup: Create a buffer with little endian uint16 values
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16(0x1234))
	binary.Write(buf, binary.LittleEndian, uint16(0xABCD))

	// Create reader with little endian
	reader := NewReader(buf, binary.LittleEndian)

	// Read first value
	val1, err := reader.ReadUint16()
	require.NoError(t, err)
	assert.Equal(t, uint16(0x1234), val1)

	// Read second value
	val2, err := reader.ReadUint16()
	require.NoError(t, err)
	assert.Equal(t, uint16(0xABCD), val2)

	// Reading past EOF should return error
	_, err = reader.ReadUint16()
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
}

// TestReader_ReadUint16_BigEndian tests reading 16-bit unsigned integers in big endian.
func TestReader_ReadUint16_BigEndian(t *testing.T) {
	// Setup: Create a buffer with big endian uint16 values
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint16(0x1234))
	binary.Write(buf, binary.BigEndian, uint16(0xABCD))

	// Create reader with big endian
	reader := NewReader(buf, binary.BigEndian)

	// Read first value
	val1, err := reader.ReadUint16()
	require.NoError(t, err)
	assert.Equal(t, uint16(0x1234), val1)

	// Read second value
	val2, err := reader.ReadUint16()
	require.NoError(t, err)
	assert.Equal(t, uint16(0xABCD), val2)
}

// TestReader_ReadUint32_LittleEndian tests reading 32-bit unsigned integers in little endian.
func TestReader_ReadUint32_LittleEndian(t *testing.T) {
	// Setup: Create a buffer with little endian uint32 values
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(0x12345678))
	binary.Write(buf, binary.LittleEndian, uint32(0xABCDEF00))

	// Create reader with little endian
	reader := NewReader(buf, binary.LittleEndian)

	// Read first value
	val1, err := reader.ReadUint32()
	require.NoError(t, err)
	assert.Equal(t, uint32(0x12345678), val1)

	// Read second value
	val2, err := reader.ReadUint32()
	require.NoError(t, err)
	assert.Equal(t, uint32(0xABCDEF00), val2)

	// Reading past EOF should return error
	_, err = reader.ReadUint32()
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
}

// TestReader_ReadUint32_BigEndian tests reading 32-bit unsigned integers in big endian.
func TestReader_ReadUint32_BigEndian(t *testing.T) {
	// Setup: Create a buffer with big endian uint32 values
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(0x12345678))
	binary.Write(buf, binary.BigEndian, uint32(0xABCDEF00))

	// Create reader with big endian
	reader := NewReader(buf, binary.BigEndian)

	// Read first value
	val1, err := reader.ReadUint32()
	require.NoError(t, err)
	assert.Equal(t, uint32(0x12345678), val1)

	// Read second value
	val2, err := reader.ReadUint32()
	require.NoError(t, err)
	assert.Equal(t, uint32(0xABCDEF00), val2)
}

// TestReader_ReadBytes tests reading exact byte sequences.
func TestReader_ReadBytes(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		readSize int
		expected []byte
		wantErr  bool
	}{
		{
			name:     "read 4 bytes",
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
			readSize: 4,
			expected: []byte{0x01, 0x02, 0x03, 0x04},
			wantErr:  false,
		},
		{
			name:     "read exact length",
			data:     []byte{0xAA, 0xBB, 0xCC},
			readSize: 3,
			expected: []byte{0xAA, 0xBB, 0xCC},
			wantErr:  false,
		},
		{
			name:     "read zero bytes",
			data:     []byte{0x01, 0x02},
			readSize: 0,
			expected: []byte{},
			wantErr:  false,
		},
		{
			name:     "read past EOF",
			data:     []byte{0x01, 0x02},
			readSize: 10,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tc.data)
			reader := NewReader(buf, binary.LittleEndian)

			result, err := reader.ReadBytes(tc.readSize)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestReader_ReadString tests reading string data.
func TestReader_ReadString(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		length   int
		expected string
		wantErr  bool
	}{
		{
			name:     "read ASCII string",
			data:     []byte("HELLO WORLD"),
			length:   11,
			expected: "HELLO WORLD",
			wantErr:  false,
		},
		{
			name:     "read string with null terminator",
			data:     []byte("HELLO\x00WORLD"),
			length:   11,
			expected: "HELLO\x00WORLD",
			wantErr:  false,
		},
		{
			name:     "read string with trailing space",
			data:     []byte("TEST    "),
			length:   8,
			expected: "TEST    ",
			wantErr:  false,
		},
		{
			name:     "read empty string",
			data:     []byte{},
			length:   0,
			expected: "",
			wantErr:  false,
		},
		{
			name:     "read past EOF",
			data:     []byte("SHORT"),
			length:   10,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tc.data)
			reader := NewReader(buf, binary.LittleEndian)

			result, err := reader.ReadString(tc.length)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestReader_SetByteOrder tests changing byte order dynamically.
func TestReader_SetByteOrder(t *testing.T) {
	// Setup: Create a buffer with mixed endian data
	buf := new(bytes.Buffer)
	// Little endian value
	binary.Write(buf, binary.LittleEndian, uint16(0x1234))
	// Big endian value
	binary.Write(buf, binary.BigEndian, uint16(0x5678))

	// Start with little endian
	reader := NewReader(buf, binary.LittleEndian)

	// Read first value as little endian
	val1, err := reader.ReadUint16()
	require.NoError(t, err)
	assert.Equal(t, uint16(0x1234), val1)

	// Switch to big endian
	reader.SetByteOrder(binary.BigEndian)

	// Read second value as big endian
	val2, err := reader.ReadUint16()
	require.NoError(t, err)
	assert.Equal(t, uint16(0x5678), val2)
}

// TestReader_Sequential tests sequential mixed reads.
func TestReader_Sequential(t *testing.T) {
	// Setup: Create a buffer with mixed data types
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16(0x1234))     // 2 bytes
	buf.Write([]byte("TEST"))                                  // 4 bytes
	binary.Write(buf, binary.LittleEndian, uint32(0xABCDEF00)) // 4 bytes

	reader := NewReader(buf, binary.LittleEndian)

	// Read uint16
	val1, err := reader.ReadUint16()
	require.NoError(t, err)
	assert.Equal(t, uint16(0x1234), val1)

	// Read string
	str, err := reader.ReadString(4)
	require.NoError(t, err)
	assert.Equal(t, "TEST", str)

	// Read uint32
	val2, err := reader.ReadUint32()
	require.NoError(t, err)
	assert.Equal(t, uint32(0xABCDEF00), val2)
}

// TestReader_EmptyReader tests reading from empty reader.
func TestReader_EmptyReader(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	reader := NewReader(buf, binary.LittleEndian)

	// All reads should return EOF
	_, err := reader.ReadUint16()
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)

	_, err = reader.ReadUint32()
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)

	_, err = reader.ReadBytes(1)
	assert.Error(t, err)

	str, err := reader.ReadString(1)
	assert.Error(t, err)
	assert.Empty(t, str)
}
