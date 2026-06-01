//go:build cgo && dicom_libjpeg

/*
 * turbojpeg_bridge.c implements the JPEG (baseline/extended) bridge declared in
 * turbojpeg_bridge.h. It owns the full TurboJPEG handle lifecycle for one decode
 * call and never leaves a live handle for Go to manage.
 *
 * Safety posture (DCM-014):
 *   - tj3Init and the output malloc are NULL-checked; a NULL is a typed failure,
 *     never a deref.
 *   - The JPEG header's width/height/precision/component-count are read and
 *     validated against the caller's caps BEFORE any pixel buffer is allocated, so
 *     an attacker-controlled header cannot drive an unbounded allocation. A
 *     defense-in-depth TJPARAM_MAXPIXELS cap is also set on the handle.
 *   - The output byte size is computed in 64-bit with explicit overflow checks.
 *   - Cleanup runs on every path via a single goto-error epilogue.
 *   - libjpeg-turbo's error string is captured into a fixed caller buffer; the
 *     captured text is library state, never PHI.
 */
#include "turbojpeg_bridge.h"

#include <turbojpeg.h>
#include <stdlib.h>
#include <string.h>

/* GORADX_TJ_ABS_MAX_DIM is a hard ceiling on any single dimension, independent of
 * the caller's geometry-derived cap. 65535 is the largest value DICOM Rows/Columns
 * (US) can hold, so no conformant image exceeds it. */
#define GORADX_TJ_ABS_MAX_DIM 65535u

/* GORADX_TJ_ABS_MAX_PIXELS bounds the total pixel count regardless of the caller's
 * caps; it is the TJPARAM_MAXPIXELS value handed to libjpeg-turbo so the library
 * itself rejects an oversized image during decode. */
#define GORADX_TJ_ABS_MAX_PIXELS (GORADX_TJ_ABS_MAX_DIM * GORADX_TJ_ABS_MAX_DIM)

static void set_err(char *errbuf, size_t errbuflen, const char *msg) {
  if (errbuf == NULL || errbuflen == 0 || msg == NULL) {
    return;
  }
  if (errbuf[0] != '\0') {
    return; /* keep an already-captured library message */
  }
  size_t n = strlen(msg);
  if (n >= errbuflen) {
    n = errbuflen - 1;
  }
  memcpy(errbuf, msg, n);
  while (n > 0 && (errbuf[n - 1] == '\n' || errbuf[n - 1] == '\r')) {
    n--;
  }
  errbuf[n] = '\0';
}

/* capture_lib_err copies libjpeg-turbo's last error string for handle into errbuf. */
static void capture_lib_err(tjhandle handle, char *errbuf, size_t errbuflen) {
  if (errbuf == NULL || errbuflen == 0) {
    return;
  }
  const char *msg = tj3GetErrorStr(handle);
  if (msg == NULL || msg[0] == '\0') {
    return;
  }
  size_t n = strlen(msg);
  if (n >= errbuflen) {
    n = errbuflen - 1;
  }
  memcpy(errbuf, msg, n);
  while (n > 0 && (errbuf[n - 1] == '\n' || errbuf[n - 1] == '\r')) {
    n--;
  }
  errbuf[n] = '\0';
}

/* byte_size computes width*height*numcomps*bytes_per_sample in 64-bit, failing if
 * any product overflows or exceeds what a size_t can index. */
static int byte_size(uint32_t w, uint32_t h, uint32_t comps, uint32_t bps,
                     uint64_t *out) {
  uint64_t pixels = (uint64_t)w * (uint64_t)h;
  if (w != 0 && pixels / w != (uint64_t)h) {
    return 0;
  }
  uint64_t samples = pixels * (uint64_t)comps;
  if (comps != 0 && samples / comps != pixels) {
    return 0;
  }
  uint64_t total = samples * (uint64_t)bps;
  if (bps != 0 && total / bps != samples) {
    return 0;
  }
  if (total == 0 || total > (uint64_t)SIZE_MAX) {
    return 0;
  }
  *out = total;
  return 1;
}

goradx_tj_status goradx_tj_decode(const uint8_t *src, size_t srclen,
                                  uint32_t max_dim, uint32_t want_comps,
                                  uint32_t want_ybr, goradx_tj_decoded *out,
                                  char *errbuf, size_t errbuflen) {
  if (src == NULL || out == NULL || srclen == 0) {
    set_err(errbuf, errbuflen, "turbojpeg: invalid decode argument");
    return GORADX_TJ_ERR_ARGUMENT;
  }
  memset(out, 0, sizeof(*out));
  if (errbuf != NULL && errbuflen > 0) {
    errbuf[0] = '\0';
  }
  if (max_dim == 0 || max_dim > GORADX_TJ_ABS_MAX_DIM) {
    max_dim = GORADX_TJ_ABS_MAX_DIM;
  }
  if (want_comps != 1 && want_comps != 3) {
    set_err(errbuf, errbuflen, "turbojpeg: unsupported samples per pixel");
    return GORADX_TJ_ERR_COMPONENTS;
  }

  goradx_tj_status status = GORADX_TJ_ERR_DECODE;
  tjhandle handle = NULL;
  uint8_t *buf = NULL;

  handle = tj3Init(TJINIT_DECOMPRESS);
  if (handle == NULL) {
    set_err(errbuf, errbuflen, "turbojpeg: tj3Init returned NULL");
    return GORADX_TJ_ERR_ALLOC;
  }

  /* Bound the library's own allocations: reject before decode if the header would
   * exceed the absolute pixel ceiling. tj3Set on a valid param cannot fail here. */
  tj3Set(handle, TJPARAM_MAXPIXELS, (int)GORADX_TJ_ABS_MAX_PIXELS);
  tj3Set(handle, TJPARAM_STOPONWARNING, 1);

  if (tj3DecompressHeader(handle, src, srclen) != 0) {
    capture_lib_err(handle, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "turbojpeg: tj3DecompressHeader failed");
    status = GORADX_TJ_ERR_DECODE;
    goto done;
  }

  int w = tj3Get(handle, TJPARAM_JPEGWIDTH);
  int h = tj3Get(handle, TJPARAM_JPEGHEIGHT);
  int prec = tj3Get(handle, TJPARAM_PRECISION);
  int lossless = tj3Get(handle, TJPARAM_LOSSLESS);

  if (w <= 0 || h <= 0) {
    set_err(errbuf, errbuflen, "turbojpeg: non-positive JPEG dimensions");
    status = GORADX_TJ_ERR_DIMENSIONS;
    goto done;
  }
  if ((uint32_t)w > max_dim || (uint32_t)h > max_dim) {
    set_err(errbuf, errbuflen, "turbojpeg: JPEG dimensions exceed cap");
    status = GORADX_TJ_ERR_DIMENSIONS;
    goto done;
  }
  /* libjpeg-turbo 3.x decodes both the lossy DCT processes (DICOM Baseline .50 /
   * Extended .51) and the predictive lossless process (DICOM .57 / .70). The valid
   * precision range depends on the process: lossy JPEG is 8-bit (Process 1) or
   * 9..12-bit (Process 2 & 4); lossless JPEG is 2..16-bit. Reject a precision outside
   * the stream's own process range before the precision-keyed decompress dispatch
   * below, so a malformed header cannot select a mismatched decompressor. */
  if (lossless == 1) {
    if (prec < 2 || prec > 16) {
      set_err(errbuf, errbuflen, "turbojpeg: unsupported lossless JPEG precision");
      status = GORADX_TJ_ERR_PRECISION;
      goto done;
    }
  } else if (prec != 8 && !(prec >= 9 && prec <= 12)) {
    set_err(errbuf, errbuflen, "turbojpeg: unsupported JPEG precision");
    status = GORADX_TJ_ERR_PRECISION;
    goto done;
  }

  uint32_t width = (uint32_t)w;
  uint32_t height = (uint32_t)h;
  uint32_t numcomps = want_comps;
  uint32_t bps = (prec <= 8) ? 1u : 2u;

  uint64_t total;
  if (!byte_size(width, height, numcomps, bps, &total)) {
    set_err(errbuf, errbuflen, "turbojpeg: output size overflow");
    status = GORADX_TJ_ERR_OVERFLOW;
    goto done;
  }

  buf = (uint8_t *)malloc((size_t)total);
  if (buf == NULL) {
    set_err(errbuf, errbuflen, "turbojpeg: output buffer allocation failed");
    status = GORADX_TJ_ERR_ALLOC;
    goto done;
  }

  if (numcomps == 3 && want_ybr) {
    /* The dataset declares a YBR colour model, so the decoded samples must remain
     * YCbCr (no RGB conversion). Decode to YUV planes and interleave. Only
     * full-resolution chroma (TJSAMP_444) is representable as YBR_FULL packed
     * samples; a subsampled JPEG would need chroma upsampling, which would change
     * the sample values, so reject it as unsupported. The YBR path is 8-bit only
     * (DICOM YBR JPEGs are baseline 8-bit). */
    if (bps != 1) {
      set_err(errbuf, errbuflen, "turbojpeg: YBR path supports 8-bit only");
      status = GORADX_TJ_ERR_PRECISION;
      goto done;
    }
    int subsamp = tj3Get(handle, TJPARAM_SUBSAMP);
    if (subsamp != TJSAMP_444) {
      set_err(errbuf, errbuflen,
              "turbojpeg: chroma-subsampled YBR JPEG is not supported");
      status = GORADX_TJ_ERR_SUBSAMPLING;
      goto done;
    }
    /* For TJSAMP_444 every plane is width x height with a width-byte stride. Decode
     * into three temporary planes, then interleave Y,Cb,Cr per pixel into buf. */
    size_t plane = (size_t)width * (size_t)height;
    uint8_t *planes_buf = (uint8_t *)malloc(plane * 3);
    if (planes_buf == NULL) {
      set_err(errbuf, errbuflen, "turbojpeg: YUV plane allocation failed");
      status = GORADX_TJ_ERR_ALLOC;
      goto done;
    }
    unsigned char *dstPlanes[3] = {planes_buf, planes_buf + plane,
                                   planes_buf + 2 * plane};
    int strides[3] = {(int)width, (int)width, (int)width};
    int rc = tj3DecompressToYUVPlanes8(handle, src, srclen, dstPlanes, strides);
    if (rc != 0) {
      capture_lib_err(handle, errbuf, errbuflen);
      set_err(errbuf, errbuflen, "turbojpeg: YUV decompress failed");
      free(planes_buf);
      status = GORADX_TJ_ERR_DECODE;
      goto done;
    }
    for (size_t p = 0; p < plane; p++) {
      buf[p * 3 + 0] = dstPlanes[0][p];
      buf[p * 3 + 1] = dstPlanes[1][p];
      buf[p * 3 + 2] = dstPlanes[2][p];
    }
    free(planes_buf);
  } else {
    /* TJPF_RGB asks libjpeg-turbo for interleaved RGB: a YCbCr JPEG is converted to
     * RGB, an RGB JPEG passes through. TJPF_GRAY yields one component. This matches
     * what DICOM expects after decode (RGB or MONOCHROME samples). */
    int pixfmt = (numcomps == 3) ? TJPF_RGB : TJPF_GRAY;
    int pitch = 0; /* 0 = tightly packed rows */
    int rc;
    if (prec <= 8) {
      rc = tj3Decompress8(handle, src, srclen, buf, pitch, pixfmt);
    } else if (prec <= 12) {
      rc = tj3Decompress12(handle, src, srclen, (short *)buf, pitch, pixfmt);
    } else {
      /* 13..16-bit samples (lossless only) need the 16-bit decompressor. buf is
       * sized for 2 bytes per sample (bps == 2), so the unsigned-short write fits. */
      rc = tj3Decompress16(handle, src, srclen, (unsigned short *)buf, pitch, pixfmt);
    }
    if (rc != 0) {
      capture_lib_err(handle, errbuf, errbuflen);
      set_err(errbuf, errbuflen, "turbojpeg: decompress failed");
      status = GORADX_TJ_ERR_DECODE;
      goto done;
    }
  }

  out->width = width;
  out->height = height;
  out->numcomps = numcomps;
  out->precision = (uint32_t)prec;
  out->lossless = (uint32_t)(lossless == 1);
  out->data = buf;
  out->data_len = (size_t)total;
  buf = NULL; /* ownership transferred to out */
  status = GORADX_TJ_OK;

done:
  if (buf != NULL) {
    free(buf);
  }
  if (handle != NULL) {
    tj3Destroy(handle);
  }
  return status;
}

void goradx_tj_free_decoded(goradx_tj_decoded *out) {
  if (out != NULL && out->data != NULL) {
    free(out->data);
    out->data = NULL;
    out->data_len = 0;
  }
}

goradx_tj_status goradx_tj_encode_lossless(const uint8_t *samples, uint32_t width,
                                           uint32_t height, uint32_t numcomps,
                                           uint32_t precision, int psv,
                                           uint8_t **out, size_t *outlen,
                                           char *errbuf, size_t errbuflen) {
  if (samples == NULL || out == NULL || outlen == NULL || width == 0 ||
      height == 0) {
    set_err(errbuf, errbuflen, "turbojpeg: invalid encode argument");
    return GORADX_TJ_ERR_ARGUMENT;
  }
  *out = NULL;
  *outlen = 0;
  if (errbuf != NULL && errbuflen > 0) {
    errbuf[0] = '\0';
  }
  if (numcomps != 1 && numcomps != 3) {
    set_err(errbuf, errbuflen, "turbojpeg: unsupported encode component count");
    return GORADX_TJ_ERR_COMPONENTS;
  }
  if (precision < 2 || precision > 16) {
    set_err(errbuf, errbuflen, "turbojpeg: unsupported encode precision");
    return GORADX_TJ_ERR_PRECISION;
  }

  goradx_tj_status status = GORADX_TJ_ERR_DECODE;
  tjhandle handle = NULL;
  unsigned char *jpegBuf = NULL; /* tj3-allocated; freed with tj3Free */
  uint8_t *copy = NULL;          /* malloc copy returned to the caller */
  size_t jpegSize = 0;
  int pixfmt = (numcomps == 3) ? TJPF_RGB : TJPF_GRAY;
  int rc;

  handle = tj3Init(TJINIT_COMPRESS);
  if (handle == NULL) {
    set_err(errbuf, errbuflen, "turbojpeg: tj3Init(compress) returned NULL");
    return GORADX_TJ_ERR_ALLOC;
  }
  tj3Set(handle, TJPARAM_LOSSLESS, 1);
  tj3Set(handle, TJPARAM_LOSSLESSPSV, psv);
  tj3Set(handle, TJPARAM_PRECISION, (int)precision);
  /* Lossless JPEG never subsamples chroma; pin TJSAMP_444 so a 3-component encode
   * keeps every sample (matching what the decode path requires). */
  tj3Set(handle, TJPARAM_SUBSAMP, TJSAMP_444);

  if (precision <= 8) {
    rc = tj3Compress8(handle, samples, (int)width, 0, (int)height, pixfmt, &jpegBuf,
                      &jpegSize);
  } else if (precision <= 12) {
    rc = tj3Compress12(handle, (const short *)samples, (int)width, 0, (int)height,
                       pixfmt, &jpegBuf, &jpegSize);
  } else {
    rc = tj3Compress16(handle, (const unsigned short *)samples, (int)width, 0,
                       (int)height, pixfmt, &jpegBuf, &jpegSize);
  }
  if (rc != 0) {
    capture_lib_err(handle, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "turbojpeg: lossless compress failed");
    status = GORADX_TJ_ERR_DECODE;
    goto done;
  }

  /* Hand the caller a malloc'd copy so it frees with free(), not tj3Free. */
  copy = (uint8_t *)malloc(jpegSize);
  if (copy == NULL) {
    set_err(errbuf, errbuflen, "turbojpeg: encode output copy allocation failed");
    status = GORADX_TJ_ERR_ALLOC;
    goto done;
  }
  memcpy(copy, jpegBuf, jpegSize);
  *out = copy;
  *outlen = jpegSize;
  status = GORADX_TJ_OK;

done:
  if (jpegBuf != NULL) {
    tj3Free(jpegBuf);
  }
  if (handle != NULL) {
    tj3Destroy(handle);
  }
  return status;
}
