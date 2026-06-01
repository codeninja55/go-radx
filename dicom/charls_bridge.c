//go:build cgo && dicom_charls

/*
 * charls_bridge.c implements the JPEG-LS bridge declared in charls_bridge.h. It
 * owns the full CharLS encoder/decoder lifecycle for one call and never leaves a
 * live handle for Go to manage.
 *
 * Safety posture (DCM-014):
 *   - Every charls_*_create and malloc result is NULL-checked; a NULL is a typed
 *     failure, never a deref.
 *   - The codestream's own width/height/precision/component-count are read and
 *     validated against the caller's caps BEFORE any pixel buffer is allocated, so
 *     an attacker-controlled header cannot drive an unbounded allocation.
 *   - The output byte size is computed in 64-bit with explicit overflow checks.
 *   - Cleanup runs on every path via a single goto-error epilogue.
 *   - CharLS's error string is captured into a fixed caller buffer; the captured
 *     text is library state, never PHI.
 */
#include "charls_bridge.h"

#include <charls/charls.h>
#include <stdlib.h>
#include <string.h>

/* GORADX_CHARLS_ABS_MAX_DIM is a hard ceiling on any single dimension, independent
 * of the caller's geometry-derived cap. 65535 is the largest value DICOM
 * Rows/Columns (US) can hold and also the upper bound CharLS frame_info allows. */
#define GORADX_CHARLS_ABS_MAX_DIM 65535u

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

/* capture_errc copies CharLS's message for an error code into errbuf. */
static void capture_errc(charls_jpegls_errc errc, char *errbuf, size_t errbuflen) {
  if (errbuf == NULL || errbuflen == 0) {
    return;
  }
  const char *msg = charls_get_error_message(errc);
  set_err(errbuf, errbuflen, msg);
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

goradx_charls_status goradx_charls_decode(const uint8_t *src, size_t srclen,
                                          uint32_t max_dim, uint32_t want_comps,
                                          uint32_t want_bits,
                                          goradx_charls_decoded *out, char *errbuf,
                                          size_t errbuflen) {
  if (src == NULL || out == NULL || srclen == 0) {
    set_err(errbuf, errbuflen, "charls: invalid decode argument");
    return GORADX_CHARLS_ERR_ARGUMENT;
  }
  memset(out, 0, sizeof(*out));
  if (errbuf != NULL && errbuflen > 0) {
    errbuf[0] = '\0';
  }
  if (max_dim == 0 || max_dim > GORADX_CHARLS_ABS_MAX_DIM) {
    max_dim = GORADX_CHARLS_ABS_MAX_DIM;
  }

  goradx_charls_status status = GORADX_CHARLS_ERR_DECODE;
  charls_jpegls_decoder *dec = NULL;
  uint8_t *buf = NULL;
  uint8_t *planar = NULL;

  dec = charls_jpegls_decoder_create();
  if (dec == NULL) {
    set_err(errbuf, errbuflen, "charls: decoder_create returned NULL");
    return GORADX_CHARLS_ERR_ALLOC;
  }

  charls_jpegls_errc errc;
  errc = charls_jpegls_decoder_set_source_buffer(dec, src, srclen);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: set_source_buffer failed");
    goto done;
  }
  errc = charls_jpegls_decoder_read_header(dec);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: read_header failed");
    goto done;
  }

  struct charls_frame_info fi;
  memset(&fi, 0, sizeof(fi));
  errc = charls_jpegls_decoder_get_frame_info(dec, &fi);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: get_frame_info failed");
    goto done;
  }

  /* Validate the header's claims BEFORE allocating any pixel buffer (DCM-014). */
  if (fi.width == 0 || fi.height == 0) {
    set_err(errbuf, errbuflen, "charls: non-positive frame dimensions");
    status = GORADX_CHARLS_ERR_DIMENSIONS;
    goto done;
  }
  if (fi.width > max_dim || fi.height > max_dim) {
    set_err(errbuf, errbuflen, "charls: frame dimensions exceed cap");
    status = GORADX_CHARLS_ERR_DIMENSIONS;
    goto done;
  }
  if (fi.component_count <= 0 ||
      (uint32_t)fi.component_count > GORADX_CHARLS_MAX_COMPONENTS) {
    set_err(errbuf, errbuflen, "charls: component count out of range");
    status = GORADX_CHARLS_ERR_COMPONENTS;
    goto done;
  }
  if (fi.bits_per_sample < 2 || fi.bits_per_sample > 16) {
    set_err(errbuf, errbuflen, "charls: unsupported bits per sample");
    status = GORADX_CHARLS_ERR_PRECISION;
    goto done;
  }
  if (want_comps != 0 && (uint32_t)fi.component_count != want_comps) {
    set_err(errbuf, errbuflen, "charls: component count does not match dataset");
    status = GORADX_CHARLS_ERR_COMPONENTS;
    goto done;
  }
  if (want_bits != 0 && (uint32_t)fi.bits_per_sample > want_bits) {
    set_err(errbuf, errbuflen, "charls: precision exceeds dataset BitsStored");
    status = GORADX_CHARLS_ERR_PRECISION;
    goto done;
  }

  uint32_t width = fi.width;
  uint32_t height = fi.height;
  uint32_t numcomps = (uint32_t)fi.component_count;
  uint32_t bps = (fi.bits_per_sample <= 8) ? 1u : 2u;

  uint64_t total;
  if (!byte_size(width, height, numcomps, bps, &total)) {
    set_err(errbuf, errbuflen, "charls: output size overflow");
    status = GORADX_CHARLS_ERR_OVERFLOW;
    goto done;
  }

  /* Cross-check CharLS's own destination size against ours for the default
   * (sample-interleaved) stride; they must agree. */
  size_t need = 0;
  errc = charls_jpegls_decoder_get_destination_size(dec, 0, &need);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS || need != (size_t)total) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: destination size mismatch");
    goto done;
  }

  buf = (uint8_t *)malloc((size_t)total);
  if (buf == NULL) {
    set_err(errbuf, errbuflen, "charls: output buffer allocation failed");
    status = GORADX_CHARLS_ERR_ALLOC;
    goto done;
  }

  errc = charls_jpegls_decoder_decode_to_buffer(dec, buf, (size_t)total, 0);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: decode_to_buffer failed");
    goto done;
  }

  /* Normalise multi-component output to DICOM sample-interleaved layout (planar
   * configuration 0). CharLS writes the data in the codestream's interleave mode;
   * a planar (NONE) codestream yields component-planar bytes that must be
   * re-interleaved. SAMPLE and LINE interleave already produce sample-interleaved
   * output. */
  if (numcomps > 1) {
    charls_interleave_mode mode = CHARLS_INTERLEAVE_MODE_NONE;
    errc = charls_jpegls_decoder_get_interleave_mode(dec, &mode);
    if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
      capture_errc(errc, errbuf, errbuflen);
      set_err(errbuf, errbuflen, "charls: get_interleave_mode failed");
      goto done;
    }
    if (mode == CHARLS_INTERLEAVE_MODE_NONE) {
      planar = (uint8_t *)malloc((size_t)total);
      if (planar == NULL) {
        set_err(errbuf, errbuflen, "charls: re-interleave buffer allocation failed");
        status = GORADX_CHARLS_ERR_ALLOC;
        goto done;
      }
      memcpy(planar, buf, (size_t)total);
      size_t per = (size_t)width * (size_t)height;
      if (bps == 1) {
        for (size_t p = 0; p < per; p++) {
          for (uint32_t c = 0; c < numcomps; c++) {
            buf[p * numcomps + c] = planar[(size_t)c * per + p];
          }
        }
      } else {
        for (size_t p = 0; p < per; p++) {
          for (uint32_t c = 0; c < numcomps; c++) {
            size_t srcoff = ((size_t)c * per + p) * 2;
            size_t dstoff = (p * numcomps + c) * 2;
            buf[dstoff] = planar[srcoff];
            buf[dstoff + 1] = planar[srcoff + 1];
          }
        }
      }
      free(planar);
      planar = NULL;
    }
  }

  int32_t near = 0;
  errc = charls_jpegls_decoder_get_near_lossless(dec, 0, &near);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    near = 0; /* near is informational; a failure here does not invalidate pixels */
  }

  out->width = width;
  out->height = height;
  out->numcomps = numcomps;
  out->bits_per_sample = (uint32_t)fi.bits_per_sample;
  out->near_lossless = (near < 0) ? 0u : (uint32_t)near;
  out->data = buf;
  out->data_len = (size_t)total;
  buf = NULL; /* ownership transferred to out */
  status = GORADX_CHARLS_OK;

done:
  if (planar != NULL) {
    free(planar);
  }
  if (buf != NULL) {
    free(buf);
  }
  if (dec != NULL) {
    charls_jpegls_decoder_destroy(dec);
  }
  return status;
}

void goradx_charls_free_decoded(goradx_charls_decoded *out) {
  if (out != NULL && out->data != NULL) {
    free(out->data);
    out->data = NULL;
    out->data_len = 0;
  }
}

goradx_charls_status goradx_charls_encode(const uint8_t *src, size_t srclen,
                                          uint32_t width, uint32_t height,
                                          uint32_t numcomps, uint32_t bits,
                                          uint32_t near_lossless,
                                          goradx_charls_encoded *out, char *errbuf,
                                          size_t errbuflen) {
  if (src == NULL || out == NULL) {
    set_err(errbuf, errbuflen, "charls: invalid encode argument");
    return GORADX_CHARLS_ERR_ARGUMENT;
  }
  memset(out, 0, sizeof(*out));
  if (errbuf != NULL && errbuflen > 0) {
    errbuf[0] = '\0';
  }
  if (width == 0 || height == 0 || width > GORADX_CHARLS_ABS_MAX_DIM ||
      height > GORADX_CHARLS_ABS_MAX_DIM) {
    set_err(errbuf, errbuflen, "charls: invalid encode dimensions");
    return GORADX_CHARLS_ERR_DIMENSIONS;
  }
  if (numcomps == 0 || numcomps > GORADX_CHARLS_MAX_COMPONENTS) {
    set_err(errbuf, errbuflen, "charls: invalid component count");
    return GORADX_CHARLS_ERR_COMPONENTS;
  }
  if (bits < 2 || bits > 16) {
    set_err(errbuf, errbuflen, "charls: unsupported bits per sample");
    return GORADX_CHARLS_ERR_PRECISION;
  }
  uint32_t bps = (bits <= 8) ? 1u : 2u;
  uint64_t total;
  if (!byte_size(width, height, numcomps, bps, &total) || total != srclen) {
    set_err(errbuf, errbuflen, "charls: source size does not match geometry");
    return GORADX_CHARLS_ERR_ARGUMENT;
  }

  goradx_charls_status status = GORADX_CHARLS_ERR_ENCODE;
  charls_jpegls_encoder *enc = NULL;
  uint8_t *dst = NULL;

  enc = charls_jpegls_encoder_create();
  if (enc == NULL) {
    set_err(errbuf, errbuflen, "charls: encoder_create returned NULL");
    return GORADX_CHARLS_ERR_ALLOC;
  }

  struct charls_frame_info fi;
  fi.width = width;
  fi.height = height;
  fi.bits_per_sample = (int32_t)bits;
  fi.component_count = (int32_t)numcomps;

  charls_jpegls_errc errc;
  errc = charls_jpegls_encoder_set_frame_info(enc, &fi);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: set_frame_info failed");
    goto done;
  }
  if (numcomps > 1) {
    errc = charls_jpegls_encoder_set_interleave_mode(enc, CHARLS_INTERLEAVE_MODE_SAMPLE);
    if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
      capture_errc(errc, errbuf, errbuflen);
      set_err(errbuf, errbuflen, "charls: set_interleave_mode failed");
      goto done;
    }
  }
  errc = charls_jpegls_encoder_set_near_lossless(enc, (int32_t)near_lossless);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: set_near_lossless failed");
    goto done;
  }

  size_t estimated = 0;
  errc = charls_jpegls_encoder_get_estimated_destination_size(enc, &estimated);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS || estimated == 0) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: get_estimated_destination_size failed");
    goto done;
  }

  dst = (uint8_t *)malloc(estimated);
  if (dst == NULL) {
    set_err(errbuf, errbuflen, "charls: destination buffer allocation failed");
    status = GORADX_CHARLS_ERR_ALLOC;
    goto done;
  }
  errc = charls_jpegls_encoder_set_destination_buffer(enc, dst, estimated);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: set_destination_buffer failed");
    goto done;
  }

  errc = charls_jpegls_encoder_encode_from_buffer(enc, src, srclen, 0);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: encode_from_buffer failed");
    goto done;
  }

  size_t written = 0;
  errc = charls_jpegls_encoder_get_bytes_written(enc, &written);
  if (errc != CHARLS_JPEGLS_ERRC_SUCCESS || written == 0 || written > estimated) {
    capture_errc(errc, errbuf, errbuflen);
    set_err(errbuf, errbuflen, "charls: get_bytes_written failed");
    goto done;
  }

  out->data = dst;
  out->len = written;
  dst = NULL; /* ownership transferred to out */
  status = GORADX_CHARLS_OK;

done:
  if (dst != NULL) {
    free(dst);
  }
  if (enc != NULL) {
    charls_jpegls_encoder_destroy(enc);
  }
  return status;
}

void goradx_charls_free_encoded(goradx_charls_encoded *out) {
  if (out != NULL && out->data != NULL) {
    free(out->data);
    out->data = NULL;
    out->len = 0;
  }
}
