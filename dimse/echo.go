package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
)

// echoMessageID is the Message ID used for a C-ECHO request. A C-ECHO is a single
// request/response with no concurrent operations on the association, so a fixed non-zero ID is
// sufficient (PS3.7 §9.1.5); the message-id allocator for concurrent operations lands with the
// query/retrieve services.
const echoMessageID uint16 = 1

// Echo sends a C-ECHO (Verification) over the association and returns the peer's status (PS3.4
// A.4). It selects the presentation context accepted for the Verification SOP Class, builds the
// C-ECHO-RQ command set, sends it as a P-DATA-TF, reads the C-ECHO-RSP through dul.DriveInbound
// (so the provider A-ABORT on an unexpected or invalid PDU is handled by the shared inbound
// hardening), and returns the status carried in the response.
//
// A Failure-category status is data the caller inspects with status.IsFailure(), not a Go error;
// the returned error reports a transport, association, or protocol fault — "the conversation
// broke", distinct from "the peer answered, and said no" (dimse.md "Error model"). It guards
// against an unestablished or released association with a typed error, never a panic (DIMSE-017).
func (a *Association) Echo(ctx context.Context) (Status, error) {
	if a == nil || a.requestor == nil {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "Echo on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "Echo on a released association"}
	}

	pcID, ok := a.contextForVerification()
	if !ok {
		return Status{}, &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for the Verification SOP Class",
		}
	}

	conn := a.requestor.Conn()
	m := a.requestor.Machine()

	// Bound the request/response by the configured DIMSE timeout, so a peer that accepts the
	// association then never sends the C-ECHO-RSP cannot block Echo indefinitely.
	opCtx, cancel := a.dimseContext(ctx)
	defer cancel()

	rq := CommandSet{
		CommandField:        CommandCEchoRQ,
		MessageID:           echoMessageID,
		AffectedSOPClassUID: dicom.UID(verificationSOPClass),
		CommandDataSetType:  CommandDataSetNotPresent,
	}
	if err := sendCommand(opCtx, conn, m, pcID, rq); err != nil {
		return Status{}, err
	}

	rsp, _, err := receiveCommand(opCtx, conn, m)
	if err != nil {
		return Status{}, err
	}
	if rsp.CommandField != CommandCEchoRSP {
		return Status{}, &ProtocolError{
			State:  m.CurrentState(),
			Detail: "expected a C-ECHO-RSP command field in the response",
		}
	}
	// The Status element is mandatory in a C-ECHO-RSP (PS3.7 §9.3.5.2); a response without it
	// is malformed, not an implicit success.
	if !rsp.HasStatus {
		return Status{}, &ProtocolError{
			State:  m.CurrentState(),
			Detail: "C-ECHO-RSP missing the mandatory Status element",
		}
	}
	return NewStatus(rsp.Status, ServiceClassVerification), nil
}

// contextForVerification returns the presentation context ID accepted for the Verification SOP
// Class, or false when none was accepted (so Echo fails closed rather than guessing a context).
func (a *Association) contextForVerification() (uint8, bool) {
	for _, pc := range a.accepted {
		if pc.Result != ContextAccepted {
			continue
		}
		if pc.AbstractSyntax == verificationSOPClass {
			return pc.ID, true
		}
	}
	return 0, false
}
