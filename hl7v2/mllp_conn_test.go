package hl7v2

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
	"sync"
	"testing"
	"time"
)

// selfSignedCert mints an in-memory ECDSA certificate for 127.0.0.1 and returns
// the server certificate plus a pool that trusts it, so the TLS tests verify a
// real handshake without touching the filesystem or the network.
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
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
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

// sampleMessage builds a small, PHI-free ADT^A01 used across the transport
// tests. The values are synthetic placeholders, never real patient data.
func sampleMessage(t *testing.T) *Message {
	t.Helper()
	raw := "MSH|^~\\&|SEND|SFAC|RECV|RFAC|20240101010101||ADT^A01|CTRL1|P|2.5.1\rEVN|A01|20240101010101\r"
	m, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("Parse sample: %v", err)
	}
	return m
}

// startServer binds a loopback server with the default BuildACK handler and
// returns it plus a teardown that shuts it down within a deadline.
func startServer(t *testing.T, opts ...MLLPServerOption) (*Server, func()) {
	t.Helper()
	srv := NewServer(nil, opts...)
	go func() {
		_ = srv.ListenAndServe(context.Background(), "127.0.0.1:0")
	}()
	waitForAddr(t, srv)
	return srv, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	}
}

func waitForAddr(t *testing.T, srv *Server) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if srv.Addr() != nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("server did not bind within the deadline")
}

func TestClientServerRoundTrip(t *testing.T) {
	srv, stop := startServer(t)
	defer stop()

	client, err := NewClient(srv.Addr().String())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ack, err := client.Send(context.Background(), sampleMessage(t))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	msa := requireMSA(t, ack)
	if !msa.AckCode.IsPositive() {
		t.Fatalf("MSA-1 = %q, want a positive code", msa.AckCode)
	}
	// MSA-2 echoes the source MSH-10 so the sender can correlate the reply.
	if msa.ControlID != "CTRL1" {
		t.Fatalf("MSA-2 = %q, want CTRL1", msa.ControlID)
	}
}

// requireMSA extracts the MSA segment from an acknowledgement message via the
// ACK lens, failing the test if the reply is not a well-formed acknowledgement.
func requireMSA(t *testing.T, ack *Message) MSA {
	t.Helper()
	view, ok := AsACK(ack)
	if !ok {
		t.Fatal("reply is not a typed ACK")
	}
	msa, ok := view.MSA()
	if !ok {
		t.Fatal("ack has no MSA segment")
	}
	return msa
}

func TestServerBindsLoopbackByDefault(t *testing.T) {
	srv, stop := startServer(t)
	defer stop()

	host, _, err := net.SplitHostPort(srv.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		t.Fatalf("server bound %s, want a loopback address", host)
	}
}

func TestServerCustomHandler(t *testing.T) {
	// A custom handler can reject deliberately; the server must return whatever
	// the handler decides rather than auto-accept.
	h := HandlerFunc(func(_ context.Context, m *Message) (*Message, error) {
		return m.BuildACK(AckReject, WithACKText("policy rejected"))
	})
	srv := NewServer(h)
	go func() { _ = srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	client, err := NewClient(srv.Addr().String())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ack, err := client.Send(context.Background(), sampleMessage(t))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	msa := requireMSA(t, ack)
	if !msa.AckCode.IsReject() {
		t.Fatalf("MSA-1 = %q, want a reject code", msa.AckCode)
	}
}

func TestClientSendRaw(t *testing.T) {
	srv, stop := startServer(t)
	defer stop()

	client, err := NewClient(srv.Addr().String())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	raw, err := sampleMessage(t).MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	ackBytes, err := client.SendRaw(context.Background(), raw)
	if err != nil {
		t.Fatalf("SendRaw: %v", err)
	}
	ack, err := Parse(ackBytes)
	if err != nil {
		t.Fatalf("Parse ack: %v", err)
	}
	requireMSA(t, ack)
}

func TestServerShutdownIsReRunnable(t *testing.T) {
	srv, _ := startServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("first Shutdown: %v", err)
	}
	// A second Shutdown on an already-stopped server is a clean no-op, never a
	// panic or a hang.
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
}

func TestServerShutdownDrainsOnContextCancel(t *testing.T) {
	// ListenAndServe is started with a context the test cancels; the server must
	// stop accepting and a subsequent Shutdown must report a clean join.
	srv := NewServer(nil)
	serveCtx, cancelServe := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = srv.ListenAndServe(serveCtx, "127.0.0.1:0")
		close(done)
	}()
	waitForAddr(t, srv)

	cancelServe()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ListenAndServe did not return after context cancel")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown after cancel: %v", err)
	}
}

func TestServerConcurrentClients(t *testing.T) {
	srv, stop := startServer(t)
	defer stop()

	const n = 16
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := NewClient(srv.Addr().String())
			if err != nil {
				errCh <- err
				return
			}
			defer client.Close()
			if _, err := client.Send(context.Background(), sampleMessage(t)); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent client: %v", err)
	}
}

func TestClientSendContextCancelled(t *testing.T) {
	srv, stop := startServer(t)
	defer stop()

	client, err := NewClient(srv.Addr().String())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.Send(ctx, sampleMessage(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestClientServerTLSRoundTrip(t *testing.T) {
	cert, pool := selfSignedCert(t)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}

	srv := NewServer(nil, WithServerTLS(serverTLS))
	go func() { _ = srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	clientTLS := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	client, err := NewClient(srv.Addr().String(), WithClientTLS(clientTLS))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ack, err := client.Send(context.Background(), sampleMessage(t))
	if err != nil {
		t.Fatalf("Send over TLS: %v", err)
	}
	requireMSA(t, ack)
}

func TestClientTLSRejectsUntrustedServer(t *testing.T) {
	cert, _ := selfSignedCert(t)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}

	srv := NewServer(nil, WithServerTLS(serverTLS))
	go func() { _ = srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	// NewClient dials eagerly, so an empty RootCAs pool that does not trust the
	// server's self-signed certificate fails the handshake here rather than
	// connecting insecurely and failing later.
	clientTLS := &tls.Config{RootCAs: x509.NewCertPool(), ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
	if _, err := NewClient(srv.Addr().String(), WithClientTLS(clientTLS)); err == nil {
		t.Fatal("NewClient succeeded against an untrusted server certificate, want a TLS error")
	}
}

func TestServerRejectsOversizeFrame(t *testing.T) {
	// A server with a frame cap that admits a normal message but rejects a much
	// larger payload must close the oversize connection with a frame error
	// rather than buffer it, and must not crash the accept loop.
	const cap = 1024
	srv := NewServer(nil, WithMaxFrameSize(cap))
	go func() { _ = srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	conn, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	oversize := make([]byte, 0, cap*2)
	oversize = append(oversize, StartBlock)
	for len(oversize) < cap*2 {
		oversize = append(oversize, 'A') // never an EndBlock, past the cap
	}
	if _, err := conn.Write(oversize); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// The server closes the connection on a frame error; a read returns EOF or an
	// error, never the start of an ACK frame.
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 1)
	if n, _ := conn.Read(buf); n > 0 && buf[0] == StartBlock {
		t.Fatal("server replied with a frame to an oversize input, want the connection closed")
	}

	// The server is still serving: a fresh well-formed client still gets an ACK.
	client, err := NewClient(srv.Addr().String())
	if err != nil {
		t.Fatalf("NewClient after oversize: %v", err)
	}
	defer client.Close()
	if _, err := client.Send(context.Background(), sampleMessage(t)); err != nil {
		t.Fatalf("Send after oversize rejection: %v", err)
	}
}
