package pixel

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Helper function to create encapsulated pixel data for testing
func createEncapsulatedData(offsetTable []uint32, fragments [][]byte) []byte {
	buf := new(bytes.Buffer)

	// Write Basic Offset Table item
	binary.Write(buf, binary.LittleEndian, ItemTagGroup)               // Group FFFE
	binary.Write(buf, binary.LittleEndian, ItemTag)                    // Element E000
	binary.Write(buf, binary.LittleEndian, uint32(len(offsetTable)*4)) // Length

	// Write offsets
	for _, offset := range offsetTable {
		binary.Write(buf, binary.LittleEndian, offset)
	}

	// Write fragment items
	for _, fragment := range fragments {
		binary.Write(buf, binary.LittleEndian, ItemTagGroup)          // Group FFFE
		binary.Write(buf, binary.LittleEndian, ItemTag)               // Element E000
		binary.Write(buf, binary.LittleEndian, uint32(len(fragment))) // Length
		buf.Write(fragment)
	}

	// Write sequence delimiter
	binary.Write(buf, binary.LittleEndian, ItemTagGroup)         // Group FFFE
	binary.Write(buf, binary.LittleEndian, SequenceDelimiterTag) // Element E0DD
	binary.Write(buf, binary.LittleEndian, uint32(0))            // Length 0

	return buf.Bytes()
}

func TestParseEncapsulatedPixelData_WithOffsetTable(t *testing.T) {
	// Create test data with 2 frames
	// Frame 0: fragments at offset 0
	// Frame 1: fragments at offset 100
	offsetTable := []uint32{0, 100}
	fragments := [][]byte{
		{0x01, 0x02, 0x03},       // Fragment 0 (frame 0)
		{0x04, 0x05, 0x06, 0x07}, // Fragment 1 (frame 1)
	}

	data := createEncapsulatedData(offsetTable, fragments)

	result, err := ParseEncapsulatedPixelData(data)
	if err != nil {
		t.Fatalf("ParseEncapsulatedPixelData failed: %v", err)
	}

	// Verify Basic Offset Table
	if len(result.BasicOffsetTable.Offsets) != 2 {
		t.Errorf("expected 2 offsets, got %d", len(result.BasicOffsetTable.Offsets))
	}
	if result.BasicOffsetTable.Offsets[0] != 0 {
		t.Errorf("expected offset 0, got %d", result.BasicOffsetTable.Offsets[0])
	}
	if result.BasicOffsetTable.Offsets[1] != 100 {
		t.Errorf("expected offset 100, got %d", result.BasicOffsetTable.Offsets[1])
	}

	// Verify fragments
	if len(result.Fragments) != 2 {
		t.Errorf("expected 2 fragments, got %d", len(result.Fragments))
	}

	if !bytes.Equal(result.Fragments[0].Data, fragments[0]) {
		t.Errorf("fragment 0 data mismatch: got %v, want %v", result.Fragments[0].Data, fragments[0])
	}

	if !bytes.Equal(result.Fragments[1].Data, fragments[1]) {
		t.Errorf("fragment 1 data mismatch: got %v, want %v", result.Fragments[1].Data, fragments[1])
	}
}

func TestParseEncapsulatedPixelData_WithoutOffsetTable(t *testing.T) {
	// Create test data without offset table (empty first item)
	fragments := [][]byte{
		{0x01, 0x02, 0x03},
		{0x04, 0x05, 0x06, 0x07},
	}

	data := createEncapsulatedData(nil, fragments)

	result, err := ParseEncapsulatedPixelData(data)
	if err != nil {
		t.Fatalf("ParseEncapsulatedPixelData failed: %v", err)
	}

	// Verify Basic Offset Table is empty
	if len(result.BasicOffsetTable.Offsets) != 0 {
		t.Errorf("expected empty offset table, got %d offsets", len(result.BasicOffsetTable.Offsets))
	}

	// Verify fragments
	if len(result.Fragments) != 2 {
		t.Errorf("expected 2 fragments, got %d", len(result.Fragments))
	}

	if !bytes.Equal(result.Fragments[0].Data, fragments[0]) {
		t.Errorf("fragment 0 data mismatch")
	}

	if !bytes.Equal(result.Fragments[1].Data, fragments[1]) {
		t.Errorf("fragment 1 data mismatch")
	}
}

func TestParseEncapsulatedPixelData_EmptyData(t *testing.T) {
	_, err := ParseEncapsulatedPixelData([]byte{})
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestParseEncapsulatedPixelData_TruncatedData(t *testing.T) {
	// Create valid data then truncate it
	fragments := [][]byte{{0x01, 0x02, 0x03}}
	data := createEncapsulatedData(nil, fragments)

	// Truncate to less than 8 bytes
	truncated := data[:5]

	_, err := ParseEncapsulatedPixelData(truncated)
	if err == nil {
		t.Error("expected error for truncated data, got nil")
	}
}

func TestParseEncapsulatedPixelData_InvalidTag(t *testing.T) {
	buf := new(bytes.Buffer)

	// Write invalid tag (not FFFE,E000)
	binary.Write(buf, binary.LittleEndian, uint16(0x0008)) // Invalid group
	binary.Write(buf, binary.LittleEndian, uint16(0x0005)) // Invalid element
	binary.Write(buf, binary.LittleEndian, uint32(0))      // Length

	_, err := ParseEncapsulatedPixelData(buf.Bytes())
	if err == nil {
		t.Error("expected error for invalid tag, got nil")
	}
}

func TestGetFrameFragments_WithoutOffsetTable(t *testing.T) {
	// Each fragment is a complete frame
	fragments := [][]byte{
		{0x01, 0x02, 0x03},       // Frame 0
		{0x04, 0x05, 0x06, 0x07}, // Frame 1
		{0x08, 0x09},             // Frame 2
	}

	data := createEncapsulatedData(nil, fragments)
	result, err := ParseEncapsulatedPixelData(data)
	if err != nil {
		t.Fatalf("ParseEncapsulatedPixelData failed: %v", err)
	}

	// Test getting each frame
	for i := 0; i < 3; i++ {
		frameFragments, err := result.GetFrameFragments(i)
		if err != nil {
			t.Errorf("GetFrameFragments(%d) failed: %v", i, err)
			continue
		}

		if len(frameFragments) != 1 {
			t.Errorf("expected 1 fragment for frame %d, got %d", i, len(frameFragments))
		}

		if !bytes.Equal(frameFragments[0].Data, fragments[i]) {
			t.Errorf("frame %d data mismatch", i)
		}
	}

	// Test out of range
	_, err = result.GetFrameFragments(3)
	if err == nil {
		t.Error("expected error for out of range frame, got nil")
	}
}

func TestGetFrameFragments_WithOffsetTable_SingleFragmentPerFrame(t *testing.T) {
	// Create offset table for 3 frames
	// Each fragment has 8-byte header (group + element + length)
	// Fragment 0: offset 0, length 3, next fragment at 0 + 8 + 3 = 11
	// Fragment 1: offset 11, length 4, next fragment at 11 + 8 + 4 = 23
	// Fragment 2: offset 23, length 2
	offsetTable := []uint32{0, 11, 23}

	fragments := [][]byte{
		{0x01, 0x02, 0x03},       // Frame 0
		{0x04, 0x05, 0x06, 0x07}, // Frame 1
		{0x08, 0x09},             // Frame 2
	}

	data := createEncapsulatedData(offsetTable, fragments)
	result, err := ParseEncapsulatedPixelData(data)
	if err != nil {
		t.Fatalf("ParseEncapsulatedPixelData failed: %v", err)
	}

	// Test getting each frame - each should have exactly one fragment
	for i := 0; i < 3; i++ {
		frameFragments, err := result.GetFrameFragments(i)
		if err != nil {
			t.Fatalf("GetFrameFragments(%d) failed: %v", i, err)
		}

		if len(frameFragments) != 1 {
			t.Errorf("expected 1 fragment for frame %d, got %d", i, len(frameFragments))
		}

		if !bytes.Equal(frameFragments[0].Data, fragments[i]) {
			t.Errorf("frame %d data mismatch: got %v, want %v",
				i, frameFragments[0].Data, fragments[i])
		}
	}
}

func TestGetFrameFragments_OutOfRange(t *testing.T) {
	fragments := [][]byte{
		{0x01, 0x02, 0x03},
	}

	data := createEncapsulatedData(nil, fragments)
	result, err := ParseEncapsulatedPixelData(data)
	if err != nil {
		t.Fatalf("ParseEncapsulatedPixelData failed: %v", err)
	}

	// Test out of range frame index
	_, err = result.GetFrameFragments(10)
	if err == nil {
		t.Error("expected error for out of range frame, got nil")
	}
}

func TestConcatenateFragments_Single(t *testing.T) {
	fragments := []Fragment{
		{Data: []byte{0x01, 0x02, 0x03}},
	}

	result := ConcatenateFragments(fragments)

	expected := []byte{0x01, 0x02, 0x03}
	if !bytes.Equal(result, expected) {
		t.Errorf("concatenate failed: got %v, want %v", result, expected)
	}
}

func TestConcatenateFragments_Multiple(t *testing.T) {
	fragments := []Fragment{
		{Data: []byte{0x01, 0x02}},
		{Data: []byte{0x03, 0x04, 0x05}},
		{Data: []byte{0x06}},
	}

	result := ConcatenateFragments(fragments)

	expected := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	if !bytes.Equal(result, expected) {
		t.Errorf("concatenate failed: got %v, want %v", result, expected)
	}
}

func TestConcatenateFragments_Empty(t *testing.T) {
	fragments := []Fragment{}
	result := ConcatenateFragments(fragments)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d bytes", len(result))
	}
}

func TestNumFrames_WithOffsetTable(t *testing.T) {
	offsetTable := []uint32{0, 100, 200}
	fragments := [][]byte{
		{0x01}, {0x02}, {0x03},
	}

	data := createEncapsulatedData(offsetTable, fragments)
	result, err := ParseEncapsulatedPixelData(data)
	if err != nil {
		t.Fatalf("ParseEncapsulatedPixelData failed: %v", err)
	}

	numFrames := result.NumFrames()
	if numFrames != 3 {
		t.Errorf("expected 3 frames, got %d", numFrames)
	}
}

func TestNumFrames_WithoutOffsetTable(t *testing.T) {
	fragments := [][]byte{
		{0x01}, {0x02}, {0x03}, {0x04},
	}

	data := createEncapsulatedData(nil, fragments)
	result, err := ParseEncapsulatedPixelData(data)
	if err != nil {
		t.Fatalf("ParseEncapsulatedPixelData failed: %v", err)
	}

	numFrames := result.NumFrames()
	if numFrames != 4 {
		t.Errorf("expected 4 frames, got %d", numFrames)
	}
}

func TestParseBasicOffsetTable_Valid(t *testing.T) {
	// Create offset table data (3 offsets)
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], 0)
	binary.LittleEndian.PutUint32(data[4:8], 100)
	binary.LittleEndian.PutUint32(data[8:12], 200)

	table, err := parseBasicOffsetTable(data)
	if err != nil {
		t.Fatalf("parseBasicOffsetTable failed: %v", err)
	}

	if len(table.Offsets) != 3 {
		t.Errorf("expected 3 offsets, got %d", len(table.Offsets))
	}

	if table.Offsets[0] != 0 || table.Offsets[1] != 100 || table.Offsets[2] != 200 {
		t.Errorf("offset values incorrect: got %v", table.Offsets)
	}
}

func TestParseBasicOffsetTable_InvalidLength(t *testing.T) {
	// Create data with invalid length (not multiple of 4)
	data := []byte{0x00, 0x01, 0x02}

	_, err := parseBasicOffsetTable(data)
	if err == nil {
		t.Error("expected error for invalid length, got nil")
	}
}

func TestParseBasicOffsetTable_Empty(t *testing.T) {
	// Empty offset table is valid
	table, err := parseBasicOffsetTable([]byte{})
	if err != nil {
		t.Fatalf("parseBasicOffsetTable failed for empty data: %v", err)
	}

	if len(table.Offsets) != 0 {
		t.Errorf("expected empty offset table, got %d offsets", len(table.Offsets))
	}
}
