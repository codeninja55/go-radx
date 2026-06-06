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
//	rt := aws.SigV4RoundTripper(cfg, "us-east-1", nil)
//	c, err := dicomweb.NewClient(endpoint, dicomweb.WithRoundTripper(rt))
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

// SigV4RoundTripper returns an http.RoundTripper that signs every request with AWS
// Signature Version 4 for the HealthImaging medical-imaging service in region, drawing
// credentials from cfg's provider on each request so a rotating or assumed-role
// credential is always current. base is the underlying transport the signed request is
// sent on; a nil base uses http.DefaultTransport.
//
// Each request is signed independently with a current timestamp and a payload hash
// computed over its body, so the signature matches the bytes actually sent and no static
// Authorization header is reused across requests.
func SigV4RoundTripper(cfg aws.Config, region string, base http.RoundTripper) http.RoundTripper {
	return &sigV4Transport{
		credentials: cfg.Credentials,
		region:      region,
		signer:      v4.NewSigner(),
		base:        base,
		now:         time.Now,
	}
}

// sigV4Transport signs each request with SigV4 before forwarding it. It clones the
// request so a retried or shared *http.Request is never mutated, and buffers the body so
// the signed payload hash matches the bytes sent.
type sigV4Transport struct {
	credentials aws.CredentialsProvider
	region      string
	signer      sigV4Signer
	base        http.RoundTripper
	now         func() time.Time
}

// sigV4Signer is the v4 signer method this transport uses, narrowed to one method so a
// test can substitute a recording double (ISP). It is satisfied by *v4.Signer.
type sigV4Signer interface {
	SignHTTP(ctx context.Context, credentials aws.Credentials, r *http.Request, payloadHash, service, region string, signingTime time.Time, optFns ...func(*v4.SignerOptions)) error
}

func (t *sigV4Transport) RoundTrip(req *http.Request) (*http.Response, error) {
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

	transport := t.base
	if transport == nil {
		transport = http.DefaultTransport
	}
	return transport.RoundTrip(signed)
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
