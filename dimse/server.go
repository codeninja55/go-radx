package dimse

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// defaultMaxAssociations bounds concurrent inbound associations when WithMaxAssociations is not
// set. It mirrors the pynetdicom default (AE.maximum_associations = 10): generous for a skeleton
// SCP yet finite, so a flood of inbound connections cannot spawn unbounded goroutines (PRD §9.4).
const defaultMaxAssociations = 10

// serverConfig holds the resolved Server options. There is no global mutable state (PRD §9.4);
// every Server carries its own configuration, immutable after NewServer.
type serverConfig struct {
	maxAssociations        int
	requireCalledAETitle   AETitle
	requireCallingAETitles []AETitle
}

// ServerOption configures a Server at construction.
type ServerOption func(*serverConfig)

// WithMaxAssociations bounds the number of concurrently served inbound associations. A further
// inbound connection is refused (its transport closed) before any handler goroutine is spawned for
// it, so capacity is enforced ahead of work, never after N+1 goroutines already exist (Codex
// DIMSE-013). A value <= 0 restores the default.
func WithMaxAssociations(n int) ServerOption {
	return func(c *serverConfig) {
		if n <= 0 {
			n = defaultMaxAssociations
		}
		c.maxAssociations = n
	}
}

// WithRequireCalledAETitle rejects, at negotiation, any association whose Called AE Title does not
// match t (PS3.8 called-AE-title-not-recognized). Unset (the zero AETitle) accepts any Called AE
// Title the contexts otherwise allow.
func WithRequireCalledAETitle(t AETitle) ServerOption {
	return func(c *serverConfig) { c.requireCalledAETitle = t }
}

// WithRequireCallingAETitles restricts the SCUs the Server serves to the listed Calling AE Titles,
// rejecting any other at negotiation (PS3.8 calling-AE-title-not-recognized). Unset (no titles)
// accepts any Calling AE Title.
func WithRequireCallingAETitles(ts ...AETitle) ServerOption {
	return func(c *serverConfig) { c.requireCallingAETitles = ts }
}

// Server is an embeddable DIMSE SCP. It hosts a Handler, accepts inbound associations, negotiates
// each against the supported presentation contexts, and dispatches received C-ECHO / C-STORE
// operations to the handler. It binds to loopback by default (PRD §9.1); a non-loopback bind is
// explicit in the listen address. Every goroutine it spawns — the accept loop and each
// per-association handler — is tracked and joined by Shutdown; there are no fire-and-forget
// goroutines (PRD §9.4). It is safe for concurrent use.
//
// Shutdown cancels the context passed to every in-flight Handler (Echo/Store) and closes the
// active connections, so a handler MUST observe its context: a handler that selects on ctx.Done()
// (or hands ctx to its I/O) returns promptly on Shutdown. A handler that ignores its context AND is
// not blocked in a connection read cannot be woken — Go cannot forcibly kill a goroutine — so it
// can outlive Shutdown's deadline. Observing the context is therefore the handler's contract.
type Server struct {
	ae        *AE
	supported []acse.SupportedContext
	handler   any
	cfg       serverConfig

	// sem is the capacity semaphore: a slot is acquired before a per-association goroutine is
	// spawned and released when it returns, so WithMaxAssociations bounds concurrency ahead of
	// work (Codex DIMSE-013).
	sem chan struct{}

	mu       sync.Mutex
	listener net.Listener
	conns    map[*dul.Conn]struct{} // active association connections, closed first by Shutdown
	shutdown bool

	// cancelHandlers cancels the handler context derived in ListenAndServe. Shutdown calls it so a
	// handler observing its context returns promptly (cooperative shutdown), rather than only being
	// woken if it happens to be blocked in a connection read (Codex/concurrency review DIMSE-014).
	cancelHandlers context.CancelFunc

	wg           sync.WaitGroup
	shutdownOnce sync.Once
}

// NewServer builds an SCP for the AE, advertising the supported presentation contexts and
// dispatching inbound operations to h. The supported contexts are the abstract syntaxes (and their
// acceptable transfer syntaxes) the Server negotiates as acceptor; a proposed context outside this
// set is rejected at negotiation. Options configure capacity and the AE-title policy.
//
// h MUST implement at least one of EchoHandler / StoreHandler. It is typed as any so a
// service-specific SCP can implement only the narrower capability it offers (interface segregation,
// PRD §8.2) without dummy methods: a store-only SCP implements StoreHandler alone, an echo-only SCP
// EchoHandler alone, and a full SCP both (the Handler union). The dispatcher type-asserts the
// capability per inbound operation; an operation whose capability h does not implement is refused
// with StatusSOPClassNotSupported (a peer-visible RSP), never a panic. A handler implementing
// neither capability is a configuration error: every inbound operation it could be sent is refused.
func NewServer(ae *AE, supported []PresentationContext, h any, opts ...ServerOption) *Server {
	cfg := serverConfig{maxAssociations: defaultMaxAssociations}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Server{
		ae:        ae,
		supported: toSupportedContexts(supported),
		handler:   h,
		cfg:       cfg,
		sem:       make(chan struct{}, cfg.maxAssociations),
		conns:     make(map[*dul.Conn]struct{}),
	}
}

// ListenAndServe binds addr and serves inbound associations until Shutdown is called or ctx is
// cancelled. The bind defaults to loopback: an address with no host (a bare ":port" or "") binds
// 127.0.0.1, so a non-loopback bind must name the interface explicitly (PRD §9.1). It returns nil
// on a clean Shutdown (the listener closed) and a typed error on a bind failure. It blocks for the
// lifetime of the listener.
//
// Every per-association handler runs under a context derived from ctx that Shutdown cancels, so a
// handler observing its context is woken cooperatively on Shutdown (it is NOT the Shutdown context).
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", loopbackAddr(addr))
	if err != nil {
		return fmt.Errorf("dimse: listen on %q: %w", addr, err)
	}

	handlerCtx, cancelHandlers := context.WithCancel(ctx)
	defer cancelHandlers()

	s.mu.Lock()
	if s.shutdown {
		// Shutdown raced ahead of the bind; close immediately and report a clean stop.
		s.mu.Unlock()
		_ = ln.Close()
		return nil
	}
	s.listener = ln
	s.cancelHandlers = cancelHandlers
	s.mu.Unlock()

	return s.acceptLoop(handlerCtx, ln)
}

// acceptLoop accepts connections until the listener is closed (Shutdown) or ctx is cancelled. For
// each accepted connection it acquires a capacity slot BEFORE spawning the per-association
// goroutine (Codex DIMSE-013): if no slot is available the connection is refused (closed) without a
// goroutine ever being created for it. Every spawned goroutine is tracked on the WaitGroup so
// Shutdown joins it. A tracked watcher closes the listener on ctx cancellation so the documented
// "serves until ctx is cancelled" contract is honoured promptly, not only on the next accept.
func (s *Server) acceptLoop(ctx context.Context, ln net.Listener) error {
	watchDone := make(chan struct{})
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case <-ctx.Done():
			_ = ln.Close()
		case <-watchDone:
		}
	}()
	defer close(watchDone)

	for {
		nc, err := ln.Accept()
		if err != nil {
			// A closed listener (Shutdown) or a cancelled context is a clean stop, not a fault.
			if s.isShuttingDown() || ctx.Err() != nil {
				return nil
			}
			// A transient accept error (e.g. a half-open connection) is not fatal; keep serving.
			continue
		}

		// Acquire capacity before spawning. A full semaphore means every slot is in use, so the
		// connection is refused (closed) here — no per-association goroutine is created for it.
		select {
		case s.sem <- struct{}{}:
		default:
			_ = nc.Close()
			continue
		}

		conn := dul.NewConn(nc, s.ae.config().acseTimeout)
		if !s.trackConn(conn) {
			// Shutdown happened between Accept and tracking; refuse this connection.
			_ = conn.Close()
			<-s.sem
			continue
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() { <-s.sem }()
			defer s.untrackConn(conn)
			s.serveConn(ctx, conn)
		}()
	}
}

// serveConn runs one association to completion (negotiate, dispatch, release/abort/close), closing
// the connection when the dispatch returns. Errors are protocol/transport faults already handled by
// the typed error path; the connection close is the terminal action either way.
func (s *Server) serveConn(ctx context.Context, conn *dul.Conn) {
	params := acse.AcceptParams{
		CalledAETitle:          string(s.ae.Title()),
		MaxPDULength:           uint32(s.ae.config().maxPDULength),
		Supported:              s.supported,
		RequireCalledAETitle:   string(s.cfg.requireCalledAETitle),
		RequireCallingAETitles: aeTitleStrings(s.cfg.requireCallingAETitles),
		// Carry the AE's configured implementation identity into the A-ASSOCIATE-AC user
		// information, so an inbound association advertises it just as an outbound SCU does
		// (PS3.7 D.3.3.2); without this the inbound side silently dropped the configured identity.
		ImplementationClassUID: string(s.ae.config().implementationClassUID),
		ImplementationVersion:  s.ae.config().implementationVersion,
	}
	if err := dispatchAssociation(ctx, conn, params, s.ae.config().acseTimeout, s.ae.config().networkTimeout, s.handler); err != nil {
		// The association ended on a fault (rejection, abort, protocol error, or a connection the
		// peer or Shutdown closed under us); the connection is closed below regardless. The error
		// is intentionally not logged here (no logger wired into the skeleton yet) and never
		// carries PHI by construction.
		_ = conn.Close()
		return
	}
	_ = conn.Close()
}

// Shutdown stops accepting new associations and then drives a cooperative, then forced-wake,
// stop of the in-flight handlers, bounded by ctx. It is idempotent: a second Shutdown is a safe
// no-op. It returns ctx.Err() if the handlers do not finish within the deadline, nil otherwise.
//
// The ordering matters. Shutdown sets the shutdown flag, closes the listener (no new association
// can start), then CANCELS the handler context so a handler doing application work — a C-STORE
// persisting to disk, the realistic case — that observes its context returns promptly. It then
// closes the active connections so a handler parked in a DriveInbound/ReadPDU (which the context
// cancellation alone may not interrupt at the socket) is also woken. Finally it joins the tracked
// goroutines, bounded by ctx.
//
// Cancelling the context is the fix for a Shutdown that only closed connections: such a Shutdown
// woke a handler blocked in a socket read but NOT one busy in application work, so it waited out
// the full deadline (Codex/concurrency review DIMSE-014). The dataset already in flight is not
// lost — a handler that observes its context can complete the in-flight store and then return.
//
// A handler that ignores its context AND is not in a connection read cannot be woken (Go cannot
// forcibly kill a goroutine); it can outlive this deadline, and the waiter goroutine outlives
// Shutdown with it until that handler finally returns. Observing the cancelled context is the
// handler's contract (see Server).
func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	s.shutdownOnce.Do(func() {
		s.mu.Lock()
		s.shutdown = true
		ln := s.listener
		cancelHandlers := s.cancelHandlers
		conns := make([]*dul.Conn, 0, len(s.conns))
		for c := range s.conns {
			conns = append(conns, c)
		}
		s.mu.Unlock()

		// Stop accepting first so no new association can start mid-shutdown.
		if ln != nil {
			_ = ln.Close()
		}
		// Cancel the handler context so a handler observing it (doing application work, not blocked
		// in a read) returns promptly — cooperative shutdown (DIMSE-014).
		if cancelHandlers != nil {
			cancelHandlers()
		}
		// Close active connections too, so a handler parked in DriveInbound/ReadPDU wakes and its
		// goroutine can finish; the in-flight dataset is not lost — a cooperative handler completes
		// it and returns.
		for _, c := range conns {
			_ = c.Close()
		}

		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}

// Addr reports the network address the server is listening on, or nil before it has bound. With a
// ":0" bind it surfaces the OS-assigned port, so callers (and tests) can discover the actual port.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// trackConn records an active association connection so Shutdown can close it. It returns false
// when Shutdown has already begun, so the caller refuses the connection rather than serving an
// association that Shutdown will not join.
func (s *Server) trackConn(conn *dul.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdown {
		return false
	}
	s.conns[conn] = struct{}{}
	return true
}

// untrackConn removes a connection from the active set once its association has ended.
func (s *Server) untrackConn(conn *dul.Conn) {
	s.mu.Lock()
	delete(s.conns, conn)
	s.mu.Unlock()
}

// isShuttingDown reports whether Shutdown has begun (the accept loop uses it to distinguish a
// listener closed by Shutdown from a genuine accept fault).
func (s *Server) isShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdown
}

// loopbackAddr resolves the bind address to loopback by default: an empty address or one with no
// host (a bare ":port") binds 127.0.0.1, so a non-loopback bind must name the interface explicitly
// (PRD §9.1). An address that already names a host is returned unchanged.
func loopbackAddr(addr string) string {
	if addr == "" {
		return "127.0.0.1:0"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// Not host:port (e.g. a bare port or a malformed value); let net.Listen report the error,
		// but still default a leading-colon form to loopback.
		return addr
	}
	if host == "" {
		return net.JoinHostPort("127.0.0.1", port)
	}
	return addr
}

// toSupportedContexts translates the public presentation contexts to the acse acceptor-side
// supported set (string UIDs), keeping the layering acyclic: acse never sees the public
// PresentationContext type. Each abstract syntax carries its proposed transfer syntaxes in
// preference order (the acceptor's first match wins).
func toSupportedContexts(contexts []PresentationContext) []acse.SupportedContext {
	out := make([]acse.SupportedContext, 0, len(contexts))
	for _, pc := range contexts {
		ts := make([]string, 0, len(pc.TransferSyntaxes))
		for _, t := range pc.TransferSyntaxes {
			ts = append(ts, string(t))
		}
		out = append(out, acse.SupportedContext{
			AbstractSyntax:   string(pc.AbstractSyntax),
			TransferSyntaxes: ts,
		})
	}
	return out
}

// aeTitleStrings projects AE titles to their string form for the acse AcceptParams.
func aeTitleStrings(titles []AETitle) []string {
	if len(titles) == 0 {
		return nil
	}
	out := make([]string, len(titles))
	for i, t := range titles {
		out[i] = string(t)
	}
	return out
}
