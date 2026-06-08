package rest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/codeninja55/go-radx/fhir"
)

// response is the parsed result of one interaction: the HTTP status, the bytes of the body (capped
// on read), and the response headers the client surfaces (ETag, Location, Last-Modified). The body
// is read fully into memory because a FHIR resource or Bundle is decoded as a whole; the cap bounds
// it so a hostile origin cannot exhaust memory (PRD §9.3).
type response struct {
	status int
	body   []byte
	header http.Header
}

// doRequest builds and executes one interaction against path (relative to the service base) and
// returns the parsed response. A nil body is a request with no entity. The method sets the FHIR
// JSON Accept header on every request; a caller that submits a body sets the Content-Type through
// the headers map. A non-nil headers map carries conditional and content-type headers (If-Match,
// If-None-Exist, Content-Type). A transport failure is wrapped as a *TransportError with the
// redacted path; the HTTP status is left to the caller to classify so a 4xx body can be parsed
// first.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*response, error) {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", mediaTypeFHIRJSON)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(c.boundedBody(resp))
	if err != nil {
		return nil, c.transportError(method, path, err)
	}
	return &response{status: resp.StatusCode, body: raw, header: resp.Header}, nil
}

// newRequest builds a request against the service base. Authentication is a transport concern: the
// configured credential scheme attaches its header in the http.Client transport (see applyAuth), so
// no credential is set on the request here and none can leak into a logged *http.Request (PRD §9.8).
// path is joined to the base URL; a path that already names an absolute URL (a Bundle.link the
// client follows) is used verbatim so paging can follow a server-supplied absolute link.
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	target := c.resolveURL(path)
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, fmt.Errorf("fhir/rest: build %s request: %w", method, err)
	}
	return req, nil
}

// resolveURL turns a path into an absolute request URL by RFC 3986 reference resolution, the rule a
// Bundle.link.url the client follows for paging must obey. An absolute link (http/https) is used
// verbatim; an origin-relative link (a leading slash, for example "/fhir/Patient?page=2") resolves
// against the base URL's scheme and host only, never against the base path, so it does not double
// the path prefix when the base already carries one (a non-root deployment under "/fhir"); a
// relative link (no leading slash) joins to the service base path; an empty path is the base itself
// (the system-root transaction POST). net/url.ResolveReference implements exactly these semantics
// against the parsed base, so a server's own next/previous links resolve correctly whatever form
// they take.
func (c *Client) resolveURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if path == "" {
		return c.baseURL
	}
	if strings.HasPrefix(path, "/") {
		ref, err := url.Parse(path)
		if err != nil {
			return c.baseURL + path
		}
		return c.origin.ResolveReference(ref).String()
	}
	return c.baseURL + "/" + path
}

// decodeResource decodes a non-empty response body into a concrete resource of the client's release
// through the release registry, so the dynamic type is (for example) *r5.Patient. An empty body is
// an error here: the read-style interactions that call it (read, vread, history, search, metadata)
// promise a resource, so a missing one is an honest failure, never a silent nil (PRD §9.2). The
// write methods do not call this on a bodyless success — a return=minimal write is handled from the
// response headers in resultFromResponse.
func (c *Client) decodeResource(resp *response) (fhir.Resource, error) {
	if len(resp.body) == 0 {
		return nil, fmt.Errorf("fhir/rest: response carried no resource body")
	}
	r, err := c.registry.UnmarshalResource(resp.body)
	if err != nil {
		return nil, fmt.Errorf("fhir/rest: decode response: %w", err)
	}
	return r, nil
}

// errorForResponse maps a non-2xx response to a typed error. When the body parses as a FHIR
// OperationOutcome, its issues are carried through so the caller can classify by severity and code;
// the HTTP status maps to an errors.Is-comparable sentinel. A body that is not an OperationOutcome
// (an empty body, a non-FHIR error page) still yields an *OperationOutcomeError with a nil Outcome
// and the mapped sentinel, so a non-2xx is never read as a success. The OperationOutcome is decoded
// through the release registry and reduced to the release-agnostic fhir.OperationOutcome the error
// carries, matching the fhir package's error model.
func (c *Client) errorForResponse(method, path string, resp *response) error {
	outcome := c.parseOutcome(resp.body)
	return &OperationOutcomeError{
		StatusCode: resp.status,
		Sentinel:   sentinelForStatus(resp.status),
		Outcome:    outcome,
		Method:     method,
		URL:        redactURL(path),
	}
}

// parseOutcome decodes a response body into a release OperationOutcome and reduces it to the
// release-agnostic fhir.OperationOutcome the typed error carries. A body that is absent or not an
// OperationOutcome resource yields nil, so a non-FHIR error page does not masquerade as an outcome.
// The reduction keeps only the severity, code, diagnostics, and the first expression of each issue,
// all of which are structural locators, never patient values (PRD §9.1).
func (c *Client) parseOutcome(body []byte) *fhir.OperationOutcome {
	if len(body) == 0 {
		return nil
	}
	r, err := c.registry.UnmarshalResource(body)
	if err != nil {
		return nil
	}
	return outcomeFromResource(r)
}

// boundedBody wraps a response body in a reader that fails closed when the body exceeds the
// configured cap, so a hostile origin cannot stream an unbounded body into memory (PRD §9.3). An
// over-cap body returns an error rather than a silent EOF, so an over-limit transfer is never read
// as a complete one.
func (c *Client) boundedBody(resp *http.Response) io.Reader {
	return &cappedReader{r: resp.Body, remaining: c.maxResponseBytes, limit: c.maxResponseBytes}
}

// cappedReader reads at most limit bytes from r, then returns an error on the next read that would
// exceed the cap. It distinguishes a body that fits exactly (clean EOF) from one that overruns
// (error), so truncation is never mistaken for completion.
type cappedReader struct {
	r         io.Reader
	remaining int64
	limit     int64
}

func (cr *cappedReader) Read(p []byte) (int, error) {
	if cr.remaining <= 0 {
		var probe [1]byte
		n, err := cr.r.Read(probe[:])
		if n > 0 {
			return 0, fmt.Errorf("fhir/rest: response body exceeds %d-byte cap", cr.limit)
		}
		return 0, err
	}
	if int64(len(p)) > cr.remaining {
		p = p[:cr.remaining]
	}
	n, err := cr.r.Read(p)
	cr.remaining -= int64(n)
	return n, err
}

// transportError wraps a transport-level failure with the redacted path. It unwraps a *url.Error
// first: net/http embeds the full request URL (which carries the resource id and the query string)
// in url.Error.Error(), so wrapping it verbatim would reintroduce identifiers the redacted path was
// meant to remove (PRD §9.1). Only the underlying cause is wrapped.
func (c *Client) transportError(method, path string, err error) error {
	return &TransportError{Method: method, URL: redactURL(path), Err: sanitizeTransportError(err)}
}

// sanitizeTransportError unwraps a *url.Error to its underlying cause so the full request URL
// net/http embeds in url.Error.Error() never reaches a log (PRD §9.1). A non-url.Error is returned
// unchanged.
func sanitizeTransportError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) && ue.Err != nil {
		return ue.Err
	}
	return err
}

// redactURL strips a request path's query string, so a logged error names the interaction without
// exposing PHI-bearing search parameters (a search query can carry a patient name or identifier).
// The path segments (the resource type and id) are kept: a resource type is not PHI, and an id is a
// server-assigned logical id, not a patient value, so the diagnostic stays legible.
func redactURL(path string) string {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		return path[:i]
	}
	return path
}
