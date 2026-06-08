// Package rest is go-radx's FHIR R4/R5 REST client: a type-safe HTTP client for the FHIR RESTful
// API built over the generated fhir/r4 and fhir/r5 models. A Client targets exactly one FHIR
// release, fixed at construction with NewClient(release, baseURL, ...): the release determines which
// generated registry decodes a server response and which release's Bundle a search or transaction
// exchanges, so a consumer never mixes R4 and R5 types through one client by accident. To talk to
// both releases from one process, construct two clients.
//
// The client implements read, vread, create, update, patch, delete, history, type-level search
// (with typed parameters, chaining, _include/_revinclude, and Bundle.link paging), transaction and
// batch submission, conditional create/update with ETag concurrency, and CapabilityStatement
// negotiation. A non-2xx FHIR response whose body is an OperationOutcome is surfaced as a typed
// *OperationOutcomeError the caller classifies by issue severity, aligning with the fhir package's
// OperationOutcome error model and exitcode.FromOperationOutcome.
//
// Authentication is a pluggable transport concern layered through an http.RoundTripper, mirroring
// the dicomweb client: a credential is injected only on same-origin requests, so a server-supplied
// absolute reference never carries the client's credential. SMART on FHIR is deferred and not
// implemented in v1 (see fhir.md); a bearer token from a SMART flow is supplied through
// WithBearerToken or WithRoundTripper like any other token.
//
// Every method takes a context.Context and honours its cancellation and deadline. Response bodies
// are capped before allocation so a hostile origin cannot stream an unbounded body into memory
// (PRD §9.3). Errors are PHI-free by default: a problem names the resource type, the id, and the
// structural locator of an issue, never a patient value (PRD §9.1).
//
// Stability: experimental. Pre-1.0; the API may change between v0.x releases.
package rest

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/codeninja55/go-radx/fhir"
)

const (
	// mediaTypeFHIRJSON is the FHIR JSON media type the client sends and accepts. It is the only
	// wire format the fhir package serialises in v1 (XML and YAML are deferred), so it is both the
	// Accept and the Content-Type for every interaction that carries a body.
	mediaTypeFHIRJSON = "application/fhir+json"

	// mediaTypeJSONPatch is the Content-Type for a JSON Patch (RFC 6902) document, the patch format
	// the FHIR patch interaction uses for a structural patch.
	mediaTypeJSONPatch = "application/json-patch+json"

	// preferReturnRepresentation is the Prefer header value the client sends on a write (create,
	// update, patch) to ask the server to return the stored resource in the response. A server
	// honouring return=minimal answers 2xx with no body; the client handles that bodyless success
	// from the response headers (Location, ETag) rather than treating it as a failure, so a
	// minimal-returning server is supported either way.
	preferReturnRepresentation = "return=representation"
)

// defaultMaxResponseBytes caps a single response body before allocation (PRD §9.3). It is 32 MiB,
// generous for a large searchset Bundle page yet bounded against an origin that would stream an
// unbounded body.
const defaultMaxResponseBytes int64 = 32 << 20

// defaultClientTimeout bounds a single request when the caller's context carries no deadline of its
// own. A request still aborts immediately on context cancellation.
const defaultClientTimeout = 2 * time.Minute

// Client is a FHIR REST client for a single origin server and a single FHIR release. It is safe for
// concurrent use: it holds no per-request mutable state and the underlying http.Client is
// concurrent-safe.
type Client struct {
	release  fhir.Release
	registry *fhir.Registry
	baseURL  string
	origin   *url.URL

	httpClient       *http.Client
	bearerToken      string
	maxResponseBytes int64

	// transport, when set by WithRoundTripper, replaces the client's base transport wholesale.
	// authLayer, when set by a credential option, wraps the base transport so every same-origin
	// request carries the scheme's header. Auth is a transport concern layered through these fields,
	// not a per-request branch (DIP, PRD §8.2), mirroring the dicomweb client.
	transport http.RoundTripper
	authLayer func(base http.RoundTripper, origin *url.URL) http.RoundTripper
}

// ClientOption configures a Client. There is no global configuration; every knob is an option.
type ClientOption func(*Client)

// WithHTTPClient supplies the transport. The default verifies TLS 1.2+ peers; an overriding client
// controls its own TLS policy.
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = h }
}

// WithBearerToken sets a static Authorization: Bearer token attached to every same-origin request
// through the client's transport. The token is never logged or placed in an error message (PRD
// §9.8). A SMART on FHIR access token (SMART itself being deferred) is supplied here.
func WithBearerToken(token string) ClientOption {
	return func(c *Client) { c.bearerToken = token }
}

// WithMaxResponseBytes caps any single response body (default 32 MiB).
func WithMaxResponseBytes(n int64) ClientOption {
	return func(c *Client) {
		if n > 0 {
			c.maxResponseBytes = n
		}
	}
}

// WithInsecureSkipVerify disables TLS peer verification. It is reachable only through this explicit
// option and is intended for tests against a self-signed origin; it is never the default path (PRD
// §9.7).
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

// defaultHTTPClient returns a TLS-verifying client (TLS 1.2 floor) with a bounded per-request
// timeout. Peer verification is on; InsecureSkipVerify is reachable only via the explicit option.
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: defaultClientTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
}

// registryForRelease returns the generated registry that decodes a server response for the given
// release. The registry is what makes a response's resourceType dispatch to the correct release's
// concrete type; an unknown release fails closed at construction rather than decoding into the
// wrong type space.
func registryForRelease(release fhir.Release) (*fhir.Registry, error) {
	reg, ok := releaseRegistry(release)
	if !ok {
		return nil, fmt.Errorf("fhir/rest: unsupported FHIR release %q (v1 supports %s and %s)",
			release, fhir.R4, fhir.R5)
	}
	return reg, nil
}

// NewClient returns a FHIR REST client for the origin at baseURL serving the given FHIR release.
// baseURL is the FHIR service base — the path that precedes a resource type, for example
// https://hapi.example.org/fhir. release must be fhir.R4 or fhir.R5; any other value is a
// construction error. The trailing slash is trimmed so path joins are unambiguous.
func NewClient(release fhir.Release, baseURL string, opts ...ClientOption) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("fhir/rest: NewClient requires a base URL")
	}
	reg, err := registryForRelease(release)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimRight(baseURL, "/")
	origin, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("fhir/rest: NewClient base URL %q does not parse: %w", baseURL, err)
	}
	c := &Client{
		release:          release,
		registry:         reg,
		baseURL:          trimmed,
		origin:           origin,
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

// Release reports the FHIR release this client targets, so a caller holding the client through an
// interface can assert the release it will decode into.
func (c *Client) Release() fhir.Release { return c.release }

// applyAuth composes the configured authentication into the client's http.Client transport, exactly
// as the dicomweb client does: the base transport is the one WithRoundTripper installed, else the
// http.Client's own transport (or the TLS-verifying default), and the credential layer (the static
// bearer or a custom layer) wraps the result so every same-origin request carries the scheme's
// header. A caller-supplied *http.Client is never mutated: the credential layer is set on a shallow
// copy, leaving the caller's shared client untouched.
func (c *Client) applyAuth() {
	layer := c.authLayer
	if layer == nil && c.bearerToken != "" {
		layer = bearerAuthLayer(c.bearerToken)
	}
	if layer == nil && c.transport == nil {
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
	if layer != nil {
		base = layer(base, c.origin)
	}
	clientCopy := *c.httpClient
	clientCopy.Transport = base
	c.httpClient = &clientCopy
}
