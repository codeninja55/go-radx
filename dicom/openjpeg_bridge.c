//go:build cgo && dicom_openjpeg

/*
 * openjpeg_bridge.c implements the JPEG 2000 bridge declared in
 * openjpeg_bridge.h. It owns the full OpenJPEG lifecycle (codec, stream, image)
 * for one decode or encode call and never leaves a live handle for Go to manage.
 *
 * Safety posture (DCM-014):
 *   - Every opj_* and malloc result is NULL-checked; a NULL is a typed failure,
 *     never a deref.
 *   - The codestream's own width/height/component-count are validated against the
 *     caller's caps BEFORE any pixel buffer is allocated, so an attacker-controlled
 *     SIZ marker cannot drive an unbounded allocation.
 *   - Sample-count and byte-size products are computed in 64-bit with explicit
 *     overflow checks, so no size silently truncates.
 *   - Cleanup runs on every path via a single goto-error epilogue.
 *   - The OpenJPEG error/warning handlers capture into a fixed caller buffer
 *     instead of writing to stderr; the captured text is library state, never PHI.
 */
#include "openjpeg_bridge.h"

#include <openjpeg.h>
#include <stdlib.h>
#include <string.h>

/* GORADX_OPJ_ABS_MAX_DIM is a hard ceiling on any single dimension, independent of
 * the caller's geometry-derived cap. It bounds the worst case even if a caller
 * passes an over-large max_dim. 65535 is the largest value DICOM Rows/Columns (US)
 * can hold, so no conformant image exceeds it. */
#define GORADX_OPJ_ABS_MAX_DIM 65535u

/* errctx carries a single decode/encode call's captured message. The OpenJPEG
 * handlers write the first message they see; later messages are ignored so the
 * earliest (usually root-cause) text survives. */
typedef struct {
  char *buf;
  size_t cap;
  int written;
} errctx;

static void capture_msg(const char *msg, void *client) {
  errctx *e = (errctx *)client;
  if (e == NULL || e->buf == NULL || e->cap == 0 || e->written) {
    return;
  }
  if (msg == NULL) {
    msg = "openjpeg: unspecified error";
  }
  /* Copy at most cap-1 bytes and always NUL-terminate; strip a trailing newline
   * so the Go-side error is a single line. */
  size_t n = strlen(msg);
  if (n >= e->cap) {
    n = e->cap - 1;
  }
  memcpy(e->buf, msg, n);
  while (n > 0 && (e->buf[n - 1] == '\n' || e->buf[n - 1] == '\r')) {
    n--;
  }
  e->buf[n] = '\0';
  e->written = 1;
}

static void set_err(char *errbuf, size_t errbuflen, const char *msg) {
  if (errbuf == NULL || errbuflen == 0) {
    return;
  }
  if (errbuf[0] != '\0') {
    return; /* keep a handler-captured message if one is already present */
  }
  size_t n = strlen(msg);
  if (n >= errbuflen) {
    n = errbuflen - 1;
  }
  memcpy(errbuf, msg, n);
  errbuf[n] = '\0';
}

/* membuf is the user data for the memory read-stream backing a decode. The bytes
 * are owned by the Go caller for the duration of the call; the bridge only reads. */
typedef struct {
  const uint8_t *p;
  size_t len;
  size_t pos;
} membuf;

static OPJ_SIZE_T membuf_read(void *buffer, OPJ_SIZE_T nb, void *user) {
  membuf *m = (membuf *)user;
  size_t rem = m->len - m->pos;
  if (rem == 0) {
    return (OPJ_SIZE_T)-1; /* OpenJPEG's end-of-stream sentinel */
  }
  if (nb > rem) {
    nb = rem;
  }
  memcpy(buffer, m->p + m->pos, nb);
  m->pos += nb;
  return nb;
}

static OPJ_OFF_T membuf_skip(OPJ_OFF_T nb, void *user) {
  membuf *m = (membuf *)user;
  if (nb < 0) {
    return -1;
  }
  size_t rem = m->len - m->pos;
  if ((size_t)nb > rem) {
    nb = (OPJ_OFF_T)rem;
  }
  m->pos += (size_t)nb;
  return nb;
}

static OPJ_BOOL membuf_seek(OPJ_OFF_T nb, void *user) {
  membuf *m = (membuf *)user;
  if (nb < 0 || (size_t)nb > m->len) {
    return OPJ_FALSE;
  }
  m->pos = (size_t)nb;
  return OPJ_TRUE;
}

/* growbuf is the user data for the memory write-stream backing an encode. It grows
 * a heap buffer as OpenJPEG writes the codestream. A failed realloc aborts the
 * write by returning the error sentinel. */
typedef struct {
  uint8_t *p;
  size_t len;
  size_t cap;
  int failed;
} growbuf;

static OPJ_SIZE_T growbuf_write(void *buffer, OPJ_SIZE_T nb, void *user) {
  growbuf *g = (growbuf *)user;
  if (g->failed) {
    return (OPJ_SIZE_T)-1;
  }
  size_t need = g->len + nb;
  if (need < g->len) { /* size_t overflow */
    g->failed = 1;
    return (OPJ_SIZE_T)-1;
  }
  if (need > g->cap) {
    size_t newcap = g->cap ? g->cap : 4096;
    while (newcap < need) {
      size_t doubled = newcap * 2;
      if (doubled < newcap) { /* overflow: clamp to exact need */
        newcap = need;
        break;
      }
      newcap = doubled;
    }
    uint8_t *np = (uint8_t *)realloc(g->p, newcap);
    if (np == NULL) {
      g->failed = 1;
      return (OPJ_SIZE_T)-1;
    }
    g->p = np;
    g->cap = newcap;
  }
  memcpy(g->p + g->len, buffer, nb);
  g->len = need;
  return nb;
}

static OPJ_OFF_T growbuf_skip(OPJ_OFF_T nb, void *user) {
  /* OpenJPEG's J2K writer does not require seekable output for a single-tile
   * codestream, but it does call skip; advancing the logical length with zero
   * fill keeps the position consistent. */
  growbuf *g = (growbuf *)user;
  if (nb < 0 || g->failed) {
    return -1;
  }
  size_t need = g->len + (size_t)nb;
  if (need < g->len) {
    g->failed = 1;
    return -1;
  }
  if (need > g->cap) {
    uint8_t *np = (uint8_t *)realloc(g->p, need);
    if (np == NULL) {
      g->failed = 1;
      return -1;
    }
    g->p = np;
    g->cap = need;
  }
  memset(g->p + g->len, 0, (size_t)nb);
  g->len = need;
  return nb;
}

static OPJ_BOOL growbuf_seek(OPJ_OFF_T nb, void *user) {
  growbuf *g = (growbuf *)user;
  if (nb < 0 || (size_t)nb > g->len) {
    return OPJ_FALSE;
  }
  g->len = (size_t)nb;
  return OPJ_TRUE;
}

/* mul_overflow multiplies three uint32 values into a uint64 sample count, failing
 * if the product exceeds what a size_t can index. */
static int sample_count(uint32_t w, uint32_t h, uint32_t comps, uint64_t *out) {
  uint64_t pixels = (uint64_t)w * (uint64_t)h;
  uint64_t total = pixels * (uint64_t)comps;
  if (w != 0 && pixels / w != (uint64_t)h) {
    return 0;
  }
  if (comps != 0 && total / comps != pixels) {
    return 0;
  }
  if (total > (uint64_t)SIZE_MAX / sizeof(int32_t)) {
    return 0;
  }
  *out = total;
  return 1;
}

goradx_opj_status goradx_opj_decode(const uint8_t *src, size_t srclen,
                                    uint32_t max_dim, goradx_opj_decoded *out,
                                    char *errbuf, size_t errbuflen) {
  if (src == NULL || out == NULL || srclen == 0) {
    set_err(errbuf, errbuflen, "openjpeg: invalid decode argument");
    return GORADX_OPJ_ERR_ARGUMENT;
  }
  memset(out, 0, sizeof(*out));
  if (errbuf != NULL && errbuflen > 0) {
    errbuf[0] = '\0';
  }
  if (max_dim == 0 || max_dim > GORADX_OPJ_ABS_MAX_DIM) {
    max_dim = GORADX_OPJ_ABS_MAX_DIM;
  }

  errctx ectx = {errbuf, errbuflen, 0};
  goradx_opj_status status = GORADX_OPJ_ERR_DECODE;

  opj_stream_t *stream = NULL;
  opj_codec_t *codec = NULL;
  opj_image_t *image = NULL;
  membuf mb = {src, srclen, 0};

  stream = opj_stream_create(srclen < (1u << 20) ? srclen : (1u << 20), OPJ_TRUE);
  if (stream == NULL) {
    set_err(errbuf, errbuflen, "openjpeg: opj_stream_create returned NULL");
    status = GORADX_OPJ_ERR_ALLOC;
    goto done;
  }
  opj_stream_set_read_function(stream, membuf_read);
  opj_stream_set_skip_function(stream, membuf_skip);
  opj_stream_set_seek_function(stream, membuf_seek);
  opj_stream_set_user_data(stream, &mb, NULL);
  opj_stream_set_user_data_length(stream, srclen);

  codec = opj_create_decompress(OPJ_CODEC_J2K);
  if (codec == NULL) {
    set_err(errbuf, errbuflen, "openjpeg: opj_create_decompress returned NULL");
    status = GORADX_OPJ_ERR_ALLOC;
    goto done;
  }
  opj_set_error_handler(codec, capture_msg, &ectx);
  opj_set_warning_handler(codec, capture_msg, &ectx);

  opj_dparameters_t dparams;
  opj_set_default_decoder_parameters(&dparams);
  if (!opj_setup_decoder(codec, &dparams)) {
    set_err(errbuf, errbuflen, "openjpeg: opj_setup_decoder failed");
    goto done;
  }

  if (!opj_read_header(stream, codec, &image) || image == NULL) {
    set_err(errbuf, errbuflen, "openjpeg: opj_read_header failed");
    goto done;
  }

  /* Validate the header's claims BEFORE allocating any pixel buffer (DCM-014). */
  if (image->x1 <= image->x0 || image->y1 <= image->y0) {
    set_err(errbuf, errbuflen, "openjpeg: non-positive image dimensions");
    status = GORADX_OPJ_ERR_DIMENSIONS;
    goto done;
  }
  uint32_t width = image->x1 - image->x0;
  uint32_t height = image->y1 - image->y0;
  if (width > max_dim || height > max_dim) {
    set_err(errbuf, errbuflen, "openjpeg: image dimensions exceed cap");
    status = GORADX_OPJ_ERR_DIMENSIONS;
    goto done;
  }
  if (image->numcomps == 0 || image->numcomps > GORADX_OPJ_MAX_COMPONENTS) {
    set_err(errbuf, errbuflen, "openjpeg: component count out of range");
    status = GORADX_OPJ_ERR_COMPONENTS;
    goto done;
  }

  if (!opj_decode(codec, stream, image) || !opj_end_decompress(codec, stream)) {
    set_err(errbuf, errbuflen, "openjpeg: opj_decode failed");
    goto done;
  }

  /* Every component must match the image grid; a codestream with mismatched
   * component sizes is rejected rather than read past a short buffer. */
  for (uint32_t i = 0; i < image->numcomps; i++) {
    opj_image_comp_t *c = &image->comps[i];
    if (c->data == NULL || c->w != width || c->h != height) {
      set_err(errbuf, errbuflen, "openjpeg: component grid mismatch");
      status = GORADX_OPJ_ERR_DECODE;
      goto done;
    }
  }

  uint64_t nsamples;
  if (!sample_count(width, height, image->numcomps, &nsamples)) {
    set_err(errbuf, errbuflen, "openjpeg: sample count overflow");
    status = GORADX_OPJ_ERR_OVERFLOW;
    goto done;
  }

  int32_t *buf = (int32_t *)malloc((size_t)nsamples * sizeof(int32_t));
  if (buf == NULL) {
    set_err(errbuf, errbuflen, "openjpeg: output buffer allocation failed");
    status = GORADX_OPJ_ERR_ALLOC;
    goto done;
  }

  size_t per = (size_t)width * (size_t)height;
  for (uint32_t i = 0; i < image->numcomps; i++) {
    memcpy(buf + (size_t)i * per, image->comps[i].data, per * sizeof(int32_t));
    out->prec[i] = image->comps[i].prec;
    out->sgnd[i] = image->comps[i].sgnd;
  }

  out->width = width;
  out->height = height;
  out->numcomps = image->numcomps;
  out->data = buf;
  out->data_len = (size_t)nsamples;
  status = GORADX_OPJ_OK;

done:
  if (image != NULL) {
    opj_image_destroy(image);
  }
  if (codec != NULL) {
    opj_destroy_codec(codec);
  }
  if (stream != NULL) {
    opj_stream_destroy(stream);
  }
  return status;
}

void goradx_opj_free_decoded(goradx_opj_decoded *out) {
  if (out != NULL && out->data != NULL) {
    free(out->data);
    out->data = NULL;
    out->data_len = 0;
  }
}

goradx_opj_status goradx_opj_encode(const int32_t *samples, size_t nsamples,
                                    uint32_t width, uint32_t height,
                                    uint32_t numcomps, uint32_t prec,
                                    uint32_t sgnd, goradx_opj_encoded *out,
                                    char *errbuf, size_t errbuflen) {
  if (samples == NULL || out == NULL) {
    set_err(errbuf, errbuflen, "openjpeg: invalid encode argument");
    return GORADX_OPJ_ERR_ARGUMENT;
  }
  memset(out, 0, sizeof(*out));
  if (errbuf != NULL && errbuflen > 0) {
    errbuf[0] = '\0';
  }
  if (width == 0 || height == 0 || numcomps == 0 ||
      numcomps > GORADX_OPJ_MAX_COMPONENTS) {
    set_err(errbuf, errbuflen, "openjpeg: invalid encode geometry");
    return GORADX_OPJ_ERR_ARGUMENT;
  }
  if (prec == 0 || prec > 32) {
    set_err(errbuf, errbuflen, "openjpeg: precision out of range");
    return GORADX_OPJ_ERR_ARGUMENT;
  }
  uint64_t expect;
  if (!sample_count(width, height, numcomps, &expect) || expect != nsamples) {
    set_err(errbuf, errbuflen, "openjpeg: sample count does not match geometry");
    return GORADX_OPJ_ERR_ARGUMENT;
  }

  errctx ectx = {errbuf, errbuflen, 0};
  goradx_opj_status status = GORADX_OPJ_ERR_ENCODE;

  opj_image_cmptparm_t cmptparm[GORADX_OPJ_MAX_COMPONENTS];
  memset(cmptparm, 0, sizeof(cmptparm));
  for (uint32_t i = 0; i < numcomps; i++) {
    cmptparm[i].dx = 1;
    cmptparm[i].dy = 1;
    cmptparm[i].w = width;
    cmptparm[i].h = height;
    cmptparm[i].x0 = 0;
    cmptparm[i].y0 = 0;
    cmptparm[i].prec = prec;
    cmptparm[i].sgnd = sgnd ? 1 : 0;
  }

  opj_codec_t *codec = NULL;
  opj_image_t *image = NULL;
  opj_stream_t *stream = NULL;
  growbuf gb = {NULL, 0, 0, 0};

  OPJ_COLOR_SPACE clrspc = (numcomps >= 3) ? OPJ_CLRSPC_SRGB : OPJ_CLRSPC_GRAY;
  image = opj_image_create(numcomps, cmptparm, clrspc);
  if (image == NULL) {
    set_err(errbuf, errbuflen, "openjpeg: opj_image_create returned NULL");
    status = GORADX_OPJ_ERR_ALLOC;
    goto done;
  }
  image->x0 = 0;
  image->y0 = 0;
  image->x1 = width;
  image->y1 = height;

  size_t per = (size_t)width * (size_t)height;
  for (uint32_t i = 0; i < numcomps; i++) {
    if (image->comps[i].data == NULL) {
      set_err(errbuf, errbuflen, "openjpeg: image component data is NULL");
      status = GORADX_OPJ_ERR_ALLOC;
      goto done;
    }
    memcpy(image->comps[i].data, samples + (size_t)i * per,
           per * sizeof(int32_t));
  }

  opj_cparameters_t cparams;
  opj_set_default_encoder_parameters(&cparams);
  cparams.tcp_numlayers = 1;
  cparams.cp_disto_alloc = 1;
  cparams.tcp_rates[0] = 0; /* rate 0 selects lossless allocation */
  cparams.irreversible = 0; /* reversible 5-3 wavelet: lossless */
  cparams.cod_format = 0;   /* raw J2K codestream, matching DICOM */

  /* The default 6 resolution levels require each dimension to be at least
   * 2^(levels-1); a smaller frame (e.g. a tiny ROI) would make opj_setup_encoder
   * reject the parameters. Clamp the level count so 2^(levels-1) <= min(w,h),
   * which is what opj_compress does for small images. */
  uint32_t min_dim = width < height ? width : height;
  int levels = 1;
  while (levels < cparams.numresolution && (1u << (levels - 1)) < min_dim) {
    levels++;
  }
  cparams.numresolution = levels;

  codec = opj_create_compress(OPJ_CODEC_J2K);
  if (codec == NULL) {
    set_err(errbuf, errbuflen, "openjpeg: opj_create_compress returned NULL");
    status = GORADX_OPJ_ERR_ALLOC;
    goto done;
  }
  opj_set_error_handler(codec, capture_msg, &ectx);
  opj_set_warning_handler(codec, capture_msg, &ectx);

  if (!opj_setup_encoder(codec, &cparams, image)) {
    set_err(errbuf, errbuflen, "openjpeg: opj_setup_encoder failed");
    goto done;
  }

  stream = opj_stream_create(1u << 16, OPJ_FALSE);
  if (stream == NULL) {
    set_err(errbuf, errbuflen, "openjpeg: opj_stream_create returned NULL");
    status = GORADX_OPJ_ERR_ALLOC;
    goto done;
  }
  opj_stream_set_write_function(stream, growbuf_write);
  opj_stream_set_skip_function(stream, growbuf_skip);
  opj_stream_set_seek_function(stream, growbuf_seek);
  opj_stream_set_user_data(stream, &gb, NULL);

  if (!opj_start_compress(codec, image, stream) ||
      !opj_encode(codec, stream) || !opj_end_compress(codec, stream)) {
    set_err(errbuf, errbuflen, "openjpeg: opj_encode failed");
    goto done;
  }
  if (gb.failed || gb.p == NULL || gb.len == 0) {
    set_err(errbuf, errbuflen, "openjpeg: encode output allocation failed");
    status = GORADX_OPJ_ERR_ALLOC;
    goto done;
  }

  out->data = gb.p;
  out->len = gb.len;
  gb.p = NULL; /* ownership transferred to out */
  status = GORADX_OPJ_OK;

done:
  if (gb.p != NULL) {
    free(gb.p);
  }
  if (stream != NULL) {
    opj_stream_destroy(stream);
  }
  if (image != NULL) {
    opj_image_destroy(image);
  }
  if (codec != NULL) {
    opj_destroy_codec(codec);
  }
  return status;
}

void goradx_opj_free_encoded(goradx_opj_encoded *out) {
  if (out != NULL && out->data != NULL) {
    free(out->data);
    out->data = NULL;
    out->len = 0;
  }
}
