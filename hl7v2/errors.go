package hl7v2

import (
	"fmt"
	"io"
)

// ParseError reports malformed HL7 v2 input at a byte offset. Diagnostics name
// structure and identifiers — segment IDs, field positions, delimiter names —
// never field values, because HL7 fields routinely carry PHI (PRD §9.1).
type ParseError struct {
	Offset int    // byte offset into the source where the fault was detected
	Reason string // structural description, free of field values
	err    error  // optional wrapped sentinel (e.g. io.ErrUnexpectedEOF on truncation)
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("hl7v2: parse error at offset %d: %s", e.Offset, e.Reason)
}

// Unwrap exposes the wrapped sentinel so callers can match truncation with
// errors.Is(err, io.ErrUnexpectedEOF).
func (e *ParseError) Unwrap() error { return e.err }

// truncatedAt builds a ParseError marking a body that ended inside a segment.
// It wraps io.ErrUnexpectedEOF so errors.Is identifies truncation, while the
// message names only the structural position (PRD §9.2).
func truncatedAt(offset int, reason string) *ParseError {
	return &ParseError{Offset: offset, Reason: reason, err: io.ErrUnexpectedEOF}
}

// SegmentError reports a failed conversion from a generic Segment to a typed
// segment view: the wrong segment ID, or a composite datatype that does not
// parse. It names the segment and the structural reason, never a field value.
type SegmentError struct {
	Segment string // the three-character segment ID involved, e.g. "PID"
	Reason  string // structural description, free of field values
}

func (e *SegmentError) Error() string {
	return fmt.Sprintf("hl7v2: %s segment: %s", e.Segment, e.Reason)
}

// AccessorError reports a malformed accessor key or a path that descends past a
// leaf node. It names the offending key and the structural reason; the key is a
// 1-based path such as "PID-5-1-2", never a field value, so it carries no PHI.
type AccessorError struct {
	Key    string // the accessor key involved, e.g. "PID-5-1-2"
	Reason string // structural description, free of field values
}

func (e *AccessorError) Error() string {
	return fmt.Sprintf("hl7v2: accessor %q: %s", e.Key, e.Reason)
}
