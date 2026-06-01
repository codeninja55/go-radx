//go:build cgo && dicom_libjpeg

package dicom

// #cgo pkg-config: libturbojpeg
// #include <stdlib.h>
// #include "turbojpeg_bridge.h"
import "C"

import "unsafe"

// encodeLosslessJPEG encodes sample-interleaved samples into a predictive lossless
// JPEG codestream via libjpeg-turbo. It backs the lossless round-trip correctness
// tests of the decode path; the registered codec is decode-only, so this helper sits
// on no production decode or transcode path. samples are one byte per sample for
// precision <= 8, otherwise a native-order uint16. psv is the lossless predictor
// selection value: 1 is the SV1 form DICOM .70 uses, 2..7 the general Process 14 form
// DICOM .57 uses. The returned codestream is a Go-owned copy.
func encodeLosslessJPEG(samples []byte, width, height, numcomps, precision, psv int) ([]byte, error) {
	if len(samples) == 0 {
		return nil, &jpegError{op: "encode", detail: "empty samples"}
	}
	var out *C.uint8_t
	var outlen C.size_t
	errbuf := make([]C.char, jpegErrBufLen)
	status := C.goradx_tj_encode_lossless(
		(*C.uint8_t)(unsafe.Pointer(&samples[0])),
		C.uint32_t(width), C.uint32_t(height), C.uint32_t(numcomps),
		C.uint32_t(precision), C.int(psv),
		&out, &outlen, &errbuf[0], C.size_t(jpegErrBufLen),
	)
	if status != C.GORADX_TJ_OK {
		return nil, &jpegError{op: "encode", status: int(status), detail: jpegErrString(errbuf)}
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoBytes(unsafe.Pointer(out), C.int(outlen)), nil
}
