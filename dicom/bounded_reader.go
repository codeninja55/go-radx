package dicom

import (
	"encoding/binary"
	"errors"
	"io"
)

// defaultMaxElementLen caps a single element's value field when the caller does
// not override it with WithMaxElementLen. It is generous enough for native pixel
// data on a typical study but small enough that a hostile multi-gigabyte length is
// rejected before allocation (Codex DCM-004). The undefined-length sentinel
// 0xFFFFFFFF is never an allocation size.
const defaultMaxElementLen uint32 = 256 << 20 // 256 MiB

// undefinedLength is the 32-bit sentinel for an undefined value length (SQ and
// encapsulated pixel data). It is never used as an allocation size.
const undefinedLength uint32 = 0xFFFFFFFF

// lener is the optional interface (satisfied by bytes.Reader and strings.Reader)
// that lets the bounded reader know how many bytes remain without consuming them,
// so a length larger than the bytes actually present is rejected before any
// make([]byte, n).
type lener interface{ Len() int }

// boundedReader is the single read path for Part 10 parsing. Every length read
// from the wire passes checkLen before any allocation: it is validated against the
// configured per-element cap and, when the source size is known, against the bytes
// actually remaining. It tracks a byte offset for diagnostics.
type boundedReader struct {
	r      io.Reader
	ln     lener  // non-nil when remaining bytes are knowable
	maxLen uint32 // per-element value-field cap
	pos    int64  // bytes consumed so far
}

// newBoundedReader wraps r with a per-element length cap. If r exposes Len() (as
// bytes.Reader and strings.Reader do) the remaining-byte bound is enforced too.
func newBoundedReader(r io.Reader, maxLen uint32) *boundedReader {
	br := &boundedReader{r: r, maxLen: maxLen}
	if l, ok := r.(lener); ok {
		br.ln = l
	}
	return br
}

// offset returns the number of bytes consumed so far.
func (br *boundedReader) offset() int64 { return br.pos }

// remaining reports the bytes left in the source and whether that is knowable.
func (br *boundedReader) remaining() (int64, bool) {
	if br.ln == nil {
		return 0, false
	}
	return int64(br.ln.Len()), true
}

// checkLen rejects a value length before allocation. It rejects the undefined
// sentinel (which is not a size), lengths over the configured cap, and lengths
// larger than the bytes actually remaining when that is knowable. tag is included
// in the error for diagnostics (0 if unknown).
func (br *boundedReader) checkLen(n uint32, tag Tag) error {
	if n == undefinedLength {
		return &LimitExceededError{Tag: tag, Limit: uint64(br.maxLen), Actual: uint64(n), Kind: "element-length"}
	}
	if n > br.maxLen {
		return &LimitExceededError{Tag: tag, Limit: uint64(br.maxLen), Actual: uint64(n), Kind: "element-length"}
	}
	if rem, ok := br.remaining(); ok && int64(n) > rem {
		// A declared length larger than the bytes actually present is a truncated
		// object, not a policy violation: surface the same io.ErrUnexpectedEOF that
		// io.ReadFull would, but before allocating (Codex DCM-003). The configured
		// cap check above is the distinct guard for a hostile oversized declared
		// length (Codex DCM-004).
		return io.ErrUnexpectedEOF
	}
	return nil
}

// readN reads exactly n bytes after bounds-checking n against the cap and the
// bytes remaining. A short read inside the n bytes is io.ErrUnexpectedEOF, never a
// truncated success (Codex DCM-003, DCM-004).
func (br *boundedReader) readN(n uint32) ([]byte, error) {
	if err := br.checkLen(n, 0); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	buf := make([]byte, n)
	got, err := io.ReadFull(br.r, buf)
	br.pos += int64(got)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return buf, nil
}

// readExact reads exactly n bytes that are NOT subject to the element cap (used for
// fixed-size headers like the preamble and element headers). A short read is
// io.ErrUnexpectedEOF; a zero-byte read at the very start returns io.EOF so a clean
// boundary is distinguishable.
func (br *boundedReader) readExact(n int) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	buf := make([]byte, n)
	got, err := io.ReadFull(br.r, buf)
	br.pos += int64(got)
	if err != nil {
		if errors.Is(err, io.EOF) && got == 0 {
			return nil, io.EOF // clean boundary: nothing was consumed
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, io.ErrUnexpectedEOF // partial header is a truncation
		}
		return nil, err
	}
	return buf, nil
}

// readTag reads a 4-byte (group, element) tag in the given byte order. A clean EOF
// before any byte is consumed is io.EOF (end of the element loop); a partial tag is
// io.ErrUnexpectedEOF.
func (br *boundedReader) readTag(bo binary.ByteOrder) (Tag, error) {
	b, err := br.readExact(4)
	if err != nil {
		return 0, err
	}
	group := bo.Uint16(b[0:2])
	element := bo.Uint16(b[2:4])
	return NewTag(group, element), nil
}

// binaryLittleEndian is a small helper so tests and call sites read clearly.
func binaryLittleEndian() binary.ByteOrder { return binary.LittleEndian }

// inflateLimitReader bounds the total number of bytes that may be inflated from a
// DEFLATE stream. The Deflated Explicit VR LE main dataset arrives through
// flate.NewReader, whose reader does not expose Len(), so the boundedReader's
// remaining-byte guard cannot fire on that path; the only other bound is the
// per-element cap, which a tiny crafted stream that inflates into a long run of
// small valid elements never trips. This reader is the total-inflated-bytes guard
// for the deflated path: once the budget is exhausted it returns a typed
// *LimitExceededError instead of letting the element loop spin (a decompression
// bomb). It reads at most budget+1 bytes from the source, so a hostile stream
// cannot drive an unbounded allocation before the bound fires.
type inflateLimitReader struct {
	r      io.Reader
	budget int64 // total inflated bytes still permitted
	read   int64 // total inflated bytes delivered so far
}

// newInflateLimitReader bounds r to at most budget inflated bytes. A budget of zero
// or less means unbounded, but callers thread a positive budget so the guard
// applies.
func newInflateLimitReader(r io.Reader, budget int64) *inflateLimitReader {
	return &inflateLimitReader{r: r, budget: budget}
}

// Read delivers inflated bytes up to the budget. When the source would exceed the
// budget it fills the remaining allowance, then on the next read (or immediately if
// the allowance is already zero) returns *LimitExceededError so an over-large
// inflated stream fails promptly rather than spinning the element loop.
func (lr *inflateLimitReader) Read(p []byte) (int, error) {
	if lr.budget <= 0 {
		return lr.r.Read(p) // unbounded
	}
	if len(p) == 0 {
		return 0, nil
	}
	if lr.read >= lr.budget {
		// The budget is fully delivered. The dataset parser still issues one more read to
		// observe EOF, so probe a single byte: a clean EOF exactly at the budget must pass
		// through (a valid stream whose inflated size equals the cap), while a real byte
		// beyond the budget is the decompression bomb this guard exists to stop.
		var probe [1]byte
		n, err := lr.r.Read(probe[:])
		if n > 0 {
			return 0, &LimitExceededError{
				Limit:  uint64(lr.budget),
				Actual: uint64(lr.budget) + 1,
				Kind:   "inflated-bytes",
			}
		}
		return 0, err
	}
	remaining := lr.budget - lr.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := lr.r.Read(p)
	lr.read += int64(n)
	return n, err
}
