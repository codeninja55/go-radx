package pixel

import (
	"fmt"
	"math"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicom/tag"
)

// WindowLevel represents window/level parameters for VOI LUT transformation.
//
// Window/Level (also called Window Width/Window Center) is used to map the full
// range of pixel values to a display range optimized for viewing specific tissues.
//
// Example:
//   - Lung Window: Center=-600, Width=1500 (shows lung tissue detail)
//   - Mediastinum Window: Center=50, Width=350 (shows mediastinal structures)
//   - Bone Window: Center=300, Width=1500 (shows bone detail)
type WindowLevel struct {
	WindowCenter float64 // Center of window (WL)
	WindowWidth  float64 // Width of window (WW)
}

// ModalityLUT represents the Modality LUT transformation parameters.
//
// Used to convert raw pixel values to modality-specific units (e.g., Hounsfield Units for CT).
//
// Formula: output = m * pixel_value + b
// Where m = RescaleSlope, b = RescaleIntercept
type ModalityLUT struct {
	RescaleSlope     float64 // Multiplier (m in y = mx + b)
	RescaleIntercept float64 // Offset (b in y = mx + b)
	RescaleType      string  // Unit type (e.g., "HU" for Hounsfield Units)
}

// VOILUT represents VOI LUT transformation parameters for display.
//
// Can use either window/level parameters OR a lookup table, but not both.
type VOILUT struct {
	// Window/Level parameters (most common)
	WindowCenter *float64
	WindowWidth  *float64

	// LUT-based transformation (alternative to window/level)
	LUTData       []uint16 // Lookup table data
	LUTDescriptor [3]uint16
	// [num_entries, first_mapped_value, bits_per_entry]
	LUTExplanation string // Description of LUT purpose
}

// ApplyWindowLevel applies window/level transformation to pixel data.
//
// This is the most common VOI LUT transformation, used to optimize image contrast
// for viewing specific anatomical structures.
//
// The transformation maps pixel values to a display range [0, outputMax]:
//   - Values below (center - width/2) → 0
//   - Values above (center + width/2) → outputMax
//   - Values in between → linear mapping
//
// Parameters:
//   - p: Source pixel data
//   - center: Window center (WL)
//   - width: Window width (WW)
//   - outputBits: Output bit depth (8 or 16)
//
// Returns new PixelData with window/level applied.
//
// Example:
//
//	// Apply lung window to CT image
//	windowed, err := pixel.ApplyWindowLevel(pixelData, -600, 1500, 8)
func ApplyWindowLevel(p *PixelData, center, width float64, outputBits uint16) (*PixelData, error) {
	if width <= 0 {
		return nil, fmt.Errorf("window width must be positive, got %f", width)
	}

	if outputBits != 8 && outputBits != 16 {
		return nil, fmt.Errorf("output bits must be 8 or 16, got %d", outputBits)
	}

	if p.SamplesPerPixel != 1 {
		return nil, fmt.Errorf("window/level only applies to grayscale images (SamplesPerPixel=1), got %d",
			p.SamplesPerPixel)
	}

	outputMax := float64(uint16(1<<outputBits) - 1)
	lowerBound := center - width/2
	upperBound := center + width/2

	data := make([]byte, int(p.Rows)*int(p.Columns)*p.NumberOfFrames*int((outputBits+7)/8))

	if p.BitsAllocated <= 8 {
		// 8-bit input
		for i := 0; i < len(p.data); i++ {
			val := float64(p.data[i])
			if p.PixelRepresentation == 1 {
				// Signed
				val = float64(int8(p.data[i]))
			}

			// Apply window/level
			windowed := applyWindowLevelValue(val, lowerBound, upperBound, outputMax)

			if outputBits == 8 {
				data[i] = uint8(windowed)
			} else {
				// 16-bit output
				data[i*2] = byte(uint16(windowed))
				data[i*2+1] = byte(uint16(windowed) >> 8)
			}
		}
	} else {
		// 16-bit input
		for i := 0; i < len(p.data)/2; i++ {
			val16 := uint16(p.data[i*2]) | uint16(p.data[i*2+1])<<8
			var val float64
			if p.PixelRepresentation == 1 {
				// Signed
				val = float64(int16(val16))
			} else {
				val = float64(val16)
			}

			// Apply window/level
			windowed := applyWindowLevelValue(val, lowerBound, upperBound, outputMax)

			if outputBits == 8 {
				data[i] = uint8(windowed)
			} else {
				// 16-bit output
				data[i*2] = byte(uint16(windowed))
				data[i*2+1] = byte(uint16(windowed) >> 8)
			}
		}
	}

	result := &PixelData{
		Rows:                      p.Rows,
		Columns:                   p.Columns,
		BitsAllocated:             outputBits,
		BitsStored:                outputBits,
		HighBit:                   outputBits - 1,
		PixelRepresentation:       0, // Output is always unsigned
		SamplesPerPixel:           p.SamplesPerPixel,
		PhotometricInterpretation: p.PhotometricInterpretation,
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// applyWindowLevelValue applies window/level to a single pixel value.
func applyWindowLevelValue(val, lowerBound, upperBound, outputMax float64) float64 {
	if val <= lowerBound {
		return 0
	}
	if val >= upperBound {
		return outputMax
	}
	// Linear mapping
	return ((val - lowerBound) / (upperBound - lowerBound)) * outputMax
}

// ApplyModalityLUT applies modality LUT transformation to convert pixel values to modality units.
//
// This is typically the first transformation in the image pipeline, converting raw pixel
// values to meaningful physical units (e.g., Hounsfield Units for CT).
//
// Formula: output = RescaleSlope * input + RescaleIntercept
//
// Parameters:
//   - p: Source pixel data
//   - slope: Rescale slope (default 1.0)
//   - intercept: Rescale intercept (default 0.0)
//
// Returns new PixelData with modality LUT applied.
//
// Example:
//
//	// Apply HU conversion to CT image
//	// If RescaleSlope=1.0 and RescaleIntercept=-1024
//	hu, err := pixel.ApplyModalityLUT(pixelData, 1.0, -1024)
func ApplyModalityLUT(p *PixelData, slope, intercept float64) (*PixelData, error) {
	if p.SamplesPerPixel != 1 {
		return nil, fmt.Errorf("modality LUT only applies to grayscale images (SamplesPerPixel=1), got %d",
			p.SamplesPerPixel)
	}

	// Calculate output range to determine bit depth needed
	var minVal, maxVal float64
	if p.PixelRepresentation == 1 {
		// Signed
		if p.BitsAllocated <= 8 {
			minVal = float64(int8(-128))
			maxVal = float64(int8(127))
		} else {
			minVal = float64(int16(-(1 << (p.BitsStored - 1))))
			maxVal = float64(int16((1 << (p.BitsStored - 1)) - 1))
		}
	} else {
		// Unsigned
		minVal = 0
		if p.BitsAllocated <= 8 {
			maxVal = 255
		} else {
			maxVal = float64(uint16(1<<p.BitsStored) - 1)
		}
	}

	minOutput := slope*minVal + intercept
	maxOutput := slope*maxVal + intercept

	// Determine if output needs to be signed
	needsSigned := minOutput < 0
	outputBits := p.BitsAllocated

	// Calculate required bits
	absMax := math.Max(math.Abs(minOutput), math.Abs(maxOutput))
	if absMax > 32767 {
		outputBits = 16
	}

	data := make([]byte, len(p.data))

	if p.BitsAllocated <= 8 {
		// 8-bit input
		for i := 0; i < len(p.data); i++ {
			var val float64
			if p.PixelRepresentation == 1 {
				val = float64(int8(p.data[i]))
			} else {
				val = float64(p.data[i])
			}

			output := slope*val + intercept

			if outputBits <= 8 {
				if needsSigned {
					data[i] = byte(int8(output))
				} else {
					data[i] = byte(output)
				}
			} else {
				// 16-bit output
				if needsSigned {
					val16 := int16(output)
					data[i*2] = byte(val16)
					data[i*2+1] = byte(val16 >> 8)
				} else {
					val16 := uint16(output)
					data[i*2] = byte(val16)
					data[i*2+1] = byte(val16 >> 8)
				}
			}
		}
	} else {
		// 16-bit input
		for i := 0; i < len(p.data)/2; i++ {
			val16 := uint16(p.data[i*2]) | uint16(p.data[i*2+1])<<8
			var val float64
			if p.PixelRepresentation == 1 {
				val = float64(int16(val16))
			} else {
				val = float64(val16)
			}

			output := slope*val + intercept

			if needsSigned {
				outVal := int16(output)
				data[i*2] = byte(outVal)
				data[i*2+1] = byte(outVal >> 8)
			} else {
				outVal := uint16(output)
				data[i*2] = byte(outVal)
				data[i*2+1] = byte(outVal >> 8)
			}
		}
	}

	pixelRep := uint16(0)
	if needsSigned {
		pixelRep = 1
	}

	result := &PixelData{
		Rows:                      p.Rows,
		Columns:                   p.Columns,
		BitsAllocated:             outputBits,
		BitsStored:                outputBits,
		HighBit:                   outputBits - 1,
		PixelRepresentation:       pixelRep,
		SamplesPerPixel:           p.SamplesPerPixel,
		PhotometricInterpretation: p.PhotometricInterpretation,
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// ExtractWindowLevelFromDataSet extracts window/level parameters from a DICOM DataSet.
//
// Reads:
//   - (0028,1050) Window Center
//   - (0028,1051) Window Width
//
// Returns the first window/level if multiple are present.
func ExtractWindowLevelFromDataSet(ds *dicom.DataSet) (*WindowLevel, error) {
	// Window Center (0028,1050)
	centerElem, err := ds.Get(tag.New(0x0028, 0x1050))
	if err != nil {
		return nil, fmt.Errorf("window center not found: %w", err)
	}

	// Window Width (0028,1051)
	widthElem, err := ds.Get(tag.New(0x0028, 0x1051))
	if err != nil {
		return nil, fmt.Errorf("window width not found: %w", err)
	}

	// Parse values (can be multi-valued)
	centerStr := centerElem.Value().String()
	widthStr := widthElem.Value().String()

	var center, width float64
	if _, err := fmt.Sscanf(centerStr, "%f", &center); err != nil {
		return nil, fmt.Errorf("failed to parse window center: %w", err)
	}
	if _, err := fmt.Sscanf(widthStr, "%f", &width); err != nil {
		return nil, fmt.Errorf("failed to parse window width: %w", err)
	}

	return &WindowLevel{
		WindowCenter: center,
		WindowWidth:  width,
	}, nil
}

// ExtractModalityLUTFromDataSet extracts modality LUT parameters from a DICOM DataSet.
//
// Reads:
//   - (0028,1052) Rescale Intercept
//   - (0028,1053) Rescale Slope
//   - (0028,1054) Rescale Type (optional)
//
// Defaults to slope=1.0, intercept=0.0 if not present.
func ExtractModalityLUTFromDataSet(ds *dicom.DataSet) (*ModalityLUT, error) {
	result := &ModalityLUT{
		RescaleSlope:     1.0,
		RescaleIntercept: 0.0,
		RescaleType:      "",
	}

	// Rescale Intercept (0028,1052)
	if interceptElem, err := ds.Get(tag.New(0x0028, 0x1052)); err == nil {
		interceptStr := interceptElem.Value().String()
		if _, err := fmt.Sscanf(interceptStr, "%f", &result.RescaleIntercept); err != nil {
			return nil, fmt.Errorf("failed to parse rescale intercept: %w", err)
		}
	}

	// Rescale Slope (0028,1053)
	if slopeElem, err := ds.Get(tag.New(0x0028, 0x1053)); err == nil {
		slopeStr := slopeElem.Value().String()
		if _, err := fmt.Sscanf(slopeStr, "%f", &result.RescaleSlope); err != nil {
			return nil, fmt.Errorf("failed to parse rescale slope: %w", err)
		}
	}

	// Rescale Type (0028,1054) - optional
	if typeElem, err := ds.Get(tag.New(0x0028, 0x1054)); err == nil {
		result.RescaleType = typeElem.Value().String()
	}

	return result, nil
}

// ApplyFullImagePipeline applies the complete image transformation pipeline:
//  1. Modality LUT (if present) - converts to modality units
//  2. VOI LUT (window/level) - prepares for display
//
// This is the standard DICOM image display pipeline.
//
// Parameters:
//   - ds: DICOM DataSet containing LUT parameters
//   - p: Source pixel data
//   - outputBits: Output bit depth (8 for display, 16 for processing)
//
// Example:
//
//	// Apply complete pipeline to CT image
//	display, err := pixel.ApplyFullImagePipeline(dataset, pixelData, 8)
func ApplyFullImagePipeline(ds *dicom.DataSet, p *PixelData, outputBits uint16) (*PixelData, error) {
	result := p

	// Step 1: Apply Modality LUT if present
	if modalityLUT, err := ExtractModalityLUTFromDataSet(ds); err == nil {
		if modalityLUT.RescaleSlope != 1.0 || modalityLUT.RescaleIntercept != 0.0 {
			result, err = ApplyModalityLUT(result, modalityLUT.RescaleSlope, modalityLUT.RescaleIntercept)
			if err != nil {
				return nil, fmt.Errorf("failed to apply modality LUT: %w", err)
			}
		}
	}

	// Step 2: Apply VOI LUT (window/level) if present
	if windowLevel, err := ExtractWindowLevelFromDataSet(ds); err == nil {
		result, err = ApplyWindowLevel(result, windowLevel.WindowCenter, windowLevel.WindowWidth, outputBits)
		if err != nil {
			return nil, fmt.Errorf("failed to apply window/level: %w", err)
		}
	}

	return result, nil
}
