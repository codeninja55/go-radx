package pixel

import (
	"encoding/binary"
	"fmt"
)

// RLEDecoder implements DICOM RLE Lossless decompression.
//
// DICOM RLE encoding is specified in DICOM PS3.5 Part 5, Annex G.
// It uses the PackBits RLE algorithm with segments organized by byte position.
type RLEDecoder struct{}

// Decode decompresses RLE Lossless encoded pixel data.
func (d *RLEDecoder) Decode(encapsulated []byte, info *PixelInfo) ([]byte, error) {
	if len(encapsulated) < 64 {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.TransferSyntaxUID(),
			Cause:             fmt.Errorf("RLE data too small (< 64 bytes): %d bytes", len(encapsulated)),
		}
	}

	// Read RLE header (64 bytes total)
	// First 4 bytes: number of segments (little-endian uint32)
	numSegments := binary.LittleEndian.Uint32(encapsulated[0:4])

	// Next 60 bytes: 15 uint32 offsets for segment positions
	offsets := make([]uint32, 15)
	for i := 0; i < 15; i++ {
		offsets[i] = binary.LittleEndian.Uint32(encapsulated[4+i*4 : 8+i*4])
	}

	// Validate number of segments
	if numSegments == 0 || numSegments > 15 {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.TransferSyntaxUID(),
			Cause:             fmt.Errorf("invalid number of RLE segments: %d (must be 1-15)", numSegments),
		}
	}

	// Calculate expected output size
	expectedSize := CalculateExpectedSize(info)
	output := make([]byte, expectedSize)

	// Determine bytes per pixel
	bytesPerSample := (int(info.BitsAllocated) + 7) / 8
	samplesPerFrame := int(info.Rows) * int(info.Columns) * int(info.SamplesPerPixel)

	// Decompress each segment
	for segmentIdx := 0; segmentIdx < int(numSegments); segmentIdx++ {
		segmentOffset := int(offsets[segmentIdx])

		// Determine segment end (next segment start or end of data)
		var segmentEnd int
		if segmentIdx < int(numSegments)-1 {
			segmentEnd = int(offsets[segmentIdx+1])
		} else {
			segmentEnd = len(encapsulated)
		}

		if segmentOffset >= len(encapsulated) || segmentEnd > len(encapsulated) {
			return nil, &DecompressionError{
				TransferSyntaxUID: d.TransferSyntaxUID(),
				Cause:             fmt.Errorf("segment %d offset out of bounds: %d-%d (data size: %d)", segmentIdx, segmentOffset, segmentEnd, len(encapsulated)),
			}
		}

		segmentData := encapsulated[segmentOffset:segmentEnd]

		// Decompress segment using PackBits RLE
		decompressed, err := decodePackBits(segmentData)
		if err != nil {
			return nil, &DecompressionError{
				TransferSyntaxUID: d.TransferSyntaxUID(),
				Cause:             fmt.Errorf("segment %d decompression failed: %w", segmentIdx, err),
			}
		}

		// Interleave decompressed bytes into output
		// For multi-byte pixels, segments correspond to byte positions (LSB, MSB for 16-bit)
		bytePosition := segmentIdx % bytesPerSample
		for i := 0; i < len(decompressed) && i < samplesPerFrame; i++ {
			outputIdx := i*bytesPerSample + bytePosition
			if outputIdx < len(output) {
				output[outputIdx] = decompressed[i]
			}
		}
	}

	return output, nil
}

// TransferSyntaxUID returns the RLE Lossless transfer syntax UID.
func (d *RLEDecoder) TransferSyntaxUID() string {
	return "1.2.840.10008.1.2.5" // RLE Lossless
}

// decodePackBits implements the PackBits RLE decompression algorithm.
//
// PackBits encoding:
//   - If byte n is in range [0, 127], copy next n+1 bytes literally
//   - If byte n is in range [129, 255] (i.e., -127 to -1 as signed int8), repeat next byte (257-n) times
//   - If byte n is 128, no operation (skip)
func decodePackBits(data []byte) ([]byte, error) {
	output := make([]byte, 0, len(data)*2) // Pre-allocate with estimated size
	pos := 0

	for pos < len(data) {
		// Read control byte
		control := int8(data[pos])
		pos++

		if control >= 0 {
			// Literal run: copy next (control + 1) bytes
			count := int(control) + 1

			if pos+count > len(data) {
				return nil, fmt.Errorf("literal run extends beyond data: pos=%d, count=%d, len=%d", pos, count, len(data))
			}

			output = append(output, data[pos:pos+count]...)
			pos += count

		} else if control != -128 {
			// Repeat run: repeat next byte (1 - control) times
			// control is negative, so (1 - control) = (1 + |control|)
			count := 1 - int(control)

			if pos >= len(data) {
				return nil, fmt.Errorf("repeat run missing data byte: pos=%d, len=%d", pos, len(data))
			}

			repeatByte := data[pos]
			pos++

			for i := 0; i < count; i++ {
				output = append(output, repeatByte)
			}
		}
		// control == -128: no operation, just skip
	}

	return output, nil
}

func init() {
	// Register RLE decoder
	RegisterDecoder("1.2.840.10008.1.2.5", &RLEDecoder{})
}
