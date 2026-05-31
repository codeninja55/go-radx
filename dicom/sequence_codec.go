package dicom

// Delimiter tags shared by sequences, items, and encapsulated pixel data (PS3.5
// §7.5). They carry no VR and always use the 4-byte length form in the transfer
// syntax's byte order.
var (
	tagItem            = NewTag(0xFFFE, 0xE000)
	tagItemDelimiter   = NewTag(0xFFFE, 0xE00D)
	tagSequenceDelimit = NewTag(0xFFFE, 0xE0DD)
)

// decodeSequence parses an SQ value field into a structured Sequence. h is the SQ
// element's header: h.length is either a defined byte count or the undefinedLength
// sentinel. depth is the current nesting level; each nested sequence increments it
// and a level beyond cfg.maxSequenceDepth is a LimitExceededError before any further
// recursion, so a maliciously deep object cannot overflow the stack (PS3.5 §7.5;
// Codex DCM-005). Every length read from the wire is bounds-checked through br
// before allocation (Codex DCM-003/DCM-004).
func decodeSequence(br *boundedReader, h elementHeader, ts TransferSyntax, cfg readConfig, depth int) (*Sequence, error) {
	if depth > cfg.maxSequenceDepth {
		return nil, &LimitExceededError{
			Tag:    h.tag,
			Limit:  uint64(cfg.maxSequenceDepth),
			Actual: uint64(depth),
			Kind:   "sequence-depth",
		}
	}

	if h.length == undefinedLength {
		return decodeUndefinedLengthSequence(br, h.tag, ts, cfg, depth)
	}
	return decodeDefinedLengthSequence(br, h, ts, cfg, depth)
}

// decodeUndefinedLengthSequence reads items until the Sequence Delimitation Item
// (FFFE,E0DD). A stream that ends before the delimiter is a truncation
// (io.ErrUnexpectedEOF), never a graceful end (Codex DCM-003).
func decodeUndefinedLengthSequence(br *boundedReader, owner Tag, ts TransferSyntax, cfg readConfig, depth int) (*Sequence, error) {
	seq := &Sequence{undefinedLength: true}
	for {
		tag, length, err := readDelimiterHeader(br, ts)
		if err != nil {
			return nil, err
		}
		switch tag {
		case tagSequenceDelimit:
			return seq, nil
		case tagItem:
			item, err := decodeItem(br, owner, length, ts, cfg, depth)
			if err != nil {
				return nil, err
			}
			seq.items = append(seq.items, item)
		default:
			return nil, &ValueError{Tag: owner, VR: VRSQ, Msg: "unexpected tag inside undefined-length sequence"}
		}
	}
}

// decodeDefinedLengthSequence reads items from exactly h.length value bytes. The
// length is validated against the bytes remaining before parsing (Codex DCM-004); a
// short region is a truncation.
func decodeDefinedLengthSequence(br *boundedReader, h elementHeader, ts TransferSyntax, cfg readConfig, depth int) (*Sequence, error) {
	if err := br.checkLen(h.length, h.tag); err != nil {
		return nil, err
	}
	seq := &Sequence{undefinedLength: false}
	end := br.offset() + int64(h.length)
	for br.offset() < end {
		tag, length, err := readDelimiterHeader(br, ts)
		if err != nil {
			return nil, err
		}
		if tag != tagItem {
			return nil, &ValueError{Tag: h.tag, VR: VRSQ, Msg: "expected an item inside defined-length sequence"}
		}
		item, err := decodeItem(br, h.tag, length, ts, cfg, depth)
		if err != nil {
			return nil, err
		}
		seq.items = append(seq.items, item)
	}
	if br.offset() != end {
		return nil, &ValueError{Tag: h.tag, VR: VRSQ, Msg: "sequence items overran the declared length"}
	}
	return seq, nil
}

// decodeItem parses one item's content into a nested DataSet. length is the item
// header's length field: the undefinedLength sentinel (terminated by an Item
// Delimitation Item) or a defined byte count. owner is the enclosing sequence tag,
// used for diagnostics only.
func decodeItem(br *boundedReader, owner Tag, length uint32, ts TransferSyntax, cfg readConfig, depth int) (Item, error) {
	if length == undefinedLength {
		ds, err := decodeItemElements(br, ts, cfg, depth, -1)
		if err != nil {
			return Item{}, err
		}
		return Item{DataSet: ds, undefinedLength: true}, nil
	}
	if err := br.checkLen(length, owner); err != nil {
		return Item{}, err
	}
	end := br.offset() + int64(length)
	ds, err := decodeItemElements(br, ts, cfg, depth, end)
	if err != nil {
		return Item{}, err
	}
	if br.offset() != end {
		return Item{}, &ValueError{Tag: owner, VR: VRSQ, Msg: "item elements overran the declared length"}
	}
	return Item{DataSet: ds, undefinedLength: false}, nil
}

// decodeItemElements reads elements into a DataSet. When end >= 0 it reads until the
// byte offset reaches end (defined-length item); when end < 0 it reads until the
// Item Delimitation Item (FFFE,E00D) (undefined-length item). A nested SQ element
// recurses through decodeSequence at depth+1.
func decodeItemElements(br *boundedReader, ts TransferSyntax, cfg readConfig, depth int, end int64) (*DataSet, error) {
	ds := NewDataSet()
	for {
		if end >= 0 && br.offset() >= end {
			return ds, nil
		}

		h, err := readItemElementHeader(br, ts)
		if err != nil {
			// Inside an item, a clean EOF is a truncation: the item or sequence was
			// not closed (Codex DCM-003).
			return nil, midElementEOF(err)
		}

		if h.tag == tagItemDelimiter {
			if end >= 0 {
				return nil, &ValueError{Tag: tagItemDelimiter, VR: VRSQ, Msg: "item delimiter inside defined-length item"}
			}
			return ds, nil // undefined-length item closed
		}

		v, err := decodeElementValue(br, h, ts, cfg, depth)
		if err != nil {
			return nil, err
		}
		ds.Set(Element{Tag: h.tag, VR: h.vr, Value: v})
	}
}

// decodeElementValue decodes one element's value, recursing into a structured
// Sequence for an SQ (by VR) or for any undefined-length value (an implicit-VR SQ).
// depth is the current sequence nesting level.
func decodeElementValue(br *boundedReader, h elementHeader, ts TransferSyntax, cfg readConfig, depth int) (Value, error) {
	if h.vr == VRSQ || h.length == undefinedLength {
		seq, err := decodeSequence(br, elementHeader{tag: h.tag, vr: VRSQ, length: h.length}, ts, cfg, depth+1)
		if err != nil {
			return nil, err
		}
		return &sequenceValue{seq: seq}, nil
	}
	return decodeValue(br, h, encodingFor(ts))
}

// readDelimiterHeader reads a bare 8-byte item/delimiter header (4-byte tag + 4-byte
// length, no VR) in ts's byte order. A short read is io.ErrUnexpectedEOF.
func readDelimiterHeader(br *boundedReader, ts TransferSyntax) (Tag, uint32, error) {
	bo := encodingFor(ts).byteOrder
	b, err := br.readExact(8)
	if err != nil {
		return 0, 0, midElementEOF(err)
	}
	tag := NewTag(bo.Uint16(b[0:2]), bo.Uint16(b[2:4]))
	length := bo.Uint32(b[4:8])
	return tag, length, nil
}

// readItemElementHeader reads one element header inside an item. A group-0xFFFE tag
// (the Item Delimitation Item) carries no VR even under explicit VR and uses the
// bare 4-byte length form (PS3.5 §7.5), so it is read as an 8-byte header; any other
// tag is a normal data element read in ts's encoding.
func readItemElementHeader(br *boundedReader, ts TransferSyntax) (elementHeader, error) {
	enc := encodingFor(ts)
	tagBytes, err := br.readExact(4)
	if err != nil {
		return elementHeader{}, err
	}
	tag := NewTag(enc.byteOrder.Uint16(tagBytes[0:2]), enc.byteOrder.Uint16(tagBytes[2:4]))

	if tag.Group() == 0xFFFE {
		lenBytes, err := br.readExact(4)
		if err != nil {
			return elementHeader{}, midElementEOF(err)
		}
		return elementHeader{tag: tag, length: enc.byteOrder.Uint32(lenBytes)}, nil
	}

	return readElementHeaderBody(br, tag, enc)
}
