package dicom

import (
	"fmt"
	"math"
	"strconv"
)

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
	geom := src.Geometry
	geom.TransferSyntax = target
	if src.IsEncapsulated() {
		// A codec decode produces the frames below, and every built-in decoder emits
		// sample-interleaved output (planar configuration 0) whose colour model can
		// differ from the declared one (inverse colour transforms, full-resolution
		// chroma). Reconcile the geometry before decoding, failing closed when the
		// decoded colour model cannot be determined.
		pi, err := decodedPhotometricInterpretation(src.Geometry.TransferSyntax, src.Geometry.PhotometricInterpretation)
		if err != nil {
			return nil, err
		}
		geom.PhotometricInterpretation = pi
		geom.PlanarConfiguration = 0
	}

	frames, err := decodeAllFrames(src)
	if err != nil {
		return nil, err
	}
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

// decodedPhotometricInterpretation maps the declared (0028,0004) colour model of an
// encapsulated source to the colour model of its decoded samples. OpenJPEG applies
// the codestream's inverse multiple-component transform during decode, so the JPEG
// 2000 transform terms come out RGB; the JPEG decode path rejects chroma-subsampled
// codestreams and keeps YCbCr at full resolution, so a YBR_FULL_422 source that
// decodes at all yields YBR_FULL samples. A colour term whose decoded layout this
// package cannot determine (a partial-range YCbCr, or a transform term under a
// syntax whose decoder does not perform it) fails closed rather than letting the
// Image Pixel attributes mismatch the decoded bytes.
func decodedPhotometricInterpretation(src TransferSyntax, pi string) (string, error) {
	switch pi {
	case "YBR_ICT", "YBR_RCT":
		if isJPEG2000Family(src) {
			return "RGB", nil
		}
	case "YBR_FULL_422":
		if isJPEGFamily(src) {
			return "YBR_FULL", nil
		}
	case "YBR_PARTIAL_422", "YBR_PARTIAL_420":
		// No defined term names full-resolution partial-range YCbCr; fall through to
		// the fail-closed error.
	default:
		return pi, nil
	}
	return "", &ValueError{
		Tag: TagPhotometricInterpretation, VR: VRCS,
		Msg: fmt.Sprintf("cannot determine the decoded colour model for %s under %s; failing closed rather than writing mismatched Image Pixel attributes", pi, src.Name()),
	}
}

// isJPEG2000Family reports the JPEG 2000 / HTJ2K syntaxes, whose decoder applies the
// codestream's inverse multiple-component transform.
func isJPEG2000Family(ts TransferSyntax) bool {
	switch ts {
	case JPEG2000Lossless, JPEG2000, HTJ2KLossless, HTJ2KLosslessRPCL, HTJ2K:
		return true
	default:
		return false
	}
}

// isJPEGFamily reports the ISO 10918 JPEG syntaxes served by the libjpeg codec.
func isJPEGFamily(ts TransferSyntax) bool {
	switch ts {
	case JPEGBaseline8Bit, JPEGExtended12Bit, JPEGLossless, JPEGLosslessSV1:
		return true
	default:
		return false
	}
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
		if len(encoded)%2 == 1 {
			// Fragment item values must be even (PS3.5 A.4); pad with a trailing NULL
			// so the stream serialises with conformant item lengths.
			encoded = append(encoded, 0x00)
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

// SetPixelData replaces f's (7FE0,0010) element with pd's pixel stream and aligns
// f.Meta.TransferSyntaxUID with pd.Geometry.TransferSyntax, so a Transcode result
// re-enters the dataset: Read -> NewPixelData -> Transcode -> SetPixelData ->
// Write. An encapsulated target is written as the undefined-length fragment stream
// with its Basic Offset Table (PS3.5 A.4); an uncompressed target is written as a
// native OB/OW value (OW when BitsAllocated exceeds 8). The Extended Offset Table
// elements (7FE0,0001)/(7FE0,0002) are removed: they describe the previous stream's
// byte layout, and a stale table would map frames into the wrong bytes.
//
// The Image Pixel attributes are reconciled with pd's geometry so the dataset
// describes the bytes it now carries: PlanarConfiguration (0028,0006) and
// PhotometricInterpretation (0028,0004) follow the geometry Transcode resolved for
// the decoded samples, and NumberOfFrames (0028,0008) follows the actual frame
// count. When the file's transfer syntax at entry — the syntax the pixels being
// replaced were encoded with — is lossy, LossyImageCompression (0028,2110) is set to
// "01" (PS3.3 C.7.6.1.1.5); the ratio and method attributes are left untouched.
func (f *File) SetPixelData(pd *PixelData) error {
	if f == nil || f.Meta == nil || f.DataSet == nil {
		return fmt.Errorf("dicom: SetPixelData requires a File with Meta and DataSet")
	}
	if pd == nil {
		return &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "SetPixelData requires a non-nil PixelData"}
	}

	ts := pd.Geometry.TransferSyntax
	if ts.IsEncapsulated() {
		if pd.encaps == nil {
			return &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "PixelData carries no fragment stream for its encapsulated transfer syntax"}
		}
		stream, err := pd.encaps.encodeStream()
		if err != nil {
			return err
		}
		f.DataSet.Set(Element{Tag: TagPixelData, VR: VROB, Value: &encapsulatedValue{stream: stream}})
	} else {
		if pd.encaps != nil {
			return &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "PixelData carries encapsulated fragments but its transfer syntax is uncompressed"}
		}
		vr := VROB
		if pd.Geometry.BitsAllocated > 8 {
			vr = VROW
		}
		f.DataSet.Set(Element{Tag: TagPixelData, VR: vr, Value: NewBytes(vr, pd.native)})
	}

	f.DataSet.Delete(TagExtendedOffsetTable)
	f.DataSet.Delete(TagExtendedOffsetTableLengths)
	// (7FE0,0003) Encapsulated Pixel Data Value Total Length describes the previous
	// stream's byte length and is meaningless for native pixel data. It is optional
	// (Type 3), so it is deleted rather than recomputed for an encapsulated target:
	// a stale total would misdescribe the new stream.
	f.DataSet.Delete(TagEncapsulatedPixelDataValueTotalLength)

	// Reconcile the Image Pixel attributes with the stream just written (PS3.3
	// C.7.6.3): the geometry carries the layout and colour model pd's frames actually
	// have, and the frame count is what the new stream encodes. PlanarConfiguration
	// is conditional on multi-sample data, and NumberOfFrames is never invented for a
	// single-frame object that did not carry it.
	if pd.Geometry.SamplesPerPixel > 1 {
		f.DataSet.Set(Element{Tag: TagPlanarConfiguration, VR: VRUS, Value: NewInts(VRUS, int64(pd.Geometry.PlanarConfiguration))})
	}
	if pi := pd.Geometry.PhotometricInterpretation; pi != "" {
		f.DataSet.SetString(TagPhotometricInterpretation, pi)
	}
	if _, ok := f.DataSet.Get(TagNumberOfFrames); ok || pd.Geometry.NumberOfFrames > 1 {
		// IS values are carried as lexical Decimals (see DataSet.GetInt).
		nf, err := ParseDecimal(strconv.Itoa(pd.Geometry.NumberOfFrames))
		if err != nil {
			return err
		}
		f.DataSet.Set(Element{Tag: TagNumberOfFrames, VR: VRIS, Value: NewDecimals(VRIS, nf)})
	}
	// Lossy bookkeeping (PS3.3 C.7.6.1.1.5): pixels that were ever lossy-compressed
	// stay lossy through any further transcode, so a lossy source syntax forces
	// (0028,2110) to "01". The ratio and method ((0028,2112)/(0028,2114)) are left
	// as stored: this seam cannot know them and must not invent values.
	if f.Meta.TransferSyntaxUID.IsLossy() {
		f.DataSet.SetString(TagLossyImageCompression, "01")
	}
	f.Meta.TransferSyntaxUID = ts
	return nil
}
