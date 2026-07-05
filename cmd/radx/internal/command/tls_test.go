package command

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dimse"
)

// tlsTestIdentity is one ephemeral, in-test TLS identity: the PEM file paths the CLI flags
// consume, plus the parsed certificate and a pool trusting it for in-process peers. The
// certificate is minted per test with a one-hour validity and never committed anywhere, so no
// long-lived secret enters the repository (mirrors dimse/tls_test.go selfSignedCert).
type tlsTestIdentity struct {
	certFile string
	keyFile  string
	caFile   string
	cert     tls.Certificate
	pool     *x509.CertPool
}

// newTLSTestIdentity mints a self-signed ECDSA certificate for 127.0.0.1 (a CA, so the same
// pair serves as leaf and trust anchor) and writes its PEM cert/key files under dir.
func newTLSTestIdentity(t *testing.T, dir string) tlsTestIdentity {
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
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatalf("write cert.pem: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatalf("write key.pem: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return tlsTestIdentity{
		certFile: certFile,
		keyFile:  keyFile,
		caFile:   certFile, // self-signed: the leaf is its own trust anchor
		cert:     tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf},
		pool:     pool,
	}
}

// startTLSSCP serves an in-process SCP whose AE terminates TLS with serverTLS, advertising the
// given contexts and dispatching to h. It returns the loopback host and bound port and shuts
// the server down on test cleanup, mirroring the plaintext start*Server helpers.
func startTLSSCP(t *testing.T, serverTLS *tls.Config, contexts []dimse.PresentationContext, h any) (host string, port int) {
	t.Helper()
	ae, err := dimse.NewAE(dimse.AETitle("RADX-SCP"), dimse.WithTLS(serverTLS))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := dimse.NewServer(ae, contexts, h)

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("TLS SCP did not bind within the deadline")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})
	tcp, ok := srv.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("Addr() = %T, want *net.TCPAddr", srv.Addr())
	}
	return "127.0.0.1", tcp.Port
}

// TestEchoTLSRoundTripWithCustomCA is the TLS success golden: a C-ECHO over a verified TLS
// association against an SCP whose certificate chains to the --tls-ca pool exits 0.
func TestEchoTLSRoundTripWithCustomCA(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	serverTLS := &tls.Config{Certificates: []tls.Certificate{id.cert}, MinVersion: tls.VersionTLS12}
	host, port := startTLSSCP(t, serverTLS, dimse.VerificationContexts(), &echoSCPHandler{status: dimse.StatusEchoSuccess})

	stdout, stderr, code := runRadx(t, "echo", "--format", "json",
		"--tls", "--tls-ca", id.caFile,
		"--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.Success {
		t.Fatalf("echo --tls exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	var got echoResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Status != "success" {
		t.Errorf("status = %q, want success", got.Status)
	}
}

// TestEchoTLSUntrustedPeerExits4 pins verification-on-by-default: without --tls-ca the system
// roots do not trust the SCP's self-signed certificate, so the handshake fails closed and echo
// exits 4 (network error), never connecting insecurely.
func TestEchoTLSUntrustedPeerExits4(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	serverTLS := &tls.Config{Certificates: []tls.Certificate{id.cert}, MinVersion: tls.VersionTLS12}
	host, port := startTLSSCP(t, serverTLS, dimse.VerificationContexts(), &echoSCPHandler{status: dimse.StatusEchoSuccess})

	_, _, code := runRadx(t, "echo", "--format", "json", "--tls", "--timeout", "5s",
		"--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.NetworkError {
		t.Fatalf("echo --tls against an untrusted peer exit = %d, want %d", code, exitcode.NetworkError)
	}
}

// TestEchoTLSSkipVerifyConnects confirms the loud escape hatch: --tls-skip-verify accepts the
// untrusted certificate the previous test refused, so the operator's insecure opt-in is explicit.
func TestEchoTLSSkipVerifyConnects(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	serverTLS := &tls.Config{Certificates: []tls.Certificate{id.cert}, MinVersion: tls.VersionTLS12}
	host, port := startTLSSCP(t, serverTLS, dimse.VerificationContexts(), &echoSCPHandler{status: dimse.StatusEchoSuccess})

	_, stderr, code := runRadx(t, "echo", "--format", "json", "--tls", "--tls-skip-verify",
		"--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.Success {
		t.Fatalf("echo --tls --tls-skip-verify exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
}

// TestEchoTLSMutualAuth proves the client-identity pair: an SCP requiring client certificates
// accepts an echo presenting --tls-cert/--tls-key, and refuses one presenting none (exit 4).
func TestEchoTLSMutualAuth(t *testing.T) {
	serverID := newTLSTestIdentity(t, t.TempDir())
	clientID := newTLSTestIdentity(t, t.TempDir())

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{serverID.cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientID.pool,
		MinVersion:   tls.VersionTLS12,
	}
	host, port := startTLSSCP(t, serverTLS, dimse.VerificationContexts(), &echoSCPHandler{status: dimse.StatusEchoSuccess})

	_, stderr, code := runRadx(t, "echo", "--format", "json",
		"--tls", "--tls-ca", serverID.caFile,
		"--tls-cert", clientID.certFile, "--tls-key", clientID.keyFile,
		"--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.Success {
		t.Fatalf("echo with mutual TLS exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}

	_, _, code = runRadx(t, "echo", "--format", "json", "--timeout", "5s",
		"--tls", "--tls-ca", serverID.caFile,
		"--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.NetworkError {
		t.Fatalf("echo without the required client certificate exit = %d, want %d", code, exitcode.NetworkError)
	}
}

// TestTLSOptionsRequireTLSFlag pins the master-switch rule: every TLS option other than --tls
// itself is a usage error without --tls, so a half-configured invocation never silently runs
// plaintext.
func TestTLSOptionsRequireTLSFlag(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	for name, args := range map[string][]string{
		"tls-ca":          {"--tls-ca", id.caFile},
		"tls-cert-key":    {"--tls-cert", id.certFile, "--tls-key", id.keyFile},
		"tls-skip-verify": {"--tls-skip-verify"},
	} {
		full := append([]string{"echo", "--format", "json"}, args...)
		full = append(full, "127.0.0.1", "104")
		if _, _, code := runRadx(t, full...); code != exitcode.UsageError {
			t.Errorf("%s without --tls exit = %d, want %d", name, code, exitcode.UsageError)
		}
	}
}

// TestTLSCertWithoutKeyIsUsageError pins the pair rule for the client identity flags.
func TestTLSCertWithoutKeyIsUsageError(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	if _, _, code := runRadx(t, "echo", "--tls", "--tls-cert", id.certFile, "127.0.0.1", "104"); code != exitcode.UsageError {
		t.Errorf("--tls-cert without --tls-key exit = %d, want %d", code, exitcode.UsageError)
	}
	if _, _, code := runRadx(t, "echo", "--tls", "--tls-key", id.keyFile, "127.0.0.1", "104"); code != exitcode.UsageError {
		t.Errorf("--tls-key without --tls-cert exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestEchoTLSMissingCAFileExits5 confirms a --tls-ca path that cannot be read is a fail-closed
// file-I/O fault before any dial.
func TestEchoTLSMissingCAFileExits5(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-ca.pem")
	_, _, code := runRadx(t, "echo", "--tls", "--tls-ca", missing, "127.0.0.1", "104")
	if code != exitcode.FileIOError {
		t.Fatalf("echo --tls-ca with a missing file exit = %d, want %d", code, exitcode.FileIOError)
	}
}

// TestEchoTLSMalformedCAFileIsUsageError confirms a --tls-ca file with no usable PEM
// certificates is refused before any dial, never a silent empty trust pool.
func TestEchoTLSMalformedCAFileIsUsageError(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "not-a-ca.pem")
	if err := os.WriteFile(bad, []byte("not pem data"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, code := runRadx(t, "echo", "--tls", "--tls-ca", bad, "127.0.0.1", "104")
	if code != exitcode.UsageError {
		t.Fatalf("echo --tls-ca with a malformed file exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestStoreTLSRoundTrip proves the batch storage path over TLS: a C-STORE through the worker
// pool's TLS-dialling associations lands on the TLS SCP and exits 0.
func TestStoreTLSRoundTrip(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	serverTLS := &tls.Config{Certificates: []tls.Certificate{id.cert}, MinVersion: tls.VersionTLS12}
	host, port := startTLSSCP(t, serverTLS, dimse.StorageContexts(), &captureStoreHandler{})
	f := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.5.80")

	stdout, stderr, code := runRadx(t, "store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP",
		"--tls", "--tls-ca", id.caFile, f)
	if code != exitcode.Success {
		t.Fatalf("store --tls exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	if !strings.Contains(stdout, `"status": "success"`) && !strings.Contains(stdout, `"status":"success"`) {
		t.Errorf("store --tls output carries no success line:\n%s", stdout)
	}
}

// TestFindTLSRoundTrip proves the query path over TLS: a C-FIND against a TLS SCP streams its
// matches and exits 0.
func TestFindTLSRoundTrip(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	serverTLS := &tls.Config{Certificates: []tls.Certificate{id.cert}, MinVersion: tls.VersionTLS12}
	host, port := startTLSSCP(t, serverTLS, dimse.QueryRetrieveContexts(), &cannedFindHandler{studyUIDs: []string{"1.2.3.4.5.90"}})

	stdout, stderr, code := runRadx(t, "find", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP",
		"--tls", "--tls-ca", id.caFile,
		"--match", "StudyInstanceUID=")
	if code != exitcode.Success {
		t.Fatalf("find --tls exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	if !strings.Contains(stdout, "1.2.3.4.5.90") {
		t.Errorf("find --tls stream is missing the canned match:\n%s", stdout)
	}
}

// TestScpTLSListenerServesTLSSCU runs the full radx scp command with --tls-cert/--tls-key on a
// free port, verifies a TLS SCU (trusting the test CA) completes a C-ECHO and a C-STORE through
// the TLS listener, then stops the SCP with SIGINT and asserts a clean exit (the serve-test
// signal pattern).
func TestScpTLSListenerServesTLSSCU(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	outDir := t.TempDir()
	port := freeLoopbackPort(t)

	done := make(chan int, 1)
	go func() {
		_, _, code := runRadx(t, "scp", "--format", "json",
			"--port", strconv.Itoa(port), "--output-dir", outDir,
			"--tls-cert", id.certFile, "--tls-key", id.keyFile)
		done <- code
	}()

	clientTLS := &tls.Config{RootCAs: id.pool, MinVersion: tls.VersionTLS12}
	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"), dimse.WithTLS(clientTLS))
	if err != nil {
		t.Fatalf("NewAE (SCU): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Poll until the SCP's TLS listener answers an association (it may still be binding). The
	// echo and store ride separate associations so each proposes one preset's context IDs.
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	var assoc *dimse.Association
	deadline := time.Now().Add(10 * time.Second)
	for {
		assoc, err = scu.Associate(ctx, addr, dimse.AETitle("RADX-SCP"), dimse.VerificationContexts())
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Associate over TLS to radx scp: %v", err)
	}
	echoStatus, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("Echo over TLS: %v", err)
	}
	if !echoStatus.IsSuccess() {
		t.Errorf("Echo status = %s, want success", echoStatus)
	}
	_ = assoc.Release(ctx)

	storeAssoc, err := scu.Associate(ctx, addr, dimse.AETitle("RADX-SCP"), dimse.StorageContexts())
	if err != nil {
		t.Fatalf("Associate (storage) over TLS: %v", err)
	}
	storeStatus, err := storeAssoc.Store(ctx, getInstance("1.2.3.4.5.100"))
	if err != nil {
		t.Fatalf("Store over TLS: %v", err)
	}
	if !storeStatus.IsSuccess() {
		t.Errorf("Store status = %s, want success", storeStatus)
	}
	_ = storeAssoc.Release(ctx)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case code := <-done:
		if code != exitcode.Success {
			t.Errorf("scp exit = %d, want %d after SIGINT", code, exitcode.Success)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("scp did not stop within the deadline after SIGINT")
	}
}

// TestScpTLSCertWithoutKeyIsUsageError pins the server-side pair rule: --tls-cert and
// --tls-key enable the TLS listener together; either alone is a usage error at startup.
func TestScpTLSCertWithoutKeyIsUsageError(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	if _, _, code := runRadx(t, "scp", "--output-dir", t.TempDir(),
		"--tls-cert", id.certFile); code != exitcode.UsageError {
		t.Errorf("scp --tls-cert without --tls-key exit = %d, want %d", code, exitcode.UsageError)
	}
	if _, _, code := runRadx(t, "scp", "--output-dir", t.TempDir(),
		"--tls-key", id.keyFile); code != exitcode.UsageError {
		t.Errorf("scp --tls-key without --tls-cert exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestScpTLSMissingCertFileExits5 confirms a certificate path that cannot be read fails the
// SCP closed at startup as a file-I/O fault, before any bind.
func TestScpTLSMissingCertFileExits5(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	missing := filepath.Join(t.TempDir(), "no-such-cert.pem")
	_, _, code := runRadx(t, "scp", "--output-dir", t.TempDir(),
		"--tls-cert", missing, "--tls-key", id.keyFile)
	if code != exitcode.FileIOError {
		t.Fatalf("scp with a missing --tls-cert exit = %d, want %d", code, exitcode.FileIOError)
	}
}

// TestTLSSkipVerifyWithCAIsUsageError pins the fix: --tls-skip-verify and --tls-ca together are
// contradictory (a pinned CA means "verify against this root", skip-verify means "verify nothing"),
// so the combination is rejected rather than silently nullifying the pin.
func TestTLSSkipVerifyWithCAIsUsageError(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	_, _, code := runRadx(t, "echo", "--tls", "--tls-skip-verify", "--tls-ca", id.caFile,
		"127.0.0.1", "104")
	if code != exitcode.UsageError {
		t.Fatalf("--tls-skip-verify with --tls-ca exit = %d, want %d", code, exitcode.UsageError)
	}
}

// TestTLSSkipVerifyWarnsOnStderr confirms the loud runtime signal: whenever verification is
// actually disabled, the command warns on stderr so an operator cannot silently run insecure.
func TestTLSSkipVerifyWarnsOnStderr(t *testing.T) {
	id := newTLSTestIdentity(t, t.TempDir())
	serverTLS := &tls.Config{Certificates: []tls.Certificate{id.cert}, MinVersion: tls.VersionTLS12}
	host, port := startTLSSCP(t, serverTLS, dimse.VerificationContexts(), &echoSCPHandler{status: dimse.StatusEchoSuccess})

	_, stderr, code := runRadx(t, "echo", "--tls", "--tls-skip-verify",
		"--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.Success {
		t.Fatalf("echo --tls-skip-verify exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "verification") {
		t.Errorf("expected a loud stderr warning about disabled verification, got:\n%s", stderr)
	}
}
