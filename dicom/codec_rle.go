package dicom

// rleCodec implements RLE Lossless (1.2.840.10008.1.2.5) decode and encode in pure
// Go per PS3.5 Annex G. It is always registered regardless of build tag.
type rleCodec struct{}

func (rleCodec) TransferSyntax() TransferSyntax { return RLELossless }

func (rleCodec) CanEncode() bool { return true }

// Decode expands one RLE-encoded frame into contiguously packed pixel bytes.
func (rleCodec) Decode(frame []byte, geom PixelGeometry) ([]byte, error) {
	return decodeRLEFrame(frame, geom)
}

// Encode packs one contiguous frame into an RLE-encoded frame.
func (rleCodec) Encode(frame []byte, geom PixelGeometry) ([]byte, error) {
	return encodeRLEFrame(frame, geom)
}

func init() {
	RegisterCodec(rleCodec{})
}
