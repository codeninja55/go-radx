package dicom

import (
	"bufio"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"os"
)

// ReadFile opens path and parses it as a Part 10 file.
func ReadFile(path string, opts ...ReadOption) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return Read(bufio.NewReader(file), opts...)
}

// Read parses a Part 10 stream. It honours the transfer syntax declared in the file
// meta for the main dataset; it never assumes Implicit VR LE (Codex DCM-002). For
// Deflated Explicit VR LE the main dataset is inflated before parsing.
func Read(r io.Reader, opts ...ReadOption) (*File, error) {
	cfg := newReadConfig(opts...)
	br := newBoundedReader(r, cfg.maxElementLen)

	preamble, err := readPreamble(br)
	if err != nil {
		return nil, err
	}
	meta, err := readFileMeta(br)
	if err != nil {
		return nil, err
	}

	ts := meta.TransferSyntaxUID
	if err := checkReadableTransferSyntax(ts); err != nil {
		return nil, err
	}

	main := br
	if ts.IsDeflated() {
		// The main dataset follows the file-meta group as a raw DEFLATE stream. The
		// flate reader does not expose Len(), so the boundedReader's remaining-byte
		// guard cannot fire on this path; bound the total inflated bytes so a tiny
		// crafted stream cannot inflate without end (a decompression bomb).
		fr := flate.NewReader(br.r)
		defer func() { _ = fr.Close() }()
		main = newBoundedReader(newInflateLimitReader(fr, cfg.maxInflatedBytes), cfg.maxElementLen)
	}

	ds, err := readDataSet(main, ts, cfg)
	if err != nil {
		return nil, err
	}

	f := &File{Meta: meta, DataSet: ds}
	copy(f.Preamble[:], preamble)
	return f, nil
}

// WriteFile encodes f to path in f.Meta.TransferSyntaxUID.
func WriteFile(path string, f *File, opts ...WriteOption) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(out)
	if err := Write(bw, f, opts...); err != nil {
		_ = out.Close()
		return err
	}
	if err := bw.Flush(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// Write encodes f to w. The encoder is selected from the declared transfer syntax;
// an unsupported transfer syntax is rejected before any bytes are written (Codex
// DCM-002). For Deflated Explicit VR LE the main dataset is deflated after the
// file-meta group.
func Write(w io.Writer, f *File, opts ...WriteOption) error {
	_ = newWriteConfig(opts...)
	if f == nil || f.Meta == nil || f.DataSet == nil {
		return fmt.Errorf("dicom: Write requires a File with Meta and DataSet")
	}
	ts := f.Meta.TransferSyntaxUID
	if err := checkWritableTransferSyntax(ts); err != nil {
		return err
	}

	// Encode the main dataset to a buffer first so an encoding error surfaces before
	// any bytes reach w, and so the deflated path can compress the whole dataset.
	var mainBuf bytes.Buffer
	if err := writeDataSet(&mainBuf, f.DataSet, ts); err != nil {
		return err
	}

	if err := writeFileMeta(w, f.Preamble, f.Meta); err != nil {
		return err
	}

	if ts.IsDeflated() {
		fw, err := flate.NewWriter(w, flate.DefaultCompression)
		if err != nil {
			return err
		}
		if _, err := fw.Write(mainBuf.Bytes()); err != nil {
			return err
		}
		return fw.Close()
	}

	_, err := w.Write(mainBuf.Bytes())
	return err
}

// checkReadableTransferSyntax rejects a syntax whose main dataset go-radx cannot
// decode. v1 reads the four uncompressed syntaxes; the pixel data of an
// encapsulated syntax is handled by the pixel pipeline, not the dataset reader, so
// an encapsulated syntax has no readable main-dataset encoding here.
func checkReadableTransferSyntax(ts TransferSyntax) error {
	if ts == "" {
		return fmt.Errorf("dicom: empty transfer syntax")
	}
	if ts.IsEncapsulated() {
		return fmt.Errorf("dicom: transfer syntax %s (%s) is encapsulated; v1 reads only the four uncompressed syntaxes",
			ts.Name(), string(ts))
	}
	return nil
}

// checkWritableTransferSyntax rejects any syntax other than the four uncompressed
// ones; the main dataset is never written compressed (Codex DCM-002).
func checkWritableTransferSyntax(ts TransferSyntax) error {
	switch ts {
	case ImplicitVRLittleEndian, ExplicitVRLittleEndian,
		DeflatedExplicitVRLittleEndian, ExplicitVRBigEndian:
		return nil
	case "":
		return fmt.Errorf("dicom: cannot write with an empty transfer syntax")
	default:
		return fmt.Errorf("dicom: transfer syntax %s (%s) is not writable; v1 writes only the four uncompressed syntaxes",
			ts.Name(), string(ts))
	}
}
