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

		v, err := decodeValue(br, h, encodingFor(ts))
		if err != nil {
			return nil, err
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
