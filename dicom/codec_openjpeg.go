//go:build cgo && dicom_openjpeg

// Package dicom's JPEG 2000 codec is built only when both cgo is enabled and the
// dicom_openjpeg build tag is set. It binds OpenJPEG (libopenjp2) through the C
// bridge in openjpeg_bridge.{h,c}. The default build (no tag) and CGO_ENABLED=0
// builds register no JPEG 2000 codec, so a JPEG 2000 instance degrades to the
// typed ErrCodecUnavailable (PRD §7.3); building with -tags dicom_openjpeg against
// an installed OpenJPEG makes liver_j2k.dcm decode.
package dicom

// #cgo pkg-config: libopenjp2
// #include <stdlib.h>
// #include "openjpeg_bridge.h"
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// ErrJPEG2000 is the sentinel wrapped by every JPEG 2000 codec failure, so a caller
// can match the codec family with errors.Is without parsing the message. The message
// names the OpenJPEG-reported cause and never carries pixel values (no PHI).
var ErrJPEG2000 = errors.New("dicom: jpeg 2000 codec error")

// jpeg2000Error reports a JPEG 2000 decode or encode failure. It wraps ErrJPEG2000
// and carries the OpenJPEG status code and the captured library message.
type jpeg2000Error struct {
	op     string // "decode" or "encode"
	status int
	detail string
}

func (e *jpeg2000Error) Error() string {
	if e.detail == "" {
		return fmt.Sprintf("dicom: jpeg 2000 %s failed (status %d)", e.op, e.status)
	}
	return fmt.Sprintf("dicom: jpeg 2000 %s failed: %s (status %d)", e.op, e.detail, e.status)
}

func (e *jpeg2000Error) Unwrap() error { return ErrJPEG2000 }

// errBufLen bounds the OpenJPEG message captured for one decode/encode call. The
// captured text is library state, never pixel data.
const errBufLen = 256

// openjpegCodec decodes, and losslessly encodes, the JPEG 2000 transfer syntaxes
// via OpenJPEG. One instance is registered per transfer syntax; encode is offered
// only for the lossless syntax (.90), where a reversible 5-3 wavelet round-trips
// pixel-exact. The lossy syntax (.91) is decode-only.
type openjpegCodec struct {
	ts        TransferSyntax
	canEncode bool
}

func (c openjpegCodec) TransferSyntax() TransferSyntax { return c.ts }

func (c openjpegCodec) CanEncode() bool { return c.canEncode }

// Decode expands one JPEG 2000 codestream into contiguously packed native pixel
// bytes laid out per geom. The decoded image's dimensions and component count are
// validated against geom before any output buffer is sized, so a codestream whose
// SIZ marker disagrees with the dataset header fails with a typed error rather than
// producing a misaligned frame.
func (c openjpegCodec) Decode(frame []byte, geom PixelGeometry) ([]byte, error) {
	if len(frame) == 0 {
		return nil, &jpeg2000Error{op: "decode", detail: "empty codestream"}
	}

	var dec C.goradx_opj_decoded
	errbuf := make([]C.char, errBufLen)

	// Cap the accepted dimension at the dataset's declared Rows/Columns (a frame
	// cannot legitimately be larger than the header says) so an over-large SIZ
	// marker is rejected before the C side allocates the pixel buffer (DCM-014).
	maxDim := geom.Rows
	if geom.Columns > maxDim {
		maxDim = geom.Columns
	}

	status := C.goradx_opj_decode(
		(*C.uint8_t)(unsafe.Pointer(&frame[0])),
		C.size_t(len(frame)),
		C.uint32_t(maxDim),
		&dec,
		&errbuf[0],
		C.size_t(errBufLen),
	)
	defer C.goradx_opj_free_decoded(&dec)

	if status != C.GORADX_OPJ_OK {
		return nil, &jpeg2000Error{op: "decode", status: int(status), detail: goErrString(errbuf)}
	}

	return packDecodedFrame(&dec, geom)
}

// Encode losslessly compresses one contiguous native frame into a JPEG 2000
// codestream. It returns ErrEncodeUnsupported for a decode-only (lossy) syntax.
func (c openjpegCodec) Encode(frame []byte, geom PixelGeometry) ([]byte, error) {
	if !c.canEncode {
		return nil, newEncodeUnsupported(c.ts)
	}
	samples, err := unpackFrameToSamples(frame, geom)
	if err != nil {
		return nil, err
	}

	var enc C.goradx_opj_encoded
	errbuf := make([]C.char, errBufLen)

	var samplePtr *C.int32_t
	if len(samples) > 0 {
		samplePtr = (*C.int32_t)(unsafe.Pointer(&samples[0]))
	}
	sgnd := 0
	if geom.PixelRepresentation == 1 {
		sgnd = 1
	}
	prec := geom.BitsStored
	if prec == 0 {
		prec = geom.BitsAllocated
	}

	status := C.goradx_opj_encode(
		samplePtr,
		C.size_t(len(samples)),
		C.uint32_t(geom.Columns),
		C.uint32_t(geom.Rows),
		C.uint32_t(geom.SamplesPerPixel),
		C.uint32_t(prec),
		C.uint32_t(sgnd),
		&enc,
		&errbuf[0],
		C.size_t(errBufLen),
	)
	defer C.goradx_opj_free_encoded(&enc)

	if status != C.GORADX_OPJ_OK {
		return nil, &jpeg2000Error{op: "encode", status: int(status), detail: goErrString(errbuf)}
	}

	out := C.GoBytes(unsafe.Pointer(enc.data), C.int(enc.len))
	return out, nil
}

// packDecodedFrame turns the C bridge's component-planar int32 samples into the
// DICOM frame's contiguous packed layout. It validates the decoded dimensions and
// component count against geom first, then packs interleaved samples (planar
// configuration 0) at the declared bit depth.
func packDecodedFrame(dec *C.goradx_opj_decoded, geom PixelGeometry) ([]byte, error) {
	width := uint32(dec.width)
	height := uint32(dec.height)
	numcomps := uint32(dec.numcomps)

	if width != uint32(geom.Columns) || height != uint32(geom.Rows) {
		return nil, &jpeg2000Error{op: "decode", detail: fmt.Sprintf(
			"decoded %dx%d does not match dataset %dx%d", width, height, geom.Columns, geom.Rows)}
	}
	if numcomps != uint32(geom.SamplesPerPixel) {
		return nil, &jpeg2000Error{op: "decode", detail: fmt.Sprintf(
			"decoded %d components, dataset declares %d samples per pixel", numcomps, geom.SamplesPerPixel)}
	}

	per := uint64(width) * uint64(height)
	if uint64(dec.data_len) != per*uint64(numcomps) {
		return nil, &jpeg2000Error{op: "decode", detail: "decoded sample count mismatch"}
	}

	// View the C buffer as a Go int32 slice without copying; it lives until the
	// deferred free in Decode.
	samples := unsafe.Slice((*int32)(unsafe.Pointer(dec.data)), int(dec.data_len))

	out := make([]byte, geom.FrameLength())
	if geom.BitsAllocated < 8 {
		packSubByte(out, samples, geom)
		return out, nil
	}
	if err := packWholeByte(out, samples, per, geom); err != nil {
		return nil, err
	}
	return out, nil
}

// packSubByte packs samples whose BitsAllocated is below 8 (a 1-bit segmentation
// frame) LSB-first into the output bytes (PS3.5 §8.1.1). Sub-byte data is single
// component by construction, so samples are in raster order.
func packSubByte(out []byte, samples []int32, geom PixelGeometry) {
	bits := uint(geom.BitsAllocated)
	mask := int32(1)<<bits - 1
	var bitPos uint
	var idx int
	for _, s := range samples {
		v := s & mask
		out[idx] |= byte(v) << bitPos
		bitPos += bits
		if bitPos >= 8 {
			bitPos -= 8
			idx++
		}
	}
}

// packWholeByte packs samples whose BitsAllocated is a whole number of bytes (8 or
// 16) in interleaved sample order (planar configuration 0), little-endian for
// 16-bit. The component-planar C layout is interleaved on the fly.
func packWholeByte(out []byte, samples []int32, per uint64, geom PixelGeometry) error {
	numcomps := int(geom.SamplesPerPixel)
	bytesPer := int(geom.BitsAllocated) / 8
	pixels := int(per)

	switch geom.BitsAllocated {
	case 8:
		for p := 0; p < pixels; p++ {
			for ci := 0; ci < numcomps; ci++ {
				out[p*numcomps+ci] = byte(samples[ci*pixels+p])
			}
		}
	case 16:
		for p := 0; p < pixels; p++ {
			for ci := 0; ci < numcomps; ci++ {
				v := uint16(samples[ci*pixels+p])
				o := (p*numcomps + ci) * bytesPer
				out[o] = byte(v)
				out[o+1] = byte(v >> 8)
			}
		}
	default:
		return &jpeg2000Error{op: "decode", detail: fmt.Sprintf("unsupported BitsAllocated %d", geom.BitsAllocated)}
	}
	return nil
}

// unpackFrameToSamples is the inverse of packDecodedFrame for encode: it reads one
// DICOM-packed native frame into component-planar int32 samples that the C encoder
// consumes. It validates the input length against the geometry first.
func unpackFrameToSamples(frame []byte, geom PixelGeometry) ([]int32, error) {
	want := geom.FrameLength()
	if len(frame) != want {
		return nil, &jpeg2000Error{op: "encode", detail: fmt.Sprintf(
			"frame is %d bytes, geometry needs %d", len(frame), want)}
	}
	per := int(geom.Rows) * int(geom.Columns)
	numcomps := int(geom.SamplesPerPixel)
	samples := make([]int32, per*numcomps)

	if geom.BitsAllocated < 8 {
		bits := uint(geom.BitsAllocated)
		mask := byte(1)<<bits - 1
		var bitPos uint
		var idx int
		for i := 0; i < per; i++ {
			samples[i] = int32((frame[idx] >> bitPos) & mask)
			bitPos += bits
			if bitPos >= 8 {
				bitPos -= 8
				idx++
			}
		}
		return samples, nil
	}

	switch geom.BitsAllocated {
	case 8:
		for p := 0; p < per; p++ {
			for ci := 0; ci < numcomps; ci++ {
				samples[ci*per+p] = int32(frame[p*numcomps+ci])
			}
		}
	case 16:
		for p := 0; p < per; p++ {
			for ci := 0; ci < numcomps; ci++ {
				o := (p*numcomps + ci) * 2
				v := uint16(frame[o]) | uint16(frame[o+1])<<8
				if geom.PixelRepresentation == 1 {
					samples[ci*per+p] = int32(int16(v))
				} else {
					samples[ci*per+p] = int32(v)
				}
			}
		}
	default:
		return nil, &jpeg2000Error{op: "encode", detail: fmt.Sprintf("unsupported BitsAllocated %d", geom.BitsAllocated)}
	}
	return samples, nil
}

// goErrString turns the C error buffer (NUL-terminated) into a Go string.
func goErrString(buf []C.char) string {
	return C.GoString(&buf[0])
}

func init() {
	RegisterCodec(openjpegCodec{ts: JPEG2000Lossless, canEncode: true})
	RegisterCodec(openjpegCodec{ts: JPEG2000, canEncode: false})
}
