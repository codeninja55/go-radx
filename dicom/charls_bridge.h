//go:build cgo && dicom_charls

/*
 * charls_bridge.h declares a narrow, safe surface over the CharLS library's C API
 * for the DICOM JPEG-LS transfer syntaxes: JPEG-LS Lossless (.80) and JPEG-LS
 * Near-Lossless (.81). It is compiled only under the `cgo && dicom_charls` build
 * tag; the default build never touches CharLS.
 *
 * The bridge keeps the CharLS encoder/decoder lifecycle, the header probe, the
 * interleave normalisation to DICOM planar-configuration-0 (sample-interleaved)
 * layout, and the error-message capture in C, so the Go side deals only with a
 * fixed-shape result struct and never holds a live CharLS handle across the cgo
 * boundary. Every allocation is NULL-checked and every frame-info value that drives
 * an allocation is validated against caller-supplied caps before the allocation
 * happens (DCM-014).
 *
 * CharLS is a C++ library exposing a C API; the cgo directives link the C++ runtime.
 */
#ifndef GORADX_CHARLS_BRIDGE_H
#define GORADX_CHARLS_BRIDGE_H

#include <stddef.h>
#include <stdint.h>

/* GORADX_CHARLS_MAX_COMPONENTS caps the component count the bridge accepts before
 * allocating buffers. DICOM JPEG-LS pixel data is 1 (grayscale) or 3 (colour)
 * samples per pixel; a codestream claiming more is rejected. */
#define GORADX_CHARLS_MAX_COMPONENTS 3

/* goradx_charls_status is the bridge result code. Negative values are bridge-level
 * failures with a human-readable message in the caller's error buffer; 0 is
 * success. The Go side maps every nonzero status to a typed error. */
typedef enum {
  GORADX_CHARLS_OK = 0,
  GORADX_CHARLS_ERR_ALLOC = -1,       /* a C allocation returned NULL */
  GORADX_CHARLS_ERR_DECODE = -2,      /* CharLS reported a decode failure */
  GORADX_CHARLS_ERR_ENCODE = -3,      /* CharLS reported an encode failure */
  GORADX_CHARLS_ERR_DIMENSIONS = -4,  /* frame dimensions exceed the caller cap */
  GORADX_CHARLS_ERR_COMPONENTS = -5,  /* component count out of range */
  GORADX_CHARLS_ERR_ARGUMENT = -6,    /* a caller argument was invalid */
  GORADX_CHARLS_ERR_OVERFLOW = -7,    /* a size computation would overflow */
  GORADX_CHARLS_ERR_PRECISION = -8    /* unsupported sample precision */
} goradx_charls_status;

/* goradx_charls_decoded is the fixed-shape result of a decode. data points to
 * width*height*numcomps samples in sample-interleaved order (DICOM planar
 * configuration 0): for <=8-bit precision each sample is one byte; for 9..16-bit
 * precision each sample is a little-endian uint16. data is allocated by the bridge
 * and released with goradx_charls_free_decoded. */
typedef struct {
  uint32_t width;
  uint32_t height;
  uint32_t numcomps;     /* 1 (grayscale) or 3 (colour) */
  uint32_t bits_per_sample; /* as reported by the codestream, range [2, 16] */
  uint32_t near_lossless;   /* NEAR parameter: 0 means lossless */
  uint8_t *data;            /* width*height*numcomps samples, sample-interleaved */
  size_t data_len;          /* byte length of data */
} goradx_charls_decoded;

/* goradx_charls_encoded is the result of an encode: a freshly allocated JPEG-LS
 * codestream buffer. data must be released with goradx_charls_free_encoded. */
typedef struct {
  uint8_t *data;
  size_t len;
} goradx_charls_encoded;

/* goradx_charls_decode decodes a JPEG-LS codestream of srclen bytes at src into out.
 * max_dim caps both width and height; a header declaring a larger dimension is
 * rejected before any pixel buffer is allocated. want_comps and want_bits are the
 * dataset's SamplesPerPixel and BitsStored; the decoded frame-info is validated
 * against them. The decoded samples are normalised to DICOM sample-interleaved
 * layout regardless of the codestream's interleave mode. errbuf receives a
 * NUL-terminated message on any nonzero return (never PHI: only library state). */
goradx_charls_status goradx_charls_decode(const uint8_t *src, size_t srclen,
                                          uint32_t max_dim, uint32_t want_comps,
                                          uint32_t want_bits,
                                          goradx_charls_decoded *out, char *errbuf,
                                          size_t errbuflen);

/* goradx_charls_free_decoded releases the buffer in out (safe on a zeroed struct). */
void goradx_charls_free_decoded(goradx_charls_decoded *out);

/* goradx_charls_encode losslessly encodes width*height*numcomps samples (DICOM
 * sample-interleaved layout) into a JPEG-LS codestream. bits is the bits-per-sample
 * (range [2, 16]). For numcomps > 1, sample-interleave is used. near_lossless is the
 * NEAR parameter (0 for lossless). errbuf receives a message on failure. */
goradx_charls_status goradx_charls_encode(const uint8_t *src, size_t srclen,
                                          uint32_t width, uint32_t height,
                                          uint32_t numcomps, uint32_t bits,
                                          uint32_t near_lossless,
                                          goradx_charls_encoded *out, char *errbuf,
                                          size_t errbuflen);

/* goradx_charls_free_encoded releases the buffer in out (safe on a zeroed struct). */
void goradx_charls_free_encoded(goradx_charls_encoded *out);

#endif /* GORADX_CHARLS_BRIDGE_H */
