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

	native []byte // contiguous samples for an uncompressed transfer syntax
}

// NewPixelData builds the PixelData for ds under ts. For an uncompressed transfer
// syntax it reads the native (7FE0,0010) value from ds. For an encapsulated transfer
// syntax the fragment stream must be supplied through NewEncapsulatedPixelData,
// because the dataset reader does not retain the raw fragment bytes; this
// constructor returns a typed error for an encapsulated syntax.
func NewPixelData(ds *DataSet, ts TransferSyntax) (*PixelData, error) {
	geom, err := ResolvePixelGeometry(ds, ts)
	if err != nil {
		return nil, err
	}
	if ts.IsEncapsulated() {
		return nil, &ValueError{
			Tag: TagPixelData, VR: VROBorOW,
			Msg: "encapsulated pixel data must be built with NewEncapsulatedPixelData",
		}
	}

	e, ok := ds.Get(TagPixelData)
	if !ok {
		return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "dataset has no Pixel Data element"}
	}
	b, ok := e.Value.(*Bytes)
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

// IsEncapsulated reports whether the pixel data is carried as compressed fragments.
func (p *PixelData) IsEncapsulated() bool { return p.Geometry.TransferSyntax.IsEncapsulated() }

// Frames iterates decoded frames. For native data it slices the contiguous buffer
// into NumberOfFrames frames of FrameLength bytes each, failing closed if the buffer
// is shorter than the declared frame count (no truncated final frame). For
// encapsulated data it decodes each frame's fragment group through the codec
// registered for the transfer syntax, yielding a typed CodecUnavailableError when no
// codec is built in.
func (p *PixelData) Frames() iter.Seq2[Frame, error] {
	return p.nativeFrames()
}

// nativeFrames slices the contiguous native buffer per the geometry.
func (p *PixelData) nativeFrames() iter.Seq2[Frame, error] {
	return func(yield func(Frame, error) bool) {
		frameLen := p.Geometry.FrameLength()
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
