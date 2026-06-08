package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/codeninja55/go-radx/hl7v2"
)

// defaultMLLPPort is the MLLP listen port when WithMLLPPort is not set. 2575 is the IANA-registered
// HL7 MLLP port.
const defaultMLLPPort = 2575

// mllpRoleConfig holds the resolved MLLPRole options.
type mllpRoleConfig struct {
	port         int
	maxFrameSize int
}

// MLLPRoleOption configures an MLLPRole at construction.
type MLLPRoleOption func(*mllpRoleConfig)

// WithMLLPPort sets the MLLP listen port (default 2575).
func WithMLLPPort(port int) MLLPRoleOption {
	return func(c *mllpRoleConfig) { c.port = port }
}

// WithMaxFrameSize caps the inbound MLLP frame the server accumulates before an end block (default
// hl7v2.DefaultMaxFrameSize). A larger frame is rejected rather than buffered (PRD §9.3).
func WithMaxFrameSize(n int) MLLPRoleOption {
	return func(c *mllpRoleConfig) {
		if n > 0 {
			c.maxFrameSize = n
		}
	}
}

// MLLPRole configures the HL7 v2 MLLP server over an hl7v2.Handler. The handler decides each inbound
// message's acknowledgement; the role applies the daemon's shared bind, TLS, and lifecycle policy.
type MLLPRole struct {
	cfg     mllpRoleConfig
	handler hl7v2.Handler

	srv *hl7v2.Server

	mu    sync.Mutex
	bound net.Addr
}

// NewMLLPRole builds the MLLP role dispatching each inbound message to h. A nil handler is a
// configuration error; pass an explicit accept-everything handler if that is the intent rather than
// relying on a silent default, so the receive policy is always a visible choice.
func NewMLLPRole(h hl7v2.Handler, opts ...MLLPRoleOption) (*MLLPRole, error) {
	if h == nil {
		return nil, errors.New("server: mllp role requires a non-nil hl7v2.Handler")
	}
	cfg := mllpRoleConfig{port: defaultMLLPPort}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &MLLPRole{cfg: cfg, handler: h}, nil
}

func (r *MLLPRole) name() string { return "mllp" }

// start binds the MLLP listener and serves inbound connections in the background. The daemon's shared
// TLS config terminates TLS on the listener (PRD §9.7); the listener is bound synchronously so the
// daemon's fail-closed startup aborts on a bind failure.
//
// MLLP carries no application-level identity, so a generic Authenticator cannot gate it the way the
// HTTP and DIMSE roles gate their callers. The fail-closed bind policy is therefore stronger for
// MLLP: a non-loopback bind MUST terminate mutual TLS (a TLS config that requires and verifies a
// client certificate), so the transport, not the message, authenticates the peer. A non-loopback
// MLLP bind without client-certificate-verifying TLS is refused with ErrInsecureBind rather than
// silently serving every TCP peer that can reach the port.
func (r *MLLPRole) start(ctx context.Context, host string, env roleEnv) error {
	if err := requireMLLPTransportAuth(host, env.tlsConfig); err != nil {
		return err
	}

	var opts []hl7v2.MLLPServerOption
	if r.cfg.maxFrameSize > 0 {
		opts = append(opts, hl7v2.WithMaxFrameSize(r.cfg.maxFrameSize))
	}
	if env.tlsConfig != nil {
		opts = append(opts, hl7v2.WithServerTLS(env.tlsConfig))
	}
	r.srv = hl7v2.NewServer(r.handler, opts...)

	addr := joinHostPort(host, r.cfg.port)
	served := make(chan error, 1)
	go func() { served <- r.srv.ListenAndServe(ctx, addr) }()

	return waitBound(served, func() net.Addr { return r.srv.Addr() }, r.setBound)
}

// setBound records the bound listen address under the role lock so addr() (called concurrently by
// Daemon.Addrs) reads a consistent value.
func (r *MLLPRole) setBound(a net.Addr) {
	r.mu.Lock()
	r.bound = a
	r.mu.Unlock()
}

func (r *MLLPRole) addr() net.Addr {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bound
}

// shutdown stops accepting, cancels in-flight handler contexts, and closes active connections,
// bounded by ctx (a connection still draining at the deadline is force-closed and the daemon names
// the role in the returned error).
func (r *MLLPRole) shutdown(ctx context.Context) error {
	if r.srv == nil {
		return nil
	}
	return r.srv.Shutdown(ctx)
}

// requireMLLPTransportAuth enforces the MLLP-specific bind policy: a loopback bind is unconstrained
// (reachable only from localhost), but a non-loopback bind must terminate mutual TLS — a TLS config
// whose ClientAuth requires and verifies a client certificate — because MLLP has no message-level
// identity for an Authenticator to check. A non-loopback bind with no TLS, or with TLS that does not
// verify a client certificate, is refused with ErrInsecureBind.
func requireMLLPTransportAuth(host string, cfg *tls.Config) error {
	if isLoopbackHost(host) {
		return nil
	}
	if cfg == nil || cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		return fmt.Errorf("%w: MLLP network exposure (bind host %q) requires mutual TLS "+
			"(tls.RequireAndVerifyClientCert), as MLLP has no application-level identity to authenticate",
			ErrInsecureBind, host)
	}
	return nil
}
