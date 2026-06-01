package acse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// applicationContextUID is the DICOM Application Context Name negotiated in every
// association (PS3.7 A.2.1). There is exactly one defined value.
const applicationContextUID = "1.2.840.10008.3.1.1.1"

// protocolVersion is the DICOM Upper Layer protocol version (PS3.8 9.3.2): bit 0 set.
const protocolVersion uint16 = 1

// Service-user-source A-ASSOCIATE-RJ diagnostic reasons (PS3.8 Table 9-21). The acceptor
// returns these when it rejects an association because of an AE-title mismatch, matching
// pynetdicom's require_called_aet / require_calling_aet rejection behaviour (a Called AE Title
// mismatch is reported as "Called AE title not recognised", a Calling AE Title mismatch as
// "Calling AE title not recognised").
const (
	reasonCallingAETitleNotRecognized uint8 = 3
	reasonCalledAETitleNotRecognized  uint8 = 7
)

// Request is the requestor-side input to Associate: the called and calling AE titles, the
// proposed presentation contexts, and the local maximum PDU length to advertise (0 means
// unlimited, PS3.8 Annex D.1). The acse layer works in plain UID strings and 16-byte AE
// fields are derived here, so it never imports the root dimse package (acyclic layering).
type Request struct {
	CalledAETitle  string
	CallingAETitle string
	MaxPDULength   uint32
	Contexts       []pdu.PresentationContextRQ

	ImplementationClassUID string
	ImplementationVersion  string
}

// AcceptParams is the acceptor-side input to Accept: the local called AE title, the
// supported presentation contexts to negotiate against, and the local maximum PDU length.
// RejectAll forces an A-ASSOCIATE-RJ regardless of the proposal (used when the acceptor
// can offer no service at all).
type AcceptParams struct {
	CalledAETitle string
	MaxPDULength  uint32
	Supported     []SupportedContext
	RejectAll     bool

	// RequireCalledAETitle, when non-empty, rejects an A-ASSOCIATE-RQ whose Called AE Title
	// does not match it (PS3.8 Table 9-21 called-AE-title-not-recognized). RequireCallingAETitles,
	// when non-empty, rejects an RQ whose Calling AE Title is not one of the listed titles
	// (calling-AE-title-not-recognized). Both are enforced at negotiation, before any
	// presentation-context matching, so a mismatch is refused with an A-ASSOCIATE-RJ rather than
	// established (mirrors pynetdicom's require_called_aet / require_calling_aet).
	RequireCalledAETitle   string
	RequireCallingAETitles []string

	ImplementationClassUID string
	ImplementationVersion  string
}

// Requestor is an established outbound association from the requestor's perspective. It
// owns the dul.Conn for PDU I/O and a dul.StateMachine tracking the PS3.8 lifecycle. The
// state machine is advanced through local primitives (Evt1, Evt2, Evt11, Evt14, Evt15) and
// received-PDU events; the root dimse package wraps a Requestor in its public Association.
type Requestor struct {
	conn     *dul.Conn
	machine  *dul.StateMachine
	accepted []pdu.PresentationContextAC
	peerMax  uint32
}

// Acceptor is an established inbound association from the acceptor's perspective.
type Acceptor struct {
	conn     *dul.Conn
	machine  *dul.StateMachine
	accepted []pdu.PresentationContextAC
	request  *pdu.AssociateRQ
}

// Associate drives the requestor side of association establishment over conn: it advances
// the state machine through A-ASSOCIATE request and transport-connect (the transport is
// already open on conn), sends an A-ASSOCIATE-RQ, and awaits the response. On
// A-ASSOCIATE-AC it returns an established Requestor in Sta6; on A-ASSOCIATE-RJ it returns
// a typed *RejectedError; on A-ABORT a typed *AbortedError; on any other PDU or a transport
// fault a typed *ProtocolError. It never panics on malformed input (PRD §9.3).
func Associate(ctx context.Context, conn *dul.Conn, req Request) (*Requestor, error) {
	m := dul.NewStateMachine()

	// Evt1 (A-ASSOCIATE request) -> AE-1 (transport connect, already open) -> Sta4.
	if _, _, err := m.Apply(dul.Evt1); err != nil {
		return nil, wrapState(m, err)
	}
	// Evt2 (transport connect confirmation) -> AE-2 (send A-ASSOCIATE-RQ) -> Sta5.
	if _, _, err := m.Apply(dul.Evt2); err != nil {
		return nil, wrapState(m, err)
	}
	rq := buildAssociateRQ(req)
	if err := conn.WritePDU(ctx, rq); err != nil {
		return nil, err
	}

	// DriveInbound reads the response and applies its received-event to m, sending the
	// provider A-ABORT itself if the response is an unexpected or invalid PDU (the shared
	// PS3.8 hardening). A-ASSOCIATE-AC drives AE-3 -> Sta6, A-ASSOCIATE-RJ drives AE-4 -> Sta1,
	// and an A-ABORT drives AA-3 -> Sta1; all three are defined transitions and surface here as
	// the decoded PDU with a nil DriveInbound error, which we interpret below.
	resp, _, err := dul.DriveInbound(ctx, conn, m)
	if err != nil {
		return nil, translateDriveError(m, err)
	}

	switch p := resp.(type) {
	case *pdu.AssociateAC:
		return &Requestor{
			conn:     conn,
			machine:  m,
			accepted: p.PresentationContexts,
			peerMax:  p.UserInfo.MaxPDULength,
		}, nil
	case *pdu.AssociateRJ:
		return nil, &RejectedError{Result: p.Result, Source: p.Source, Reason: p.Reason}
	case *pdu.Abort:
		return nil, &AbortedError{Provider: p.Source == pdu.AbortSourceServiceProvider, Source: p.Source, Reason: p.Reason}
	default:
		// A recognised-but-unexpected PDU has already driven AA-8 inside DriveInbound, which
		// returned the *StateError translated above; reaching here means a PDU type with a
		// defined transition we do not consume during establishment.
		return nil, wrapUnexpected(m, resp.Type(), nil)
	}
}

// Accept drives the acceptor side over conn whose transport is already open: it advances to
// Sta2, reads the A-ASSOCIATE-RQ, negotiates the proposed contexts against params.Supported,
// and either sends an A-ASSOCIATE-AC (returning an established Acceptor in Sta6) or, when no
// context is acceptable (or params.RejectAll), sends an A-ASSOCIATE-RJ and returns a typed
// *RejectedError. It never panics on malformed input.
func Accept(ctx context.Context, conn *dul.Conn, params AcceptParams) (*Acceptor, error) {
	m := dul.NewStateMachine()

	// Evt5 (transport indication) -> AE-5 (transport response, already open) -> Sta2.
	if _, _, err := m.Apply(dul.Evt5); err != nil {
		return nil, wrapState(m, err)
	}

	// DriveInbound reads the A-ASSOCIATE-RQ and applies Evt6 (-> AE-6 -> Sta3). Any other
	// recognised PDU is unexpected in Sta2 and drives AA-1 (send user-source A-ABORT) -> Sta13,
	// and a malformed PDU drives the provider A-ABORT; DriveInbound sends the abort and returns
	// the typed *StateError.
	resp, _, err := dul.DriveInbound(ctx, conn, m)
	if err != nil {
		return nil, translateDriveError(m, err)
	}
	rq, ok := resp.(*pdu.AssociateRQ)
	if !ok {
		// A recognised PDU with a defined non-fault transition that is not an A-ASSOCIATE-RQ;
		// the fault PDUs were already aborted inside DriveInbound.
		return nil, wrapUnexpected(m, resp.Type(), nil)
	}

	// Enforce the AE-title policy at negotiation: a mismatch is refused with an A-ASSOCIATE-RJ
	// (service-user source) carrying the matching diagnostic reason, BEFORE any context
	// matching, so the peer never sees an acceptance for an association the acceptor will not
	// serve (PS3.8 Table 9-21).
	if reason, reject := aeTitleRejection(params, rq); reject {
		return nil, rejectAssociation(ctx, conn, m, reason)
	}

	results := NegotiateAcceptor(rq.PresentationContexts, params.Supported)
	if params.RejectAll || !anyAccepted(results) {
		reason := uint8(pdu.PresentationContextAbstractSyntaxNotSupported)
		if params.RejectAll {
			reason = 1 // application-context-name-not-supported (PS3.8 9.3.4) — no service offered
		}
		return nil, rejectAssociation(ctx, conn, m, reason)
	}

	ac := buildAssociateAC(params, rq, results)
	if _, _, serr := m.Apply(dul.Evt7); serr != nil { // accept -> AE-7 (send AC) -> Sta6
		return nil, wrapState(m, serr)
	}
	if err := conn.WritePDU(ctx, ac); err != nil {
		return nil, err
	}
	return &Acceptor{conn: conn, machine: m, accepted: results, request: rq}, nil
}

// aeTitleRejection reports whether the inbound A-ASSOCIATE-RQ violates the acceptor's AE-title
// policy, returning the matching service-user diagnostic reason (PS3.8 Table 9-21). A Called AE
// Title mismatch wins over a Calling AE Title mismatch (the called title is the addressed AE; a
// wrong one means the request reached the wrong endpoint).
func aeTitleRejection(params AcceptParams, rq *pdu.AssociateRQ) (uint8, bool) {
	if params.RequireCalledAETitle != "" && trimAETitle(rq.CalledAETitle) != params.RequireCalledAETitle {
		return reasonCalledAETitleNotRecognized, true
	}
	if len(params.RequireCallingAETitles) > 0 {
		calling := trimAETitle(rq.CallingAETitle)
		for _, allowed := range params.RequireCallingAETitles {
			if calling == allowed {
				return 0, false
			}
		}
		return reasonCallingAETitleNotRecognized, true
	}
	return 0, false
}

// rejectAssociation performs the AE-8 reject: send an A-ASSOCIATE-RJ (service-user source,
// permanent) carrying reason and advance to Sta13, returning the typed *RejectedError.
func rejectAssociation(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, reason uint8) error {
	rj := &pdu.AssociateRJ{
		Result: pdu.AssociateRJResultPermanent,
		Source: pdu.AssociateRJSourceServiceUser,
		Reason: reason,
	}
	if _, _, serr := m.Apply(dul.Evt8); serr != nil { // reject -> AE-8 (send RJ) -> Sta13
		return wrapState(m, serr)
	}
	if err := conn.WritePDU(ctx, rj); err != nil {
		return err
	}
	return &RejectedError{Result: rj.Result, Source: rj.Source, Reason: rj.Reason}
}

// Release performs the requestor-side graceful A-RELEASE: send an A-RELEASE-RQ, await the
// A-RELEASE-RP, then close the transport (AR-3), leaving the machine in Sta1. It is bounded
// by ctx and never panics.
func (r *Requestor) Release(ctx context.Context) error {
	if _, _, err := r.machine.Apply(dul.Evt11); err != nil { // -> AR-1 (send RELEASE-RQ) -> Sta7
		return wrapState(r.machine, err)
	}
	if err := r.conn.WritePDU(ctx, &pdu.ReleaseRQ{}); err != nil {
		return err
	}
	// DriveInbound reads the A-RELEASE-RP and applies Evt13 (Sta7 -> AR-3 -> Sta1). A peer's
	// A-RELEASE-RQ would collide here (Evt12 -> AR-8 -> Sta9); any unexpected or invalid PDU
	// drives AA-8 with the provider A-ABORT sent by DriveInbound.
	resp, _, err := dul.DriveInbound(ctx, r.conn, r.machine)
	if err != nil {
		return translateDriveError(r.machine, err)
	}
	if _, ok := resp.(*pdu.ReleaseRP); !ok {
		return wrapUnexpected(r.machine, resp.Type(), nil)
	}
	return r.conn.Close()
}

// ServeRelease handles the acceptor side of a graceful release: read the A-RELEASE-RQ, send
// the A-RELEASE-RP (AR-4), then observe the transport close (Evt17) and return to Sta1. It
// is bounded by ctx and never panics.
func (a *Acceptor) ServeRelease(ctx context.Context) error {
	// DriveInbound reads the A-RELEASE-RQ and applies Evt12 (Sta6 -> AR-2 -> Sta8). Any
	// unexpected or invalid PDU drives AA-8 with the provider A-ABORT sent by DriveInbound.
	resp, _, err := dul.DriveInbound(ctx, a.conn, a.machine)
	if err != nil {
		return translateDriveError(a.machine, err)
	}
	if _, ok := resp.(*pdu.ReleaseRQ); !ok {
		return wrapUnexpected(a.machine, resp.Type(), nil)
	}
	return a.CompleteRelease(ctx)
}

// CompleteRelease finishes the acceptor-side graceful release once the A-RELEASE-RQ has already
// been read and applied (Evt12 -> AR-2 -> Sta8): it sends the A-RELEASE-RP (AR-4) and observes
// the orderly transport close. The SCP dispatch loop reads each inbound PDU through
// dul.DriveInbound itself (the architect's DUL-ownership decision) and calls this when the PDU it
// read is the A-RELEASE-RQ, so the release-completion hardening is not reimplemented. It is bounded
// by ctx and never panics.
func (a *Acceptor) CompleteRelease(ctx context.Context) error {
	// Evt14 (local A-RELEASE response) is a local primitive: AR-4 (send A-RELEASE-RP) -> Sta13.
	if _, _, serr := a.machine.Apply(dul.Evt14); serr != nil {
		return wrapState(a.machine, serr)
	}
	if err := a.conn.WritePDU(ctx, &pdu.ReleaseRP{}); err != nil {
		return err
	}
	// Await the orderly transport close the requestor performs after AR-3. A clean io.EOF
	// drives Evt17 (Sta13 -> AR-5 -> Sta1); a PDU arriving where a close was expected is
	// treated as unexpected.
	resp, _, rerr := dul.DriveInbound(ctx, a.conn, a.machine)
	if rerr != nil {
		if errors.Is(rerr, io.EOF) {
			return a.conn.Close()
		}
		return translateDriveError(a.machine, rerr)
	}
	return wrapUnexpected(a.machine, resp.Type(), nil)
}

// Abort sends a user-initiated A-ABORT and closes the transport, leaving the machine in
// Sta1 (AA-1 path semantics for the local user). It is bounded by ctx and never panics.
func (r *Requestor) Abort(ctx context.Context) error { return abort(ctx, r.conn, r.machine) }

// Abort sends a user-initiated A-ABORT from the acceptor side.
func (a *Acceptor) Abort(ctx context.Context) error { return abort(ctx, a.conn, a.machine) }

func abort(ctx context.Context, conn *dul.Conn, m *dul.StateMachine) error {
	// Evt15 (A-ABORT request) drives AA-1 (send A-ABORT, service-user source) from the
	// active states; advancing the machine records the abort even if the write fails.
	_, _, _ = m.Apply(dul.Evt15)
	ab := &pdu.Abort{Source: pdu.AbortSourceServiceUser, Reason: pdu.AbortReasonNotSpecified}
	werr := conn.WritePDU(ctx, ab)
	cerr := conn.Close()
	if werr != nil {
		return werr
	}
	return cerr
}

// State reports the current DUL state (for observability).
func (r *Requestor) State() dul.State { return r.machine.CurrentState() }

// State reports the current DUL state (for observability).
func (a *Acceptor) State() dul.State { return a.machine.CurrentState() }

// AcceptedContexts returns the negotiated presentation contexts the acceptor returned.
func (r *Requestor) AcceptedContexts() []pdu.PresentationContextAC { return r.accepted }

// AcceptedContexts returns the presentation contexts the acceptor accepted.
func (a *Acceptor) AcceptedContexts() []pdu.PresentationContextAC { return a.accepted }

// PeerMaxPDULength reports the maximum PDU length the peer advertised (0 = unlimited).
func (r *Requestor) PeerMaxPDULength() uint32 { return r.peerMax }

// RequestedContexts returns the presentation contexts the requestor proposed (acceptor side).
func (a *Acceptor) RequestedContexts() []pdu.PresentationContextRQ {
	if a.request == nil {
		return nil
	}
	return a.request.PresentationContexts
}

// CallingAETitle returns the Calling AE Title from the inbound A-ASSOCIATE-RQ, trimmed of the
// fixed-field padding (the SCU's title). It is empty for a not-yet-established acceptor.
func (a *Acceptor) CallingAETitle() string {
	if a.request == nil {
		return ""
	}
	return trimAETitle(a.request.CallingAETitle)
}

// CalledAETitle returns the Called AE Title from the inbound A-ASSOCIATE-RQ, trimmed of the
// fixed-field padding (the title the SCU addressed). It is empty for a not-yet-established acceptor.
func (a *Acceptor) CalledAETitle() string {
	if a.request == nil {
		return ""
	}
	return trimAETitle(a.request.CalledAETitle)
}

// Conn returns the underlying DUL connection so the DIMSE message layer can send and
// receive P-DATA-TF PDUs over the established association.
func (r *Requestor) Conn() *dul.Conn { return r.conn }

// Conn returns the underlying DUL connection (acceptor side).
func (a *Acceptor) Conn() *dul.Conn { return a.conn }

// Machine returns the association's DUL StateMachine so the DIMSE message layer routes its
// inbound P-DATA-TF reads through dul.DriveInbound against the same machine the association
// lifecycle advances (the architect's DUL-ownership decision: inbound reads never reimplement
// the abort-send/clean-close hardening). It is the same machine State() reports.
func (r *Requestor) Machine() *dul.StateMachine { return r.machine }

// Machine returns the association's DUL StateMachine (acceptor side).
func (a *Acceptor) Machine() *dul.StateMachine { return a.machine }

func buildAssociateRQ(req Request) *pdu.AssociateRQ {
	return &pdu.AssociateRQ{
		ProtocolVersion:      protocolVersion,
		CalledAETitle:        padAETitle(req.CalledAETitle),
		CallingAETitle:       padAETitle(req.CallingAETitle),
		ApplicationContext:   applicationContextUID,
		PresentationContexts: req.Contexts,
		UserInfo: pdu.UserInformation{
			MaxPDULength:           req.MaxPDULength,
			ImplementationClassUID: req.ImplementationClassUID,
			ImplementationVersion:  req.ImplementationVersion,
		},
	}
}

func buildAssociateAC(params AcceptParams, rq *pdu.AssociateRQ, results []pdu.PresentationContextAC) *pdu.AssociateAC {
	return &pdu.AssociateAC{
		ProtocolVersion:      protocolVersion,
		CalledAETitle:        rq.CalledAETitle,
		CallingAETitle:       rq.CallingAETitle,
		ApplicationContext:   applicationContextUID,
		PresentationContexts: results,
		UserInfo: pdu.UserInformation{
			MaxPDULength:           params.MaxPDULength,
			ImplementationClassUID: params.ImplementationClassUID,
			ImplementationVersion:  params.ImplementationVersion,
		},
	}
}

// anyAccepted reports whether at least one negotiated context was accepted.
func anyAccepted(results []pdu.PresentationContextAC) bool {
	for _, r := range results {
		if r.Result == pdu.PresentationContextAcceptance {
			return true
		}
	}
	return false
}

// padAETitle pads an AE title to the 16-byte DICOM field with trailing spaces (PS3.8 the
// fixed 16-byte Called/Calling AE-Title fields). The title is validated by the root dimse
// package's ParseAETitle before reaching here.
func padAETitle(title string) [16]byte {
	var field [16]byte
	n := copy(field[:], title)
	for i := n; i < len(field); i++ {
		field[i] = ' '
	}
	return field
}

// trimAETitle is the inverse of padAETitle: it strips the insignificant surrounding spaces
// from a 16-byte fixed AE-Title field (PS3.5 VR AE; leading and trailing spaces are not
// significant) to recover the AE Title text.
func trimAETitle(field [16]byte) string {
	return strings.Trim(string(field[:]), " ")
}

// translateDriveError maps a dul.DriveInbound error to a typed acse error. A context
// cancellation or deadline is surfaced as-is; a clean io.EOF is a transport close (the
// conversation broke before the expected PDU arrived); a *dul.StateError is a protocol
// violation DriveInbound has already aborted on the wire, surfaced as a *ProtocolError naming
// the violating state and event; anything else is a transport read fault.
func translateDriveError(m *dul.StateMachine, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, io.EOF) {
		return &ProtocolError{State: m.CurrentState(), Detail: "transport closed before the expected PDU arrived"}
	}
	var se *dul.StateError
	if errors.As(err, &se) {
		return &ProtocolError{State: se.State, Detail: se.Error()}
	}
	return &ProtocolError{State: m.CurrentState(), Detail: fmt.Sprintf("reading PDU: %v", err)}
}

// wrapState wraps a state-machine fault as a typed *ProtocolError naming the state.
func wrapState(m *dul.StateMachine, err error) error {
	return &ProtocolError{State: m.CurrentState(), Detail: err.Error()}
}

// wrapUnexpected reports an unexpected PDU type for the current state as a *ProtocolError.
func wrapUnexpected(m *dul.StateMachine, pt pdu.PDUType, _ error) error {
	return &ProtocolError{State: m.CurrentState(), Detail: fmt.Sprintf("unexpected %s in this state", pt)}
}
