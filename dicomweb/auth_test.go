package dicomweb

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// authProbeServer answers every request 500 after recording the Authorization header it
// saw, so a Store/Retrieve produces an error to inspect while the recorded header proves the
// credential reached the origin. It records the last header seen.
func authProbeServer(t *testing.T, seen *string) *httptest.Server {
	t.Helper()
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(hs.Close)
	return hs
}

// TestWithBasicAuthRoundTrips asserts WithBasicAuth attaches an Authorization: Basic header
// the origin sees, and that the credential never appears in the returned error (PRD §9.8).
func TestWithBasicAuthRoundTrips(t *testing.T) {
	const user, pass = "operator", "s3cr3t-password"
	var seen string
	hs := authProbeServer(t, &seen)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()), WithBasicAuth(user, pass))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err == nil {
		t.Fatal("Store against a 500 origin returned nil error")
	}
	if !strings.HasPrefix(seen, "Basic ") {
		t.Fatalf("origin saw Authorization = %q, want a Basic credential", seen)
	}
	// Decode and confirm the credential, then confirm it is absent from the error.
	r, _ := http.NewRequest(http.MethodGet, hs.URL, nil)
	r.SetBasicAuth(user, pass)
	if seen != r.Header.Get("Authorization") {
		t.Fatalf("Basic credential mismatch: saw %q", seen)
	}
	if strings.Contains(err.Error(), pass) {
		t.Fatalf("error leaked the basic-auth password: %q", err.Error())
	}
}

// TestWithBearerTokenAsRoundTripper asserts the re-expressed static bearer still reaches the
// origin and is never logged.
func TestWithBearerTokenAsRoundTripper(t *testing.T) {
	const secret = "static-bearer-token"
	var seen string
	hs := authProbeServer(t, &seen)

	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()), WithBearerToken(secret))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if err == nil {
		t.Fatal("Store against a 500 origin returned nil error")
	}
	if seen != "Bearer "+secret {
		t.Fatalf("origin saw Authorization = %q, want the bearer token", seen)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked the bearer token: %q", err.Error())
	}
}

// refreshingTokenSource is a fake oauth2.TokenSource that hands out a distinct token each
// time it is asked, with an already-elapsed expiry so oauth2.Transport refreshes on every
// request. It lets a test prove the bearer changes mid-session without a real token endpoint.
type refreshingTokenSource struct {
	n atomic.Int64
}

func (s *refreshingTokenSource) Token() (*oauth2.Token, error) {
	v := s.n.Add(1)
	return &oauth2.Token{
		AccessToken: "rotating-token-" + strconv.FormatInt(v, 10),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(-time.Minute), // already expired: force a refresh next call
	}, nil
}

// TestWithTokenSourceRefreshesMidSession asserts WithTokenSource attaches a bearer that
// refreshes between requests: the always-expired fake source yields a new token on each
// request, so two requests carry two distinct bearers.
func TestWithTokenSourceRefreshesMidSession(t *testing.T) {
	var tokens []string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokens = append(tokens, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(hs.Close)

	src := &refreshingTokenSource{}
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()), WithTokenSource(src))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	for range 2 {
		_, _ = c.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	}
	if len(tokens) != 2 {
		t.Fatalf("origin saw %d requests, want 2", len(tokens))
	}
	if tokens[0] == tokens[1] {
		t.Fatalf("token did not refresh mid-session: both requests carried %q", tokens[0])
	}
	for i, tok := range tokens {
		if !strings.HasPrefix(tok, "Bearer rotating-token-") {
			t.Fatalf("request %d carried Authorization = %q, want a rotating bearer", i, tok)
		}
	}
}

// TestWithRoundTripperHonored asserts a custom RoundTripper sees every request: it stamps a
// marker header the origin echoes back through its recorded value.
func TestWithRoundTripperHonored(t *testing.T) {
	var seen string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("X-Custom-Auth")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(hs.Close)

	custom := &markerTransport{base: hs.Client().Transport, marker: "custom-rt-value"}
	c, err := NewClient(hs.URL, WithRoundTripper(custom))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, _ = c.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if seen != "custom-rt-value" {
		t.Fatalf("custom RoundTripper not honored: origin saw X-Custom-Auth = %q", seen)
	}
}

type markerTransport struct {
	base   http.RoundTripper
	marker string
}

func (t *markerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("X-Custom-Auth", t.marker)
	return baseOrDefault(t.base).RoundTrip(r)
}

// TestSharedHTTPClientNotMutated asserts building two authenticated DICOMweb clients over one
// shared *http.Client leaves the first client's credential behaviour intact and gives each
// client its own correct credential. Mutating the shared client (the prior bug) would rewrite
// the first client's transport and double-wrap the second over the first's credential layer,
// so for a same-origin request the inner layer would overwrite the outer Authorization header
// and the second client would send the wrong bearer.
func TestSharedHTTPClientNotMutated(t *testing.T) {
	const tokenA, tokenB = "bearer-for-client-a", "bearer-for-client-b"
	var seen string
	hs := authProbeServer(t, &seen)

	shared := hs.Client()
	baseTransport := shared.Transport

	clientA, err := NewClient(hs.URL, WithHTTPClient(shared), WithBearerToken(tokenA))
	if err != nil {
		t.Fatalf("NewClient A: %v", err)
	}
	// Building the second client over the same shared *http.Client must not disturb the shared
	// client or client A.
	clientB, err := NewClient(hs.URL, WithHTTPClient(shared), WithBearerToken(tokenB))
	if err != nil {
		t.Fatalf("NewClient B: %v", err)
	}

	if shared.Transport != baseTransport {
		t.Fatal("the shared *http.Client's transport was mutated by building DICOMweb clients over it")
	}

	_, _ = clientB.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if seen != "Bearer "+tokenB {
		t.Fatalf("client B sent Authorization = %q, want its own bearer %q", seen, "Bearer "+tokenB)
	}

	// Client A's behaviour is intact: it still sends its own credential, not B's.
	_, _ = clientA.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if seen != "Bearer "+tokenA {
		t.Fatalf("client A sent Authorization = %q, want its own bearer %q", seen, "Bearer "+tokenA)
	}
}

// TestWithClientCertificateMutualTLS asserts WithClientCertificate presents a client
// certificate that a TLS server requiring client auth accepts, so the handshake completes and
// the handler is reached. A self-signed certificate generated in the test acts as both the
// server identity and the trusted client CA, keeping the test self-contained.
func TestWithClientCertificateMutualTLS(t *testing.T) {
	cert, pool := selfSignedCert(t)

	var reached bool
	hs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	hs.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS12,
	}
	hs.StartTLS()
	t.Cleanup(hs.Close)

	base := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}}
	c, err := NewClient(hs.URL, WithHTTPClient(base), WithClientCertificate(cert))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, _ = c.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5"))
	if !reached {
		t.Fatal("mutual-TLS handshake did not reach the handler; client certificate not presented")
	}

	// A client with no certificate against the same server fails the handshake, confirming
	// the server genuinely required client authentication.
	noCert, err := NewClient(hs.URL, WithHTTPClient(&http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := noCert.Store(context.Background(), sampleInstance("1.2.3", "1.2.3.4", "1.2.3.4.5")); err == nil {
		t.Fatal("Store with no client certificate succeeded against a mutual-TLS server")
	}
}

// selfSignedCert returns a self-signed certificate usable as both a server identity and a
// client certificate, plus a pool trusting it. It is generated fresh per test run.
func selfSignedCert(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "go-radx-test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		DNSNames:              []string{"localhost"},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, pool
}
