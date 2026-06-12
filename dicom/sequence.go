package dicom

import (
	"encoding/binary"
	"iter"
)

// Item is one entry of a Sequence: a nested DataSet. The undefinedLength field
// remembers whether the item was read with the undefined-length encoding (an Item
// Delimitation Item terminates it) versus a defined byte count, so the writer can
// re-emit the same on-wire form and a sequence-bearing file round-trips
// byte-identically (PS3.5 §7.5).
type Item struct {
	DataSet *DataSet

	// undefinedLength is true when the item was read (or is to be written) with the
	// 0xFFFFFFFF length sentinel terminated by an Item Delimitation Item (FFFE,E00D).
	undefinedLength bool

	// fileOffset is the byte offset of this item's (FFFE,E000) tag from the start of
	// the stream the enclosing dataset was read from. For a Part 10 file read through
	// Read/ReadFile that origin is byte 0 of the file (the preamble), which is exactly
	// the reference point of the DICOMDIR offset elements (PS3.10 Table 8.1-1), so the
	// file-set loader can resolve offset-linked directory records without re-scanning.
	// It is a read artifact: programmatically built items carry 0 and a clone does not
	// preserve it.
	fileOffset int64
}

// Sequence is VR SQ: an ordered list of Item values, each a nested DataSet (PS3.5
// §7.5). Sequences nest arbitrarily and may be encoded defined-length or
// undefined-length (terminated by a Sequence Delimitation Item). The undefinedLength
// field preserves the source encoding so the writer round-trips it without loss
// (Codex DCM-005: sequences are first-class nested datasets, never dropped).
//
// Sequence is NOT safe for concurrent mutation; the same single-threaded ownership
// rule as DataSet applies (Codex DCM-016).
type Sequence struct {
	items []Item

	// undefinedLength is true when the sequence itself was read (or is to be written)
	// with the 0xFFFFFFFF length sentinel terminated by a Sequence Delimitation Item
	// (FFFE,E0DD).
	undefinedLength bool
}

// NewSequence builds a sequence from the given item datasets. Programmatically
// constructed sequences default to the undefined-length encoding (both the sequence
// and each item), the broadly accepted length-agnostic form: a writer need not
// pre-count nested content. Read sequences override these flags to match the source.
func NewSequence(items ...*DataSet) *Sequence {
	s := &Sequence{undefinedLength: true}
	for _, ds := range items {
		s.Append(ds)
	}
	return s
}

// Append adds ds as a new undefined-length item at the end of the sequence.
func (s *Sequence) Append(ds *DataSet) {
	s.items = append(s.items, Item{DataSet: ds, undefinedLength: true})
}

// Len returns the number of items in the sequence.
func (s *Sequence) Len() int { return len(s.items) }

// Items iterates the sequence items in order.
func (s *Sequence) Items() iter.Seq[Item] {
	return func(yield func(Item) bool) {
		for _, it := range s.items {
			if !yield(it) {
				return
			}
		}
	}
}

// sequenceValue is the Value wrapper for an SQ element.
type sequenceValue struct {
	seq *Sequence
}

// NewSequenceValue wraps s as an SQ Value. s must be non-nil.
func NewSequenceValue(s *Sequence) Value {
	return &sequenceValue{seq: s}
}

func (v *sequenceValue) VR() VR { return VRSQ }

// EncodedLen reports the value-field byte length the sequence serialises to under
// Explicit VR Little Endian. The exact on-wire length depends on the dataset's
// transfer syntax (implicit vs explicit VR changes element-header widths), which the
// Value interface's byte-order-only signature cannot convey, so the SQ writer
// computes the authoritative length from the full transfer syntax. This method is a
// best-effort hint for callers sizing buffers; for an undefined-length sequence the
// header still carries the 0xFFFFFFFF sentinel rather than this count.
func (v *sequenceValue) EncodedLen(bo binary.ByteOrder) uint32 {
	ts := ExplicitVRLittleEndian
	if bo == binary.BigEndian {
		ts = ExplicitVRBigEndian
	}
	n, err := sequenceEncodedLen(v.seq, ts)
	if err != nil {
		return 0
	}
	return n
}

// cloneSequence deep-copies a sequence: every item's nested DataSet is cloned so a
// mutation of the copy never reaches the source (Codex DCM-016). Length-form flags
// are preserved so a clone re-encodes byte-identically.
func cloneSequence(s *Sequence) *Sequence {
	out := &Sequence{undefinedLength: s.undefinedLength}
	out.items = make([]Item, len(s.items))
	for i, it := range s.items {
		out.items[i] = Item{DataSet: it.DataSet.Clone(), undefinedLength: it.undefinedLength}
	}
	return out
}
