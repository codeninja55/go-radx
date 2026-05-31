package dicom

// nativeCodec handles the uncompressed transfer syntaxes. Native pixel data is a
// single contiguous OB/OW value already laid out as packed samples; the codec is an
// identity transform on a single frame's bytes (the per-frame slicing and any
// byte-order handling happen in the PixelData frame iterator, which owns the whole
// buffer). It exists so the pipeline treats native and encapsulated data uniformly.
type nativeCodec struct {
	ts TransferSyntax
}

func (c nativeCodec) TransferSyntax() TransferSyntax { return c.ts }

// Decode returns the frame bytes unchanged: native data is already decoded.
func (c nativeCodec) Decode(frame []byte, _ PixelGeometry) ([]byte, error) {
	out := make([]byte, len(frame))
	copy(out, frame)
	return out, nil
}

// Encode returns the frame bytes unchanged: encoding to a native syntax is a copy.
func (c nativeCodec) Encode(frame []byte, _ PixelGeometry) ([]byte, error) {
	out := make([]byte, len(frame))
	copy(out, frame)
	return out, nil
}

func (c nativeCodec) CanEncode() bool { return true }

func init() {
	for _, ts := range []TransferSyntax{
		ImplicitVRLittleEndian,
		ExplicitVRLittleEndian,
		ExplicitVRBigEndian,
		DeflatedExplicitVRLittleEndian,
	} {
		RegisterCodec(nativeCodec{ts: ts})
	}
}
