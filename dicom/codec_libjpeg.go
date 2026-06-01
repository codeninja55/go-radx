//go:build cgo && dicom_libjpeg

// Package dicom's JPEG (baseline/extended) codec is built only when both cgo is
// enabled and the dicom_libjpeg build tag is set. It binds libjpeg-turbo's
// TurboJPEG 3.x API through the C bridge in turbojpeg_bridge.{h,c}. The default
// build (no tag) and CGO_ENABLED=0 builds register no JPEG codec, so a JPEG
// Baseline/Extended instance degrades to the typed ErrCodecUnavailable (PRD §7.3);
// building with -tags dicom_libjpeg against an installed libjpeg-turbo makes the
// baseline and extended fixtures decode.
//
// Scope: this codec covers the lossy DCT-based processes DICOM carries as JPEG
// Baseline (Process 1, 8-bit, .50) and JPEG Extended (Process 2 & 4, 12-bit, .51).
// libjpeg-turbo does not implement the lossless-JPEG processes that DICOM .57
// (Process 14) and .70 (Process 14 SV1) use, so those syntaxes register no codec
// here and degrade to ErrCodecUnavailable. Encode is not offered: re-encoding to a
// lossy syntax is never a safe default.
package dicom

// #cgo pkg-config: libturbojpeg
// #include <stdlib.h>
// #include "turbojpeg_bridge.h"
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// ErrJPEG is the sentinel wrapped by every baseline/extended JPEG codec failure, so
// a caller can match the codec family with errors.Is without parsing the message.
// The message names the libjpeg-turbo-reported cause and never carries pixel values
// (no PHI).
var ErrJPEG = errors.New("dicom: jpeg codec error")

// jpegError reports a JPEG decode failure. It wraps ErrJPEG and carries the
// libjpeg-turbo status code and the captured library message.
type jpegError struct {
	op     string // "decode"
	status int
	detail string
}

func (e *jpegError) Error() string {
	if e.detail == "" {
		return fmt.Sprintf("dicom: jpeg %s failed (status %d)", e.op, e.status)
	}
	return fmt.Sprintf("dicom: jpeg %s failed: %s (status %d)", e.op, e.detail, e.status)
}

func (e *jpegError) Unwrap() error { return ErrJPEG }

// jpegErrBufLen bounds the libjpeg-turbo message captured for one decode call. The
// captured text is library state, never pixel data.
const jpegErrBufLen = 256

// libjpegCodec decodes the lossy JPEG transfer syntaxes (Baseline .50, Extended
// .51) via libjpeg-turbo. One instance is registered per transfer syntax; encode is
// never offered (decode-only).
type libjpegCodec struct {
	ts TransferSyntax
}

func (c libjpegCodec) TransferSyntax() TransferSyntax { return c.ts }

func (c libjpegCodec) CanEncode() bool { return false }

// Decode expands one baseline or extended JPEG codestream into contiguously packed
// native pixel bytes laid out per geom. The decoded image's dimensions and component
// count are validated against geom before the output frame is sized, so a codestream
// whose dimensions disagree with the dataset header fails with a typed error rather
// than producing a misaligned frame.
func (c libjpegCodec) Decode(frame []byte, geom PixelGeometry) ([]byte, error) {
	if len(frame) == 0 {
		return nil, &jpegError{op: "decode", detail: "empty codestream"}
	}
	if geom.SamplesPerPixel != 1 && geom.SamplesPerPixel != 3 {
		return nil, &jpegError{op: "decode", detail: fmt.Sprintf(
			"unsupported samples per pixel %d", geom.SamplesPerPixel)}
	}

	var dec C.goradx_tj_decoded
	errbuf := make([]C.char, jpegErrBufLen)

	maxDim := geom.Rows
	if geom.Columns > maxDim {
		maxDim = geom.Columns
	}

	// A YBR photometric interpretation means the decoded samples must stay YCbCr
	// (no RGB conversion), so the frame matches the dataset's declared colour model
	// (PS3.5 §8.2.1). RGB and any non-YBR interpretation take the RGB path.
	var wantYBR C.uint32_t
	if geom.SamplesPerPixel == 3 && isYBRPhotometric(geom.PhotometricInterpretation) {
		wantYBR = 1
	}

	status := C.goradx_tj_decode(
		(*C.uint8_t)(unsafe.Pointer(&frame[0])),
		C.size_t(len(frame)),
		C.uint32_t(maxDim),
		C.uint32_t(geom.SamplesPerPixel),
		wantYBR,
		&dec,
		&errbuf[0],
		C.size_t(jpegErrBufLen),
	)
	defer C.goradx_tj_free_decoded(&dec)

	if status != C.GORADX_TJ_OK {
		return nil, &jpegError{op: "decode", status: int(status), detail: jpegErrString(errbuf)}
	}

	return packJPEGFrame(&dec, geom)
}

// Encode is never supported for the lossy JPEG syntaxes; it returns the typed
// ErrEncodeUnsupported.
func (c libjpegCodec) Encode(_ []byte, _ PixelGeometry) ([]byte, error) {
	return nil, newEncodeUnsupported(c.ts)
}

// packJPEGFrame turns the bridge's interleaved samples into the DICOM frame's packed
// layout. The bridge already produces planar-configuration-0 interleaved samples, so
// 8-bit data is a direct copy after validation; 12-bit data arrives as native-order
// uint16 and is re-emitted little-endian to match the DICOM native byte order.
func packJPEGFrame(dec *C.goradx_tj_decoded, geom PixelGeometry) ([]byte, error) {
	width := uint32(dec.width)
	height := uint32(dec.height)
	numcomps := uint32(dec.numcomps)
	prec := uint32(dec.precision)

	if width != uint32(geom.Columns) || height != uint32(geom.Rows) {
		return nil, &jpegError{op: "decode", detail: fmt.Sprintf(
			"decoded %dx%d does not match dataset %dx%d", width, height, geom.Columns, geom.Rows)}
	}
	if numcomps != uint32(geom.SamplesPerPixel) {
		return nil, &jpegError{op: "decode", detail: fmt.Sprintf(
			"decoded %d components, dataset declares %d samples per pixel", numcomps, geom.SamplesPerPixel)}
	}
	if prec > uint32(geom.BitsStored) && geom.BitsStored != 0 {
		return nil, &jpegError{op: "decode", detail: fmt.Sprintf(
			"decoded precision %d exceeds dataset BitsStored %d", prec, geom.BitsStored)}
	}

	want := geom.FrameLength()
	out := make([]byte, want)

	switch {
	case geom.BitsAllocated == 8 && prec <= 8:
		samples := unsafe.Slice((*byte)(unsafe.Pointer(dec.data)), int(dec.data_len))
		if len(samples) != want {
			return nil, &jpegError{op: "decode", detail: "decoded sample count mismatch"}
		}
		copy(out, samples)
		return out, nil
	case geom.BitsAllocated == 16 && prec > 8:
		// The bridge emitted width*height*numcomps uint16 samples in native order.
		count := int(width) * int(height) * int(numcomps)
		if int(dec.data_len) != count*2 {
			return nil, &jpegError{op: "decode", detail: "decoded sample byte count mismatch"}
		}
		if want != count*2 {
			return nil, &jpegError{op: "decode", detail: "frame length does not match 16-bit sample count"}
		}
		samples := unsafe.Slice((*uint16)(unsafe.Pointer(dec.data)), count)
		for i, v := range samples {
			out[i*2] = byte(v)
			out[i*2+1] = byte(v >> 8)
		}
		return out, nil
	default:
		return nil, &jpegError{op: "decode", detail: fmt.Sprintf(
			"unsupported geometry: BitsAllocated %d with decoded precision %d", geom.BitsAllocated, prec)}
	}
}

// jpegErrString turns the C error buffer (NUL-terminated) into a Go string.
func jpegErrString(buf []C.char) string {
	return C.GoString(&buf[0])
}

// isYBRPhotometric reports whether pi names a YBR (luma/chroma) colour model whose
// JPEG samples must remain YCbCr after decode. The DICOM YBR defined terms all begin
// with "YBR_" (YBR_FULL, YBR_FULL_422, YBR_PARTIAL_422, ...).
func isYBRPhotometric(pi string) bool {
	const prefix = "YBR_"
	return len(pi) >= len(prefix) && pi[:len(prefix)] == prefix
}

func init() {
	RegisterCodec(libjpegCodec{ts: JPEGBaseline8Bit})
	RegisterCodec(libjpegCodec{ts: JPEGExtended12Bit})
}
