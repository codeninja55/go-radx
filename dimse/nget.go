package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
)

// NGetResult is the parsed result of an N-GET operation (PS3.7 §10.1.2): the typed status the SCP
// returned and, on success, the attribute list the SCP read from the managed SOP Instance. The
// returned attributes are nil on a failure status or when the SCP returned none. Attributes is the
// command's data set, decoded with the negotiated transfer syntax; it is never laundered into
// success — inspect Status.
type NGetResult struct {
	// Status is the N-GET-RSP status, interpreted against ServiceClassGeneral (the N-services use the
	// PS3.7 Annex C general DIMSE status codes). A Failure-category status is in-band data, not a Go
	// error, and is never accompanied by attributes.
	Status Status
	// Attributes is the returned attribute list the N-GET-RSP carried as its data set, or nil when the
	// status was not Success or the SCP returned no data set.
	Attributes *dicom.DataSet
	// AffectedSOPInstanceUID is the SOP Instance UID the N-GET-RSP echoed (0000,1000); it identifies
	// the managed object the attributes were read from.
	AffectedSOPInstanceUID dicom.UID
}

// NGet issues an N-GET that reads attributes of an existing managed SOP Instance (PS3.7 §10.1.2).
// sopClassUID is the Requested SOP Class UID (0000,0003) and instanceUID the Requested SOP Instance
// UID (0000,1001) of the managed object to read; both reference an existing object, the reference
// pair the DIMSE-N operations use rather than the C-service Affected pair. attributeList is the
// optional Attribute Identifier List (0000,1005): the DICOM tags of the attributes the SCU wants
// returned; pass nil or an empty slice to request every attribute the SCP chooses to return (PS3.7
// §10.3.2, Type 2).
//
// A presentation context for sopClassUID must have been negotiated. An empty SOP Class or instance
// UID, or no negotiated context, is a fail-closed *ValidationError/*AssociationError returned before
// any wire I/O (PRD §9.2). The returned NGetResult.Status is meaningful only when the returned error
// is nil; a Failure-category status (for example, No Such SOP Instance) is in-band data the caller
// inspects, never laundered to success.
//
// No patient value or SOP Instance value is ever logged or embedded in an error string (PRD §9.1).
func (a *Association) NGet(ctx context.Context, sopClassUID dicom.SOPClassUID, instanceUID dicom.UID, attributeList []dicom.Tag) (NGetResult, error) {
	if a == nil || a.requestor == nil {
		return NGetResult{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "N-GET on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return NGetResult{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "N-GET on a released association"}
	}
	if sopClassUID == "" {
		return NGetResult{}, &ValidationError{Detail: "N-GET requires the Requested SOP Class UID of the managed object to read"}
	}
	if instanceUID == "" {
		return NGetResult{}, &ValidationError{Detail: "N-GET requires the Requested SOP Instance UID of the managed object to read"}
	}

	pcID, ts, ok := a.contextForQuery(sopClassUID)
	if !ok {
		return NGetResult{}, &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for the N-GET Requested SOP Class",
		}
	}

	conn := a.requestor.Conn()
	machine := a.requestor.Machine()
	opCtx, cancel := a.dimseContext(ctx)
	defer cancel()

	rq := CommandSet{
		CommandField:            CommandNGetRQ,
		MessageID:               a.nextMessageID(),
		RequestedSOPClassUID:    dicom.UID(sopClassUID),
		RequestedSOPInstanceUID: instanceUID,
		AttributeIdentifierList: attributeList,
		// An N-GET-RQ carries no data set: the requested attributes are named in the command set.
		CommandDataSetType: CommandDataSetNotPresent,
	}
	if err := sendCommand(opCtx, conn, machine, pcID, rq); err != nil {
		return NGetResult{}, err
	}

	rsp, ds, _, err := receiveMessage(opCtx, conn, machine, newMessageReassembler(ts))
	if err != nil {
		return NGetResult{}, err
	}
	if rsp.CommandField != CommandNGetRSP {
		return NGetResult{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "expected an N-GET-RSP command field in the response",
		}
	}
	if !rsp.HasStatus {
		return NGetResult{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "N-GET-RSP missing the mandatory Status element",
		}
	}

	status := NewStatus(rsp.Status, ServiceClassGeneral)
	result := NGetResult{
		Status:                 status,
		AffectedSOPInstanceUID: rsp.AffectedSOPInstanceUID,
	}
	// Attributes accompany a successful read only; a failure RSP carries no data set, and a stray
	// data set on a non-success status is not surfaced as a success result.
	if status.IsSuccess() {
		result.Attributes = ds
	}
	return result, nil
}
