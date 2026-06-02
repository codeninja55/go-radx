package dicomweb

import (
	"errors"
	"fmt"

	"github.com/codeninja55/go-radx/dicom"
)

// Sentinel errors for the hostile-input caps (PRD §9.3) and unsupported features.
// Check with errors.Is. They name structure, never PHI (PRD §9.1).
var (
	// ErrLimitExceeded is returned when a body, part-count, or per-part-size cap is hit.
	ErrLimitExceeded = errors.New("dicomweb: input limit exceeded")
	// ErrNotAcceptable is returned when content negotiation fails (HTTP 406).
	ErrNotAcceptable = errors.New("dicomweb: no acceptable representation")
	// ErrUnsupported is returned for a service or media type not supported in v1
	// (the 501/deferred surface).
	ErrUnsupported = errors.New("dicomweb: service or media type not supported in v1")
	// ErrInvalidResource is returned for an invalid resource path or UID.
	ErrInvalidResource = errors.New("dicomweb: invalid resource path or UID")
)

// LimitExceededError reports a count, size, or depth cap hit before any allocation
// (PRD §9.3). It unwraps to ErrLimitExceeded so callers can check with errors.Is, and
// names the offending structure, never a patient value (PRD §9.1).
type LimitExceededError struct {
	Limit  uint64 // the configured bound
	Actual uint64 // the value that exceeded it
	Kind   string // "multipart-part-count", "multipart-part-bytes", "json-sequence-depth", ...
}

func (e *LimitExceededError) Error() string {
	return fmt.Sprintf("dicomweb: %s limit exceeded: %d > %d", e.Kind, e.Actual, e.Limit)
}

// Unwrap ties the typed error to the ErrLimitExceeded sentinel.
func (e *LimitExceededError) Unwrap() error { return ErrLimitExceeded }

// TruncatedError reports input that ended mid-part or mid-value: a transfer that ends
// before its declared structure completes is failure, never a graceful end (PRD §9.2).
// It wraps io.ErrUnexpectedEOF so callers can check with errors.Is.
type TruncatedError struct {
	// Detail names where the truncation was detected without any patient value.
	Detail string
	err    error
}

func (e *TruncatedError) Error() string {
	return fmt.Sprintf("dicomweb: truncated input: %s", e.Detail)
}

// Unwrap exposes the wrapped io.ErrUnexpectedEOF.
func (e *TruncatedError) Unwrap() error { return e.err }

// MalformedPartError reports a multipart/related body the parser could not frame: a bad
// boundary, a malformed part header, or other structural fault. It carries only a
// structural detail, never the raw offending bytes, which could contain PHI (PRD §9.1).
type MalformedPartError struct {
	// Detail names the structural fault without any raw input bytes.
	Detail string
}

func (e *MalformedPartError) Error() string {
	return "dicomweb: " + e.Detail
}

// EncodeError reports a value that cannot be rendered to DICOM JSON. It names the tag
// keyword and VR, never the offending value (PRD §9.1).
type EncodeError struct {
	Tag dicom.Tag
	VR  dicom.VR
	Msg string
}

func (e *EncodeError) Error() string {
	if e.Tag != 0 {
		return fmt.Sprintf("dicomweb: cannot encode %s %s (%s): %s", keywordOf(e.Tag), e.Tag, e.VR, e.Msg)
	}
	return fmt.Sprintf("dicomweb: cannot encode %s value: %s", e.VR, e.Msg)
}

// DecodeError reports a DICOM-JSON document that does not conform to PS3.18 Annex F. It
// names the tag and VR without any patient value (PRD §9.1).
type DecodeError struct {
	Tag dicom.Tag
	VR  dicom.VR
	Msg string
}

func (e *DecodeError) Error() string {
	if e.Tag != 0 {
		return fmt.Sprintf("dicomweb: cannot decode %s %s: %s", keywordOf(e.Tag), e.Tag, e.Msg)
	}
	return fmt.Sprintf("dicomweb: cannot decode DICOM JSON: %s", e.Msg)
}

// keywordOf renders a tag's dictionary keyword for diagnostics, falling back to the
// empty string for unknown tags. It never returns a patient value.
func keywordOf(t dicom.Tag) string {
	if info, ok := dicom.Lookup(t); ok {
		return info.Keyword
	}
	return ""
}
