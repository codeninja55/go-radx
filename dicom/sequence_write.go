package dicom

import (
	"bytes"
	"io"
)

// encodeSequenceValue writes a sequence's value field (its items and any
// delimiters) in ts's encoding and returns the byte count written. The SQ element
// header itself is written by the caller; for an undefined-length sequence the
// caller writes the 0xFFFFFFFF length sentinel and this body ends with the Sequence
// Delimitation Item (FFFE,E0DD). Each item re-emits the length form it was read with
// so a sequence round-trips byte-identically (PS3.5 §7.5; Codex DCM-005).
func encodeSequenceValue(w io.Writer, seq *Sequence, ts TransferSyntax) (uint32, error) {
	var n uint32
	for _, it := range seq.items {
		written, err := encodeItem(w, it, ts)
		if err != nil {
			return n, err
		}
		// The running total feeds a defined-length SQ header's 32-bit field; reject a
		// sequence that grows past it rather than wrapping the recorded length. The cap
		// is 0xFFFFFFFE: 0xFFFFFFFF is the undefined-length sentinel (PS3.5 §7.1.1).
		if written > uint32(maxValueFieldLen)-n {
			return n, &ValueError{Tag: tagItem, VR: VRSQ, Msg: "sequence value field exceeds the 32-bit element length"}
		}
		n += written
	}
	if seq.undefinedLength {
		written, err := writeDelimiterHeader(w, tagSequenceDelimit, 0, ts)
		if err != nil {
			return n, err
		}
		n += written
	}
	return n, nil
}

// encodeItem writes one item: its (FFFE,E000) header, the nested dataset, and (for
// an undefined-length item) the Item Delimitation Item (FFFE,E00D).
func encodeItem(w io.Writer, it Item, ts TransferSyntax) (uint32, error) {
	// Serialise the nested dataset to a buffer so a defined-length item can prefix
	// its exact byte count.
	var body bytes.Buffer
	if err := writeDataSet(&body, it.DataSet, ts); err != nil {
		return 0, err
	}
	// The item length and the caller's running sequence length are 32-bit fields
	// (PS3.5 §7.5); an item body past that cannot be represented, and exactly
	// 0xFFFFFFFF would collide with the undefined-length sentinel (PS3.5 §7.1.1).
	// The cap reserves the item's framing too (8-byte header plus an optional
	// 8-byte delimiter), so the returned byte count cannot wrap the 32-bit
	// accounting the running sequence length is summed in.
	if int64(body.Len()) > maxValueFieldLen-16 {
		return 0, &ValueError{Tag: tagItem, VR: VRSQ, Msg: "sequence item exceeds the 32-bit length field"}
	}

	itemLen := undefinedLength
	if !it.undefinedLength {
		itemLen = uint32(body.Len()) // #nosec G115 -- bounded by the maxValueFieldLen check above
	}

	var n uint32
	written, err := writeDelimiterHeader(w, tagItem, itemLen, ts)
	if err != nil {
		return n, err
	}
	n += written

	bw, err := w.Write(body.Bytes())
	n += uint32(bw) // #nosec G115 -- bw <= body.Len(), bounded by the MaxUint32 check above
	if err != nil {
		return n, err
	}

	if it.undefinedLength {
		written, err := writeDelimiterHeader(w, tagItemDelimiter, 0, ts)
		if err != nil {
			return n, err
		}
		n += written
	}
	return n, nil
}

// writeDelimiterHeader writes a bare 8-byte item/delimiter header (tag + 4-byte
// length, no VR) in ts's byte order.
func writeDelimiterHeader(w io.Writer, tag Tag, length uint32, ts TransferSyntax) (uint32, error) {
	bo := encodingFor(ts).byteOrder
	var b [8]byte
	bo.PutUint16(b[0:2], tag.Group())
	bo.PutUint16(b[2:4], tag.Element())
	bo.PutUint32(b[4:8], length)
	n, err := w.Write(b[:])
	return uint32(n), err // #nosec G115 -- n <= 8 (the fixed header size)
}

// sequenceEncodedLen returns the value-field byte length the sequence serialises to
// in ts, computed by encoding into a discarding counter so it matches
// encodeSequenceValue exactly.
func sequenceEncodedLen(seq *Sequence, ts TransferSyntax) (uint32, error) {
	return encodeSequenceValue(io.Discard, seq, ts)
}
