package dimse

import (
	"context"
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
	requestor    *acse.Requestor
	accepted     []PresentationContext
	dimseTimeout time.Duration
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

// AssociateOption configures an outbound association (reserved for role selection,
// async-ops, user-identity, and extended negotiation in later increments).
type AssociateOption func(*associateConfig)

type associateConfig struct{}

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
		CalledAETitle:          string(called),
		CallingAETitle:         string(ae.title),
		MaxPDULength:           uint32(ae.cfg.maxPDULength),
		Contexts:               toPDUContextsRQ(contexts),
		ImplementationClassUID: string(ae.cfg.implementationClassUID),
		ImplementationVersion:  ae.cfg.implementationVersion,
	}

	acseCtx, cancel := ae.acseContext(ctx)
	defer cancel()

	requestor, err := acse.Associate(acseCtx, conn, req)
	if err != nil {
		_ = conn.Close()
		return nil, translateAssociateError(err)
	}
	return &Association{
		requestor:         requestor,
		accepted:          fromPDUContextsAC(contexts, requestor.AcceptedContexts()),
		dimseTimeout:      ae.cfg.dimseTimeout,
		localMaxPDULength: ae.cfg.maxPDULength,
		peerMaxPDULength:  MaxPDULength(requestor.PeerMaxPDULength()),
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

// dial opens the TCP connection to addr, honouring the AE's connection timeout (when set)
// and ctx cancellation.
func (ae *AE) dial(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{Timeout: ae.cfg.connectionTimeout}
	nc, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, &AssociationError{Kind: AssociationNotEstablished, Detail: "dial: " + err.Error()}
	}
	return nc, nil
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
