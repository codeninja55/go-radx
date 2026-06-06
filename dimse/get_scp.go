package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
)

// serveGetMessage services one inbound C-GET-RQ over an established acceptor association. It validates
// the negotiated context is a Query/Retrieve GET model (the guard analogous to validateMoveContext),
// reads the query identifier and level, drives the GetHandler's iterator, and C-STOREs each matched
// instance back to the requestor on the SAME association as a sub-operation — each with a distinct,
// non-zero Message ID and the Move Originator set to the AE that invoked the C-GET (PS3.7 §9.1.1),
// reading the requestor's C-STORE-RSP back on the same association. This same-association store is
// what distinguishes C-GET from C-MOVE: there is no separate destination AE, so the requestor must
// have been granted the Storage SCP role at negotiation. It reports the running
// Completed/Failed/Warning counts in one Pending C-GET-RSP per matched instance, then a terminal
// C-GET-RSP: 0x0000 when all sub-operations succeeded, 0xA702 when every one failed, and 0xB000
// (Warning) when some failed or warned (never laundered into Success, PRD §9.2).
//
// A sub-operation C-STORE that faults on the wire ends the retrieve with a terminal Failure (the
// association is broken); a sub-operation the requestor rejects (a Failure C-STORE-RSP status) counts
// as a failed sub-operation and the terminal status surfaces it as a Warning or all-failed Failure.
// The handler's context is derived from ctx, so Server.Shutdown stops an in-flight retrieve.
func serveGetMessage(ctx context.Context, acc *acse.Acceptor, h GetHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, info OpInfo) error {
	m := acc.Machine()
	if err := validateGetContext(cmd, pcID, acceptedAbstractSyntaxResolver(acc), m.CurrentState()); err != nil {
		return err
	}

	model := dicom.SOPClassUID(cmd.AffectedSOPClassUID)

	// A C-GET-RQ ALWAYS carries an identifier (CommandDataSetType present); a request that declares
	// none is malformed. Fail closed with 0xA900 rather than fabricating an empty identifier (PRD §9.2).
	if ds == nil {
		return sendGetResponse(ctx, acc, cmd, pcID, NewStatus(0xA900, ServiceClassGet), SubOperationCounts{})
	}
	query := ds

	// The query identifier carries the Query/Retrieve Level (0008,0052) the SCU wrote (DIMSE-015). A
	// missing or unparseable level for a levelled GET model is a malformed query: fail closed.
	level, levelErr := queryLevelFromIdentifier(query, model)
	if levelErr != nil {
		return sendGetResponse(ctx, acc, cmd, pcID, NewStatus(0xA900, ServiceClassGet), SubOperationCounts{})
	}

	ts, _ := acceptedTransferSyntaxResolver(acc)(pcID)
	info.PresentationID = pcID
	info.TransferSyntax = ts
	info.MessageID = cmd.MessageID
	info.SOPClassUID = model

	// Derive a cancellable context for the handler so Server.Shutdown stops its iterator.
	handlerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	store := &getSubOperationStore{acc: acc, originator: info.CallingAETitle, getMessageID: cmd.MessageID}

	var counts SubOperationCounts
	for status, instance := range h.Get(handlerCtx, query, level, info) {
		// Cooperative shutdown: if the dispatch context ended (Server.Shutdown) while the handler was
		// producing matches, stop promptly and return the context error rather than sending more RSPs
		// over a connection that is being torn down (DIMSE-014 applied to the C-GET drain).
		if err := ctx.Err(); err != nil {
			return err
		}
		// A non-Pending yield signals the handler's matches are done. A handler-level FAILURE (e.g.
		// out-of-resources resolving the matches) propagates as the terminal status. A handler-level
		// Success/Warning is NOT forwarded verbatim: the handler does not know the sub-operation store
		// outcomes, so the runtime computes the real terminal from the accumulated counts (a partial
		// store failure must surface as 0xB000, never be laundered into the handler's Success — PRD §9.2).
		if !status.IsPending() {
			if status.IsFailure() {
				return sendGetResponse(ctx, acc, cmd, pcID, status, counts)
			}
			return sendGetResponse(ctx, acc, cmd, pcID, getTerminalStatus(counts), counts)
		}
		if instance == nil {
			// A Pending match must carry an instance to store; a nil instance is a handler bug. Count it
			// as a failed sub-operation rather than panicking (fail-closed).
			counts.Failed++
			if perr := sendGetResponse(ctx, acc, cmd, pcID, StatusGetPending, counts); perr != nil {
				return perr
			}
			continue
		}

		// C-STORE the matched instance back to the requestor on the SAME association as a sub-operation.
		// A wire fault here breaks the association: end the retrieve with a terminal Failure rather than
		// reporting partial progress over a connection that is gone (PRD §9.2/§9.4).
		subStatus, storeErr := store.send(ctx, instance)
		if storeErr != nil {
			return storeErr
		}
		switch {
		case subStatus.IsWarning():
			counts.Warning++
		case subStatus.IsSuccess():
			counts.Completed++
		default:
			counts.Failed++
		}

		if perr := sendGetResponse(ctx, acc, cmd, pcID, StatusGetPending, counts); perr != nil {
			return perr
		}
	}

	// The handler may have ended only because its context was cancelled (Server.Shutdown woke it
	// between yields). Return the context error rather than sending a terminal over a connection being
	// torn down, so a cooperatively-shut-down retrieve does not race a final RSP against the close.
	if err := ctx.Err(); err != nil {
		return err
	}

	// All matches exhausted: send the terminal C-GET-RSP carrying the final counts. The status is
	// faithful — Success only when nothing failed or warned, the all-failed 0xA702 when every
	// sub-operation failed, otherwise the 0xB000 partial-failure Warning (PRD §9.2 fail-closed).
	return sendGetResponse(ctx, acc, cmd, pcID, getTerminalStatus(counts), counts)
}

// getSubOperationStore C-STOREs C-GET sub-operation instances back to the requestor on the acceptor
// association. It allocates a distinct, non-zero Message ID per sub-operation (DIMSE-016), selects the
// requestor's accepted Storage presentation context for each instance's SOP Class, and reads the
// matching C-STORE-RSP, returning that status to the caller. The originator (the AE that invoked the
// C-GET — the calling AE of this inbound association) and the original C-GET Message ID are propagated
// so the requestor can attribute the instances to the retrieve it asked for (PS3.7 §9.1.1).
type getSubOperationStore struct {
	acc          *acse.Acceptor
	originator   AETitle
	getMessageID uint16
	nextMsgID    uint16
}

// send transmits one sub-operation C-STORE-RQ and reads its C-STORE-RSP on the acceptor association,
// returning the requestor's status. A returned error is a wire/protocol fault (the association is
// broken); a Failure-category status is in-band data the caller accumulates as a failed sub-operation,
// never an error. A SOP Class with no accepted Storage context selects no context, so the instance
// cannot be sent: it is reported as a failed sub-operation via a Storage failure status (fail-closed),
// not a fault, so one unstorable instance does not break the whole retrieve.
func (s *getSubOperationStore) send(ctx context.Context, ds *dicom.DataSet) (Status, error) {
	sopClass, _ := ds.GetString(tagSOPClassUID)
	sopInstance, _ := ds.GetString(tagSOPInstanceUID)
	if sopClass == "" || sopInstance == "" {
		return StatusStoreDataSetDoesNotMatchSOPClass, nil
	}
	pcID, ts, ok := s.contextForStorage(dicom.SOPClassUID(sopClass))
	if !ok {
		return StatusStoreCannotUnderstand, nil
	}

	s.nextMsgID++
	if s.nextMsgID == 0 {
		s.nextMsgID = 1
	}
	rq := CommandSet{
		CommandField:            CommandCStoreRQ,
		MessageID:               s.nextMsgID,
		AffectedSOPClassUID:     dicom.UID(sopClass),
		AffectedSOPInstanceUID:  dicom.UID(sopInstance),
		HasPriority:             true,
		Priority:                PriorityMedium,
		CommandDataSetType:      CommandDataSetPresent,
		HasMoveOriginator:       true,
		MoveOriginatorAETitle:   s.originator,
		MoveOriginatorMessageID: s.getMessageID,
	}
	conn := s.acc.Conn()
	m := s.acc.Machine()
	sendCap := MaxPDULength(s.acc.PeerMaxPDULength()).SendCap(defaultMaxPDULength)
	if err := sendMessage(ctx, conn, m, pcID, rq, ds, ts, sendCap); err != nil {
		return Status{}, err
	}

	rsp, _, _, err := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if err != nil {
		return Status{}, err
	}
	if rsp.CommandField != CommandCStoreRSP || !rsp.HasStatus {
		return Status{}, &ProtocolError{
			State:  m.CurrentState(),
			Detail: "expected a C-STORE-RSP for the C-GET sub-operation",
		}
	}
	return NewStatus(rsp.Status, ServiceClassStorage), nil
}

// contextForStorage returns the acceptor's accepted presentation context ID and its negotiated
// transfer syntax for the given Storage SOP Class, or false when none was accepted. Selection is by
// abstract syntax; the first accepted matching context wins.
func (s *getSubOperationStore) contextForStorage(sopClass dicom.SOPClassUID) (uint8, dicom.TransferSyntax, bool) {
	requested := s.acc.RequestedContexts()
	for _, pc := range s.acc.AcceptedContexts() {
		if pc.Result != 0 { // 0 == acceptance (PS3.8 9.3.3.2)
			continue
		}
		for _, rq := range requested {
			if rq.ID == pc.ID && dicom.SOPClassUID(rq.AbstractSyntax) == sopClass {
				return pc.ID, dicom.TransferSyntax(pc.TransferSyntax), true
			}
		}
	}
	return 0, "", false
}

// getTerminalStatus resolves the terminal C-GET-RSP status from the accumulated sub-operation counts
// (parity with pynetdicom's QueryRetrieveGetServiceClass and moveTerminalStatus): a clean Success only
// when every sub-operation completed without failure or warning, the 0xA702 "unable to perform
// sub-operations" Failure when every attempted sub-operation failed, and the 0xB000 "Sub-operations
// Complete — One or More Failures or Warnings" Warning when at least one sub-operation failed or
// warned but not all failed (PS3.4 C.4.3.1.4 — a warning is not laundered into Success, PRD §9.2).
func getTerminalStatus(counts SubOperationCounts) Status {
	if counts.Failed == 0 && counts.Warning == 0 {
		return StatusGetSuccess
	}
	attempted := counts.Completed + counts.Failed + counts.Warning
	if counts.Failed == attempted {
		return NewStatus(0xA702, ServiceClassGet)
	}
	return StatusGetSubOpsCompleteWithFailures
}

// sendGetResponse writes one C-GET-RSP carrying the status and the running Completed/Failed/Warning
// sub-operation counts (a C-GET-RSP never carries a dataset), echoing the request's Affected SOP Class
// UID and Message ID into the Message ID Being Responded To. Both Pending and terminal responses carry
// the tallies so the SCU can track progress. NumberOfRemainingSubOperations (0000,1020) is OMITTED:
// the handler streams matches without a pre-count, so the SCP does not know the outstanding total, and
// PS3.4 C.4.3.1.4 makes that element conditional (present only when known, absent on the terminal).
func sendGetResponse(ctx context.Context, acc *acse.Acceptor, cmd CommandSet, pcID uint8, status Status, counts SubOperationCounts) error {
	rsp := CommandSet{
		CommandField:              CommandCGetRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		HasStatus:                 true,
		Status:                    status.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasSubOpCounts:            true,
		OmitRemainingSubOp:        true,
		CompletedSubOperations:    counts.Completed,
		FailedSubOperations:       counts.Failed,
		WarningSubOperations:      counts.Warning,
	}
	return sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, rsp)
}

// refuseUnsupportedGet answers a C-GET-RQ that reached a handler with no Get capability: it writes a
// single terminal C-GET-RSP carrying StatusSOPClassNotSupported (0x0122) and no dataset, so the peer
// learns the retrieve service is unsupported rather than the SCP panicking or aborting (interface
// segregation, PRD §8.2). The query identifier the RQ carried has already been read from the wire by
// the dispatch loop and is discarded.
func refuseUnsupportedGet(ctx context.Context, acc *acse.Acceptor, cmd CommandSet, pcID uint8) error {
	return sendGetResponse(ctx, acc, cmd, pcID, StatusSOPClassNotSupported, SubOperationCounts{})
}
