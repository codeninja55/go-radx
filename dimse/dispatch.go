package dimse

import (
	"context"
	"errors"
	"io"

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
func dispatchAssociation(ctx context.Context, conn *dul.Conn, params acse.AcceptParams, h Handler) error {
	acc, err := acse.Accept(ctx, conn, params)
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
		cmd, ds, pcID, kind, err := readInbound(ctx, acc, resolveTS)
		if err != nil {
			return err
		}
		switch kind {
		case inboundReleased:
			return acc.CompleteRelease(ctx)
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
func readInbound(
	ctx context.Context,
	acc *acse.Acceptor,
	resolveTS func(pcID uint8) (dicom.TransferSyntax, error),
) (CommandSet, *dicom.DataSet, uint8, inboundKind, error) {
	conn := acc.Conn()
	m := acc.Machine()
	r := newMessageReassemblerFunc(resolveTS)
	for {
		p, _, err := dul.DriveInbound(ctx, conn, m)
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

// dispatchMessage routes a complete inbound DIMSE message to the handler by its command field. A
// command field this SCP does not service is answered as a protocol fault rather than silently
// dropped (an SCP that negotiated a context it cannot service is a configuration error). The Echo
// and Store paths reuse the shared dispatch primitives (serveEchoCommand / serveStoreMessage) so
// the response logic lives in one place.
func dispatchMessage(
	ctx context.Context,
	acc *acse.Acceptor,
	h Handler,
	cmd CommandSet,
	ds *dicom.DataSet,
	pcID uint8,
	calling, called AETitle,
) error {
	base := OpInfo{CallingAETitle: calling, CalledAETitle: called}
	switch cmd.CommandField {
	case CommandCEchoRQ:
		base.PresentationID = pcID
		base.MessageID = cmd.MessageID
		base.SOPClassUID = dicom.SOPClassUID(cmd.AffectedSOPClassUID)
		_, err := serveEchoCommand(ctx, acc, h, cmd, pcID, base)
		return err
	case CommandCStoreRQ:
		_, err := serveStoreMessage(ctx, acc, h, cmd, ds, pcID, base)
		return err
	default:
		return &ProtocolError{
			State:  acc.Machine().CurrentState(),
			Detail: "received an unsupported command field for this SCP",
		}
	}
}
