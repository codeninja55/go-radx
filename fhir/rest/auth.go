package rest

import (
	"net/http"
	"net/url"
	"strings"
)

// Client authentication is a pluggable transport concern: every header-injecting scheme is an
// http.RoundTripper layered over the client's base transport, so a new scheme adds no per-request
// branch to the request path. This mirrors the dicomweb client's auth seam. The token a credential
// option carries is wired into the transport here and is never logged or placed in an error message
// (PRD §9.8).
//
// Every credential layer is origin-scoped: it injects its credential only when the request targets
// the client's configured origin (matching scheme, host, and port). A FHIR response can carry an
// absolute reference (Bundle.entry.fullUrl, a Bundle.link.url) on another host, and sending the
// credential there would let a malicious or compromised origin harvest the client's token, so a
// cross-origin request is sent without credentials (PRD §9.8). Scoping it in the transport makes
// the guard uniform across the bearer and any custom scheme rather than a per-call check.

// WithRoundTripper installs a custom http.RoundTripper as the client's transport. It is the escape
// hatch a SMART on FHIR token source, a mutual-TLS client, or any bespoke signing scheme builds on:
// the supplied transport sees every request and may inject credentials. It replaces the transport
// wholesale, so combine it with the base TLS policy yourself (for example by wrapping
// http.DefaultTransport); WithBearerToken layers over the client's existing transport instead. A
// custom RoundTripper enforces its own origin scoping.
func WithRoundTripper(rt http.RoundTripper) ClientOption {
	return func(c *Client) { c.transport = rt }
}

// bearerAuthLayer wraps a base transport so the static bearer token set by WithBearerToken is
// expressed through the same origin-scoped RoundTripper seam as a custom scheme. It is built from
// the stored token rather than at option time so WithBearerToken keeps its field-set behaviour and
// order-independence with the other transport options.
func bearerAuthLayer(token string) func(http.RoundTripper, *url.URL) http.RoundTripper {
	return func(base http.RoundTripper, origin *url.URL) http.RoundTripper {
		inject := func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
		return &credentialTransport{inject: inject, origin: origin, base: base}
	}
}

// credentialTransport injects a credential header on each same-origin request, cloning the request
// first so a retried or shared *http.Request is never mutated. A request to a different origin is
// forwarded unmodified, so a server-supplied cross-origin reference never carries the credential
// (PRD §9.8).
type credentialTransport struct {
	inject func(*http.Request)
	origin *url.URL
	base   http.RoundTripper
}

func (t *credentialTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !requestSameOrigin(req, t.origin) {
		return baseOrDefault(t.base).RoundTrip(req)
	}
	r := req.Clone(req.Context())
	t.inject(r)
	return baseOrDefault(t.base).RoundTrip(r)
}

// requestSameOrigin reports whether req targets origin (matching scheme, host, and port). A nil
// origin (the client base URL did not parse) fails closed: no request is treated as same-origin, so
// no credential is sent.
func requestSameOrigin(req *http.Request, origin *url.URL) bool {
	if origin == nil || req.URL == nil {
		return false
	}
	return sameOrigin(req.URL, origin)
}

// baseOrDefault returns rt, or http.DefaultTransport when rt is nil, so an auth layer wrapping a nil
// base still has a working transport.
func baseOrDefault(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		return http.DefaultTransport
	}
	return rt
}

// sameOrigin reports whether two URLs share scheme, host, and port. The comparison is
// case-insensitive on scheme and host (per RFC 3986) and normalises the default port so that, for
// example, https without an explicit port matches https on 443.
func sameOrigin(a, b *url.URL) bool {
	if !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	if !strings.EqualFold(a.Hostname(), b.Hostname()) {
		return false
	}
	return originPort(a) == originPort(b)
}

// originPort returns a URL's effective port, substituting the scheme's default when the URL names
// none, so an explicit and an implicit default port compare equal.
func originPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}
