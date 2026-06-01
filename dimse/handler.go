package dimse

import (
	"context"
	"fmt"

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

// EchoHandler answers a C-ECHO (Verification). A store-only SCP need not implement it; the
// dispatcher type-asserts for the capability (interface segregation, PRD §8.2).
type EchoHandler interface {
	// Echo answers a C-ECHO. Return StatusEchoSuccess unless the SCP is degraded.
	Echo(ctx context.Context, info OpInfo) Status
}

// StoreHandler receives a single dataset, as a C-STORE SCP (and, in later increments, as the
// C-GET sub-operation sink on the requestor). Persisting the dataset before returning success is
// the handler's responsibility; returning success without storing violates the honest-failure
// rule (PRD §9.2 fail-closed).
type StoreHandler interface {
	// Store receives one dataset (C-STORE). Return StatusStoreSuccess only after persisting it.
	Store(ctx context.Context, ds *dicom.DataSet, info OpInfo) Status
}

// Handler answers inbound DIMSE-C operations dispatched by the SCP. An intervention operation is
// answered with a typed Status, so a handler cannot forget to answer (PRD §8.2). It is the union
// of the per-service capabilities; an SCP that supports only some services implements the narrower
// interfaces (EchoHandler, StoreHandler) and the dispatcher type-asserts for each. A handler
// returning success on work it did not do is a defect (PRD §9.2 fail-closed). The query/retrieve
// capabilities (Find/Get/Move) join this union with their services in M3.
type Handler interface {
	EchoHandler
	StoreHandler
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
func serveEcho(ctx context.Context, acc *acse.Acceptor, calling, called AETitle, h EchoHandler) (Status, error) {
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
	return serveEchoCommand(ctx, acc, h, cmd, pcID, info)
}

// serveEchoCommand dispatches an already-read C-ECHO-RQ command set to the handler and writes the
// C-ECHO-RSP. The SCP dispatch loop (Increment 6) reads each inbound message once and routes by
// command field, calling this so the C-ECHO response logic is not duplicated; serveEcho is the
// single-shot wrapper that reads then dispatches (preserving the in-process unit-test contract).
//
// The response echoes the request's Affected SOP Class UID and Message ID into the
// Message ID Being Responded To, carries no data set (CommandDataSetType 0x0101), and reports
// the handler's status — the verification contract (PS3.7 §9.1.5).
func serveEchoCommand(ctx context.Context, acc *acse.Acceptor, h EchoHandler, cmd CommandSet, pcID uint8, info OpInfo) (Status, error) {
	status := h.Echo(ctx, info)

	rsp := CommandSet{
		CommandField:              CommandCEchoRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    status.Code,
	}
	if err := sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp); err != nil {
		return Status{}, err
	}
	return status, nil
}

// serveStore services one inbound C-STORE over an established acceptor association: it reads the
// C-STORE-RQ command set and the dataset (through dul.DriveInbound, decoding the dataset with the
// negotiated transfer syntax of the presentation context the RQ arrived on — DIMSE-003), dispatches
// to the StoreHandler, and writes the C-STORE-RSP carrying the handler's status. The SCP server
// scaffolding (Increment 6) drives this per accepted association.
//
// The handler is responsible for persisting the dataset before returning a success status; the
// dispatch never launders a non-success status into success (PRD §9.2 fail-closed). A C-STORE-RQ
// that arrives with no dataset (CommandDataSetType not present) is malformed — a C-STORE always
// carries the composite SOP Instance.
func serveStore(ctx context.Context, acc *acse.Acceptor, calling, called AETitle, h StoreHandler) (Status, error) {
	conn := acc.Conn()
	m := acc.Machine()

	r := newMessageReassemblerFunc(acceptedTransferSyntaxResolver(acc))
	cmd, ds, pcID, err := receiveMessage(ctx, conn, m, r)
	if err != nil {
		return Status{}, err
	}
	info := OpInfo{CallingAETitle: calling, CalledAETitle: called}
	return serveStoreMessage(ctx, acc, h, cmd, ds, pcID, info)
}

// serveStoreMessage dispatches an already-read C-STORE-RQ (command set plus dataset) to the
// StoreHandler and writes the C-STORE-RSP. The SCP dispatch loop reads each inbound message once
// and routes by command field, calling this; serveStore is the single-shot wrapper that reads then
// dispatches. info carries the AE titles the caller resolved; this fills in the per-operation
// presentation/transfer/SOP fields from the read message.
//
// The handler is responsible for persisting the dataset before returning a success status; the
// dispatch never launders a non-success status into success (PRD §9.2 fail-closed). A C-STORE-RQ
// that arrives with no dataset (CommandDataSetType not present) is malformed — a C-STORE always
// carries the composite SOP Instance.
func serveStoreMessage(ctx context.Context, acc *acse.Acceptor, h StoreHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, info OpInfo) (Status, error) {
	m := acc.Machine()
	if cmd.CommandField != CommandCStoreRQ {
		return Status{}, &ProtocolError{
			State:  m.CurrentState(),
			Detail: "expected a C-STORE-RQ command field, got a different command",
		}
	}
	if ds == nil {
		return Status{}, &ProtocolError{
			State:  m.CurrentState(),
			Detail: "C-STORE-RQ declared no data set; a C-STORE always carries the composite SOP Instance",
		}
	}
	// The C-STORE's Affected SOP Class UID must match the abstract syntax negotiated for the
	// context it arrived on; a mismatch bypasses presentation-context negotiation (storing an
	// unnegotiated SOP Class on an accepted context) and is a protocol fault.
	if err := validateStoreContext(cmd, pcID, acceptedAbstractSyntaxResolver(acc), m.CurrentState()); err != nil {
		return Status{}, err
	}
	// The command and dataset must agree on the instance identity, else the SCP would persist
	// one instance while acknowledging another (the RSP echoes the command's Affected SOP
	// Instance UID), or emit a malformed RSP with an empty mandatory UID.
	if err := validateStoreInstance(cmd, ds, m.CurrentState()); err != nil {
		return Status{}, err
	}

	ts, _ := acceptedTransferSyntaxResolver(acc)(pcID)
	info.PresentationID = pcID
	info.TransferSyntax = ts
	info.MessageID = cmd.MessageID
	info.SOPClassUID = dicom.SOPClassUID(cmd.AffectedSOPClassUID)
	status := h.Store(ctx, ds, info)

	rsp := CommandSet{
		CommandField:              CommandCStoreRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.AffectedSOPInstanceUID,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    status.Code,
	}
	if err := sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp); err != nil {
		return Status{}, err
	}
	return status, nil
}

// acceptedTransferSyntaxResolver maps a presentation context ID to the transfer syntax the
// acceptor negotiated for it. A C-STORE-RQ arriving on a context the acceptor did not accept is a
// protocol fault (the peer must not transmit on an unaccepted context).
func acceptedTransferSyntaxResolver(acc *acse.Acceptor) func(uint8) (dicom.TransferSyntax, error) {
	accepted := acc.AcceptedContexts()
	return func(pcID uint8) (dicom.TransferSyntax, error) {
		for _, pc := range accepted {
			if pc.ID == pcID && pc.Result == 0 { // 0 == acceptance (PS3.8 9.3.3.2)
				return dicom.TransferSyntax(pc.TransferSyntax), nil
			}
		}
		return "", &ProtocolError{
			State:  acc.Machine().CurrentState(),
			Detail: "C-STORE-RQ arrived on a presentation context that was not accepted",
		}
	}
}

// acceptedAbstractSyntaxResolver maps a presentation context ID to the abstract syntax (SOP Class
// UID) the peer proposed for it, from the requested contexts in the A-ASSOCIATE-RQ. The acceptor
// accepted the context for that abstract syntax, so it is the SOP Class a C-STORE on that context
// must carry.
func acceptedAbstractSyntaxResolver(acc *acse.Acceptor) func(uint8) (dicom.SOPClassUID, bool) {
	requested := acc.RequestedContexts()
	return func(pcID uint8) (dicom.SOPClassUID, bool) {
		for _, pc := range requested {
			if pc.ID == pcID {
				return dicom.SOPClassUID(pc.AbstractSyntax), true
			}
		}
		return "", false
	}
}

// validateStoreContext fails closed when a C-STORE-RQ's Affected SOP Class UID does not match the
// abstract syntax negotiated for the presentation context it arrived on. Accepting a mismatch
// would let a peer store an unnegotiated SOP Class on an accepted context, bypassing
// presentation-context negotiation (PS3.7/PS3.8) — a protocol fault, not a storage failure.
func validateStoreContext(cmd CommandSet, pcID uint8, abstractFor func(uint8) (dicom.SOPClassUID, bool), state State) error {
	abstract, ok := abstractFor(pcID)
	if !ok || dicom.SOPClassUID(cmd.AffectedSOPClassUID) != abstract {
		return &ProtocolError{State: state, Detail: fmt.Sprintf(
			"C-STORE Affected SOP Class %s does not match the abstract syntax negotiated for presentation context %d",
			cmd.AffectedSOPClassUID, pcID)}
	}
	return nil
}

// validateStoreInstance fails closed when a C-STORE-RQ's mandatory Affected SOP Instance UID is
// absent or disagrees with the dataset's own SOP Instance UID (0008,0018). Storing on a mismatch
// would let the SCP persist one instance while acknowledging another (the RSP echoes the command
// UID), or emit an RSP missing the mandatory UID — an integrity fault, not a storage failure.
func validateStoreInstance(cmd CommandSet, ds *dicom.DataSet, state State) error {
	dsInstance, _ := ds.GetString(tagSOPInstanceUID)
	if cmd.AffectedSOPInstanceUID == "" || string(cmd.AffectedSOPInstanceUID) != dsInstance {
		return &ProtocolError{State: state, Detail: fmt.Sprintf(
			"C-STORE Affected SOP Instance UID %q does not match the dataset SOP Instance UID %q",
			cmd.AffectedSOPInstanceUID, dsInstance)}
	}
	return nil
}
