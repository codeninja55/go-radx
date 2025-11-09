//go:build cgo
// +build cgo

package pixel

/*
#cgo pkg-config: libopenjp2
#include <stdlib.h>
#include <string.h>
#include <openjpeg.h>

// Memory stream structure for reading from Go byte slice
typedef struct {
    OPJ_UINT8* data;
    OPJ_SIZE_T size;
    OPJ_SIZE_T offset;
} memory_stream_t;

// Stream read callback
static OPJ_SIZE_T stream_read(void* p_buffer, OPJ_SIZE_T p_nb_bytes, void* p_user_data) {
    memory_stream_t* stream = (memory_stream_t*)p_user_data;
    OPJ_SIZE_T bytes_to_read = p_nb_bytes;

    if (stream->offset + bytes_to_read > stream->size) {
        bytes_to_read = stream->size - stream->offset;
    }

    if (bytes_to_read > 0) {
        memcpy(p_buffer, stream->data + stream->offset, bytes_to_read);
        stream->offset += bytes_to_read;
    }

    return bytes_to_read;
}

// Stream skip callback
static OPJ_OFF_T stream_skip(OPJ_OFF_T p_nb_bytes, void* p_user_data) {
    memory_stream_t* stream = (memory_stream_t*)p_user_data;
    OPJ_OFF_T skip_bytes = p_nb_bytes;

    if (stream->offset + skip_bytes > stream->size) {
        skip_bytes = stream->size - stream->offset;
    }

    stream->offset += skip_bytes;
    return skip_bytes;
}

// Stream seek callback
static OPJ_BOOL stream_seek(OPJ_OFF_T p_nb_bytes, void* p_user_data) {
    memory_stream_t* stream = (memory_stream_t*)p_user_data;

    if (p_nb_bytes < 0 || (OPJ_SIZE_T)p_nb_bytes > stream->size) {
        return OPJ_FALSE;
    }

    stream->offset = (OPJ_SIZE_T)p_nb_bytes;
    return OPJ_TRUE;
}

// Quiet error callback (suppress OpenJPEG console output)
static void quiet_error_callback(const char* msg, void* client_data) {
    // Store error message in client data if provided
    if (client_data != NULL) {
        char** error_msg = (char**)client_data;
        if (*error_msg == NULL) {
            *error_msg = strdup(msg);
        }
    }
}

// Quiet warning callback
static void quiet_warning_callback(const char* msg, void* client_data) {
    // Suppress warnings
}

// Quiet info callback
static void quiet_info_callback(const char* msg, void* client_data) {
    // Suppress info messages
}

// Decompress JPEG 2000 data using OpenJPEG
// Returns: 0 on success, -1 on error
int decompress_jpeg2000(
    unsigned char* input_data,
    unsigned long input_size,
    unsigned char* output_data,
    unsigned long output_size,
    int* width,
    int* height,
    int* components,
    int* precision,
    int is_htj2k,
    char* error_message,
    int error_message_size
) {
    opj_dparameters_t parameters;
    opj_codec_t* codec = NULL;
    opj_image_t* image = NULL;
    opj_stream_t* stream = NULL;
    memory_stream_t mem_stream;
    char* opj_error = NULL;
    int result = -1;

    // Initialize decompression parameters
    opj_set_default_decoder_parameters(&parameters);

    // Create codec
    if (is_htj2k) {
        // HTJ2K uses J2K format
        codec = opj_create_decompress(OPJ_CODEC_J2K);
    } else {
        // Standard JPEG 2000 can be J2K or JP2
        // Try J2K first (codestream format - most common in DICOM)
        codec = opj_create_decompress(OPJ_CODEC_J2K);
    }

    if (codec == NULL) {
        strncpy(error_message, "Failed to create OpenJPEG decoder", error_message_size - 1);
        goto cleanup;
    }

    // Set error handlers (quiet mode)
    opj_set_error_handler(codec, quiet_error_callback, &opj_error);
    opj_set_warning_handler(codec, quiet_warning_callback, NULL);
    opj_set_info_handler(codec, quiet_info_callback, NULL);

    // Setup decoder
    if (!opj_setup_decoder(codec, &parameters)) {
        strncpy(error_message, "Failed to setup OpenJPEG decoder", error_message_size - 1);
        goto cleanup;
    }

    // Create memory stream
    mem_stream.data = input_data;
    mem_stream.size = input_size;
    mem_stream.offset = 0;

    stream = opj_stream_create(OPJ_J2K_STREAM_CHUNK_SIZE, OPJ_TRUE);
    if (stream == NULL) {
        strncpy(error_message, "Failed to create OpenJPEG stream", error_message_size - 1);
        goto cleanup;
    }

    // Set stream callbacks
    opj_stream_set_read_function(stream, stream_read);
    opj_stream_set_skip_function(stream, stream_skip);
    opj_stream_set_seek_function(stream, stream_seek);
    opj_stream_set_user_data(stream, &mem_stream, NULL);
    opj_stream_set_user_data_length(stream, input_size);

    // Read header
    if (!opj_read_header(stream, codec, &image)) {
        // Try JP2 format if J2K failed
        opj_destroy_codec(codec);
        opj_stream_destroy(stream);

        codec = opj_create_decompress(OPJ_CODEC_JP2);
        if (codec == NULL) {
            strncpy(error_message, "Failed to create JP2 decoder", error_message_size - 1);
            goto cleanup;
        }

        opj_set_error_handler(codec, quiet_error_callback, &opj_error);
        opj_set_warning_handler(codec, quiet_warning_callback, NULL);
        opj_set_info_handler(codec, quiet_info_callback, NULL);

        if (!opj_setup_decoder(codec, &parameters)) {
            strncpy(error_message, "Failed to setup JP2 decoder", error_message_size - 1);
            goto cleanup;
        }

        mem_stream.offset = 0;
        stream = opj_stream_create(OPJ_J2K_STREAM_CHUNK_SIZE, OPJ_TRUE);
        if (stream == NULL) {
            strncpy(error_message, "Failed to create JP2 stream", error_message_size - 1);
            goto cleanup;
        }

        opj_stream_set_read_function(stream, stream_read);
        opj_stream_set_skip_function(stream, stream_skip);
        opj_stream_set_seek_function(stream, stream_seek);
        opj_stream_set_user_data(stream, &mem_stream, NULL);
        opj_stream_set_user_data_length(stream, input_size);

        if (!opj_read_header(stream, codec, &image)) {
            if (opj_error != NULL) {
                strncpy(error_message, opj_error, error_message_size - 1);
            } else {
                strncpy(error_message, "Failed to read JPEG 2000 header", error_message_size - 1);
            }
            goto cleanup;
        }
    }

    // Decode image
    if (!opj_decode(codec, stream, image)) {
        if (opj_error != NULL) {
            strncpy(error_message, opj_error, error_message_size - 1);
        } else {
            strncpy(error_message, "Failed to decode JPEG 2000 image", error_message_size - 1);
        }
        goto cleanup;
    }

    // Get image info
    *width = image->x1 - image->x0;
    *height = image->y1 - image->y0;
    *components = image->numcomps;
    *precision = image->comps[0].prec;

    // Convert image data to output buffer
    OPJ_UINT32 pixels = (*width) * (*height);
    int bytes_per_sample = (*precision + 7) / 8;
    unsigned long expected_size = pixels * (*components) * bytes_per_sample;

    if (expected_size > output_size) {
        strncpy(error_message, "Output buffer too small", error_message_size - 1);
        goto cleanup;
    }

    // Copy pixel data (interleaved format)
    if (*components == 1) {
        // Grayscale
        for (OPJ_UINT32 i = 0; i < pixels; i++) {
            OPJ_INT32 val = image->comps[0].data[i];
            if (bytes_per_sample == 1) {
                output_data[i] = (unsigned char)val;
            } else {
                // 16-bit (little-endian)
                output_data[i * 2] = (unsigned char)(val & 0xFF);
                output_data[i * 2 + 1] = (unsigned char)((val >> 8) & 0xFF);
            }
        }
    } else {
        // Multi-component (RGB)
        for (OPJ_UINT32 i = 0; i < pixels; i++) {
            for (int c = 0; c < *components && c < 3; c++) {
                OPJ_INT32 val = image->comps[c].data[i];
                if (bytes_per_sample == 1) {
                    output_data[i * (*components) + c] = (unsigned char)val;
                } else {
                    int idx = (i * (*components) + c) * 2;
                    output_data[idx] = (unsigned char)(val & 0xFF);
                    output_data[idx + 1] = (unsigned char)((val >> 8) & 0xFF);
                }
            }
        }
    }

    result = 0;

cleanup:
    if (opj_error != NULL) {
        free(opj_error);
    }
    if (image != NULL) {
        opj_image_destroy(image);
    }
    if (codec != NULL) {
        opj_destroy_codec(codec);
    }
    if (stream != NULL) {
        opj_stream_destroy(stream);
    }

    return result;
}
*/
import "C"
import (
	"fmt"
)

// JPEG2000Decoder implements JPEG 2000 decompression using OpenJPEG via CGo.
//
// JPEG 2000 is specified in:
//   - Transfer Syntax 1.2.840.10008.1.2.4.90: JPEG 2000 Image Compression (Lossless Only)
//   - Transfer Syntax 1.2.840.10008.1.2.4.91: JPEG 2000 Image Compression (Lossy)
//   - Transfer Syntax 1.2.840.10008.1.2.4.201: High-Throughput JPEG 2000 (HTJ2K) Lossless Only
//   - Transfer Syntax 1.2.840.10008.1.2.4.203: High-Throughput JPEG 2000 (HTJ2K) Lossless/Lossy
//
// This decoder uses OpenJPEG 2.5+, which supports:
//   - Standard JPEG 2000 (J2K codestream and JP2 file format)
//   - High-Throughput JPEG 2000 (HTJ2K) - requires OpenJPEG 2.5+
//   - 8-bit, 12-bit, and 16-bit pixel data
//   - Grayscale and multi-component (RGB) images
//   - Lossless and lossy compression
//
// CGo Dependencies:
//   - OpenJPEG 2.5+ (libopenjp2)
//
// DICOM Standard Reference:
// https://dicom.nema.org/medical/dicom/current/output/html/part05.html#sect_8.2.4
type JPEG2000Decoder struct {
	transferSyntaxUID string
	isHTJ2K           bool
}

// NewJPEG2000Decoder creates a new JPEG 2000 decoder for a specific transfer syntax.
func NewJPEG2000Decoder(transferSyntaxUID string, isHTJ2K bool) *JPEG2000Decoder {
	return &JPEG2000Decoder{
		transferSyntaxUID: transferSyntaxUID,
		isHTJ2K:           isHTJ2K,
	}
}

// Decode decompresses JPEG 2000 encoded pixel data using OpenJPEG.
//
// This function:
//  1. Allocates C memory for input and output buffers
//  2. Calls the C decompression function
//  3. Validates the decompressed image metadata
//  4. Returns the decompressed pixel data
//  5. Cleans up all C memory (even on error)
func (d *JPEG2000Decoder) Decode(encapsulated []byte, info *PixelInfo) ([]byte, error) {
	if len(encapsulated) == 0 {
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause:             fmt.Errorf("empty JPEG 2000 data"),
		}
	}

	// Calculate expected output size
	expectedSize := CalculateExpectedSize(info)

	// Allocate C memory for input data
	inputData := C.CBytes(encapsulated)
	defer C.free(inputData)

	// Allocate C memory for output data (generous buffer)
	outputData := C.malloc(C.ulong(expectedSize))
	defer C.free(outputData)

	// Variables to receive image info from C
	var width, height, components, precision C.int
	isHTJ2K := C.int(0)
	if d.isHTJ2K {
		isHTJ2K = C.int(1)
	}

	// Error message buffer
	const errorMessageSize = 512
	errorMessage := C.malloc(errorMessageSize)
	defer C.free(errorMessage)

	// Call C decompression function
	result := C.decompress_jpeg2000(
		(*C.uchar)(inputData),
		C.ulong(len(encapsulated)),
		(*C.uchar)(outputData),
		C.ulong(expectedSize),
		&width,
		&height,
		&components,
		&precision,
		isHTJ2K,
		(*C.char)(errorMessage),
		errorMessageSize,
	)

	if result != 0 {
		// Decompression failed
		errMsg := C.GoString((*C.char)(errorMessage))
		return nil, &DecompressionError{
			TransferSyntaxUID: d.transferSyntaxUID,
			Cause:             fmt.Errorf("OpenJPEG decompression failed: %s", errMsg),
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

	// Copy decompressed data from C memory to Go slice
	goData := C.GoBytes(outputData, C.int(expectedSize))

	return goData, nil
}

// TransferSyntaxUID returns the transfer syntax UID this decoder handles.
func (d *JPEG2000Decoder) TransferSyntaxUID() string {
	return d.transferSyntaxUID
}

func init() {
	// Register JPEG 2000 decoders
	// Transfer Syntax 1.2.840.10008.1.2.4.90: JPEG 2000 Lossless Only
	RegisterDecoder("1.2.840.10008.1.2.4.90", NewJPEG2000Decoder("1.2.840.10008.1.2.4.90", false))

	// Transfer Syntax 1.2.840.10008.1.2.4.91: JPEG 2000 Lossy
	RegisterDecoder("1.2.840.10008.1.2.4.91", NewJPEG2000Decoder("1.2.840.10008.1.2.4.91", false))

	// Transfer Syntax 1.2.840.10008.1.2.4.201: HTJ2K Lossless Only
	RegisterDecoder("1.2.840.10008.1.2.4.201", NewJPEG2000Decoder("1.2.840.10008.1.2.4.201", true))

	// Transfer Syntax 1.2.840.10008.1.2.4.203: HTJ2K Lossless/Lossy
	RegisterDecoder("1.2.840.10008.1.2.4.203", NewJPEG2000Decoder("1.2.840.10008.1.2.4.203", true))
}
