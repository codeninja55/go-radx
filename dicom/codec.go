package dicom

import (
	"errors"
	"fmt"
)

// Codec decodes, and where an encoder exists encodes, one transfer syntax's pixel
// frames. A decode-only codec returns ErrEncodeUnsupported from Encode and reports
// CanEncode() == false. The native (uncompressed) and RLE Lossless codecs are pure
// Go and always registered; the JPEG family is registered only behind the optional
// CGo build tag (Increment 6b).
type Codec interface {
	// TransferSyntax is the syntax this codec handles.
	TransferSyntax() TransferSyntax
	// Decode turns one encoded frame into contiguously packed native-order pixel
	// bytes laid out per geom.
	Decode(frame []byte, geom PixelGeometry) ([]byte, error)
	// Encode turns one frame of contiguously packed pixel bytes into this syntax's
	// encoded form. It returns ErrEncodeUnsupported for a decode-only codec.
	Encode(frame []byte, geom PixelGeometry) ([]byte, error)
	// CanEncode reports whether Encode is supported.
	CanEncode() bool
}

// ErrCodecUnavailable is returned when no codec is registered for an encapsulated
// transfer syntax (for example a JPEG 2000 instance in a pure-Go build). The
// returned error wraps this sentinel and names the missing transfer syntax, so a
// caller can both match it with errors.Is and surface which syntax is missing.
var ErrCodecUnavailable = errors.New("dicom: codec unavailable for transfer syntax")

// ErrEncodeUnsupported is returned when a transcode is requested for a transfer
// syntax whose registered codec is decode-only. The returned error wraps this
// sentinel and names the transfer syntax.
var ErrEncodeUnsupported = errors.New("dicom: encode unsupported for transfer syntax")

// CodecUnavailableError reports a missing codec for ts. It wraps ErrCodecUnavailable
// so errors.Is(err, ErrCodecUnavailable) holds while the message names ts (PRD §8.2:
// the failure is unambiguous, never a build break or a silent partial image).
type CodecUnavailableError struct {
	TransferSyntax TransferSyntax
}

func (e *CodecUnavailableError) Error() string {
	return fmt.Sprintf("dicom: codec unavailable for transfer syntax %s (%s)",
		e.TransferSyntax.Name(), string(e.TransferSyntax))
}

func (e *CodecUnavailableError) Unwrap() error { return ErrCodecUnavailable }

// newCodecUnavailable builds the typed missing-codec error for ts.
func newCodecUnavailable(ts TransferSyntax) error { return &CodecUnavailableError{TransferSyntax: ts} }

// EncodeUnsupportedError reports that ts's registered codec is decode-only. It wraps
// ErrEncodeUnsupported and names ts.
type EncodeUnsupportedError struct {
	TransferSyntax TransferSyntax
}

func (e *EncodeUnsupportedError) Error() string {
	return fmt.Sprintf("dicom: encode unsupported for transfer syntax %s (%s)",
		e.TransferSyntax.Name(), string(e.TransferSyntax))
}

func (e *EncodeUnsupportedError) Unwrap() error { return ErrEncodeUnsupported }

// newEncodeUnsupported builds the typed encode-unsupported error for ts.
func newEncodeUnsupported(ts TransferSyntax) error { return &EncodeUnsupportedError{TransferSyntax: ts} }

// codecRegistry maps a transfer syntax to its codec. There is no mutable global the
// pixel pipeline reads under concurrency (PRD §9): the package-level registry is
// populated once, during package init, from the always-available pure-Go codecs and
// any build-tagged CGo codecs. After init it is read-only, so concurrent Frames /
// Transcode calls never race a writer.
type codecRegistry struct {
	byTS map[TransferSyntax]Codec
}

// defaultCodecs is the package registry. It is assembled in init from the pure-Go
// codecs; the CGo build adds JPEG-family codecs in their own init via RegisterCodec.
// It is never mutated after init.
var defaultCodecs = &codecRegistry{byTS: make(map[TransferSyntax]Codec)}

// RegisterCodec makes c available to the pixel pipeline. It is intended to be called
// only from package init (the native and RLE codecs register themselves there, and
// build-tagged CGo codecs register in their own init). Registering after init, while
// other goroutines read the registry, is a data race and is not supported.
func RegisterCodec(c Codec) {
	defaultCodecs.byTS[c.TransferSyntax()] = c
}

// lookupCodec returns the codec registered for ts. The native codecs cover the four
// uncompressed syntaxes; RLE covers RLE Lossless; the JPEG family is present only in
// a CGo build.
func lookupCodec(ts TransferSyntax) (Codec, bool) {
	c, ok := defaultCodecs.byTS[ts]
	return c, ok
}
