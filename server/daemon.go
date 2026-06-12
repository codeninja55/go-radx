package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel/trace"
)

// defaultShutdownTimeout bounds the graceful drain when WithShutdownTimeout is not set. It is
// generous enough for an in-flight C-STORE persisting to disk or an HTTP request completing, yet
// finite so a stuck role cannot block shutdown forever (PRD §9.4).
const defaultShutdownTimeout = 30 * time.Second

// readHeaderTimeout bounds how long an HTTP role waits for a client to finish sending request
// headers, so a slowloris peer trickling header bytes cannot pin connections open indefinitely.
const readHeaderTimeout = 10 * time.Second

// role is the lifecycle contract every mounted server role satisfies. The Daemon drives start and
// graceful shutdown through it uniformly, so adding a role is mounting one more role value, not a
// new branch in the lifecycle. Each implementation wraps one protocol server (dimse.Server,
// dicomweb.Server, hl7v2.Server) and applies the daemon's shared bind, TLS, and observability policy.
type role interface {
	// name is the stable, PHI-free role identifier used in Addrs keys and shutdown error messages
	// (for example "dimse", "dicomweb", "mllp").
	name() string

	// start binds the role's listener at host (the daemon's resolved bind host) and begins serving in
	// the background. It returns once the listener is bound (so Addr is readable) or with a typed
	// error if the bind fails; it does not block for the lifetime of the listener. The shared
	// observability and TLS policy is applied through env before start is called.
	start(ctx context.Context, host string, env roleEnv) error

	// addr reports the bound listen address, or nil before start has bound it.
	addr() net.Addr

	// shutdown gracefully drains the role bounded by ctx, returning ctx.Err() (or a wrapped fault) if
	// it does not drain in time. It is idempotent and safe to call after start returned an error.
	shutdown(ctx context.Context) error
}

// roleEnv carries the shared cross-cutting wiring the daemon hands every role at start: the logger,
// the tracer and meter providers, the TLS config, the resolved authenticator, and the audit hook. A
// role reads only what it needs (a DIMSE role ignores the HTTP authenticator path). None of these
// ever carries PHI.
type roleEnv struct {
	logger    *zap.Logger
	tracer    trace.TracerProvider
	meter     metric.MeterProvider
	tlsConfig *tls.Config
	auth      Authenticator
	audit     AuditFunc
}

// config holds the resolved Daemon options. There is no global mutable state (PRD §9.4); every knob
// is per-Daemon and immutable after New.
type config struct {
	logger    *zap.Logger
	tracer    trace.TracerProvider
	meter     metric.MeterProvider
	bindHost  string
	bindSet   bool
	tlsConfig *tls.Config
	shutdown  time.Duration
	auth      Authenticator
	authSet   bool
	audit     AuditFunc
	roles     []role
}

// Option configures a Daemon at construction.
type Option func(*config)

// WithLogger sets the structured logger the daemon and every role log through. Default: zap.NewNop().
// The logger honours the no-PHI rule — fields carry identifiers and structure, never patient values
// (PRD §9.1, §9.10).
func WithLogger(l *zap.Logger) Option {
	return func(c *config) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithTracerProvider sets the OpenTelemetry tracer provider. Default: the no-op provider, which
// exports nowhere until the operator wires an exporter (PRD §9.10). Span attributes follow the
// no-PHI rule.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *config) {
		if tp != nil {
			c.tracer = tp
		}
	}
}

// WithMeterProvider sets the OpenTelemetry meter provider. Default: the no-op provider. The metric
// set is low-cardinality by construction so a metrics backend cannot become an inadvertent PHI store.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return func(c *config) {
		if mp != nil {
			c.meter = mp
		}
	}
}

// WithBind is the single explicit opt-out of loopback-only binding. host is an interface host
// ("0.0.0.0" or a specific interface IP); it is surfaced to the CLI as --bind. A non-loopback host
// without an explicit Authenticator makes New return ErrInsecureBind (see the bind policy in
// servers.md).
func WithBind(host string) Option {
	return func(c *config) {
		c.bindHost = host
		c.bindSet = true
	}
}

// WithTLS applies cfg to every HTTP and DIMSE-TLS role. The protocol layers enforce a TLS 1.2 floor
// (preferring 1.3) and never set InsecureSkipVerify themselves (PRD §9.7).
func WithTLS(cfg *tls.Config) Option {
	return func(c *config) { c.tlsConfig = cfg }
}

// WithShutdownTimeout bounds the graceful drain (default 30s).
func WithShutdownTimeout(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.shutdown = d
		}
	}
}

// WithAuthenticator sets the Authenticator applied to every role that supports it. It is required to
// bind a non-loopback address (the fail-closed default); on a loopback bind the default is
// AllowAll() because the surface is reachable only from localhost.
//
// A nil Authenticator is treated as NOT set: it does not satisfy the non-loopback bind requirement and
// leaves the safe loopback default in place. Were a nil authenticator to count as "set", a
// non-loopback bind would be accepted with no authenticator at all — the DIMSE role would install no
// association authorizer and the DICOMweb middleware would have no authenticator to consult — which is
// exactly the unauthenticated exposure the fail-closed bind policy exists to prevent (PRD §9.1).
func WithAuthenticator(a Authenticator) Option {
	return func(c *config) {
		if a == nil {
			return
		}
		c.auth = a
		c.authSet = true
	}
}

// WithDIMSE mounts the DIMSE SCP role.
func WithDIMSE(r *DIMSERole) Option {
	return func(c *config) {
		if r != nil {
			c.roles = append(c.roles, r)
		}
	}
}

// WithDICOMweb mounts the DICOMweb HTTP role.
func WithDICOMweb(r *DICOMwebRole) Option {
	return func(c *config) {
		if r != nil {
			c.roles = append(c.roles, r)
		}
	}
}

// WithMLLP mounts the HL7 v2 MLLP role.
func WithMLLP(r *MLLPRole) Option {
	return func(c *config) {
		if r != nil {
			c.roles = append(c.roles, r)
		}
	}
}

// WithFHIR mounts the FHIR REST role. The role wraps a server.Repository over the fhir/r4 or fhir/r5
// package and fixes the release at role construction (one role serves one release; see "FHIR REST
// server" in docs/reference/servers.md). It satisfies the role interface, so the bind policy,
// observability, and lifecycle accept it uniformly — a non-loopback FHIR bind without an
// Authenticator fails closed with ErrInsecureBind exactly like the DICOMweb role.
func WithFHIR(r *FHIRRole) Option {
	return func(c *config) {
		if r != nil {
			c.roles = append(c.roles, r)
		}
	}
}

// Daemon composes one or more server roles behind shared observability and lifecycle policy.
// Construct one per process. Safe for concurrent use after construction.
type Daemon struct {
	cfg config

	mu      sync.Mutex
	started bool          // Run has begun (roles owned by the Run goroutine)
	runDone chan struct{} // closed by Run once its drain completes
	runErr  error         // the drain result, valid once runDone is closed

	shutdownReq chan struct{} // closed by the first Shutdown/Run-cancellation to trigger the drain
	once        sync.Once
}

// New builds a Daemon with the given options. Zero options yield a daemon with loopback-only binds,
// a no-op logger, no-op tracer/meter, and no roles mounted (a daemon with no roles serves nothing
// and Run returns immediately). It returns ErrInsecureBind when a non-loopback WithBind is combined
// with no explicit Authenticator, so the fail-closed default cannot be bypassed by omission.
func New(opts ...Option) (*Daemon, error) {
	cfg := config{
		logger:   zap.NewNop(),
		tracer:   tracenoop.NewTracerProvider(),
		meter:    noop.NewMeterProvider(),
		bindHost: loopbackHost,
		shutdown: defaultShutdownTimeout,
		auth:     AllowAll(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Fail-closed bind policy, applied uniformly across every mounted role: a non-loopback bind
	// without an explicit Authenticator is refused rather than silently exposing an unauthenticated
	// server to the network (PRD §9.1). Passing WithAuthenticator(AllowAll()) is the deliberate,
	// reviewable override.
	if cfg.bindSet && !isLoopbackHost(cfg.bindHost) && !cfg.authSet {
		return nil, fmt.Errorf("%w (bind host %q)", ErrInsecureBind, cfg.bindHost)
	}

	return &Daemon{
		cfg:         cfg,
		runDone:     make(chan struct{}),
		shutdownReq: make(chan struct{}),
	}, nil
}

// Run starts every mounted role and blocks until ctx is cancelled, Shutdown is called, or any role
// fails to start. On ctx cancellation (or Shutdown) it performs a graceful shutdown of all roles
// bounded by the configured shutdown timeout and returns the first non-nil shutdown error, or nil on
// a clean stop. A role that fails to bind aborts startup and Run returns that error after stopping
// any roles already started (no half-started daemon is left running).
func (d *Daemon) Run(ctx context.Context) error {
	d.mu.Lock()
	if d.started {
		d.mu.Unlock()
		return errors.New("server: daemon already started")
	}
	d.started = true
	d.mu.Unlock()

	// Derive a cancellable context the roles serve under; cancelling it on shutdown wakes any role
	// blocked accepting. Each role also receives an explicit shutdown call for a bounded drain.
	serveCtx, cancelServe := context.WithCancel(ctx)
	defer cancelServe()

	env := roleEnv{
		logger:    d.cfg.logger,
		tracer:    d.cfg.tracer,
		meter:     d.cfg.meter,
		tlsConfig: d.cfg.tlsConfig,
		auth:      d.cfg.auth,
		audit:     d.cfg.audit,
	}

	started := make([]role, 0, len(d.cfg.roles))
	for _, r := range d.cfg.roles {
		if err := r.start(serveCtx, d.cfg.bindHost, env); err != nil {
			// Fail-closed composition: a role that cannot bind aborts startup. Drain the roles already
			// started so no half-started daemon is left running (PRD §9.2).
			err = fmt.Errorf("server: role %q failed to start: %w", r.name(), err)
			d.finishRun(d.combineErr(d.drain(started), err))
			return err
		}
		started = append(started, r)
	}

	// A daemon with no roles serves nothing, so there is nothing to drain and no reason to block until
	// cancellation: Run returns immediately, honouring New's documented contract rather than parking on
	// the shutdown wait until the caller's context is cancelled.
	if len(started) == 0 {
		d.finishRun(nil)
		return nil
	}

	d.cfg.logger.Info("daemon started", zap.Int("roles", len(started)))

	// Block until the context is cancelled or a Shutdown is requested, whichever fires first. Run is
	// the single owner of the drain, so a concurrent Shutdown call only signals and then waits on the
	// drain result — start and shutdown never run concurrently on the same role.
	select {
	case <-ctx.Done():
	case <-d.shutdownReq:
	}

	err := d.drain(started)
	d.finishRun(err)
	return err
}

// finishRun records the drain result and wakes any Shutdown caller waiting on it. It runs once; a
// second call (it cannot normally happen, but the guard keeps finishRun idempotent) is a no-op.
func (d *Daemon) finishRun(err error) {
	d.mu.Lock()
	select {
	case <-d.runDone:
		d.mu.Unlock()
		return
	default:
	}
	d.runErr = err
	close(d.runDone)
	d.mu.Unlock()
}

// combineErr returns the first non-nil of two errors, preferring the start failure so Run reports
// why it aborted rather than the consequent drain noise.
func (d *Daemon) combineErr(drainErr, startErr error) error {
	if startErr != nil {
		return startErr
	}
	return drainErr
}

// Shutdown triggers a graceful shutdown independently of Run's context, bounded by ctx. It is safe
// to call concurrently with Run; the first to fire wins and the other observes a clean stop. The
// shutdown is idempotent: a second call after Run has returned is a no-op returning the same result.
// When Run owns the lifecycle, Shutdown signals the drain and waits for Run's drain result (so the
// two never drive a role's start and shutdown concurrently); when Run was never started, Shutdown
// drains the roles itself (each an idempotent no-op on an unstarted role).
func (d *Daemon) Shutdown(ctx context.Context) error {
	d.once.Do(func() { close(d.shutdownReq) })

	d.mu.Lock()
	active := d.started
	roles := d.cfg.roles
	d.mu.Unlock()

	if !active {
		// No Run owns the roles; drain them here (a no-op per unstarted role) so a caller driving the
		// lifecycle without Run still gets a clean, bounded stop.
		return d.drainCtx(ctx, roles)
	}

	// Run owns the drain. Wait for it to finish (it was woken by shutdownReq above), bounded by ctx so
	// a caller's deadline still applies even if Run's own shutdown timeout is longer.
	select {
	case <-d.runDone:
		d.mu.Lock()
		err := d.runErr
		d.mu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Addrs reports the resolved listen address per role after a successful start, so a test that bound
// to ":0" can read the OS-assigned port. It returns nil before Run has bound the listeners. The map
// keys are the stable role names.
func (d *Daemon) Addrs() map[string]net.Addr {
	d.mu.Lock()
	started := d.started
	d.mu.Unlock()
	if !started {
		return nil
	}
	out := make(map[string]net.Addr, len(d.cfg.roles))
	for _, r := range d.cfg.roles {
		if a := r.addr(); a != nil {
			out[r.name()] = a
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// drain shuts every role down within the configured timeout, returning the first non-nil error. A
// role that does not drain in time yields a wrapped ErrShutdownTimeout naming the role.
func (d *Daemon) drain(roles []role) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.shutdown)
	defer cancel()
	return d.drainCtx(ctx, roles)
}

// drainCtx shuts every role down bounded by ctx. Each role is drained in turn; the first failure is
// remembered and reported, but every role is still asked to stop so none is left accepting. A
// deadline-exceeded drain is reported as a role-naming ErrShutdownTimeout, an honest report that the
// stop was not clean (PRD §9.2).
func (d *Daemon) drainCtx(ctx context.Context, roles []role) error {
	var firstErr error
	for _, r := range roles {
		err := r.shutdown(ctx)
		if err == nil {
			continue
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			err = fmt.Errorf("%w: role %q", ErrShutdownTimeout, r.name())
		}
		d.cfg.logger.Warn("role did not drain cleanly", zap.String("role", r.name()))
		if firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
