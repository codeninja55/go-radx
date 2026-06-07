// Package exitcode maps the library's typed errors onto the radx exit-code taxonomy
// (docs/reference/cli.md "Exit-code taxonomy"). A single classification point lets every
// command surface a failure class an operator can branch on without scraping text, and keeps
// the honest-failure rules in one auditable place: an unclassified or unimplemented path is
// never laundered into success.
package exitcode

import (
	"errors"
	"io"
	"io/fs"
	"os"

	"github.com/alecthomas/kong"

	"github.com/codeninja55/go-radx/convert"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/hl7v2"
)

// The exit-code taxonomy (docs/reference/cli.md). Each command maps its typed errors onto
// these so an operator can distinguish failure classes without parsing human text.
const (
	// Success reports that all work completed with no failures.
	Success = 0
	// GeneralFailure is an unclassified runtime error or an unimplemented capability
	// (the fail-closed default — nothing reaches here by accident; it is the deliberate
	// floor for "not yet built" and "we have no better classification").
	GeneralFailure = 1
	// UsageError is a flag/argument fault: an unknown flag, a missing required argument,
	// mutually exclusive flags, or a bad enum value. It is a Kong parse failure.
	UsageError = 2
	// ParseError is a malformed or truncated DICOM, HL7 v2, or FHIR input, a failed
	// validation, or a requested feature the standard supports but this build cannot.
	ParseError = 3
	// NetworkError is a refused connection, a rejected or aborted association, a
	// non-success DIMSE terminal status, or an HTTP transport failure.
	NetworkError = 4
	// FileIOError is an input that cannot be read or an output that cannot be created:
	// a path error, a missing file, or a permission denial.
	FileIOError = 5
)

// NotImplementedError is the typed error a fail-closed stub returns: a capability that is
// committed surface but not yet built. It classifies to GeneralFailure (exit 1) so a stub
// errors and writes nothing rather than no-opping and reporting success (the fail-closed
// rule, docs/reference/cli.md "Honest-failure rules").
type NotImplementedError struct {
	// Capability names the command or feature that is not yet built, e.g. "find".
	Capability string
}

func (e *NotImplementedError) Error() string {
	return "radx: " + e.Capability + " is not implemented yet (committed surface; fails closed until built)"
}

// StatusError promotes a non-success DIMSE terminal Status to a Go error. The library treats
// a Failure-category Status as in-band data the caller inspects (the conversation succeeded;
// the peer answered and said no), so a command that wants a non-zero exit on a refused or
// failed operation wraps the Status in this error. It classifies to NetworkError (exit 4):
// the operation reached the peer and the peer reported it could not perform the work.
type StatusError struct {
	// Status is the non-success DIMSE status the peer returned, rendered by name (never
	// raw hex) through its String method, so the diagnostic names the class and meaning.
	Status dimse.Status
}

func (e *StatusError) Error() string {
	return "radx: peer returned a non-success DIMSE status: " + e.Status.String()
}

// UsageErr is a usage fault a command raises itself (a bad flag combination, a format not
// supported for this command) that Kong's grammar cannot express, so it is rejected at run
// time rather than parse time. It classifies to UsageError (exit 2), the same class as a Kong
// parse failure, so an operator sees a consistent code for "you invoked it wrongly".
type UsageErr struct {
	// Message names the violated rule without any patient value.
	Message string
}

func (e *UsageErr) Error() string { return "radx: " + e.Message }

// Classify maps err onto the exit-code taxonomy. A nil error is Success. The mapping checks
// the most specific classes first (typed parse/network/file errors), then the broad parse and
// network families, and falls through to GeneralFailure only for genuinely unclassified
// errors — the deliberate fail-closed floor, never a silent success.
func Classify(err error) int {
	if err == nil {
		return Success
	}

	// Usage errors: a Kong parse failure (unknown flag, missing required argument, bad enum)
	// or a command-raised usage fault (a format unsupported for that command). Both exit 2.
	var (
		parseErr *kong.ParseError
		usageErr *UsageErr
	)
	if errors.As(err, &parseErr) || errors.As(err, &usageErr) {
		return UsageError
	}

	// File-I/O faults take precedence over the parse family: a *.dcm that cannot be opened
	// is a file error (exit 5), not a DICOM parse error, even though it surfaces from a
	// reader. fs.ErrNotExist / fs.ErrPermission also cover *os.PathError, which wraps them.
	if isFileIOError(err) {
		return FileIOError
	}

	// Network / protocol faults: the conversation broke. Association rejects/aborts, the
	// DUL/ACSE/PDU protocol errors, a non-success DIMSE terminal status, a Storage
	// Commitment failure, and the DICOMweb HTTP/store transport failures (exit 4).
	if isNetworkError(err) {
		return NetworkError
	}

	// Parse / validation / unsupported-feature faults: malformed or truncated input, a
	// failed validation, or a standard feature this build cannot perform (exit 3).
	if isParseError(err) {
		return ParseError
	}

	// Unclassified or unimplemented — the fail-closed floor (exit 1).
	return GeneralFailure
}

// isFileIOError reports an input that cannot be read or an output that cannot be created. It
// matches *os.PathError directly and the fs sentinels it (and other I/O paths) wrap, so a
// permission denial or a missing file is exit 5 regardless of which library raised it.
func isFileIOError(err error) bool {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return true
	}
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission)
}

// isNetworkError reports a transport or protocol fault. It covers the public dimse error
// types and their internal acse/dul/pdu causes (a public *dimse error usually wraps one), a
// non-success DIMSE terminal status promoted to a *StatusError, the Storage Commitment
// failure, and the DICOMweb HTTP/store transport failures.
func isNetworkError(err error) bool {
	var (
		statusErr     *StatusError
		assocErr      *dimse.AssociationError
		abortErr      *dimse.AbortError
		protoErr      *dimse.ProtocolError
		commitErr     *dimse.CommitmentFailureError
		acseRejected  *acse.RejectedError
		acseAborted   *acse.AbortedError
		acseProtocol  *acse.ProtocolError
		dulStateErr   *dul.StateError
		pduErr        *pdu.PDUError
		httpErr       *dicomweb.HTTPError
		storeErr      *dicomweb.StoreError
		failReasonErr *dicomweb.FailureReasonError
		crossOrigin   *dicomweb.CrossOriginBulkDataError
	)
	switch {
	case errors.As(err, &statusErr),
		errors.As(err, &assocErr),
		errors.As(err, &abortErr),
		errors.As(err, &protoErr),
		errors.As(err, &commitErr),
		errors.As(err, &acseRejected),
		errors.As(err, &acseAborted),
		errors.As(err, &acseProtocol),
		errors.As(err, &dulStateErr),
		errors.As(err, &pduErr),
		errors.As(err, &httpErr),
		errors.As(err, &storeErr),
		errors.As(err, &failReasonErr),
		errors.As(err, &crossOrigin):
		return true
	}
	return errors.Is(err, dicomweb.ErrCrossOriginBulkData)
}

// isParseError reports malformed/truncated input, a failed validation, or an unsupported
// feature the standard defines but this build cannot perform. The unsupported-feature family
// (a missing codec, an encode-only gap, an unmapped character set, a DICOMweb media type the
// build does not offer) classifies here rather than as a general failure: the input is
// well-formed but cannot be processed faithfully, which is the same "cannot honour this data"
// class as malformed input (docs/reference/cli.md resolved mapping).
func isParseError(err error) bool {
	var (
		// dicom
		dcmLimit   *dicom.LimitExceededError
		dcmValue   *dicom.ValueError
		dcmCodec   *dicom.CodecUnavailableError
		dcmEncode  *dicom.EncodeUnsupportedError
		dcmCharset *dicom.UnsupportedCharacterSetError
		// dicomweb
		webTrunc    *dicomweb.TruncatedError
		webMalform  *dicomweb.MalformedPartError
		webDecode   *dicomweb.DecodeError
		webEncode   *dicomweb.EncodeError
		webQuery    *dicomweb.QueryError
		webLimit    *dicomweb.LimitExceededError
		webValidErr *dimse.ValidationError
		// hl7v2
		hl7Parse   *hl7v2.ParseError
		hl7Segment *hl7v2.SegmentError
		hl7Frame   *hl7v2.FrameError
	)
	switch {
	case errors.As(err, &dcmLimit),
		errors.As(err, &dcmValue),
		errors.As(err, &dcmCodec),
		errors.As(err, &dcmEncode),
		errors.As(err, &dcmCharset),
		errors.As(err, &webTrunc),
		errors.As(err, &webMalform),
		errors.As(err, &webDecode),
		errors.As(err, &webEncode),
		errors.As(err, &webQuery),
		errors.As(err, &webLimit),
		errors.As(err, &webValidErr),
		errors.As(err, &hl7Parse),
		errors.As(err, &hl7Segment),
		errors.As(err, &hl7Frame):
		return true
	}
	// Truncation is a parse failure: a short read mid-value propagates io.ErrUnexpectedEOF
	// (docs/reference/cli.md "Truncation and incompleteness are failures"). The dicom reader
	// wraps it in ErrTruncated, but a bare io.ErrUnexpectedEOF reaching here — past the
	// network typed-error checks above, which already absorb a transport truncation — is a
	// parser ending mid-object, so it classifies as a parse failure rather than the general
	// floor. The convert and fhir sentinels are matched the same way.
	switch {
	case errors.Is(err, io.ErrUnexpectedEOF),
		errors.Is(err, dicom.ErrTruncated),
		errors.Is(err, dicom.ErrCodecUnavailable),
		errors.Is(err, dicom.ErrEncodeUnsupported),
		errors.Is(err, dicomweb.ErrUnsupported),
		errors.Is(err, dicomweb.ErrNotAcceptable),
		errors.Is(err, dicomweb.ErrInvalidResource),
		errors.Is(err, dicomweb.ErrLimitExceeded),
		errors.Is(err, convert.ErrMalformedSource),
		errors.Is(err, convert.ErrMissingIdentifier),
		errors.Is(err, convert.ErrUnsupportedSource),
		errors.Is(err, fhir.ErrNilResource),
		errors.Is(err, fhir.ErrResourceTypeMismatch),
		errors.Is(err, fhir.ErrUnknownResourceType),
		errors.Is(err, fhir.ErrUnknownCode):
		return true
	}
	return false
}

// FromOperationOutcome maps a FHIR structural validation result onto the taxonomy. A FHIR
// *OperationOutcome is NOT a Go error, so a command that validates a resource calls this
// helper rather than Classify: an outcome carrying at least one error- or fatal-severity
// issue is a parse/validation failure (exit 3); an outcome with no error issues (or a nil
// outcome) is Success. The issue diagnostics name element paths and codes, never PHI, so the
// caller may log them safely (PRD §9.1).
func FromOperationOutcome(oo *fhir.OperationOutcome) int {
	if oo.HasErrors() {
		return ParseError
	}
	return Success
}
