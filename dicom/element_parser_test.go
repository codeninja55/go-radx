// Package dicom provides DICOM file parsing and manipulation.
package dicom

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/vr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestElementParser_ReadElement_ExplicitVR_UI tests parsing a UI element.
func TestElementParser_ReadElement_ExplicitVR_UI(t *testing.T) {
	// Setup: Create a buffer with a UI element
	// (0002,0010) UI Transfer Syntax UID = "1.2.840.10008.1.2.1" (Explicit VR Little Endian)
	buf := new(bytes.Buffer)

	// Tag: (0002,0010)
	binary.Write(buf, binary.LittleEndian, uint16(0x0002)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x0010)) // element

	// VR: UI (2 bytes)
	buf.WriteString("UI")

	// Length: 2 bytes for UI
	uidValue := "1.2.840.10008.1.2.1"
	binary.Write(buf, binary.LittleEndian, uint16(len(uidValue)))

	// Value
	buf.WriteString(uidValue)

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element
	elem, err := parser.ReadElement()
	require.NoError(t, err)
	require.NotNil(t, elem)

	// Verify tag
	assert.True(t, elem.Tag().Equals(tag.New(0x0002, 0x0010)))

	// Verify VR
	assert.Equal(t, vr.UniqueIdentifier, elem.VR())

	// Verify value
	assert.Equal(t, uidValue, elem.Value().String())
}

// TestElementParser_ReadElement_ExplicitVR_PN tests parsing a PN element.
func TestElementParser_ReadElement_ExplicitVR_PN(t *testing.T) {
	// Setup: Create a buffer with a PN element
	// (0010,0010) PN Patient's Name = "Doe^John"
	buf := new(bytes.Buffer)

	// Tag: (0010,0010)
	binary.Write(buf, binary.LittleEndian, uint16(0x0010)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x0010)) // element

	// VR: PN (2 bytes)
	buf.WriteString("PN")

	// Length: 2 bytes for PN
	pnValue := "Doe^John"
	binary.Write(buf, binary.LittleEndian, uint16(len(pnValue)))

	// Value
	buf.WriteString(pnValue)

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element
	elem, err := parser.ReadElement()
	require.NoError(t, err)
	require.NotNil(t, elem)

	// Verify tag
	assert.True(t, elem.Tag().Equals(tag.New(0x0010, 0x0010)))

	// Verify VR
	assert.Equal(t, vr.PersonName, elem.VR())

	// Verify value
	assert.Equal(t, pnValue, elem.Value().String())
}

// TestElementParser_ReadElement_ExplicitVR_US tests parsing a US element.
func TestElementParser_ReadElement_ExplicitVR_US(t *testing.T) {
	// Setup: Create a buffer with a US element
	// (0028,0010) US Rows = 512
	buf := new(bytes.Buffer)

	// Tag: (0028,0010)
	binary.Write(buf, binary.LittleEndian, uint16(0x0028)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x0010)) // element

	// VR: US (2 bytes)
	buf.WriteString("US")

	// Length: 2 bytes for US (value is 2 bytes)
	binary.Write(buf, binary.LittleEndian, uint16(2))

	// Value: uint16
	binary.Write(buf, binary.LittleEndian, uint16(512))

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element
	elem, err := parser.ReadElement()
	require.NoError(t, err)
	require.NotNil(t, elem)

	// Verify tag
	assert.True(t, elem.Tag().Equals(tag.New(0x0028, 0x0010)))

	// Verify VR
	assert.Equal(t, vr.UnsignedShort, elem.VR())

	// Verify value
	assert.Equal(t, "512", elem.Value().String())
}

// TestElementParser_ReadElement_ExplicitVR_UL tests parsing a UL element.
func TestElementParser_ReadElement_ExplicitVR_UL(t *testing.T) {
	// Setup: Create a buffer with a UL element
	// (0002,0000) UL File Meta Information Group Length = 192
	buf := new(bytes.Buffer)

	// Tag: (0002,0000)
	binary.Write(buf, binary.LittleEndian, uint16(0x0002)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // element

	// VR: UL (2 bytes)
	buf.WriteString("UL")

	// Length: 2 bytes for UL (value is 4 bytes)
	binary.Write(buf, binary.LittleEndian, uint16(4))

	// Value: uint32
	binary.Write(buf, binary.LittleEndian, uint32(192))

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element
	elem, err := parser.ReadElement()
	require.NoError(t, err)
	require.NotNil(t, elem)

	// Verify tag
	assert.True(t, elem.Tag().Equals(tag.New(0x0002, 0x0000)))

	// Verify VR
	assert.Equal(t, vr.UnsignedLong, elem.VR())

	// Verify value
	assert.Equal(t, "192", elem.Value().String())
}

// TestElementParser_ReadElement_ExplicitVR_OB tests parsing an OB element (32-bit length).
func TestElementParser_ReadElement_ExplicitVR_OB(t *testing.T) {
	// Setup: Create a buffer with an OB element
	// (0028,1200) OB Gray Lookup Table Data = [0x00, 0x01, 0x02, 0x03]
	buf := new(bytes.Buffer)

	// Tag: (0028,1200)
	binary.Write(buf, binary.LittleEndian, uint16(0x0028)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x1200)) // element

	// VR: OB (2 bytes)
	buf.WriteString("OB")

	// Reserved: 2 bytes (must be 0x0000)
	binary.Write(buf, binary.LittleEndian, uint16(0x0000))

	// Length: 4 bytes (uint32) for OB
	obData := []byte{0x00, 0x01, 0x02, 0x03}
	binary.Write(buf, binary.LittleEndian, uint32(len(obData)))

	// Value: binary data
	buf.Write(obData)

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element
	elem, err := parser.ReadElement()
	require.NoError(t, err)
	require.NotNil(t, elem)

	// Verify tag
	assert.True(t, elem.Tag().Equals(tag.New(0x0028, 0x1200)))

	// Verify VR
	assert.Equal(t, vr.OtherByte, elem.VR())

	// Verify value (binary data)
	assert.Contains(t, elem.Value().String(), "00 01 02 03")
}

// TestElementParser_ReadElement_ExplicitVR_FL tests parsing a FL element.
func TestElementParser_ReadElement_ExplicitVR_FL(t *testing.T) {
	// Setup: Create a buffer with a FL element
	buf := new(bytes.Buffer)

	// Tag: (0018,1318)
	binary.Write(buf, binary.LittleEndian, uint16(0x0018)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x1318)) // element

	// VR: FL (2 bytes)
	buf.WriteString("FL")

	// Length: 2 bytes for FL (value is 4 bytes)
	binary.Write(buf, binary.LittleEndian, uint16(4))

	// Value: float32
	binary.Write(buf, binary.LittleEndian, float32(3.14159))

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element
	elem, err := parser.ReadElement()
	require.NoError(t, err)
	require.NotNil(t, elem)

	// Verify VR
	assert.Equal(t, vr.FloatingPointSingle, elem.VR())

	// Verify value (approximate due to float precision)
	assert.Contains(t, elem.Value().String(), "3.14")
}

// TestElementParser_ReadElement_ExplicitVR_EmptyValue tests parsing an element with empty value.
func TestElementParser_ReadElement_ExplicitVR_EmptyValue(t *testing.T) {
	// Setup: Create a buffer with an element with length 0
	buf := new(bytes.Buffer)

	// Tag: (0010,0030) DA Patient's Birth Date
	binary.Write(buf, binary.LittleEndian, uint16(0x0010)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x0030)) // element

	// VR: DA (2 bytes)
	buf.WriteString("DA")

	// Length: 0
	binary.Write(buf, binary.LittleEndian, uint16(0))

	// No value data

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element
	elem, err := parser.ReadElement()
	require.NoError(t, err)
	require.NotNil(t, elem)

	// Verify tag
	assert.True(t, elem.Tag().Equals(tag.New(0x0010, 0x0030)))

	// Verify VR
	assert.Equal(t, vr.Date, elem.VR())

	// Verify value is empty
	assert.Equal(t, "", elem.Value().String())
}

// TestElementParser_ReadElement_ExplicitVR_MultipleValues tests parsing an element with multiple values.
func TestElementParser_ReadElement_ExplicitVR_MultipleValues(t *testing.T) {
	// Setup: Create a buffer with a US element with VM=3
	// (0020,9157) US Dimension Index Values = [1, 2, 3]
	buf := new(bytes.Buffer)

	// Tag: (0020,9157)
	binary.Write(buf, binary.LittleEndian, uint16(0x0020)) // group
	binary.Write(buf, binary.LittleEndian, uint16(0x9157)) // element

	// VR: US (2 bytes)
	buf.WriteString("US")

	// Length: 6 bytes (3 uint16 values)
	binary.Write(buf, binary.LittleEndian, uint16(6))

	// Values: 3 uint16 values
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(2))
	binary.Write(buf, binary.LittleEndian, uint16(3))

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element
	elem, err := parser.ReadElement()
	require.NoError(t, err)
	require.NotNil(t, elem)

	// Verify VR
	assert.Equal(t, vr.UnsignedShort, elem.VR())

	// Verify value contains all three values
	valueStr := elem.Value().String()
	assert.Contains(t, valueStr, "1")
	assert.Contains(t, valueStr, "2")
	assert.Contains(t, valueStr, "3")
}

// TestElementParser_ReadElement_InvalidVR tests parsing with invalid VR.
func TestElementParser_ReadElement_InvalidVR(t *testing.T) {
	// Setup: Create a buffer with invalid VR
	buf := new(bytes.Buffer)

	// Tag: (0010,0010)
	binary.Write(buf, binary.LittleEndian, uint16(0x0010))
	binary.Write(buf, binary.LittleEndian, uint16(0x0010))

	// Invalid VR: "XX"
	buf.WriteString("XX")

	// Length
	binary.Write(buf, binary.LittleEndian, uint16(4))

	// Value
	buf.WriteString("TEST")

	// Create element parser
	reader := NewReader(buf, binary.LittleEndian)
	ts := &TransferSyntax{
		ExplicitVR: true,
		ByteOrder:  binary.LittleEndian,
	}
	parser := NewElementParser(reader, ts)

	// Parse element - should fail
	_, err := parser.ReadElement()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidVR)
}
