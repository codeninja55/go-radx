package pixel

import (
	"fmt"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/tag"
	"github.com/codeninja55/go-radx/dicom/value"
)

// Extract extracts and decompresses pixel data from a DICOM dataset.
//
// This function:
//   - Extracts required pixel metadata (Rows, Columns, BitsAllocated, etc.)
//   - Retrieves the raw pixel data from the PixelData element
//   - Detects the transfer syntax and selects the appropriate decoder
//   - Decompresses the pixel data if necessary
//   - Returns a PixelData struct with decompressed data and metadata
//
// For uncompressed transfer syntaxes (native data), no decompression is performed.
// For compressed transfer syntaxes (JPEG, RLE, etc.), the appropriate decoder is used.
//
// Required DICOM attributes:
//   - (0028,0010) Rows
//   - (0028,0011) Columns
//   - (0028,0100) BitsAllocated
//   - (0028,0101) BitsStored
//   - (0028,0102) HighBit
//   - (0028,0103) PixelRepresentation
//   - (0028,0002) SamplesPerPixel
//   - (0028,0004) PhotometricInterpretation
//   - (7FE0,0010) PixelData
//   - (0002,0010) TransferSyntaxUID (from File Meta Information)
//
// Optional DICOM attributes:
//   - (0028,0006) PlanarConfiguration (defaults to 0)
//   - (0028,0008) NumberOfFrames (defaults to 1)
func Extract(ds *dicom.DataSet) (*PixelData, error) {
	// Extract required metadata
	rows, err := getUint16(ds, tag.Rows, "Rows")
	if err != nil {
		return nil, err
	}

	columns, err := getUint16(ds, tag.Columns, "Columns")
	if err != nil {
		return nil, err
	}

	bitsAllocated, err := getUint16(ds, tag.BitsAllocated, "BitsAllocated")
	if err != nil {
		return nil, err
	}

	bitsStored, err := getUint16(ds, tag.BitsStored, "BitsStored")
	if err != nil {
		return nil, err
	}

	highBit, err := getUint16(ds, tag.HighBit, "HighBit")
	if err != nil {
		return nil, err
	}

	pixelRepresentation, err := getUint16(ds, tag.PixelRepresentation, "PixelRepresentation")
	if err != nil {
		return nil, err
	}

	samplesPerPixel, err := getUint16(ds, tag.SamplesPerPixel, "SamplesPerPixel")
	if err != nil {
		return nil, err
	}

	photometricInterpretation, err := getString(ds, tag.PhotometricInterpretation, "PhotometricInterpretation")
	if err != nil {
		return nil, err
	}

	// Extract optional metadata with defaults
	planarConfiguration := getUint16WithDefault(ds, tag.PlanarConfiguration, 0)
	numberOfFrames := getIntWithDefault(ds, tag.NumberOfFrames, 1)

	// Get transfer syntax UID
	transferSyntaxUID, err := getString(ds, tag.TransferSyntaxUID, "TransferSyntaxUID")
	if err != nil {
		return nil, err
	}

	// Get raw pixel data
	pixelDataElem, err := ds.Get(tag.PixelData)
	if err != nil {
		return nil, &MissingAttributeError{
			AttributeName: "PixelData",
			Tag:           tag.PixelData.String(),
		}
	}

	pixelDataValue := pixelDataElem.Value()
	bytesVal, ok := pixelDataValue.(*value.BytesValue)
	if !ok {
		return nil, &PixelDataError{
			Field:    "PixelData value type",
			Expected: "*value.BytesValue",
			Actual:   fmt.Sprintf("%T", pixelDataValue),
		}
	}

	encapsulatedData := bytesVal.Bytes()

	// Get decoder for transfer syntax
	decoder, err := GetDecoder(transferSyntaxUID)
	if err != nil {
		return nil, err
	}

	// Create PixelInfo for decoder
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
		TransferSyntaxUID:         transferSyntaxUID,
	}

	var decompressedData []byte

	// Check if transfer syntax uses encapsulation
	if isEncapsulated(transferSyntaxUID) {
		// Parse encapsulated pixel data into fragments
		encapsulated, err := ParseEncapsulatedPixelData(encapsulatedData)
		if err != nil {
			return nil, &PixelDataError{
				Field:    "encapsulated pixel data parsing",
				Expected: "valid encapsulated format",
				Actual:   fmt.Sprintf("parse error: %v", err),
			}
		}

		// Verify number of frames matches
		if numberOfFrames > 1 && encapsulated.NumFrames() != numberOfFrames {
			// Allow mismatch if Basic Offset Table is empty (single fragment per frame)
			if len(encapsulated.BasicOffsetTable.Offsets) > 0 {
				return nil, &PixelDataError{
					Field:    "number of frames",
					Expected: fmt.Sprintf("%d frames", numberOfFrames),
					Actual:   fmt.Sprintf("%d fragments/frames in encapsulated data", encapsulated.NumFrames()),
				}
			}
		}

		// Decompress each frame
		frameSize := CalculateExpectedSize(info) / numberOfFrames
		decompressedData = make([]byte, 0, CalculateExpectedSize(info))

		for frameIndex := 0; frameIndex < numberOfFrames; frameIndex++ {
			// Get fragments for this frame
			frameFragments, err := encapsulated.GetFrameFragments(frameIndex)
			if err != nil {
				return nil, &PixelDataError{
					Field:    fmt.Sprintf("frame %d fragments", frameIndex),
					Expected: "valid frame fragments",
					Actual:   fmt.Sprintf("error: %v", err),
				}
			}

			// Concatenate fragments into a single compressed frame
			compressedFrame := ConcatenateFragments(frameFragments)

			// Decompress this frame
			frameInfo := *info // Copy info
			frameInfo.NumberOfFrames = 1

			decompressedFrame, err := decoder.Decode(compressedFrame, &frameInfo)
			if err != nil {
				return nil, &PixelDataError{
					Field:    fmt.Sprintf("frame %d decompression", frameIndex),
					Expected: "successful decompression",
					Actual:   fmt.Sprintf("error: %v", err),
				}
			}

			// Validate frame size
			if len(decompressedFrame) != frameSize {
				return nil, &PixelDataError{
					Field:    fmt.Sprintf("frame %d size", frameIndex),
					Expected: fmt.Sprintf("%d bytes", frameSize),
					Actual:   fmt.Sprintf("%d bytes", len(decompressedFrame)),
				}
			}

			decompressedData = append(decompressedData, decompressedFrame...)
		}
	} else {
		// Native/uncompressed data - decode as a single block
		decompressedData, err = decoder.Decode(encapsulatedData, info)
		if err != nil {
			return nil, err
		}
	}

	// Validate decompressed data size
	if err := ValidatePixelData(decompressedData, info); err != nil {
		return nil, err
	}

	// Return PixelData struct
	return &PixelData{
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
		data:                      decompressedData,
		TransferSyntaxUID:         transferSyntaxUID,
	}, nil
}

// getUint16 extracts a uint16 value from a DICOM element.
func getUint16(ds *dicom.DataSet, t tag.Tag, name string) (uint16, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return 0, &MissingAttributeError{
			AttributeName: name,
			Tag:           t.String(),
		}
	}

	intVal, ok := elem.Value().(*value.IntValue)
	if !ok {
		return 0, &PixelDataError{
			Field:    fmt.Sprintf("%s value type", name),
			Expected: "*value.IntValue",
			Actual:   fmt.Sprintf("%T", elem.Value()),
		}
	}

	ints := intVal.Ints()
	if len(ints) == 0 {
		return 0, &PixelDataError{
			Field:    fmt.Sprintf("%s value", name),
			Expected: "non-empty integer array",
			Actual:   "empty array",
		}
	}

	val := ints[0]
	if val < 0 || val > 65535 {
		return 0, &PixelDataError{
			Field:    fmt.Sprintf("%s value", name),
			Expected: "uint16 range [0, 65535]",
			Actual:   fmt.Sprintf("%d", val),
		}
	}

	return uint16(val), nil
}

// getUint16WithDefault extracts a uint16 value with a default if the element is missing.
func getUint16WithDefault(ds *dicom.DataSet, t tag.Tag, defaultVal uint16) uint16 {
	elem, err := ds.Get(t)
	if err != nil {
		return defaultVal
	}

	intVal, ok := elem.Value().(*value.IntValue)
	if !ok {
		return defaultVal
	}

	ints := intVal.Ints()
	if len(ints) == 0 {
		return defaultVal
	}

	val := ints[0]
	if val < 0 || val > 65535 {
		return defaultVal
	}

	return uint16(val)
}

// getIntWithDefault extracts an int value with a default if the element is missing.
func getIntWithDefault(ds *dicom.DataSet, t tag.Tag, defaultVal int) int {
	elem, err := ds.Get(t)
	if err != nil {
		return defaultVal
	}

	// NumberOfFrames can be either IntValue or StringValue (IS - Integer String)
	switch v := elem.Value().(type) {
	case *value.IntValue:
		ints := v.Ints()
		if len(ints) == 0 {
			return defaultVal
		}
		return int(ints[0])

	case *value.StringValue:
		strs := v.Strings()
		if len(strs) == 0 {
			return defaultVal
		}
		// Parse integer string
		var val int
		if _, err := fmt.Sscanf(strs[0], "%d", &val); err != nil {
			return defaultVal
		}
		return val

	default:
		return defaultVal
	}
}

// getString extracts a string value from a DICOM element.
func getString(ds *dicom.DataSet, t tag.Tag, name string) (string, error) {
	elem, err := ds.Get(t)
	if err != nil {
		return "", &MissingAttributeError{
			AttributeName: name,
			Tag:           t.String(),
		}
	}

	strVal, ok := elem.Value().(*value.StringValue)
	if !ok {
		return "", &PixelDataError{
			Field:    fmt.Sprintf("%s value type", name),
			Expected: "*value.StringValue",
			Actual:   fmt.Sprintf("%T", elem.Value()),
		}
	}

	strs := strVal.Strings()
	if len(strs) == 0 {
		return "", &PixelDataError{
			Field:    fmt.Sprintf("%s value", name),
			Expected: "non-empty string array",
			Actual:   "empty array",
		}
	}

	return strs[0], nil
}

// isEncapsulated returns true if the transfer syntax uses encapsulated pixel data format.
//
// Encapsulated format is used by all compressed transfer syntaxes:
//   - RLE Lossless
//   - JPEG Baseline and Extended
//   - JPEG Lossless
//   - JPEG 2000 (Lossless and Lossy)
//   - HTJ2K (High-Throughput JPEG 2000)
//
// Native/uncompressed transfer syntaxes do not use encapsulation.
func isEncapsulated(transferSyntaxUID string) bool {
	// Native/uncompressed transfer syntaxes (no encapsulation)
	switch transferSyntaxUID {
	case "1.2.840.10008.1.2", // Implicit VR Little Endian
		"1.2.840.10008.1.2.1",    // Explicit VR Little Endian
		"1.2.840.10008.1.2.2",    // Explicit VR Big Endian
		"1.2.840.10008.1.2.1.99": // Deflated Explicit VR Little Endian
		return false
	default:
		// All other transfer syntaxes use encapsulation (compressed formats)
		return true
	}
}
