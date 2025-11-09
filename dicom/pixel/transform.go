package pixel

import (
	"fmt"
)

// ConvertPhotometricInterpretation converts pixel data between different color spaces.
//
// Supported conversions:
//   - RGB → YBR_FULL
//   - YBR_FULL → RGB
//   - MONOCHROME1 → MONOCHROME2 (inversion)
//   - MONOCHROME2 → MONOCHROME1 (inversion)
//
// Returns a new PixelData with the converted color space.
//
// Example:
//
//	converted, err := pixel.ConvertPhotometricInterpretation(pixelData, "RGB")
//	if err != nil {
//	    log.Fatal(err)
//	}
func ConvertPhotometricInterpretation(p *PixelData, targetPI string) (*PixelData, error) {
	sourcePI := p.PhotometricInterpretation

	// No conversion needed
	if sourcePI == targetPI {
		return p, nil
	}

	// Determine conversion path
	switch {
	case sourcePI == "RGB" && targetPI == "YBR_FULL":
		return convertRGBToYBR(p)
	case sourcePI == "RGB" && targetPI == "YBR_FULL_422":
		return convertRGBToYBR422(p)
	case sourcePI == "YBR_FULL" && targetPI == "RGB":
		return convertYBRToRGB(p)
	case sourcePI == "YBR_FULL_422" && targetPI == "RGB":
		return convertYBR422ToRGB(p)
	case sourcePI == "MONOCHROME1" && targetPI == "MONOCHROME2":
		return invertMonochrome(p)
	case sourcePI == "MONOCHROME2" && targetPI == "MONOCHROME1":
		return invertMonochrome(p)
	default:
		return nil, fmt.Errorf("unsupported photometric interpretation conversion: %s → %s",
			sourcePI, targetPI)
	}
}

// convertRGBToYBR converts RGB to YBR_FULL color space.
//
// YBR_FULL uses ITU-R BT.601 color space:
//
//	Y  =  0.299*R + 0.587*G + 0.114*B
//	Cb = -0.169*R - 0.331*G + 0.500*B + 128
//	Cr =  0.500*R - 0.419*G - 0.081*B + 128
func convertRGBToYBR(p *PixelData) (*PixelData, error) {
	if p.SamplesPerPixel != 3 {
		return nil, fmt.Errorf("RGB conversion requires SamplesPerPixel=3, got %d", p.SamplesPerPixel)
	}

	if p.BitsAllocated != 8 {
		return nil, fmt.Errorf("RGB → YBR conversion currently only supports 8-bit data")
	}

	data := make([]byte, len(p.data))

	if p.PlanarConfiguration == 0 {
		// Interleaved: RGBRGBRGB...
		for i := 0; i < len(p.data); i += 3 {
			r := float64(p.data[i])
			g := float64(p.data[i+1])
			b := float64(p.data[i+2])

			y := 0.299*r + 0.587*g + 0.114*b
			cb := -0.169*r - 0.331*g + 0.500*b + 128
			cr := 0.500*r - 0.419*g - 0.081*b + 128

			data[i] = clampUint8(int32(y))
			data[i+1] = clampUint8(int32(cb))
			data[i+2] = clampUint8(int32(cr))
		}
	} else {
		// Planar: RRR...GGG...BBB...
		planeSize := len(p.data) / 3
		for i := 0; i < planeSize; i++ {
			r := float64(p.data[i])
			g := float64(p.data[planeSize+i])
			b := float64(p.data[2*planeSize+i])

			y := 0.299*r + 0.587*g + 0.114*b
			cb := -0.169*r - 0.331*g + 0.500*b + 128
			cr := 0.500*r - 0.419*g - 0.081*b + 128

			data[i] = clampUint8(int32(y))
			data[planeSize+i] = clampUint8(int32(cb))
			data[2*planeSize+i] = clampUint8(int32(cr))
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
		PhotometricInterpretation: "YBR_FULL",
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// convertRGBToYBR422 converts RGB to YBR_FULL_422 (4:2:2 chroma subsampling).
//
// In 4:2:2 subsampling, Cb and Cr are horizontally subsampled by factor of 2.
func convertRGBToYBR422(p *PixelData) (*PixelData, error) {
	if p.SamplesPerPixel != 3 {
		return nil, fmt.Errorf("RGB conversion requires SamplesPerPixel=3, got %d", p.SamplesPerPixel)
	}

	if p.BitsAllocated != 8 {
		return nil, fmt.Errorf("RGB → YBR_FULL_422 conversion currently only supports 8-bit data")
	}

	if p.PlanarConfiguration != 0 {
		return nil, fmt.Errorf("YBR_FULL_422 requires interleaved data (PlanarConfiguration=0)")
	}

	// For 4:2:2, we subsample Cb/Cr horizontally
	// Output format: Y0 Cb Y1 Cr (2 pixels share Cb/Cr)
	data := make([]byte, len(p.data))

	for y := 0; y < int(p.Rows); y++ {
		for x := 0; x < int(p.Columns); x += 2 {
			idx := (y*int(p.Columns) + x) * 3

			// First pixel
			r1 := float64(p.data[idx])
			g1 := float64(p.data[idx+1])
			b1 := float64(p.data[idx+2])

			y1 := 0.299*r1 + 0.587*g1 + 0.114*b1

			// Second pixel (if exists)
			var y2, r2, g2, b2 float64
			if x+1 < int(p.Columns) {
				r2 = float64(p.data[idx+3])
				g2 = float64(p.data[idx+4])
				b2 = float64(p.data[idx+5])
				y2 = 0.299*r2 + 0.587*g2 + 0.114*b2
			} else {
				r2, g2, b2, y2 = r1, g1, b1, y1
			}

			// Average chroma from both pixels
			rAvg := (r1 + r2) / 2
			gAvg := (g1 + g2) / 2
			bAvg := (b1 + b2) / 2

			cb := -0.169*rAvg - 0.331*gAvg + 0.500*bAvg + 128
			cr := 0.500*rAvg - 0.419*gAvg - 0.081*bAvg + 128

			// Write Y0 Cb Y1 Cr
			data[idx] = clampUint8(int32(y1))
			data[idx+1] = clampUint8(int32(cb))
			if x+1 < int(p.Columns) {
				data[idx+3] = clampUint8(int32(y2))
				data[idx+4] = clampUint8(int32(cr))
			}
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
		PhotometricInterpretation: "YBR_FULL_422",
		PlanarConfiguration:       0, // 422 is always interleaved
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// convertYBRToRGB converts YBR_FULL to RGB color space.
//
// Inverse transformation of YBR → RGB using ITU-R BT.601:
//
//	R = Y                   + 1.402 * (Cr - 128)
//	G = Y - 0.344 * (Cb - 128) - 0.714 * (Cr - 128)
//	B = Y + 1.772 * (Cb - 128)
func convertYBRToRGB(p *PixelData) (*PixelData, error) {
	if p.SamplesPerPixel != 3 {
		return nil, fmt.Errorf("YBR conversion requires SamplesPerPixel=3, got %d", p.SamplesPerPixel)
	}

	if p.BitsAllocated != 8 {
		return nil, fmt.Errorf("YBR → RGB conversion currently only supports 8-bit data")
	}

	data := make([]byte, len(p.data))

	if p.PlanarConfiguration == 0 {
		// Interleaved: YCbCr YCbCr YCbCr...
		for i := 0; i < len(p.data); i += 3 {
			y := float64(p.data[i])
			cb := float64(p.data[i+1]) - 128
			cr := float64(p.data[i+2]) - 128

			r := y + 1.402*cr
			g := y - 0.344*cb - 0.714*cr
			b := y + 1.772*cb

			data[i] = clampUint8(int32(r))
			data[i+1] = clampUint8(int32(g))
			data[i+2] = clampUint8(int32(b))
		}
	} else {
		// Planar: YYY...CbCbCb...CrCrCr...
		planeSize := len(p.data) / 3
		for i := 0; i < planeSize; i++ {
			y := float64(p.data[i])
			cb := float64(p.data[planeSize+i]) - 128
			cr := float64(p.data[2*planeSize+i]) - 128

			r := y + 1.402*cr
			g := y - 0.344*cb - 0.714*cr
			b := y + 1.772*cb

			data[i] = clampUint8(int32(r))
			data[planeSize+i] = clampUint8(int32(g))
			data[2*planeSize+i] = clampUint8(int32(b))
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
		PhotometricInterpretation: "RGB",
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// convertYBR422ToRGB converts YBR_FULL_422 to RGB color space.
//
// Upsamples 4:2:2 chroma back to 4:4:4 RGB.
func convertYBR422ToRGB(p *PixelData) (*PixelData, error) {
	if p.SamplesPerPixel != 3 {
		return nil, fmt.Errorf("YBR_FULL_422 conversion requires SamplesPerPixel=3, got %d", p.SamplesPerPixel)
	}

	if p.BitsAllocated != 8 {
		return nil, fmt.Errorf("YBR_FULL_422 → RGB conversion currently only supports 8-bit data")
	}

	data := make([]byte, len(p.data))

	for y := 0; y < int(p.Rows); y++ {
		for x := 0; x < int(p.Columns); x += 2 {
			idx := (y*int(p.Columns) + x) * 3

			// Read Y0 Cb Y1 Cr (both pixels share Cb/Cr)
			y1 := float64(p.data[idx])
			cb := float64(p.data[idx+1]) - 128

			var y2, cr float64
			if x+1 < int(p.Columns) {
				y2 = float64(p.data[idx+3])
				cr = float64(p.data[idx+4]) - 128
			} else {
				y2 = y1
				cr = float64(p.data[idx+2]) - 128
			}

			// Convert both pixels using shared chroma
			r1 := y1 + 1.402*cr
			g1 := y1 - 0.344*cb - 0.714*cr
			b1 := y1 + 1.772*cb

			r2 := y2 + 1.402*cr
			g2 := y2 - 0.344*cb - 0.714*cr
			b2 := y2 + 1.772*cb

			data[idx] = clampUint8(int32(r1))
			data[idx+1] = clampUint8(int32(g1))
			data[idx+2] = clampUint8(int32(b1))

			if x+1 < int(p.Columns) {
				data[idx+3] = clampUint8(int32(r2))
				data[idx+4] = clampUint8(int32(g2))
				data[idx+5] = clampUint8(int32(b2))
			}
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
		PhotometricInterpretation: "RGB",
		PlanarConfiguration:       0, // Output is always interleaved
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// invertMonochrome inverts grayscale pixel values.
//
// Converts MONOCHROME1 ↔ MONOCHROME2:
//   - MONOCHROME1: Higher values = darker (0=white, max=black)
//   - MONOCHROME2: Higher values = brighter (0=black, max=white)
//
// Formula: inverted_value = max_value - original_value
func invertMonochrome(p *PixelData) (*PixelData, error) {
	if p.SamplesPerPixel != 1 {
		return nil, fmt.Errorf("monochrome inversion requires SamplesPerPixel=1, got %d", p.SamplesPerPixel)
	}

	data := make([]byte, len(p.data))

	if p.BitsAllocated <= 8 {
		// 8-bit inversion
		for i := 0; i < len(p.data); i++ {
			data[i] = 255 - p.data[i]
		}
	} else {
		// 16-bit inversion
		maxVal := uint16((1 << p.BitsStored) - 1)
		for i := 0; i < len(p.data)/2; i++ {
			val := uint16(p.data[i*2]) | uint16(p.data[i*2+1])<<8
			inverted := maxVal - val
			data[i*2] = byte(inverted)
			data[i*2+1] = byte(inverted >> 8)
		}
	}

	// Determine target PI
	targetPI := "MONOCHROME2"
	if p.PhotometricInterpretation == "MONOCHROME2" {
		targetPI = "MONOCHROME1"
	}

	result := &PixelData{
		Rows:                      p.Rows,
		Columns:                   p.Columns,
		BitsAllocated:             p.BitsAllocated,
		BitsStored:                p.BitsStored,
		HighBit:                   p.HighBit,
		PixelRepresentation:       p.PixelRepresentation,
		SamplesPerPixel:           p.SamplesPerPixel,
		PhotometricInterpretation: targetPI,
		PlanarConfiguration:       p.PlanarConfiguration,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}

// ConvertPlanarConfiguration converts between interleaved and planar pixel data organization.
//
// Converts:
//   - Planar (1) → Interleaved (0): RRR...GGG...BBB... → RGBRGBRGB...
//   - Interleaved (0) → Planar (1): RGBRGBRGB... → RRR...GGG...BBB...
//
// Only applicable when SamplesPerPixel > 1.
//
// Example:
//
//	// Convert from planar to interleaved
//	interleaved, err := pixel.ConvertPlanarConfiguration(pixelData, 0)
func ConvertPlanarConfiguration(p *PixelData, targetConfig uint16) (*PixelData, error) {
	if p.SamplesPerPixel <= 1 {
		return nil, fmt.Errorf("planar configuration conversion requires SamplesPerPixel > 1, got %d",
			p.SamplesPerPixel)
	}

	if targetConfig > 1 {
		return nil, fmt.Errorf("invalid planar configuration: %d (must be 0 or 1)", targetConfig)
	}

	// No conversion needed
	if p.PlanarConfiguration == targetConfig {
		return p, nil
	}

	if p.BitsAllocated != 8 {
		return nil, fmt.Errorf("planar configuration conversion currently only supports 8-bit data")
	}

	data := make([]byte, len(p.data))
	pixelCount := int(p.Rows) * int(p.Columns) * p.NumberOfFrames
	samplesPerPixel := int(p.SamplesPerPixel)

	if targetConfig == 0 {
		// Planar → Interleaved (RRR...GGG...BBB... → RGBRGBRGB...)
		planeSize := pixelCount
		for i := 0; i < pixelCount; i++ {
			for s := 0; s < samplesPerPixel; s++ {
				data[i*samplesPerPixel+s] = p.data[s*planeSize+i]
			}
		}
	} else {
		// Interleaved → Planar (RGBRGBRGB... → RRR...GGG...BBB...)
		planeSize := pixelCount
		for i := 0; i < pixelCount; i++ {
			for s := 0; s < samplesPerPixel; s++ {
				data[s*planeSize+i] = p.data[i*samplesPerPixel+s]
			}
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
		PlanarConfiguration:       targetConfig,
		NumberOfFrames:            p.NumberOfFrames,
		data:                      data,
		TransferSyntaxUID:         p.TransferSyntaxUID,
	}

	return result, nil
}
