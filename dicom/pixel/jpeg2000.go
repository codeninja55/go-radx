//go:build cgo

package pixel

/*
#cgo CFLAGS: -I/usr/include -I/usr/include/openjpeg-2.5
#cgo LDFLAGS: -lopenjp2
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

// ============================================================================
// JPEG 2000 COMPRESSION (ENCODER)
// ============================================================================

// Memory write stream structure for writing to dynamically allocated buffer
typedef struct {
    OPJ_UINT8* data;
    OPJ_SIZE_T size;
    OPJ_SIZE_T capacity;
} memory_write_stream_t;

// Write callback for compression output stream
static OPJ_SIZE_T stream_write(void* p_buffer, OPJ_SIZE_T p_nb_bytes, void* p_user_data) {
    memory_write_stream_t* stream = (memory_write_stream_t*)p_user_data;

    // Ensure capacity
    if (stream->size + p_nb_bytes > stream->capacity) {
        OPJ_SIZE_T new_capacity = stream->capacity * 2;
        if (new_capacity < stream->size + p_nb_bytes) {
            new_capacity = stream->size + p_nb_bytes;
        }

        OPJ_UINT8* new_data = (OPJ_UINT8*)realloc(stream->data, new_capacity);
        if (new_data == NULL) {
            return 0; // Allocation failed
        }

        stream->data = new_data;
        stream->capacity = new_capacity;
    }

    // Write data
    memcpy(stream->data + stream->size, p_buffer, p_nb_bytes);
    stream->size += p_nb_bytes;

    return p_nb_bytes;
}

// Skip callback for compression output stream
static OPJ_OFF_T stream_write_skip(OPJ_OFF_T p_nb_bytes, void* p_user_data) {
    memory_write_stream_t* stream = (memory_write_stream_t*)p_user_data;

    // Ensure capacity for skip (fill with zeros)
    if (stream->size + p_nb_bytes > stream->capacity) {
        OPJ_SIZE_T new_capacity = stream->capacity * 2;
        if (new_capacity < stream->size + p_nb_bytes) {
            new_capacity = stream->size + p_nb_bytes;
        }

        OPJ_UINT8* new_data = (OPJ_UINT8*)realloc(stream->data, new_capacity);
        if (new_data == NULL) {
            return -1;
        }

        stream->data = new_data;
        stream->capacity = new_capacity;
    }

    // Fill skipped bytes with zeros
    memset(stream->data + stream->size, 0, p_nb_bytes);
    stream->size += p_nb_bytes;

    return p_nb_bytes;
}

// Seek callback for compression output stream
static OPJ_BOOL stream_write_seek(OPJ_OFF_T p_nb_bytes, void* p_user_data) {
    memory_write_stream_t* stream = (memory_write_stream_t*)p_user_data;

    if (p_nb_bytes < 0 || (OPJ_SIZE_T)p_nb_bytes > stream->size) {
        return OPJ_FALSE;
    }

    stream->size = (OPJ_SIZE_T)p_nb_bytes;
    return OPJ_TRUE;
}

// Compress pixel data to JPEG 2000 Lossless format
// Returns 0 on success, non-zero on error
// Output data is allocated by this function and must be freed by caller
int compress_jpeg2000_lossless(
    unsigned char* input_data,
    unsigned long input_size,
    int width,
    int height,
    int components,
    int precision,
    unsigned char** output_data,
    unsigned long* output_size,
    char* error_message,
    int error_message_size
) {
    opj_cparameters_t parameters;
    opj_codec_t* codec = NULL;
    opj_image_t* image = NULL;
    opj_stream_t* stream = NULL;
    memory_write_stream_t write_stream;
    int result = -1;

    // Initialize write stream
    write_stream.capacity = input_size; // Start with input size as estimate
    write_stream.size = 0;
    write_stream.data = (OPJ_UINT8*)malloc(write_stream.capacity);
    if (write_stream.data == NULL) {
        snprintf(error_message, error_message_size, "Failed to allocate output buffer");
        return -1;
    }

    // Initialize encoder parameters with defaults
    opj_set_default_encoder_parameters(&parameters);

    // Set lossless compression parameters
    parameters.tcp_numlayers = 1;
    parameters.cp_disto_alloc = 1;
    parameters.tcp_rates[0] = 0; // 0 = lossless
    parameters.irreversible = 0; // 0 = reversible 5-3 wavelet (lossless)

    // Create image color space
    OPJ_COLOR_SPACE color_space = (components == 1) ? OPJ_CLRSPC_GRAY : OPJ_CLRSPC_SRGB;

    // Create image component parameters
    opj_image_cmptparm_t* cmptparms = (opj_image_cmptparm_t*)calloc(components, sizeof(opj_image_cmptparm_t));
    if (cmptparms == NULL) {
        snprintf(error_message, error_message_size, "Failed to allocate component parameters");
        free(write_stream.data);
        return -1;
    }

    for (int i = 0; i < components; i++) {
        cmptparms[i].dx = 1;
        cmptparms[i].dy = 1;
        cmptparms[i].w = width;
        cmptparms[i].h = height;
        cmptparms[i].x0 = 0;
        cmptparms[i].y0 = 0;
        cmptparms[i].prec = precision;
        cmptparms[i].bpp = precision;
        cmptparms[i].sgnd = 0; // Unsigned
    }

    // Create image
    image = opj_image_create(components, cmptparms, color_space);
    free(cmptparms);

    if (image == NULL) {
        snprintf(error_message, error_message_size, "Failed to create OpenJPEG image");
        free(write_stream.data);
        return -1;
    }

    // Set image offset and reference grid
    image->x0 = 0;
    image->y0 = 0;
    image->x1 = width;
    image->y1 = height;

    // Copy pixel data to image components
    int bytes_per_sample = (precision <= 8) ? 1 : 2;
    int pixels_per_frame = width * height;

    for (int comp = 0; comp < components; comp++) {
        unsigned char* src = input_data + (comp * pixels_per_frame * bytes_per_sample);
        OPJ_INT32* dst = image->comps[comp].data;

        if (bytes_per_sample == 1) {
            // 8-bit data
            for (int i = 0; i < pixels_per_frame; i++) {
                dst[i] = (OPJ_INT32)src[i];
            }
        } else {
            // 16-bit data (little-endian)
            for (int i = 0; i < pixels_per_frame; i++) {
                dst[i] = (OPJ_INT32)(src[i * 2] | (src[i * 2 + 1] << 8));
            }
        }
    }

    // Create encoder
    codec = opj_create_compress(OPJ_CODEC_J2K);
    if (codec == NULL) {
        snprintf(error_message, error_message_size, "Failed to create encoder");
        goto cleanup;
    }

    // Setup encoder
    if (!opj_setup_encoder(codec, &parameters, image)) {
        snprintf(error_message, error_message_size, "Failed to setup encoder");
        goto cleanup;
    }

    // Create output stream
    stream = opj_stream_create(OPJ_J2K_STREAM_CHUNK_SIZE, OPJ_FALSE); // OPJ_FALSE = output stream
    if (stream == NULL) {
        snprintf(error_message, error_message_size, "Failed to create output stream");
        goto cleanup;
    }

    // Set stream callbacks
    opj_stream_set_write_function(stream, stream_write);
    opj_stream_set_skip_function(stream, stream_write_skip);
    opj_stream_set_seek_function(stream, stream_write_seek);
    opj_stream_set_user_data(stream, &write_stream, NULL);

    // Encode image
    if (!opj_start_compress(codec, image, stream)) {
        snprintf(error_message, error_message_size, "Failed to start compression");
        goto cleanup;
    }

    if (!opj_encode(codec, stream)) {
        snprintf(error_message, error_message_size, "Failed to encode image");
        goto cleanup;
    }

    if (!opj_end_compress(codec, stream)) {
        snprintf(error_message, error_message_size, "Failed to end compression");
        goto cleanup;
    }

    // Success - transfer ownership of output data to caller
    *output_data = write_stream.data;
    *output_size = write_stream.size;
    result = 0;

cleanup:
    if (result != 0) {
        // On error, free the output buffer
        if (write_stream.data != NULL) {
            free(write_stream.data);
        }
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
	"unsafe"
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

// JPEG2000Encoder implements JPEG 2000 lossless compression using OpenJPEG via CGo.
//
// This encoder produces JPEG 2000 Lossless Only format (1.2.840.10008.1.2.4.90)
// using the reversible 5-3 wavelet transform.
//
// The encoder:
//   - Accepts uncompressed pixel data with image parameters
//   - Uses OpenJPEG 2.5+ for compression
//   - Produces DICOM-compliant JPEG 2000 codestreams
//   - Supports 8-bit, 12-bit, and 16-bit precision
//   - Handles grayscale and RGB images
type JPEG2000Encoder struct{}

// NewJPEG2000Encoder creates a new JPEG 2000 lossless encoder.
func NewJPEG2000Encoder() *JPEG2000Encoder {
	return &JPEG2000Encoder{}
}

// Encode compresses uncompressed pixel data to JPEG 2000 Lossless format.
//
// This function:
//  1. Validates input parameters
//  2. Allocates C memory for input data
//  3. Calls the C compression function
//  4. Returns the compressed JPEG 2000 codestream
//  5. Cleans up all C memory
func (e *JPEG2000Encoder) Encode(pixelData []byte, info *PixelInfo) ([]byte, error) {
	if len(pixelData) == 0 {
		return nil, fmt.Errorf("empty pixel data")
	}

	// Validate pixel info
	if info.Columns == 0 || info.Rows == 0 {
		return nil, fmt.Errorf("invalid image dimensions: %dx%d", info.Columns, info.Rows)
	}

	if info.SamplesPerPixel == 0 {
		return nil, fmt.Errorf("invalid samples per pixel: %d", info.SamplesPerPixel)
	}

	if info.BitsAllocated == 0 {
		return nil, fmt.Errorf("invalid bits allocated: %d", info.BitsAllocated)
	}

	// Calculate expected input size
	expectedSize := CalculateExpectedSize(info)
	if len(pixelData) != int(expectedSize) {
		return nil, fmt.Errorf("pixel data size mismatch: got %d bytes, expected %d bytes",
			len(pixelData), expectedSize)
	}

	// Allocate C memory for input data
	inputData := C.CBytes(pixelData)
	defer C.free(inputData)

	// Variables for output
	var outputData *C.uchar
	var outputSize C.ulong

	// Error message buffer
	const errorMessageSize = 512
	errorMessage := C.malloc(errorMessageSize)
	defer C.free(errorMessage)

	// Call C compression function
	result := C.compress_jpeg2000_lossless(
		(*C.uchar)(inputData),
		C.ulong(len(pixelData)),
		C.int(info.Columns),
		C.int(info.Rows),
		C.int(info.SamplesPerPixel),
		C.int(info.BitsAllocated),
		&outputData,
		&outputSize,
		(*C.char)(errorMessage),
		errorMessageSize,
	)

	if result != 0 {
		// Compression failed
		errMsg := C.GoString((*C.char)(errorMessage))
		return nil, fmt.Errorf("OpenJPEG compression failed: %s", errMsg)
	}

	// Copy compressed data from C memory to Go slice
	// NOTE: The C function allocates outputData, so we must free it
	compressedData := C.GoBytes(unsafe.Pointer(outputData), C.int(outputSize))
	C.free(unsafe.Pointer(outputData))

	return compressedData, nil
}

// EncodeJPEG2000Lossless is a convenience function that encodes pixel data
// to JPEG 2000 Lossless format.
func EncodeJPEG2000Lossless(pixelData []byte, info *PixelInfo) ([]byte, error) {
	encoder := NewJPEG2000Encoder()
	return encoder.Encode(pixelData, info)
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
