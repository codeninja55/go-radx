package dicom

import (
	"iter"
)

// Frame is one decoded frame of pixel data.
type Frame struct {
	Index  int
	Pixels []byte // decoded, one frame's worth; native byte order resolved
}

// PixelData is the parsed (7FE0,0010) element together with the geometry needed to
// decode it. It is either native (a single contiguous OB/OW buffer under an
// uncompressed transfer syntax) or encapsulated (fragmented frames under a
// compressed transfer syntax). Frames yields decoded frames either way; native data
// is sliced from the contiguous buffer, encapsulated data is decoded fragment-by-
// fragment through the codec registered for the transfer syntax.
type PixelData struct {
	Geometry PixelGeometry

	native   []byte           // contiguous samples for an uncompressed transfer syntax
	encaps   *encapsulated    // parsed fragment stream for a compressed transfer syntax
	extended *extendedOffsets // Extended Offset Table from the dataset, if present

	// frameLen overrides the geometry-derived frame length for native slicing when it
	// is positive. It is set when frames are produced by a decode (transcode to native),
	// where the decoded frame length is authoritative even if the dataset's BitsAllocated
	// disagrees, so a non-conformant source still slices into the correct frames.
	frameLen int
}

// NewPixelData builds the PixelData for ds under ts. For an uncompressed transfer
// syntax it reads the native (7FE0,0010) value from ds. For an encapsulated transfer
// syntax it reads the fragment stream the reader retained on the dataset; a dataset
// whose pixel element does not carry retained fragments (one built by hand) supplies
// the stream through NewEncapsulatedPixelData instead.
func NewPixelData(ds *DataSet, ts TransferSyntax) (*PixelData, error) {
	e, ok := ds.Get(TagPixelData)
	if !ok {
		return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "dataset has no Pixel Data element"}
	}

	val := e.Value
	if dv, isDeferred := val.(*DeferredValue); isDeferred {
		loaded, err := dv.Load()
		if err != nil {
			return nil, err
		}
		val = loaded
	}

	if ts.IsEncapsulated() {
		ev, ok := val.(*encapsulatedValue)
		if !ok {
			return nil, &ValueError{
				Tag: TagPixelData, VR: e.VR,
				Msg: "encapsulated Pixel Data carries no retained fragment stream; build it with NewEncapsulatedPixelData",
			}
		}
		return NewEncapsulatedPixelData(ds, ts, ev.stream)
	}

	geom, err := ResolvePixelGeometry(ds, ts)
	if err != nil {
		return nil, err
	}
	b, ok := val.(*Bytes)
	if !ok {
		return nil, &ValueError{Tag: TagPixelData, VR: e.VR, Msg: "native Pixel Data is not an OB/OW byte value"}
	}
	return newNativePixelData(geom, b.Bytes()), nil
}

// newNativePixelData wraps a contiguous native buffer with its geometry. The buffer
// is owned by the PixelData; callers pass a copy.
func newNativePixelData(geom PixelGeometry, native []byte) *PixelData {
	return &PixelData{Geometry: geom, native: native}
}

// NewEncapsulatedPixelData parses the raw encapsulated (7FE0,0010) value bytes for ds
// under ts into a PixelData. The fragment stream is parsed as a bounded item stream
// (Codex DCM-006); the Extended Offset Table, when present in ds, is preferred over
// the Basic Offset Table for frame mapping. It returns a typed error for a
// non-encapsulated transfer syntax or a malformed fragment stream.
func NewEncapsulatedPixelData(ds *DataSet, ts TransferSyntax, value []byte) (*PixelData, error) {
	if !ts.IsEncapsulated() {
		return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "transfer syntax is not encapsulated"}
	}
	geom, err := ResolvePixelGeometry(ds, ts)
	if err != nil {
		return nil, err
	}
	enc, err := parseEncapsulated(value, geom.NumberOfFrames)
	if err != nil {
		return nil, err
	}
	pd := &PixelData{Geometry: geom, encaps: enc}
	if eot, ok := extendedOffsetTable(ds); ok {
		pd.extended = eot
	}
	return pd, nil
}

// IsEncapsulated reports whether the pixel data is carried as compressed fragments.
func (p *PixelData) IsEncapsulated() bool { return p.encaps != nil }

// BasicOffsetTable returns the Basic Offset Table (32-bit per-frame offsets) of
// encapsulated pixel data. ok is false for native data or when there was no Basic
// Offset Table item.
func (p *PixelData) BasicOffsetTable() ([]uint32, bool) {
	if p.encaps == nil {
		return nil, false
	}
	return p.encaps.basicOffsetTable()
}

// Frames iterates decoded frames. For native data it slices the contiguous buffer
// into NumberOfFrames frames of FrameLength bytes each, failing closed if the buffer
// is shorter than the declared frame count (no truncated final frame). For
// encapsulated data it decodes each frame's fragment group through the codec
// registered for the transfer syntax, yielding a typed CodecUnavailableError when no
// codec is built in.
func (p *PixelData) Frames() iter.Seq2[Frame, error] {
	if p.encaps != nil {
		return p.encapsulatedFrames()
	}
	return p.nativeFrames()
}

// encapsulatedFrames decodes each frame's fragment group through the registered
// codec. When no codec is registered for the transfer syntax (a JPEG-family instance
// in a pure-Go build) it yields a single typed CodecUnavailableError naming the
// transfer syntax, never a partial image.
func (p *PixelData) encapsulatedFrames() iter.Seq2[Frame, error] {
	return func(yield func(Frame, error) bool) {
		codec, ok := lookupCodec(p.Geometry.TransferSyntax)
		if !ok {
			yield(Frame{}, newCodecUnavailable(p.Geometry.TransferSyntax))
			return
		}

		encoded, err := p.frameEncodedBytes()
		if err != nil {
			yield(Frame{}, err)
			return
		}
		for i, enc := range encoded {
			decoded, err := codec.Decode(enc, p.Geometry)
			if err != nil {
				yield(Frame{Index: i}, err)
				return
			}
			if !yield(Frame{Index: i, Pixels: decoded}, nil) {
				return
			}
		}
	}
}

// frameEncodedBytes returns each frame's concatenated encoded bytes, mapped by the
// Extended Offset Table when present, otherwise by the Basic Offset Table or the
// one-fragment-per-frame fallback.
func (p *PixelData) frameEncodedBytes() ([][]byte, error) {
	if p.extended != nil {
		return p.encaps.framesViaExtendedOffsets(p.extended)
	}
	ranges, err := p.encaps.validateFrameMapping(p.Geometry.NumberOfFrames)
	if err != nil {
		return nil, err
	}
	out := make([][]byte, len(ranges))
	for i, r := range ranges {
		out[i] = p.encaps.frameBytes(r)
	}
	return out, nil
}

// nativeFrames slices the contiguous native buffer per the geometry, or per the
// explicit frameLen override when frames were produced by a decode.
func (p *PixelData) nativeFrames() iter.Seq2[Frame, error] {
	return func(yield func(Frame, error) bool) {
		frameLen := p.Geometry.FrameLength()
		if p.frameLen > 0 {
			frameLen = p.frameLen
		}
		frames := p.Geometry.NumberOfFrames
		if frameLen <= 0 || frames <= 0 {
			yield(Frame{}, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "non-positive frame length or frame count"})
			return
		}
		need := frameLen * frames
		if len(p.native) < need {
			yield(Frame{}, &ValueError{
				Tag: TagPixelData, VR: VROBorOW,
				Msg: "native pixel data shorter than the declared frame count",
			})
			return
		}
		for i := 0; i < frames; i++ {
			start := i * frameLen
			out := make([]byte, frameLen)
			copy(out, p.native[start:start+frameLen])
			if !yield(Frame{Index: i, Pixels: out}, nil) {
				return
			}
		}
	}
}
