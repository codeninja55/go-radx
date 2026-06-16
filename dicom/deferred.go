package dicom

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync"
)

// errDeferralNeedsPath rejects WithDeferredValues on an entry point that has no
// re-openable source: a deferred value recorded from a generic io.Reader could
// never be loaded, so the read fails closed up front instead.
var errDeferralNeedsPath = errors.New(
	"dicom: WithDeferredValues requires a re-openable file source; use ReadFile (a generic io.Reader cannot be re-read on demand)")

// errDeferralDeflated rejects WithDeferredValues under Deflated Explicit VR LE:
// element offsets there address the inflated stream, which is not seekable in the
// source file, so a recorded window could never be re-read.
var errDeferralDeflated = errors.New(
	"dicom: WithDeferredValues cannot apply to Deflated Explicit VR Little Endian; the inflated stream is not seekable in the source file")

// DeferredLoadError reports a deferred value whose on-demand load failed: the
// source file is gone, shrank past the recorded byte window, or no longer parses
// as the value it held at read time. It names structure only, never patient values.
type DeferredLoadError struct {
	Tag    Tag
	Offset int64
	Length int64
	Err    error
}

func (e *DeferredLoadError) Error() string {
	return fmt.Sprintf("dicom: deferred load of %s %s (offset %d, length %d) failed: %v",
		keywordFor(e.Tag), e.Tag, e.Offset, e.Length, e.Err)
}

func (e *DeferredLoadError) Unwrap() error { return e.Err }

// DeferredValue is the placeholder a WithDeferredValues read records for an
// element value larger than the threshold: the value bytes were skipped, and only
// the source path, byte window, and decode context (VR, byte order, the character
// set active at the element's position) are held, so reading a large object keeps
// memory bounded. The first access re-opens the source file and decodes the value
// on demand; the result is cached, and concurrent first accesses are safe (the
// load runs exactly once).
//
// Source-lifetime contract: the recorded path must still name the same, unmodified
// file when the value loads. The window is re-validated against the file on load —
// a source that shrank, a window past EOF, or content that no longer parses is a
// typed *DeferredLoadError, never a panic or a silently wrong value. Dataset
// accessors (GetStrings, GetInt, ...) and the write path load deferred values
// transparently; call Load directly when the load error matters.
type DeferredValue struct {
	tag     Tag
	vr      VR
	path    string
	offset  int64
	length  int64
	maxLen  uint32 // the per-element cap the original read enforced
	enc     encoding
	charset *SpecificCharacterSet
	// encapTS is set when the recorded window is an encapsulated (7FE0,0010)
	// fragment stream; the load re-validates its item structure under this syntax.
	encapTS TransferSyntax

	once   sync.Once
	loaded Value
	err    error
}

func (v *DeferredValue) VR() VR { return v.vr }

// EncodedLen reports the recorded on-wire value-field length without loading the
// value (the undefined-length sentinel for a deferred encapsulated stream, which
// is delimited rather than counted). The write path materialises the value first
// and uses the materialised length, so an emitted header never disagrees with the
// bytes that follow it.
func (v *DeferredValue) EncodedLen(binary.ByteOrder) uint32 {
	if v.encapTS != "" {
		return undefinedLength
	}
	return uint32(v.length) // #nosec G115 -- recorded from a 32-bit length field on the read path
}

// Load returns the decoded value, reading it from the source file on first call.
// The load runs exactly once; every subsequent call (from any goroutine) returns
// the cached result.
func (v *DeferredValue) Load() (Value, error) {
	v.once.Do(func() { v.loaded, v.err = v.load() })
	return v.loaded, v.err
}

func (v *DeferredValue) load() (Value, error) {
	raw, err := v.readWindow()
	if err != nil {
		return nil, err
	}
	if v.encapTS != "" {
		// Re-parse the window through the same structural validator the retaining
		// read path uses, so a source mutated since the read fails typed here rather
		// than handing an unvalidated stream to the pixel pipeline.
		br := newBoundedReader(bytes.NewReader(raw), v.maxLen)
		stream, err := readEncapsulatedValue(br, v.encapTS)
		if err != nil {
			return nil, v.loadErr(err)
		}
		if rem, _ := br.remaining(); rem != 0 {
			return nil, v.loadErr(fmt.Errorf("%d trailing bytes after the fragment stream; the source changed since the read", rem))
		}
		return &encapsulatedValue{stream: stream}, nil
	}
	val, err := decodeValueBytes(v.vr, raw, v.enc, v.charset)
	if err != nil {
		return nil, v.loadErr(err)
	}
	return val, nil
}

// readWindow re-opens the source and reads the recorded byte window, re-validating
// the window against the file's current size first so a shrunk or swapped source
// is a typed error before any read.
func (v *DeferredValue) readWindow() ([]byte, error) {
	f, err := os.Open(v.path) // #nosec G304 -- re-opening the path ReadFile was called with is this type's contract
	if err != nil {
		return nil, v.loadErr(err)
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return nil, v.loadErr(err)
	}
	if v.offset < 0 || v.length < 0 || v.offset > st.Size() || v.length > st.Size()-v.offset {
		return nil, v.loadErr(fmt.Errorf("recorded window lies past the source's %d bytes; the file shrank or was replaced", st.Size()))
	}
	raw := make([]byte, v.length)
	if _, err := f.ReadAt(raw, v.offset); err != nil {
		return nil, v.loadErr(err)
	}
	return raw, nil
}

func (v *DeferredValue) loadErr(err error) error {
	return &DeferredLoadError{Tag: v.tag, Offset: v.offset, Length: v.length, Err: err}
}

// materialise resolves v to its decoded form, loading a deferred value from its
// source on demand. ok is false when a deferred load fails; callers that need the
// typed error use (*DeferredValue).Load directly.
func materialise(v Value) (Value, bool) {
	dv, isDeferred := v.(*DeferredValue)
	if !isDeferred {
		return v, true
	}
	loaded, err := dv.Load()
	if err != nil {
		return nil, false
	}
	return loaded, true
}

// materialiseDeferred loads every deferred value in the dataset and its nested
// sequence items in place, so the dataset no longer depends on any external source.
// WriteFile calls it before truncating the destination: a deferred value may load
// from the very path being written (a ReadFile + WriteFile round-trip over one file),
// and os.Create would destroy that source before the value could be read. A failed
// load surfaces as a typed *DeferredLoadError before any output byte is written, so a
// vanished source fails the write rather than corrupting the destination. With no
// deferred values present it is a cheap walk and a no-op.
func (ds *DataSet) materialiseDeferred() error {
	for e := range ds.All() {
		switch v := e.Value.(type) {
		case *DeferredValue:
			loaded, err := v.Load()
			if err != nil {
				return err
			}
			ds.Set(Element{Tag: e.Tag, VR: e.VR, Value: loaded})
		case *sequenceValue:
			for it := range v.seq.Items() {
				if err := it.DataSet.materialiseDeferred(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// deferElementValue records h's value as deferred: it bounds-checks the declared
// length exactly as the materialising path would, captures the value field's byte
// window and decode context, and skips the bytes without retaining them.
func deferElementValue(br *boundedReader, h elementHeader, ts TransferSyntax, cfg readConfig) (*DeferredValue, error) {
	off := br.offset()
	if err := br.discardN(h.length, h.tag); err != nil {
		return nil, err
	}
	return &DeferredValue{
		tag:     h.tag,
		vr:      h.vr,
		path:    cfg.deferPath,
		offset:  off,
		length:  int64(h.length),
		maxLen:  br.maxLen,
		enc:     encodingFor(ts),
		charset: cfg.activeCharset,
	}, nil
}

// deferEncapsulatedValue records an encapsulated (7FE0,0010) fragment stream as
// deferred: the items are scanned and structurally validated exactly as the
// retaining path validates them, but the bytes are discarded and only the stream's
// byte window (first item header through the Sequence Delimitation Item) is
// recorded. The load re-parses the window through the same validator.
func deferEncapsulatedValue(br *boundedReader, ts TransferSyntax, cfg readConfig) (*DeferredValue, error) {
	start := br.offset()
	if err := skimEncapsulatedValue(br, ts); err != nil {
		return nil, err
	}
	return &DeferredValue{
		tag:     TagPixelData,
		vr:      VROB,
		path:    cfg.deferPath,
		offset:  start,
		length:  br.offset() - start,
		maxLen:  br.maxLen,
		encapTS: ts,
	}, nil
}
