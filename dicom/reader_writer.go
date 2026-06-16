package dicom

import (
	"bufio"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ReadFile opens path and parses it as a Part 10 file. It is the only entry point
// that supports WithDeferredValues: the path is recorded on each deferred value so
// it can be re-opened and loaded on demand.
func ReadFile(path string, opts ...ReadOption) (*File, error) {
	file, err := os.Open(path) // #nosec G304 -- reading a caller-supplied path is this API's contract
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	// Record the absolute path so a deferred value re-opens the same file even if the
	// working directory changes between read and load; a relative path would resolve
	// against the cwd at load time, which could name a different file.
	source := path
	if abs, err := filepath.Abs(path); err == nil {
		source = abs
	}
	return read(bufio.NewReader(file), source, opts...)
}

// Read parses a Part 10 stream. It honours the transfer syntax declared in the file
// meta for the main dataset; it never assumes Implicit VR LE (Codex DCM-002). For
// Deflated Explicit VR LE the main dataset is inflated before parsing. Read rejects
// WithDeferredValues fail-closed: a generic io.Reader cannot be re-read on demand,
// so deferral is a ReadFile-only option.
func Read(r io.Reader, opts ...ReadOption) (*File, error) {
	return read(r, "", opts...)
}

// read is the shared Part 10 parse. sourcePath is the re-openable file deferred
// values load from; empty means the source cannot be re-read, so deferral is
// rejected fail-closed rather than recording values that could never load.
func read(r io.Reader, sourcePath string, opts ...ReadOption) (*File, error) {
	cfg := newReadConfig(opts...)
	cfg.deferPath = sourcePath
	if cfg.deferralEnabled() && sourcePath == "" {
		return nil, errDeferralNeedsPath
	}
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
		if cfg.deferralEnabled() {
			// A deferred value's offset would address the inflated stream, which has
			// no seekable counterpart in the source file, so the load could never
			// honour it; reject rather than record windows that cannot be re-read.
			return nil, errDeferralDeflated
		}
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
	if f != nil && f.DataSet != nil {
		// Materialise any deferred values before os.Create truncates path: a deferred
		// value may load from this same file (a ReadFile + WriteFile round-trip), and
		// truncating first would destroy its source. A failed load errors here, with
		// path still untouched.
		if err := f.DataSet.materialiseDeferred(); err != nil {
			return err
		}
	}
	out, err := os.Create(path) // #nosec G304 -- writing a caller-supplied path is this API's contract
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

// supportedEncapsulated reports whether ts is an encapsulated syntax whose Part 10
// layout go-radx reads and writes: the main dataset is Explicit VR Little Endian
// and (7FE0,0010) is an undefined-length fragment stream (PS3.5 A.4). Pixel decode
// stays a separate, codec-gated concern (Frames returns ErrCodecUnavailable when no
// codec is built in). An unknown or private syntax is NOT in this set: its dataset
// encoding cannot be assumed, so reads and writes of it stay rejected.
func supportedEncapsulated(ts TransferSyntax) bool {
	switch ts {
	case RLELossless,
		JPEGBaseline8Bit, JPEGExtended12Bit, JPEGLossless, JPEGLosslessSV1,
		JPEGLSLossless, JPEGLSNearLossless,
		JPEG2000Lossless, JPEG2000,
		HTJ2KLossless, HTJ2KLosslessRPCL, HTJ2K:
		return true
	default:
		return false
	}
}

// checkReadableTransferSyntax rejects a syntax whose main dataset go-radx cannot
// decode: the four uncompressed syntaxes and the recognised encapsulated syntaxes
// are readable; an unrecognised syntax has no assumable main-dataset encoding.
func checkReadableTransferSyntax(ts TransferSyntax) error {
	if ts == "" {
		return fmt.Errorf("dicom: empty transfer syntax")
	}
	if ts.IsEncapsulated() && !supportedEncapsulated(ts) {
		return fmt.Errorf("dicom: transfer syntax %s (%s) is not readable; the dataset encoding of an unrecognised syntax cannot be assumed",
			ts.Name(), string(ts))
	}
	return nil
}

// checkWritableTransferSyntax rejects any syntax other than the four uncompressed
// ones and the recognised encapsulated syntaxes. Under an encapsulated syntax the
// main dataset is still written Explicit VR LE; only the retained (7FE0,0010)
// fragment stream is compressed (PS3.5 A.4).
func checkWritableTransferSyntax(ts TransferSyntax) error {
	switch ts {
	case ImplicitVRLittleEndian, ExplicitVRLittleEndian,
		DeflatedExplicitVRLittleEndian, ExplicitVRBigEndian:
		return nil
	case "":
		return fmt.Errorf("dicom: cannot write with an empty transfer syntax")
	default:
		if supportedEncapsulated(ts) {
			return nil
		}
		return fmt.Errorf("dicom: transfer syntax %s (%s) is not writable; the dataset encoding of an unrecognised syntax cannot be assumed",
			ts.Name(), string(ts))
	}
}
