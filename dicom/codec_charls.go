//go:build cgo && dicom_charls

// Package dicom's JPEG-LS codec is built only when both cgo is enabled and the
// dicom_charls build tag is set. It binds CharLS through the C bridge in
// charls_bridge.{h,c}. The default build (no tag) and CGO_ENABLED=0 builds register
// no JPEG-LS codec, so a JPEG-LS instance degrades to the typed ErrCodecUnavailable
// (PRD §7.3); building with -tags dicom_charls against an installed CharLS makes the
// JPEG-LS fixtures decode.
//
// Scope: JPEG-LS Lossless (.80, decode + lossless encode) and JPEG-LS Near-Lossless
// (.81, decode-only). Encode is offered for the lossless syntax only, where the
// round-trip is pixel-exact; re-encoding to a lossy near-lossless syntax is never a
// safe default.
//
// CharLS is a C++ library; the cgo LDFLAGS link the C++ runtime in addition to the
// pkg-config flags. The runtime is platform-specific: libc++ on macOS (clang) and
// libstdc++ on Linux (gcc), so the link flag is selected per GOOS.
package dicom

// #cgo pkg-config: charls
// #cgo darwin LDFLAGS: -lc++
// #cgo linux LDFLAGS: -lstdc++
// #include <stdlib.h>
// #include "charls_bridge.h"
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// ErrJPEGLS is the sentinel wrapped by every JPEG-LS codec failure, so a caller can
// match the codec family with errors.Is without parsing the message. The message
// names the CharLS-reported cause and never carries pixel values (no PHI).
var ErrJPEGLS = errors.New("dicom: jpeg-ls codec error")

// jpeglsError reports a JPEG-LS decode or encode failure. It wraps ErrJPEGLS and
// carries the CharLS bridge status code and the captured library message.
type jpeglsError struct {
	op     string // "decode" or "encode"
	status int
	detail string
}

func (e *jpeglsError) Error() string {
	if e.detail == "" {
		return fmt.Sprintf("dicom: jpeg-ls %s failed (status %d)", e.op, e.status)
	}
	return fmt.Sprintf("dicom: jpeg-ls %s failed: %s (status %d)", e.op, e.detail, e.status)
}

func (e *jpeglsError) Unwrap() error { return ErrJPEGLS }

// jpeglsErrBufLen bounds the CharLS message captured for one call. The captured text
// is library state, never pixel data.
const jpeglsErrBufLen = 256

// charlsCodec decodes, and (for the lossless syntax) losslessly encodes, the JPEG-LS
// transfer syntaxes via CharLS. One instance is registered per transfer syntax;
// encode is offered only for JPEG-LS Lossless (.80).
type charlsCodec struct {
	ts        TransferSyntax
	canEncode bool
}

func (c charlsCodec) TransferSyntax() TransferSyntax { return c.ts }

func (c charlsCodec) CanEncode() bool { return c.canEncode }

// Decode expands one JPEG-LS codestream into contiguously packed native pixel bytes
// laid out per geom (sample-interleaved, planar configuration 0). The decoded
// frame-info is validated against geom before the output frame is finalised, so a
// codestream whose dimensions or component count disagree with the dataset header
// fails with a typed error rather than producing a misaligned frame.
func (c charlsCodec) Decode(frame []byte, geom PixelGeometry) ([]byte, error) {
	if len(frame) == 0 {
		return nil, &jpeglsError{op: "decode", detail: "empty codestream"}
	}
	if geom.SamplesPerPixel != 1 && geom.SamplesPerPixel != 3 {
		return nil, &jpeglsError{op: "decode", detail: fmt.Sprintf(
			"unsupported samples per pixel %d", geom.SamplesPerPixel)}
	}

	var dec C.goradx_charls_decoded
	errbuf := make([]C.char, jpeglsErrBufLen)

	maxDim := geom.Rows
	if geom.Columns > maxDim {
		maxDim = geom.Columns
	}
	wantBits := geom.BitsStored
	if wantBits == 0 {
		wantBits = geom.BitsAllocated
	}

	status := C.goradx_charls_decode(
		(*C.uint8_t)(unsafe.Pointer(&frame[0])),
		C.size_t(len(frame)),
		C.uint32_t(maxDim),
		C.uint32_t(geom.SamplesPerPixel),
		C.uint32_t(wantBits),
		&dec,
		&errbuf[0],
		C.size_t(jpeglsErrBufLen),
	)
	defer C.goradx_charls_free_decoded(&dec)

	if status != C.GORADX_CHARLS_OK {
		return nil, &jpeglsError{op: "decode", status: int(status), detail: jpeglsErrString(errbuf)}
	}

	return packJPEGLSFrame(&dec, geom)
}

// Encode losslessly compresses one contiguous native frame into a JPEG-LS codestream.
// It returns the typed ErrEncodeUnsupported for the near-lossless (decode-only)
// syntax.
func (c charlsCodec) Encode(frame []byte, geom PixelGeometry) ([]byte, error) {
	if !c.canEncode {
		return nil, newEncodeUnsupported(c.ts)
	}
	want := geom.FrameLength()
	if len(frame) != want {
		return nil, &jpeglsError{op: "encode", detail: fmt.Sprintf(
			"frame is %d bytes, geometry needs %d", len(frame), want)}
	}
	if geom.SamplesPerPixel != 1 && geom.SamplesPerPixel != 3 {
		return nil, &jpeglsError{op: "encode", detail: fmt.Sprintf(
			"unsupported samples per pixel %d", geom.SamplesPerPixel)}
	}
	bits := geom.BitsStored
	if bits == 0 {
		bits = geom.BitsAllocated
	}

	var enc C.goradx_charls_encoded
	errbuf := make([]C.char, jpeglsErrBufLen)

	var srcPtr *C.uint8_t
	if len(frame) > 0 {
		srcPtr = (*C.uint8_t)(unsafe.Pointer(&frame[0]))
	}
	status := C.goradx_charls_encode(
		srcPtr,
		C.size_t(len(frame)),
		C.uint32_t(geom.Columns),
		C.uint32_t(geom.Rows),
		C.uint32_t(geom.SamplesPerPixel),
		C.uint32_t(bits),
		0, // near_lossless 0 = lossless
		&enc,
		&errbuf[0],
		C.size_t(jpeglsErrBufLen),
	)
	defer C.goradx_charls_free_encoded(&enc)

	if status != C.GORADX_CHARLS_OK {
		return nil, &jpeglsError{op: "encode", status: int(status), detail: jpeglsErrString(errbuf)}
	}

	out := C.GoBytes(unsafe.Pointer(enc.data), C.int(enc.len))
	return out, nil
}

// packJPEGLSFrame validates the bridge's decoded frame-info against geom and copies
// the already-sample-interleaved samples into the DICOM frame buffer. The bridge
// emits <=8-bit samples as bytes and 9..16-bit samples as little-endian uint16,
// which is the DICOM native byte order, so the copy is direct.
func packJPEGLSFrame(dec *C.goradx_charls_decoded, geom PixelGeometry) ([]byte, error) {
	width := uint32(dec.width)
	height := uint32(dec.height)
	numcomps := uint32(dec.numcomps)
	bps := uint32(dec.bits_per_sample)

	if width != uint32(geom.Columns) || height != uint32(geom.Rows) {
		return nil, &jpeglsError{op: "decode", detail: fmt.Sprintf(
			"decoded %dx%d does not match dataset %dx%d", width, height, geom.Columns, geom.Rows)}
	}
	if numcomps != uint32(geom.SamplesPerPixel) {
		return nil, &jpeglsError{op: "decode", detail: fmt.Sprintf(
			"decoded %d components, dataset declares %d samples per pixel", numcomps, geom.SamplesPerPixel)}
	}
	wantBytesPer := uint32(1)
	if geom.BitsAllocated > 8 {
		wantBytesPer = 2
	}
	gotBytesPer := uint32(1)
	if bps > 8 {
		gotBytesPer = 2
	}
	if gotBytesPer != wantBytesPer {
		return nil, &jpeglsError{op: "decode", detail: fmt.Sprintf(
			"decoded %d-bit samples but dataset BitsAllocated is %d", bps, geom.BitsAllocated)}
	}

	want := geom.FrameLength()
	if int(dec.data_len) != want {
		return nil, &jpeglsError{op: "decode", detail: fmt.Sprintf(
			"decoded %d bytes, frame length is %d", int(dec.data_len), want)}
	}

	out := make([]byte, want)
	samples := unsafe.Slice((*byte)(unsafe.Pointer(dec.data)), int(dec.data_len))
	copy(out, samples)
	return out, nil
}

// jpeglsErrString turns the C error buffer (NUL-terminated) into a Go string.
func jpeglsErrString(buf []C.char) string {
	return C.GoString(&buf[0])
}

func init() {
	RegisterCodec(charlsCodec{ts: JPEGLSLossless, canEncode: true})
	RegisterCodec(charlsCodec{ts: JPEGLSNearLossless, canEncode: false})
}
