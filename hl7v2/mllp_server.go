package hl7v2

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"
)

// Handler decides the acknowledgement for one inbound MLLP message. Handle
// receives the parsed message and the per-connection context (cancelled on
// Shutdown) and returns the acknowledgement message to frame back, or an error.
// Returning an error closes the connection without replying — used when no
// meaningful ACK can be produced — so a handler that wants to reject deliberately
// returns a rejecting ACK (AR/CR) with a nil error rather than an error. A
// Handler is invoked once per inbound frame and MUST be safe for concurrent use:
// the Server handles each connection on its own goroutine.
type Handler interface {
	Handle(ctx context.Context, m *Message) (*Message, error)
}

// HandlerFunc adapts a function to the Handler interface.
type HandlerFunc func(ctx context.Context, m *Message) (*Message, error)

// Handle calls f.
func (f HandlerFunc) Handle(ctx context.Context, m *Message) (*Message, error) { return f(ctx, m) }

// defaultHandler is the Handler used when NewServer is given a nil handler: it
// parses the inbound message and replies with the BuildACK acceptance. It is the
// turnkey "accept everything" server for a receiver that has no policy of its
// own.
type defaultHandler struct{}

func (defaultHandler) Handle(_ context.Context, m *Message) (*Message, error) {
	return m.BuildACK(AckAccept)
}

// defaultMaxConnections bounds concurrently served inbound connections when
// WithMaxConnections is not set. An MLLP listener exposed to a hospital network is
// a realistic DoS target: a peer that opens connections without sending pins a
// goroutine and a file descriptor each. The cap keeps that growth finite — high
// enough for an interface engine multiplexing many feeds, yet far below the point
// where accept fails for legitimate peers (PRD §9.4). It mirrors dimse's
// WithMaxAssociations bound, scaled up because an MLLP connection is far lighter
// than a DICOM association.
const defaultMaxConnections = 128

// defaultReadTimeout is the per-frame read deadline applied when WithReadTimeout
// is not set. Without it a peer that opens a connection but sends nothing parks a
// goroutine forever, so a flood of idle half-open connections exhausts the
// connection cap and starves legitimate peers. It is deliberately generous — an
// MLLP frame arrives in seconds, never minutes — so it reaps stuck connections
// without cutting off a slow but live sender. An operator who wants no deadline
// passes WithReadTimeout(0) to clear it.
const defaultReadTimeout = 5 * time.Minute

// serverConfig holds the resolved Server options. There is no global mutable
// state (PRD §9.4); every Server carries its own configuration, immutable after
// NewServer.
type serverConfig struct {
	readTimeout    time.Duration
	maxFrameSize   int
	maxConnections int
	tlsConfig      *tls.Config
}

// MLLPServerOption configures a Server at construction. It is named to avoid
// colliding with option types in sibling layers; the public constructors take it
// as a variadic.
type MLLPServerOption func(*serverConfig)

// WithReadTimeout sets the deadline applied to each inbound frame read,
// overriding defaultReadTimeout. It bounds how long a connection goroutine blocks
// on a peer that opens a connection but sends nothing further, so an idle
// half-open connection is reaped rather than parking a goroutine indefinitely. A
// non-positive value clears the deadline (the connection may block until the peer
// sends or disconnects).
func WithReadTimeout(d time.Duration) MLLPServerOption {
	return func(c *serverConfig) { c.readTimeout = d }
}

// WithMaxFrameSize caps the inbound payload the server accumulates before an end
// block, rejecting a larger frame rather than buffering it (PRD §9.3). A
// non-positive value uses DefaultMaxFrameSize.
func WithMaxFrameSize(n int) MLLPServerOption {
	return func(c *serverConfig) {
		if n > 0 {
			c.maxFrameSize = n
		}
	}
}

// WithMaxConnections bounds the number of concurrently served inbound
// connections. A further inbound connection is refused (its transport closed)
// before any handler goroutine is spawned for it, so capacity is enforced ahead of
// work, never after N+1 goroutines already exist. This bounds the goroutine and
// file-descriptor growth a connection flood can cause (PRD §9.4), mirroring
// dimse's WithMaxAssociations. A value <= 0 restores defaultMaxConnections.
func WithMaxConnections(n int) MLLPServerOption {
	return func(c *serverConfig) {
		if n <= 0 {
			n = defaultMaxConnections
		}
		c.maxConnections = n
	}
}

// WithServerTLS terminates TLS on the listener using cfg, so an inbound
// connection completes a TLS handshake before any frame is read (PRD §9.7). The
// library enforces a TLS 1.2 floor: any cfg.MinVersion below 1.2 (unset, or a
// pinned 1.0/1.1) is raised to 1.2; a caller-pinned higher floor is preserved.
// To require client certificates, set cfg.ClientAuth and cfg.ClientCAs. A nil
// cfg leaves the server on plain TCP.
func WithServerTLS(cfg *tls.Config) MLLPServerOption {
	return func(c *serverConfig) { c.tlsConfig = cfg }
}

// Server is an MLLP receiver. It accepts inbound connections, reads one framed
// message at a time per connection, dispatches each to its Handler, and frames
// the returned acknowledgement back on the same connection. It binds to loopback
// by default (PRD §9.1); a non-loopback bind is explicit in the listen address.
// Every goroutine it spawns — the accept loop, its watcher, and each
// per-connection handler — is tracked and joined by Shutdown; there are no
// fire-and-forget goroutines (PRD §9.4). It is safe for concurrent use.
//
// The number of concurrently served connections is capped (WithMaxConnections,
// default defaultMaxConnections): a connection accepted while every slot is in use
// is refused before any goroutine is spawned, so a connection flood cannot exhaust
// goroutines or file descriptors. Each per-frame read carries a deadline
// (WithReadTimeout, default defaultReadTimeout) so an idle half-open connection is
// reaped rather than parking a goroutine indefinitely.
//
// Shutdown cancels the context passed to every in-flight Handler and closes the
// active connections, so a long-running handler that observes its context
// returns promptly on Shutdown.
type Server struct {
	handler Handler
	cfg     serverConfig

	// sem is the capacity semaphore: a slot is acquired before a per-connection
	// goroutine is spawned and released when it returns, so WithMaxConnections
	// bounds concurrency ahead of work rather than after N+1 goroutines exist.
	sem chan struct{}

	mu       sync.Mutex
	listener net.Listener
	conns    map[net.Conn]struct{}
	shutdown bool

	// cancelHandlers cancels the handler context derived in ListenAndServe so a
	// handler observing its context returns promptly on Shutdown.
	cancelHandlers context.CancelFunc

	wg           sync.WaitGroup
	shutdownOnce sync.Once
	// joined is closed by a single waiter goroutine when every tracked goroutine
	// has finished. Every Shutdown selects on it against its own ctx, so repeated
	// deadline-bounded retries share one waiter rather than leaking a goroutine
	// per call.
	joined chan struct{}
}

// NewServer builds an MLLP receiver dispatching each inbound message to h. A nil
// h installs the default handler, which replies with the BuildACK acceptance —
// the turnkey accept-everything receiver. Options configure read timeouts, the
// maximum frame length, and TLS.
func NewServer(h Handler, opts ...MLLPServerOption) *Server {
	cfg := serverConfig{
		readTimeout:    defaultReadTimeout,
		maxFrameSize:   DefaultMaxFrameSize,
		maxConnections: defaultMaxConnections,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if h == nil {
		h = defaultHandler{}
	}
	return &Server{
		handler: h,
		cfg:     cfg,
		sem:     make(chan struct{}, cfg.maxConnections),
		conns:   make(map[net.Conn]struct{}),
		joined:  make(chan struct{}),
	}
}

// ListenAndServe binds addr and serves inbound connections until Shutdown is
// called or ctx is cancelled. The bind defaults to loopback: an address with no
// host (a bare ":port" or "") binds 127.0.0.1, so a non-loopback bind must name
// the interface explicitly (PRD §9.1). Under WithServerTLS the listener
// terminates TLS. It returns nil on a clean Shutdown and a typed error on a bind
// failure. It blocks for the lifetime of the listener.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", mllpLoopbackAddr(addr))
	if err != nil {
		return fmt.Errorf("hl7v2: listen on %q: %w", addr, err)
	}
	if s.cfg.tlsConfig != nil {
		ln = tls.NewListener(ln, tlsConfigWithFloor(s.cfg.tlsConfig))
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
	// Count the accept loop under the SAME lock that publishes the listener,
	// BEFORE Shutdown can observe the listener and call wg.Wait, so no wg.Add is
	// ever a from-zero Add racing a concurrent Wait.
	s.wg.Add(1)
	s.mu.Unlock()

	defer s.wg.Done()
	return s.acceptLoop(handlerCtx, ln)
}

// acceptLoop accepts connections until the listener is closed (Shutdown) or ctx
// is cancelled. For each accepted connection it acquires a capacity slot BEFORE
// spawning the per-connection goroutine: if no slot is available the connection is
// refused (closed) without a goroutine ever being created for it, so a connection
// flood cannot spawn unbounded goroutines. Each served connection is tracked on
// its own tracked goroutine so Shutdown joins it. A tracked watcher closes the
// listener on ctx cancellation so the "serves until ctx is cancelled" contract is
// honoured promptly, not only on the next accept.
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
		conn, err := ln.Accept()
		if err != nil {
			// A closed listener (Shutdown) or a cancelled context is a clean stop.
			if s.isShuttingDown() || ctx.Err() != nil {
				return nil
			}
			// A transient accept error is not fatal; keep serving.
			continue
		}

		// Acquire capacity before spawning. A full semaphore means every slot is in
		// use, so the connection is refused (closed) here — no per-connection
		// goroutine is created for it.
		select {
		case s.sem <- struct{}{}:
		default:
			_ = conn.Close()
			continue
		}

		if !s.trackConn(conn) {
			// Shutdown happened between Accept and tracking; refuse this connection.
			_ = conn.Close()
			<-s.sem
			continue
		}

		// trackConn already did wg.Add(1) under s.mu; this goroutine pairs it.
		go func() {
			defer s.wg.Done()
			defer func() { <-s.sem }()
			defer s.untrackConn(conn)
			s.serveConn(ctx, conn)
		}()
	}
}

// serveConn reads framed messages from conn and replies with the handler's
// acknowledgement, one exchange at a time, until the peer closes the connection,
// a frame error occurs, or ctx is cancelled. A persistent MLLP connection can
// carry several messages, so the loop continues after each ACK until EOF. A
// frame error or handler error closes the connection without crashing the accept
// loop.
func (s *Server) serveConn(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// One buffered reader for the lifetime of the connection: a persistent MLLP
	// stream can carry frames back to back, and a per-frame reader would discard
	// bytes it prefetched for the next frame. Reusing it keeps those bytes.
	br := bufio.NewReader(conn)

	for {
		if s.cfg.readTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.cfg.readTimeout))
		}

		payload, err := ReadFrame(ctx, br, s.cfg.maxFrameSize)
		if err != nil {
			// A peer that closed the connection (io.EOF) is a clean end, not a
			// fault; any other error (framing fault, truncation, timeout,
			// cancellation) ends this connection. None is logged with payload
			// bytes, so no PHI leaks.
			return
		}

		msg, err := Parse(payload)
		if err != nil {
			// A malformed inbound message ends the connection. The connection
			// cannot produce a correlated ACK without a parseable MSH, and the
			// error names only structure, never the bytes.
			return
		}

		ack, err := s.handler.Handle(ctx, msg)
		if err != nil {
			// The handler declined to produce an acknowledgement; close without
			// replying. The error is not logged here (no logger wired into the
			// transport) and never carries PHI by construction.
			return
		}

		raw, err := ack.MarshalText()
		if err != nil {
			return
		}
		if err := WriteFrame(conn, raw); err != nil {
			return
		}
	}
}

// Shutdown stops accepting new connections, cancels the in-flight handler
// context, closes the active connections, and then joins the spawned goroutines
// bounded by ctx. It returns ctx.Err() if they do not finish within the
// deadline, nil once they have actually finished. The teardown runs once and is
// idempotent; the bounded join is re-runnable, so a second Shutdown after a
// deadline reports the real outcome rather than a false nil.
func (s *Server) Shutdown(ctx context.Context) error {
	s.teardownOnce()

	select {
	case <-s.joined:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// teardownOnce performs the idempotent, once-only Shutdown teardown: mark the
// server shutting down, close the listener (no new connection can start), cancel
// the handler context (a handler observing it returns promptly), and close the
// active connections (a handler parked in a frame read wakes). It is separated
// from the bounded wait so a second Shutdown re-attempts the wait without
// re-running these side effects.
func (s *Server) teardownOnce() {
	s.shutdownOnce.Do(func() {
		s.mu.Lock()
		s.shutdown = true
		ln := s.listener
		cancelHandlers := s.cancelHandlers
		conns := make([]net.Conn, 0, len(s.conns))
		for c := range s.conns {
			conns = append(conns, c)
		}
		s.mu.Unlock()

		if ln != nil {
			_ = ln.Close()
		}
		if cancelHandlers != nil {
			cancelHandlers()
		}
		for _, c := range conns {
			_ = c.Close()
		}
		go func() {
			s.wg.Wait()
			close(s.joined)
		}()
	})
}

// Addr reports the network address the server is listening on, or nil before it
// has bound. With a ":0" bind it surfaces the OS-assigned port, so callers (and
// tests) can discover the actual port.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// trackConn records an active connection so Shutdown can close it. It returns
// false when Shutdown has already begun, so the caller refuses the connection
// rather than serving one Shutdown will not join. The per-connection goroutine
// is counted under the SAME lock that records the connection (and that
// Shutdown's teardown snapshots under), so the wg.Add never races Shutdown's
// wg.Wait.
func (s *Server) trackConn(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdown {
		return false
	}
	s.wg.Add(1)
	s.conns[conn] = struct{}{}
	return true
}

// untrackConn removes a connection from the active set once its goroutine ends.
func (s *Server) untrackConn(conn net.Conn) {
	s.mu.Lock()
	delete(s.conns, conn)
	s.mu.Unlock()
}

// isShuttingDown reports whether Shutdown has begun, so the accept loop can
// distinguish a listener closed by Shutdown from a genuine accept fault.
func (s *Server) isShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdown
}

// mllpLoopbackAddr resolves the bind address to loopback by default: an empty
// address or one with no host (a bare ":port") binds 127.0.0.1, so a
// non-loopback bind must name the interface explicitly (PRD §9.1). An address
// that already names a host is returned unchanged.
func mllpLoopbackAddr(addr string) string {
	if addr == "" {
		return "127.0.0.1:0"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" {
		return net.JoinHostPort("127.0.0.1", port)
	}
	return addr
}
