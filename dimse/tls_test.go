package dimse

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

// selfSignedCert mints an in-memory ECDSA certificate for 127.0.0.1 and returns the certificate
// plus a pool that trusts it, so the TLS tests verify a real handshake without touching the
// filesystem or the network. It mirrors the hl7v2 MLLP TLS test helper. The certificate is a CA so
// the same pair serves as both the leaf presented on the wire and the trust anchor, which keeps the
// mutual-TLS setup to one mint per role.
func selfSignedCert(t *testing.T) (tls.Certificate, *x509.CertPool) {
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
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
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
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}
	return cert, pool
}

// startTLSServer builds an SCP whose AE is configured with serverTLS and serves it on loopback,
// returning the listen address. It mirrors startServer but constructs the AE with WithTLS so the
// listener terminates TLS, exercising the real ListenAndServe + tls.NewListener path.
func startTLSServer(t *testing.T, serverTLS *tls.Config, h any) string {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-SCP"), WithTLS(serverTLS))
	if err != nil {
		t.Fatalf("NewAE (SCP): %v", err)
	}
	srv := NewServer(ae, echoAndStorageContexts(), h)
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Error("ListenAndServe did not return after Shutdown")
		}
	})
	return srv.Addr().String()
}

// TestTLSEchoRoundTrip proves a full DIMSE association completes over TLS: an SCU configured with
// WithTLS dials over TLS, verifies the SCP's certificate against a test CA, negotiates, and runs a
// C-ECHO end to end. This is the P0 plaintext-PHI closure: the primary PHI transport now runs over
// a verified TLS channel.
func TestTLSEchoRoundTrip(t *testing.T) {
	cert, pool := selfSignedCert(t)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	addr := startTLSServer(t, serverTLS, &serverTestHandler{echoStatus: StatusEchoSuccess})

	clientTLS := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assoc, err := scu.Associate(ctx, addr, AETitle("RADX-SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate over TLS: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	status, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo over TLS: %v", err)
	}
	if !status.IsSuccess() {
		t.Errorf("Echo over TLS status = %s, want success", status)
	}
}

// TestTLSRejectsUntrustedServer confirms peer-certificate verification is on by default: an SCU
// whose trust pool does not include the SCP's self-signed certificate fails the handshake rather
// than connecting insecurely. The library never sets InsecureSkipVerify, so verification cannot be
// silently skipped.
func TestTLSRejectsUntrustedServer(t *testing.T) {
	cert, _ := selfSignedCert(t)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	addr := startTLSServer(t, serverTLS, &serverTestHandler{echoStatus: StatusEchoSuccess})

	// An empty trust pool does not trust the server's certificate, so the handshake must fail.
	clientTLS := &tls.Config{RootCAs: x509.NewCertPool(), ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := scu.Associate(ctx, addr, AETitle("RADX-SCP"), VerificationContexts()); err == nil {
		t.Fatal("Associate against an untrusted server certificate = nil error, want a TLS handshake error")
	}
}

// TestTLSMutualAuthAcceptsClientCert proves mutual TLS: an SCP requiring client certificates
// (ClientAuth = RequireAndVerifyClientCert) accepts an SCU that presents a certificate the SCP
// trusts, and the association and C-ECHO complete.
func TestTLSMutualAuthAcceptsClientCert(t *testing.T) {
	serverCert, serverPool := selfSignedCert(t)
	clientCert, clientPool := selfSignedCert(t)

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientPool, // trust the SCU's certificate
		MinVersion:   tls.VersionTLS12,
	}
	addr := startTLSServer(t, serverTLS, &serverTestHandler{echoStatus: StatusEchoSuccess})

	clientTLS := &tls.Config{
		Certificates: []tls.Certificate{clientCert}, // present the client certificate
		RootCAs:      serverPool,
		ServerName:   "127.0.0.1",
		MinVersion:   tls.VersionTLS12,
	}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	assoc, err := scu.Associate(ctx, addr, AETitle("RADX-SCP"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate with mutual TLS: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	status, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo with mutual TLS: %v", err)
	}
	if !status.IsSuccess() {
		t.Errorf("Echo with mutual TLS status = %s, want success", status)
	}
}

// TestTLSMutualAuthRejectsMissingClientCert is the counterpart to the accept case: an SCP requiring
// client certificates rejects an SCU that presents none. The SCU trusts the server (the server-auth
// half succeeds), so the only failure is the missing client certificate the SCP demands.
func TestTLSMutualAuthRejectsMissingClientCert(t *testing.T) {
	serverCert, serverPool := selfSignedCert(t)
	_, clientPool := selfSignedCert(t)

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientPool,
		MinVersion:   tls.VersionTLS12,
	}
	addr := startTLSServer(t, serverTLS, &serverTestHandler{echoStatus: StatusEchoSuccess})

	// The SCU trusts the server but presents NO client certificate, so the SCP's
	// RequireAndVerifyClientCert must fail the handshake.
	clientTLS := &tls.Config{RootCAs: serverPool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := scu.Associate(ctx, addr, AETitle("RADX-SCP"), VerificationContexts()); err == nil {
		t.Fatal("Associate without a client certificate = nil error, want the SCP to reject the handshake")
	}
}

// TestTLSRejectsBelowFloor asserts the TLS 1.2 floor: an SCU pinned to a maximum of TLS 1.1 cannot
// handshake with the SCP, whose minimum the library defaults to 1.2 when the config leaves
// MinVersion unset. This proves the floor is enforced on the listener even when the caller did not
// set it.
func TestTLSRejectsBelowFloor(t *testing.T) {
	cert, pool := selfSignedCert(t)
	// Server config leaves MinVersion unset; the library must raise it to TLS 1.2.
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}}
	addr := startTLSServer(t, serverTLS, &serverTestHandler{echoStatus: StatusEchoSuccess})

	// The SCU offers at most TLS 1.1, below the server's enforced 1.2 floor, so the handshake fails.
	clientTLS := &tls.Config{
		RootCAs:    pool,
		ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS10,
		MaxVersion: tls.VersionTLS11,
	}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := scu.Associate(ctx, addr, AETitle("RADX-SCP"), VerificationContexts()); err == nil {
		t.Fatal("Associate with a TLS 1.1 ceiling = nil error, want the 1.2 floor to reject it")
	}
}

// TestTLSFloorDefaultsToTLS12 asserts the library raises an unset MinVersion to TLS 1.2 without
// mutating the caller's config. It inspects the resolved config directly so the floor default is
// verified at the unit level, independent of a live handshake.
func TestTLSFloorDefaultsToTLS12(t *testing.T) {
	caller := &tls.Config{} // MinVersion unset (0)
	ae, err := NewAE(AETitle("RADX-SCU"), WithTLS(caller))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}

	resolved := ae.config().tlsConfigWithFloor()
	if resolved.MinVersion != tls.VersionTLS12 {
		t.Errorf("resolved MinVersion = %#04x, want TLS 1.2 (%#04x)", resolved.MinVersion, tls.VersionTLS12)
	}
	if caller.MinVersion != 0 {
		t.Errorf("caller config was mutated: MinVersion = %#04x, want it left at 0 (clone expected)", caller.MinVersion)
	}

	// A caller-pinned higher floor is preserved, not lowered.
	pinned := &tls.Config{MinVersion: tls.VersionTLS13}
	aePinned, err := NewAE(AETitle("RADX-SCU"), WithTLS(pinned))
	if err != nil {
		t.Fatalf("NewAE (pinned): %v", err)
	}
	if got := aePinned.config().tlsConfigWithFloor().MinVersion; got != tls.VersionTLS13 {
		t.Errorf("resolved MinVersion = %#04x, want the caller-pinned TLS 1.3 (%#04x)", got, tls.VersionTLS13)
	}
}
