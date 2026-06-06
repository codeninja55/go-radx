// Package aws adapts AWS Signature Version 4 (SigV4) request signing to the dicomweb
// client's pluggable authentication seam, for AWS HealthImaging in its traditional
// (SigV4) access mode. HealthImaging signs every request to the medical-imaging service
// with the caller's AWS credentials rather than a static header, so this package exposes
// an http.RoundTripper that derives a fresh per-request signature and stamps the
// Authorization: AWS4-HMAC-SHA256 header before the request leaves.
//
// A caller wires the signer through the core dicomweb.WithRoundTripper option, so the
// AWS SDK stays isolated in this subpackage and never enters the core dicomweb import
// graph:
//
//	cfg, err := config.LoadDefaultConfig(ctx)
//	if err != nil {
//		return err
//	}
//	rt, err := aws.SigV4RoundTripper(cfg, "us-east-1", endpoint, nil)
//	if err != nil {
//		return err
//	}
//	c, err := dicomweb.NewClient(endpoint, dicomweb.WithRoundTripper(rt))
//
// Signing is scoped to the HealthImaging endpoint origin: the transport signs only requests
// whose scheme/host/port match endpoint and forwards any other request UNSIGNED, so a
// cross-origin BulkDataURI in a metadata response can never make the client attach AWS
// credentials to an arbitrary host (matching the core client's credential-scoping guarantee).
//
// AWS HealthImaging's OIDC access mode needs no adapter: it authenticates with a standard
// OAuth2 bearer token, so a caller wires that through the core dicomweb.WithTokenSource
// option instead of this package.
package aws

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// ServiceName is the AWS service identifier HealthImaging signs requests against. It is
// the value the SigV4 credential scope and string-to-sign must carry.
const ServiceName = "medical-imaging"

// emptyPayloadHash is the hex-encoded SHA-256 of the empty string, used as the SigV4
// payload hash when a request has no body.
const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// SigV4RoundTripper returns an http.RoundTripper that signs requests to the HealthImaging
// medical-imaging service in region with AWS Signature Version 4, drawing credentials from
// cfg's provider on each request so a rotating or assumed-role credential is always current.
// endpoint is the HealthImaging base URL the client targets (the same URL passed to
// dicomweb.NewClient); signing is scoped to that origin. base is the underlying transport a
// request is sent on; a nil base uses http.DefaultTransport. It returns an error when
// endpoint is not an absolute URL with a host.
//
// Each in-scope request is signed independently with a current timestamp and a payload hash
// computed over its body, so the signature matches the bytes actually sent and no static
// Authorization header is reused across requests. A request to any other origin — for
// example a cross-origin BulkDataURI a metadata response names — is forwarded UNSIGNED, so
// the caller's AWS credentials never reach a host the operator did not target.
func SigV4RoundTripper(cfg aws.Config, region, endpoint string, base http.RoundTripper) (http.RoundTripper, error) {
	origin, err := originOf(endpoint)
	if err != nil {
		return nil, fmt.Errorf("aws: invalid HealthImaging endpoint: %w", err)
	}
	return &sigV4Transport{
		credentials:   cfg.Credentials,
		region:        region,
		signer:        v4.NewSigner(),
		base:          base,
		now:           time.Now,
		trustedOrigin: origin,
	}, nil
}

// sigV4Transport signs each in-origin request with SigV4 before forwarding it. It clones the
// request so a retried or shared *http.Request is never mutated, and buffers the body so
// the signed payload hash matches the bytes sent. A request to an origin other than
// trustedOrigin is forwarded unsigned.
type sigV4Transport struct {
	credentials   aws.CredentialsProvider
	region        string
	signer        sigV4Signer
	base          http.RoundTripper
	now           func() time.Time
	trustedOrigin string
}

// sigV4Signer is the v4 signer method this transport uses, narrowed to one method so a
// test can substitute a recording double (ISP). It is satisfied by *v4.Signer.
type sigV4Signer interface {
	SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash, service, region string, signingTime time.Time, optFns ...func(*v4.SignerOptions)) error
}

func (t *sigV4Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := t.base
	if transport == nil {
		transport = http.DefaultTransport
	}

	// Sign only requests to the HealthImaging endpoint origin. A request to any other origin
	// (e.g. a cross-origin BulkDataURI named in a metadata response) is forwarded unsigned, so
	// the caller's AWS credentials never reach an untargeted host.
	if reqOrigin, err := originOf(req.URL.String()); err != nil || reqOrigin != t.trustedOrigin {
		return transport.RoundTrip(req)
	}

	creds, err := t.credentials.Retrieve(req.Context())
	if err != nil {
		return nil, fmt.Errorf("aws: retrieve credentials: %w", err)
	}

	signed := req.Clone(req.Context())
	payloadHash, err := bufferBody(signed)
	if err != nil {
		return nil, fmt.Errorf("aws: read request body for signing: %w", err)
	}

	if err := t.signer.SignHTTP(req.Context(), creds, signed, payloadHash, ServiceName, t.region, t.now().UTC()); err != nil {
		return nil, fmt.Errorf("aws: sign request: %w", err)
	}

	return transport.RoundTrip(signed)
}

// originOf returns the scheme://host[:port] origin of rawURL with the default port for the
// scheme normalised away, so two URLs to the same endpoint compare equal regardless of an
// explicit default port. It errors when rawURL is not an absolute URL with a host.
func originOf(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "" || u.Host == "" {
		return "", fmt.Errorf("not an absolute URL: %q", rawURL)
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" || (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		return scheme + "://" + host, nil
	}
	return scheme + "://" + host + ":" + port, nil
}

// bufferBody reads req's body fully, replaces it with a fresh reader over the same bytes
// so the signed request can be sent, and returns the hex SHA-256 of the body. A request
// with no body uses the empty-string hash SigV4 requires.
func bufferBody(req *http.Request) (string, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return emptyPayloadHash, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
