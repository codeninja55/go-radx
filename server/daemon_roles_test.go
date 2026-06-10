package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/hl7v2"
)

// TestDaemonMLLPRoundTrip starts a daemon with the MLLP role on loopback, sends one HL7 v2 message
// over MLLP, and asserts the configured handler's acknowledgement comes back, then shuts down cleanly.
func TestDaemonMLLPRoundTrip(t *testing.T) {
	t.Parallel()
	handler := hl7v2.HandlerFunc(func(_ context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
		return m.BuildACK(hl7v2.AckAccept)
	})
	mllpRole, err := NewMLLPRole(handler, WithMLLPPort(0))
	if err != nil {
		t.Fatalf("NewMLLPRole: %v", err)
	}
	d, err := New(WithMLLP(mllpRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "mllp")

	addr := d.Addrs()["mllp"]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := hl7v2.NewClient(addr.String())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	msg, err := hl7v2.Parse([]byte(
		"MSH|^~\\&|SEND|FAC|RECV|FAC|20260101000000||ADT^A01^ADT_A01|MSG0001|P|2.5.1\r" +
			"EVN|A01|20260101000000\r" +
			"PID|1||MRN001^^^HOSP^MR||DOE^JOHN||19700101|M\r" +
			"PV1|1|I\r"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ack, err := client.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if ack == nil {
		t.Fatal("nil acknowledgement from MLLP round-trip")
	}

	cancelRun()
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned %v on clean shutdown, want nil", err)
	}
}

// TestMLLPNonLoopbackRequiresMTLS asserts the MLLP transport-auth fix: a non-loopback MLLP bind
// WITHOUT client-certificate-verifying TLS fails to start with ErrInsecureBind, while the same bind
// WITH mutual TLS starts cleanly (Finding 2). MLLP carries no application-level identity, so a
// generic Authenticator cannot gate it; the bind policy requires mTLS for network exposure.
func TestMLLPNonLoopbackRequiresMTLS(t *testing.T) {
	t.Parallel()
	handler := hl7v2.HandlerFunc(func(_ context.Context, m *hl7v2.Message) (*hl7v2.Message, error) {
		return m.BuildACK(hl7v2.AckAccept)
	})

	// Non-loopback MLLP bind with an Authenticator but NO mTLS: refused with ErrInsecureBind.
	noTLSRole, err := NewMLLPRole(handler, WithMLLPPort(0))
	if err != nil {
		t.Fatalf("NewMLLPRole: %v", err)
	}
	dNoTLS, err := New(WithMLLP(noTLSRole), WithBind("0.0.0.0"), WithAuthenticator(AllowAll()))
	if err != nil {
		t.Fatalf("New (no mTLS): %v", err)
	}
	if err := dNoTLS.Run(context.Background()); !errors.Is(err, ErrInsecureBind) {
		t.Fatalf("Run on a non-loopback MLLP bind without mTLS = %v, want ErrInsecureBind", err)
	}

	// Non-loopback MLLP bind WITH client-certificate-verifying TLS: starts cleanly.
	mtlsRole, err := NewMLLPRole(handler, WithMLLPPort(0))
	if err != nil {
		t.Fatalf("NewMLLPRole (mTLS): %v", err)
	}
	dMTLS, err := New(WithMLLP(mtlsRole), WithBind("0.0.0.0"),
		WithAuthenticator(AllowAll()), WithTLS(mutualTLSConfig(t)))
	if err != nil {
		t.Fatalf("New (mTLS): %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- dMTLS.Run(runCtx) }()
	waitForAddrs(t, dMTLS, "mllp")
	cancelRun()
	if err := <-runErr; err != nil {
		t.Fatalf("Run on a non-loopback MLLP bind with mTLS = %v, want a clean start/stop", err)
	}
}

// mutualTLSConfig builds a server TLS config that requires and verifies a client certificate
// (mTLS), so a non-loopback MLLP bind authenticates the peer at the transport. The CA pool trusts
// the same self-signed certificate the server presents, which is sufficient for the bind-policy
// check (the test does not complete a handshake).
func mutualTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "go-radx-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS12,
	}
}

// TestDaemonDICOMwebServes starts a daemon with the DICOMweb role on loopback and asserts the HTTP
// surface answers under the configured base path (a QIDO-RS studies search against the empty
// catalogue returns a non-5xx response), then shuts down cleanly. It exercises the role's HTTP
// listener, base-path mount, and auth middleware end-to-end.
func TestDaemonDICOMwebServes(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	webRole, err := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	d, err := New(WithDICOMweb(webRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dicomweb")

	addr := d.Addrs()["dicomweb"]
	url := "http://" + addr.String() + "/dicom-web/studies"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Errorf("QIDO-RS studies search returned %d, want a non-5xx response", resp.StatusCode)
	}

	cancelRun()
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned %v on clean shutdown, want nil", err)
	}
}

// serverTLSConfig mints a self-signed server identity for 127.0.0.1 and returns a TLS config
// presenting it plus the pool that trusts it, for the HTTP-role TLS tests that complete a real
// handshake (unlike mutualTLSConfig, which only satisfies the bind-policy check).
func serverTLSConfig(t *testing.T) (*tls.Config, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		IsCA:         true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}},
	}, pool
}

// TestHTTPRoleTLSFloorClampsCallerConfig asserts the TLS 1.2 floor WithTLS documents holds on the
// HTTP roles' listeners even when the caller pins a weaker floor: a daemon whose WithTLS config
// pins MinVersion = TLS 1.0 still rejects a TLS 1.1-limited client, because listen() clamps the
// floor up to 1.2. The control proves the same listener accepts a 1.2 client, so the rejection is
// the version, not a broken fixture.
func TestHTTPRoleTLSFloorClampsCallerConfig(t *testing.T) {
	t.Parallel()
	serverTLS, pool := serverTLSConfig(t)
	// Caller PINS a weak floor (TLS 1.0); listen() must clamp it up to 1.2.
	serverTLS.MinVersion = tls.VersionTLS10

	store, cat := newTestBackends(t)
	webRole, err := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	d, err := New(WithDICOMweb(webRole), WithTLS(serverTLS))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dicomweb")
	defer func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(10 * time.Second):
			t.Error("Run did not return after cancel")
		}
	}()
	addr := d.Addrs()["dicomweb"].String()

	legacy := &tls.Config{
		RootCAs: pool, ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11,
	}
	if conn, err := tls.Dial("tcp", addr, legacy); err == nil {
		_ = conn.Close()
		t.Fatal("a TLS 1.1-limited client completed a handshake with the HTTP role, want the clamped 1.2 floor to reject it")
	}

	// Control: a 1.2 client handshakes against the same listener.
	modern := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, modern)
	if err != nil {
		t.Fatalf("control 1.2 handshake against the clamped listener failed: %v", err)
	}
	_ = conn.Close()
}

// TestHTTPRolesSetReadHeaderTimeout asserts each HTTP role's http.Server carries the shared
// ReadHeaderTimeout, so a slowloris peer trickling header bytes cannot pin connections open
// indefinitely (gosec G112).
func TestHTTPRolesSetReadHeaderTimeout(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	webRole, err := NewDICOMwebRole(store, cat, WithDICOMwebPort(0))
	if err != nil {
		t.Fatalf("NewDICOMwebRole: %v", err)
	}
	repo, err := NewMemoryRepository(fhir.R5)
	if err != nil {
		t.Fatalf("NewMemoryRepository: %v", err)
	}
	fhirRole, err := NewFHIRRole(repo, WithFHIRPort(0), WithFHIRRelease(fhir.R5))
	if err != nil {
		t.Fatalf("NewFHIRRole: %v", err)
	}
	d, err := New(WithDICOMweb(webRole), WithFHIR(fhirRole))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dicomweb")
	waitForAddrs(t, d, fhirRole.name())
	cancelRun()
	select {
	case <-runErr:
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}

	// The srv fields are read only after Run has returned: the roles assign them on the start
	// goroutine, so an in-flight read would race the daemon lifecycle the production code sequences.
	if got := webRole.srv.ReadHeaderTimeout; got != readHeaderTimeout {
		t.Errorf("dicomweb role ReadHeaderTimeout = %v, want %v", got, readHeaderTimeout)
	}
	if got := fhirRole.srv.ReadHeaderTimeout; got != readHeaderTimeout {
		t.Errorf("fhir role ReadHeaderTimeout = %v, want %v", got, readHeaderTimeout)
	}
}
