package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
)

// OpInfo carries the per-operation context an SCP needs to answer an inbound DIMSE operation:
// the AE Titles, the presentation context and its negotiated transfer syntax, the Message ID,
// and the affected SOP Class. It is the structured, no-PHI diagnostic context (PRD §8.2, §9.1):
// it names protocol identifiers only and never carries a patient value, so it is safe to log.
type OpInfo struct {
	CallingAETitle AETitle
	CalledAETitle  AETitle
	PresentationID uint8
	TransferSyntax dicom.TransferSyntax
	MessageID      uint16
	SOPClassUID    dicom.SOPClassUID
}

// Handler answers inbound DIMSE-C operations dispatched by the SCP. An intervention operation is
// answered with a typed Status, so a handler cannot forget to answer (PRD §8.2). Implement the
// methods for the services you support; this v1 surface carries Echo, and grows with the
// data-bearing services (C-STORE in Increment 5). A handler returning success on work it did not
// do is a defect (PRD §9.2 fail-closed).
type Handler interface {
	// Echo answers a C-ECHO. Return StatusEchoSuccess unless the SCP is degraded.
	Echo(ctx context.Context, info OpInfo) Status
}

// serveEcho services one inbound C-ECHO over an established acceptor association: it reads the
// C-ECHO-RQ command set (through dul.DriveInbound against the acceptor's own StateMachine),
// dispatches to the handler, and writes the C-ECHO-RSP carrying the handler's status. The SCP
// server scaffolding (Increment 6) drives this per accepted association; this is the single
// C-ECHO dispatch primitive both share.
//
// The response echoes the request's Affected SOP Class UID and Message ID into the
// Message ID Being Responded To, carries no data set (CommandDataSetType 0x0101), and reports
// the handler's status — the verification contract (PS3.7 §9.1.5).
func serveEcho(ctx context.Context, acc *acse.Acceptor, calling, called AETitle, h Handler) (Status, error) {
	conn := acc.Conn()
	m := acc.Machine()

	cmd, pcID, err := receiveCommand(ctx, conn, m)
	if err != nil {
		return Status{}, err
	}
	if cmd.CommandField != CommandCEchoRQ {
		return Status{}, &ProtocolError{
			State:  m.CurrentState(),
			Detail: "expected a C-ECHO-RQ command field, got a different command",
		}
	}

	info := OpInfo{
		CallingAETitle: calling,
		CalledAETitle:  called,
		PresentationID: pcID,
		MessageID:      cmd.MessageID,
		SOPClassUID:    dicom.SOPClassUID(cmd.AffectedSOPClassUID),
	}
	status := h.Echo(ctx, info)

	rsp := CommandSet{
		CommandField:              CommandCEchoRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    status.Code,
	}
	if err := sendCommand(ctx, conn, m, pcID, rsp); err != nil {
		return Status{}, err
	}
	return status, nil
}
