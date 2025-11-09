package pixel

import (
	"fmt"
	"image"
	"image/color"
)

// PixelData represents decompressed DICOM pixel data with associated metadata.
type PixelData struct {
	// Dimensional attributes
	Rows    uint16 // Number of rows (height)
	Columns uint16 // Number of columns (width)

	// Pixel representation attributes
	BitsAllocated       uint16 // Number of bits allocated for each pixel sample
	BitsStored          uint16 // Number of bits actually stored for each pixel sample
	HighBit             uint16 // Most significant bit for pixel sample value
	PixelRepresentation uint16 // 0 = unsigned, 1 = signed (2's complement)

	// Color attributes
	SamplesPerPixel           uint16 // Number of samples per pixel (1 = grayscale, 3 = RGB)
	PhotometricInterpretation string // Color space (MONOCHROME1, MONOCHROME2, RGB, YBR_FULL, etc.)
	PlanarConfiguration       uint16 // 0 = interleaved (RGBRGB...), 1 = planar (RRR...GGG...BBB...)

	// Multi-frame attributes
	NumberOfFrames int // Number of frames in the dataset (1 for single-frame)

	// Decompressed pixel values
	data []byte // Raw pixel data bytes

	// Transfer syntax
	TransferSyntaxUID string // Transfer syntax used for decompression
}

// Frame represents a single frame from a multi-frame pixel data.
type Frame struct {
	Index               int    // Frame index (0-based)
	Rows                uint16 // Frame height
	Columns             uint16 // Frame width
	BitsAllocated       uint16
	BitsStored          uint16
	PixelRepresentation uint16
	SamplesPerPixel     uint16
	data                []byte // Frame pixel data
}

// Array returns the pixel data as a typed slice based on BitsAllocated and PixelRepresentation.
//
// Returns:
//   - []uint8 for BitsAllocated <= 8, unsigned
//   - []uint16 for 9 <= BitsAllocated <= 16, unsigned
//   - []int16 for 9 <= BitsAllocated <= 16, signed
//
// For multi-frame datasets, this returns all frames concatenated.
func (p *PixelData) Array() interface{} {
	if p.PixelRepresentation == 1 {
		// Signed pixel data
		if p.BitsAllocated <= 8 {
			// Convert bytes to int8
			result := make([]int8, len(p.data))
			for i, b := range p.data {
				result[i] = int8(b)
			}
			return result
		}
		// Convert to int16
		result := make([]int16, len(p.data)/2)
		for i := 0; i < len(result); i++ {
			result[i] = int16(uint16(p.data[i*2]) | uint16(p.data[i*2+1])<<8)
		}
		return result
	}

	// Unsigned pixel data
	if p.BitsAllocated <= 8 {
		return p.data
	}

	// Convert to uint16
	result := make([]uint16, len(p.data)/2)
	for i := 0; i < len(result); i++ {
		result[i] = uint16(p.data[i*2]) | uint16(p.data[i*2+1])<<8
	}
	return result
}

// Frames returns individual frames from a multi-frame dataset.
//
// For single-frame datasets, returns a slice with one frame.
func (p *PixelData) Frames() []Frame {
	if p.NumberOfFrames <= 1 {
		return []Frame{{
			Index:               0,
			Rows:                p.Rows,
			Columns:             p.Columns,
			BitsAllocated:       p.BitsAllocated,
			BitsStored:          p.BitsStored,
			PixelRepresentation: p.PixelRepresentation,
			SamplesPerPixel:     p.SamplesPerPixel,
			data:                p.data,
		}}
	}

	// Calculate frame size in bytes
	bytesPerSample := int(p.BitsAllocated+7) / 8
	frameSize := int(p.Rows) * int(p.Columns) * int(p.SamplesPerPixel) * bytesPerSample

	frames := make([]Frame, p.NumberOfFrames)
	for i := 0; i < p.NumberOfFrames; i++ {
		start := i * frameSize
		end := start + frameSize
		if end > len(p.data) {
			end = len(p.data)
		}

		frames[i] = Frame{
			Index:               i,
			Rows:                p.Rows,
			Columns:             p.Columns,
			BitsAllocated:       p.BitsAllocated,
			BitsStored:          p.BitsStored,
			PixelRepresentation: p.PixelRepresentation,
			SamplesPerPixel:     p.SamplesPerPixel,
			data:                p.data[start:end],
		}
	}

	return frames
}

// Image converts pixel data to Go's standard image.Image interface.
//
// Returns:
//   - *image.Gray for 8-bit grayscale
//   - *image.Gray16 for 16-bit grayscale
//   - *image.RGBA for RGB color images
//
// For multi-frame datasets, only the first frame is returned.
// Use Frames() to access individual frames.
func (p *PixelData) Image() image.Image {
	if p.SamplesPerPixel == 1 {
		// Grayscale image
		return p.grayscaleImage()
	}
	// Color image (RGB)
	return p.rgbImage()
}

// grayscaleImage converts grayscale pixel data to image.Gray or image.Gray16.
func (p *PixelData) grayscaleImage() image.Image {
	rect := image.Rect(0, 0, int(p.Columns), int(p.Rows))

	if p.BitsAllocated <= 8 {
		img := image.NewGray(rect)
		copy(img.Pix, p.data)
		return img
	}

	// 16-bit grayscale
	img := image.NewGray16(rect)
	for i := 0; i < len(img.Pix)/2; i++ {
		img.Pix[i*2] = p.data[i*2+1] // High byte
		img.Pix[i*2+1] = p.data[i*2] // Low byte
	}
	return img
}

// rgbImage converts RGB pixel data to image.RGBA.
func (p *PixelData) rgbImage() image.Image {
	rect := image.Rect(0, 0, int(p.Columns), int(p.Rows))
	img := image.NewRGBA(rect)

	// Handle planar vs interleaved configuration
	if p.PlanarConfiguration == 0 {
		// Interleaved: RGBRGBRGB...
		for i := 0; i < int(p.Rows)*int(p.Columns); i++ {
			img.Pix[i*4] = p.data[i*3]     // R
			img.Pix[i*4+1] = p.data[i*3+1] // G
			img.Pix[i*4+2] = p.data[i*3+2] // B
			img.Pix[i*4+3] = 255           // A (fully opaque)
		}
	} else {
		// Planar: RRR...GGG...BBB...
		planeSize := int(p.Rows) * int(p.Columns)
		for i := 0; i < planeSize; i++ {
			img.Pix[i*4] = p.data[i]               // R
			img.Pix[i*4+1] = p.data[planeSize+i]   // G
			img.Pix[i*4+2] = p.data[2*planeSize+i] // B
			img.Pix[i*4+3] = 255                   // A (fully opaque)
		}
	}

	return img
}

// Array returns the frame's pixel data as a typed slice.
func (f *Frame) Array() interface{} {
	if f.PixelRepresentation == 1 {
		// Signed pixel data
		if f.BitsAllocated <= 8 {
			result := make([]int8, len(f.data))
			for i, b := range f.data {
				result[i] = int8(b)
			}
			return result
		}
		// Convert to int16
		result := make([]int16, len(f.data)/2)
		for i := 0; i < len(result); i++ {
			result[i] = int16(uint16(f.data[i*2]) | uint16(f.data[i*2+1])<<8)
		}
		return result
	}

	// Unsigned pixel data
	if f.BitsAllocated <= 8 {
		return f.data
	}

	// Convert to uint16
	result := make([]uint16, len(f.data)/2)
	for i := 0; i < len(result); i++ {
		result[i] = uint16(f.data[i*2]) | uint16(f.data[i*2+1])<<8
	}
	return result
}

// Image converts the frame to image.Image.
func (f *Frame) Image() image.Image {
	rect := image.Rect(0, 0, int(f.Columns), int(f.Rows))

	if f.SamplesPerPixel == 1 {
		// Grayscale
		if f.BitsAllocated <= 8 {
			img := image.NewGray(rect)
			copy(img.Pix, f.data)
			return img
		}
		// 16-bit grayscale
		img := image.NewGray16(rect)
		for i := 0; i < len(img.Pix)/2; i++ {
			img.Pix[i*2] = f.data[i*2+1] // High byte
			img.Pix[i*2+1] = f.data[i*2] // Low byte
		}
		return img
	}

	// RGB (simplified - assumes interleaved)
	img := image.NewRGBA(rect)
	for i := 0; i < int(f.Rows)*int(f.Columns); i++ {
		img.Pix[i*4] = f.data[i*3]     // R
		img.Pix[i*4+1] = f.data[i*3+1] // G
		img.Pix[i*4+2] = f.data[i*3+2] // B
		img.Pix[i*4+3] = 255           // A
	}
	return img
}

// At returns the color of the pixel at (x, y).
func (f *Frame) At(x, y int) color.Color {
	if x < 0 || x >= int(f.Columns) || y < 0 || y >= int(f.Rows) {
		return color.RGBA{}
	}

	if f.SamplesPerPixel == 1 {
		// Grayscale
		idx := y*int(f.Columns) + x
		if f.BitsAllocated <= 8 {
			return color.Gray{Y: f.data[idx]}
		}
		val := uint16(f.data[idx*2]) | uint16(f.data[idx*2+1])<<8
		return color.Gray16{Y: val}
	}

	// RGB
	idx := (y*int(f.Columns) + x) * 3
	return color.RGBA{
		R: f.data[idx],
		G: f.data[idx+1],
		B: f.data[idx+2],
		A: 255,
	}
}

// RawBytes returns the raw pixel data bytes.
//
// This provides direct access to the underlying pixel data for performance-sensitive
// operations like benchmarking or custom processing.
func (p *PixelData) RawBytes() []byte {
	return p.data
}

// String returns a human-readable description of the pixel data.
func (p *PixelData) String() string {
	return fmt.Sprintf("PixelData{%dx%dx%d, %d bits, %s, %d frames}",
		p.Columns, p.Rows, p.SamplesPerPixel, p.BitsStored,
		p.PhotometricInterpretation, p.NumberOfFrames)
}

// String returns a human-readable description of the frame.
func (f *Frame) String() string {
	return fmt.Sprintf("Frame{%d: %dx%d, %d bits}",
		f.Index, f.Columns, f.Rows, f.BitsStored)
}
