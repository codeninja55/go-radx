//go:build cgo && dicom_libjpeg

/*
 * turbojpeg_bridge.h declares a narrow, safe surface over libjpeg-turbo's
 * TurboJPEG 3.x API for the DICOM JPEG transfer syntaxes it can decode: the lossy
 * JPEG Baseline (Process 1, 8-bit, .50) and JPEG Extended (Process 2 & 4, up to
 * 12-bit, .51), and the predictive lossless JPEG Lossless (Process 14, .57) and JPEG
 * Lossless SV1 (Process 14 Selection Value 1, .70), which libjpeg-turbo 3.x decodes
 * at 2..16-bit precision. It is compiled only under the `cgo && dicom_libjpeg` build
 * tag; the default build never touches libjpeg-turbo.
 *
 * The bridge keeps the TurboJPEG handle lifecycle, the header probe, the
 * precision-dependent decompress dispatch (tj3Decompress8 / tj3Decompress12 /
 * tj3Decompress16), and the error-string capture in C, so the Go side deals only
 * with a fixed-shape result struct and never holds a live tjhandle across the cgo
 * boundary. Every allocation is NULL-checked and every header value that drives an
 * allocation is validated against caller-supplied caps before the allocation happens
 * (DCM-014).
 *
 * goradx_tj_encode_lossless is a test-support entry point that produces lossless
 * codestreams for round-trip correctness tests of the decode path; the production
 * codec is decode-only.
 */
#ifndef GORADX_TURBOJPEG_BRIDGE_H
#define GORADX_TURBOJPEG_BRIDGE_H

#include <stddef.h>
#include <stdint.h>

/* GORADX_TJ_MAX_COMPONENTS caps the component count the bridge produces. DICOM
 * JPEG pixel data is 1 (grayscale) or 3 (colour) samples per pixel; the bridge
 * normalises a decoded JPEG to grayscale or RGB and never emits more than 3. */
#define GORADX_TJ_MAX_COMPONENTS 3

/* goradx_tj_status is the bridge result code. Negative values are bridge-level
 * failures with a human-readable message in the caller's error buffer; 0 is
 * success. The Go side maps every nonzero status to a typed error. */
typedef enum {
  GORADX_TJ_OK = 0,
  GORADX_TJ_ERR_ALLOC = -1,       /* a C allocation returned NULL */
  GORADX_TJ_ERR_DECODE = -2,      /* libjpeg-turbo reported a decode failure */
  GORADX_TJ_ERR_DIMENSIONS = -4,  /* JPEG dimensions exceed the caller cap */
  GORADX_TJ_ERR_COMPONENTS = -5,  /* component count out of range */
  GORADX_TJ_ERR_ARGUMENT = -6,    /* a caller argument was invalid */
  GORADX_TJ_ERR_OVERFLOW = -7,    /* a size computation would overflow */
  GORADX_TJ_ERR_PRECISION = -8,   /* unsupported sample precision */
  GORADX_TJ_ERR_SUBSAMPLING = -9  /* chroma-subsampled YBR JPEG on the YBR path */
} goradx_tj_status;

/* goradx_tj_decoded is the fixed-shape result of a decode. data points to
 * width*height*numcomps samples in sample-interleaved order (planar configuration
 * 0): for precision <= 8 each sample is one byte; for 9..16-bit precision each
 * sample is a native-order uint16. The Go side packs them into the DICOM frame
 * layout. data is allocated by the bridge and released with goradx_tj_free_decoded. */
typedef struct {
  uint32_t width;
  uint32_t height;
  uint32_t numcomps;   /* 1 (grayscale) or 3 (RGB) */
  uint32_t precision;  /* bits per sample as reported by the codestream (2..16) */
  uint32_t lossless;   /* 1 if the codestream uses the predictive lossless process */
  uint8_t *data;       /* width*height*numcomps samples, interleaved */
  size_t data_len;     /* byte length of data */
} goradx_tj_decoded;

/* goradx_tj_decode decodes a baseline or extended JPEG codestream of srclen bytes
 * at src into out. max_dim caps both width and height; a header declaring a larger
 * dimension is rejected before any pixel buffer is allocated. want_comps is the
 * dataset's SamplesPerPixel (1 or 3).
 *
 * want_ybr selects the colour handling for a 3-component image so the decoded
 * samples match the dataset's PhotometricInterpretation (PS3.5 §8.2.1):
 *   - want_ybr == 0: the dataset declares RGB; the bridge requests TJPF_RGB, so a
 *     YCbCr JPEG is converted to RGB and an RGB JPEG passes through.
 *   - want_ybr == 1: the dataset declares a YBR colour model; the bridge decodes to
 *     YUV planes WITHOUT the YCbCr->RGB conversion and interleaves them, preserving
 *     the JPEG's native YCbCr samples. Only full-resolution chroma (TJSAMP_444) is
 *     supported on this path; a chroma-subsampled YBR JPEG is rejected rather than
 *     upsampled (an honest unsupported, not a silently wrong frame).
 * want_ybr is ignored for grayscale (want_comps == 1).
 *
 * errbuf receives a NUL-terminated message on any nonzero return (never PHI: only
 * library state). Returns GORADX_TJ_OK on success. */
goradx_tj_status goradx_tj_decode(const uint8_t *src, size_t srclen,
                                  uint32_t max_dim, uint32_t want_comps,
                                  uint32_t want_ybr, goradx_tj_decoded *out,
                                  char *errbuf, size_t errbuflen);

/* goradx_tj_free_decoded releases the buffer in out (safe on a zeroed struct). */
void goradx_tj_free_decoded(goradx_tj_decoded *out);

/* goradx_tj_encode_lossless encodes width*height*numcomps interleaved samples into a
 * predictive lossless JPEG codestream (the process DICOM uses for .57/.70). It exists
 * to support round-trip correctness tests of the lossless decode path; the production
 * codec is decode-only. Each sample is one byte for precision <= 8, otherwise a
 * native-order uint16. psv is the lossless predictor selection value: 1 yields the
 * SV1 form DICOM .70 uses, 2..7 the general Process 14 (.57) form. On success *out is
 * a malloc'd codestream of *outlen bytes that the caller frees with free(). errbuf
 * receives a NUL-terminated message (library state, never PHI) on any nonzero return. */
goradx_tj_status goradx_tj_encode_lossless(const uint8_t *samples, uint32_t width,
                                           uint32_t height, uint32_t numcomps,
                                           uint32_t precision, int psv,
                                           uint8_t **out, size_t *outlen,
                                           char *errbuf, size_t errbuflen);

#endif /* GORADX_TURBOJPEG_BRIDGE_H */
