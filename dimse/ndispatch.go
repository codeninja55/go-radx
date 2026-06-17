package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
)

// nRequestFromCommand builds the no-PHI NRequest a DIMSE-N handler receives from an inbound command
// set, its (optional) data set, and the resolved per-operation context. It carries the command's
// DIMSE-N reference fields verbatim; which are populated depends on the operation (see NRequest).
func nRequestFromCommand(acc *acse.Acceptor, cmd CommandSet, ds *dicom.DataSet, pcID uint8, base OpInfo) NRequest {
	ts, _ := acceptedTransferSyntaxResolver(acc)(pcID)
	info := base
	info.PresentationID = pcID
	info.TransferSyntax = ts
	info.MessageID = cmd.MessageID
	// The affected SOP Class names the managed object's class on an N-CREATE/N-EVENT-REPORT; on the
	// reference-pair operations (N-GET/N-SET/N-ACTION/N-DELETE) it is the Requested SOP Class.
	if cmd.AffectedSOPClassUID != "" {
		info.SOPClassUID = dicom.SOPClassUID(cmd.AffectedSOPClassUID)
	} else {
		info.SOPClassUID = dicom.SOPClassUID(cmd.RequestedSOPClassUID)
	}
	return NRequest{
		Info:                    info,
		RequestedSOPClassUID:    cmd.RequestedSOPClassUID,
		RequestedSOPInstanceUID: cmd.RequestedSOPInstanceUID,
		AffectedSOPClassUID:     cmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:  cmd.AffectedSOPInstanceUID,
		AttributeIdentifierList: cmd.AttributeIdentifierList,
		HasActionTypeID:         cmd.HasActionTypeID,
		ActionTypeID:            cmd.ActionTypeID,
		HasEventTypeID:          cmd.HasEventTypeID,
		EventTypeID:             cmd.EventTypeID,
		DataSet:                 ds,
	}
}

// serveNGetMessage dispatches an already-read N-GET-RQ to the NGetHandler and writes the N-GET-RSP
// (PS3.7 §10.1.2). The RSP echoes the Requested SOP Class/Instance UID into the Affected SOP
// Class/Instance UID of the response (PS3.7 §10.3.2 maps the request's Requested pair to the
// response's Affected pair), carries the handler's status, and — on a Success status — carries the
// returned attribute list as the response data set. A non-success status carries no data set, and
// the dispatch never launders a failure into a success (PRD §9.2 fail-closed).
func serveNGetMessage(ctx context.Context, acc *acse.Acceptor, h NGetHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, base OpInfo) error {
	req := nRequestFromCommand(acc, cmd, ds, pcID, base)
	status, attrs := h.NGet(ctx, req)

	rsp := CommandSet{
		CommandField:              CommandNGetRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.RequestedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.RequestedSOPInstanceUID,
		HasStatus:                 true,
		Status:                    status.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	if status.IsSuccess() && attrs != nil {
		rsp.CommandDataSetType = CommandDataSetPresent
		ts, _ := acceptedTransferSyntaxResolver(acc)(pcID)
		sendCap := MaxPDULength(acc.PeerMaxPDULength()).SendCap(defaultMaxPDULength)
		return sendMessage(ctx, acc.Conn(), acc.Machine(), pcID, rsp, attrs, ts, sendCap)
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}

// serveNDeleteMessage dispatches an already-read N-DELETE-RQ to the NDeleteHandler and writes the
// N-DELETE-RSP (PS3.7 §10.1.6). The RSP echoes the Requested SOP Class/Instance UID into the
// Affected SOP Class/Instance UID of the response, carries the handler's status, and carries no data
// set. The handler returns Success only after the object is removed (PRD §9.2 fail-closed).
func serveNDeleteMessage(ctx context.Context, acc *acse.Acceptor, h NDeleteHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, base OpInfo) error {
	req := nRequestFromCommand(acc, cmd, ds, pcID, base)
	status := h.NDelete(ctx, req)

	rsp := CommandSet{
		CommandField:              CommandNDeleteRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.RequestedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.RequestedSOPInstanceUID,
		HasStatus:                 true,
		Status:                    status.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}

// serveNCreateMessage dispatches an already-read N-CREATE-RQ to the NCreateHandler and writes the
// N-CREATE-RSP (PS3.7 §10.1.5). The RSP echoes the Affected SOP Class UID and, on success, the
// created object's SOP Instance UID — the one the SCU proposed or the one the handler assigned. This
// is the foundation hook a later MPPS SCP plugs into.
func serveNCreateMessage(ctx context.Context, acc *acse.Acceptor, h NCreateHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, base OpInfo) error {
	req := nRequestFromCommand(acc, cmd, ds, pcID, base)
	status, instanceUID := h.NCreate(ctx, req)

	affectedInstance := cmd.AffectedSOPInstanceUID
	if status.IsSuccess() && instanceUID != "" {
		affectedInstance = instanceUID
	}
	rsp := CommandSet{
		CommandField:              CommandNCreateRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    affectedInstance,
		HasStatus:                 true,
		Status:                    status.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}

// serveNSetMessage dispatches an already-read N-SET-RQ to the NSetHandler and writes the N-SET-RSP
// (PS3.7 §10.1.3). The RSP echoes the Requested SOP Class/Instance UID into the Affected pair. This
// is the foundation hook a later MPPS SCP plugs into.
func serveNSetMessage(ctx context.Context, acc *acse.Acceptor, h NSetHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, base OpInfo) error {
	req := nRequestFromCommand(acc, cmd, ds, pcID, base)
	status := h.NSet(ctx, req)

	rsp := CommandSet{
		CommandField:              CommandNSetRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.RequestedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.RequestedSOPInstanceUID,
		HasStatus:                 true,
		Status:                    status.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}

// serveNActionMessage dispatches an already-read N-ACTION-RQ to the NActionHandler and writes the
// N-ACTION-RSP (PS3.7 §10.1.4). The RSP echoes the Requested SOP Class/Instance UID into the
// Affected pair and the Action Type ID. This is the foundation hook a later Storage Commitment SCP
// plugs into; the commitment result itself follows asynchronously via N-EVENT-REPORT.
func serveNActionMessage(ctx context.Context, acc *acse.Acceptor, h NActionHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, base OpInfo) error {
	req := nRequestFromCommand(acc, cmd, ds, pcID, base)
	status := h.NAction(ctx, req)

	rsp := CommandSet{
		CommandField:              CommandNActionRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.RequestedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.RequestedSOPInstanceUID,
		HasActionTypeID:           cmd.HasActionTypeID,
		ActionTypeID:              cmd.ActionTypeID,
		HasStatus:                 true,
		Status:                    status.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}

// serveNEventReportMessage dispatches an already-read N-EVENT-REPORT-RQ to the NEventReportHandler
// and writes the N-EVENT-REPORT-RSP (PS3.7 §10.1.1). The RSP echoes the Affected SOP Class/Instance
// UID and the Event Type ID. This is the foundation hook a general N-EVENT-REPORT receiver plugs
// into.
func serveNEventReportMessage(ctx context.Context, acc *acse.Acceptor, h NEventReportHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, base OpInfo) error {
	req := nRequestFromCommand(acc, cmd, ds, pcID, base)
	status := h.NEventReport(ctx, req)

	rsp := CommandSet{
		CommandField:              CommandNEventReportRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.AffectedSOPInstanceUID,
		HasEventTypeID:            cmd.HasEventTypeID,
		EventTypeID:               cmd.EventTypeID,
		HasStatus:                 true,
		Status:                    status.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}

// refuseUnsupportedN answers a DIMSE-N request that reached an SCP with no handler capability for
// that operation: it writes the matching N-*-RSP carrying StatusSOPClassNotSupported (0x0122) so the
// peer learns the service is unsupported, rather than the SCP panicking or aborting (interface
// segregation, PRD §8.2). It echoes the request's reference UIDs (whichever pair the operation
// carried) so the peer can correlate the refusal. The request data set, if any, has already been
// read from the wire by the dispatch loop and is discarded — nothing is acted on (fail-closed).
func refuseUnsupportedN(ctx context.Context, acc *acse.Acceptor, rspField CommandField, cmd CommandSet, pcID uint8) error {
	rsp := CommandSet{
		CommandField:              rspField,
		MessageIDBeingRespondedTo: cmd.MessageID,
		HasStatus:                 true,
		Status:                    StatusSOPClassNotSupported.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	// Echo whichever reference pair the request carried so the peer can correlate the refusal: the
	// Affected pair (N-CREATE/N-EVENT-REPORT) or the Requested pair mapped to the response's Affected
	// pair (N-GET/N-SET/N-ACTION/N-DELETE).
	if cmd.AffectedSOPClassUID != "" {
		rsp.AffectedSOPClassUID = cmd.AffectedSOPClassUID
		rsp.AffectedSOPInstanceUID = cmd.AffectedSOPInstanceUID
	} else {
		rsp.AffectedSOPClassUID = cmd.RequestedSOPClassUID
		rsp.AffectedSOPInstanceUID = cmd.RequestedSOPInstanceUID
	}
	if cmd.HasActionTypeID {
		rsp.HasActionTypeID = true
		rsp.ActionTypeID = cmd.ActionTypeID
	}
	if cmd.HasEventTypeID {
		rsp.HasEventTypeID = true
		rsp.EventTypeID = cmd.EventTypeID
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}
