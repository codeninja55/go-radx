package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
)

// moveSupport carries the dependencies the C-MOVE SCP dispatch needs beyond the handler: the
// Server's local AE (the calling AE that opens the outbound sub-operation association to the
// destination) and the configured Move Destination AE-Title → address table. It is passed by value
// through the dispatch path; a Server with no C-MOVE destinations leaves destinations nil and every
// move is refused as "Move Destination Unknown".
type moveSupport struct {
	ae           *AE
	destinations map[AETitle]string
}

// resolveDestination maps a Move Destination AE Title to its network address, returning false when
// the destination is not in the configured table (an unresolvable destination, answered 0xA801).
func (s moveSupport) resolveDestination(dest AETitle) (string, bool) {
	addr, ok := s.destinations[dest]
	return addr, ok
}

// serveMoveMessage services one inbound C-MOVE-RQ over an established acceptor association. It
// validates the negotiated context is a Query/Retrieve MOVE model (the guard analogous to
// validateFindContext), reads the query identifier and level, resolves the Move Destination AE
// against the Server's known-AE table, opens a SEPARATE outbound association to that destination,
// drives the MoveHandler's iterator, and C-STOREs each matched instance to the destination as a
// sub-operation — each with a distinct, non-zero Message ID read through the full receiveMessage
// reassembly loop (DIMSE-016). It reports the running Remaining/Completed/Failed/Warning counts in
// one Pending C-MOVE-RSP per matched instance, then a terminal C-MOVE-RSP: 0x0000 when all
// sub-operations succeeded, 0xA702 when every one failed, and 0xB000 (Warning) when some failed
// (never laundered into Success, PRD §9.2). An unresolvable destination is answered with a single
// terminal 0xA801 RSP before any match is requested.
//
// The outbound destination association is opened from this dispatch goroutine, released cleanly when
// the move ends (success, failure, or fault), and never left dangling (PRD §9.4); a failure to open
// it is a terminal 0xA702, not a panic. The handler's context is derived from ctx, so Server.Shutdown
// (which cancels ctx) stops an in-flight move and the destination association is torn down.
//
// The dispatcher honours a C-CANCEL-RQ arriving on the inbound association mid-retrieve by cancelling
// the handler's context (so no further sub-operation is dispatched) and sending the terminal Cancel
// RSP carrying the counts accumulated so far (PS3.4 C.4.2.3, PS3.7 §9.3.2.3), reusing the C-FIND
// cancel watcher: the sub-operation C-STOREs ride the separate destination association, so the only
// valid inbound message while a move is in flight is a C-CANCEL-RQ.
func serveMoveMessage(ctx context.Context, acc *acse.Acceptor, h MoveHandler, move moveSupport, cmd CommandSet, ds *dicom.DataSet, pcID uint8, info OpInfo) error {
	m := acc.Machine()
	if err := validateMoveContext(cmd, pcID, acceptedAbstractSyntaxResolver(acc), m.CurrentState()); err != nil {
		return err
	}

	model := dicom.SOPClassUID(cmd.AffectedSOPClassUID)

	// A C-MOVE-RQ ALWAYS carries an identifier (CommandDataSetType present); a request that declares
	// none is malformed. Fail closed with 0xA900 rather than fabricating an empty identifier (PRD §9.2).
	if ds == nil {
		return sendMoveResponse(ctx, acc, cmd, pcID, NewStatus(0xA900, ServiceClassMove), SubOperationCounts{})
	}
	query := ds

	// The query identifier carries the Query/Retrieve Level (0008,0052) the SCU wrote (DIMSE-015). A
	// missing or unparseable level for a levelled MOVE model is a malformed query: fail closed.
	level, levelErr := queryLevelFromIdentifier(query, model)
	if levelErr != nil {
		return sendMoveResponse(ctx, acc, cmd, pcID, NewStatus(0xA900, ServiceClassMove), SubOperationCounts{})
	}

	dest := cmd.MoveDestination
	destAddr, ok := move.resolveDestination(dest)
	if !ok || move.ae == nil {
		// The Move Destination AE Title is not in the Server's known-AE table: the retrieve cannot be
		// performed. Answer the terminal 0xA801 before requesting any match (PS3.4 C.4.2.1.5).
		return sendMoveResponse(ctx, acc, cmd, pcID, StatusMoveDestinationUnknown, SubOperationCounts{})
	}

	ts, _ := acceptedTransferSyntaxResolver(acc)(pcID)
	info.PresentationID = pcID
	info.TransferSyntax = ts
	info.MessageID = cmd.MessageID
	info.SOPClassUID = model

	// Open the SEPARATE outbound association to the destination AE, proposing the Storage contexts so
	// the sub-operation C-STOREs negotiate a Storage presentation context. A failure to associate is
	// 0xA702 "unable to perform sub-operations", not a panic (PRD §9.4).
	destAssoc, err := move.ae.Associate(ctx, destAddr, dest, StorageContexts())
	if err != nil {
		return sendMoveResponse(ctx, acc, cmd, pcID, NewStatus(0xA702, ServiceClassMove), SubOperationCounts{})
	}
	// The destination association is tracked and released when the move ends — never fire-and-forget
	// (PRD §9.4). The release rides a fresh context (NOT the move ctx, which may be cancelled) bounded
	// by the AE's ACSE timeout; a non-positive timeout means unbounded (mirroring AE.acseContext) so a
	// Server configured with WithACSETimeout(0) still writes the A-RELEASE-RQ rather than handing the
	// release an already-expired context.
	defer func() {
		releaseCtx := context.Background()
		if d := move.ae.config().acseTimeout; d > 0 {
			var cancel context.CancelFunc
			releaseCtx, cancel = context.WithTimeout(releaseCtx, d)
			defer cancel()
		}
		_ = destAssoc.Release(releaseCtx)
	}()

	// Derive a cancellable context for the handler so a C-CANCEL-RQ (or Server.Shutdown) stops its
	// iterator.
	handlerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	matches := newFindMatchPump(handlerCtx, h.Move(handlerCtx, query, level, dest, info))
	defer matches.stop()

	// A single inbound watcher reads the inbound association for an interleaved C-CANCEL-RQ for the
	// lifetime of the sub-operation loop. A C-MOVE occupies the inbound association until it
	// terminates (the sub-operation C-STOREs ride the separate destination association), so the only
	// valid inbound message is a C-CANCEL-RQ; the watcher's context is the drain's, so ending the
	// move cancels its blocking read (no dangling goroutine, PRD §9.4).
	watcher := newFindCancelWatcher(handlerCtx, acc.Conn(), m, cmd.MessageID)
	defer watcher.stop()

	var counts SubOperationCounts
	for {
		// Block on the next match from the handler OR an inbound C-CANCEL-RQ. A handler that blocks
		// awaiting more matches must not wedge the SCP: the cancel watch runs in parallel, so a
		// C-CANCEL stops the retrieve promptly (PS3.7 §9.3.2.3).
		select {
		case <-ctx.Done():
			// The parent context ended (Server.Shutdown) while the drain was waiting between events.
			// Neither helper is guaranteed to send in that case, so without this arm the select would
			// block forever (DIMSE-014 cooperative shutdown applied to the C-MOVE drain). Returning
			// promptly lets the deferred teardown stop both helpers and release the destination
			// association.
			return ctx.Err()
		case res := <-watcher.result:
			if res.err != nil {
				return res.err
			}
			// A C-CANCEL-RQ for this move: stop the handler (no further sub-operation is dispatched)
			// and send the terminal Cancel RSP carrying the counts accumulated so far (PS3.4 C.4.2.3).
			cancel()
			matches.stop()
			return sendMoveResponse(ctx, acc, cmd, pcID, StatusMoveCancel, counts)
		case match := <-matches.ch:
			if !match.ok {
				// All matches exhausted: send the terminal C-MOVE-RSP carrying the final counts. The
				// status is faithful — Success only when nothing failed, the all-failed 0xA702 when
				// every sub-operation failed, otherwise the 0xB000 partial-failure Warning (PRD §9.2
				// fail-closed). Stop the cancel watcher's blocking read FIRST so the PDU the SCU sends
				// next (an A-RELEASE-RQ or its next request) is not consumed by the watcher and lost
				// from the main dispatch loop.
				watcher.stop()
				return sendMoveResponse(ctx, acc, cmd, pcID, moveTerminalStatus(counts), counts)
			}
			// A non-Pending yield signals the handler's matches are done. A handler-level FAILURE
			// (e.g. out-of-resources resolving the matches) propagates as the terminal status — the
			// handler could not produce the instances. A handler-level Success/Warning is NOT
			// forwarded verbatim: the handler does not know the sub-operation store outcomes, so the
			// runtime computes the real terminal from the accumulated counts (a partial store failure
			// must surface as 0xB000, never be laundered into the handler's Success — PRD §9.2). Stop
			// the watcher before sending so the SCU's next PDU reaches the main dispatch loop.
			if !match.status.IsPending() {
				watcher.stop()
				if match.status.IsFailure() {
					return sendMoveResponse(ctx, acc, cmd, pcID, match.status, counts)
				}
				return sendMoveResponse(ctx, acc, cmd, pcID, moveTerminalStatus(counts), counts)
			}
			instance := match.identifier
			if instance == nil {
				// A Pending match must carry an instance to store; a nil instance is a handler bug.
				// Count it as a failed sub-operation rather than panicking (fail-closed).
				counts.Failed++
				if perr := sendMoveResponse(ctx, acc, cmd, pcID, StatusMovePending, counts); perr != nil {
					return perr
				}
				continue
			}

			// C-STORE the matched instance to the destination as a sub-operation, with a distinct
			// non-zero Message ID from the destination association's allocator (DIMSE-016),
			// propagating the move originator: the AE Title that INVOKED the C-MOVE (the calling AE
			// of the inbound C-MOVE association) and the original C-MOVE Message ID (PS3.7 §9.1.1;
			// the originator is the request SCU, not this Move SCP, so destinations attribute the
			// retrieve to the right AE).
			subStatus, storeErr := destAssoc.Store(ctx, instance,
				WithStoreMessageID(destAssoc.nextMessageID()),
				WithMoveOriginator(info.CallingAETitle, cmd.MessageID),
			)
			switch {
			case storeErr != nil:
				// The sub-operation C-STORE faulted on the wire (the destination association broke);
				// count it as a failed sub-operation and continue reporting progress.
				counts.Failed++
			case subStatus.IsWarning():
				counts.Warning++
			case subStatus.IsSuccess():
				counts.Completed++
			default:
				counts.Failed++
			}

			if perr := sendMoveResponse(ctx, acc, cmd, pcID, StatusMovePending, counts); perr != nil {
				return perr
			}
		}
	}
}

// moveTerminalStatus resolves the terminal C-MOVE-RSP status from the accumulated sub-operation
// counts (parity with pynetdicom's QueryRetrieveMoveServiceClass): a clean Success only when every
// sub-operation completed without failure or warning, the 0xA702 "unable to perform sub-operations"
// Failure when every attempted sub-operation failed, and the 0xB000 "Sub-operations Complete — One
// or More Failures" Warning when at least one sub-operation failed OR warned but not all failed
// (PS3.4 C.4.2.1.5 — a warning is not laundered into Success, PRD §9.2).
func moveTerminalStatus(counts SubOperationCounts) Status {
	if counts.Failed == 0 && counts.Warning == 0 {
		return StatusMoveSuccess
	}
	attempted := counts.Completed + counts.Failed + counts.Warning
	if counts.Failed == attempted {
		return NewStatus(0xA702, ServiceClassMove)
	}
	return StatusMoveSubOpsCompleteWithFailures
}

// sendMoveResponse writes one C-MOVE-RSP carrying the status and the running Completed/Failed/Warning
// sub-operation counts (a C-MOVE-RSP never carries a dataset), echoing the request's Affected SOP
// Class UID and Message ID into the Message ID Being Responded To. Both Pending and terminal responses
// carry the tallies so the SCU can track progress.
//
// NumberOfRemainingSubOperations (0000,1020) is OMITTED: the handler streams matches without a
// pre-count, so the SCP does not know the outstanding total, and PS3.4 C.4.2.1.5 makes that element
// conditional (present only when known, absent on the terminal). Omitting it reports honest
// progress rather than advertising a misleading Remaining of zero (the Codex review finding).
func sendMoveResponse(ctx context.Context, acc *acse.Acceptor, cmd CommandSet, pcID uint8, status Status, counts SubOperationCounts) error {
	rsp := CommandSet{
		CommandField:              CommandCMoveRSP,
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

// refuseUnsupportedMove answers a C-MOVE-RQ that reached a handler with no Move capability: it writes
// a single terminal C-MOVE-RSP carrying StatusSOPClassNotSupported (0x0122) and no dataset, so the
// peer learns the retrieve service is unsupported rather than the SCP panicking or aborting
// (interface segregation, PRD §8.2). The query identifier the RQ carried has already been read from
// the wire by the dispatch loop and is discarded.
func refuseUnsupportedMove(ctx context.Context, acc *acse.Acceptor, cmd CommandSet, pcID uint8) error {
	return sendMoveResponse(ctx, acc, cmd, pcID, StatusSOPClassNotSupported, SubOperationCounts{})
}
