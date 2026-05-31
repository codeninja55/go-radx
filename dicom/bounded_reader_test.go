package dicom

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestBoundedReaderReadFull(t *testing.T) {
	br := newBoundedReader(bytes.NewReader([]byte("hello world")), 64)
	got, err := br.readN(5)
	if err != nil {
		t.Fatalf("readN(5): %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("readN(5) = %q, want hello", got)
	}
	if br.offset() != 5 {
		t.Errorf("offset() = %d, want 5", br.offset())
	}
}

func TestBoundedReaderTruncationIsUnexpectedEOF(t *testing.T) {
	// Asking for more bytes than the stream holds is io.ErrUnexpectedEOF, never a
	// short success (Codex DCM-003).
	br := newBoundedReader(strings.NewReader("abc"), 64)
	_, err := br.readN(8)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("readN past end = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestBoundedReaderHostileLengthRejectedBeforeAlloc(t *testing.T) {
	// A 3 GB length must be rejected against the configured cap before any
	// allocation (Codex DCM-004): no make([]byte, n) for a hostile n.
	const cap = 1 << 20 // 1 MiB cap
	br := newBoundedReader(strings.NewReader("abc"), cap)
	const hostile = uint32(3 * 1024 * 1024 * 1024 >> 0) // wraps; use a value over cap
	_, err := br.readN(0xFFFFFFFE)
	var le *LimitExceededError
	if !errors.As(err, &le) {
		t.Fatalf("readN(huge) = %v, want *LimitExceededError", err)
	}
	if le.Kind != "element-length" {
		t.Errorf("Kind = %q, want element-length", le.Kind)
	}
	if le.Limit != cap {
		t.Errorf("Limit = %d, want %d", le.Limit, cap)
	}
}

func TestBoundedReaderCheckLengthAgainstRemaining(t *testing.T) {
	// Even under a generous cap, a length larger than the bytes actually present
	// is rejected before allocation, not as a short read after a giant make.
	br := newBoundedReader(strings.NewReader("abcd"), 1<<30)
	if err := br.checkLen(1000, 0); err == nil {
		t.Error("checkLen for 1000 bytes over a 4-byte stream should fail")
	}
	if err := br.checkLen(4, 0); err != nil {
		t.Errorf("checkLen for 4 bytes over a 4-byte stream should pass, got %v", err)
	}
}

func TestBoundedReaderCleanEOFAtBoundary(t *testing.T) {
	// A zero-length read at a clean boundary returns io.EOF so the caller can end
	// the element loop, distinct from io.ErrUnexpectedEOF mid-element.
	br := newBoundedReader(strings.NewReader(""), 64)
	_, err := br.readTag(binaryLittleEndian())
	if !errors.Is(err, io.EOF) {
		t.Errorf("readTag at clean EOF = %v, want io.EOF", err)
	}
}

func TestBoundedReaderPartialTagIsUnexpectedEOF(t *testing.T) {
	// Two bytes into a four-byte tag is a truncation, not a clean boundary.
	br := newBoundedReader(strings.NewReader("\x10\x00"), 64)
	_, err := br.readTag(binaryLittleEndian())
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("readTag with 2 bytes = %v, want io.ErrUnexpectedEOF", err)
	}
}
