package dicom

import "encoding/binary"

// TransferSyntax is the UID-identified encoding of a dataset: byte order,
// implicit-versus-explicit VR, and compression. It is the single transfer-syntax
// type reused by dimse and dicomweb. The reader and writer handle the four
// uncompressed syntaxes and the recognised encapsulated syntaxes below (whose main
// dataset is Explicit VR LE and whose pixel data is a retained fragment stream);
// pixel decode/encode is a separate, codec-gated concern.
type TransferSyntax UID

const (
	ImplicitVRLittleEndian         TransferSyntax = "1.2.840.10008.1.2"
	ExplicitVRLittleEndian         TransferSyntax = "1.2.840.10008.1.2.1"
	DeflatedExplicitVRLittleEndian TransferSyntax = "1.2.840.10008.1.2.1.99"
	ExplicitVRBigEndian            TransferSyntax = "1.2.840.10008.1.2.2" // retired, read+write
	RLELossless                    TransferSyntax = "1.2.840.10008.1.2.5"
	JPEGBaseline8Bit               TransferSyntax = "1.2.840.10008.1.2.4.50"
	JPEGExtended12Bit              TransferSyntax = "1.2.840.10008.1.2.4.51"
	JPEGLossless                   TransferSyntax = "1.2.840.10008.1.2.4.57" // Process 14
	JPEGLosslessSV1                TransferSyntax = "1.2.840.10008.1.2.4.70" // Process 14 SV1, default lossless
	JPEGLSLossless                 TransferSyntax = "1.2.840.10008.1.2.4.80"
	JPEGLSNearLossless             TransferSyntax = "1.2.840.10008.1.2.4.81"
	JPEG2000Lossless               TransferSyntax = "1.2.840.10008.1.2.4.90"
	JPEG2000                       TransferSyntax = "1.2.840.10008.1.2.4.91"
	HTJ2KLossless                  TransferSyntax = "1.2.840.10008.1.2.4.201"
	HTJ2KLosslessRPCL              TransferSyntax = "1.2.840.10008.1.2.4.202"
	HTJ2K                          TransferSyntax = "1.2.840.10008.1.2.4.203"
)

// IsImplicitVR reports whether the syntax encodes elements without an explicit VR.
func (ts TransferSyntax) IsImplicitVR() bool {
	return ts == ImplicitVRLittleEndian
}

// IsBigEndian reports whether multi-byte values are big-endian.
func (ts TransferSyntax) IsBigEndian() bool {
	return ts == ExplicitVRBigEndian
}

// IsDeflated reports whether the main dataset (after the file-meta group) is
// compressed with raw DEFLATE.
func (ts TransferSyntax) IsDeflated() bool {
	return ts == DeflatedExplicitVRLittleEndian
}

// IsEncapsulated reports whether pixel data is carried as compressed fragments
// rather than a single native value. All compressed syntaxes are encapsulated.
func (ts TransferSyntax) IsEncapsulated() bool {
	switch ts {
	case ImplicitVRLittleEndian, ExplicitVRLittleEndian,
		DeflatedExplicitVRLittleEndian, ExplicitVRBigEndian:
		return false
	default:
		return true
	}
}

// Name returns the registered name for a known transfer syntax or the raw UID if
// unregistered.
func (ts TransferSyntax) Name() string {
	return UID(ts).Name()
}

// byteOrder returns the binary.ByteOrder for the syntax. It defaults to little
// endian for any non-big-endian syntax, which covers the v1 uncompressed set.
func (ts TransferSyntax) byteOrder() binary.ByteOrder {
	if ts.IsBigEndian() {
		return binary.BigEndian
	}
	return binary.LittleEndian
}
