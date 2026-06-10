package dicom

import "math"

// Transcode re-encodes src's pixel frames into target. It is an explicit, opt-in
// operation: re-compressing clinical pixels is a data-integrity hazard, so it is never
// performed automatically by the reader or writer (PRD §7.3). Transcoding requires a
// decode codec for src's transfer syntax and an encode codec for target; either being
// unavailable in the current build is a typed error (ErrCodecUnavailable for a missing
// decoder, ErrEncodeUnsupported for a missing or decode-only encoder), never a silent
// or corrupt result. In a pure-Go build the encodable targets are the uncompressed
// syntaxes and RLE Lossless; JPEG-family targets return ErrEncodeUnsupported until the
// CGo codecs are built in (Increment 6b).
func Transcode(src *PixelData, target TransferSyntax) (*PixelData, error) {
	frames, err := decodeAllFrames(src)
	if err != nil {
		return nil, err
	}

	geom := src.Geometry
	geom.TransferSyntax = target
	geom.NumberOfFrames = len(frames)

	if !target.IsEncapsulated() {
		return transcodeToNative(geom, frames), nil
	}

	codec, ok := lookupCodec(target)
	if !ok || !codec.CanEncode() {
		return nil, newEncodeUnsupported(target)
	}
	return transcodeToEncapsulated(geom, frames, codec)
}

// decodeAllFrames decodes every frame of src into contiguous native pixel bytes,
// surfacing a missing-decoder failure as ErrCodecUnavailable.
func decodeAllFrames(src *PixelData) ([][]byte, error) {
	var frames [][]byte
	for frame, err := range src.Frames() {
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame.Pixels)
	}
	return frames, nil
}

// transcodeToNative concatenates the decoded frames into one contiguous native buffer.
// The decoded frame length is recorded so the resulting PixelData slices into the same
// frames even when the source geometry's BitsAllocated disagrees (a non-conformant
// RLE source).
func transcodeToNative(geom PixelGeometry, frames [][]byte) *PixelData {
	frameLen := 0
	if len(frames) > 0 {
		frameLen = len(frames[0])
	}
	var buf []byte
	for _, f := range frames {
		buf = append(buf, f...)
	}
	pd := newNativePixelData(geom, buf)
	pd.frameLen = frameLen
	return pd
}

// transcodeToEncapsulated encodes each decoded frame through codec and assembles a
// fresh encapsulated stream with a Basic Offset Table (one fragment per frame). The
// resulting PixelData decodes back through the same codec.
func transcodeToEncapsulated(geom PixelGeometry, frames [][]byte, codec Codec) (*PixelData, error) {
	enc := &encapsulated{}
	offsets := make([]uint32, len(frames))
	var pos uint32
	for i, f := range frames {
		encoded, err := codec.Encode(f, geom)
		if err != nil {
			return nil, err
		}
		offsets[i] = pos
		enc.fragments = append(enc.fragments, fragment{offset: pos, data: encoded})
		// The next fragment's item header begins after this fragment's 8-byte item
		// header and even-length value. Basic Offset Table entries are 32-bit
		// (PS3.5 A.4), so a stream growing past that cannot be represented.
		if uint64(pos)+uint64(8+len(encoded)) > math.MaxUint32 {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "encapsulated stream exceeds the 32-bit offset table"}
		}
		pos += uint32(8 + len(encoded)) // #nosec G115 -- bounded by the MaxUint32 check above
	}
	enc.bot = offsets
	return &PixelData{Geometry: geom, encaps: enc}, nil
}
