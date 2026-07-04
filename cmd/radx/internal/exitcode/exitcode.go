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
	"net"
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

// ProtocolErr is an application-level protocol failure a command raises when the exchange itself
// completed — the request was framed and sent, the peer answered — but the peer rejected the work
// at the application level. The HL7 v2 AE (application error) and AR (application reject)
// acknowledgements are the canonical case: the message parsed and sent fine, and the peer said no.
// It classifies to NetworkError (exit 4), mirroring how a non-success DIMSE Status maps, so an
// operator branches on a peer "no" the same way regardless of protocol, and never confuses it with
// a usage fault (exit 2) the way a flag mistake would surface.
type ProtocolErr struct {
	// Message names the protocol-level rejection without any patient value (e.g. the ack code).
	Message string
}

func (e *ProtocolErr) Error() string { return "radx: " + e.Message }

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
	if isA[*kong.ParseError](err) || isA[*UsageErr](err) {
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
	if isA[*os.PathError](err) {
		return true
	}
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission)
}

// isNetworkError reports a transport or protocol fault. It covers the public dimse error
// types and their internal acse/dul/pdu causes (a public *dimse error usually wraps one), a
// non-success DIMSE terminal status promoted to a *StatusError, an application-level protocol
// rejection promoted to a *ProtocolErr (an HL7 v2 AE/AR ack), the Storage Commitment failure,
// the DICOMweb HTTP/store transport failures, and the raw transport errors the standard library
// raises before any typed wrapper is built: a refused connection, a reset, or a timeout dialling
// the DICOMweb or HL7 v2 endpoint surfaces as a wrapped *net.OpError or as a value satisfying the
// net.Error interface, which would otherwise fall through to the general floor (exit 1) rather than
// the network class (exit 4) the taxonomy assigns to a broken conversation.
func isNetworkError(err error) bool {
	switch {
	case isA[*StatusError](err),
		isA[*ProtocolErr](err),
		isA[*dimse.AssociationError](err),
		isA[*dimse.AbortError](err),
		isA[*dimse.ProtocolError](err),
		isA[*dimse.CommitmentFailureError](err),
		isA[*acse.RejectedError](err),
		isA[*acse.AbortedError](err),
		isA[*acse.ProtocolError](err),
		isA[*dul.StateError](err),
		isA[*pdu.PDUError](err),
		isA[*dicomweb.HTTPError](err),
		isA[*dicomweb.StoreError](err),
		isA[*dicomweb.FailureReasonError](err),
		isA[*dicomweb.CrossOriginBulkDataError](err),
		isA[*net.OpError](err),
		isA[net.Error](err):
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
	switch {
	case isA[*dicom.LimitExceededError](err),
		isA[*dicom.ValueError](err),
		isA[*dicom.CodecUnavailableError](err),
		isA[*dicom.EncodeUnsupportedError](err),
		isA[*dicom.UnsupportedCharacterSetError](err),
		isA[*dicomweb.TruncatedError](err),
		isA[*dicomweb.MalformedPartError](err),
		isA[*dicomweb.DecodeError](err),
		isA[*dicomweb.EncodeError](err),
		isA[*dicomweb.QueryError](err),
		isA[*dicomweb.LimitExceededError](err),
		isA[*dimse.ValidationError](err),
		isA[*hl7v2.ParseError](err),
		isA[*hl7v2.SegmentError](err),
		isA[*hl7v2.FrameError](err):
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

// isA reports whether err's chain contains an error of type T. The classifier branches on
// the error class alone, so the matched value is discarded.
func isA[T error](err error) bool {
	_, ok := errors.AsType[T](err)
	return ok
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
