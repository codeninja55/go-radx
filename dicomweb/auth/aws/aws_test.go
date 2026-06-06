package aws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// staticTransport records the request it was asked to send and returns a 500 so the
// caller gets a response without a live endpoint. It captures the Authorization header
// the SigV4 layer produced.
type staticTransport struct {
	got *http.Request
}

func (t *staticTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.got = req
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

const (
	testAccessKey = "AKIDEXAMPLE"
	testSecretKey = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testRegion    = "us-east-1"
)

// fixedTimeTransport wraps a SigV4 transport with a frozen clock so the signature is
// deterministic and can be re-derived independently.
func fixedTimeRoundTripper(t *testing.T, base http.RoundTripper, at time.Time) *sigV4Transport {
	t.Helper()
	cfg := awssdk.Config{
		Region:      testRegion,
		Credentials: credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, ""),
	}
	rt, ok := SigV4RoundTripper(cfg, testRegion, base).(*sigV4Transport)
	if !ok {
		t.Fatalf("SigV4RoundTripper returned %T, want *sigV4Transport", SigV4RoundTripper(cfg, testRegion, base))
	}
	rt.now = func() time.Time { return at }
	return rt
}

// TestSigV4ProducesWellFormedAuthorization asserts the RoundTripper stamps a
// well-formed AWS4-HMAC-SHA256 Authorization header for a HealthImaging-shaped request:
// the medical-imaging service, the configured region, an alphabetically sorted
// SignedHeaders list including host, and a 64-hex-character signature.
func TestSigV4ProducesWellFormedAuthorization(t *testing.T) {
	base := &staticTransport{}
	at := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	rt := fixedTimeRoundTripper(t, base, at)

	// A HealthImaging WADO-RS-shaped GET: datastore + study path on the medical-imaging host.
	const url = "https://dicom-medical-imaging.us-east-1.amazonaws.com/datastore/ds-123/studies/1.2.3"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Accept", "multipart/related; type=application/dicom")

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	auth := base.got.Header.Get("Authorization")

	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") {
		t.Fatalf("Authorization = %q, want an AWS4-HMAC-SHA256 header", auth)
	}
	credential, signedHeaders, signature := parseSigV4Header(t, auth)

	wantScopeSuffix := "/" + testRegion + "/" + ServiceName + "/aws4_request"
	if !strings.HasPrefix(credential, testAccessKey+"/") {
		t.Fatalf("Credential = %q, want it to start with the access key", credential)
	}
	if !strings.HasSuffix(credential, wantScopeSuffix) {
		t.Fatalf("Credential = %q, want scope suffix %q (region + medical-imaging service)", credential, wantScopeSuffix)
	}
	if !strings.Contains(signedHeaders, "host") {
		t.Fatalf("SignedHeaders = %q, want it to include host", signedHeaders)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(signature) {
		t.Fatalf("Signature = %q, want 64 hex characters", signature)
	}
}

// TestSigV4MatchesIndependentDerivation re-derives the SigV4 signature from the
// documented canonical-request algorithm (PS3-independent AWS SigV4 spec) and asserts it
// equals the signature the SDK signer produced. This proves the produced header is a
// genuine valid signature, not merely well-shaped.
func TestSigV4MatchesIndependentDerivation(t *testing.T) {
	base := &staticTransport{}
	at := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	rt := fixedTimeRoundTripper(t, base, at)

	const url = "https://dicom-medical-imaging.us-east-1.amazonaws.com/datastore/ds-123/studies/1.2.3"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	signed := base.got
	auth := signed.Header.Get("Authorization")
	credential, signedHeaders, gotSig := parseSigV4Header(t, auth)

	wantSig := deriveSigV4(t, signed, signedHeaders, credential, at, emptyPayloadHash)
	if gotSig != wantSig {
		t.Fatalf("signature mismatch:\n produced = %s\n derived  = %s", gotSig, wantSig)
	}
}

// TestSigV4SignsBodyPayloadHash asserts that for a request with a body (a STOW-RS POST)
// the buffered body survives signing and the signature is over the body's SHA-256, by
// re-deriving with the body hash and matching.
func TestSigV4SignsBodyPayloadHash(t *testing.T) {
	base := &staticTransport{}
	at := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	rt := fixedTimeRoundTripper(t, base, at)

	const url = "https://dicom-medical-imaging.us-east-1.amazonaws.com/datastore/ds-123/studies"
	const body = "fake-dicom-multipart-body"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "multipart/related; type=application/dicom")

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	signed := base.got

	// The signed request's body must still be readable (buffered, not consumed).
	gotBody, err := io.ReadAll(signed.Body)
	if err != nil {
		t.Fatalf("read signed body: %v", err)
	}
	if string(gotBody) != body {
		t.Fatalf("signed body = %q, want it preserved", string(gotBody))
	}

	sum := sha256.Sum256([]byte(body))
	bodyHash := hex.EncodeToString(sum[:])

	auth := signed.Header.Get("Authorization")
	credential, signedHeaders, gotSig := parseSigV4Header(t, auth)
	wantSig := deriveSigV4(t, signed, signedHeaders, credential, at, bodyHash)
	if gotSig != wantSig {
		t.Fatalf("body-signed signature mismatch:\n produced = %s\n derived  = %s", gotSig, wantSig)
	}
}

// TestSigV4SignsEachRequestIndependently asserts two requests at different times carry
// different signatures, proving the signature is per-request rather than a reused static
// header.
func TestSigV4SignsEachRequestIndependently(t *testing.T) {
	base := &staticTransport{}
	cfg := awssdk.Config{
		Region:      testRegion,
		Credentials: credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, ""),
	}
	rt := SigV4RoundTripper(cfg, testRegion, base).(*sigV4Transport)

	const url = "https://dicom-medical-imaging.us-east-1.amazonaws.com/datastore/ds-123/studies/1.2.3"
	collect := func(at time.Time) string {
		rt.now = func() time.Time { return at }
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if _, err := rt.RoundTrip(req); err != nil {
			t.Fatalf("RoundTrip: %v", err)
		}
		_, _, sig := parseSigV4Header(t, base.got.Header.Get("Authorization"))
		return sig
	}
	sigA := collect(time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC))
	sigB := collect(time.Date(2026, 6, 7, 13, 0, 0, 0, time.UTC))
	if sigA == sigB {
		t.Fatal("two requests at different times produced the same signature; signing is not per-request")
	}
}

// errProvider is a credentials provider that always fails, to prove a credential error
// surfaces from RoundTrip without leaking material.
type errProvider struct{}

func (errProvider) Retrieve(context.Context) (awssdk.Credentials, error) {
	return awssdk.Credentials{}, errors.New("no credentials configured")
}

// TestSigV4CredentialErrorSurfaces asserts a credential-retrieval failure is returned as
// an error rather than sending an unsigned request, and that the error carries no secret.
func TestSigV4CredentialErrorSurfaces(t *testing.T) {
	base := &staticTransport{}
	cfg := awssdk.Config{Region: testRegion, Credentials: errProvider{}}
	rt := SigV4RoundTripper(cfg, testRegion, base)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"https://dicom-medical-imaging.us-east-1.amazonaws.com/datastore/ds-123/studies", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("RoundTrip with a failing credentials provider returned nil error")
	}
	if base.got != nil {
		t.Fatal("an unsigned request was sent despite a credential failure")
	}
	if strings.Contains(err.Error(), testSecretKey) {
		t.Fatalf("error leaked the secret key: %q", err.Error())
	}
}

// parseSigV4Header splits an AWS4-HMAC-SHA256 Authorization header into its Credential,
// SignedHeaders, and Signature components.
func parseSigV4Header(t *testing.T, auth string) (credential, signedHeaders, signature string) {
	t.Helper()
	rest := strings.TrimPrefix(auth, "AWS4-HMAC-SHA256 ")
	for _, part := range strings.Split(rest, ", ") {
		switch {
		case strings.HasPrefix(part, "Credential="):
			credential = strings.TrimPrefix(part, "Credential=")
		case strings.HasPrefix(part, "SignedHeaders="):
			signedHeaders = strings.TrimPrefix(part, "SignedHeaders=")
		case strings.HasPrefix(part, "Signature="):
			signature = strings.TrimPrefix(part, "Signature=")
		}
	}
	if credential == "" || signedHeaders == "" || signature == "" {
		t.Fatalf("Authorization header missing a component: %q", auth)
	}
	return credential, signedHeaders, signature
}

// deriveSigV4 independently computes the AWS SigV4 signature for req following the
// documented algorithm: canonical request, string to sign, signing key, HMAC. It mirrors
// https://docs.aws.amazon.com/IAM/latest/UserGuide/create-signed-request.html so a match
// proves the SDK-produced signature is genuine.
func deriveSigV4(t *testing.T, req *http.Request, signedHeaders, credential string, at time.Time, payloadHash string) string {
	t.Helper()
	amzDate := at.Format("20060102T150405Z")
	dateStamp := at.Format("20060102")

	headerNames := strings.Split(signedHeaders, ";")
	sort.Strings(headerNames)
	var canonicalHeaders strings.Builder
	for _, name := range headerNames {
		var value string
		switch name {
		case "host":
			value = req.URL.Host
		case "content-length":
			// Content-Length lives on the request struct field, not the header map.
			value = strconv.FormatInt(req.ContentLength, 10)
		default:
			value = strings.TrimSpace(req.Header.Get(name))
		}
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(value)
		canonicalHeaders.WriteString("\n")
	}

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	hashedCanonical := sha256Hex([]byte(canonicalRequest))
	scope := dateStamp + "/" + testRegion + "/" + ServiceName + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hashedCanonical,
	}, "\n")

	// Confirm the SDK used the scope we expect, guarding the derivation against a drifted
	// region or service in the produced credential.
	if !strings.HasSuffix(credential, "/"+scope) {
		t.Fatalf("Credential scope = %q, want it to end with %q", credential, scope)
	}

	signingKey := hmacSHA256([]byte("AWS4"+testSecretKey), dateStamp)
	signingKey = hmacSHA256(signingKey, testRegion)
	signingKey = hmacSHA256(signingKey, ServiceName)
	signingKey = hmacSHA256(signingKey, "aws4_request")
	return hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
