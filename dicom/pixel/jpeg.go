package pixel

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
)

// JPEGBaselineDecoder implements JPEG Baseline decompression using stdlib image/jpeg.
//
// JPEG Baseline is specified in:
//   - Transfer Syntax 1.2.840.10008.1.2.4.50: JPEG Baseline (Process 1) - 8-bit lossy
//   - Transfer Syntax 1.2.840.10008.1.2.4.51: JPEG Baseline (Processes 2 & 4) - 8-bit and 12-bit lossy
//
// This decoder uses Go's standard library image/jpeg package, which supports
// 8-bit JPEG Baseline compression. For 12-bit JPEG (Process 4), this decoder
// may not work correctly.
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_8.2.1
type JPEGBaselineDecoder struct {
	transferSyntaxUID string
}

// NewJPEGBaselineDecoder creates a new JPEG Baseline decoder for a specific transfer syntax.
func NewJPEGBaselineDecoder(transferSyntaxUID string) *JPEGBaselineDecoder {
	return &JPEGBaselineDecoder{
		transferSyntaxUID: transferSyntaxUID,
	}
}

// Decode decompresses JPEG Baseline encoded pixel data.
//
// The encapsulated data should be a valid JPEG stream. This decoder:
//  1. Decodes the JPEG stream using image/jpeg.Decode()
//  2. Converts the resulting image.Image to raw pixel bytes
//  3. Returns the decompressed pixel data in the expected format
//
// For grayscale images, returns 8-bit grayscale data.
// For RGB images, returns interleaved RGB data (RGBRGBRGB...).
func (d *JPEGBaselineDecoder) Decode(encapsulated []byte, info *PixelInfo) ([]byte, error) {
	if len(encapsulated) == 0 {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause:             fmt.Errorf("empty JPEG data"),
		}
	}

	// Decode JPEG using stdlib
	reader := bytes.NewReader(encapsulated)
	img, err := jpeg.Decode(reader)
	if err != nil {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause:             fmt.Errorf("JPEG decode failed: %w", err),
		}
	}

	// Verify image dimensions match expected dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width != int(info.Columns) || height != int(info.Rows) {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause: fmt.Errorf("image dimensions mismatch: got %dx%d, expected %dx%d",
				width, height, info.Columns, info.Rows),
		}
	}

	// Convert image.Image to raw pixel bytes
	var pixelData []byte

	switch imgTyped := img.(type) {
	case *image.Gray:
		// 8-bit grayscale
		if info.SamplesPerPixel != 1 {
			return nil, &DecompressionError{
				TransferSyntaxUID: d.transferSyntaxUID,
				Cause:             fmt.Errorf("grayscale image but SamplesPerPixel=%d (expected 1)", info.SamplesPerPixel),
			}
		}
		pixelData = imgTyped.Pix

	case *image.YCbCr:
		// YCbCr (color) image - need to convert to RGB
		if info.SamplesPerPixel != 3 {
			return nil, &DecompressionError{
				TransferSyntaxUID: d.transferSyntaxUID,
				Cause:             fmt.Errorf("color image but SamplesPerPixel=%d (expected 3)", info.SamplesPerPixel),
			}
		}
		pixelData = ycbcrToRGB(imgTyped)

	case *image.RGBA:
		// RGBA image - extract RGB only
		if info.SamplesPerPixel != 3 {
			return nil, &DecompressionError{
				TransferSyntaxUID: d.transferSyntaxUID,
				Cause:             fmt.Errorf("color image but SamplesPerPixel=%d (expected 3)", info.SamplesPerPixel),
			}
		}
		pixelData = rgbaToRGB(imgTyped)

	case *image.NRGBA:
		// NRGBA image - extract RGB only
		if info.SamplesPerPixel != 3 {
			return nil, &DecompressionError{
				TransferSyntaxUID: d.transferSyntaxUID,
				Cause:             fmt.Errorf("color image but SamplesPerPixel=%d (expected 3)", info.SamplesPerPixel),
			}
		}
		pixelData = nrgbaToRGB(imgTyped)

	default:
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause:             fmt.Errorf("unsupported image type: %T", img),
		}
	}

	// Validate output size
	expectedSize := CalculateExpectedSize(info)
	if len(pixelData) != expectedSize {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause:             fmt.Errorf("decompressed size mismatch: got %d bytes, expected %d bytes", len(pixelData), expectedSize),
		}
	}

	return pixelData, nil
}

// TransferSyntaxUID returns the transfer syntax UID this decoder handles.
func (d *JPEGBaselineDecoder) TransferSyntaxUID() string {
	return d.transferSyntaxUID
}

// ycbcrToRGB converts an image.YCbCr to interleaved RGB bytes.
//
// JPEG images are often decoded as YCbCr, but DICOM expects RGB for
// 3-channel pixel data. This function performs the color space conversion.
func ycbcrToRGB(img *image.YCbCr) []byte {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgb := make([]byte, width*height*3)
	idx := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get YCbCr values
			yi := img.YOffset(x, y)
			ci := img.COffset(x, y)

			yy := int32(img.Y[yi])
			cb := int32(img.Cb[ci])
			cr := int32(img.Cr[ci])

			// Convert YCbCr to RGB using JPEG color space conversion
			// R = Y + 1.402 * (Cr - 128)
			// G = Y - 0.344136 * (Cb - 128) - 0.714136 * (Cr - 128)
			// B = Y + 1.772 * (Cb - 128)
			r := yy + (91881*(cr-128))>>16
			g := yy - (22554*(cb-128))>>16 - (46802*(cr-128))>>16
			b := yy + (116130*(cb-128))>>16

			// Clamp to [0, 255]
			rgb[idx] = clampUint8(r)
			rgb[idx+1] = clampUint8(g)
			rgb[idx+2] = clampUint8(b)
			idx += 3
		}
	}

	return rgb
}

// rgbaToRGB extracts RGB bytes from image.RGBA (discarding alpha channel).
func rgbaToRGB(img *image.RGBA) []byte {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgb := make([]byte, width*height*3)
	srcIdx := 0
	dstIdx := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			rgb[dstIdx] = img.Pix[srcIdx]     // R
			rgb[dstIdx+1] = img.Pix[srcIdx+1] // G
			rgb[dstIdx+2] = img.Pix[srcIdx+2] // B
			// Skip alpha at srcIdx+3
			srcIdx += 4
			dstIdx += 3
		}
	}

	return rgb
}

// nrgbaToRGB extracts RGB bytes from image.NRGBA (discarding alpha channel).
func nrgbaToRGB(img *image.NRGBA) []byte {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgb := make([]byte, width*height*3)
	srcIdx := 0
	dstIdx := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			rgb[dstIdx] = img.Pix[srcIdx]     // R
			rgb[dstIdx+1] = img.Pix[srcIdx+1] // G
			rgb[dstIdx+2] = img.Pix[srcIdx+2] // B
			// Skip alpha at srcIdx+3
			srcIdx += 4
			dstIdx += 3
		}
	}

	return rgb
}

// clampUint8 clamps an int32 value to the uint8 range [0, 255].
func clampUint8(v int32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func init() {
	// Register JPEG Baseline decoders
	// Transfer Syntax 1.2.840.10008.1.2.4.50: JPEG Baseline (Process 1)
	RegisterDecoder("1.2.840.10008.1.2.4.50", NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.50"))

	// Transfer Syntax 1.2.840.10008.1.2.4.51: JPEG Baseline (Processes 2 & 4)
	RegisterDecoder("1.2.840.10008.1.2.4.51", NewJPEGBaselineDecoder("1.2.840.10008.1.2.4.51"))
}
