package command

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dimse"
)

// tlsFlags is the TLS flag group shared by every SCU command (echo, store, find, get, move),
// embedded so all five expose the same names and semantics. --tls is the master switch: the
// library enforces a TLS 1.2 floor and peer-certificate verification is ON by default
// (dimse.WithTLS never sets InsecureSkipVerify itself); --tls-skip-verify is the loud,
// deliberate opt-out. Certificate and key material is loaded fail-closed before any dial and
// is never logged (PRD §9.7, §9.8).
type tlsFlags struct {
	TLS           bool   `name:"tls" help:"Connect over TLS (TLS 1.2+; peer certificate verification is on by default)."`
	TLSCA         string `name:"tls-ca" help:"PEM bundle of root CAs to trust instead of the system roots (requires --tls)."`
	TLSCert       string `name:"tls-cert" help:"PEM client certificate for mutual TLS; requires --tls and --tls-key."`
	TLSKey        string `name:"tls-key" help:"PEM private key for --tls-cert; requires --tls."`
	TLSSkipVerify bool   `name:"tls-skip-verify" help:"DANGEROUS: accept any peer certificate without verification, exposing the association to interception. Requires --tls; prefer --tls-ca."`
}

// clientTLSConfig validates the flag combination and builds the SCU TLS configuration, or
// (nil, nil) when --tls is not set. Every load failure is fail-closed before any dial: a
// missing or unreadable file keeps its file-I/O class (exit 5), and a file that parses to no
// usable material is a usage error naming the flag (exit 2). Key material never enters a log.
func (f *tlsFlags) clientTLSConfig() (*tls.Config, error) {
	if !f.TLS {
		switch {
		case f.TLSCA != "":
			return nil, &exitcode.UsageErr{Message: "--tls-ca requires --tls"}
		case f.TLSCert != "" || f.TLSKey != "":
			return nil, &exitcode.UsageErr{Message: "--tls-cert/--tls-key require --tls"}
		case f.TLSSkipVerify:
			return nil, &exitcode.UsageErr{Message: "--tls-skip-verify requires --tls"}
		}
		return nil, nil
	}
	if (f.TLSCert == "") != (f.TLSKey == "") {
		return nil, &exitcode.UsageErr{Message: "--tls-cert and --tls-key must be provided together"}
	}
	if f.TLSSkipVerify && f.TLSCA != "" {
		// A pinned CA and "verify nothing" are contradictory: the pin would be silently ignored.
		// Reject rather than let the operator believe the CA is enforced.
		return nil, &exitcode.UsageErr{Message: "--tls-skip-verify and --tls-ca are mutually exclusive (a pinned CA means verification is on)"}
	}

	// The library raises the floor to 1.2 regardless (dimse.WithTLS); setting it here keeps the
	// config honest if it is ever inspected before the AE applies it.
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if f.TLSCA != "" {
		pool, err := loadCertPool(f.TLSCA)
		if err != nil {
			return nil, err
		}
		cfg.RootCAs = pool
	}
	if f.TLSCert != "" {
		cert, err := loadKeyPair(f.TLSCert, f.TLSKey)
		if err != nil {
			return nil, err
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	if f.TLSSkipVerify {
		cfg.InsecureSkipVerify = true // #nosec G402 -- the loudly-named --tls-skip-verify opt-out; verification is the default
	}
	return cfg, nil
}

// loadCertPool reads a PEM CA bundle into a certificate pool, failing closed on an unreadable
// file (file-I/O class) or a file carrying no usable PEM certificates (usage class).
func loadCertPool(path string) (*x509.CertPool, error) {
	pemBytes, err := os.ReadFile(path) // #nosec G304 -- the operator-named --tls-ca file
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, &exitcode.UsageErr{Message: fmt.Sprintf("--tls-ca %q contains no usable PEM certificates", path)}
	}
	return pool, nil
}

// loadKeyPair loads a PEM certificate/key pair, keeping a missing file in its file-I/O class
// and reporting a malformed pair as a usage error naming the flags. The error text names files
// and structure only, never key material.
func loadKeyPair(certFile, keyFile string) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
			return tls.Certificate{}, err
		}
		return tls.Certificate{}, &exitcode.UsageErr{Message: fmt.Sprintf("--tls-cert/--tls-key pair could not be loaded: %v", err)}
	}
	return cert, nil
}

// resolveClientTLS validates the flags, builds the SCU TLS configuration, and emits a loud runtime
// warning whenever verification is actually disabled, so an operator who passes --tls-skip-verify
// sees an explicit signal rather than silently running an unauthenticated channel. It returns
// (nil, nil) when --tls is not set. Key material and paths are never logged (PRD §9.7, §9.8).
func (f *tlsFlags) resolveClientTLS(log *zap.Logger) (*tls.Config, error) {
	cfg, err := f.clientTLSConfig()
	if err != nil {
		return nil, err
	}
	if f.TLS && f.TLSSkipVerify {
		log.Warn("TLS certificate verification is DISABLED (--tls-skip-verify); the association is exposed to interception - prefer --tls-ca to pin a trusted root")
	}
	return cfg, nil
}

// scuAEOptions assembles the shared AE options every SCU command opens an association with: the
// timeouts, the maximum PDU length, and (when non-nil) the resolved TLS config. Centralising the
// TLS wiring here keeps the security-critical "did TLS actually get applied" decision in one place
// rather than duplicated across echo/store/find/get/move.
func scuAEOptions(timeout time.Duration, maxPDU uint32, tlsCfg *tls.Config) []dimse.AEOption {
	opts := []dimse.AEOption{
		dimse.WithMaxPDULength(dimse.MaxPDULength(maxPDU)),
		dimse.WithACSETimeout(timeout),
		dimse.WithDIMSETimeout(timeout),
		dimse.WithConnectionTimeout(timeout),
	}
	if tlsCfg != nil {
		opts = append(opts, dimse.WithTLS(tlsCfg))
	}
	return opts
}

// serverTLSConfig builds the SCP listener configuration from the scp command's certificate
// pair, with the same fail-closed load classes as the client side. Both files are required
// together; the caller enforces that pairing as a usage rule.
func serverTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := loadKeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}, nil
}
