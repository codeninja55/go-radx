package dicomweb

import (
	"crypto/tls"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

// Client authentication is a pluggable transport concern: every header-injecting scheme is
// an http.RoundTripper layered over the client's base transport, so a new scheme adds no
// per-request branch to the request path and the cloud adapters (Google ADC, AWS SigV4)
// compose through the same seam without touching the core client (DIP, PRD §8.2). The
// token, password, or certificate an option carries is wired into the transport here and is
// never logged or placed in an error message (PRD §9.8).
//
// Every credential layer is origin-scoped: it injects its credential only when the request
// targets the client's configured origin (matching scheme, host, and port). Server-supplied
// metadata can carry an absolute BulkDataURI on an arbitrary host, and sending the credential
// there would let a malicious or compromised origin harvest the PACS credential, so a
// cross-origin request is sent without credentials (PRD §9.8). Scoping it in the transport
// makes the guard uniform across bearer, basic, and OAuth2 token-source schemes rather than a
// per-call check.

// WithRoundTripper installs a custom http.RoundTripper as the client's transport. It is the
// escape hatch the cloud auth adapters and any bespoke signing scheme build on: the supplied
// transport sees every request and may inject credentials or sign it. It replaces the
// transport wholesale, so combine it with the base TLS policy yourself (for example by
// wrapping http.DefaultTransport); the other auth options layer over the client's existing
// transport instead. A custom RoundTripper enforces its own origin scoping.
func WithRoundTripper(rt http.RoundTripper) ClientOption {
	return func(c *Client) { c.transport = rt }
}

// WithBasicAuth sets HTTP Basic authentication (RFC 7617). The username and password are sent
// as an Authorization: Basic header on every same-origin request and are never logged or
// placed in an error message (PRD §9.8). It layers over the client's existing transport.
func WithBasicAuth(username, password string) ClientOption {
	return func(c *Client) {
		c.authLayer = func(base http.RoundTripper, origin *url.URL) http.RoundTripper {
			inject := func(r *http.Request) { r.SetBasicAuth(username, password) }
			return &credentialTransport{inject: inject, origin: origin, base: base}
		}
	}
}

// WithTokenSource authenticates with an auto-refreshing OAuth2 bearer token drawn from ts. An
// oauth2.TokenSource covers a static token, a refresh-token flow, and a client-credentials
// flow; the token source caches a token until it expires and fetches a fresh one on demand, so
// a long-lived client re-authenticates mid-session without caller involvement. The token is
// attached as an Authorization: Bearer header on same-origin requests and is never logged
// (PRD §9.8). It is the substrate the cloud auth adapters build on.
func WithTokenSource(ts oauth2.TokenSource) ClientOption {
	return func(c *Client) {
		c.authLayer = func(base http.RoundTripper, origin *url.URL) http.RoundTripper {
			// oauth2.Transport injects the bearer for every request; scope it to the origin so
			// a cross-origin reference is fetched without the credential (PRD §9.8).
			return &originScopedTransport{
				sameOrigin:  &oauth2.Transport{Source: ts, Base: base},
				crossOrigin: base,
				origin:      origin,
			}
		}
	}
}

// WithClientCertificate authenticates with a client certificate for mutual TLS. The
// certificate is added to the client's TLS configuration so the transport presents it during
// the handshake; the origin authenticates the client by its certificate rather than a bearer
// credential. It replaces the transport with one carrying the certificate, so apply it before
// WithHTTPClient only when the supplied client is the one to configure.
func WithClientCertificate(cert tls.Certificate) ClientOption {
	return func(c *Client) { c.clientCert = &cert }
}

// bearerAuthLayer wraps a base transport so the static bearer token set by WithBearerToken is
// expressed through the same origin-scoped RoundTripper seam as every other scheme. It is
// built from the stored token rather than at option time so WithBearerToken keeps its existing
// field-set behaviour and order-independence with the other transport options.
func bearerAuthLayer(token string) func(http.RoundTripper, *url.URL) http.RoundTripper {
	return func(base http.RoundTripper, origin *url.URL) http.RoundTripper {
		inject := func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
		return &credentialTransport{inject: inject, origin: origin, base: base}
	}
}

// credentialTransport injects a credential header on each same-origin request, cloning the
// request first so a retried or shared *http.Request is never mutated. A request to a
// different origin is forwarded unmodified, so a server-supplied cross-origin reference never
// carries the credential (PRD §9.8).
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

// originScopedTransport routes a same-origin request through sameOrigin (the credential-bearing
// transport) and a cross-origin request through crossOrigin (the bare base), so a token-source
// credential never leaves the configured origin. It wraps transports that inject on every
// request, such as oauth2.Transport, which has no origin awareness of its own.
type originScopedTransport struct {
	sameOrigin  http.RoundTripper
	crossOrigin http.RoundTripper
	origin      *url.URL
}

func (t *originScopedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if requestSameOrigin(req, t.origin) {
		return t.sameOrigin.RoundTrip(req)
	}
	return baseOrDefault(t.crossOrigin).RoundTrip(req)
}

// requestSameOrigin reports whether req targets origin (matching scheme, host, and port). A
// nil origin (the client base URL did not parse) fails closed: no request is treated as
// same-origin, so no credential is sent.
func requestSameOrigin(req *http.Request, origin *url.URL) bool {
	if origin == nil || req.URL == nil {
		return false
	}
	return sameOrigin(req.URL, origin)
}

// baseOrDefault returns rt, or http.DefaultTransport when rt is nil, so an auth layer wrapping
// a nil base still has a working transport.
func baseOrDefault(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		return http.DefaultTransport
	}
	return rt
}
