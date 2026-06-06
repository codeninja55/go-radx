package dicomweb

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// defaultMaxResponseBytes caps a single response body before allocation (PRD §9.3). It
// is 512 MiB, large enough for a multi-frame instance yet bounded against a hostile
// origin that would stream an unbounded body.
const defaultMaxResponseBytes int64 = 512 << 20

// defaultClientTimeout bounds a single request when the caller's context has no deadline
// of its own. A request still aborts immediately on context cancellation.
const defaultClientTimeout = 5 * time.Minute

// Client is a DICOMweb client for a single origin server. It is safe for concurrent use:
// it holds no per-request mutable state and the underlying http.Client is concurrent-safe.
type Client struct {
	baseURL          string
	httpClient       *http.Client
	bearerToken      string
	maxResponseBytes int64
	transferSyntaxes []dicom.TransferSyntax
	bulkDataBaseURL  string
}

// ClientOption configures a Client. There is no global configuration; every knob is an
// option (PRD §8.1, §8.2).
type ClientOption func(*Client)

// WithHTTPClient supplies the transport. The default verifies TLS 1.2+ peers; an
// overriding client controls its own TLS policy.
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = h }
}

// WithBearerToken sets the Authorization: Bearer token. The token is sent on every
// request and is never logged or placed in an error message (PRD §9.8).
func WithBearerToken(token string) ClientOption {
	return func(c *Client) { c.bearerToken = token }
}

// WithMaxResponseBytes caps any single response body (default 512 MiB).
func WithMaxResponseBytes(n int64) ClientOption {
	return func(c *Client) { c.maxResponseBytes = n }
}

// WithTransferSyntaxes sets the WADO-RS Accept transfer-syntax preference, ordered
// most-preferred first.
func WithTransferSyntaxes(ts ...dicom.TransferSyntax) ClientOption {
	return func(c *Client) { c.transferSyntaxes = append([]dicom.TransferSyntax(nil), ts...) }
}

// WithClientBulkDataBaseURL sets the base URL a relative BulkDataURI is resolved against by
// ResolveBulkDataURI. A metadata response may carry a BulkDataURI relative to the origin;
// configuring the base so it matches the origin lets the client resolve it without the
// caller reconstructing the absolute URL. Without it, a relative reference is resolved
// against the client's own origin base URL. An absolute BulkDataURI is always fetched as
// given, regardless of this option.
func WithClientBulkDataBaseURL(base string) ClientOption {
	return func(c *Client) { c.bulkDataBaseURL = strings.TrimRight(base, "/") }
}

// WithInsecureSkipVerify disables TLS peer verification. It is reachable only through
// this explicit option and is intended for tests against a self-signed origin; it is
// never the default path (PRD §9.7). It replaces the transport, so combine it with
// WithHTTPClient only when the supplied client is the one to weaken.
func WithInsecureSkipVerify() ClientOption {
	return func(c *Client) {
		c.httpClient = &http.Client{
			Timeout: defaultClientTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // explicit opt-in test path (PRD §9.7)
			},
		}
	}
}

// defaultHTTPClient returns a TLS-verifying client (TLS 1.2 floor) with a bounded
// per-request timeout. Peer verification is on; InsecureSkipVerify is reachable only via
// the explicit WithInsecureSkipVerify option (PRD §9.7).
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: defaultClientTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
}

// NewClient returns a DICOMweb client for the origin at baseURL (the path that precedes
// /studies, e.g. https://pacs.example.org/dicom-web). A baseURL is required; the trailing
// slash is trimmed so path joins are unambiguous.
func NewClient(baseURL string, opts ...ClientOption) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("dicomweb: NewClient requires a base URL")
	}
	c := &Client{
		baseURL:          strings.TrimRight(baseURL, "/"),
		maxResponseBytes: defaultMaxResponseBytes,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = defaultHTTPClient()
	}
	return c, nil
}

// Store POSTs one or more instances as a multipart/related body of application/dicom
// parts to /studies and parses the application/dicom+json store response. It is
// fail-closed (PRD §9.2): when any instance failed it returns the parsed StoreResponse
// together with a non-nil *StoreError, so the caller sees both the partial success and
// the failure. A transport error returns a nil response and the transport error.
func (c *Client) Store(ctx context.Context, instances ...*dicom.DataSet) (*StoreResponse, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("dicomweb: Store requires at least one instance")
	}

	var body bytes.Buffer
	mw := NewMultipartWriter(&body, mediaTypeDICOM)
	for _, ds := range instances {
		raw, err := encodeInstance(ds, defaultStorageTransferSyntax)
		if err != nil {
			return nil, err
		}
		if err := mw.AddPart(mediaTypeDICOM, bytes.NewReader(raw)); err != nil {
			return nil, err
		}
	}
	if _, err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/studies", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.ContentType())
	req.Header.Set("Accept", mediaTypeDICOMJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(http.MethodPost, "/studies", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// PS3.18 §10.5.3: 200 OK every instance accepted, 202 Accepted some accepted, 409
	// Conflict none accepted. All three carry a store-response document; a 4xx/5xx outside
	// that set is a transport-level failure with no parseable response.
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusConflict:
	default:
		return nil, c.httpError(http.MethodPost, "/studies", resp)
	}

	store, err := c.parseStoreResponseBody(resp)
	if err != nil {
		return nil, err
	}
	// Fail closed on the HTTP status, not only on the parsed body: a 202/409 means the
	// origin reported a partial or total failure, so Store must return a *StoreError even
	// when the response body omits or under-reports the Failed SOP Sequence (a sparse or
	// malformed store response must never read as a clean success, PRD §9.2).
	if resp.StatusCode != http.StatusOK || !store.IsComplete() {
		return store, &StoreError{
			Failed:   store.Failed,
			Accepted: len(store.Referenced),
			Status:   resp.StatusCode,
		}
	}
	return store, nil
}

// RetrieveInstance fetches a single instance, issuing a multipart/related GET and
// parsing the application/dicom part into a *dicom.DataSet. A response that is not
// multipart/related, or that carries no application/dicom part, is a typed error.
func (c *Client) RetrieveInstance(ctx context.Context, p ResourcePath) (*dicom.DataSet, error) {
	path, err := p.Path()
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", acceptInstances(c.transferSyntaxes...))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(http.MethodGet, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, c.httpError(http.MethodGet, path, resp)
	}
	if !isMultipartRelated(resp.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("%w: WADO-RS response is not multipart/related", ErrNotAcceptable)
	}

	body := c.boundedBody(resp)
	mr, err := NewMultipartReader(body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	ct, part, err := mr.NextPart()
	if errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: WADO-RS response carried no instance part", ErrNotAcceptable)
	}
	if err != nil {
		return nil, err
	}
	if mt := mediaTypeOf(ct); mt != mediaTypeDICOM {
		return nil, fmt.Errorf("%w: WADO-RS part media type %q is not application/dicom", ErrNotAcceptable, mt)
	}
	return decodeInstance(part)
}

// parseStoreResponseBody reads and decodes the application/dicom+json store-response body
// through the bounded reader, then parses it into a StoreResponse.
func (c *Client) parseStoreResponseBody(resp *http.Response) (*StoreResponse, error) {
	raw, err := io.ReadAll(c.boundedBody(resp))
	if err != nil {
		return nil, c.readError("POST", "/studies", err)
	}
	ds, err := UnmarshalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: parse STOW-RS store response: %w", err)
	}
	return parseStoreResponse(ds), nil
}

// newRequest builds an authenticated request against the origin. The bearer token, if
// set, is attached here and never logged.
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: build %s request: %w", method, err)
	}
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
	return req, nil
}

// boundedBody wraps a response body in a reader that fails closed when the body exceeds
// the configured cap, so a hostile origin cannot stream an unbounded body into memory
// (PRD §9.3). Unlike io.LimitReader, an over-cap body returns *LimitExceededError rather
// than a silent EOF, so an over-limit transfer is never read as a complete one.
func (c *Client) boundedBody(resp *http.Response) io.Reader {
	return &cappedReader{r: resp.Body, remaining: c.maxResponseBytes, limit: c.maxResponseBytes}
}

// cappedReader reads at most limit bytes from r, then returns *LimitExceededError on the
// next read that would exceed the cap. It distinguishes a body that fits exactly (clean
// EOF) from one that overruns (error), so truncation is never mistaken for completion.
type cappedReader struct {
	r         io.Reader
	remaining int64
	limit     int64
}

func (cr *cappedReader) Read(p []byte) (int, error) {
	if cr.remaining <= 0 {
		// Peek one byte: if the body still has content, it exceeds the cap.
		var probe [1]byte
		n, err := cr.r.Read(probe[:])
		if n > 0 {
			return 0, &LimitExceededError{
				Limit:  uint64(cr.limit),
				Actual: uint64(cr.limit) + 1,
				Kind:   "response-body-bytes",
			}
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

// transportError wraps a transport-level failure with the redacted path. It unwraps a
// *url.Error first: net/http embeds the full request URL (which carries the resource
// UIDs) in url.Error.Error(), so wrapping it verbatim would reintroduce identifiers the
// redacted path argument was meant to remove (PRD §9.1). Only the underlying cause is
// wrapped.
func (c *Client) transportError(method, path string, err error) error {
	return fmt.Errorf("dicomweb: %s %s: %w", method, redactPath(path), sanitizeTransportError(err))
}

// readError wraps a body-read failure with the same redaction.
func (c *Client) readError(method, path string, err error) error {
	return fmt.Errorf("dicomweb: read %s %s response: %w", method, redactPath(path), sanitizeTransportError(err))
}

// sanitizeTransportError unwraps a *url.Error to its underlying cause so the full,
// UID-bearing request URL net/http embeds in url.Error.Error() never reaches a log
// (PRD §9.1). A non-url.Error is returned unchanged.
func sanitizeTransportError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) && ue.Err != nil {
		return ue.Err
	}
	return err
}

// httpError returns a typed *HTTPError for a non-success response. It deliberately does
// not copy the remote response body into the error: an origin PACS error body can carry
// patient identifiers or query diagnostics, and errors are PHI-free by default (PRD §9.1).
// The URL also has its query stripped, since a QIDO query string can carry PHI-bearing
// match keys. The Detail field is reserved for a server-provided, caller-supplied
// PHI-safe detail and is left empty here.
func (c *Client) httpError(method, path string, resp *http.Response) error {
	return &HTTPError{
		StatusCode: resp.StatusCode,
		Method:     method,
		URL:        redactPath(path),
	}
}

// redactPath returns the request path with its query string dropped and every resource
// UID replaced by a "{uid}" placeholder, so a logged error reveals which service was
// called without exposing study/series/instance identifiers. DICOM UIDs are scrubbed
// during de-identification because they can link back to a patient, so they are treated
// as PHI-bearing in diagnostics (PRD §9.1). The structural keywords (studies, series,
// instances, metadata, frames, bulkdata) are kept so the path stays legible.
func redactPath(path string) string {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		switch seg {
		case "", "studies", "series", "instances", "metadata", "frames", "bulkdata", "pixeldata":
			// A structural keyword or the leading empty segment: keep it.
		default:
			segs[i] = "{uid}"
		}
	}
	return strings.Join(segs, "/")
}

// HTTPError carries a transport-level failure with enough context to act on, without
// PHI. The URL is the request path with its query stripped and every resource UID
// replaced by a "{uid}" placeholder, so it names the service without exposing
// identifiers.
type HTTPError struct {
	StatusCode int
	Method     string
	URL        string
	Detail     string
}

func (e *HTTPError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("dicomweb: %s %s: HTTP %d: %s", e.Method, e.URL, e.StatusCode, e.Detail)
	}
	return fmt.Sprintf("dicomweb: %s %s: HTTP %d", e.Method, e.URL, e.StatusCode)
}
