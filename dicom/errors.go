package dicom

import (
	"errors"
	"fmt"
	"io"
)

// ErrTruncated wraps io.ErrUnexpectedEOF for input that ends inside an element,
// value, item, or fragment. The reader accepts EOF only at a clean top-level tag
// boundary; anything shorter is corruption, never a graceful end (Codex DCM-003).
var ErrTruncated = fmt.Errorf("dicom: truncated input: %w", io.ErrUnexpectedEOF)

// LimitExceededError is returned when a length, depth, or count cap is hit. A
// hostile or malformed length is rejected through this error before any allocation
// (Codex DCM-004); the message names structure, never patient values.
type LimitExceededError struct {
	Tag    Tag    // the offending element, if known
	Limit  uint64 // the configured or remaining bound
	Actual uint64 // the value that exceeded it
	Kind   string // "element-length", "sequence-depth", "remaining-bytes", ...
}

func (e *LimitExceededError) Error() string {
	if e.Tag != 0 {
		return fmt.Sprintf("dicom: %s limit exceeded at %s %s: %d > %d",
			e.Kind, keywordFor(e.Tag), e.Tag, e.Actual, e.Limit)
	}
	return fmt.Sprintf("dicom: %s limit exceeded: %d > %d", e.Kind, e.Actual, e.Limit)
}

// isEOF reports a clean end-of-stream (io.EOF) as distinct from a mid-element
// truncation.
func isEOF(err error) bool { return errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) }

// ValueError reports a value that does not conform to its VR (bad UID, bad date,
// odd binary length, over-long PN component). It names the tag and VR; it never
// carries the offending PHI value (PRD §9.1).
type ValueError struct {
	Tag Tag
	VR  VR
	Msg string
}

func (e *ValueError) Error() string {
	if e.Tag != 0 {
		return fmt.Sprintf("dicom: invalid %s at %s %s: %s", e.VR, keywordFor(e.Tag), e.Tag, e.Msg)
	}
	return fmt.Sprintf("dicom: invalid %s: %s", e.VR, e.Msg)
}

// keywordFor renders a tag's keyword for diagnostics; falls back to "" if unknown.
func keywordFor(t Tag) string {
	if info, ok := Lookup(t); ok {
		return info.Keyword
	}
	return ""
}
