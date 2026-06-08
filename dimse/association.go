package dimse

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// Association is an established outbound DIMSE association. It wraps the acse requestor that
// owns the DUL connection and tracks the negotiated presentation contexts. Every operation
// guards against being called on an unestablished or released association and returns a
// typed *AssociationError rather than panicking (Codex DIMSE-017). It is not safe for
// concurrent use by multiple goroutines issuing operations; a single goroutine drives it.
type Association struct {
	requestor *acse.Requestor
	accepted  []PresentationContext
	// negotiatedRoles holds the SCP/SCU role-selection grants the acceptor returned in its
	// A-ASSOCIATE-AC (PS3.7 D.3.3.4), one per SOP Class the requestor proposed a role for. A
	// granted SCP role for a storage SOP Class is what lets a same-association C-GET proceed.
	negotiatedRoles []RoleSelection
	// negotiatedAsyncOps holds the asynchronous-operations window the acceptor echoed (PS3.7 D.3.3.3),
	// or nil when none was negotiated. go-radx negotiates the window but delivers concurrency through
	// goroutines, so an echoed (1,1) reports the honest synchronous window.
	negotiatedAsyncOps *AsyncOps
	// userIdentityResponse holds the user-identity server response the acceptor returned (PS3.7
	// D.3.3.7), or nil when the requestor asked for no positive response. The bytes are opaque and
	// never logged.
	userIdentityResponse []byte
	// extendedNegotiations and commonExtendedNegotiations hold the SOP-class extended- and
	// common-extended-negotiation response sub-items the acceptor returned in its A-ASSOCIATE-AC
	// (PS3.7 D.3.3.5, D.3.3.6), or nil when the acceptor accepted none. A requestor that proposed
	// extended negotiation reads the acceptor's negotiated service-class application information here.
	extendedNegotiations       []ExtendedNegotiation
	commonExtendedNegotiations []CommonExtendedNegotiation
	dimseTimeout               time.Duration
	// localMaxPDULength is the maximum PDU length this AE advertised; peerMaxPDULength is the one
	// the peer advertised in its A-ASSOCIATE-AC. A C-STORE is fragmented so every P-DATA-TF stays
	// within both limits (MaxPDULength.SendCap), and an unlimited peer max (0) resolves to the
	// local cap rather than a zero or negative bound (Codex DIMSE-005).
	localMaxPDULength MaxPDULength
	peerMaxPDULength  MaxPDULength

	mu       sync.Mutex
	released bool
	// nextMsgID is the high-water mark of the per-association Message ID allocator. A single
	// goroutine drives an association's primary operation, but C-GET/C-MOVE sub-operation send
	// paths allocate IDs too, so the counter is read and bumped under a.mu (Codex DIMSE-016).
	nextMsgID uint16
	// lastError holds the transport/protocol fault, if any, that terminated the most recent
	// query/retrieve iterator (C-FIND/C-GET/C-MOVE) before a clean terminal DIMSE status. It is
	// per-association, not per-call: a query/retrieve iterator clears it on entry and sets it on a
	// fault, and the caller reads it via LastError() immediately after the range loop (dimse.md
	// "the streaming query contract"). A terminal Failure/Cancel Status is in-band data, never
	// recorded here. Guarded by a.mu because the iterator body and LastError() may be called from
	// different goroutines, though an Association is not safe for concurrent queries.
	lastError error
	// subOpCounts holds the four sub-operation counts (Remaining/Completed/Failed/Warning) from the
	// most recently yielded C-MOVE/C-GET response on this association, refreshed on each yield so a
	// caller can read progress via SubOperationCounts() inside or after the range loop. It is
	// per-association like lastError (an Association is not safe for concurrent queries) and guarded
	// by a.mu because the iterator body and SubOperationCounts() may run on different goroutines.
	subOpCounts SubOperationCounts
}

// SubOperationCounts is the C-MOVE/C-GET sub-operation progress carried by each response: the
// number of sub-operation C-STOREs still Remaining, and the running Completed/Failed/Warning tallies
// (0000,1020–0000,1023, PS3.7 §9.1.4). The retrieve SCU reads it via Association.SubOperationCounts()
// to track progress; the terminal response carries the final tallies.
//
// NumberOfRemainingSubOperations (0000,1020) is conditional (PS3.4 C.4.2.1.5): a peer omits it when
// the outstanding total is unknown and on the terminal response. RemainingKnown reports whether the
// most recent response actually carried it, so a caller can tell an unknown/omitted Remaining (the
// Remaining field is then 0 only as a default) apart from a genuine zero remaining.
type SubOperationCounts struct {
	Remaining      uint16
	RemainingKnown bool
	Completed      uint16
	Failed         uint16
	Warning        uint16
}

// nextMessageID returns the next Message ID for an operation on this association: a distinct,
// non-zero, monotonically increasing 16-bit value (1, 2, 3, …). It wraps past the reserved 0 at
// the 16-bit boundary, so 0xFFFF is followed by 1. Sub-operation C-STOREs (C-GET/C-MOVE) and the
// chained N-services use it so every in-flight request carries its own ID; the single-operation
// C-ECHO/C-STORE paths keep their fixed echoMessageID/storeMessageID constants (Codex DIMSE-016).
func (a *Association) nextMessageID() uint16 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nextMsgID++
	if a.nextMsgID == 0 {
		a.nextMsgID = 1
	}
	return a.nextMsgID
}

// sendCap resolves the P-DATA-TF body byte cap for outbound DIMSE messages: the smaller of the
// peer's advertised maximum and the local maximum, treating an unlimited (0) peer max as the local
// cap (Codex DIMSE-005).
func (a *Association) sendCap() MaxPDULength {
	return a.peerMaxPDULength.SendCap(a.localMaxPDULength)
}

// LastError returns the transport or protocol fault, if any, that terminated the most recent
// query/retrieve iterator (Find/Get/Move) before a clean terminal DIMSE status — a dropped
// connection, an A-ABORT, or a malformed PDU. It is nil when the operation completed with a clean
// terminal status, and must be read immediately after the range loop, before starting another
// query: it is scoped to the most recent iterator on this association, not per-call (dimse.md "the
// streaming query contract"). A terminal Failure/Cancel Status is in-band data the caller inspects,
// never reported here.
func (a *Association) LastError() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastError
}

// clearLastError resets the per-association last-error slot at the start of a query/retrieve
// iterator, so LastError() reflects only the most recent operation. It is a no-op on a nil
// receiver (a pre-flight fault on an unestablished association has no slot to clear).
func (a *Association) clearLastError() {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.lastError = nil
	a.mu.Unlock()
}

// SubOperationCounts returns the four sub-operation counts (Remaining/Completed/Failed/Warning) the
// most recently yielded C-MOVE or C-GET response on this association carried. A retrieve SCU reads
// it after each Pending yield to track progress, and after the loop to read the terminal tallies. It
// is the zero value before the first retrieve and is reset at the start of each retrieve iterator,
// so it is scoped to the most recent operation, not per-call. It is nil-safe.
func (a *Association) SubOperationCounts() SubOperationCounts {
	if a == nil {
		return SubOperationCounts{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.subOpCounts
}

// setSubOperationCounts records the sub-operation counts a C-MOVE/C-GET response carried, read by
// SubOperationCounts(). It is a no-op on a nil receiver.
func (a *Association) setSubOperationCounts(c SubOperationCounts) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.subOpCounts = c
	a.mu.Unlock()
}

// clearSubOperationCounts resets the per-association sub-operation count slot at the start of a
// retrieve iterator, so SubOperationCounts() reflects only the most recent operation. It is a no-op
// on a nil receiver.
func (a *Association) clearSubOperationCounts() {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.subOpCounts = SubOperationCounts{}
	a.mu.Unlock()
}

// setLastError records the transport/protocol fault that ended a query/retrieve iterator. It is
// read by LastError() after the range loop. It is a no-op on a nil receiver: a Find on a nil
// *Association still yields a typed terminal failure, and LastError() (also nil-safe) returns nil
// since there is no association to carry the error (Codex DIMSE-017).
func (a *Association) setLastError(err error) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.lastError = err
	a.mu.Unlock()
}

// AssociateOption configures an outbound association: SCP/SCU role selection, the
// asynchronous-operations window, SOP-class extended and common-extended negotiation, and
// user-identity negotiation (PS3.7 Annex D). The negotiation types and the With...() options live in
// negotiation.go.
type AssociateOption func(*associateConfig)

type associateConfig struct {
	roleSelections             []pdu.RoleSelection
	asyncOps                   *pdu.AsyncOperations
	extendedNegotiations       []pdu.ExtendedNegotiation
	commonExtendedNegotiations []pdu.CommonExtendedNegotiation
	userIdentity               *pdu.UserIdentityRQ
}

// RoleSelection requests the SCP/SCU roles the requestor proposes to play for one SOP Class
// (PS3.7 D.3.3.4). SCURole and SCPRole are the roles the requestor offers; the acceptor's
// grant is read back from Association.NegotiatedRoles. Proposing the acceptor as SCP (and the
// requestor as SCU) for a storage SOP Class is the prerequisite for same-association C-GET.
type RoleSelection struct {
	SOPClassUID dicom.SOPClassUID
	SCURole     bool
	SCPRole     bool
}

// WithRoleSelection adds an SCP/SCU role-selection request to the outbound A-ASSOCIATE-RQ for
// one SOP Class (PS3.7 D.3.3.4). Repeating the option accumulates one request per SOP Class.
func WithRoleSelection(role RoleSelection) AssociateOption {
	return func(c *associateConfig) {
		c.roleSelections = append(c.roleSelections, pdu.RoleSelection{
			SOPClassUID: string(role.SOPClassUID),
			SCURole:     role.SCURole,
			SCPRole:     role.SCPRole,
		})
	}
}

// Associate opens an A-ASSOCIATE-RQ to the peer at addr ("host:port") and blocks until the
// association is accepted, rejected, aborted, or ctx is cancelled. On success it returns an
// established Association; on rejection or abort it returns a typed *AssociationError. The
// called and calling AE titles are validated before any dial, so an invalid title is a
// *ValidationError, never a wasted connection.
func (ae *AE) Associate(
	ctx context.Context,
	addr string,
	called AETitle,
	contexts []PresentationContext,
	opts ...AssociateOption,
) (*Association, error) {
	if _, err := ParseAETitle(string(called)); err != nil {
		return nil, err
	}
	var cfg associateConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	nc, err := ae.dial(ctx, addr)
	if err != nil {
		return nil, err
	}

	conn := dul.NewConn(nc, ae.cfg.acseTimeout)
	req := acse.Request{
		CalledAETitle:              string(called),
		CallingAETitle:             string(ae.title),
		MaxPDULength:               uint32(ae.cfg.maxPDULength),
		Contexts:                   toPDUContextsRQ(contexts),
		ImplementationClassUID:     string(ae.cfg.implementationClassUID),
		ImplementationVersion:      ae.cfg.implementationVersion,
		RoleSelections:             cfg.roleSelections,
		AsyncOps:                   cfg.asyncOps,
		ExtendedNegotiations:       cfg.extendedNegotiations,
		CommonExtendedNegotiations: cfg.commonExtendedNegotiations,
		UserIdentity:               cfg.userIdentity,
	}

	acseCtx, cancel := ae.acseContext(ctx)
	defer cancel()

	requestor, err := acse.Associate(acseCtx, conn, req)
	if err != nil {
		_ = conn.Close()
		return nil, translateAssociateError(err)
	}
	return &Association{
		requestor:                  requestor,
		accepted:                   fromPDUContextsAC(contexts, requestor.AcceptedContexts()),
		negotiatedRoles:            fromPDURoles(requestor.NegotiatedRoles()),
		negotiatedAsyncOps:         fromPDUAsyncOps(requestor.NegotiatedAsyncOps()),
		userIdentityResponse:       fromPDUUserIdentityAC(requestor.UserIdentityResponse()),
		extendedNegotiations:       fromPDUExtendedNegotiations(requestor.NegotiatedExtendedNegotiations()),
		commonExtendedNegotiations: fromPDUCommonExtendedNegotiations(requestor.NegotiatedCommonExtendedNegotiations()),
		dimseTimeout:               ae.cfg.dimseTimeout,
		localMaxPDULength:          ae.cfg.maxPDULength,
		peerMaxPDULength:           MaxPDULength(requestor.PeerMaxPDULength()),
	}, nil
}

// dimseContext derives the per-operation context, applying the association's DIMSE timeout
// when the caller's context has no earlier deadline. It mirrors AE.acseContext for the
// data-transfer phase, so a peer that accepts the association and then never answers a DIMSE
// request cannot block the operation indefinitely (the configured WithDIMSETimeout promise).
func (a *Association) dimseContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if a.dimseTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, a.dimseTimeout)
}

// dial opens the connection to addr, honouring the AE's connection timeout (when set) and ctx
// cancellation. With no TLS config it is a plain TCP connection; with WithTLS it dials over TLS
// and completes the handshake (including peer-certificate verification) before returning, so a
// certificate or handshake failure surfaces here rather than on the first PDU. The TLS handshake is
// bounded by the negotiation deadline: a peer that completes the TCP connect but then stalls the
// handshake cannot block the dial indefinitely (the dialer's Timeout covers only the TCP connect,
// not the handshake that follows). When the caller's ctx already carries a deadline the handshake
// honours it; otherwise a deadline is derived from the AE's negotiation bound so a default-timeout
// SCU is still protected.
func (ae *AE) dial(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{Timeout: ae.cfg.connectionTimeout}
	tlsCfg := ae.cfg.tlsConfigWithFloor()
	if tlsCfg == nil {
		nc, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, &AssociationError{Kind: AssociationNotEstablished, Detail: "dial: " + err.Error()}
		}
		return nc, nil
	}

	// Default ServerName to the dialled host so certificate verification has a name to check when
	// the caller did not pin one, mirroring crypto/tls's own behaviour for tls.Dial.
	if tlsCfg.ServerName == "" {
		if host, _, splitErr := net.SplitHostPort(addr); splitErr == nil {
			tlsCfg.ServerName = host
		}
	}

	// Bound the TLS handshake when the caller supplied no deadline of its own, matching the
	// plaintext path's negotiation precedence (caller ctx deadline > WithConnectionTimeout > ACSE
	// timeout default). Without this a peer that finishes the TCP connect but stalls the TLS
	// handshake would block DialContext forever for a default-configured SCU.
	dialCtx, cancel := ae.handshakeContext(ctx)
	defer cancel()

	td := tls.Dialer{NetDialer: &d, Config: tlsCfg}
	nc, err := td.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return nil, &AssociationError{Kind: AssociationNotEstablished, Detail: "tls dial: " + err.Error()}
	}
	return nc, nil
}

// handshakeContext derives the deadline that bounds a TLS handshake (SCU dial or SCP accept). The
// bound comes from the AE's negotiation knobs, preferring an explicit WithConnectionTimeout and
// falling back to the ACSE timeout, so the same precedence governs the TLS handshake that governs
// the rest of negotiation. It is applied even when ctx already carries a (possibly longer) deadline:
// context.WithTimeout keeps whichever deadline is earlier, mirroring acseContext, so a peer that
// finishes TCP and then stalls the handshake cannot hold the outbound call or an inbound association
// slot past the configured bound. A zero bound (both knobs disabled) leaves the handshake bounded
// only by ctx.
func (ae *AE) handshakeContext(ctx context.Context) (context.Context, context.CancelFunc) {
	bound := ae.cfg.connectionTimeout
	if bound <= 0 {
		bound = ae.cfg.acseTimeout
	}
	if bound <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, bound)
}

// acseContext derives the negotiation context, applying the AE's ACSE timeout when the
// caller's context has no earlier deadline.
func (ae *AE) acseContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ae.cfg.acseTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, ae.cfg.acseTimeout)
}

// State reports the current DUL state. An unestablished association reports Sta1 (Idle).
func (a *Association) State() State {
	if a == nil || a.requestor == nil {
		return Sta1
	}
	return a.requestor.State()
}

// AcceptedContexts returns the presentation contexts the peer accepted, each carrying its
// single accepted transfer syntax and the negotiation result. It is nil for an
// unestablished association.
func (a *Association) AcceptedContexts() []PresentationContext {
	if a == nil || a.requestor == nil {
		return nil
	}
	return a.accepted
}

// NegotiatedRoles returns the SCP/SCU role-selection grants the peer (acceptor) returned in
// its A-ASSOCIATE-AC (PS3.7 D.3.3.4), one per SOP Class the requestor proposed a role for via
// WithRoleSelection. It is nil for an unestablished association or when no role selection was
// negotiated. A granted SCP role for a storage SOP Class is the prerequisite for a
// same-association C-GET.
func (a *Association) NegotiatedRoles() []RoleSelection {
	if a == nil || a.requestor == nil {
		return nil
	}
	return a.negotiatedRoles
}

// PeerImplementationClassUID returns the Implementation Class UID the peer (acceptor) advertised
// in its A-ASSOCIATE-AC user information (PS3.7 D.3.3.2), or "" for an unestablished association or
// a peer that advertised none.
func (a *Association) PeerImplementationClassUID() dicom.UID {
	if a == nil || a.requestor == nil {
		return ""
	}
	return dicom.UID(a.requestor.PeerImplementationClassUID())
}

// PeerImplementationVersionName returns the Implementation Version Name the peer (acceptor)
// advertised in its A-ASSOCIATE-AC user information (PS3.7 D.3.3.2), or "" for an unestablished
// association or a peer that advertised none.
func (a *Association) PeerImplementationVersionName() string {
	if a == nil || a.requestor == nil {
		return ""
	}
	return a.requestor.PeerImplementationVersionName()
}

// Release performs a graceful A-RELEASE, bounded by ctx. It is idempotent: a second Release
// on an already-released association is a safe no-op (Codex DIMSE-017). It returns a typed
// *AssociationError when called on an unestablished association.
func (a *Association) Release(ctx context.Context) error {
	if a == nil || a.requestor == nil {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Release on an unestablished association"}
	}
	a.mu.Lock()
	if a.released {
		a.mu.Unlock()
		return nil // double-Release is safe
	}
	a.released = true
	a.mu.Unlock()
	if err := a.requestor.Release(ctx); err != nil {
		return translateAssociateError(err)
	}
	return nil
}

// Abort sends a user-initiated A-ABORT, bounded by ctx. It returns a typed *AssociationError
// when called on an unestablished or already-released association (Codex DIMSE-017), never
// panicking.
func (a *Association) Abort(ctx context.Context) error {
	if a == nil || a.requestor == nil {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Abort on an unestablished association"}
	}
	a.mu.Lock()
	if a.released {
		a.mu.Unlock()
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Abort on a released association"}
	}
	a.released = true
	a.mu.Unlock()
	if err := a.requestor.Abort(ctx); err != nil {
		return translateAssociateError(err)
	}
	return nil
}

// toPDUContextsRQ translates the public presentation contexts to the pdu-level RQ items the
// acse layer negotiates (string UIDs), keeping the layering acyclic: acse never sees the
// public PresentationContext type.
func toPDUContextsRQ(contexts []PresentationContext) []pdu.PresentationContextRQ {
	out := make([]pdu.PresentationContextRQ, 0, len(contexts))
	for _, pc := range contexts {
		ts := make([]string, 0, len(pc.TransferSyntaxes))
		for _, t := range pc.TransferSyntaxes {
			ts = append(ts, string(t))
		}
		out = append(out, pdu.PresentationContextRQ{
			ID:               pc.ID,
			AbstractSyntax:   string(pc.AbstractSyntax),
			TransferSyntaxes: ts,
		})
	}
	return out
}

// fromPDUContextsAC translates the acceptor's pdu-level results back to public presentation
// contexts, looking up each proposed context's abstract syntax by its echoed ID so the
// public result carries the SOP Class the result refers to.
func fromPDUContextsAC(proposed []PresentationContext, results []pdu.PresentationContextAC) []PresentationContext {
	byID := make(map[uint8]dicom.SOPClassUID, len(proposed))
	for _, pc := range proposed {
		byID[pc.ID] = pc.AbstractSyntax
	}
	out := make([]PresentationContext, 0, len(results))
	for _, r := range results {
		out = append(out, PresentationContext{
			ID:               r.ID,
			AbstractSyntax:   byID[r.ID],
			TransferSyntaxes: []dicom.TransferSyntax{dicom.TransferSyntax(r.TransferSyntax)},
			Result:           ContextResult(r.Result),
		})
	}
	return out
}

// fromPDURoles translates the acceptor's pdu-level role-selection grants back to public
// RoleSelection values, preserving order. It returns nil for an empty input so an association
// that negotiated no roles reports nil rather than a non-nil empty slice.
func fromPDURoles(roles []pdu.RoleSelection) []RoleSelection {
	if len(roles) == 0 {
		return nil
	}
	out := make([]RoleSelection, 0, len(roles))
	for _, r := range roles {
		out = append(out, RoleSelection{
			SOPClassUID: dicom.SOPClassUID(r.SOPClassUID),
			SCURole:     r.SCURole,
			SCPRole:     r.SCPRole,
		})
	}
	return out
}

// translateAssociateError maps an acse-layer typed error to the public dimse error model
// (dimse.md "Error model"). A context cancellation or deadline becomes an
// AssociationTimeout; an acse rejection, abort, or protocol fault maps to the matching
// public type. An already-public error passes through.
func translateAssociateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return &AssociationError{Kind: AssociationTimeout, Detail: err.Error()}
	}

	var rej *acse.RejectedError
	if errors.As(err, &rej) {
		return &AssociationError{Kind: AssociationRejected, Source: rej.Source, Reason: rej.Reason}
	}
	var ab *acse.AbortedError
	if errors.As(err, &ab) {
		kind := AssociationAborted
		if ab.Provider {
			kind = AssociationProviderAborted
		}
		return &AssociationError{Kind: kind, Source: ab.Source, Reason: ab.Reason}
	}
	var pe *acse.ProtocolError
	if errors.As(err, &pe) {
		return &ProtocolError{State: pe.State, Detail: pe.Detail}
	}
	return err
}
