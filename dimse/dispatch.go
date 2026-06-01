package dimse

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
	"github.com/codeninja55/go-radx/dimse/pdu"
)

// dispatchAssociation negotiates one inbound association and then dispatches its DIMSE operations
// until the peer releases, aborts, or the transport closes. It is the per-association handler the
// Server runs in a tracked goroutine for each accepted connection.
//
// Negotiation runs through acse.Accept, the single owner of acceptor-side association establishment
// (the architect's DUL-ownership decision): it reads the A-ASSOCIATE-RQ, enforces the configured
// AE-title policy (rejecting at negotiation with an A-ASSOCIATE-RJ on a mismatch), matches the
// proposed presentation contexts against supported, and replies A-ASSOCIATE-AC. A rejected or
// faulted negotiation returns the acse error; the caller closes the connection.
//
// Once established, every inbound read goes through dul.DriveInbound against the acceptor's own
// StateMachine so the provider A-ABORT on an unexpected/invalid PDU and the clean-close distinction
// are never reimplemented. A received DIMSE message (P-DATA-TF stream) is routed by command field
// to the Echo or Store dispatch primitive; an A-RELEASE-RQ completes the graceful release; an
// A-ABORT or an orderly transport close ends the association cleanly.
//
// The negotiation read (acse.Accept) is bounded by acseTimeout (the AE's association-negotiation
// bound) when it is positive: a peer that opens TCP, acquires a capacity slot, but NEVER sends the
// A-ASSOCIATE-RQ must not hold the slot (and a goroutine) until Shutdown — a slot-exhaustion DoS in
// the negotiation phase (P1 adversarial review). On the timeout acse.Accept returns a deadline
// error, this returns it, the connection is closed, and the Server releases the slot.
//
// Each post-negotiation inbound read is bounded by networkTimeout (the AE's idle-association bound)
// when it is positive: a peer that completes negotiation then sends nothing must not hold a capacity
// slot and a goroutine forever (a slot-exhaustion DoS — Codex/concurrency review). On the timeout
// the read returns context.DeadlineExceeded, the association ends, and the Server releases the slot.
func dispatchAssociation(ctx context.Context, conn *dul.Conn, params acse.AcceptParams, acseTimeout, networkTimeout time.Duration, h any) error {
	negotiateCtx := ctx
	if acseTimeout > 0 {
		var cancel context.CancelFunc
		negotiateCtx, cancel = context.WithTimeout(ctx, acseTimeout)
		defer cancel()
	}
	acc, err := acse.Accept(negotiateCtx, conn, params)
	if err != nil {
		return err
	}
	calling := AETitle(acc.CallingAETitle())
	called := AETitle(acc.CalledAETitle())
	resolveTS := acceptedTransferSyntaxResolver(acc)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		cmd, ds, pcID, kind, err := readInbound(ctx, acc, networkTimeout, resolveTS)
		if err != nil {
			return err
		}
		switch kind {
		case inboundReleased:
			// CompleteRelease sends the A-RELEASE-RP then awaits the peer's orderly transport close (a
			// final inbound read). A peer that sends A-RELEASE-RQ then HOLDS the TCP open would block
			// that read forever, holding the capacity slot and goroutine until Shutdown — a
			// slot-exhaustion DoS in the release phase (P1 adversarial review). Bound the await-close
			// read by networkTimeout: it is an idle wait for the peer to close, the same idle-bound the
			// post-negotiation reads use.
			releaseCtx := ctx
			if networkTimeout > 0 {
				var cancel context.CancelFunc
				releaseCtx, cancel = context.WithTimeout(ctx, networkTimeout)
				defer cancel()
			}
			return acc.CompleteRelease(releaseCtx)
		case inboundAborted, inboundClosed:
			// The peer aborted or closed the transport in an orderly way; the association is over.
			return conn.Close()
		case inboundMessage:
			if derr := dispatchMessage(ctx, acc, h, cmd, ds, pcID, calling, called); derr != nil {
				return derr
			}
		}
	}
}

// inboundKind classifies what readInbound observed on the association.
type inboundKind uint8

const (
	inboundMessage  inboundKind = iota // a complete DIMSE message (command + optional dataset)
	inboundReleased                    // an A-RELEASE-RQ: the peer requested a graceful release
	inboundAborted                     // an A-ABORT: the peer aborted the association
	inboundClosed                      // an orderly transport close before any further operation
)

// readInbound reads the next inbound event on an established acceptor association. It reads PDUs
// through dul.DriveInbound (the shared inbound hardening) and either reassembles a complete DIMSE
// message from one or more P-DATA-TF PDUs, or reports an A-RELEASE-RQ / A-ABORT / orderly close.
// A protocol violation surfaces as a typed error (DriveInbound has already sent the provider
// A-ABORT); a clean io.EOF reports inboundClosed.
//
// When networkTimeout is positive it bounds EACH inbound PDU read individually — it is the AE's
// IDLE-association bound, not a cap on the whole message. The timeout is derived fresh per read
// (and cancelled immediately after that read returns), so an idle gap — no PDU for networkTimeout,
// whether before the first PDU or stalled mid-reassembly — times out and the association ends,
// while continuous progress resets the deadline. A legitimate large/slow C-STORE spanning many
// P-DATA-TF PDUs over a total longer than networkTimeout therefore succeeds as long as each PDU
// arrives within networkTimeout of the previous, rather than being aborted mid-transfer.
func readInbound(
	ctx context.Context,
	acc *acse.Acceptor,
	networkTimeout time.Duration,
	resolveTS func(pcID uint8) (dicom.TransferSyntax, error),
) (CommandSet, *dicom.DataSet, uint8, inboundKind, error) {
	conn := acc.Conn()
	m := acc.Machine()
	r := newMessageReassemblerFunc(resolveTS)
	for {
		// Bound this single read by the idle timeout. The context is derived fresh per read and
		// cancelled explicitly the moment DriveInbound returns (NOT deferred, which would accumulate
		// one live timer per PDU across the reassembly loop), so the deadline tracks the gap since the
		// last PDU, not the elapsed time of the whole message.
		readCtx := ctx
		var cancel context.CancelFunc
		if networkTimeout > 0 {
			readCtx, cancel = context.WithTimeout(ctx, networkTimeout)
		}
		p, _, err := dul.DriveInbound(readCtx, conn, m)
		if cancel != nil {
			cancel()
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return CommandSet{}, nil, 0, inboundClosed, nil
			}
			return CommandSet{}, nil, 0, inboundClosed, translateInboundError(m, err)
		}
		switch pv := p.(type) {
		case *pdu.DataTF:
			for _, item := range pv.Items {
				done, aerr := r.add(item)
				if aerr != nil {
					return CommandSet{}, nil, 0, inboundClosed, aerr
				}
				if done {
					return *r.command, r.dataset, r.pcID, inboundMessage, nil
				}
			}
		case *pdu.ReleaseRQ:
			return CommandSet{}, nil, 0, inboundReleased, nil
		case *pdu.Abort:
			return CommandSet{}, nil, 0, inboundAborted, nil
		default:
			// A recognised-but-unexpected PDU in Sta6 drives AA-8 inside DriveInbound, which
			// returns a *StateError translated above; reaching here means a PDU type with a
			// defined non-fault transition we do not consume during data transfer.
			return CommandSet{}, nil, 0, inboundClosed, &ProtocolError{
				State:  m.CurrentState(),
				Detail: "unexpected " + p.Type().String() + " on an established association",
			}
		}
	}
}

// dispatchMessage routes a complete inbound DIMSE message to the handler by its command field. It
// type-asserts the per-operation capability on h (EchoHandler for a C-ECHO, StoreHandler for a
// C-STORE): a handler that does not implement the routed operation's capability is refused
// gracefully with a StatusSOPClassNotSupported RSP rather than panicking or accepting (interface
// segregation, PRD §8.2). A command field this SCP services no capability for is answered as a
// protocol fault rather than silently dropped. The Echo and Store paths reuse the shared dispatch
// primitives (serveEchoCommand / serveStoreMessage) so the response logic lives in one place.
func dispatchMessage(
	ctx context.Context,
	acc *acse.Acceptor,
	h any,
	cmd CommandSet,
	ds *dicom.DataSet,
	pcID uint8,
	calling, called AETitle,
) error {
	base := OpInfo{CallingAETitle: calling, CalledAETitle: called}
	switch cmd.CommandField {
	case CommandCEchoRQ:
		// Like the C-STORE path, the C-ECHO must arrive on a context whose negotiated abstract
		// syntax is the Verification SOP Class; a peer that negotiated only Storage contexts must
		// not run Verification on one (asymmetry with validateStoreContext — P2 review).
		if err := validateEchoContext(pcID, acceptedAbstractSyntaxResolver(acc), acc.Machine().CurrentState()); err != nil {
			return err
		}
		base.PresentationID = pcID
		base.MessageID = cmd.MessageID
		base.SOPClassUID = dicom.SOPClassUID(cmd.AffectedSOPClassUID)
		eh, ok := h.(EchoHandler)
		if !ok {
			return refuseUnsupportedEcho(ctx, acc, cmd, pcID)
		}
		_, err := serveEchoCommand(ctx, acc, eh, cmd, pcID, base)
		return err
	case CommandCStoreRQ:
		sh, ok := h.(StoreHandler)
		if !ok {
			return refuseUnsupportedStore(ctx, acc, cmd, pcID)
		}
		_, err := serveStoreMessage(ctx, acc, sh, cmd, ds, pcID, base)
		return err
	default:
		return &ProtocolError{
			State:  acc.Machine().CurrentState(),
			Detail: "received an unsupported command field for this SCP",
		}
	}
}

// refuseUnsupportedEcho answers a C-ECHO-RQ that reached a handler with no EchoHandler capability:
// it writes a C-ECHO-RSP carrying StatusSOPClassNotSupported (0x0122) so the peer learns the
// service is unsupported, rather than the SCP panicking or aborting (interface segregation, PRD
// §8.2). The dataset has already been consumed from the wire by the read loop, so no PDV is left
// dangling.
func refuseUnsupportedEcho(ctx context.Context, acc *acse.Acceptor, cmd CommandSet, pcID uint8) error {
	rsp := CommandSet{
		CommandField:              CommandCEchoRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    StatusSOPClassNotSupported.Code,
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}

// refuseUnsupportedStore answers a C-STORE-RQ that reached a handler with no StoreHandler
// capability: it writes a C-STORE-RSP carrying StatusSOPClassNotSupported (0x0122). The composite
// SOP Instance the RQ carried has already been read from the wire by the dispatch loop and is
// discarded — nothing is persisted (fail-closed), and the RSP echoes the command's Affected SOP
// Class/Instance UIDs so the peer can correlate the refusal.
func refuseUnsupportedStore(ctx context.Context, acc *acse.Acceptor, cmd CommandSet, pcID uint8) error {
	rsp := CommandSet{
		CommandField:              CommandCStoreRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.AffectedSOPInstanceUID,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    StatusSOPClassNotSupported.Code,
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}
