package pixel

import (
	"encoding/binary"
	"fmt"
)

// DICOM Item Tags for encapsulated pixel data
const (
	// ItemTag is the tag for pixel data fragments (FFFE,E000)
	ItemTag uint16 = 0xE000
	// ItemDelimiterTag is the tag for item delimiters (FFFE,E00D)
	ItemDelimiterTag uint16 = 0xE00D
	// SequenceDelimiterTag is the tag for sequence delimiters (FFFE,E0DD)
	SequenceDelimiterTag uint16 = 0xE0DD
	// ItemTagGroup is the group for all item-related tags
	ItemTagGroup uint16 = 0xFFFE
)

// Fragment represents a single fragment of encapsulated pixel data.
type Fragment struct {
	// Data is the raw fragment data (without Item tag/length header)
	Data []byte
	// Offset is the byte offset of this fragment in the original encapsulated data
	Offset int
}

// BasicOffsetTable contains frame boundary offsets for multi-frame images.
//
// The Basic Offset Table is the first item in encapsulated pixel data and contains
// byte offsets to the first fragment of each frame (relative to the first byte
// following the Basic Offset Table).
//
// If the table is empty, each fragment represents a complete frame.
type BasicOffsetTable struct {
	// Offsets contains byte offsets for each frame
	Offsets []uint32
}

// EncapsulatedPixelData represents parsed encapsulated pixel data.
type EncapsulatedPixelData struct {
	// BasicOffsetTable contains frame offsets (may be empty)
	BasicOffsetTable BasicOffsetTable
	// Fragments contains all pixel data fragments
	Fragments []Fragment
}

// ParseEncapsulatedPixelData parses DICOM encapsulated pixel data into fragments.
//
// Encapsulated pixel data format:
//   - Item (FFFE,E000) + Length: Basic Offset Table (may be empty)
//   - Item (FFFE,E000) + Length: Fragment 1 data
//   - Item (FFFE,E000) + Length: Fragment 2 data
//   - ...
//   - Sequence Delimiter (FFFE,E0DD) + Length: 0
//
// This function extracts all fragments and the Basic Offset Table if present.
func ParseEncapsulatedPixelData(data []byte) (*EncapsulatedPixelData, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("encapsulated pixel data too short: need at least 8 bytes, got %d", len(data))
	}

	result := &EncapsulatedPixelData{
		Fragments: make([]Fragment, 0),
	}

	offset := 0
	firstItem := true

	for offset < len(data) {
		// Check if we have enough bytes for tag + length
		if offset+8 > len(data) {
			break
		}

		// Read Item tag (4 bytes: group + element)
		group := binary.LittleEndian.Uint16(data[offset : offset+2])
		element := binary.LittleEndian.Uint16(data[offset+2 : offset+4])
		length := binary.LittleEndian.Uint32(data[offset+4 : offset+8])

		offset += 8

		// Check for sequence delimiter
		if group == ItemTagGroup && element == SequenceDelimiterTag {
			// End of encapsulated data
			break
		}

		// Verify this is an Item tag
		if group != ItemTagGroup || element != ItemTag {
			return nil, fmt.Errorf("expected Item tag (FFFE,E000), got (%04X,%04X) at offset %d",
				group, element, offset-8)
		}

		// Check if we have enough data for this fragment
		if offset+int(length) > len(data) {
			return nil, fmt.Errorf("fragment length %d exceeds available data at offset %d", length, offset)
		}

		fragmentData := data[offset : offset+int(length)]
		offset += int(length)

		// First item is the Basic Offset Table
		if firstItem {
			firstItem = false
			if length > 0 {
				// Parse Basic Offset Table
				table, err := parseBasicOffsetTable(fragmentData)
				if err != nil {
					return nil, fmt.Errorf("parse basic offset table: %w", err)
				}
				result.BasicOffsetTable = *table
			}
			// Don't add Basic Offset Table to fragments
			continue
		}

		// Add fragment
		result.Fragments = append(result.Fragments, Fragment{
			Data:   fragmentData,
			Offset: offset - int(length),
		})
	}

	return result, nil
}

// parseBasicOffsetTable parses the Basic Offset Table from the first item.
//
// The table contains uint32 offsets (4 bytes each) for each frame.
func parseBasicOffsetTable(data []byte) (*BasicOffsetTable, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("basic offset table length must be multiple of 4, got %d", len(data))
	}

	numOffsets := len(data) / 4
	offsets := make([]uint32, numOffsets)

	for i := 0; i < numOffsets; i++ {
		offsets[i] = binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
	}

	return &BasicOffsetTable{Offsets: offsets}, nil
}

// GetFrameFragments returns all fragments for a specific frame.
//
// If the Basic Offset Table is present, it uses the offsets to determine frame boundaries.
// If the table is empty, each fragment is assumed to be a complete frame.
//
// Returns:
//   - fragments: slice of Fragment for the requested frame
//   - error: if frame index is out of range or fragments cannot be determined
func (e *EncapsulatedPixelData) GetFrameFragments(frameIndex int) ([]Fragment, error) {
	// Case 1: No Basic Offset Table - each fragment is a complete frame
	if len(e.BasicOffsetTable.Offsets) == 0 {
		if frameIndex >= len(e.Fragments) {
			return nil, fmt.Errorf("frame index %d out of range (have %d fragments)",
				frameIndex, len(e.Fragments))
		}
		return []Fragment{e.Fragments[frameIndex]}, nil
	}

	// Case 2: Basic Offset Table present - use offsets to group fragments
	numFrames := len(e.BasicOffsetTable.Offsets)
	if frameIndex >= numFrames {
		return nil, fmt.Errorf("frame index %d out of range (have %d frames)", frameIndex, numFrames)
	}

	if len(e.Fragments) == 0 {
		return nil, fmt.Errorf("no fragments available for frame %d", frameIndex)
	}

	// Basic Offset Table offsets are relative to the first fragment
	// Convert fragment offsets to be relative to the first fragment
	firstFragmentOffset := uint32(e.Fragments[0].Offset)

	// Get offset for this frame (relative to first fragment)
	frameOffset := e.BasicOffsetTable.Offsets[frameIndex]

	// Determine end offset (either next frame's offset or end of data)
	var endOffset uint32
	if frameIndex+1 < numFrames {
		endOffset = e.BasicOffsetTable.Offsets[frameIndex+1]
	} else {
		// Last frame - use end of last fragment (relative to first fragment)
		lastFragment := e.Fragments[len(e.Fragments)-1]
		endOffset = uint32(lastFragment.Offset-int(firstFragmentOffset)) + uint32(len(lastFragment.Data))
	}

	// Collect all fragments between frameOffset and endOffset
	frameFragments := make([]Fragment, 0)
	for _, fragment := range e.Fragments {
		// Convert fragment offset to be relative to first fragment
		fragOffset := uint32(fragment.Offset - int(firstFragmentOffset))
		if fragOffset >= frameOffset && fragOffset < endOffset {
			frameFragments = append(frameFragments, fragment)
		}
	}

	if len(frameFragments) == 0 {
		return nil, fmt.Errorf("no fragments found for frame %d (offset %d to %d)",
			frameIndex, frameOffset, endOffset)
	}

	return frameFragments, nil
}

// ConcatenateFragments concatenates multiple fragments into a single byte slice.
//
// This is used to reassemble a complete compressed frame from multiple fragments.
func ConcatenateFragments(fragments []Fragment) []byte {
	// Calculate total size
	totalSize := 0
	for _, fragment := range fragments {
		totalSize += len(fragment.Data)
	}

	// Allocate result buffer
	result := make([]byte, 0, totalSize)

	// Concatenate all fragments
	for _, fragment := range fragments {
		result = append(result, fragment.Data...)
	}

	return result
}

// NumFrames returns the number of frames in the encapsulated pixel data.
//
// If the Basic Offset Table is present, it returns the number of offsets.
// Otherwise, it returns the number of fragments (assuming one fragment per frame).
func (e *EncapsulatedPixelData) NumFrames() int {
	if len(e.BasicOffsetTable.Offsets) > 0 {
		return len(e.BasicOffsetTable.Offsets)
	}
	return len(e.Fragments)
}
