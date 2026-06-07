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

	// A caller-pinned floor BELOW 1.2 (here TLS 1.0) is clamped up to the 1.2 floor, not left as the
	// caller supplied it — the "TLS 1.2+" contract holds even when the caller pinned a weaker floor.
	below := &tls.Config{MinVersion: tls.VersionTLS10}
	aeBelow, err := NewAE(AETitle("RADX-SCU"), WithTLS(below))
	if err != nil {
		t.Fatalf("NewAE (below floor): %v", err)
	}
	if got := aeBelow.config().tlsConfigWithFloor().MinVersion; got != tls.VersionTLS12 {
		t.Errorf("resolved MinVersion = %#04x, want the TLS 1.2 floor (%#04x) clamping the pinned 1.0", got, tls.VersionTLS12)
	}
	if below.MinVersion != tls.VersionTLS10 {
		t.Errorf("caller config was mutated: MinVersion = %#04x, want it left at the pinned 1.0 (clone expected)", below.MinVersion)
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

// TestTLSClampsCallerFloorBelowTLS12 is the live-handshake counterpart to the unit clamp check: an
// SCP whose caller config pins MinVersion = TLS 1.0 still rejects a TLS-1.0-only peer, because the
// library clamps the supplied floor up to TLS 1.2 before terminating TLS. Without the clamp the
// pinned 1.0 floor would let a 1.0 peer negotiate, violating the "TLS 1.2+" contract.
func TestTLSClampsCallerFloorBelowTLS12(t *testing.T) {
	cert, pool := selfSignedCert(t)
	// Caller PINS a weak floor (TLS 1.0); the library must clamp it up to 1.2 on the listener.
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS10}
	addr := startTLSServer(t, serverTLS, &serverTestHandler{echoStatus: StatusEchoSuccess})

	// The SCU offers ONLY TLS 1.0, which the clamped 1.2 server floor must reject.
	clientTLS := &tls.Config{
		RootCAs:    pool,
		ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS10,
		MaxVersion: tls.VersionTLS10,
	}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := scu.Associate(ctx, addr, AETitle("RADX-SCP"), VerificationContexts()); err == nil {
		t.Fatal("Associate with a TLS 1.0-only SCU against a caller-pinned 1.0 server = nil error, " +
			"want the clamped 1.2 floor to reject it")
	}
}

// TestTLSSCUDialBoundsStalledHandshake proves the outbound TLS handshake is bounded by the
// negotiation deadline even when the caller supplies neither a ctx deadline nor a connection
// timeout. The peer accepts TCP but never sends a ServerHello, so the handshake would block forever;
// a short WithACSETimeout must make Associate return a timeout error promptly. The dial is run on a
// background context (no deadline of its own) so the AE's negotiation bound is what does the work.
func TestTLSSCUDialBoundsStalledHandshake(t *testing.T) {
	// A plain TCP listener that accepts the connection and then holds it open without ever speaking
	// TLS: it completes the TCP connect but stalls the handshake (no ServerHello).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		accepted <- c // keep the conn referenced (and open) for the duration of the test
	}()

	_, pool := selfSignedCert(t)
	clientTLS := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS), WithACSETimeout(250*time.Millisecond))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}

	done := make(chan error, 1)
	go func() {
		// context.Background has no deadline, so the AE's ACSE timeout is the only bound on the
		// handshake. A regression (unbounded handshake) would block here past the outer guard.
		_, assocErr := scu.Associate(context.Background(), ln.Addr().String(), AETitle("RADX-SCP"), VerificationContexts())
		done <- assocErr
	}()

	select {
	case assocErr := <-done:
		if assocErr == nil {
			t.Fatal("Associate against a stalled TLS handshake = nil error, want a timeout error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Associate did not return within 5s: the TLS handshake was not bounded by the ACSE timeout")
	}

	select {
	case c := <-accepted:
		_ = c.Close()
	default:
	}
}

// TestTLSSCUDialBoundsStalledHandshakeUnderLongDeadline confirms the configured handshake bound
// applies even when the caller's context already carries a much longer deadline: a stalled TLS
// handshake must not hold the outbound call until that long deadline, only until the ACSE bound.
func TestTLSSCUDialBoundsStalledHandshakeUnderLongDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		accepted <- c
	}()

	_, pool := selfSignedCert(t)
	clientTLS := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS), WithACSETimeout(250*time.Millisecond))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}

	// A parent deadline far longer than the ACSE bound: the handshake must still be capped at the
	// configured bound, not held for the full hour.
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, assocErr := scu.Associate(ctx, ln.Addr().String(), AETitle("RADX-SCP"), VerificationContexts())
		done <- assocErr
	}()

	select {
	case assocErr := <-done:
		if assocErr == nil {
			t.Fatal("Associate against a stalled TLS handshake = nil error, want a timeout error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Associate did not return within 5s: the handshake ignored the configured bound under a long parent deadline")
	}

	select {
	case c := <-accepted:
		_ = c.Close()
	default:
	}
}

// TestTLSSCPAcceptBoundsStalledHandshake proves a stalled accept-side TLS handshake releases its
// association slot end to end. A raw TCP client connects to the TLS SCP but never completes the
// handshake; with WithMaxAssociations(1) it would hold the single slot forever if the accept-side
// handshake were never bounded. A short WithACSETimeout times out the stalled handshake, frees the
// slot, and lets a legitimate TLS client associate after. TestTLSHandshakeAcceptedBoundsStall
// isolates the same bound at the unit level, including the full-handshake (both-direction) coverage
// the explicit HandshakeContext adds over the read deadline the PDU path sets.
func TestTLSSCPAcceptBoundsStalledHandshake(t *testing.T) {
	cert, pool := selfSignedCert(t)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}

	ae, err := NewAE(AETitle("RADX-SCP"), WithTLS(serverTLS), WithACSETimeout(250*time.Millisecond))
	if err != nil {
		t.Fatalf("NewAE (SCP): %v", err)
	}
	srv := NewServer(ae, echoAndStorageContexts(), &serverTestHandler{echoStatus: StatusEchoSuccess}, WithMaxAssociations(1))
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
	addr := srv.Addr().String()

	// A raw TCP client that completes the TCP connect but never starts the TLS handshake, occupying
	// the single association slot until the accept-side handshake times out.
	staller, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial (staller): %v", err)
	}
	defer func() { _ = staller.Close() }()

	// Give the server time to time out the stalled handshake and release the slot. With the slot
	// freed, a legitimate TLS SCU must be able to associate and run a C-ECHO.
	clientTLS := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	scu, err := NewAE(AETitle("RADX-SCU"), WithTLS(clientTLS), WithACSETimeout(2*time.Second))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}

	var assoc *Association
	deadline := time.Now().Add(5 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		assoc, err = scu.Associate(ctx, addr, AETitle("RADX-SCP"), VerificationContexts())
		cancel()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Associate never succeeded after the stalled handshake should have freed the slot: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = assoc.Release(ctx)
	}()

	echoCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	status, err := assoc.Echo(echoCtx)
	if err != nil {
		t.Fatalf("Echo after the slot was freed: %v", err)
	}
	if !status.IsSuccess() {
		t.Errorf("Echo status = %s, want success", status)
	}
}

// TestTLSHandshakeAcceptedBoundsStall isolates the accept-side handshake bound at the unit level: a
// peer that opens the transport but never drives the handshake must not block handshakeAccepted past
// the negotiation deadline. Over a synchronous net.Pipe the server's handshake stalls (the peer
// never speaks TLS); the explicit HandshakeContext, bounded by the AE's negotiation deadline, must
// return a deadline error within the bound rather than hanging. Unlike the read deadline the PDU
// path sets, HandshakeContext covers the whole handshake (both directions) for its duration.
func TestTLSHandshakeAcceptedBoundsStall(t *testing.T) {
	cert, _ := selfSignedCert(t)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}

	// net.Pipe is fully synchronous and unbuffered. The peer (clientSide) never speaks TLS, so the
	// server's handshake blocks; the bound must release it.
	serverSide, clientSide := net.Pipe()
	defer func() { _ = clientSide.Close() }()
	tc := tls.Server(serverSide, serverTLS)

	ae, err := NewAE(AETitle("RADX-SCP"), WithTLS(serverTLS), WithACSETimeout(250*time.Millisecond))
	if err != nil {
		t.Fatalf("NewAE (SCP): %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- ae.handshakeAccepted(context.Background(), tc) }()

	select {
	case hsErr := <-done:
		if hsErr == nil {
			t.Fatal("handshakeAccepted over a stalled handshake = nil error, want a deadline error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handshakeAccepted did not return within 5s: the stalled handshake was not bounded")
	}
	_ = serverSide.Close()
}

// TestTLSHandshakeAcceptedPlaintextNoOp confirms handshakeAccepted leaves a plaintext (non-TLS)
// connection untouched: it is a no-op returning nil, so the plaintext accept path is unchanged.
func TestTLSHandshakeAcceptedPlaintextNoOp(t *testing.T) {
	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	a, b := net.Pipe()
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()
	if err := ae.handshakeAccepted(context.Background(), a); err != nil {
		t.Fatalf("handshakeAccepted on a plaintext conn = %v, want nil (no-op)", err)
	}
}
