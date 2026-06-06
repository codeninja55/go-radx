package dimse

import (
	"context"
	"errors"
	"iter"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// Get issues a C-GET and yields each response Status (PS3.4 C.4.3). Unlike a C-MOVE, a C-GET carries
// no Move Destination: the peer (the C-GET SCP) C-STOREs each matched instance back to THIS requestor
// on the SAME association as a sub-operation, interleaved with the C-GET-RSP responses. The requestor
// must therefore act as the Storage SCP for those sub-operations, which requires the acceptor to have
// granted the Storage SCP role at negotiation (propose it with WithRoleSelection{<StorageSOPClassUID>,
// SCPRole: true}); the inbound sub-operation instances are dispatched to store, the StoreHandler sink
// the caller supplies, and the runtime answers each with a C-STORE-RSP carrying the handler's status.
//
// Each Pending (0xFF00) C-GET-RSP yields with a nil dataset (a C-GET-RSP carries no dataset) and
// refreshes the four sub-operation counts readable via Association.SubOperationCounts(); the terminal
// Success/Warning/Failure/Cancel status yields last and ends iteration, also refreshing the final
// counts. The 0xB000 "sub-operations complete, one or more failures or warnings" Warning is reported
// faithfully as a terminal status, NEVER laundered into Success (PRD §9.2): a partial-failure retrieve
// surfaces as a Warning or Failure final status the caller inspects, not a clean Success.
//
// The requested QueryLevel is written into Query/Retrieve Level (0008,0052) of a copy of the
// identifier before sending (DIMSE-015). Breaking out of the range loop, or cancelling ctx, sends a
// C-CANCEL-RQ for the operation's Message ID on the same presentation context. A transport or protocol
// fault that ends iteration before a clean terminal status is exposed via Association.LastError(),
// read immediately after the loop; a terminal Failure/Cancel Status is in-band data, not an error. A
// pre-flight fault (an unestablished or released association, no negotiated GET context, no granted
// Storage SCP role, a nil sink, or a non-GET model) yields a single terminal Failure status and sets
// LastError, never panicking (DIMSE-017).
//
// An Association is not safe for concurrent queries: LastError() and SubOperationCounts() are
// per-association. Run one Find/Get/Move iterator per association at a time.
func (a *Association) Get(
	ctx context.Context,
	query *dicom.DataSet,
	level QueryLevel,
	store StoreHandler,
	opts ...QueryOption,
) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {
		cfg := queryConfig{priority: PriorityMedium}
		for _, opt := range opts {
			opt(&cfg)
		}

		model := cfg.model
		if model == "" {
			model = defaultGetModel(level)
		}

		const svc = ServiceClassGet

		// Pre-flight before touching any per-association state, so a nil/unestablished association
		// yields a typed terminal Failure rather than panicking (DIMSE-017). It validates the model is
		// a C-GET information model, an accepted context for the model was negotiated, the level is one
		// of the four defined constants, a non-nil sink was supplied, and the association is
		// established and not released. Any failure is fail-closed: one terminal Failure plus
		// LastError, transmitting nothing (PRD §9.2).
		if err := a.getPreflight(model, level, store); err != nil {
			a.setLastError(err)
			yield(NewStatus(0xC000, svc), nil)
			return
		}
		a.clearLastError()
		a.clearSubOperationCounts()

		pcID, ts, _ := a.contextForQuery(model)

		// Write the Query/Retrieve Level into a COPY of the identifier so the caller's dataset is
		// untouched; a nil query becomes an empty identifier carrying only the level (DIMSE-015).
		identifier := dicom.NewDataSet()
		if query != nil {
			identifier = query.Clone()
		}
		identifier.SetString(dicom.TagQueryRetrieveLevel, level.String())

		conn := a.requestor.Conn()
		m := a.requestor.Machine()

		opCtx, cancel := a.dimseContext(ctx)
		defer cancel()

		msgID := a.nextMessageID()
		rq := CommandSet{
			CommandField:        CommandCGetRQ,
			MessageID:           msgID,
			AffectedSOPClassUID: dicom.UID(model),
			HasPriority:         true,
			Priority:            cfg.priority,
			CommandDataSetType:  CommandDataSetPresent,
		}
		if err := sendMessage(opCtx, conn, m, pcID, rq, identifier, ts, a.sendCap()); err != nil {
			a.setLastError(err)
			yield(NewStatus(0xC000, svc), nil)
			return
		}

		// Read inbound messages until the terminal C-GET-RSP. The SCP interleaves two kinds of message
		// on this association: C-STORE-RQ sub-operations (the matched instances, which the runtime
		// stores via the sink and answers with a C-STORE-RSP) and C-GET-RSP responses (Pending refresh
		// the counts and yield nil; the terminal yields last and ends iteration). An early break, or a
		// ctx cancellation interrupting a read, sends a C-CANCEL-RQ for this operation's Message ID and
		// drains the outstanding responses to the terminal, leaving the association clean for reuse.
		for {
			rsp, rspDS, inPCID, err := receiveMessage(opCtx, conn, m, newMessageReassemblerFunc(a.scuTransferSyntaxResolver()))
			if err != nil {
				if (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) && ctx.Err() != nil {
					a.cancelGetOperation(ctx, conn, m, pcID, ts, msgID, store)
					return
				}
				a.setLastError(err)
				yield(NewStatus(0xC000, svc), nil)
				return
			}

			// A C-STORE-RQ sub-operation: store the instance via the sink and answer with a C-STORE-RSP
			// on the same association. The handler's status is reported verbatim back to the SCP, which
			// accumulates the Completed/Failed/Warning counts from those statuses (the SCU must answer
			// each sub-operation, else the SCP wedges awaiting the response). A store fault on the wire
			// ends the retrieve via LastError.
			if rsp.CommandField == CommandCStoreRQ {
				if serr := a.answerGetSubOperation(opCtx, conn, m, store, rsp, rspDS, inPCID); serr != nil {
					a.setLastError(serr)
					yield(NewStatus(0xC000, svc), nil)
					return
				}
				continue
			}

			if rsp.CommandField != CommandCGetRSP {
				a.setLastError(&ProtocolError{
					State:  m.CurrentState(),
					Detail: "expected a C-GET-RSP or a C-STORE-RQ sub-operation in the response stream",
				})
				yield(NewStatus(0xC000, svc), nil)
				return
			}
			if !rsp.HasStatus {
				a.setLastError(&ProtocolError{
					State:  m.CurrentState(),
					Detail: "C-GET-RSP missing the mandatory Status element",
				})
				yield(NewStatus(0xC000, svc), nil)
				return
			}

			// Refresh the sub-operation counts a caller reads via SubOperationCounts(). A response that
			// omits the counts (some peers send a bare terminal) leaves the previous tallies in place.
			if rsp.HasSubOpCounts {
				a.setSubOperationCounts(SubOperationCounts{
					Remaining:      rsp.RemainingSubOperations,
					RemainingKnown: rsp.HasRemainingSubOp,
					Completed:      rsp.CompletedSubOperations,
					Failed:         rsp.FailedSubOperations,
					Warning:        rsp.WarningSubOperations,
				})
			}

			status := NewStatus(rsp.Status, svc)
			if status.IsPending() {
				// A Pending C-GET-RSP carries no dataset (only the sub-operation counts); yield nil.
				if !yield(status, nil) {
					a.cancelGetOperation(ctx, conn, m, pcID, ts, msgID, store)
					return
				}
				continue
			}
			// Terminal status (Success/Warning/Cancel/Failure): yield it last. A terminal Failure/Warning
			// C-GET-RSP MAY carry a Failed SOP Instance UID List (0008,0058) identifier; surface it
			// through the iterator's dataset slot rather than discarding it (rspDS is nil on a clean
			// Success, where the peer attached none).
			yield(status, rspDS)
			return
		}
	}
}

// answerGetSubOperation dispatches one inbound C-STORE-RQ sub-operation to the sink and writes the
// C-STORE-RSP carrying the handler's status on the same association. The dataset must be present (a
// C-STORE always carries the composite SOP Instance) and the command/dataset instance identity must
// agree, mirroring serveStoreMessage's guards; a malformed sub-operation is answered with a Failure
// status rather than stored, so the SCP still receives a response and the retrieve does not wedge. The
// info carries the Move Originator the SCP attached (the AE that invoked the C-GET) when present; it
// is a protocol identifier, never a patient value, so it is safe to log.
func (a *Association) answerGetSubOperation(
	ctx context.Context,
	conn *dul.Conn,
	m *dul.StateMachine,
	store StoreHandler,
	cmd CommandSet,
	ds *dicom.DataSet,
	pcID uint8,
) error {
	status := a.dispatchGetSubOperation(ctx, store, cmd, ds, pcID)
	rsp := CommandSet{
		CommandField:              CommandCStoreRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.AffectedSOPInstanceUID,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    status.Code,
	}
	return sendCommand(ctx, conn, m, pcID, rsp)
}

// dispatchGetSubOperation resolves the C-STORE-RSP status for one sub-operation: a malformed RQ (no
// dataset, or a command/dataset instance-identity mismatch) is answered with a Storage failure rather
// than stored (fail-closed, PRD §9.2), otherwise the sink's status is returned verbatim so the SCP
// accumulates the real Completed/Failed/Warning tally.
func (a *Association) dispatchGetSubOperation(ctx context.Context, store StoreHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8) Status {
	if ds == nil {
		return StatusStoreDataSetDoesNotMatchSOPClass
	}
	if err := validateStoreInstance(cmd, ds, a.requestor.State()); err != nil {
		return StatusStoreDataSetDoesNotMatchSOPClass
	}
	ts, _ := a.scuTransferSyntaxResolver()(pcID)
	info := OpInfo{
		PresentationID: pcID,
		TransferSyntax: ts,
		MessageID:      cmd.MessageID,
		SOPClassUID:    dicom.SOPClassUID(cmd.AffectedSOPClassUID),
	}
	if cmd.HasMoveOriginator {
		info.MoveOriginatorAETitle = cmd.MoveOriginatorAETitle
	}
	return store.Store(ctx, ds, info)
}

// cancelGetOperation sends a C-CANCEL-RQ for the in-flight C-GET on the same context, then drains the
// outstanding messages to the terminal C-GET-RSP, consuming any further C-STORE-RQ sub-operations the
// SCP already queued (answering each so the SCP is not left awaiting a response) so the association is
// left clean for reuse (PS3.7 §9.3.2.3). The cancel and drain ride a fresh context bounded by the
// DIMSE timeout so a cancelled or expired operation ctx does not also block them; both are
// best-effort.
func (a *Association) cancelGetOperation(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, pcID uint8, ts dicom.TransferSyntax, msgID uint16, store StoreHandler) {
	cancelCtx, cancel := a.cancelContext(ctx)
	defer cancel()
	cmd := CommandSet{
		CommandField:              CommandCCancelRQ,
		MessageIDBeingRespondedTo: msgID,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	if err := sendCommand(cancelCtx, conn, m, pcID, cmd); err != nil {
		a.setLastError(err)
		return
	}
	a.drainGetToTerminal(cancelCtx, conn, m, store)
}

// drainGetToTerminal reads C-GET messages until a terminal (non-pending) C-GET-RSP, answering any
// further C-STORE-RQ sub-operations the SCP already queued (so it is not left awaiting a response) and
// discarding the Pending responses, so a cancelled retrieve leaves the association stream clean for
// the next operation. It is best-effort and bounded by ctx; a read fault records LastError and stops.
func (a *Association) drainGetToTerminal(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, store StoreHandler) {
	for {
		rsp, rspDS, inPCID, err := receiveMessage(ctx, conn, m, newMessageReassemblerFunc(a.scuTransferSyntaxResolver()))
		if err != nil {
			a.setLastError(err)
			return
		}
		if rsp.CommandField == CommandCStoreRQ {
			if serr := a.answerGetSubOperation(ctx, conn, m, store, rsp, rspDS, inPCID); serr != nil {
				a.setLastError(serr)
				return
			}
			continue
		}
		if rsp.CommandField != CommandCGetRSP || !rsp.HasStatus {
			a.setLastError(&ProtocolError{
				State:  m.CurrentState(),
				Detail: "draining a cancelled C-GET: unexpected response while awaiting the terminal status",
			})
			return
		}
		if !NewStatus(rsp.Status, ServiceClassGet).IsPending() {
			return // terminal status reached: the stream is drained
		}
	}
}

// scuTransferSyntaxResolver maps an inbound presentation context ID (a C-GET sub-operation C-STORE-RQ
// arrives on a Storage context) to the negotiated transfer syntax for decoding its dataset. The
// requestor knows the accepted contexts up front (it proposed them), so it resolves the syntax by the
// inbound context ID; a sub-operation on an unaccepted context is a protocol fault.
func (a *Association) scuTransferSyntaxResolver() func(uint8) (dicom.TransferSyntax, error) {
	return func(pcID uint8) (dicom.TransferSyntax, error) {
		for _, pc := range a.accepted {
			if pc.ID != pcID || pc.Result != ContextAccepted {
				continue
			}
			if len(pc.TransferSyntaxes) == 0 {
				continue
			}
			return pc.TransferSyntaxes[0], nil
		}
		return "", &ProtocolError{
			State:  a.requestor.State(),
			Detail: "C-GET sub-operation arrived on a presentation context that was not accepted",
		}
	}
}

// defaultGetModel chooses the C-GET Information Model SOP Class for a level when the caller did not
// override it with WithQueryModel. A Patient-level retrieve is only valid in the Patient Root model;
// every other level defaults to Study Root GET, the common radiology entry point (PS3.4 C.6).
func defaultGetModel(level QueryLevel) dicom.SOPClassUID {
	if level == QueryLevelPatient {
		return patientRootGetSOPClass
	}
	return studyRootGetSOPClass
}

// getModels is the set of C-GET information-model SOP Classes Get may negotiate: the Patient Root and
// Study Root GET models. A WithQueryModel naming a FIND/MOVE (or any non-GET) class is rejected so Get
// never sends a C-GET-RQ whose Affected SOP Class is not a valid retrieve information model.
var getModels = map[dicom.SOPClassUID]struct{}{
	patientRootGetSOPClass: {},
	studyRootGetSOPClass:   {},
}

// getPreflight validates the association can run a C-GET for the given model and level: the model is a
// C-GET information model, the level is one of the four defined constants, a non-nil sink was supplied
// to receive the sub-operation instances, the association is established and not released, and an
// accepted presentation context for the model exists. It returns a typed error on any failure so Get
// can fail closed before any wire I/O (PRD §9.2).
func (a *Association) getPreflight(model dicom.SOPClassUID, level QueryLevel, store StoreHandler) error {
	if _, ok := getModels[model]; !ok {
		return &ValidationError{Detail: "query model " + string(model) + " is not a C-GET information model"}
	}
	if _, ok := queryLevelKeywords[level]; !ok {
		return &ValidationError{Detail: "unknown Query/Retrieve Level for a C-GET"}
	}
	if store == nil {
		return &ValidationError{Detail: "C-GET requires a non-nil StoreHandler sink to receive the sub-operation instances"}
	}
	if a == nil || a.requestor == nil {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Get on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Get on a released association"}
	}
	if _, _, ok := a.contextForQuery(model); !ok {
		return &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for Query/Retrieve model " + string(model),
		}
	}
	return nil
}
