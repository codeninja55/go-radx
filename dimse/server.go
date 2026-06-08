package dimse

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
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
	// moveDestinations maps a C-MOVE Move Destination AE Title to its network address ("host:port").
	// The C-MOVE SCP resolves the destination through this table, opens an outbound association to
	// it, and C-STOREs each matched instance there as a sub-operation. A destination not in the table
	// is answered with the terminal 0xA801 "Move Destination Unknown" status (PS3.4 C.4.2.1.5). It is
	// nil when no destinations are configured, so a Server with no C-MOVE support refuses every move.
	moveDestinations map[AETitle]string
	// getStorageRoles lists the Storage SOP Classes the C-GET SCP grants the requestor the Storage SCP
	// role for (PS3.7 D.3.3.4). A C-GET C-STOREs each matched instance back to the requestor on the
	// same association, so the requestor must take the Storage SCP role to receive them — which the
	// acceptor only grants for a SOP Class it declares here. It is nil when no roles are configured, so
	// a Server with no C-GET support grants no SCP role and same-association C-GET cannot proceed.
	getStorageRoles []dicom.SOPClassUID
	// associationAuthorizer, when non-nil, authorizes each inbound association by its Calling AE Title
	// and peer address at negotiation: a non-nil return rejects the association with an
	// A-ASSOCIATE-RJ before any service runs, so an exposed SCP can enforce caller identity rather
	// than serving every peer the contexts otherwise allow. It is nil by default (every association
	// the AE-title policy admits is served).
	associationAuthorizer func(callingAE string, peer net.Addr) error
	// authenticator, when non-nil, authenticates the user-identity sub-item a requestor presents
	// (PS3.7 D.3.3.7) at negotiation: a non-nil error rejects the association with an A-ASSOCIATE-RJ
	// before any service runs; a nil error returns the optional positive-response bytes echoed back in
	// the A-ASSOCIATE-AC. It is the same negotiation seam as associationAuthorizer, extended to the
	// identity DICOM itself negotiates. It is nil by default (no user-identity authentication: an
	// identity a peer presents is accepted without challenge).
	authenticator func(id *UserIdentity, peer net.Addr) ([]byte, error)
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

// WithMoveDestinations configures the known-AE table the C-MOVE SCP resolves a Move Destination AE
// Title against: each entry maps a destination AE Title to its network address ("host:port"). On a
// C-MOVE-RQ the SCP looks up the requested destination here, opens an outbound association to it, and
// C-STOREs each matched instance there as a sub-operation. A destination absent from the table is
// answered with the terminal 0xA801 "Move Destination Unknown" status, never a panic. It copies the
// map so a later caller mutation cannot change the Server's resolution (no shared mutable state, PRD
// §9.4); a nil or empty map leaves the Server with no C-MOVE destinations (every move is refused as
// unknown).
func WithMoveDestinations(dests map[AETitle]string) ServerOption {
	return func(c *serverConfig) {
		if len(dests) == 0 {
			c.moveDestinations = nil
			return
		}
		copied := make(map[AETitle]string, len(dests))
		for aet, addr := range dests {
			copied[aet] = addr
		}
		c.moveDestinations = copied
	}
}

// WithGetStorageRoles declares the Storage SOP Classes the C-GET SCP grants the requestor the Storage
// SCP role for at negotiation (PS3.7 D.3.3.4). A C-GET differs from a C-MOVE in that the matched
// instances are C-STOREd back to the requestor on the SAME association, so the requestor must take the
// Storage SCP role to receive them; the acceptor grants that role only for a SOP Class declared here
// (and only when the requestor proposed it). Pass the Storage SOP Classes the Server's C-GET SCP may
// return — typically the same set the Server advertises as Storage contexts. It copies the slice so a
// later caller mutation cannot change the Server's grant (no shared mutable state, PRD §9.4); a nil or
// empty slice leaves the Server granting no SCP role, so same-association C-GET cannot proceed.
func WithGetStorageRoles(sopClasses ...dicom.SOPClassUID) ServerOption {
	return func(c *serverConfig) {
		if len(sopClasses) == 0 {
			c.getStorageRoles = nil
			return
		}
		copied := make([]dicom.SOPClassUID, len(sopClasses))
		copy(copied, sopClasses)
		c.getStorageRoles = copied
	}
}

// WithAssociationAuthorizer authorizes each inbound association by its negotiated Calling AE Title
// and peer address before any DIMSE service runs. The acceptor consults authorize after the
// AE-title policy and before presentation-context matching; a non-nil return rejects the
// association with an A-ASSOCIATE-RJ (calling-AE-title-not-recognized), so an unauthorized peer
// cannot C-ECHO/C-STORE/C-FIND. It is the seam an embedding server uses to enforce an
// application-level identity policy (DICOM user-identity, a Calling-AE allowlist resolved
// dynamically) on a network-exposed SCP. A nil authorize restores the default (every association
// the AE-title policy admits is served).
func WithAssociationAuthorizer(authorize func(callingAE string, peer net.Addr) error) ServerOption {
	return func(c *serverConfig) { c.associationAuthorizer = authorize }
}

// WithAuthenticator authenticates the user identity a requestor presents in its A-ASSOCIATE-RQ (PS3.7
// D.3.3.7) before any DIMSE service runs — the only association-level authentication DICOM defines.
// The acceptor consults authenticate after the Calling-AE authorizer and before presentation-context
// matching, passing the presented identity (nil when the peer presented none) and the peer address; a
// non-nil error rejects the association with an A-ASSOCIATE-RJ (service-user source), so an
// unauthenticated peer cannot run any service. When the requestor asked for a positive response, the
// returned bytes are echoed back as the user-identity server response (a Kerberos server ticket or a
// SAML response); an empty response with a nil error still accepts. It composes with
// WithAssociationAuthorizer (the Calling-AE check runs first). A nil authenticate restores the default
// (an identity a peer presents is accepted without challenge). The identity it receives carries
// secrets that must never be logged (PRD §9.8).
func WithAuthenticator(authenticate func(id *UserIdentity, peer net.Addr) ([]byte, error)) ServerOption {
	return func(c *serverConfig) { c.authenticator = authenticate }
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
	// joined is closed by a single waiter goroutine (started once by teardownOnce) when every
	// tracked goroutine has finished. Every Shutdown call selects on it against its own ctx, so
	// repeated deadline-bounded retries share one waiter rather than leaking a goroutine per call.
	joined chan struct{}
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
		joined:    make(chan struct{}),
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
	// With WithTLS the listener terminates TLS: every accepted connection completes a TLS handshake
	// (and, with cfg.ClientAuth = RequireAndVerifyClientCert, client-certificate verification)
	// before any PDU is read. Without TLS the plain listener is used unchanged.
	if tlsCfg := s.ae.config().tlsConfigWithFloor(); tlsCfg != nil {
		ln = tls.NewListener(ln, tlsCfg)
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
	// Count the accept loop under the SAME lock that publishes the listener, BEFORE Shutdown can
	// observe the listener and call wg.Wait. This keeps the WaitGroup counter >= 1 from the moment
	// the server is publicly visible until Shutdown joins, so no wg.Add (here, the accept-loop
	// watcher, or a per-association goroutine) is ever a from-zero Add racing a concurrent Wait.
	s.wg.Add(1)
	s.mu.Unlock()

	defer s.wg.Done()
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

		// trackConn already did wg.Add(1) under s.mu (so the goroutine is counted before the
		// connection is visible to Shutdown); this goroutine pairs it with wg.Done.
		go func() {
			defer s.wg.Done()
			defer func() { <-s.sem }()
			defer s.untrackConn(conn)
			// Complete a TLS handshake (when this is a TLS listener) before any PDU is read. The
			// handshake runs in this per-association goroutine, not the accept loop, so a slow peer
			// cannot stall the listener; it is bounded explicitly because the read-deadline the PDU
			// path sets does not cover the writes a handshake also performs (see handshakeAccepted).
			if err := s.ae.handshakeAccepted(ctx, nc); err != nil {
				// A failed or stalled handshake releases the slot (deferred) and closes the
				// connection here, so a peer that opens TCP but never completes the handshake cannot
				// hold a max-association slot past the negotiation bound.
				_ = conn.Close()
				return
			}
			s.serveConn(ctx, conn)
		}()
	}
}

// handshakeAccepted completes the TLS handshake on an accepted connection before it reaches the
// negotiation phase, bounded by the AE's negotiation deadline. A plaintext connection (no *tls.Conn)
// is a no-op, so the plaintext accept path is unchanged. The PDU read path sets only a read
// deadline, which does not bound the writes a TLS handshake also performs; running and bounding the
// handshake here guarantees a peer stalling mid-handshake releases its capacity slot within the
// negotiation bound rather than holding it until Shutdown.
func (ae *AE) handshakeAccepted(ctx context.Context, nc net.Conn) error {
	tc, ok := nc.(*tls.Conn)
	if !ok {
		return nil
	}
	hsCtx, cancel := ae.handshakeContext(ctx)
	defer cancel()
	return tc.HandshakeContext(hsCtx)
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
		// Grant the requestor the Storage SCP role for each configured C-GET Storage SOP Class so a
		// same-association C-GET can C-STORE the matched instances back (PS3.7 D.3.3.4). The SCU role
		// is granted too, so an everyday Storage SCU on the same SOP Class is unaffected.
		SupportedRoles: s.getStorageSupportedRoles(),
		// Carry the configured association authorizer into negotiation so an unauthorized Calling AE
		// Title is rejected with an A-ASSOCIATE-RJ before any service runs. The peer address is
		// captured here (the per-association goroutine owns conn) so the authorizer sees both the
		// Calling AE Title and where the association came from.
		AuthorizeCalling: s.authorizeCalling(conn.RemoteAddr()),
		// Carry the configured user-identity authenticator into negotiation so a presented identity is
		// authenticated (and rejected with an A-ASSOCIATE-RJ on failure) before any service runs (PS3.7
		// D.3.3.7). The peer address is bound here, as for the Calling-AE authorizer.
		Authenticate: s.authenticate(conn.RemoteAddr()),
	}
	move := moveSupport{ae: s.ae, destinations: s.cfg.moveDestinations}
	if err := dispatchAssociation(ctx, conn, params, s.ae.config().acseTimeout, s.ae.config().networkTimeout, s.handler, move); err != nil {
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
// stop of the in-flight handlers, bounded by ctx. It returns ctx.Err() if the handlers do not
// finish within the deadline, nil once they have actually finished.
//
// Two parts compose Shutdown. The teardown — set the shutdown flag, close the listener (no new
// association can start), cancel the handler context, close the active connections — runs ONCE and
// is idempotent. The bounded join (wg.Wait against ctx) is RE-RUNNABLE: every Shutdown re-attempts
// it against its own ctx and returns the real outcome. So a second Shutdown after the first hit its
// deadline does NOT rubber-stamp success: it waits again and returns nil only when the handlers
// have genuinely finished, ctx.Err() while they are still running (P2 adversarial review). The
// earlier defect gated the whole of Shutdown — teardown AND wait — behind a sync.Once, so a second
// call short-circuited to nil even with handlers still alive, falsely reporting a clean shutdown.
//
// The teardown ordering matters. It closes the listener first (no new association can start), then
// CANCELS the handler context so a handler doing application work — a C-STORE persisting to disk,
// the realistic case — that observes its context returns promptly. It then closes the active
// connections so a handler parked in a DriveInbound/ReadPDU (which the context cancellation alone
// may not interrupt at the socket) is also woken (cooperative shutdown, DIMSE-014). The dataset
// already in flight is not lost — a handler that observes its context can complete the in-flight
// store and then return.
//
// A handler that ignores its context AND is not in a connection read cannot be woken (Go cannot
// forcibly kill a goroutine); it can outlive this deadline. A subsequent Shutdown with a fresh
// deadline re-runs the join and reports nil once that handler finally returns. Observing the
// cancelled context is the handler's contract (see Server).
func (s *Server) Shutdown(ctx context.Context) error {
	s.teardownOnce()

	// Re-runnable bounded join sharing the single waiter teardownOnce started. NOT gated behind the
	// once: each Shutdown selects the shared s.joined against its own ctx and returns the real
	// outcome, so a retry after a deadline reports actual completion (never a false nil) without
	// leaking a fresh waiter goroutine per call.
	select {
	case <-s.joined:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// teardownOnce performs the idempotent, once-only Shutdown teardown: it marks the server shutting
// down, closes the listener, cancels the handler context, and closes the active connections. It is
// separated from the bounded wait so a second Shutdown re-attempts the wait (returning the real
// outcome) without re-running these side effects (which must happen exactly once).
func (s *Server) teardownOnce() {
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
		// Start the single join waiter; it closes s.joined once every tracked goroutine has
		// finished, so all Shutdown calls share one waiter rather than spawning one per retry.
		go func() {
			s.wg.Wait()
			close(s.joined)
		}()
	})
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
	// Count the per-association goroutine under the SAME lock that records the connection (and that
	// Shutdown's teardown snapshots under). Otherwise a connection could become visible to Shutdown
	// — and so be closed and waited on — before its goroutine is counted, racing this wg.Add against
	// Shutdown's wg.Wait (Go WaitGroup misuse, and a possible early return before the join).
	s.wg.Add(1)
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

// getStorageSupportedRoles builds the acceptor-side SCP/SCU role grants for the configured C-GET
// Storage SOP Classes: each is granted both the SCP role (so a same-association C-GET can C-STORE the
// matched instances back to the requestor) and the SCU role (so an everyday Storage SCU on the same
// SOP Class still negotiates the default SCU role). A SOP Class not declared here is left to the
// default roles by NegotiateRoles, never an explicit denial. It returns nil when no C-GET roles are
// configured, so a Server with no C-GET support sends no role grants.
func (s *Server) getStorageSupportedRoles() []acse.SupportedRole {
	if len(s.cfg.getStorageRoles) == 0 {
		return nil
	}
	roles := make([]acse.SupportedRole, 0, len(s.cfg.getStorageRoles))
	for _, sopClass := range s.cfg.getStorageRoles {
		roles = append(roles, acse.SupportedRole{
			SOPClassUID: string(sopClass),
			SCURole:     true,
			SCPRole:     true,
		})
	}
	return roles
}

// authorizeCalling adapts the configured association authorizer to the acse Calling-AE hook,
// binding the peer address captured at accept so the authorizer sees both the Calling AE Title and
// the peer. It returns nil when no authorizer is configured, so the acceptor admits every
// association the AE-title policy allows.
func (s *Server) authorizeCalling(peer net.Addr) func(callingAE string) error {
	if s.cfg.associationAuthorizer == nil {
		return nil
	}
	return func(callingAE string) error {
		return s.cfg.associationAuthorizer(callingAE, peer)
	}
}

// authenticate adapts the configured user-identity authenticator to the acse Authenticate hook,
// translating the pdu-level user-identity request to the public UserIdentity and binding the peer
// address captured at accept. It returns nil when no authenticator is configured, so the acceptor
// accepts a presented identity without challenge. The translated identity carries secrets that the
// authenticator must not log (PRD §9.8).
func (s *Server) authenticate(peer net.Addr) func(*pdu.UserIdentityRQ) ([]byte, error) {
	if s.cfg.authenticator == nil {
		return nil
	}
	return func(rq *pdu.UserIdentityRQ) ([]byte, error) {
		return s.cfg.authenticator(fromPDUUserIdentityRQ(rq), peer)
	}
}

// fromPDUUserIdentityRQ translates the pdu-level user-identity request to the public UserIdentity,
// returning nil when the requestor presented none so the authenticator sees a nil identity for an
// anonymous association.
func fromPDUUserIdentityRQ(rq *pdu.UserIdentityRQ) *UserIdentity {
	if rq == nil {
		return nil
	}
	return &UserIdentity{
		Type:                      UserIdentityType(rq.Type),
		PrimaryField:              rq.PrimaryField,
		SecondaryField:            rq.SecondaryField,
		PositiveResponseRequested: rq.PositiveResponseRequested,
	}
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
