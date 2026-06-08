package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/codeninja55/go-radx/fhir"
)

// Sentinel errors the client returns. Compare with errors.Is; a typed error wraps one with %w so a
// caller distinguishes the failure class without parsing human text (PRD §8.2). None carries PHI:
// a FHIR error names the resource type, the id, and the structural locator of an issue, never a
// patient value (PRD §9.1).
var (
	// ErrNotFound is returned when the server answers 404 Not Found for a read, vread, or any
	// interaction whose target resource is absent. It lets a caller distinguish a genuine miss from
	// a transport or validation failure.
	ErrNotFound = errors.New("fhir/rest: resource not found")

	// ErrConflict is returned when the server answers 409 Conflict or 412 Precondition Failed: an
	// optimistic-concurrency clash on an If-Match update, or a conditional create whose precondition
	// did not hold. The caller re-reads the current version and retries.
	ErrConflict = errors.New("fhir/rest: version conflict or failed precondition")

	// ErrUnprocessable is returned when the server answers 422 Unprocessable Entity: a well-formed
	// resource the server rejected on a business or profile rule. The accompanying OperationOutcome
	// names the rule.
	ErrUnprocessable = errors.New("fhir/rest: resource rejected as unprocessable")

	// ErrUnauthorized is returned when the server answers 401 Unauthorized or 403 Forbidden: the
	// credential the transport presented was missing, rejected, or insufficient.
	ErrUnauthorized = errors.New("fhir/rest: request not authorized")

	// ErrUnsupported is returned when the server answers 405 Method Not Allowed or 501 Not
	// Implemented: the server does not offer the interaction the client attempted. A capability
	// negotiation reports the same condition before the request is sent.
	ErrUnsupported = errors.New("fhir/rest: interaction not supported by the server")

	// ErrNoNextPage is returned by FollowNext/FollowPrev when the page carries no link to follow, so
	// a paging loop terminates cleanly with errors.Is(err, rest.ErrNoNextPage) rather than a generic
	// error.
	ErrNoNextPage = errors.New("fhir/rest: no further page link")
)

// OperationOutcomeError is the typed error a non-2xx FHIR response maps to. The server's response
// body, when it parses as a FHIR OperationOutcome, is surfaced through Outcome so a caller can
// classify the failure by issue severity and code without re-parsing the body; the HTTP status is
// carried alongside, and Sentinel is the errors.Is-comparable class the status maps to (one of the
// sentinels above), so a caller can branch on errors.Is(err, rest.ErrConflict) regardless of which
// release produced the outcome.
//
// The error message names the status, the issue codes, and the issue diagnostics, all of which are
// element paths, resource types, and rule names, never patient values (PRD §9.1). The wrapped
// Sentinel is unwrapped by errors.Is, and the outcome is reachable with errors.As for callers that
// want the structured issues. This aligns with how exitcode.FromOperationOutcome classifies a
// validation result: an error-severity outcome is a parse/validation failure.
type OperationOutcomeError struct {
	// StatusCode is the HTTP status the server returned (for example 404, 409, 422).
	StatusCode int

	// Sentinel is the errors.Is-comparable class the status maps to, so errors.Is(err, sentinel)
	// works without inspecting the status code directly.
	Sentinel error

	// Outcome holds the issues parsed from the response body when it was a FHIR OperationOutcome,
	// or nil when the body was absent or not an OperationOutcome. Issue diagnostics name structural
	// locators, never patient values.
	Outcome *fhir.OperationOutcome

	// Method and URL name the failed interaction without its query string, so a logged error reveals
	// which interaction failed without exposing PHI-bearing search parameters.
	Method string
	URL    string
}

// Error renders the failure with the status, the interaction, and the issue diagnostics, all
// PHI-free.
func (e *OperationOutcomeError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "fhir/rest: %s %s: HTTP %d", e.Method, e.URL, e.StatusCode)
	if e.Outcome != nil {
		for i := range e.Outcome.Issue {
			issue := &e.Outcome.Issue[i]
			b.WriteString("; ")
			b.WriteString(string(issue.Severity))
			if issue.Code != "" {
				b.WriteString(" ")
				b.WriteString(string(issue.Code))
			}
			if issue.Expression != "" {
				b.WriteString(" at ")
				b.WriteString(issue.Expression)
			}
			if issue.Diagnostics != "" {
				b.WriteString(": ")
				b.WriteString(issue.Diagnostics)
			}
		}
	}
	return b.String()
}

// Unwrap returns the sentinel class so errors.Is(err, rest.ErrConflict) and the like match the
// mapped condition. The sentinel is the bridge between the rich typed error and the simple class
// checks a caller writes.
func (e *OperationOutcomeError) Unwrap() error { return e.Sentinel }

// sentinelForStatus maps an HTTP status to the errors.Is-comparable sentinel class. A status with
// no more-specific class maps to nil, which leaves the OperationOutcomeError matchable only by
// errors.As; the caller still sees the status and the outcome. The mapping mirrors the FHIR HTTP
// status table the server role serves (servers.md): 401/403 unauthorized, 404 not-found,
// 405/501 unsupported, 409/412 conflict, 422 unprocessable.
func sentinelForStatus(status int) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return ErrUnsupported
	case http.StatusConflict, http.StatusPreconditionFailed:
		return ErrConflict
	case http.StatusUnprocessableEntity:
		return ErrUnprocessable
	default:
		return nil
	}
}

// TransportError wraps a transport-level failure (a refused connection, a TLS fault, a context
// cancellation) with the redacted interaction so a logged error names the operation without
// exposing the query string or the resource id. It unwraps a *url.Error to its underlying cause
// first, because net/http embeds the full request URL in url.Error.Error(), which would
// reintroduce identifiers the redacted URL was meant to remove (PRD §9.1).
type TransportError struct {
	Method string
	URL    string
	Err    error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("fhir/rest: %s %s: %v", e.Method, e.URL, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }
