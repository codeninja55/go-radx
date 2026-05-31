package dicom

import (
	"bytes"
	"encoding/binary"
)

// Delimiter tags shared by sequences, items, and encapsulated pixel data.
var (
	tagItem            = NewTag(0xFFFE, 0xE000)
	tagItemDelimiter   = NewTag(0xFFFE, 0xE00D)
	tagSequenceDelimit = NewTag(0xFFFE, 0xE0DD)
)

// scanUndefinedLengthValue captures, as opaque bytes, an undefined-length SQ value
// field: everything from the first item header through the Sequence Delimitation
// Item (FFFE,E0DD) and its 4-byte trailer, inclusive. The bytes are preserved
// exactly so a sequence-bearing element round-trips byte-identically for Increment
// 2 (Codex DCM-005); Increment 3 replaces this with structured parsing. The scan is
// bounds-checked: a stream ending before the delimiter is io.ErrUnexpectedEOF
// (Codex DCM-003).
//
// Item and delimiter tags carry no VR and always use the 4-byte length form, in the
// transfer syntax's byte order. An undefined-length item's content is scanned
// element by element to its Item Delimitation Item (FFFE,E00D); a defined-length
// item's content is captured verbatim by its length, which already delimits any
// nested sequences.
func scanUndefinedLengthValue(br *boundedReader, ts TransferSyntax, owner Tag) ([]byte, error) {
	bo := encodingFor(ts).byteOrder
	var out bytes.Buffer
	for {
		tag, length, err := scanDelimiterHeader(br, bo, &out)
		if err != nil {
			return nil, err
		}
		switch tag {
		case tagSequenceDelimit:
			return out.Bytes(), nil
		case tagItem:
			if length == undefinedLength {
				if err := scanUndefinedLengthItem(br, ts, &out); err != nil {
					return nil, err
				}
				continue
			}
			if err := captureBytes(br, length, &out, owner); err != nil {
				return nil, err
			}
		default:
			// Any tag other than an item or the sequence delimiter at this level is
			// malformed; treat the captured stream as truncated/invalid input.
			return nil, &ValueError{Tag: owner, VR: VRSQ, Msg: "unexpected tag inside undefined-length sequence"}
		}
	}
}

// scanUndefinedLengthItem scans one undefined-length item's content into out, up to
// and including its Item Delimitation Item (FFFE,E00D). Nested undefined-length
// sequences recurse through scanUndefinedLengthValue.
func scanUndefinedLengthItem(br *boundedReader, ts TransferSyntax, out *bytes.Buffer) error {
	bo := encodingFor(ts).byteOrder
	for {
		// Peek the tag through the element header reader path so implicit/explicit VR
		// resolution matches the dataset encoding.
		h, raw, err := readAndCaptureElementHeader(br, ts, out)
		if err != nil {
			return err
		}
		if h.tag == tagItemDelimiter {
			return nil // out already holds the delimiter header
		}
		_ = raw
		if h.length == undefinedLength {
			// A nested undefined-length sequence (or item) inside this item.
			nested, err := scanUndefinedLengthValue(br, ts, h.tag)
			if err != nil {
				return err
			}
			out.Write(nested)
			continue
		}
		if err := captureBytes(br, h.length, out, h.tag); err != nil {
			return err
		}
		_ = bo
	}
}

// scanDelimiterHeader reads a bare 8-byte item/delimiter header (4-byte tag +
// 4-byte length, no VR), appending it to out, for the sequence-level scan.
func scanDelimiterHeader(br *boundedReader, bo binary.ByteOrder, out *bytes.Buffer) (Tag, uint32, error) {
	b, err := br.readExact(8)
	if err != nil {
		return 0, 0, midElementEOF(err)
	}
	out.Write(b)
	tag := NewTag(bo.Uint16(b[0:2]), bo.Uint16(b[2:4]))
	length := bo.Uint32(b[4:8])
	return tag, length, nil
}

// readAndCaptureElementHeader reads one element header in ts's encoding and appends
// its exact on-wire bytes to out, returning the decoded header. Item delimiters are
// read as bare 8-byte headers (they carry no VR even under explicit VR).
func readAndCaptureElementHeader(br *boundedReader, ts TransferSyntax, out *bytes.Buffer) (elementHeader, []byte, error) {
	enc := encodingFor(ts)
	bo := enc.byteOrder

	tagBytes, err := br.readExact(4)
	if err != nil {
		return elementHeader{}, nil, midElementEOF(err)
	}
	tag := NewTag(bo.Uint16(tagBytes[0:2]), bo.Uint16(tagBytes[2:4]))

	// FFFE group tags (items, delimiters) never carry a VR, even under explicit VR.
	if tag.Group() == 0xFFFE {
		lenBytes, err := br.readExact(4)
		if err != nil {
			return elementHeader{}, nil, midElementEOF(err)
		}
		out.Write(tagBytes)
		out.Write(lenBytes)
		return elementHeader{tag: tag, length: bo.Uint32(lenBytes)}, nil, nil
	}

	if enc.implicitVR {
		lenBytes, err := br.readExact(4)
		if err != nil {
			return elementHeader{}, nil, midElementEOF(err)
		}
		out.Write(tagBytes)
		out.Write(lenBytes)
		return elementHeader{tag: tag, vr: dictVR(tag), length: bo.Uint32(lenBytes)}, nil, nil
	}

	vrBytes, err := br.readExact(2)
	if err != nil {
		return elementHeader{}, nil, midElementEOF(err)
	}
	vr := vrFromBytes(vrBytes)
	if vr.Is32BitLength() {
		rest, err := br.readExact(6)
		if err != nil {
			return elementHeader{}, nil, midElementEOF(err)
		}
		out.Write(tagBytes)
		out.Write(vrBytes)
		out.Write(rest)
		return elementHeader{tag: tag, vr: vr, length: bo.Uint32(rest[2:6])}, nil, nil
	}
	lenBytes, err := br.readExact(2)
	if err != nil {
		return elementHeader{}, nil, midElementEOF(err)
	}
	out.Write(tagBytes)
	out.Write(vrBytes)
	out.Write(lenBytes)
	return elementHeader{tag: tag, vr: vr, length: uint32(bo.Uint16(lenBytes))}, nil, nil
}

// captureBytes reads exactly n value bytes (bounds-checked, Codex DCM-004) and
// appends them to out. A short read is io.ErrUnexpectedEOF (Codex DCM-003).
func captureBytes(br *boundedReader, n uint32, out *bytes.Buffer, owner Tag) error {
	if n == 0 {
		return nil
	}
	if err := br.checkLen(n, owner); err != nil {
		return err
	}
	b, err := br.readN(n)
	if err != nil {
		return err
	}
	out.Write(b)
	return nil
}
