//go:build !cgo
// +build !cgo

package pixel

import (
	"fmt"
)

// JPEG2000Decoder is a stub when CGo is disabled.
//
// JPEG 2000 decoding requires OpenJPEG via CGo.
// When built without CGo, this decoder will return an error.
type JPEG2000Decoder struct {
	transferSyntaxUID string
	isHTJ2K           bool
}

// NewJPEG2000Decoder creates a stub JPEG 2000 decoder.
func NewJPEG2000Decoder(transferSyntaxUID string, isHTJ2K bool) *JPEG2000Decoder {
	return &JPEG2000Decoder{
		transferSyntaxUID: transferSyntaxUID,
		isHTJ2K:           isHTJ2K,
	}
}

// Decode returns an error indicating CGo is required.
func (d *JPEG2000Decoder) Decode(encapsulated []byte, info *PixelInfo) ([]byte, error) {
	formatName := "JPEG 2000"
	if d.isHTJ2K {
		formatName = "High-Throughput JPEG 2000 (HTJ2K)"
	}

	return nil, &DecompressionError{
		TransferSyntaxUID: d.transferSyntaxUID,
		Cause: fmt.Errorf("%s decompression requires CGo and OpenJPEG 2.5+. "+
			"Rebuild with CGo enabled: CGO_ENABLED=1 go build", formatName),
	}
}

// TransferSyntaxUID returns the transfer syntax UID this decoder handles.
func (d *JPEG2000Decoder) TransferSyntaxUID() string {
	return d.transferSyntaxUID
}

func init() {
	// Register JPEG 2000 decoders (will return error when called without CGo)
	// Transfer Syntax 1.2.840.10008.1.2.4.90: JPEG 2000 Lossless Only
	RegisterDecoder("1.2.840.10008.1.2.4.90", NewJPEG2000Decoder("1.2.840.10008.1.2.4.90", false))

	// Transfer Syntax 1.2.840.10008.1.2.4.91: JPEG 2000 Lossy
	RegisterDecoder("1.2.840.10008.1.2.4.91", NewJPEG2000Decoder("1.2.840.10008.1.2.4.91", false))

	// Transfer Syntax 1.2.840.10008.1.2.4.201: HTJ2K Lossless Only
	RegisterDecoder("1.2.840.10008.1.2.4.201", NewJPEG2000Decoder("1.2.840.10008.1.2.4.201", true))

	// Transfer Syntax 1.2.840.10008.1.2.4.203: HTJ2K Lossless/Lossy
	RegisterDecoder("1.2.840.10008.1.2.4.203", NewJPEG2000Decoder("1.2.840.10008.1.2.4.203", true))
}
