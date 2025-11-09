//go:build !cgo
// +build !cgo

package pixel

import (
	"fmt"
)

// JPEGLosslessDecoder is a stub when CGo is disabled.
//
// JPEG Lossless decoding requires libjpeg-turbo via CGo.
// When built without CGo, this decoder will return an error.
type JPEGLosslessDecoder struct {
	transferSyntaxUID string
}

// NewJPEGLosslessDecoder creates a stub JPEG Lossless decoder.
func NewJPEGLosslessDecoder(transferSyntaxUID string) *JPEGLosslessDecoder {
	return &JPEGLosslessDecoder{
		transferSyntaxUID: transferSyntaxUID,
	}
}

// Decode returns an error indicating CGo is required.
func (d *JPEGLosslessDecoder) Decode(encapsulated []byte, info *PixelInfo) ([]byte, error) {
	return nil, &DecompressionError{
		TransferSyntaxUID: d.transferSyntaxUID,
		Cause: fmt.Errorf("JPEG Lossless decompression requires CGo and libjpeg-turbo. " +
			"Rebuild with CGo enabled: CGO_ENABLED=1 go build"),
	}
}

// TransferSyntaxUID returns the transfer syntax UID this decoder handles.
func (d *JPEGLosslessDecoder) TransferSyntaxUID() string {
	return d.transferSyntaxUID
}

func init() {
	// Register JPEG Lossless decoders (will return error when called without CGo)
	// Transfer Syntax 1.2.840.10008.1.2.4.57: JPEG Lossless, Non-Hierarchical, First-Order Prediction
	RegisterDecoder("1.2.840.10008.1.2.4.57", NewJPEGLosslessDecoder("1.2.840.10008.1.2.4.57"))

	// Transfer Syntax 1.2.840.10008.1.2.4.70: JPEG Lossless, Non-Hierarchical (Process 14)
	RegisterDecoder("1.2.840.10008.1.2.4.70", NewJPEGLosslessDecoder("1.2.840.10008.1.2.4.70"))
}
