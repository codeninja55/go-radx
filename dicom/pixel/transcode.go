package pixel

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/element"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
	"github.com/codeninja55/go-radx/dicom/vr"
)

// Uncompressed transfer syntax UIDs that should be transcoded
const (
	// ImplicitVRLittleEndian is 1.2.840.10008.1.2
	ImplicitVRLittleEndian = "1.2.840.10008.1.2"
	// ExplicitVRLittleEndian is 1.2.840.10008.1.2.1
	ExplicitVRLittleEndian = "1.2.840.10008.1.2.1"
	// ExplicitVRBigEndian is 1.2.840.10008.1.2.2
	ExplicitVRBigEndian = "1.2.840.10008.1.2.2"

	// JPEG2000LosslessOnly is 1.2.840.10008.1.2.4.90
	JPEG2000LosslessOnly = "1.2.840.10008.1.2.4.90"
)

// IsUncompressedTransferSyntax checks if a transfer syntax UID represents uncompressed data.
func IsUncompressedTransferSyntax(transferSyntaxUID string) bool {
	switch transferSyntaxUID {
	case ImplicitVRLittleEndian, ExplicitVRLittleEndian, ExplicitVRBigEndian:
		return true
	default:
		return false
	}
}

// CreateEncapsulatedPixelData creates DICOM encapsulated pixel data format.
//
// For single-frame images:
//   - Empty Basic Offset Table (0 offsets)
//   - Single fragment containing the compressed frame
//   - Sequence Delimiter
//
// For multi-frame images:
//   - Basic Offset Table with one offset per frame
//   - One fragment per frame
//   - Sequence Delimiter
//
// Format:
//   Item (FFFE,E000) + Length: Basic Offset Table
//   Item (FFFE,E000) + Length: Fragment 1
//   Item (FFFE,E000) + Length: Fragment 2 (if multi-frame)
//   ...
//   Sequence Delimiter (FFFE,E0DD) + Length: 0
func CreateEncapsulatedPixelData(frames [][]byte) ([]byte, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames provided")
	}

	buf := new(bytes.Buffer)

	// Write Basic Offset Table
	// For single frame: empty table (length = 0)
	// For multi-frame: one offset per frame
	if len(frames) == 1 {
		// Empty Basic Offset Table
		binary.Write(buf, binary.LittleEndian, ItemTagGroup) // FFFE
		binary.Write(buf, binary.LittleEndian, ItemTag)      // E000
		binary.Write(buf, binary.LittleEndian, uint32(0))    // Length = 0
	} else {
		// Calculate offsets for each frame
		offsets := make([]uint32, len(frames))
		currentOffset := uint32(0)
		for i := range frames {
			offsets[i] = currentOffset
			// Each fragment has: tag (2) + element (2) + length (4) + data
			currentOffset += uint32(8 + len(frames[i]))
		}

		// Write Basic Offset Table with offsets
		binary.Write(buf, binary.LittleEndian, ItemTagGroup)               // FFFE
		binary.Write(buf, binary.LittleEndian, ItemTag)                    // E000
		binary.Write(buf, binary.LittleEndian, uint32(len(offsets)*4))    // Length
		for _, offset := range offsets {
			binary.Write(buf, binary.LittleEndian, offset)
		}
	}

	// Write fragments (one per frame)
	for _, frame := range frames {
		binary.Write(buf, binary.LittleEndian, ItemTagGroup)      // FFFE
		binary.Write(buf, binary.LittleEndian, ItemTag)           // E000
		binary.Write(buf, binary.LittleEndian, uint32(len(frame))) // Length
		buf.Write(frame)
	}

	// Write Sequence Delimiter
	binary.Write(buf, binary.LittleEndian, ItemTagGroup)         // FFFE
	binary.Write(buf, binary.LittleEndian, SequenceDelimiterTag) // E0DD
	binary.Write(buf, binary.LittleEndian, uint32(0))            // Length = 0

	return buf.Bytes(), nil
}

// TranscodeToJPEG2000Lossless transcodes uncompressed DICOM datasets to JPEG 2000 Lossless.
//
// This function:
//  1. Checks if the dataset uses an uncompressed transfer syntax
//  2. Extracts pixel data and image parameters
//  3. Encodes each frame to JPEG 2000 Lossless
//  4. Creates encapsulated pixel data format
//  5. Updates the dataset with new pixel data and transfer syntax
//
// Returns:
//  - true if transcoding was performed
//  - false if dataset was already compressed or transcoding was not needed
//  - error if transcoding failed
//
// The function modifies the dataset in place by:
//  - Replacing (7FE0,0010) PixelData with encapsulated JPEG 2000 data
//  - Updating (0002,0010) TransferSyntaxUID to 1.2.840.10008.1.2.4.90
func TranscodeToJPEG2000Lossless(ds *dicom.DataSet) (bool, error) {
	// Get transfer syntax UID
	transferSyntaxElem, err := ds.Get(tag.TransferSyntaxUID)
	if err != nil {
		return false, fmt.Errorf("failed to get TransferSyntaxUID: %w", err)
	}

	transferSyntaxValue, ok := transferSyntaxElem.Value().(*value.StringValue)
	if !ok {
		return false, fmt.Errorf("TransferSyntaxUID is not a string value")
	}

	transferSyntaxUID := transferSyntaxValue.Strings()[0]

	// Check if already compressed or not an uncompressed format
	if !IsUncompressedTransferSyntax(transferSyntaxUID) {
		return false, nil // Already compressed, no transcoding needed
	}

	// Extract pixel data and metadata
	pixelDataElem, err := ds.Get(tag.PixelData)
	if err != nil {
		return false, fmt.Errorf("failed to get PixelData: %w", err)
	}

	pixelDataValue, ok := pixelDataElem.Value().(*value.BytesValue)
	if !ok {
		return false, fmt.Errorf("PixelData is not a BytesValue")
	}

	uncompressedPixelData := pixelDataValue.Bytes()

	// Extract image parameters
	rows, err := getUint16(ds, tag.Rows, "Rows")
	if err != nil {
		return false, err
	}

	columns, err := getUint16(ds, tag.Columns, "Columns")
	if err != nil {
		return false, err
	}

	bitsAllocated, err := getUint16(ds, tag.BitsAllocated, "BitsAllocated")
	if err != nil {
		return false, err
	}

	bitsStored, err := getUint16(ds, tag.BitsStored, "BitsStored")
	if err != nil {
		return false, err
	}

	highBit, err := getUint16(ds, tag.HighBit, "HighBit")
	if err != nil {
		return false, err
	}

	pixelRepresentation, err := getUint16(ds, tag.PixelRepresentation, "PixelRepresentation")
	if err != nil {
		return false, err
	}

	samplesPerPixel, err := getUint16(ds, tag.SamplesPerPixel, "SamplesPerPixel")
	if err != nil {
		return false, err
	}

	photometricInterpretation, err := getString(ds, tag.PhotometricInterpretation, "PhotometricInterpretation")
	if err != nil {
		return false, err
	}

	planarConfiguration := getUint16WithDefault(ds, tag.PlanarConfiguration, 0)
	numberOfFrames := getIntWithDefault(ds, tag.NumberOfFrames, 1)

	// Create PixelInfo for encoder
	info := &PixelInfo{
		Rows:                      rows,
		Columns:                   columns,
		BitsAllocated:             bitsAllocated,
		BitsStored:                bitsStored,
		HighBit:                   highBit,
		PixelRepresentation:       pixelRepresentation,
		SamplesPerPixel:           samplesPerPixel,
		PhotometricInterpretation: photometricInterpretation,
		PlanarConfiguration:       planarConfiguration,
		NumberOfFrames:            numberOfFrames,
		TransferSyntaxUID:         JPEG2000LosslessOnly,
	}

	// Calculate frame size
	frameSize := int(CalculateExpectedSize(info)) / numberOfFrames

	// Encode each frame to JPEG 2000 Lossless
	compressedFrames := make([][]byte, numberOfFrames)
	encoder := NewJPEG2000Encoder()

	for frameIndex := 0; frameIndex < numberOfFrames; frameIndex++ {
		// Extract frame data
		frameStart := frameIndex * frameSize
		frameEnd := frameStart + frameSize
		if frameEnd > len(uncompressedPixelData) {
			return false, fmt.Errorf("frame %d exceeds pixel data length", frameIndex)
		}
		frameData := uncompressedPixelData[frameStart:frameEnd]

		// Create frame-specific info
		frameInfo := *info
		frameInfo.NumberOfFrames = 1

		// Encode frame
		compressedFrame, err := encoder.Encode(frameData, &frameInfo)
		if err != nil {
			return false, fmt.Errorf("failed to encode frame %d: %w", frameIndex, err)
		}

		compressedFrames[frameIndex] = compressedFrame
	}

	// Create encapsulated pixel data
	encapsulatedData, err := CreateEncapsulatedPixelData(compressedFrames)
	if err != nil {
		return false, fmt.Errorf("failed to create encapsulated pixel data: %w", err)
	}

	// Update dataset with new pixel data
	newPixelDataValue, err := value.NewBytesValue(vr.OtherByte, encapsulatedData)
	if err != nil {
		return false, fmt.Errorf("failed to create pixel data value: %w", err)
	}

	newPixelDataElem, err := element.NewElement(tag.PixelData, vr.OtherByte, newPixelDataValue)
	if err != nil {
		return false, fmt.Errorf("failed to create pixel data element: %w", err)
	}

	if err := ds.Add(newPixelDataElem); err != nil {
		return false, fmt.Errorf("failed to update PixelData: %w", err)
	}

	// Update transfer syntax UID
	newTransferSyntaxValue, err := value.NewStringValue(vr.UniqueIdentifier, []string{JPEG2000LosslessOnly})
	if err != nil {
		return false, fmt.Errorf("failed to create transfer syntax value: %w", err)
	}

	newTransferSyntaxElem, err := element.NewElement(tag.TransferSyntaxUID, vr.UniqueIdentifier, newTransferSyntaxValue)
	if err != nil {
		return false, fmt.Errorf("failed to create transfer syntax element: %w", err)
	}

	if err := ds.Add(newTransferSyntaxElem); err != nil {
		return false, fmt.Errorf("failed to update TransferSyntaxUID: %w", err)
	}

	return true, nil
}
