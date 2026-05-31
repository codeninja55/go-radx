package dicom

import "io"

// readDataSet reads elements in ts's encoding until a clean EOF at a top-level tag
// boundary. EOF inside any element header or value is a truncation, surfaced as
// io.ErrUnexpectedEOF, never a graceful end (Codex DCM-003). The element loop is
// the extension point Increment 3 hooks structured SQ parsing into.
func readDataSet(br *boundedReader, ts TransferSyntax, cfg readConfig) (*DataSet, error) {
	ds := NewDataSet()
	for {
		h, err := readElementHeader(br, ts)
		if err != nil {
			if isEOF(err) {
				return ds, nil // clean boundary: the dataset is complete
			}
			return nil, err
		}

		if cfg.stopAtPixelData && h.tag == TagPixelData {
			return ds, nil
		}

		var v Value
		if h.vr == VRSQ || h.length == undefinedLength {
			// An SQ (by VR) or any undefined-length value is a sequence delimited by a
			// Sequence Delimitation Item, parsed structurally into nested datasets so it
			// is never dropped (Codex DCM-005). Encapsulated pixel data under a
			// compressed syntax is handled by the pixel pipeline (Increment 6), not here;
			// v1 reads only uncompressed syntaxes whose undefined-length values are SQ.
			seq, err := decodeSequence(br, elementHeader{tag: h.tag, vr: VRSQ, length: h.length}, ts, cfg, 1)
			if err != nil {
				return nil, err
			}
			v = &sequenceValue{seq: seq}
			h.vr = VRSQ
		} else {
			v, err = decodeValue(br, h, encodingFor(ts))
			if err != nil {
				return nil, err
			}
		}
		ds.Set(Element{Tag: h.tag, VR: h.vr, Value: v})
	}
}

// writeDataSet writes ds's elements in ascending tag order in ts's encoding. Each
// element is header-then-value with the value length taken from the encoder so a
// padded character field is counted correctly.
func writeDataSet(w io.Writer, ds *DataSet, ts TransferSyntax) error {
	enc := encodingFor(ts)
	for e := range ds.All() {
		if e.Value == nil {
			return &ValueError{Tag: e.Tag, VR: e.VR, Msg: "element has no value"}
		}

		if sv, ok := e.Value.(*sequenceValue); ok {
			if err := writeSequenceElement(w, e.Tag, sv.seq, ts); err != nil {
				return err
			}
			continue
		}

		n := e.Value.EncodedLen(enc.byteOrder)
		if err := writeElementHeader(w, elementHeader{tag: e.Tag, vr: e.VR, length: n}, ts); err != nil {
			return err
		}
		written, err := encodeValue(w, e.Value, enc)
		if err != nil {
			return err
		}
		if written != n {
			return &ValueError{Tag: e.Tag, VR: e.VR, Msg: "encoded value length disagrees with EncodedLen"}
		}
	}
	return nil
}

// writeSequenceElement writes an SQ element: its header with the recorded length
// form (the 0xFFFFFFFF sentinel for an undefined-length sequence, the exact item
// byte count otherwise) followed by the structured item body. The length form is the
// one the sequence was read with, so a sequence-bearing dataset round-trips
// byte-identically (PS3.5 §7.5; Codex DCM-005).
func writeSequenceElement(w io.Writer, tag Tag, seq *Sequence, ts TransferSyntax) error {
	headerLen := undefinedLength
	if !seq.undefinedLength {
		n, err := sequenceEncodedLen(seq, ts)
		if err != nil {
			return err
		}
		headerLen = n
	}
	if err := writeElementHeader(w, elementHeader{tag: tag, vr: VRSQ, length: headerLen}, ts); err != nil {
		return err
	}
	_, err := encodeSequenceValue(w, seq, ts)
	return err
}
