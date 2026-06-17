package dimse

import (
	"context"
	"errors"
	"iter"

	"github.com/codeninja55/go-radx/dicom"
)

// Move issues a C-MOVE and yields each response Status (PS3.4 C.4.2). The SCU sends a C-MOVE-RQ
// carrying the query identifier and the Move Destination AE Title (0000,0600); the peer (the C-MOVE
// SCP) resolves that destination, opens its own association to it, and C-STOREs each matched instance
// there as a sub-operation, reporting progress in C-MOVE-RSP responses. Each Pending (0xFF00) yields
// with a nil dataset (a C-MOVE-RSP carries no dataset) and refreshes the four sub-operation counts
// readable via Association.SubOperationCounts(); the terminal Success/Warning/Failure/Cancel status
// yields last and ends iteration, also refreshing the final counts.
//
// The requested QueryLevel is written into Query/Retrieve Level (0008,0052) of a copy of the
// identifier before sending (DIMSE-015). The 0xB000 "sub-operations complete, one or more failures"
// Warning and the 0xA801 "Move Destination Unknown" Failure are reported faithfully as terminal
// statuses, never laundered into Success (PRD §9.2).
//
// Breaking out of the range loop, or cancelling ctx, sends a C-CANCEL-RQ for the operation's Message
// ID on the same presentation context. A transport or protocol fault that ends iteration before a
// clean terminal status is exposed via Association.LastError(), read immediately after the loop; a
// terminal Failure/Cancel Status is in-band data, not an error. A pre-flight fault (an unestablished
// or released association, no negotiated MOVE context, an empty destination, a non-MOVE model) yields
// a single terminal Failure status and sets LastError, never panicking (DIMSE-017).
//
// An Association is not safe for concurrent queries: LastError() and SubOperationCounts() are
// per-association. Run one Find/Get/Move iterator per association at a time.
func (a *Association) Move(
	ctx context.Context,
	query *dicom.DataSet,
	level QueryLevel,
	dest AETitle,
	opts ...QueryOption,
) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {
		cfg := queryConfig{priority: PriorityMedium}
		for _, opt := range opts {
			opt(&cfg)
		}

		model := cfg.model
		if model == "" {
			model = defaultMoveModel(level)
		}

		// The MOVE service always categorises against the MOVE table (the 0xB000 sub-op-failure
		// Warning and 0xA801 destination-unknown meanings are MOVE-specific).
		const svc = ServiceClassMove

		// Pre-flight before touching any per-association state, so a nil/unestablished association
		// yields a typed terminal Failure rather than panicking (DIMSE-017). It validates the model is
		// a C-MOVE information model, the destination is non-empty, an accepted context for the model
		// was negotiated, and the level is one of the four defined constants. Any failure is
		// fail-closed: one terminal Failure plus LastError, transmitting nothing (PRD §9.2).
		if err := a.movePreflight(model, level, dest); err != nil {
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
			CommandField:        CommandCMoveRQ,
			MessageID:           msgID,
			AffectedSOPClassUID: dicom.UID(model),
			HasPriority:         true,
			Priority:            cfg.priority,
			MoveDestination:     dest,
			CommandDataSetType:  CommandDataSetPresent,
		}
		if err := sendMessage(opCtx, conn, m, pcID, rq, identifier, ts, a.sendCap()); err != nil {
			a.setLastError(err)
			yield(NewStatus(0xC000, svc), nil)
			return
		}

		// Read C-MOVE-RSP responses until the terminal status. A Pending refreshes the sub-operation
		// counts and yields with a nil dataset; the terminal yields last and ends iteration. An early
		// break, or a ctx cancellation interrupting the read, sends a C-CANCEL-RQ for this operation's
		// Message ID and drains the outstanding responses to the terminal, leaving the association
		// clean for reuse.
		for {
			rsp, rspDS, _, err := receiveMessage(opCtx, conn, m, newMessageReassembler(ts))
			if err != nil {
				// A cancelled or expired PARENT ctx is the cancellation contract: send a C-CANCEL and
				// drain. A deadline that fired only on opCtx (the DIMSE timeout, parent still live) is a
				// stalled-peer FAULT surfaced via LastError, never a silent end.
				if (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) && ctx.Err() != nil {
					a.cancelOperation(ctx, conn, m, pcID, ts, msgID, svc, CommandCMoveRSP)
					return
				}
				a.setLastError(err)
				yield(NewStatus(0xC000, svc), nil)
				return
			}

			if rsp.CommandField != CommandCMoveRSP {
				a.setLastError(&ProtocolError{
					State:  m.CurrentState(),
					Detail: "expected a C-MOVE-RSP command field in the response",
				})
				yield(NewStatus(0xC000, svc), nil)
				return
			}
			if !rsp.HasStatus {
				a.setLastError(&ProtocolError{
					State:  m.CurrentState(),
					Detail: "C-MOVE-RSP missing the mandatory Status element",
				})
				yield(NewStatus(0xC000, svc), nil)
				return
			}

			// Refresh the sub-operation counts a caller reads via SubOperationCounts(). A response that
			// omits the counts (some peers send a bare terminal) leaves the previous tallies in place.
			// RemainingKnown records whether the conditional NumberOfRemainingSubOperations element was
			// actually present, so a caller distinguishes an omitted/unknown Remaining from a genuine 0.
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
				// A Pending C-MOVE-RSP carries no dataset (only the sub-operation counts); yield nil.
				if !yield(status, nil) {
					a.cancelOperation(ctx, conn, m, pcID, ts, msgID, svc, CommandCMoveRSP)
					return
				}
				continue
			}
			// Terminal status (Success/Warning/Cancel/Failure): yield it last, carrying any identifier
			// the peer attached. A terminal Warning/Failure C-MOVE-RSP MAY carry a Failed SOP Instance
			// UID List (0008,0058) identifier dataset (PS3.4 C.4.2.1.6); surface it through the
			// iterator's dataset slot so a caller can inspect which instances failed, rather than
			// discarding it (rspDS is nil when the peer attached none, e.g. on a clean Success).
			yield(status, rspDS)
			return
		}
	}
}

// defaultMoveModel chooses the C-MOVE Information Model SOP Class for a level when the caller did not
// override it with WithQueryModel. A Patient-level retrieve is only valid in the Patient Root model;
// every other level defaults to Study Root MOVE, the common radiology entry point (PS3.4 C.6).
func defaultMoveModel(level QueryLevel) dicom.SOPClassUID {
	if level == QueryLevelPatient {
		return patientRootMoveSOPClass
	}
	return studyRootMoveSOPClass
}

// moveModels is the set of C-MOVE information-model SOP Classes Move may negotiate: the Patient Root
// and Study Root MOVE models. A WithQueryModel naming a FIND/GET (or any non-MOVE) class is rejected
// so Move never sends a C-MOVE-RQ whose Affected SOP Class is not a valid retrieve information model.
var moveModels = map[dicom.SOPClassUID]struct{}{
	patientRootMoveSOPClass:           {},
	studyRootMoveSOPClass:             {},
	patientStudyOnlyMoveSOPClass:      {},
	compositeInstanceRootMoveSOPClass: {},
}

// movePreflight validates the association can run a C-MOVE for the given model, level, and
// destination: the model is a C-MOVE information model, the destination AE Title is non-empty and
// valid, the association is established and not released, an accepted presentation context for the
// model exists, and the level is one of the four defined constants. It returns a typed error on any
// failure so Move can fail closed before any wire I/O (PRD §9.2).
func (a *Association) movePreflight(model dicom.SOPClassUID, level QueryLevel, dest AETitle) error {
	if _, ok := moveModels[model]; !ok {
		return &ValidationError{Detail: "query model " + string(model) + " is not a C-MOVE information model"}
	}
	if dest == "" {
		return &ValidationError{Detail: "C-MOVE requires a non-empty Move Destination AE Title"}
	}
	if _, err := ParseAETitle(string(dest)); err != nil {
		return err
	}
	if _, ok := queryLevelKeywords[level]; !ok {
		return &ValidationError{Detail: "unknown Query/Retrieve Level for a C-MOVE"}
	}
	if err := validateModelLevel(model, level); err != nil {
		return err
	}
	if a == nil || a.requestor == nil {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Move on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Move on a released association"}
	}
	if _, _, ok := a.contextForQuery(model); !ok {
		return &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for Query/Retrieve model " + string(model),
		}
	}
	return nil
}
