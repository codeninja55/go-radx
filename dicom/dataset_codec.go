package dicom

import "io"

// readDataSet reads elements in ts's encoding until a clean EOF at a top-level tag
// boundary. EOF inside any element header or value is a truncation, surfaced as
// io.ErrUnexpectedEOF, never a graceful end (Codex DCM-003). The element loop is
// the extension point Increment 3 hooks structured SQ parsing into.
//
// The active Specific Character Set flows through the parse, not a global (PRD §9.4):
// it starts from cfg.defaultCharSet and is replaced when (0008,0005) is read, so the
// low-tagged charset element governs the customisable text VRs that follow it.
//
// The active Specific Character Set lives in cfg (a value type) and is swapped in
// place when (0008,0005) is read, so the low-tagged charset element governs the
// customisable text VRs that follow it without a global (PRD §9.4; Codex DCM-011).
func readDataSet(br *boundedReader, ts TransferSyntax, cfg readConfig) (*DataSet, error) {
	if cfg.activeCharset == nil {
		cs, err := NewSpecificCharacterSet(cfg.defaultCharSet...)
		if err != nil {
			return nil, err
		}
		cfg.activeCharset = cs
	}

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
		if h.tag == TagPixelData && h.length == undefinedLength && ts.IsEncapsulated() {
			// Encapsulated pixel data: retain the verbatim fragment stream on the
			// dataset, byte-for-byte and undecoded, so metadata of a compressed file is
			// reachable and a round-trip re-emits an identical pixel stream. Decoding
			// stays in the PixelData/Frames pipeline (Codex DCM-006).
			stream, err := readEncapsulatedValue(br, ts)
			if err != nil {
				return nil, err
			}
			v = &encapsulatedValue{stream: stream}
		} else if h.vr == VRSQ || h.length == undefinedLength {
			// An SQ (by VR) or any other undefined-length value is a sequence delimited
			// by a Sequence Delimitation Item, parsed structurally into nested datasets
			// so it is never dropped (Codex DCM-005).
			seq, err := decodeSequence(br, elementHeader{tag: h.tag, vr: VRSQ, length: h.length}, ts, cfg, 1)
			if err != nil {
				return nil, err
			}
			v = &sequenceValue{seq: seq}
			h.vr = VRSQ
		} else {
			v, err = decodeValue(br, h, encodingFor(ts), cfg.activeCharset)
			if err != nil {
				return nil, err
			}
		}
		ds.Set(Element{Tag: h.tag, VR: h.vr, Value: v})

		if h.tag == TagSpecificCharacterSet {
			cs, err := charsetFromElement(v)
			if err != nil {
				return nil, err
			}
			cfg.activeCharset = cs
		}
	}
}

// charsetFromElement resolves a freshly read (0008,0005) value into the active
// character set. The element is a default-repertoire CS, so its values are the
// defined terms verbatim.
func charsetFromElement(v Value) (*SpecificCharacterSet, error) {
	sv, ok := v.(*Strings)
	if !ok {
		return NewSpecificCharacterSet()
	}
	return NewSpecificCharacterSet(sv.Strings()...)
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

		if ev, ok := e.Value.(*encapsulatedValue); ok {
			if err := writeEncapsulatedElement(w, e, ev, ts); err != nil {
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

// writeEncapsulatedElement writes an encapsulated (7FE0,0010) element: the
// undefined-length header followed by the verbatim fragment stream, which already
// ends with its Sequence Delimitation Item. Only an encapsulated transfer syntax
// may carry it — under an uncompressed syntax pixel data must be native (PS3.5
// A.4) — so a dataset still holding fragments fails closed rather than emitting a
// non-conformant stream; re-encode it first (Transcode + File.SetPixelData).
func writeEncapsulatedElement(w io.Writer, e Element, ev *encapsulatedValue, ts TransferSyntax) error {
	if !ts.IsEncapsulated() {
		return &ValueError{
			Tag: e.Tag, VR: e.VR,
			Msg: "encapsulated pixel data cannot be written under an uncompressed transfer syntax; transcode it first",
		}
	}
	if err := writeElementHeader(w, elementHeader{tag: e.Tag, vr: e.VR, length: undefinedLength}, ts); err != nil {
		return err
	}
	_, err := w.Write(ev.stream)
	return err
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
