package dimse

import (
	"context"
	"fmt"
	"iter"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/pdu"
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
	// MoveOriginatorAETitle is the Move Originator AE Title (0000,1030) a C-STORE-RQ carried when it
	// is a sub-operation of a C-MOVE/C-GET — the AE that invoked the originating retrieve (PS3.7
	// §9.1.1). It is empty for an everyday C-STORE that carries no originator. It is a protocol
	// identifier (an AE Title), never a patient value, so it is safe to log.
	MoveOriginatorAETitle AETitle
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

// FindHandler answers a C-FIND query, mirroring the SCU iterator (dimse.md "SCP handlers and the
// event model"). It yields one (Status, identifier) pair per match — a Pending status
// (0xFF00/0xFF01) carrying the matching identifier dataset — and the dispatcher sends one Pending
// C-FIND-RSP per yield. When the iterator ends, the dispatcher sends a terminal C-FIND-RSP carrying
// the iterator's final status (Success/Warning/Failure) and no dataset; a handler that yields no
// match at all still terminates with a single Success RSP (the no-hang contract). A worklist-only
// SCP implements FindHandler alone (interface segregation, PRD §8.2).
//
// The handler MUST observe its context: the dispatcher cancels it when the SCU sends a C-CANCEL-RQ
// mid-query (and on Server.Shutdown), so a handler that selects on ctx.Done() — or threads ctx
// through its match resolution — stops promptly. A handler that ignores its context cannot be woken
// (Go cannot forcibly kill a goroutine).
type FindHandler interface {
	// Find yields (Status, identifier) for each C-FIND match. A Pending status carries the matching
	// identifier; the iterator ends after the matches are exhausted.
	Find(ctx context.Context, query *dicom.DataSet, level QueryLevel, info OpInfo) iter.Seq2[Status, *dicom.DataSet]
}

// MoveHandler answers a C-MOVE retrieve, mirroring the SCU iterator (dimse.md "SCP handlers and the
// event model"). It yields one (Status, instance) pair per matched instance — a Pending status
// (0xFF00) carrying the matched instance dataset the runtime then C-STOREs to the resolved Move
// Destination AE as a sub-operation — and the iterator ends after the matches are exhausted. The
// runtime owns the destination association and the sub-operation store loop (each with a distinct
// non-zero Message ID, DIMSE-016) and the count accumulation; the handler supplies only the matched
// instances. A retrieve-only SCP implements MoveHandler alone (interface segregation, PRD §8.2).
//
// The handler MUST observe its context: the runtime cancels it on a fault, Server.Shutdown, or once
// the iteration ends, so a handler that threads ctx through its match resolution stops promptly.
type MoveHandler interface {
	// Move yields (Status, instance) for each matched instance a C-MOVE should retrieve to dest. A
	// Pending status carries the matched instance dataset; the iterator ends after the matches are
	// exhausted (an optional terminal status from the handler ends the matches early).
	Move(ctx context.Context, query *dicom.DataSet, level QueryLevel, dest AETitle, info OpInfo) iter.Seq2[Status, *dicom.DataSet]
}

// Handler answers inbound DIMSE-C operations dispatched by the SCP. An intervention operation is
// answered with a typed Status, so a handler cannot forget to answer (PRD §8.2). It is the union
// of the per-service capabilities; an SCP that supports only some services implements the narrower
// interfaces (EchoHandler, StoreHandler, FindHandler, MoveHandler) and the dispatcher type-asserts
// for each. A handler returning success on work it did not do is a defect (PRD §9.2 fail-closed).
// The remaining query/retrieve capability (Get) joins this union with its service in a later M3
// increment.
//
// A handler MUST observe the context it is passed: Server.Shutdown cancels it, and a handler that
// selects on ctx.Done() (or threads ctx through its I/O) is woken cooperatively and returns
// promptly. A handler that ignores its context and is not blocked in a connection read cannot be
// woken — Go cannot forcibly kill a goroutine — and so can outlive Shutdown's deadline.
type Handler interface {
	EchoHandler
	StoreHandler
	FindHandler
	MoveHandler
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
	// Surface the Move Originator AE Title when the C-STORE is a C-MOVE/C-GET sub-operation, so a
	// destination handler can attribute the instance to the AE that invoked the retrieve (PS3.7 §9.1.1).
	if cmd.HasMoveOriginator {
		info.MoveOriginatorAETitle = cmd.MoveOriginatorAETitle
	}
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
// UID) negotiated for it, but ONLY when that context was ACCEPTED (Result 0). It binds the pure
// abstractSyntaxFor to this acceptor's requested and accepted contexts; both validateEchoContext
// and validateStoreContext share it, so the acceptance requirement covers both.
func acceptedAbstractSyntaxResolver(acc *acse.Acceptor) func(uint8) (dicom.SOPClassUID, bool) {
	requested := acc.RequestedContexts()
	accepted := acc.AcceptedContexts()
	return func(pcID uint8) (dicom.SOPClassUID, bool) {
		return abstractSyntaxFor(requested, accepted, pcID)
	}
}

// abstractSyntaxFor resolves the abstract syntax (SOP Class UID) for a presentation context ID,
// returning (abstract, true) ONLY when the context was ACCEPTED — present in accepted with
// Result 0 (acceptance, PS3.8 §9.3.3.2) — AND has a matching proposed entry in requested (the
// A-ASSOCIATE-RQ carries the abstract syntax; the A-ASSOCIATE-AC echoes only the ID and result).
// Requiring acceptance closes the bypass where a peer operates on a context the acceptor REJECTED
// during negotiation: a proposed-but-rejected context has no negotiated abstract syntax, so a
// C-ECHO/C-STORE on its ID must not validate (P2 adversarial review).
func abstractSyntaxFor(requested []pdu.PresentationContextRQ, accepted []pdu.PresentationContextAC, pcID uint8) (dicom.SOPClassUID, bool) {
	acceptedContext := false
	for _, pc := range accepted {
		if pc.ID == pcID {
			if pc.Result != pdu.PresentationContextAcceptance {
				return "", false
			}
			acceptedContext = true
			break
		}
	}
	if !acceptedContext {
		return "", false
	}
	for _, pc := range requested {
		if pc.ID == pcID {
			return dicom.SOPClassUID(pc.AbstractSyntax), true
		}
	}
	return "", false
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

// validateEchoContext fails closed when a C-ECHO-RQ arrives on a presentation context whose
// negotiated abstract syntax is not the Verification SOP Class, OR when the command's Affected SOP
// Class UID is not the Verification SOP Class. Either lets a peer run Verification outside the
// negotiated/declared SOP Class, bypassing presentation-context negotiation (PS3.7 §9.1.5 makes the
// command's Affected SOP Class UID Type 1) — the same protocol fault the C-STORE path rejects with
// validateStoreContext, kept symmetric in both the context and the command checks.
func validateEchoContext(cmd CommandSet, pcID uint8, abstractFor func(uint8) (dicom.SOPClassUID, bool), state State) error {
	abstract, ok := abstractFor(pcID)
	if !ok || abstract != verificationSOPClass {
		return &ProtocolError{State: state, Detail: fmt.Sprintf(
			"C-ECHO arrived on presentation context %d whose abstract syntax %q is not the Verification SOP Class",
			pcID, abstract)}
	}
	if dicom.SOPClassUID(cmd.AffectedSOPClassUID) != verificationSOPClass {
		return &ProtocolError{State: state, Detail: fmt.Sprintf(
			"C-ECHO-RQ Affected SOP Class %q is not the Verification SOP Class", cmd.AffectedSOPClassUID)}
	}
	return nil
}

// validateFindContext fails closed when a C-FIND-RQ arrives on a presentation context whose
// negotiated abstract syntax is not a Query/Retrieve FIND information model, OR when the command's
// Affected SOP Class UID is not that same negotiated FIND model. Either lets a peer run a query
// outside the negotiated/declared SOP Class, bypassing presentation-context negotiation (PS3.4 C.4
// makes the C-FIND-RQ Affected SOP Class UID Type 1) — the same protocol fault the C-STORE and
// C-ECHO paths reject, kept symmetric across the context and the command checks.
func validateFindContext(cmd CommandSet, pcID uint8, abstractFor func(uint8) (dicom.SOPClassUID, bool), state State) error {
	abstract, ok := abstractFor(pcID)
	if !ok || !isFindModel(abstract) {
		return &ProtocolError{State: state, Detail: fmt.Sprintf(
			"C-FIND arrived on presentation context %d whose abstract syntax %q is not a Query/Retrieve FIND information model",
			pcID, abstract)}
	}
	if dicom.SOPClassUID(cmd.AffectedSOPClassUID) != abstract {
		return &ProtocolError{State: state, Detail: fmt.Sprintf(
			"C-FIND Affected SOP Class %q does not match the abstract syntax negotiated for presentation context %d",
			cmd.AffectedSOPClassUID, pcID)}
	}
	return nil
}

// isFindModel reports whether the SOP Class is one of the C-FIND information models go-radx serves
// as an SCP (the Patient Root / Study Root FIND models and the Modality Worklist FIND model). It
// reuses the findModels set the SCU side validates a WithQueryModel against, so the SCU and SCP
// agree on what a FIND context is.
func isFindModel(sopClass dicom.SOPClassUID) bool {
	_, ok := findModels[sopClass]
	return ok
}

// validateMoveContext fails closed when a C-MOVE-RQ arrives on a presentation context whose
// negotiated abstract syntax is not a Query/Retrieve MOVE information model, OR when the command's
// Affected SOP Class UID is not that same negotiated MOVE model. Either lets a peer run a retrieve
// outside the negotiated/declared SOP Class, bypassing presentation-context negotiation (PS3.4 C.4.2
// makes the C-MOVE-RQ Affected SOP Class UID Type 1) — the same protocol fault the C-FIND/C-STORE/
// C-ECHO paths reject, kept symmetric across the context and the command checks.
func validateMoveContext(cmd CommandSet, pcID uint8, abstractFor func(uint8) (dicom.SOPClassUID, bool), state State) error {
	abstract, ok := abstractFor(pcID)
	if !ok || !isMoveModel(abstract) {
		return &ProtocolError{State: state, Detail: fmt.Sprintf(
			"C-MOVE arrived on presentation context %d whose abstract syntax %q is not a Query/Retrieve MOVE information model",
			pcID, abstract)}
	}
	if dicom.SOPClassUID(cmd.AffectedSOPClassUID) != abstract {
		return &ProtocolError{State: state, Detail: fmt.Sprintf(
			"C-MOVE Affected SOP Class %q does not match the abstract syntax negotiated for presentation context %d",
			cmd.AffectedSOPClassUID, pcID)}
	}
	return nil
}

// isMoveModel reports whether the SOP Class is one of the C-MOVE information models go-radx serves
// as an SCP (the Patient Root / Study Root MOVE models). It reuses the moveModels set the SCU side
// validates a WithQueryModel against, so the SCU and SCP agree on what a MOVE context is.
func isMoveModel(sopClass dicom.SOPClassUID) bool {
	_, ok := moveModels[sopClass]
	return ok
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
