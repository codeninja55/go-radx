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
// Affected pair and the Action Type ID. This is the hook the Storage Commitment SCP plugs into.
//
// When the handler also implements NActionReporter AND the N-ACTION status is Success, the dispatch
// then invokes the reporter so it can emit the follow-up N-EVENT-REPORT on the SAME association — the
// synchronous Storage Commitment reporting model (PS3.4 J.3.3). The report runs only after the
// N-ACTION-RSP has been written (the request is acknowledged first) and only on Success (a refused
// request carries no report); a reporter error faults the association.
//
// The same-association report is role-reversed: the SCP that received the N-ACTION becomes the
// N-EVENT-REPORT SCU and the original requestor must act as the N-EVENT-REPORT SCP. Under the DICOM
// default roles the requestor is the SCU and the acceptor the SCP, which is the wrong role for this
// report (PS3.7 D.3.3.4, PS3.8 7.1.1.13). The dispatch therefore sends the same-association report
// only when SCP/SCU role selection granted the requestor the SCP role for the N-ACTION's SOP class;
// without that grant a strict peer would abort the role-reversed N-EVENT-REPORT, so the dispatch
// skips it (the commitment result then requires the separate-association report, which is deferred on
// the provider side — see StorageCommitmentProvider). The skip is observable, not silent.
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
	if err := sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp); err != nil {
		return err
	}
	reporter, ok := h.(NActionReporter)
	if !ok || !status.IsSuccess() {
		return nil
	}
	if !requestorHasReportRole(acc, cmd.RequestedSOPClassUID) {
		// Role selection did not grant the requestor the SCP role for this SOP class, so the
		// role-reversed same-association N-EVENT-REPORT cannot be sent without risking a peer abort.
		// Skip it: the result is left to the deferred separate-association report. This is a documented
		// no-op (no logger is wired into the package) and never an aborted association.
		return nil
	}
	return reporter.ReportAfterAction(ctx, newNReportSender(acc, pcID, cmd.RequestedSOPClassUID, cmd.RequestedSOPInstanceUID), req)
}

// requestorHasReportRole reports whether SCP/SCU role selection granted the requestor the SCP role
// for sopClass on this association (a NegotiatedRoles entry with SCPRole set, PS3.7 D.3.3.4). The
// same-association Storage Commitment N-EVENT-REPORT is role-reversed — the acceptor sends it as the
// N-EVENT-REPORT SCU and the requestor must receive it as the N-EVENT-REPORT SCP — so without the
// granted SCP role the acceptor would be transmitting in a role the peer did not negotiate. The first
// granting entry wins; an empty role set (default roles) yields false, so the report is skipped.
func requestorHasReportRole(acc *acse.Acceptor, sopClass dicom.UID) bool {
	for _, role := range acc.NegotiatedRoles() {
		if role.SOPClassUID == string(sopClass) && role.SCPRole {
			return true
		}
	}
	return false
}

// newNReportSender binds an NReportSender to one acceptor association, presentation context, and the
// reference object the originating N-ACTION named (its Requested SOP Class/Instance UID become the
// N-EVENT-REPORT's Affected pair, PS3.4 J.3.3: the report references the same well-known SOP Instance
// the N-ACTION targeted). Each call assigns a fresh non-zero Message ID, sends the N-EVENT-REPORT-RQ
// with the supplied event-information data set, and reads the requestor's N-EVENT-REPORT-RSP status.
func newNReportSender(acc *acse.Acceptor, pcID uint8, affectedSOPClass, affectedSOPInstance dicom.UID) NReportSender {
	var msgID uint16
	return func(ctx context.Context, eventTypeID uint16, ds *dicom.DataSet) (Status, error) {
		msgID++
		if msgID == 0 {
			msgID = 1
		}
		rq := CommandSet{
			CommandField:           CommandNEventReportRQ,
			MessageID:              msgID,
			AffectedSOPClassUID:    affectedSOPClass,
			AffectedSOPInstanceUID: affectedSOPInstance,
			HasEventTypeID:         true,
			EventTypeID:            eventTypeID,
			CommandDataSetType:     CommandDataSetNotPresent,
		}
		conn := acc.Conn()
		machine := acc.Machine()
		ts, err := acceptedTransferSyntaxResolver(acc)(pcID)
		if err != nil {
			return Status{}, err
		}
		if ds != nil {
			rq.CommandDataSetType = CommandDataSetPresent
			sendCap := MaxPDULength(acc.PeerMaxPDULength()).SendCap(defaultMaxPDULength)
			if serr := sendMessage(ctx, conn, machine, pcID, rq, ds, ts, sendCap); serr != nil {
				return Status{}, serr
			}
		} else if serr := sendCommand(ctx, conn, machine, pcID, rq); serr != nil {
			return Status{}, serr
		}

		reply, _, _, rerr := receiveMessage(ctx, conn, machine, newMessageReassembler(ts))
		if rerr != nil {
			return Status{}, rerr
		}
		if reply.CommandField != CommandNEventReportRSP {
			return Status{}, &ProtocolError{
				State:  machine.CurrentState(),
				Detail: "expected an N-EVENT-REPORT-RSP for the Storage Commitment report",
			}
		}
		if !reply.HasStatus {
			return Status{}, &ProtocolError{
				State:  machine.CurrentState(),
				Detail: "N-EVENT-REPORT-RSP missing the mandatory Status element",
			}
		}
		return NewStatus(reply.Status, ServiceClassStorageCommitment), nil
	}
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
