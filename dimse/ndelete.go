package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
)

// NDelete issues an N-DELETE that removes an existing managed SOP Instance (PS3.7 §10.1.6).
// sopClassUID is the Requested SOP Class UID (0000,0003) and instanceUID the Requested SOP Instance
// UID (0000,1001) of the managed object to delete; both reference an existing object, the reference
// pair the DIMSE-N operations use rather than the C-service Affected pair. An N-DELETE carries no
// data set.
//
// A presentation context for sopClassUID must have been negotiated. An empty SOP Class or instance
// UID, or no negotiated context, is a fail-closed *ValidationError/*AssociationError returned before
// any wire I/O (PRD §9.2). The returned Status is the N-DELETE-RSP status, interpreted against
// ServiceClassGeneral; it is meaningful only when the returned error is nil. A Failure-category
// status (for example, No Such SOP Instance) is in-band data the caller inspects, never laundered to
// success.
//
// No patient value or SOP Instance value is ever logged or embedded in an error string (PRD §9.1).
func (a *Association) NDelete(ctx context.Context, sopClassUID dicom.SOPClassUID, instanceUID dicom.UID) (Status, error) {
	if a == nil || a.requestor == nil {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "N-DELETE on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "N-DELETE on a released association"}
	}
	if sopClassUID == "" {
		return Status{}, &ValidationError{Detail: "N-DELETE requires the Requested SOP Class UID of the managed object to delete"}
	}
	if instanceUID == "" {
		return Status{}, &ValidationError{Detail: "N-DELETE requires the Requested SOP Instance UID of the managed object to delete"}
	}

	pcID, _, ok := a.contextForQuery(sopClassUID)
	if !ok {
		return Status{}, &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for the N-DELETE Requested SOP Class",
		}
	}

	conn := a.requestor.Conn()
	machine := a.requestor.Machine()
	opCtx, cancel := a.dimseContext(ctx)
	defer cancel()

	rq := CommandSet{
		CommandField:            CommandNDeleteRQ,
		MessageID:               a.nextMessageID(),
		RequestedSOPClassUID:    dicom.UID(sopClassUID),
		RequestedSOPInstanceUID: instanceUID,
		CommandDataSetType:      CommandDataSetNotPresent,
	}
	if err := sendCommand(opCtx, conn, machine, pcID, rq); err != nil {
		return Status{}, err
	}

	rsp, _, err := receiveCommand(opCtx, conn, machine)
	if err != nil {
		return Status{}, err
	}
	if rsp.CommandField != CommandNDeleteRSP {
		return Status{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "expected an N-DELETE-RSP command field in the response",
		}
	}
	if !rsp.HasStatus {
		return Status{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "N-DELETE-RSP missing the mandatory Status element",
		}
	}
	return NewStatus(rsp.Status, ServiceClassGeneral), nil
}
