package dicom

import (
	"fmt"
)

// Overlay element offsets within a 60xx repeating group (PS3.3 C.9; PS3.6 registry).
// The group half is variable (6000, 6002, ... even groups); these are the fixed
// element halves of each attribute the overlay-plane extraction reads.
const (
	overlayElemRows          uint16 = 0x0010 // Overlay Rows (US)
	overlayElemColumns       uint16 = 0x0011 // Overlay Columns (US)
	overlayElemType          uint16 = 0x0040 // Overlay Type (CS): G (graphics) or R (ROI)
	overlayElemOrigin        uint16 = 0x0050 // Overlay Origin (SS, VM 2): {row, column}, 1-based
	overlayElemBitsAllocated uint16 = 0x0100 // Overlay Bits Allocated (US): must be 1
	overlayElemBitPosition   uint16 = 0x0102 // Overlay Bit Position (US): 0 for non-embedded overlays
	overlayElemData          uint16 = 0x3000 // Overlay Data (OB/OW): the packed bitmap
)

// firstOverlayGroup is the lowest overlay repeating-group number (PS3.3 C.9.2). Overlay
// groups are the even values 6000, 6002, ... 60FE; lastOverlayGroup is the highest.
const (
	firstOverlayGroup uint16 = 0x6000
	lastOverlayGroup  uint16 = 0x60FE
)

// Overlay is one extracted overlay plane: its geometry, classification, and the
// unpacked 1-bit-per-pixel bitmap as a dense row-major boolean plane. Bits is
// len == Rows*Columns; Bits[r*Columns+c] is the overlay pixel at (row r, column c),
// true where the overlay is set.
type Overlay struct {
	Group   uint16 // the 60xx group this plane was read from
	Rows    int    // Overlay Rows (60xx,0010)
	Columns int    // Overlay Columns (60xx,0011)
	Type    string // Overlay Type (60xx,0040): "G" graphics, "R" ROI; empty if absent
	// OriginRow and OriginColumn are the 1-based position of the overlay's first
	// pixel in the image (Overlay Origin, 60xx,0050), defaulting to (1,1) when absent.
	OriginRow    int
	OriginColumn int
	Bits         []bool // row-major, len == Rows*Columns
}

// At reports the overlay bit at (row, column). It panics on an out-of-range index,
// matching slice-indexing semantics; callers bound row/column by Rows/Columns.
func (o *Overlay) At(row, column int) bool {
	return o.Bits[row*o.Columns+column]
}

// OverlayGroups returns the overlay repeating-group numbers present in the dataset, in
// ascending order. A group is present when its Overlay Data element (60xx,3000) exists.
// This mirrors pydicom's group_dataset / overlay_array(group) discovery.
func (ds *DataSet) OverlayGroups() []uint16 {
	var groups []uint16
	for g := firstOverlayGroup; g <= lastOverlayGroup; g += 2 {
		if _, ok := ds.Get(NewTag(g, overlayElemData)); ok {
			groups = append(groups, g)
		}
	}
	return groups
}

// OverlayArray extracts the overlay plane stored in the repeating group group
// (6000, 6002, ... 60FE), unpacking the 1-bit-per-pixel packed bitmap into a dense
// boolean plane. It is the go-radx analogue of pydicom's Dataset.overlay_array(group).
//
// Bit layout (PS3.5 §8.1.2 "Overlay data encoding of related data elements"): overlay
// data is sent as 1 bit per pixel, packed eight pixels to a byte, ordered
// least-significant bit first (the first overlay pixel of a byte is bit 0). Pixels run
// row-major; the bit stream is contiguous across rows with no per-row byte alignment, so
// pixel index i (i = r*Columns + c) lives in byte i/8 at bit i%8. The whole field is
// padded to an even byte count (OW), which this reader tolerates.
//
// Embedded-in-pixel-data overlays — Overlay Bit Position != 0, which packed the overlay
// into the high bits of the image's Pixel Data — were retired in DICOM 2004 and are out
// of scope: a non-zero Overlay Bit Position is rejected with a typed error rather than
// guessed at, because for such an object the standalone Overlay Data element is absent.
func (ds *DataSet) OverlayArray(group uint16) (*Overlay, error) {
	if group < firstOverlayGroup || group > lastOverlayGroup || group%2 != 0 {
		return nil, fmt.Errorf("dicom: %04X is not an overlay repeating group (even 6000..60FE)", group)
	}

	rows, ok := ds.GetInt(NewTag(group, overlayElemRows))
	if !ok {
		return nil, fmt.Errorf("dicom: overlay group %04X is missing Overlay Rows (%04X,%04X)", group, group, overlayElemRows)
	}
	cols, ok := ds.GetInt(NewTag(group, overlayElemColumns))
	if !ok {
		return nil, fmt.Errorf("dicom: overlay group %04X is missing Overlay Columns (%04X,%04X)", group, group, overlayElemColumns)
	}
	if rows <= 0 || cols <= 0 {
		return nil, fmt.Errorf("dicom: overlay group %04X has non-positive dimensions %dx%d", group, rows, cols)
	}

	// Overlay Bits Allocated must be 1 (PS3.3 C.9.2): an overlay is a bitmap, one bit
	// per pixel. A value other than 1 is malformed input, refused rather than guessed.
	if bits, ok := ds.GetInt(NewTag(group, overlayElemBitsAllocated)); ok && bits != 1 {
		return nil, fmt.Errorf("dicom: overlay group %04X has Overlay Bits Allocated %d, must be 1", group, bits)
	}

	// A non-zero Overlay Bit Position means the bitmap was embedded in the image Pixel
	// Data (retired in DICOM 2004). Reject it honestly instead of misreading the
	// standalone Overlay Data element, which for an embedded overlay is absent.
	if pos, ok := ds.GetInt(NewTag(group, overlayElemBitPosition)); ok && pos != 0 {
		return nil, fmt.Errorf("dicom: overlay group %04X uses retired embedded-in-pixel-data encoding (Overlay Bit Position %d); unsupported", group, pos)
	}

	dataElem, ok := ds.Get(NewTag(group, overlayElemData))
	if !ok {
		return nil, fmt.Errorf("dicom: overlay group %04X is missing Overlay Data (%04X,%04X)", group, group, overlayElemData)
	}
	packed, ok := binaryValueBytes(dataElem.Value)
	if !ok {
		return nil, fmt.Errorf("dicom: overlay group %04X Overlay Data is not a binary (OB/OW) value", group)
	}

	count := int(rows) * int(cols)
	needBytes := (count + 7) / 8
	if len(packed) < needBytes {
		return nil, fmt.Errorf("dicom: overlay group %04X Overlay Data is %d bytes, need %d for %dx%d", group, len(packed), needBytes, rows, cols)
	}

	o := &Overlay{
		Group:        group,
		Rows:         int(rows),
		Columns:      int(cols),
		OriginRow:    1,
		OriginColumn: 1,
		Bits:         unpackBitsLSB(packed, count),
	}
	if t, ok := ds.GetString(NewTag(group, overlayElemType)); ok {
		o.Type = t
	}
	if origin, ok := overlayOrigin(ds, NewTag(group, overlayElemOrigin)); ok {
		o.OriginRow = origin[0]
		o.OriginColumn = origin[1]
	}
	return o, nil
}

// overlayOrigin reads the Overlay Origin (SS, VM 2) {row, column} pair. ok is false
// unless exactly two signed-short values are present.
func overlayOrigin(ds *DataSet, t Tag) ([2]int, bool) {
	e, ok := ds.Get(t)
	if !ok {
		return [2]int{}, false
	}
	mv, ok := materialise(e.Value)
	if !ok {
		return [2]int{}, false
	}
	iv, ok := mv.(*Ints)
	if !ok {
		return [2]int{}, false
	}
	vals := iv.Ints()
	if len(vals) != 2 {
		return [2]int{}, false
	}
	return [2]int{int(vals[0]), int(vals[1])}, true
}

// unpackBitsLSB expands the first count bits of packed into a boolean slice, reading
// bits least-significant-first within each byte (PS3.5 §8.1.2). Pixel i is bit i%8 of
// byte i/8.
func unpackBitsLSB(packed []byte, count int) []bool {
	out := make([]bool, count)
	for i := range count {
		out[i] = packed[i/8]&(1<<(uint(i)%8)) != 0
	}
	return out
}

// binaryValueBytes returns the raw value field of an OB/OW/OL/OV/UN element, loading a
// deferred value transparently. It is shared by overlay (bitmap) and waveform (sample)
// extraction, both of which read the on-wire bytes directly.
func binaryValueBytes(v Value) ([]byte, bool) {
	mv, ok := materialise(v)
	if !ok {
		return nil, false
	}
	b, ok := mv.(*Bytes)
	if !ok {
		return nil, false
	}
	return b.Bytes(), true
}
