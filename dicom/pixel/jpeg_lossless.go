//go:build cgo
// +build cgo

package pixel

/*
#cgo pkg-config: libjpeg
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <jpeglib.h>
#include <setjmp.h>

// Custom error handler structure
struct my_error_mgr {
    struct jpeg_error_mgr pub;
    jmp_buf setjmp_buffer;
    char message[JMSG_LENGTH_MAX];
};

typedef struct my_error_mgr * my_error_ptr;

// Error handler callback
void my_error_exit(j_common_ptr cinfo) {
    my_error_ptr myerr = (my_error_ptr) cinfo->err;
    (*cinfo->err->format_message)(cinfo, myerr->message);
    longjmp(myerr->setjmp_buffer, 1);
}

// Decompress JPEG lossless data
// Returns: 0 on success, -1 on error
int decompress_jpeg_lossless(
    unsigned char* input_data,
    unsigned long input_size,
    unsigned char* output_data,
    unsigned long output_size,
    int* width,
    int* height,
    int* components,
    int* bits_per_sample,
    char* error_message,
    int error_message_size
) {
    struct jpeg_decompress_struct cinfo;
    struct my_error_mgr jerr;
    JSAMPARRAY buffer;
    int row_stride;
    unsigned char* output_ptr = output_data;
    unsigned long bytes_written = 0;

    // Initialize error handler
    cinfo.err = jpeg_std_error(&jerr.pub);
    jerr.pub.error_exit = my_error_exit;

    if (setjmp(jerr.setjmp_buffer)) {
        // Error occurred during decompression
        jpeg_destroy_decompress(&cinfo);
        strncpy(error_message, jerr.message, error_message_size - 1);
        error_message[error_message_size - 1] = '\0';
        return -1;
    }

    // Create decompressor
    jpeg_create_decompress(&cinfo);

    // Set up data source from memory
    jpeg_mem_src(&cinfo, input_data, input_size);

    // Read JPEG header
    if (jpeg_read_header(&cinfo, TRUE) != JPEG_HEADER_OK) {
        jpeg_destroy_decompress(&cinfo);
        strncpy(error_message, "Failed to read JPEG header", error_message_size - 1);
        return -1;
    }

    // Start decompression
    jpeg_start_decompress(&cinfo);

    // Get image info
    *width = cinfo.output_width;
    *height = cinfo.output_height;
    *components = cinfo.output_components;
    *bits_per_sample = cinfo.data_precision;

    // Calculate row stride
    row_stride = cinfo.output_width * cinfo.output_components;

    // Handle different bit depths
    int bytes_per_sample = (cinfo.data_precision + 7) / 8;
    int row_stride_bytes = row_stride * bytes_per_sample;

    // Allocate scanline buffer
    buffer = (*cinfo.mem->alloc_sarray)
        ((j_common_ptr) &cinfo, JPOOL_IMAGE, row_stride_bytes, 1);

    // Read scanlines
    while (cinfo.output_scanline < cinfo.output_height) {
        jpeg_read_scanlines(&cinfo, buffer, 1);

        // Check output buffer size
        if (bytes_written + row_stride_bytes > output_size) {
            jpeg_destroy_decompress(&cinfo);
            strncpy(error_message, "Output buffer too small", error_message_size - 1);
            return -1;
        }

        // Copy scanline to output buffer
        memcpy(output_ptr, buffer[0], row_stride_bytes);
        output_ptr += row_stride_bytes;
        bytes_written += row_stride_bytes;
    }

    // Clean up
    jpeg_finish_decompress(&cinfo);
    jpeg_destroy_decompress(&cinfo);

    return 0;
}
*/
import "C"
import (
	"fmt"
)

// JPEGLosslessDecoder implements JPEG Lossless decompression using libjpeg-turbo via CGo.
//
// JPEG Lossless is specified in:
//   - Transfer Syntax 1.2.840.10008.1.2.4.57: JPEG Lossless, Non-Hierarchical, First-Order Prediction
//   - Transfer Syntax 1.2.840.10008.1.2.4.70: JPEG Lossless, Non-Hierarchical (Process 14)
//
// This decoder uses libjpeg-turbo's lossless JPEG decoder, which supports:
//   - 8-bit, 12-bit, and 16-bit pixel data
//   - Grayscale and RGB images
//   - Lossless compression (bit-exact decompression)
//
// CGo Dependencies:
//   - libjpeg-turbo (provides libjpeg API with lossless support)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_8.2.3
type JPEGLosslessDecoder struct {
	transferSyntaxUID string
}

// NewJPEGLosslessDecoder creates a new JPEG Lossless decoder for a specific transfer syntax.
func NewJPEGLosslessDecoder(transferSyntaxUID string) *JPEGLosslessDecoder {
	return &JPEGLosslessDecoder{
		transferSyntaxUID: transferSyntaxUID,
	}
}

// Decode decompresses JPEG Lossless encoded pixel data using libjpeg-turbo.
//
// This function:
//  1. Allocates C memory for input and output buffers
//  2. Calls the C decompression function
//  3. Validates the decompressed image metadata
//  4. Returns the decompressed pixel data
//  5. Cleans up all C memory (even on error)
func (d *JPEGLosslessDecoder) Decode(encapsulated []byte, info *PixelInfo) ([]byte, error) {
	if len(encapsulated) == 0 {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause:             fmt.Errorf("empty JPEG lossless data"),
		}
	}

	// Calculate expected output size
	expectedSize := CalculateExpectedSize(info)

	// Allocate C memory for input data
	inputData := C.CBytes(encapsulated)
	defer C.free(inputData)

	// Allocate C memory for output data
	outputData := C.malloc(C.ulong(expectedSize))
	defer C.free(outputData)

	// Variables to receive image info from C
	var width, height, components, bitsPerSample C.int

	// Error message buffer
	const errorMessageSize = 256
	errorMessage := C.malloc(errorMessageSize)
	defer C.free(errorMessage)

	// Call C decompression function
	result := C.decompress_jpeg_lossless(
		(*C.uchar)(inputData),
		C.ulong(len(encapsulated)),
		(*C.uchar)(outputData),
		C.ulong(expectedSize),
		&width,
		&height,
		&components,
		&bitsPerSample,
		(*C.char)(errorMessage),
		errorMessageSize,
	)

	if result != 0 {
		// Decompression failed
		errMsg := C.GoString((*C.char)(errorMessage))
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause:             fmt.Errorf("libjpeg decompression failed: %s", errMsg),
		}
	}

	// Validate decompressed image dimensions
	if int(width) != int(info.Columns) || int(height) != int(info.Rows) {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause: fmt.Errorf("image dimensions mismatch: got %dx%d, expected %dx%d",
				width, height, info.Columns, info.Rows),
		}
	}

	// Validate components
	if int(components) != int(info.SamplesPerPixel) {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause: fmt.Errorf("components mismatch: got %d, expected %d",
				components, info.SamplesPerPixel),
		}
	}

	// Validate bits per sample
	if int(bitsPerSample) != int(info.BitsStored) {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause: fmt.Errorf("bits per sample mismatch: got %d, expected %d",
				bitsPerSample, info.BitsStored),
		}
	}

	// Copy decompressed data from C memory to Go slice
	goData := C.GoBytes(outputData, C.int(expectedSize))

	return goData, nil
}

// TransferSyntaxUID returns the transfer syntax UID this decoder handles.
func (d *JPEGLosslessDecoder) TransferSyntaxUID() string {
	return d.transferSyntaxUID
}

func init() {
	// Register JPEG Lossless decoders
	// Transfer Syntax 1.2.840.10008.1.2.4.57: JPEG Lossless, Non-Hierarchical, First-Order Prediction
	RegisterDecoder("1.2.840.10008.1.2.4.57", NewJPEGLosslessDecoder("1.2.840.10008.1.2.4.57"))

	// Transfer Syntax 1.2.840.10008.1.2.4.70: JPEG Lossless, Non-Hierarchical (Process 14)
	RegisterDecoder("1.2.840.10008.1.2.4.70", NewJPEGLosslessDecoder("1.2.840.10008.1.2.4.70"))
}
