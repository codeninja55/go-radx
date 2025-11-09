package pixel

import (
	"fmt"
)

// PixelDataBuilder provides a fluent interface for creating new pixel data.
//
// Example:
//
//	builder := pixel.NewPixelDataBuilder().
//	    WithDimensions(512, 512).
//	    WithBitsAllocated(16).
//	    WithPhotometricInterpretation("MONOCHROME2").
//	    WithPixelData(rawBytes)
//
//	pixelData, err := builder.Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
type PixelDataBuilder struct {
	rows                      uint16
	columns                   uint16
	bitsAllocated             uint16
	bitsStored                uint16
	highBit                   uint16
	pixelRepresentation       uint16
	samplesPerPixel           uint16
	photometricInterpretation string
	planarConfiguration       uint16
	numberOfFrames            int
	data                      []byte
	transferSyntaxUID         string
}

// NewPixelDataBuilder creates a new builder for PixelData.
//
// Default values:
//   - BitsStored = BitsAllocated
//   - HighBit = BitsStored - 1
//   - PixelRepresentation = 0 (unsigned)
//   - SamplesPerPixel = 1 (grayscale)
//   - PhotometricInterpretation = "MONOCHROME2"
//   - PlanarConfiguration = 0 (interleaved)
//   - NumberOfFrames = 1
func NewPixelDataBuilder() *PixelDataBuilder {
	return &PixelDataBuilder{
		pixelRepresentation:       0,
		samplesPerPixel:           1,
		photometricInterpretation: "MONOCHROME2",
		planarConfiguration:       0,
		numberOfFrames:            1,
	}
}

// WithDimensions sets the image dimensions (width x height).
func (b *PixelDataBuilder) WithDimensions(columns, rows uint16) *PixelDataBuilder {
	b.columns = columns
	b.rows = rows
	return b
}

// WithBitsAllocated sets the number of bits allocated per pixel sample.
//
// Common values: 8, 12, 16
// If not specified, auto-detected from pixel data type.
func (b *PixelDataBuilder) WithBitsAllocated(bits uint16) *PixelDataBuilder {
	b.bitsAllocated = bits
	// Auto-set BitsStored and HighBit if not explicitly set
	if b.bitsStored == 0 {
		b.bitsStored = bits
	}
	if b.highBit == 0 {
		b.highBit = bits - 1
	}
	return b
}

// WithBitsStored sets the number of bits actually stored per pixel sample.
//
// Must be <= BitsAllocated. Defaults to BitsAllocated if not specified.
func (b *PixelDataBuilder) WithBitsStored(bits uint16) *PixelDataBuilder {
	b.bitsStored = bits
	if b.highBit == 0 {
		b.highBit = bits - 1
	}
	return b
}

// WithHighBit sets the most significant bit for pixel sample values.
//
// Defaults to BitsStored - 1 if not specified.
func (b *PixelDataBuilder) WithHighBit(bit uint16) *PixelDataBuilder {
	b.highBit = bit
	return b
}

// WithPixelRepresentation sets whether pixel data is signed or unsigned.
//
// 0 = unsigned, 1 = signed (2's complement)
// Default: 0 (unsigned)
func (b *PixelDataBuilder) WithPixelRepresentation(representation uint16) *PixelDataBuilder {
	b.pixelRepresentation = representation
	return b
}

// WithSamplesPerPixel sets the number of samples per pixel.
//
// 1 = grayscale, 3 = RGB
// Default: 1
func (b *PixelDataBuilder) WithSamplesPerPixel(samples uint16) *PixelDataBuilder {
	b.samplesPerPixel = samples
	return b
}

// WithPhotometricInterpretation sets the color space.
//
// Common values: "MONOCHROME1", "MONOCHROME2", "RGB", "YBR_FULL", "YBR_FULL_422"
// Default: "MONOCHROME2"
func (b *PixelDataBuilder) WithPhotometricInterpretation(pi string) *PixelDataBuilder {
	b.photometricInterpretation = pi
	return b
}

// WithPlanarConfiguration sets the pixel data organization for multi-sample pixels.
//
// 0 = interleaved (RGBRGBRGB...)
// 1 = planar (RRR...GGG...BBB...)
// Default: 0
//
// Only relevant when SamplesPerPixel > 1
func (b *PixelDataBuilder) WithPlanarConfiguration(config uint16) *PixelDataBuilder {
	b.planarConfiguration = config
	return b
}

// WithNumberOfFrames sets the number of frames in a multi-frame dataset.
//
// Default: 1
func (b *PixelDataBuilder) WithNumberOfFrames(frames int) *PixelDataBuilder {
	b.numberOfFrames = frames
	return b
}

// WithPixelData sets the raw pixel data bytes.
//
// The byte array should match the expected size based on dimensions and bit depth.
func (b *PixelDataBuilder) WithPixelData(data []byte) *PixelDataBuilder {
	b.data = data
	return b
}

// WithTransferSyntaxUID sets the transfer syntax UID.
//
// Used for tracking the origin of pixel data.
// Default: empty string (native/uncompressed)
func (b *PixelDataBuilder) WithTransferSyntaxUID(uid string) *PixelDataBuilder {
	b.transferSyntaxUID = uid
	return b
}

// Build constructs the PixelData with validation.
//
// Returns an error if:
//   - Required fields are missing (dimensions, pixel data)
//   - Pixel data size doesn't match expected size
//   - Configuration is invalid
func (b *PixelDataBuilder) Build() (*PixelData, error) {
	// Validate required fields
	if b.rows == 0 || b.columns == 0 {
		return nil, fmt.Errorf("dimensions not set (rows=%d, columns=%d)", b.rows, b.columns)
	}

	if len(b.data) == 0 {
		return nil, fmt.Errorf("pixel data not set")
	}

	if b.bitsAllocated == 0 {
		return nil, fmt.Errorf("bits allocated not set")
	}

	// Auto-set derived values
	if b.bitsStored == 0 {
		b.bitsStored = b.bitsAllocated
	}
	if b.highBit == 0 {
		b.highBit = b.bitsStored - 1
	}

	// Validate bit relationships
	if b.bitsStored > b.bitsAllocated {
		return nil, fmt.Errorf("bits stored (%d) cannot exceed bits allocated (%d)",
			b.bitsStored, b.bitsAllocated)
	}
	if b.highBit >= b.bitsStored {
		return nil, fmt.Errorf("high bit (%d) must be < bits stored (%d)",
			b.highBit, b.bitsStored)
	}

	// Validate samples per pixel
	if b.samplesPerPixel == 0 {
		return nil, fmt.Errorf("samples per pixel cannot be 0")
	}

	// Validate photometric interpretation
	if b.photometricInterpretation == "" {
		return nil, fmt.Errorf("photometric interpretation not set")
	}

	// Validate planar configuration for multi-sample pixels
	if b.samplesPerPixel > 1 && b.planarConfiguration > 1 {
		return nil, fmt.Errorf("invalid planar configuration: %d (must be 0 or 1)", b.planarConfiguration)
	}

	// Calculate expected size
	bytesPerSample := int(b.bitsAllocated+7) / 8
	expectedSize := int(b.rows) * int(b.columns) * int(b.samplesPerPixel) * b.numberOfFrames * bytesPerSample

	if len(b.data) != expectedSize {
		return nil, fmt.Errorf("pixel data size mismatch: expected %d bytes, got %d bytes",
			expectedSize, len(b.data))
	}

	// Build PixelData
	return &PixelData{
		Rows:                      b.rows,
		Columns:                   b.columns,
		BitsAllocated:             b.bitsAllocated,
		BitsStored:                b.bitsStored,
		HighBit:                   b.highBit,
		PixelRepresentation:       b.pixelRepresentation,
		SamplesPerPixel:           b.samplesPerPixel,
		PhotometricInterpretation: b.photometricInterpretation,
		PlanarConfiguration:       b.planarConfiguration,
		NumberOfFrames:            b.numberOfFrames,
		data:                      b.data,
		TransferSyntaxUID:         b.transferSyntaxUID,
	}, nil
}

// NewPixelDataFromUint8 creates PixelData from an 8-bit unsigned grayscale array.
//
// Parameters:
//   - data: Pixel values as uint8 (0-255)
//   - width: Image width in pixels
//   - height: Image height in pixels
//
// Returns PixelData configured as MONOCHROME2, 8-bit unsigned.
//
// Example:
//
//	data := make([]uint8, 512*512)
//	// Fill data...
//	pixelData, err := pixel.NewPixelDataFromUint8(data, 512, 512)
func NewPixelDataFromUint8(data []uint8, width, height int) (*PixelData, error) {
	return NewPixelDataBuilder().
		WithDimensions(uint16(width), uint16(height)).
		WithBitsAllocated(8).
		WithPixelRepresentation(0).
		WithPhotometricInterpretation("MONOCHROME2").
		WithPixelData(data).
		Build()
}

// NewPixelDataFromUint16 creates PixelData from a 16-bit unsigned grayscale array.
//
// Parameters:
//   - data: Pixel values as uint16 (0-65535)
//   - width: Image width in pixels
//   - height: Image height in pixels
//
// Returns PixelData configured as MONOCHROME2, 16-bit unsigned.
//
// Example:
//
//	data := make([]uint16, 512*512)
//	// Fill data...
//	pixelData, err := pixel.NewPixelDataFromUint16(data, 512, 512)
func NewPixelDataFromUint16(data []uint16, width, height int) (*PixelData, error) {
	// Convert uint16 to byte array (little endian)
	byteData := make([]byte, len(data)*2)
	for i, val := range data {
		byteData[i*2] = byte(val)
		byteData[i*2+1] = byte(val >> 8)
	}

	return NewPixelDataBuilder().
		WithDimensions(uint16(width), uint16(height)).
		WithBitsAllocated(16).
		WithPixelRepresentation(0).
		WithPhotometricInterpretation("MONOCHROME2").
		WithPixelData(byteData).
		Build()
}

// NewPixelDataFromInt16 creates PixelData from a 16-bit signed grayscale array.
//
// Parameters:
//   - data: Pixel values as int16 (-32768 to 32767)
//   - width: Image width in pixels
//   - height: Image height in pixels
//
// Returns PixelData configured as MONOCHROME2, 16-bit signed.
//
// Common in CT images where Hounsfield units can be negative.
//
// Example:
//
//	data := make([]int16, 512*512)
//	// Fill data with CT Hounsfield units...
//	pixelData, err := pixel.NewPixelDataFromInt16(data, 512, 512)
func NewPixelDataFromInt16(data []int16, width, height int) (*PixelData, error) {
	// Convert int16 to byte array (little endian)
	byteData := make([]byte, len(data)*2)
	for i, val := range data {
		uval := uint16(val)
		byteData[i*2] = byte(uval)
		byteData[i*2+1] = byte(uval >> 8)
	}

	return NewPixelDataBuilder().
		WithDimensions(uint16(width), uint16(height)).
		WithBitsAllocated(16).
		WithPixelRepresentation(1). // Signed
		WithPhotometricInterpretation("MONOCHROME2").
		WithPixelData(byteData).
		Build()
}

// NewPixelDataFromRGB creates PixelData from an RGB color array.
//
// Parameters:
//   - data: Pixel values as interleaved RGB bytes (RGBRGBRGB...)
//   - width: Image width in pixels
//   - height: Image height in pixels
//
// Returns PixelData configured as RGB, 8-bit per channel, interleaved.
//
// Example:
//
//	data := make([]byte, 512*512*3)
//	// Fill data with RGB values...
//	pixelData, err := pixel.NewPixelDataFromRGB(data, 512, 512)
func NewPixelDataFromRGB(data []byte, width, height int) (*PixelData, error) {
	return NewPixelDataBuilder().
		WithDimensions(uint16(width), uint16(height)).
		WithBitsAllocated(8).
		WithSamplesPerPixel(3).
		WithPhotometricInterpretation("RGB").
		WithPlanarConfiguration(0). // Interleaved
		WithPixelData(data).
		Build()
}

// NewPixelDataFromRGBPlanar creates PixelData from RGB color planes.
//
// Parameters:
//   - r: Red channel data
//   - g: Green channel data
//   - b: Blue channel data
//   - width: Image width in pixels
//   - height: Image height in pixels
//
// Returns PixelData configured as RGB, 8-bit per channel, planar (RRR...GGG...BBB...).
//
// Example:
//
//	size := 512 * 512
//	r := make([]byte, size)
//	g := make([]byte, size)
//	b := make([]byte, size)
//	// Fill channels...
//	pixelData, err := pixel.NewPixelDataFromRGBPlanar(r, g, b, 512, 512)
func NewPixelDataFromRGBPlanar(r, g, b []byte, width, height int) (*PixelData, error) {
	expectedSize := width * height
	if len(r) != expectedSize || len(g) != expectedSize || len(b) != expectedSize {
		return nil, fmt.Errorf("channel size mismatch: expected %d bytes per channel, got R=%d, G=%d, B=%d",
			expectedSize, len(r), len(g), len(b))
	}

	// Concatenate planes: RRR...GGG...BBB...
	data := make([]byte, expectedSize*3)
	copy(data[0:expectedSize], r)
	copy(data[expectedSize:2*expectedSize], g)
	copy(data[2*expectedSize:3*expectedSize], b)

	return NewPixelDataBuilder().
		WithDimensions(uint16(width), uint16(height)).
		WithBitsAllocated(8).
		WithSamplesPerPixel(3).
		WithPhotometricInterpretation("RGB").
		WithPlanarConfiguration(1). // Planar
		WithPixelData(data).
		Build()
}
