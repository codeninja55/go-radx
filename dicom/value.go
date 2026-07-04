package dicom

import (
	"encoding/binary"
	"strings"
)

// Value is the interface every element value implements. It exposes the on-wire VR
// and the even, padded value-field length. EncodedLen never panics, and tolerates a
// nil byte order for zero-length values (used by SetEmpty). It is open to extension:
// later increments add Bytes, Sequence, and PixelData implementations.
type Value interface {
	VR() VR
	EncodedLen(bo binary.ByteOrder) uint32
}

// Strings is the value type for the text VRs (AE AS CS LO SH UC UR UT ST LT DA TM DT
// PN UI). Values are stored decoded through the dataset's Specific Character Set.
//
// For a customisable text VR read under a non-default character set, the reader
// retains the exact, padded value-field bytes in raw so the writer re-emits them
// byte-for-byte (a re-encode could pick different but equivalent ISO 2022 escapes).
// raw is nil for programmatically constructed values and for the default repertoire,
// so those re-encode from vals exactly as before and the prior byte-identical
// fixture round-trips are untouched.
type Strings struct {
	vr   VR
	vals []string
	raw  []byte // the verbatim padded value field, or nil to re-encode from vals
}

// NewStrings constructs a text value under vr.
func NewStrings(vr VR, vals ...string) Value {
	cp := make([]string, len(vals))
	copy(cp, vals)
	return &Strings{vr: vr, vals: cp}
}

// newStringsRaw constructs a text value that carries both the decoded values and the
// verbatim padded value field. Used by the reader for customisable text VRs decoded
// under a non-default character set so the write path is byte-exact.
func newStringsRaw(vr VR, raw []byte, vals ...string) *Strings {
	vcp := make([]string, len(vals))
	copy(vcp, vals)
	rcp := make([]byte, len(raw))
	copy(rcp, raw)
	return &Strings{vr: vr, vals: vcp, raw: rcp}
}

func (v *Strings) VR() VR { return v.vr }

// Strings returns a copy of the value list.
func (v *Strings) Strings() []string {
	cp := make([]string, len(v.vals))
	copy(cp, v.vals)
	return cp
}

// EncodedLen joins the values with backslash and pads the whole field to an even
// length with the VR pad byte (Codex DCM-007: the entire character field is even).
// A value retaining its verbatim raw bytes reports their length unchanged.
func (v *Strings) EncodedLen(binary.ByteOrder) uint32 {
	if v.raw != nil {
		return uint32(len(v.raw)) // #nosec G115 -- raw is only set on the read path from a 32-bit length field
	}
	if len(v.vals) == 0 {
		return 0
	}
	// #nosec G115 -- a value past maxValueFieldLen truncates here, but encodePadded's guard then fails the element's write before any value bytes, so a successful write never carries a truncated length
	n := uint32(len(strings.Join(v.vals, `\`)))
	if n%2 == 1 {
		if _, ok := v.vr.PadByte(); ok {
			n++
		}
	}
	return n
}

// Ints is the value type for SS US SL UL SV UV.
type Ints struct {
	vr   VR
	vals []int64
}

// NewInts constructs an integer value under vr.
func NewInts(vr VR, vals ...int64) Value {
	cp := make([]int64, len(vals))
	copy(cp, vals)
	return &Ints{vr: vr, vals: cp}
}

func (v *Ints) VR() VR { return v.vr }

// Ints returns a copy of the value list.
func (v *Ints) Ints() []int64 {
	cp := make([]int64, len(v.vals))
	copy(cp, v.vals)
	return cp
}

func (v *Ints) EncodedLen(binary.ByteOrder) uint32 {
	// #nosec G115 -- a value past maxValueFieldLen truncates here, but encodeInts's guard then fails the element's write before any value bytes, so a successful write never carries a truncated length
	return uint32(len(v.vals)) * uint32(intSize(v.vr))
}

// intSize is the per-element byte width of an integer VR.
func intSize(vr VR) int {
	switch vr {
	case VRSS, VRUS, VRUSorSS:
		// VRUSorSS is the unresolved Implicit VR LE placeholder for a 16-bit integer
		// (US or SS); both candidates are 2 bytes wide.
		return 2
	case VRSL, VRUL:
		return 4
	case VRSV, VRUV:
		return 8
	default:
		return 0
	}
}

// Floats is the value type for FL FD OF OD.
type Floats struct {
	vr   VR
	vals []float64
}

// NewFloats constructs a floating-point value under vr.
func NewFloats(vr VR, vals ...float64) Value {
	cp := make([]float64, len(vals))
	copy(cp, vals)
	return &Floats{vr: vr, vals: cp}
}

func (v *Floats) VR() VR { return v.vr }

// Floats returns a copy of the value list.
func (v *Floats) Floats() []float64 {
	cp := make([]float64, len(v.vals))
	copy(cp, v.vals)
	return cp
}

func (v *Floats) EncodedLen(binary.ByteOrder) uint32 {
	size := 4
	if v.vr == VRFD || v.vr == VROD {
		size = 8
	}
	// #nosec G115 -- a value past maxValueFieldLen truncates here, but encodeFloats's guard then fails the element's write before any value bytes, so a successful write never carries a truncated length
	return uint32(len(v.vals)) * uint32(size)
}

// Decimals is the value type for DS and IS, carrying lexical-preserving Decimals.
type Decimals struct {
	vr   VR
	vals []Decimal
}

// NewDecimals constructs a DS/IS value under vr.
func NewDecimals(vr VR, vals ...Decimal) Value {
	cp := make([]Decimal, len(vals))
	copy(cp, vals)
	return &Decimals{vr: vr, vals: cp}
}

func (v *Decimals) VR() VR { return v.vr }

// Decimals returns a copy of the value list.
func (v *Decimals) Decimals() []Decimal {
	cp := make([]Decimal, len(v.vals))
	copy(cp, v.vals)
	return cp
}

// EncodedLen joins the preserved lexical forms with backslash and pads to even with
// SPACE (DS/IS pad with SPACE per PS3.5; see Task 1.2 note).
func (v *Decimals) EncodedLen(binary.ByteOrder) uint32 {
	if len(v.vals) == 0 {
		return 0
	}
	parts := make([]string, len(v.vals))
	for i, d := range v.vals {
		parts[i] = d.String()
	}
	// #nosec G115 -- a value past maxValueFieldLen truncates here, but encodePadded's guard then fails the element's write before any value bytes, so a successful write never carries a truncated length
	n := uint32(len(strings.Join(parts, `\`)))
	if n%2 == 1 {
		n++
	}
	return n
}

// Tags is the value type for AT (each value is a 4-byte tag).
type Tags struct {
	vals []Tag
}

// NewTags constructs an AT value.
func NewTags(vals ...Tag) Value {
	cp := make([]Tag, len(vals))
	copy(cp, vals)
	return &Tags{vals: cp}
}

func (v *Tags) VR() VR { return VRAT }

// Tags returns a copy of the value list.
func (v *Tags) Tags() []Tag {
	cp := make([]Tag, len(v.vals))
	copy(cp, v.vals)
	return cp
}

// EncodedLen is four bytes per tag; a value past maxValueFieldLen truncates here,
// but encodeTags's guard then fails the element's write before any value bytes, so
// a successful write never carries a truncated length.
func (v *Tags) EncodedLen(binary.ByteOrder) uint32 { return uint32(len(v.vals)) * 4 } // #nosec G115

// Bytes is the value type for OB OW OL OV UN (length-bounded raw bytes).
type Bytes struct {
	vr VR
	b  []byte // owned; never aliased to a caller slice
}

// NewBytes copies b so the value owns its bytes.
func NewBytes(vr VR, b []byte) Value {
	cp := make([]byte, len(b))
	copy(cp, b)
	return &Bytes{vr: vr, b: cp}
}

// newBytesOwned wraps b without copying; the caller relinquishes b. The decode
// paths use it on buffers only they hold, so a dataset's largest value field is
// allocated once rather than twice; NewBytes keeps copying for external callers.
func newBytesOwned(vr VR, b []byte) Value {
	return &Bytes{vr: vr, b: b}
}

func (v *Bytes) VR() VR { return v.vr }

// Bytes returns a copy of the value field.
func (v *Bytes) Bytes() []byte {
	cp := make([]byte, len(v.b))
	copy(cp, v.b)
	return cp
}

// EncodedLen is the byte length padded up to even with a trailing NULL.
func (v *Bytes) EncodedLen(binary.ByteOrder) uint32 {
	n := uint32(len(v.b)) // #nosec G115 -- a value past maxValueFieldLen truncates here, but encodeBytes's guard then fails the element's write before any value bytes, so a successful write never carries a truncated length
	if n%2 == 1 {
		n++
	}
	return n
}
