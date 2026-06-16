package dicom

import (
	"encoding/binary"
	"fmt"
)

// extendedOffsets is the parsed Extended Offset Table: the per-frame 64-bit byte
// offsets (7FE0,0001) and matching byte lengths (7FE0,0002), both addressing the
// concatenated fragment value stream (the fragment value bytes laid end to end,
// excluding item headers and the Basic Offset Table). PS3.5 §A.4 defines it as the
// precise, unambiguous alternative to the 32-bit Basic Offset Table for large
// multi-frame objects.
type extendedOffsets struct {
	offsets []uint64
	lengths []uint64
}

// extendedOffsetTable reads the Extended Offset Table from ds when both the offset
// (7FE0,0001) and length (7FE0,0002) elements are present and agree in count. OV
// values are stored as raw OV bytes; each entry is a little-endian uint64.
func extendedOffsetTable(ds *DataSet) (*extendedOffsets, bool) {
	off, okO := ovUint64s(ds, TagExtendedOffsetTable)
	ln, okL := ovUint64s(ds, TagExtendedOffsetTableLengths)
	if !okO || !okL {
		return nil, false
	}
	if len(off) == 0 || len(off) != len(ln) {
		return nil, false
	}
	return &extendedOffsets{offsets: off, lengths: ln}, true
}

// ovUint64s reads a 64-bit OV value at tag into a uint64 slice.
func ovUint64s(ds *DataSet, tag Tag) ([]uint64, bool) {
	e, ok := ds.Get(tag)
	if !ok {
		return nil, false
	}
	v, ok := materialise(e.Value)
	if !ok {
		return nil, false
	}
	b, ok := v.(*Bytes)
	if !ok {
		return nil, false
	}
	raw := b.Bytes()
	if len(raw)%8 != 0 {
		return nil, false
	}
	out := make([]uint64, len(raw)/8)
	for i := range out {
		out[i] = binary.LittleEndian.Uint64(raw[i*8 : i*8+8])
	}
	return out, true
}

// framesViaExtendedOffsets reconstructs frames by slicing the concatenated fragment
// value stream with the Extended Offset Table. Each (offset, length) is bounds-checked
// against the concatenated stream before slicing, so an entry that runs past the
// stream is a typed error rather than an out-of-range read (Codex DCM-006).
func (e *encapsulated) framesViaExtendedOffsets(eot *extendedOffsets) ([][]byte, error) {
	var stream []byte
	for i := range e.fragments {
		stream = append(stream, e.fragments[i].data...)
	}
	frames := make([][]byte, len(eot.offsets))
	for i := range eot.offsets {
		off := eot.offsets[i]
		ln := eot.lengths[i]
		end := off + ln
		// Guard against overflow and an end past the concatenated stream.
		if end < off || end > uint64(len(stream)) {
			return nil, &ValueError{
				Tag: TagExtendedOffsetTable, VR: VROV,
				Msg: fmt.Sprintf("frame %d range [%d,%d) is outside the %d-byte fragment stream", i, off, end, len(stream)),
			}
		}
		out := make([]byte, ln)
		copy(out, stream[off:end])
		frames[i] = out
	}
	return frames, nil
}
