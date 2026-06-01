//go:build cgo && dicom_openjpeg

/*
 * openjpeg_bridge.h declares a narrow, safe surface over OpenJPEG (libopenjp2)
 * for the JPEG 2000 DICOM transfer syntaxes. It is compiled only under the
 * `cgo && dicom_openjpeg` build tag; the default build never touches OpenJPEG.
 *
 * The bridge exists to keep the OpenJPEG codec/stream/image lifecycle, the
 * memory-stream callbacks, and the error/warning capture in C, so the Go side
 * deals only with a fixed-shape result struct and never holds a live opj_* handle
 * across the cgo boundary. Every allocation is NULL-checked and every codestream
 * value that drives an allocation is validated against caller-supplied caps before
 * the allocation happens (DCM-014).
 */
#ifndef GORADX_OPENJPEG_BRIDGE_H
#define GORADX_OPENJPEG_BRIDGE_H

#include <stddef.h>
#include <stdint.h>

/* GORADX_OPJ_MAX_COMPONENTS caps the component count the bridge will accept from
 * a codestream header before allocating per-component buffers. DICOM pixel data
 * is at most a handful of samples per pixel (1 mono, 3 RGB/YBR); a codestream
 * claiming more is rejected rather than driving a large allocation. */
#define GORADX_OPJ_MAX_COMPONENTS 8

/* goradx_opj_status is the bridge result code. Negative values are bridge-level
 * failures with a human-readable message in the caller's error buffer; 0 is
 * success. The Go side maps every nonzero status to a typed error. */
typedef enum {
  GORADX_OPJ_OK = 0,
  GORADX_OPJ_ERR_ALLOC = -1,        /* a C allocation returned NULL */
  GORADX_OPJ_ERR_DECODE = -2,       /* OpenJPEG reported a decode failure */
  GORADX_OPJ_ERR_ENCODE = -3,       /* OpenJPEG reported an encode failure */
  GORADX_OPJ_ERR_DIMENSIONS = -4,   /* codestream dimensions exceed the caller cap */
  GORADX_OPJ_ERR_COMPONENTS = -5,   /* component count out of range */
  GORADX_OPJ_ERR_ARGUMENT = -6,     /* a caller argument was invalid */
  GORADX_OPJ_ERR_OVERFLOW = -7      /* a size computation would overflow */
} goradx_opj_status;

/* goradx_opj_decoded is the fixed-shape result of a decode. data points to
 * width*height*numcomps int32 samples in component-planar order (all of
 * component 0, then component 1, ...); the Go side packs them into the DICOM
 * frame layout. data is allocated by the bridge and must be released with
 * goradx_opj_free_decoded. */
typedef struct {
  uint32_t width;
  uint32_t height;
  uint32_t numcomps;
  uint32_t prec[GORADX_OPJ_MAX_COMPONENTS]; /* bits per sample, per component */
  uint32_t sgnd[GORADX_OPJ_MAX_COMPONENTS]; /* 1 if signed, per component */
  int32_t *data;                            /* width*height*numcomps samples */
  size_t data_len;                          /* element count of data */
} goradx_opj_decoded;

/* goradx_opj_encoded is the result of an encode: a freshly allocated J2K
 * codestream buffer. data must be released with goradx_opj_free_encoded. */
typedef struct {
  uint8_t *data;
  size_t len;
} goradx_opj_encoded;

/* goradx_opj_decode decodes a raw J2K codestream (OPJ_CODEC_J2K) of srclen bytes
 * at src into out. max_dim caps both width and height; a header declaring a larger
 * dimension is rejected before any pixel buffer is allocated. errbuf receives a
 * NUL-terminated message on any nonzero return (never PHI: only library state).
 * Returns GORADX_OPJ_OK on success. */
goradx_opj_status goradx_opj_decode(const uint8_t *src, size_t srclen,
                                    uint32_t max_dim, goradx_opj_decoded *out,
                                    char *errbuf, size_t errbuflen);

/* goradx_opj_free_decoded releases the buffer in out (safe on a zeroed struct). */
void goradx_opj_free_decoded(goradx_opj_decoded *out);

/* goradx_opj_encode losslessly encodes width*height*numcomps int32 samples
 * (component-planar order, as goradx_opj_decode produces) into a raw J2K
 * codestream. prec is the bits-per-sample to record (1..32); sgnd marks signed
 * samples. The reversible 5-3 wavelet with a single quality layer at rate 0 is
 * used so the round-trip is lossless. errbuf receives a message on failure. */
goradx_opj_status goradx_opj_encode(const int32_t *samples, size_t nsamples,
                                    uint32_t width, uint32_t height,
                                    uint32_t numcomps, uint32_t prec,
                                    uint32_t sgnd, goradx_opj_encoded *out,
                                    char *errbuf, size_t errbuflen);

/* goradx_opj_free_encoded releases the buffer in out (safe on a zeroed struct). */
void goradx_opj_free_encoded(goradx_opj_encoded *out);

#endif /* GORADX_OPENJPEG_BRIDGE_H */
