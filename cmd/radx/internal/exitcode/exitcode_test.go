package exitcode

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"testing"
	"time"

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

// TestClassifyTotality asserts a representative error of every taxonomy category maps to the
// code it must, and — critically — that nothing intended for a specific class falls through to
// GeneralFailure by accident. It is the totality gate: a typed error the library adds later
// that should map to 3/4/5 but is missing from Classify shows up as a 1 here only if a case is
// added for it, so the table is the living record of the contract (docs/reference/cli.md
// "Exit-code taxonomy").
func TestClassifyTotality(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		// Success.
		{"nil error", nil, Success},

		// Usage (exit 2): a Kong parse failure and a command-raised usage fault.
		{"kong parse error", parseErrorFixture(t), UsageError},
		{"command usage error", &UsageErr{Message: "format not supported"}, UsageError},

		// Parse / validation / unsupported-feature (exit 3): dicom.
		{"dicom ErrTruncated", dicom.ErrTruncated, ParseError},
		{"dicom truncated wrapped", fmt.Errorf("read: %w", dicom.ErrTruncated), ParseError},
		{"dicom LimitExceededError", &dicom.LimitExceededError{Kind: "element-length", Limit: 1, Actual: 2}, ParseError},
		{"dicom ValueError", &dicom.ValueError{VR: dicom.VRUI, Msg: "bad uid"}, ParseError},
		{"dicom CodecUnavailableError", &dicom.CodecUnavailableError{TransferSyntax: dicom.JPEGBaseline8Bit}, ParseError},
		{"dicom EncodeUnsupportedError", &dicom.EncodeUnsupportedError{TransferSyntax: dicom.JPEGBaseline8Bit}, ParseError},
		{"dicom UnsupportedCharacterSetError", &dicom.UnsupportedCharacterSetError{DefinedTerm: "ISO_IR 999"}, ParseError},
		{"dicom ErrCodecUnavailable sentinel", dicom.ErrCodecUnavailable, ParseError},
		{"dicom ErrEncodeUnsupported sentinel", dicom.ErrEncodeUnsupported, ParseError},

		// Parse (exit 3): dicomweb.
		{"dicomweb TruncatedError", &dicomweb.TruncatedError{Detail: "mid-part"}, ParseError},
		{"dicomweb MalformedPartError", &dicomweb.MalformedPartError{Detail: "bad boundary"}, ParseError},
		{"dicomweb DecodeError", &dicomweb.DecodeError{Msg: "bad json"}, ParseError},
		{"dicomweb EncodeError", &dicomweb.EncodeError{Msg: "bad value"}, ParseError},
		{"dicomweb QueryError", &dicomweb.QueryError{Status: 400}, ParseError},
		{"dicomweb LimitExceededError", &dicomweb.LimitExceededError{Kind: "part-count", Limit: 1, Actual: 2}, ParseError},
		{"dicomweb ErrUnsupported", dicomweb.ErrUnsupported, ParseError},
		{"dicomweb ErrNotAcceptable", dicomweb.ErrNotAcceptable, ParseError},
		{"dicomweb ErrInvalidResource", dicomweb.ErrInvalidResource, ParseError},

		// Parse (exit 3): hl7v2.
		{"hl7v2 ParseError", &hl7v2.ParseError{Offset: 3, Reason: "bad segment"}, ParseError},
		{"hl7v2 SegmentError", &hl7v2.SegmentError{Segment: "PID", Reason: "wrong id"}, ParseError},
		{"hl7v2 FrameError", &hl7v2.FrameError{Reason: "no end block"}, ParseError},

		// Parse (exit 3): convert + fhir sentinels.
		{"convert ErrMalformedSource", convert.ErrMalformedSource, ParseError},
		{"convert ErrMissingIdentifier", convert.ErrMissingIdentifier, ParseError},
		{"convert ErrUnsupportedSource", convert.ErrUnsupportedSource, ParseError},
		{"fhir ErrNilResource", fhir.ErrNilResource, ParseError},
		{"fhir ErrResourceTypeMismatch", fhir.ErrResourceTypeMismatch, ParseError},
		{"fhir ErrUnknownResourceType", fhir.ErrUnknownResourceType, ParseError},
		{"fhir ErrUnknownCode", fhir.ErrUnknownCode, ParseError},

		// Network / protocol (exit 4): dimse.
		{"dimse AssociationError", &dimse.AssociationError{Kind: dimse.AssociationRejected}, NetworkError},
		{"dimse AbortError", &dimse.AbortError{Source: 1, Reason: 2}, NetworkError},
		{"dimse ProtocolError", &dimse.ProtocolError{Detail: "bad pdu"}, NetworkError},
		{"dimse CommitmentFailureError", &dimse.CommitmentFailureError{FailedCount: 1}, NetworkError},
		{"dimse non-success Status", &StatusError{Status: dimse.NewStatus(0xC000, dimse.ServiceClassVerification)}, NetworkError},
		{"command protocol error (HL7 AE/AR ack)", &ProtocolErr{Message: "peer returned a non-accept acknowledgement: AR"}, NetworkError},
		{"acse RejectedError", &acse.RejectedError{Result: 1, Source: 1, Reason: 1}, NetworkError},
		{"acse AbortedError", &acse.AbortedError{Source: 1, Reason: 1}, NetworkError},
		{"acse ProtocolError", &acse.ProtocolError{Detail: "bad state", State: dul.State(0)}, NetworkError},
		{"dul StateError", &dul.StateError{State: dul.State(0)}, NetworkError},
		{"pdu PDUError", &pdu.PDUError{Detail: "bad length"}, NetworkError},

		// Network (exit 4): dicomweb transport.
		{"dicomweb HTTPError", &dicomweb.HTTPError{StatusCode: 503, Method: "GET", URL: "x"}, NetworkError},
		{"dicomweb StoreError", &dicomweb.StoreError{Status: 409}, NetworkError},
		{"dicomweb FailureReasonError", &dicomweb.FailureReasonError{Reason: 0xA700}, NetworkError},

		// Network (exit 4): raw transport errors the standard library raises before any typed
		// wrapper — a refused connection (dicomweb/hl7 dialling a closed port) and a net timeout.
		// These arrive as a wrapped *net.OpError / net.Error, not a library type, so the classifier
		// must match them or they fall through to the general floor.
		{"refused connection (*net.OpError)", refusedConnectionFixture(t), NetworkError},
		{"refused connection wrapped", fmt.Errorf("hl7 send: %w", refusedConnectionFixture(t)), NetworkError},
		{"net timeout (net.Error interface)", timeoutErr{}, NetworkError},
		{"net timeout wrapped", fmt.Errorf("dicomweb GET: %w", timeoutErr{}), NetworkError},

		// File I/O (exit 5).
		{"os.PathError", &os.PathError{Op: "open", Path: "x", Err: errors.New("boom")}, FileIOError},
		{"fs.ErrNotExist", fs.ErrNotExist, FileIOError},
		{"fs.ErrPermission", fs.ErrPermission, FileIOError},
		{"path error wrapping ErrNotExist", &os.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}, FileIOError},

		// General (exit 1): genuinely unclassified, plus the deliberate fail-closed stub.
		{"plain error", errors.New("something went wrong"), GeneralFailure},
		{"NotImplementedError", &NotImplementedError{Capability: "find"}, GeneralFailure},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.err); got != tc.want {
				t.Errorf("Classify(%q) = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}

// TestFileIOBeatsParse asserts the precedence rule: a *.dcm that cannot be opened is a
// file-I/O error (exit 5), not a DICOM parse error, even when both could match. A PathError
// wrapping ErrNotExist must classify to 5, never 3.
func TestFileIOBeatsParse(t *testing.T) {
	err := &os.PathError{Op: "open", Path: "missing.dcm", Err: fs.ErrNotExist}
	if got := Classify(err); got != FileIOError {
		t.Errorf("Classify(open-missing.dcm) = %d, want %d (file I/O beats parse)", got, FileIOError)
	}
}

// TestFromOperationOutcome confirms the FHIR OperationOutcome helper: an outcome with an
// error-severity issue is a parse failure (exit 3); a nil or clean outcome is Success.
func TestFromOperationOutcome(t *testing.T) {
	if got := FromOperationOutcome(nil); got != Success {
		t.Errorf("FromOperationOutcome(nil) = %d, want %d", got, Success)
	}
	clean := &fhir.OperationOutcome{}
	if got := FromOperationOutcome(clean); got != Success {
		t.Errorf("FromOperationOutcome(clean) = %d, want %d", got, Success)
	}
	withError := validateInvalidResource(t)
	if !withError.HasErrors() {
		t.Fatal("test setup: expected an outcome with error-severity issues")
	}
	if got := FromOperationOutcome(withError); got != ParseError {
		t.Errorf("FromOperationOutcome(error) = %d, want %d", got, ParseError)
	}
}

// parseErrorFixture produces a genuine *kong.ParseError by parsing an unknown flag against a
// trivial grammar, so the totality table tests the real Kong error type, not a hand-rolled
// stand-in.
func parseErrorFixture(t *testing.T) error {
	t.Helper()
	var grammar struct {
		Name string `arg:"" optional:""`
	}
	parser, err := kong.New(&grammar, kong.Name("fixture"),
		kong.Writers(io.Discard, io.Discard),
		kong.Exit(func(int) {}))
	if err != nil {
		t.Fatalf("build fixture parser: %v", err)
	}
	_, parseErr := parser.Parse([]string{"--unknown-flag"})
	if parseErr == nil {
		t.Fatal("expected a parse error from an unknown flag")
	}
	return parseErr
}

// refusedConnectionFixture produces a genuine refused-connection error by dialling a port no
// listener holds, so the totality table tests the real *net.OpError the standard library raises
// (the shape a dicomweb or hl7 send sees against a closed endpoint) rather than a hand-rolled
// stand-in. It binds and immediately closes a listener to obtain a definitely-free port.
func refusedConnectionFixture(t *testing.T) error {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	conn, dialErr := net.DialTimeout("tcp", addr, 2*time.Second)
	if dialErr == nil {
		_ = conn.Close()
		t.Fatalf("dial of a closed port unexpectedly succeeded")
	}
	var opErr *net.OpError
	if !errors.As(dialErr, &opErr) {
		t.Fatalf("refused-connection error = %T, want a *net.OpError", dialErr)
	}
	return dialErr
}

// timeoutErr is a minimal net.Error reporting a timeout, so the totality table can exercise the
// classifier's net.Error-interface match without depending on a real deadline expiring.
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// validateInvalidResource builds a FHIR OperationOutcome carrying an error-severity issue
// without depending on a release type, so the helper test exercises a real outcome.
func validateInvalidResource(t *testing.T) *fhir.OperationOutcome {
	t.Helper()
	oo := &fhir.OperationOutcome{
		Issue: []fhir.OutcomeIssue{{
			Severity:    fhir.SeverityError,
			Code:        fhir.IssueTypeRequired,
			Diagnostics: "Patient.gender: required element absent",
			Expression:  "Patient.gender",
		}},
	}
	return oo
}
