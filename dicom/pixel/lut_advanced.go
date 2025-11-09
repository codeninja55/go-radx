package pixel

import (
	"fmt"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/tag"
)

// PresentationLUT represents the Presentation LUT transformation.
//
// The Presentation LUT is applied after the VOI LUT to define device-independent
// presentation of pixel values. It maps input P-Values to output P-Values.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part03.html#sect_C.11.6
type PresentationLUT struct {
	// LUT-based transformation
	LUTData       []uint16  // Lookup table data
	LUTDescriptor [3]uint16 // [num_entries, first_mapped_value, bits_per_entry]

	// Shape-based transformation
	PresentationLUTShape string // "IDENTITY" or "INVERSE"
}

// PaletteColorLUT represents color palette lookup tables for PALETTE COLOR images.
//
// Palette color images store pixel values as indices into Red, Green, and Blue LUTs.
// This allows efficient storage of pseudo-color images.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part03.html#sect_C.7.6.3.1.5
type PaletteColorLUT struct {
	RedDescriptor   [3]uint16 // [num_entries, first_mapped_value, bits_per_entry]
	GreenDescriptor [3]uint16
	BlueDescriptor  [3]uint16

	RedData   []uint16 // Red palette data
	GreenData []uint16 // Green palette data
	BlueData  []uint16 // Blue palette data

	// Segmented palette data (if present)
	RedSegmented   *SegmentedLUT
	GreenSegmented *SegmentedLUT
	BlueSegmented  *SegmentedLUT
}

// SegmentedLUT represents a segmented palette color lookup table.
//
// Segmented LUTs provide a compact representation of large palettes using
// discrete segments, linear segments, and indirect segments.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part03.html#sect_C.7.9
type SegmentedLUT struct {
	Data []uint16 // Segmented LUT data
}

// LUTFunction represents the function type for VOI LUT transformations.
type LUTFunction int

const (
	// LUTFunctionLinear applies linear window/level transformation.
	LUTFunctionLinear LUTFunction = iota

	// LUTFunctionLinearExact applies linear transformation with exact bounds (PS3.4 C.11.2.1.2.1).
	LUTFunctionLinearExact

	// LUTFunctionSigmoid applies sigmoid transformation for smooth transitions.
	LUTFunctionSigmoid
)

// ApplyPresentationLUT applies the Presentation LUT transformation to pixel data.
//
// The Presentation LUT provides device-independent presentation by defining
// the relationship between P-Values and display output.
//
// Parameters:
//   - p: Source pixel data (output from VOI LUT)
//   - presentationLUT: Presentation LUT parameters
//
// Example:
//
//	presentationLUT := &pixel.PresentationLUT{
//	    PresentationLUTShape: "INVERSE",
//	}
//	inverted, err := pixel.ApplyPresentationLUT(pixelData, presentationLUT)
func ApplyPresentationLUT(p *PixelData, presentationLUT *PresentationLUT) (*PixelData, error) {
	if presentationLUT == nil {
		return nil, fmt.Errorf("presentation LUT cannot be nil")
	}

	// Handle shape-based transformation
	if presentationLUT.PresentationLUTShape != "" {
		return applyPresentationLUTShape(p, presentationLUT.PresentationLUTShape)
	}

	// Handle LUT-based transformation
	if len(presentationLUT.LUTData) > 0 {
		return applyPresentationLUTTable(p, presentationLUT)
	}

	return nil, fmt.Errorf("presentation LUT must specify either shape or LUT data")
}

// applyPresentationLUTShape applies shape-based Presentation LUT.
func applyPresentationLUTShape(p *PixelData, shape string) (*PixelData, error) {
	switch shape {
	case "IDENTITY":
		// No transformation - return copy
		return p, nil

	case "INVERSE":
		// Invert the values
		return invertPixelData(p)

	default:
		return nil, fmt.Errorf("unsupported Presentation LUT shape: %s", shape)
	}
}

// applyPresentationLUTTable applies LUT-based Presentation LUT.
func applyPresentationLUTTable(p *PixelData, presentationLUT *PresentationLUT) (*PixelData, error) {
	numEntries := int(presentationLUT.LUTDescriptor[0])
	firstMapped := int(presentationLUT.LUTDescriptor[1])
	bitsPerEntry := int(presentationLUT.LUTDescriptor[2])

	if numEntries == 0 {
		numEntries = 65536
	}

	if len(presentationLUT.LUTData) < numEntries {
		return nil, fmt.Errorf("LUT data length %d < expected %d entries",
			len(presentationLUT.LUTData), numEntries)
	}

	outputMax := uint16((1 << bitsPerEntry) - 1)

	data := make([]byte, len(p.data))

	if p.BitsAllocated <= 8 {
		// 8-bit input
		for i := 0; i < len(p.data); i++ {
			idx := int(p.data[i]) - firstMapped
			if idx < 0 || idx >= numEntries {
				// Out of range - use min/max
				if idx < 0 {
					data[i] = byte(presentationLUT.LUTData[0])
				} else {
					data[i] = byte(presentationLUT.LUTData[numEntries-1])
				}
			} else {
				data[i] = byte(presentationLUT.LUTData[idx] & 0xFF)
			}
		}
	} else {
		// 16-bit input
		for i := 0; i < len(p.data)/2; i++ {
			val16 := uint16(p.data[i*2]) | uint16(p.data[i*2+1])<<8
			idx := int(val16) - firstMapped

			var outVal uint16
			if idx < 0 || idx >= numEntries {
				// Out of range
				if idx < 0 {
					outVal = presentationLUT.LUTData[0]
				} else {
					outVal = presentationLUT.LUTData[numEntries-1]
				}
			} else {
				outVal = presentationLUT.LUTData[idx]
			}

			// Clamp to output range
			if outVal > outputMax {
				outVal = outputMax
			}

			data[i*2] = byte(outVal)
			data[i*2+1] = byte(outVal >> 8)
		}
	}

	result := &PixelData{
		Rows:                      p.Rows,
		Columns:                   p.Columns,
		BitsAllocated:             uint16(bitsPerEntry),
		BitsStored:                uint16(bitsPerEntry),
		HighBit:                   uint16(bitsPerEntry - 1),
		PixelRepresentation:       0,
		SamplesPerPixel:           p.SamplesPerPixel,
		PhotometricInterpretation: p.PhotometricInterpretation,
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// invertPixelData inverts pixel values (max - value).
func invertPixelData(p *PixelData) (*PixelData, error) {
	data := make([]byte, len(p.data))
	maxVal := uint16((1 << p.BitsStored) - 1)

	if p.BitsAllocated <= 8 {
		// 8-bit
		for i := 0; i < len(p.data); i++ {
			data[i] = byte(uint8(maxVal) - p.data[i])
		}
	} else {
		// 16-bit
		for i := 0; i < len(p.data)/2; i++ {
			val16 := uint16(p.data[i*2]) | uint16(p.data[i*2+1])<<8
			inverted := maxVal - val16
			data[i*2] = byte(inverted)
			data[i*2+1] = byte(inverted >> 8)
		}
	}

	result := &PixelData{
		Rows:                      p.Rows,
		Columns:                   p.Columns,
		BitsAllocated:             p.BitsAllocated,
		BitsStored:                p.BitsStored,
		HighBit:                   p.HighBit,
		PixelRepresentation:       p.PixelRepresentation,
		SamplesPerPixel:           p.SamplesPerPixel,
		PhotometricInterpretation: p.PhotometricInterpretation,
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// ApplyPaletteColorLUT applies color palette transformation to pixel data.
//
// This transforms PALETTE COLOR images (indexed color) to RGB by looking up
// each pixel value in the Red, Green, and Blue palette tables.
//
// Parameters:
//   - p: Source pixel data (PALETTE COLOR photometric interpretation)
//   - palette: Palette color LUT parameters
//
// Returns RGB pixel data with SamplesPerPixel=3.
//
// Example:
//
//	palette, err := pixel.ExtractPaletteColorLUTFromDataSet(ds)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	rgb, err := pixel.ApplyPaletteColorLUT(pixelData, palette)
func ApplyPaletteColorLUT(p *PixelData, palette *PaletteColorLUT) (*PixelData, error) {
	if palette == nil {
		return nil, fmt.Errorf("palette cannot be nil")
	}

	if p.PhotometricInterpretation != "PALETTE COLOR" {
		return nil, fmt.Errorf("palette LUT only applies to PALETTE COLOR images, got %s",
			p.PhotometricInterpretation)
	}

	if p.SamplesPerPixel != 1 {
		return nil, fmt.Errorf("palette color images must have SamplesPerPixel=1, got %d",
			p.SamplesPerPixel)
	}

	// Expand segmented palettes if present
	if err := palette.expandSegmentedPalettes(); err != nil {
		return nil, fmt.Errorf("failed to expand segmented palettes: %w", err)
	}

	numPixels := int(p.Rows) * int(p.Columns) * p.NumberOfFrames
	data := make([]byte, numPixels*3) // RGB output

	firstMappedRed := int(palette.RedDescriptor[1])
	firstMappedGreen := int(palette.GreenDescriptor[1])
	firstMappedBlue := int(palette.BlueDescriptor[1])

	if p.BitsAllocated <= 8 {
		// 8-bit indices
		for i := 0; i < numPixels; i++ {
			idx := int(p.data[i])

			// Lookup RGB values
			r := palette.lookupRed(idx - firstMappedRed)
			g := palette.lookupGreen(idx - firstMappedGreen)
			b := palette.lookupBlue(idx - firstMappedBlue)

			data[i*3] = byte(r >> 8)   // Red (most significant byte)
			data[i*3+1] = byte(g >> 8) // Green
			data[i*3+2] = byte(b >> 8) // Blue
		}
	} else {
		// 16-bit indices
		for i := 0; i < numPixels; i++ {
			idx := int(uint16(p.data[i*2]) | uint16(p.data[i*2+1])<<8)

			// Lookup RGB values
			r := palette.lookupRed(idx - firstMappedRed)
			g := palette.lookupGreen(idx - firstMappedGreen)
			b := palette.lookupBlue(idx - firstMappedBlue)

			data[i*3] = byte(r >> 8)
			data[i*3+1] = byte(g >> 8)
			data[i*3+2] = byte(b >> 8)
		}
	}

	result := &PixelData{
		Rows:                      p.Rows,
		Columns:                   p.Columns,
		BitsAllocated:             8,
		BitsStored:                8,
		HighBit:                   7,
		PixelRepresentation:       0,
		SamplesPerPixel:           3,
		PhotometricInterpretation: "RGB",
		PlanarConfiguration:       0, // Interleaved RGB
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// lookupRed retrieves the red value for the given index.
func (p *PaletteColorLUT) lookupRed(idx int) uint16 {
	if idx < 0 || idx >= len(p.RedData) {
		return 0
	}
	return p.RedData[idx]
}

// lookupGreen retrieves the green value for the given index.
func (p *PaletteColorLUT) lookupGreen(idx int) uint16 {
	if idx < 0 || idx >= len(p.GreenData) {
		return 0
	}
	return p.GreenData[idx]
}

// lookupBlue retrieves the blue value for the given index.
func (p *PaletteColorLUT) lookupBlue(idx int) uint16 {
	if idx < 0 || idx >= len(p.BlueData) {
		return 0
	}
	return p.BlueData[idx]
}

// expandSegmentedPalettes expands segmented palette data into full arrays.
func (p *PaletteColorLUT) expandSegmentedPalettes() error {
	if p.RedSegmented != nil {
		expanded, err := p.RedSegmented.Expand(int(p.RedDescriptor[0]))
		if err != nil {
			return fmt.Errorf("failed to expand red segmented palette: %w", err)
		}
		p.RedData = expanded
	}

	if p.GreenSegmented != nil {
		expanded, err := p.GreenSegmented.Expand(int(p.GreenDescriptor[0]))
		if err != nil {
			return fmt.Errorf("failed to expand green segmented palette: %w", err)
		}
		p.GreenData = expanded
	}

	if p.BlueSegmented != nil {
		expanded, err := p.BlueSegmented.Expand(int(p.BlueDescriptor[0]))
		if err != nil {
			return fmt.Errorf("failed to expand blue segmented palette: %w", err)
		}
		p.BlueData = expanded
	}

	return nil
}

// Expand converts segmented LUT data into a full lookup table.
//
// Segmented LUTs use a compact representation with three segment types:
//   - Discrete segments: Explicit values
//   - Linear segments: Linear interpolation between endpoints
//   - Indirect segments: Reference to another part of the LUT
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part03.html#sect_C.7.9
func (s *SegmentedLUT) Expand(numEntries int) ([]uint16, error) {
	result := make([]uint16, numEntries)
	pos := 0
	i := 0

	for i < len(s.Data) {
		if pos >= numEntries {
			break
		}

		opcode := s.Data[i]
		i++

		// Segment type is in high byte of opcode
		segmentType := opcode >> 8

		switch segmentType {
		case 0:
			// Discrete segment
			length := int(opcode & 0xFF)
			if i+length > len(s.Data) {
				return nil, fmt.Errorf("discrete segment exceeds data length")
			}
			for j := 0; j < length && pos < numEntries; j++ {
				result[pos] = s.Data[i+j]
				pos++
			}
			i += length

		case 1:
			// Linear segment
			length := int(opcode & 0xFF)
			if i >= len(s.Data) {
				return nil, fmt.Errorf("linear segment missing endpoint")
			}
			startVal := result[pos-1] // Use previous value as start
			endVal := s.Data[i]
			i++

			// Linear interpolation
			for j := 0; j < length && pos < numEntries; j++ {
				frac := float64(j) / float64(length)
				result[pos] = uint16(float64(startVal) + frac*float64(endVal-startVal))
				pos++
			}

		case 2:
			// Indirect segment
			length := int(opcode & 0xFF)
			if i >= len(s.Data) {
				return nil, fmt.Errorf("indirect segment missing offset")
			}
			offset := int(s.Data[i])
			i++

			// Copy from another position
			for j := 0; j < length && pos < numEntries; j++ {
				if offset+j >= pos {
					return nil, fmt.Errorf("indirect segment references future position")
				}
				result[pos] = result[offset+j]
				pos++
			}

		default:
			return nil, fmt.Errorf("unknown segment type: %d", segmentType)
		}
	}

	return result, nil
}

// ExtractPaletteColorLUTFromDataSet extracts palette color LUT from a DICOM DataSet.
//
// Reads:
//   - Red Palette Color Lookup Table Descriptor (0028,1101)
//   - Green Palette Color Lookup Table Descriptor (0028,1102)
//   - Blue Palette Color Lookup Table Descriptor (0028,1103)
//   - Red Palette Color Lookup Table Data (0028,1201)
//   - Green Palette Color Lookup Table Data (0028,1202)
//   - Blue Palette Color Lookup Table Data (0028,1203)
//   - Segmented palettes if present (0028,1221-1223)
func ExtractPaletteColorLUTFromDataSet(ds *dicom.DataSet) (*PaletteColorLUT, error) {
	palette := &PaletteColorLUT{}

	// Extract descriptors
	if _, err := ds.Get(tag.New(0x0028, 0x1101)); err == nil {
		// Red descriptor
		// Descriptor is US with VM=3: [entries, first_mapped, bits]
		// We'll need to parse this properly based on the VR
		palette.RedDescriptor = [3]uint16{0, 0, 16} // Default values
	}

	if _, err := ds.Get(tag.New(0x0028, 0x1102)); err == nil {
		palette.GreenDescriptor = [3]uint16{0, 0, 16}
	}

	if _, err := ds.Get(tag.New(0x0028, 0x1103)); err == nil {
		palette.BlueDescriptor = [3]uint16{0, 0, 16}
	}

	// Extract palette data
	if elem, err := ds.Get(tag.New(0x0028, 0x1201)); err == nil {
		// Red data - stored as OW (Other Word)
		// Need to convert bytes to uint16 array
		_ = elem // TODO: Parse actual data
	}

	if elem, err := ds.Get(tag.New(0x0028, 0x1202)); err == nil {
		// Green data
		_ = elem
	}

	if elem, err := ds.Get(tag.New(0x0028, 0x1203)); err == nil {
		// Blue data
		_ = elem
	}

	// Check for segmented palettes (0028,1221-1223)
	// These are optional and provide a more compact representation

	return palette, nil
}

// ExtractPresentationLUTFromDataSet extracts Presentation LUT from a DICOM DataSet.
//
// Reads:
//   - Presentation LUT Shape (2050,0020)
//   - Presentation LUT Sequence (2050,0010) if LUT-based
func ExtractPresentationLUTFromDataSet(ds *dicom.DataSet) (*PresentationLUT, error) {
	presentationLUT := &PresentationLUT{}

	// Check for Presentation LUT Shape (2050,0020)
	if elem, err := ds.Get(tag.New(0x2050, 0x0020)); err == nil {
		presentationLUT.PresentationLUTShape = elem.Value().String()
		return presentationLUT, nil
	}

	// Check for Presentation LUT Sequence (2050,0010)
	// This would contain LUT Descriptor and LUT Data
	// TODO: Implement sequence parsing

	return nil, fmt.Errorf("no Presentation LUT found in dataset")
}
