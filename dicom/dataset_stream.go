package dicom

import (
	"bufio"
	"compress/flate"
	"fmt"
	"io"
)

// EncodeDataSet writes ds as a bare element stream in ts: no Part 10 preamble, no file-meta
// group, just the dataset elements in ascending tag order (PS3.7 §6.3.1). It is the entry point
// the dimse and dicomweb layers use to carry a dataset over the wire, where the framing (PDV
// fragments for DIMSE, multipart parts for DICOMweb) and the file-meta group live outside the
// dataset proper. For Deflated Explicit VR LE the element stream is deflated. An unsupported
// (encapsulated or empty) transfer syntax is rejected before any bytes are written (Codex
// DCM-002).
func EncodeDataSet(w io.Writer, ds *DataSet, ts TransferSyntax) error {
	if ds == nil {
		return fmt.Errorf("dicom: cannot encode a nil dataset")
	}
	if err := checkWritableTransferSyntax(ts); err != nil {
		return err
	}
	if ts.IsDeflated() {
		fw, err := flate.NewWriter(w, flate.DefaultCompression)
		if err != nil {
			return err
		}
		if err := writeDataSet(fw, ds, ts); err != nil {
			_ = fw.Close()
			return err
		}
		return fw.Close()
	}
	return writeDataSet(w, ds, ts)
}

// DecodeDataSet reads a bare element stream from r in ts, the inverse of EncodeDataSet. A clean
// EOF at a top-level tag boundary ends the dataset; an EOF inside an element header or value is a
// truncation surfaced as io.ErrUnexpectedEOF, never a graceful end (Codex DCM-003). Every length
// read from the stream is validated against a bounded reader before allocation (PRD §9.3). An
// encapsulated syntax has no readable bare-dataset encoding and is rejected.
func DecodeDataSet(r io.Reader, ts TransferSyntax, opts ...ReadOption) (*DataSet, error) {
	if err := checkReadableTransferSyntax(ts); err != nil {
		return nil, err
	}
	cfg := newReadConfig(opts...)
	if ts.IsDeflated() {
		// The flate reader does not expose Len(), so the boundedReader's remaining-byte
		// guard cannot fire on this path; bound the total inflated bytes so a tiny
		// crafted stream cannot inflate without end (a decompression bomb).
		fr := flate.NewReader(bufio.NewReader(r))
		defer func() { _ = fr.Close() }()
		r = newInflateLimitReader(fr, cfg.maxInflatedBytes)
	}
	br := newBoundedReader(r, cfg.maxElementLen)
	return readDataSet(br, ts, cfg)
}
