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

	// allowCrossOriginBulkData lifts the same-origin restriction on an absolute BulkDataURI;
	// bulkDataHostAllowlist names hosts that may be fetched cross-origin even without that
	// blanket opt-in. Both are off by default so a server-supplied absolute reference cannot
	// steer the client at an internal address (SSRF, PRD §9.8).
	allowCrossOriginBulkData bool
	bulkDataHostAllowlist    map[string]struct{}

	// transport, when set by WithRoundTripper, replaces the client's base transport
	// wholesale. authLayer, when set by a credential option, wraps the base transport so
	// every same-origin request carries the scheme's header. clientCert, when set by
	// WithClientCertificate, is presented during the TLS handshake. Auth is a transport
	// concern layered through these fields, not a per-request branch (DIP, PRD §8.2).
	transport  http.RoundTripper
	authLayer  func(base http.RoundTripper, origin *url.URL) http.RoundTripper
	clientCert *tls.Certificate

	// retry, when set by WithRetry, applies automatic retry of idempotent reads on a transient
	// failure. It is nil by default (no retry).
	retry *RetryPolicy
}

// ClientOption configures a Client. There is no global configuration; every knob is an
// option (PRD §8.1, §8.2).
type ClientOption func(*Client)

// WithHTTPClient supplies the transport. The default verifies TLS 1.2+ peers; an
// overriding client controls its own TLS policy.
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = h }
}

// WithBearerToken sets a static Authorization: Bearer token. The token is attached to every
// request through the client's transport and is never logged or placed in an error message
// (PRD §9.8). For a token that must refresh during a long-lived session, use WithTokenSource.
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

// WithAllowCrossOriginBulkData permits ResolveBulkDataURI to fetch an absolute BulkDataURI
// whose origin differs from the client's configured origin. The safe default refuses such a
// reference, because a hostile or compromised origin could emit metadata steering the client
// at an internal address (169.254.169.254, an internal service). Enable this only when the
// origin is trusted to reference arbitrary hosts; prefer WithBulkDataHostAllowlist to permit a
// known set of hosts instead (PRD §9.8).
func WithAllowCrossOriginBulkData() ClientOption {
	return func(c *Client) { c.allowCrossOriginBulkData = true }
}

// WithBulkDataHostAllowlist permits ResolveBulkDataURI to fetch an absolute BulkDataURI whose
// host is in the given set, even when it is cross-origin. Each entry is a host, optionally with
// a port (for example "images.example.org" or "images.example.org:8443"); the match is
// case-insensitive on host. It is the narrow alternative to WithAllowCrossOriginBulkData: a
// caller whose metadata legitimately references a fixed second host opens only that host
// rather than every host (PRD §9.8).
func WithBulkDataHostAllowlist(hosts ...string) ClientOption {
	return func(c *Client) {
		if c.bulkDataHostAllowlist == nil {
			c.bulkDataHostAllowlist = make(map[string]struct{}, len(hosts))
		}
		for _, h := range hosts {
			c.bulkDataHostAllowlist[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
		}
	}
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
	c.applyAuth()
	return c, nil
}

// applyAuth composes the configured authentication into the client's http.Client transport.
// The base transport is the one WithRoundTripper installed, else the http.Client's own
// transport (or the TLS-verifying default). A client certificate is folded into that
// transport's TLS config for mutual TLS, then the credential layer (basic, OAuth2 token
// source, or the static bearer) wraps the result so every request carries the scheme's
// header. Layering through the RoundTripper seam keeps the request path branch-free and lets
// the cloud adapters compose without modifying the client (DIP, PRD §8.2).
//
// A caller-supplied *http.Client is never mutated: the credential layer is set on a shallow
// copy, leaving the caller's shared client untouched. Layering over the original base
// transport (not over whatever this client previously installed) means building a second
// authenticated client over the same shared *http.Client does not wrap the first client's
// credential layer, which would otherwise let the inner same-origin layer overwrite the outer
// Authorization header so requests carry the wrong credential.
func (c *Client) applyAuth() {
	layer := c.authLayer
	if layer == nil && c.bearerToken != "" {
		// WithBearerToken keeps its field-set behaviour; it is expressed as a RoundTripper
		// here so it composes identically to the other schemes (PRD §9.8).
		layer = bearerAuthLayer(c.bearerToken)
	}
	if layer == nil && c.transport == nil && c.clientCert == nil {
		// No auth option touches the transport; leave the supplied client's transport as-is.
		return
	}

	base := c.transport
	if base == nil {
		base = c.httpClient.Transport
	}
	if base == nil {
		base = http.DefaultTransport
	}
	if c.clientCert != nil {
		base = withClientCertificate(base, *c.clientCert)
	}
	if layer != nil {
		// The origin scopes every credential to the configured host; a base URL that does not
		// parse yields a nil origin, which fails closed (no request is same-origin).
		origin, _ := url.Parse(c.baseURL)
		base = layer(base, origin)
	}
	clientCopy := *c.httpClient
	clientCopy.Transport = base
	c.httpClient = &clientCopy
}

// withClientCertificate returns a transport that presents cert during the TLS handshake. An
// *http.Transport base is cloned so the certificate is attached without disturbing the
// caller's transport. A non-*http.Transport base (a custom RoundTripper) carries no TLS
// config, so a fresh TLS-verifying transport holding the certificate is built rather than
// silently dropping it.
func withClientCertificate(base http.RoundTripper, cert tls.Certificate) http.RoundTripper {
	tr, ok := base.(*http.Transport)
	if !ok {
		return &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}},
		}
	}
	clone := tr.Clone()
	if clone.TLSClientConfig == nil {
		clone.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	clone.TLSClientConfig.Certificates = append(clone.TLSClientConfig.Certificates, cert)
	return clone
}

// Store POSTs one or more instances as a multipart/related body of application/dicom
// parts to the unconstrained /studies target and parses the application/dicom+json store
// response. It is fail-closed (PRD §9.2): when any instance failed it returns the parsed
// StoreResponse together with a non-nil *StoreError, so the caller sees both the partial
// success and the failure. A transport error returns a nil response and the transport
// error. To target a specific study so the origin rejects instances from a different
// StudyInstanceUID, use StoreToStudy.
func (c *Client) Store(ctx context.Context, instances ...*dicom.DataSet) (*StoreResponse, error) {
	return c.store(ctx, "/studies", instances...)
}

// StoreToStudy POSTs one or more instances to the study-scoped /studies/{study} target so
// the origin constrains the request to that StudyInstanceUID (PS3.18 §10.5): an instance
// whose StudyInstanceUID differs is rejected rather than silently accepted under the root
// target. The study UID is validated as a conformant DICOM UID before it is interpolated,
// so a malformed identifier can never inject path segments. The fail-closed semantics
// match Store.
func (c *Client) StoreToStudy(ctx context.Context, study dicom.UID, instances ...*dicom.DataSet) (*StoreResponse, error) {
	path, err := NewStudy(study).Path()
	if err != nil {
		return nil, err
	}
	return c.store(ctx, path, instances...)
}

// store builds the multipart/related STOW-RS body and POSTs it to path (either the root
// /studies target or a study-scoped /studies/{study} target), then parses and fail-closes
// on the application/dicom+json store response.
func (c *Client) store(ctx context.Context, path string, instances ...*dicom.DataSet) (*StoreResponse, error) {
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

	req, err := c.newRequest(ctx, http.MethodPost, path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.ContentType())
	req.Header.Set("Accept", mediaTypeDICOMJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(http.MethodPost, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// PS3.18 §10.5.3: 200 OK every instance accepted, 202 Accepted some accepted, 409
	// Conflict none accepted. All three carry a store-response document; a 4xx/5xx outside
	// that set is a transport-level failure with no parseable response.
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusConflict:
	default:
		return nil, c.httpError(http.MethodPost, path, resp)
	}

	store, err := c.parseStoreResponseBody(resp)
	if err != nil {
		return nil, err
	}
	// Fail closed on the HTTP status, not only on the parsed body: a 202/409 means the
	// origin reported a partial or total failure, so store must return a *StoreError even
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
// parsing the application/dicom part into a *dicom.DataSet. Only the decoded dataset is returned,
// so the part is streamed straight into the decoder without capturing its raw bytes — use
// RetrieveInstanceObject when the byte-exact Part 10 representation is needed. A response that is not
// multipart/related, or that carries no application/dicom part, is a typed error.
func (c *Client) RetrieveInstance(ctx context.Context, p ResourcePath) (*dicom.DataSet, error) {
	si, err := c.retrieveSingleInstance(ctx, p, false)
	if err != nil {
		return nil, err
	}
	return si.DataSet, nil
}

// RetrieveInstanceObject fetches a single instance, preserving its transfer syntax and byte-exact
// Part 10 representation. Unlike RetrieveInstance, which returns only the decoded dataset and so
// discards the transfer syntax, this returns a RetrievedInstance whose TransferSyntax and Encoded
// fields let the caller write the object back in the syntax the origin returned rather than
// transcoding it, which matters for an encapsulated (compressed) syntax go-radx cannot re-encode
// from a dataset. A response that is not multipart/related, or that carries no application/dicom
// part, is a typed error.
func (c *Client) RetrieveInstanceObject(ctx context.Context, p ResourcePath) (RetrievedInstance, error) {
	return c.retrieveSingleInstance(ctx, p, true)
}

// retrieveSingleInstance issues a multipart/related WADO-RS GET for one instance and decodes the
// single application/dicom part. captureBytes selects the decode path: the object-returning caller
// captures the byte-exact Part 10 representation, while the dataset-only caller streams the part
// without allocating an encoded buffer it would only discard. A response that is not
// multipart/related, or that carries no application/dicom part, is a typed error.
func (c *Client) retrieveSingleInstance(ctx context.Context, p ResourcePath, captureBytes bool) (RetrievedInstance, error) {
	path, err := p.Path()
	if err != nil {
		return RetrievedInstance{}, err
	}
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return RetrievedInstance{}, err
	}
	req.Header.Set("Accept", acceptInstances(c.transferSyntaxes...))

	resp, err := c.do(req, http.MethodGet, path) // #nosec G704 -- the URL is joined from the caller-configured base URL (newRequest); requesting the configured service is the client's purpose
	if err != nil {
		return RetrievedInstance{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return RetrievedInstance{}, c.httpError(http.MethodGet, path, resp)
	}
	if !isMultipartRelated(resp.Header.Get("Content-Type")) {
		return RetrievedInstance{}, fmt.Errorf("%w: WADO-RS response is not multipart/related", ErrNotAcceptable)
	}

	body := c.boundedBody(resp)
	mr, err := NewMultipartReader(body, resp.Header.Get("Content-Type"))
	if err != nil {
		return RetrievedInstance{}, err
	}
	ct, part, err := mr.NextPart()
	if errors.Is(err, io.EOF) {
		return RetrievedInstance{}, fmt.Errorf("%w: WADO-RS response carried no instance part", ErrNotAcceptable)
	}
	if err != nil {
		return RetrievedInstance{}, err
	}
	if mt := mediaTypeOf(ct); mt != mediaTypeDICOM {
		return RetrievedInstance{}, fmt.Errorf("%w: WADO-RS part media type %q is not application/dicom", ErrNotAcceptable, mt)
	}
	return decodeRetrievedInstance(part, captureBytes)
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

// newRequest builds a request against the origin. Authentication is a transport concern: the
// configured credential scheme attaches its header in the http.Client transport (see
// applyAuth), so no credential is set on the request here and none can leak into a logged
// *http.Request (PRD §9.8).
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body) // #nosec G704 -- the URL is joined from the caller-configured base URL; requesting the configured service is the client's purpose
	if err != nil {
		return nil, fmt.Errorf("dicomweb: build %s request: %w", method, err)
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
				Limit:  uint64(cr.limit),     // #nosec G115 -- limit is a non-negative configured byte cap
				Actual: uint64(cr.limit) + 1, // #nosec G115 -- limit is a non-negative configured byte cap
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
