package rest_test

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
	"strings"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/rest"
)

// legacyTLSOrigin simulates a legacy FHIR origin limited to TLS 1.1: a raw TLS listener that
// completes handshakes and discards connections, with a self-signed certificate and the pool
// that trusts it for the control probe.
func legacyTLSOrigin(t *testing.T) (net.Listener, *x509.CertPool) {
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

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS10,
		MaxVersion:   tls.VersionTLS11,
	})
	if err != nil {
		t.Fatalf("tls.Listen (legacy origin): %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.(*tls.Conn).Handshake()
			_ = conn.Close()
		}
	}()
	return ln, pool
}

// TestClientTLSFloorRefusesDowngradedOrigin asserts the client's default transport enforces the
// TLS 1.2 floor: a request to an origin limited to TLS 1.1 fails the handshake with a protocol-
// version error (never a downgraded connection). The control probe proves the legacy origin
// genuinely negotiates 1.1 with a willing peer, so the failure is the client's floor, not a
// broken fixture; matching the protocol-version error distinguishes the floor from a certificate
// failure.
func TestClientTLSFloorRefusesDowngradedOrigin(t *testing.T) {
	ln, pool := legacyTLSOrigin(t)

	// Control: a raw 1.1-willing dialer handshakes with the legacy origin.
	probe, err := tls.Dial("tcp", ln.Addr().String(), &tls.Config{
		RootCAs: pool, ServerName: "127.0.0.1",
		MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11,
	})
	if err != nil {
		t.Fatalf("control 1.1 handshake against the legacy origin failed: %v", err)
	}
	_ = probe.Close()

	c, err := rest.NewClient(fhir.R5, "https://"+ln.Addr().String())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = c.Read(context.Background(), "Patient", "example")
	if err == nil {
		t.Fatal("Read against a TLS 1.1-limited origin succeeded, want the 1.2 floor to reject the handshake")
	}
	if !strings.Contains(err.Error(), "protocol version") {
		t.Fatalf("error = %v, want a TLS protocol-version failure (not, e.g., a certificate failure)", err)
	}
}
