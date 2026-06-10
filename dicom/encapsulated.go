package dicom

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// fragment is one (FFFE,E000) item of an encapsulated pixel-data stream. data is a
// copy of the item's value bytes (an even, bounds-checked length); offset is the byte
// position of the fragment's value relative to the first byte after the Basic Offset
// Table item, which is the coordinate the Basic Offset Table uses.
type fragment struct {
	offset uint32
	data   []byte
}

// encapsulated is the parsed fragment stream of a compressed (7FE0,0010) element: an
// optional Basic Offset Table followed by one or more fragment items, terminated by a
// Sequence Delimitation Item. It keeps fragment metadata rather than one unbounded
// blob (Codex DCM-006), so frame reconstruction is bounds-checked and a malformed
// stream is a typed error, never a partial image.
type encapsulated struct {
	bot       []uint32   // Basic Offset Table entries (32-bit frame offsets), may be empty
	fragments []fragment // fragment items in stream order
}

// encapsulatedValue is the Value wrapper for an encapsulated (7FE0,0010) element:
// the verbatim fragment item stream — Basic Offset Table item, fragment items, and
// Sequence Delimitation Item — exactly as it appeared on the wire. The reader
// retains it byte-for-byte and never decodes it (decode lives in the
// PixelData/Frames pipeline), so a compressed file round-trips with an identical
// pixel stream.
type encapsulatedValue struct {
	stream []byte
}

func (v *encapsulatedValue) VR() VR { return VROB }

// EncodedLen reports the undefined-length sentinel: an encapsulated pixel-data
// element is always written with undefined length and is delimited by the Sequence
// Delimitation Item its stream ends with (PS3.5 A.4).
func (v *encapsulatedValue) EncodedLen(binary.ByteOrder) uint32 { return undefinedLength }

// frameRange names the fragments that compose one frame as a half-open fragment
// index range [first, last).
type frameRange struct {
	first int
	last  int
}

// parseEncapsulated reads the encapsulated pixel-data value bytes as a bounded stream
// of items. The first item is the Basic Offset Table (FFFE,E000) — possibly empty —
// per pydicom's parse_basic_offsets; the remaining items are fragments, each with an
// even length validated against the bytes remaining before any allocation; the stream
// ends with a Sequence Delimitation Item (FFFE,E0DD). A truncated trailing header, an
// odd item length, a length past the bytes remaining, or a missing delimiter is a
// typed error, never a silent break (Codex DCM-006).
func parseEncapsulated(data []byte, numFrames int) (*encapsulated, error) {
	br := newBoundedReader(bytes.NewReader(data), defaultMaxElementLen)

	tag, length, err := readDelimiterHeader(br, ExplicitVRLittleEndian)
	if err != nil {
		return nil, err
	}
	if tag != tagItem {
		return nil, &ValueError{
			Tag: TagPixelData, VR: VROBorOW,
			Msg: "encapsulated pixel data must begin with a Basic Offset Table item",
		}
	}
	if length == undefinedLength || length%2 != 0 {
		return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "Basic Offset Table item has an invalid length"}
	}
	botBytes, err := br.readN(length)
	if err != nil {
		return nil, err
	}
	enc := &encapsulated{bot: parseOffsets(botBytes)}

	// Fragment offsets are measured from the first byte after the Basic Offset Table
	// item (its value end), which is the current reader position. The Basic Offset
	// Table references each fragment by its item-header position, not its value, so the
	// recorded offset is captured before the 8-byte item header is read.
	base := br.offset()
	for {
		// Fragment offsets are 32-bit Basic Offset Table values (PS3.5 A.4); a
		// stream past that range cannot be referenced and is rejected.
		if br.offset()-base > int64(math.MaxUint32) {
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "encapsulated stream exceeds the 32-bit offset table"}
		}
		headerOff := uint32(br.offset() - base) // #nosec G115 -- bounded by the MaxUint32 check above
		tag, length, err := readDelimiterHeader(br, ExplicitVRLittleEndian)
		if err != nil {
			return nil, err
		}
		switch tag {
		case tagSequenceDelimit:
			if length != 0 {
				return nil, &ValueError{Tag: tagSequenceDelimit, VR: VROBorOW, Msg: "Sequence Delimitation Item must have zero length"}
			}
			if numFrames > 0 {
				if _, err := enc.validateFrameMapping(numFrames); err != nil {
					return nil, err
				}
			}
			return enc, nil
		case tagItem:
			if length == undefinedLength {
				return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "fragment item has undefined length"}
			}
			if length%2 != 0 {
				return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: "fragment item length is odd"}
			}
			off := headerOff
			value, err := br.readN(length)
			if err != nil {
				return nil, err
			}
			cp := make([]byte, len(value))
			copy(cp, value)
			enc.fragments = append(enc.fragments, fragment{offset: off, data: cp})
		default:
			return nil, &ValueError{Tag: TagPixelData, VR: VROBorOW, Msg: fmt.Sprintf("unexpected tag %s inside encapsulated pixel data", tag)}
		}
	}
}

// parseOffsets reads a Basic Offset Table value into 32-bit offsets.
func parseOffsets(b []byte) []uint32 {
	n := len(b) / 4
	out := make([]uint32, n)
	for i := 0; i < n; i++ {
		out[i] = binary.LittleEndian.Uint32(b[i*4 : i*4+4])
	}
	return out
}

// basicOffsetTable returns the parsed Basic Offset Table offsets. ok is true whenever
// the stream had a Basic Offset Table item, even if it was empty.
func (e *encapsulated) basicOffsetTable() ([]uint32, bool) {
	return e.bot, true
}

// frameRanges maps frames to fragment index ranges. When the Basic Offset Table has
// entries, each offset marks the first fragment of a frame and the frames partition
// the fragments by those offsets. With an empty Basic Offset Table the fallback is
// one fragment per frame (the common single-fragment-per-frame layout).
func (e *encapsulated) frameRanges() []frameRange {
	if len(e.bot) == 0 {
		ranges := make([]frameRange, len(e.fragments))
		for i := range e.fragments {
			ranges[i] = frameRange{first: i, last: i + 1}
		}
		return ranges
	}

	// Map each Basic Offset Table offset to the fragment that begins there.
	starts := make([]int, 0, len(e.bot))
	for _, off := range e.bot {
		idx := e.fragmentIndexAtOffset(off)
		starts = append(starts, idx)
	}
	ranges := make([]frameRange, len(starts))
	for i := range starts {
		first := starts[i]
		last := len(e.fragments)
		if i+1 < len(starts) {
			last = starts[i+1]
		}
		ranges[i] = frameRange{first: first, last: last}
	}
	return ranges
}

// fragmentIndexAtOffset returns the index of the fragment whose offset equals off, or
// -1 if no fragment begins exactly there.
func (e *encapsulated) fragmentIndexAtOffset(off uint32) int {
	for i := range e.fragments {
		if e.fragments[i].offset == off {
			return i
		}
	}
	return -1
}

// validateFrameMapping checks that the Basic Offset Table (when present) resolves to
// real fragments and produces exactly numFrames frames. It returns the frame ranges
// or a typed error so a malformed offset table fails closed rather than yielding a
// blank or misaligned frame (Codex DCM-006).
func (e *encapsulated) validateFrameMapping(numFrames int) ([]frameRange, error) {
	if len(e.bot) > 0 {
		for i, off := range e.bot {
			if e.fragmentIndexAtOffset(off) < 0 {
				return nil, &ValueError{
					Tag: TagPixelData, VR: VROBorOW,
					Msg: fmt.Sprintf("Basic Offset Table entry %d (offset %d) matches no fragment", i, off),
				}
			}
		}
		if len(e.bot) != numFrames {
			return nil, &ValueError{
				Tag: TagPixelData, VR: VROBorOW,
				Msg: fmt.Sprintf("Basic Offset Table has %d entries but the dataset declares %d frames", len(e.bot), numFrames),
			}
		}
		return e.frameRanges(), nil
	}

	// Empty Basic Offset Table: accept the one-fragment-per-frame layout, otherwise
	// the dataset's frame count must match the fragment count.
	if len(e.fragments) == numFrames {
		return e.frameRanges(), nil
	}
	if numFrames == 1 {
		// A single frame may be split across all fragments.
		return []frameRange{{first: 0, last: len(e.fragments)}}, nil
	}
	return nil, &ValueError{
		Tag: TagPixelData, VR: VROBorOW,
		Msg: fmt.Sprintf("empty Basic Offset Table with %d fragments cannot map to %d frames", len(e.fragments), numFrames),
	}
}

// frameBytes concatenates the value bytes of the fragments in r into one encoded
// frame buffer.
func (e *encapsulated) frameBytes(r frameRange) []byte {
	var n int
	for i := r.first; i < r.last; i++ {
		n += len(e.fragments[i].data)
	}
	out := make([]byte, 0, n)
	for i := r.first; i < r.last; i++ {
		out = append(out, e.fragments[i].data...)
	}
	return out
}
