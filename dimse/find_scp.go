package dimse

import (
	"context"
	"iter"
	"sync"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// serveFindMessage services one inbound C-FIND-RQ over an established acceptor association: it
// validates the negotiated context is a Query/Retrieve FIND model (the context guard analogous to
// validateStoreContext/validateEchoContext), reads the query level from the identifier, dispatches to
// the FindHandler, and drains the handler's iter.Seq2 into one Pending C-FIND-RSP per yielded match
// (command + matching identifier) followed by a terminal C-FIND-RSP carrying the iterator's final
// status and no dataset (PS3.4 C.4.1, parity with pynetdicom's QueryRetrieveFindServiceClass). A
// handler that yields no match still terminates with a single Success RSP — the SCP always writes a
// terminal, so a zero-match query never hangs the SCU.
//
// The dispatcher honours a C-CANCEL-RQ arriving mid-drain by cancelling the handler's context (so its
// iterator stops) and sending a 0xFE00 Cancel terminal RSP (PS3.7 §9.3.2.3). The handler's context is
// derived from ctx, so Server.Shutdown also stops an in-flight query.
func serveFindMessage(ctx context.Context, acc *acse.Acceptor, h FindHandler, cmd CommandSet, ds *dicom.DataSet, pcID uint8, info OpInfo) error {
	m := acc.Machine()
	if err := validateFindContext(cmd, pcID, acceptedAbstractSyntaxResolver(acc), m.CurrentState()); err != nil {
		return err
	}

	model := dicom.SOPClassUID(cmd.AffectedSOPClassUID)
	svc := serviceClassForQueryModel(model)

	// A C-FIND-RQ ALWAYS carries an identifier (CommandDataSetType present); a request that declares
	// none is malformed. Fail closed with a terminal Failure rather than fabricating an empty
	// identifier, which many handlers would treat as a match-everything wildcard and answer with a
	// broad result (PRD §9.2 fail-closed). 0xA900 "Identifier Does Not Match SOP Class" is the FIND
	// failure for an identifier that cannot be used for matching.
	if ds == nil {
		return sendFindResponse(ctx, acc, cmd, pcID, NewStatus(0xA900, svc), nil)
	}
	query := ds

	// The query identifier carries the Query/Retrieve Level (0008,0052) the SCU wrote (DIMSE-015).
	// For a levelled (Patient/Study Root) model a missing or unparseable level is a malformed query:
	// fail closed rather than silently defaulting to Study, which would run the wrong scope. The
	// Modality Worklist model carries no level element (it has no Q/R level), so that check is skipped.
	level, levelErr := queryLevelFromIdentifier(query, model)
	if levelErr != nil {
		return sendFindResponse(ctx, acc, cmd, pcID, NewStatus(0xA900, svc), nil)
	}

	ts, _ := acceptedTransferSyntaxResolver(acc)(pcID)
	info.PresentationID = pcID
	info.TransferSyntax = ts
	info.MessageID = cmd.MessageID
	info.SOPClassUID = model

	// Derive a cancellable context for the handler so a C-CANCEL-RQ (or Shutdown) stops its iterator.
	handlerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	matches := newFindMatchPump(handlerCtx, h.Find(handlerCtx, query, level, info))
	defer matches.stop()

	// A single inbound watcher reads the association for an interleaved C-CANCEL-RQ for the lifetime
	// of the drain. A C-FIND occupies the association until it terminates, so the only valid inbound
	// message is a C-CANCEL-RQ; the watcher's context is the drain's, so ending the query cancels its
	// blocking read (no dangling goroutine, PRD §9.4).
	watcher := newFindCancelWatcher(handlerCtx, acc.Conn(), m, cmd.MessageID, nil)
	defer watcher.stop()

	for {
		// Block on the next match from the handler OR an inbound C-CANCEL-RQ. A handler that blocks
		// awaiting more matches must not wedge the SCP: the cancel watch runs in parallel, so a
		// C-CANCEL stops the handler and ends the query promptly (PS3.7 §9.3.2.3).
		select {
		case <-ctx.Done():
			// The parent context ended (Server.Shutdown) while the drain was waiting between events.
			// Neither helper is guaranteed to send in that case — the cancel watcher suppresses its
			// context error and the match pump may take its own ctx.Done() branch — so without this arm
			// the select would block forever and the association goroutine would outlive Shutdown's
			// deadline. Returning promptly here lets the deferred teardown stop both helpers (DIMSE-014
			// cooperative shutdown applied to the C-FIND drain). The peer sees the association close.
			return ctx.Err()
		case res := <-watcher.result:
			if res.err != nil {
				return res.err
			}
			// A C-CANCEL-RQ for this query: stop the handler and send the terminal Cancel RSP.
			cancel()
			matches.stop()
			return sendFindResponse(ctx, acc, cmd, pcID, NewStatus(StatusCancel.Code, svc), nil)
		case match := <-matches.ch:
			if !match.ok {
				// The handler exhausted its matches without a terminal of its own: send the terminal
				// Success RSP, the no-hang contract (zero matches still terminates). Stop the cancel
				// watcher's blocking read FIRST so the PDU the SCU sends next (an A-RELEASE-RQ or its
				// next request) is not consumed by the watcher and lost from the main dispatch loop.
				watcher.stop()
				return sendFindResponse(ctx, acc, cmd, pcID, NewStatus(0x0000, svc), nil)
			}
			if !match.status.IsPending() {
				// The handler yielded a terminal status (Success/Warning/Failure); send it with no
				// dataset and stop. A terminal mid-iteration ends the query (PS3.4 C.4.1.3). Stop the
				// watcher before sending so the SCU's next PDU reaches the main dispatch loop.
				watcher.stop()
				return sendFindResponse(ctx, acc, cmd, pcID, match.status, nil)
			}
			// A Pending match: send it with its identifier dataset.
			if err := sendFindResponse(ctx, acc, cmd, pcID, match.status, match.identifier); err != nil {
				return err
			}
		}
	}
}

// findCancelResult is the outcome of the inbound cancel watch: a C-CANCEL-RQ for the in-flight query
// arrived (err nil), or the inbound read faulted/saw an unexpected message (err non-nil). A read that
// ends only because the watcher's context was cancelled (the query terminated normally) is never
// delivered — the watcher exits silently.
type findCancelResult struct {
	err error
}

// findCancelWatcher reads the association in a tracked goroutine for a single inbound C-CANCEL-RQ for
// the in-flight query, delivering the outcome once over result. A C-FIND occupies the association
// until it terminates, so the only valid inbound message is a C-CANCEL-RQ; any other inbound message
// is a protocol fault. The watcher's context is the drain's, so terminating the query (cancel,
// terminal, or fault) cancels the blocking read and the goroutine exits without delivering — no
// dangling goroutine outlives the operation (PRD §9.4).
type findCancelWatcher struct {
	result   chan findCancelResult
	cancel   context.CancelFunc
	stopOnce sync.Once
}

// newFindCancelWatcher starts the cancel-watch goroutine bound by a context derived from parent.
// A non-nil onCancel is invoked BEFORE the cancel result is delivered, so a dispatcher blocked in a
// long sub-operation (the C-MOVE destination store) can hang that work off a context onCancel
// cancels and observe the C-CANCEL promptly, rather than only at its next select (PS3.4 C.4.2.2.3
// "as soon as possible"). The C-FIND drain passes nil: its select is never blocked outside the
// watcher.
func newFindCancelWatcher(parent context.Context, conn *dul.Conn, m *dul.StateMachine, msgID uint16, onCancel func()) *findCancelWatcher {
	ctx, cancel := context.WithCancel(parent) // #nosec G118 -- cancel is stored on the watcher and invoked via stopOnce in stop()
	w := &findCancelWatcher{
		result: make(chan findCancelResult, 1),
		cancel: cancel,
	}
	go func() {
		cmd, _, _, err := receiveMessage(ctx, conn, m, newMessageReassembler(dicom.ImplicitVRLittleEndian))
		if err != nil {
			// A context cancellation is the normal end-of-query teardown, not a fault: exit silently.
			if ctx.Err() != nil {
				return
			}
			w.result <- findCancelResult{err: err}
			return
		}
		if cmd.CommandField == CommandCCancelRQ && cmd.MessageIDBeingRespondedTo == msgID {
			if onCancel != nil {
				onCancel()
			}
			w.result <- findCancelResult{}
			return
		}
		// Any other inbound message mid-query is unexpected on this association (a single C-FIND
		// occupies it until terminated); surface it as a protocol fault rather than silently dropping.
		w.result <- findCancelResult{err: &ProtocolError{
			State:  m.CurrentState(),
			Detail: "unexpected inbound command while draining a C-FIND query; expected only a C-CANCEL-RQ",
		}}
	}()
	return w
}

// stop cancels the watcher's blocking read and is idempotent. Calling it on query teardown unblocks
// the goroutine so it exits promptly.
func (w *findCancelWatcher) stop() { w.stopOnce.Do(w.cancel) }

// findMatch is one yield pulled from a FindHandler's iterator: a status, its identifier (Pending
// only), and ok=false once the iterator is exhausted.
type findMatch struct {
	status     Status
	identifier *dicom.DataSet
	ok         bool
}

// findMatchPump drives a FindHandler's iter.Seq2 in a tracked goroutine, delivering each yield over a
// channel so the dispatch loop can select between the next match and an inbound C-CANCEL-RQ without a
// blocking handler wedging the SCP. The goroutine ends when the iterator is exhausted, when the
// handler's context is cancelled, or when stop is called (idempotent), so it is never fire-and-forget
// (PRD §9.4).
type findMatchPump struct {
	ch       chan findMatch
	quit     chan struct{}
	stopOnce sync.Once
}

// newFindMatchPump starts the pump goroutine draining seq under ctx.
func newFindMatchPump(ctx context.Context, seq iter.Seq2[Status, *dicom.DataSet]) *findMatchPump {
	p := &findMatchPump{
		ch:   make(chan findMatch),
		quit: make(chan struct{}),
	}
	go func() {
		for status, identifier := range seq {
			select {
			case p.ch <- findMatch{status: status, identifier: identifier, ok: true}:
			case <-ctx.Done():
				return
			case <-p.quit:
				return
			}
		}
		// Iterator exhausted: signal end-of-matches once, then exit.
		select {
		case p.ch <- findMatch{ok: false}:
		case <-ctx.Done():
		case <-p.quit:
		}
	}()
	return p
}

// stop signals the pump goroutine to exit (idempotent). The goroutine selects on quit on every send,
// so a handler blocked between yields — once its context is also cancelled — observes the signal and
// returns; stop covers the case where the dispatch loop ends the query (cancel, terminal, or fault)
// before the handler is done.
func (p *findMatchPump) stop() { p.stopOnce.Do(func() { close(p.quit) }) }

// sendFindResponse writes one C-FIND-RSP: a Pending response carries the matching identifier dataset
// (CommandDataSetType present), a terminal response carries none (not present), echoing the request's
// Affected SOP Class UID and Message ID into the Message ID Being Responded To and reporting status.
func sendFindResponse(ctx context.Context, acc *acse.Acceptor, cmd CommandSet, pcID uint8, status Status, identifier *dicom.DataSet) error {
	rsp := CommandSet{
		CommandField:              CommandCFindRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		HasStatus:                 true,
		Status:                    status.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	if identifier != nil {
		rsp.CommandDataSetType = CommandDataSetPresent
	}
	ts, _ := acceptedTransferSyntaxResolver(acc)(pcID)
	// Resolve the send cap the same way the SCU side does (Codex DIMSE-005): an unlimited peer max
	// (0) must NOT pass through to fragmentMessage, which would place a whole match identifier in one
	// P-DATA-TF and risk exceeding the peer/library PDU ceiling on a large identifier. SendCap floors
	// an unlimited peer at the local default cap so the identifier still fragments.
	sendCap := MaxPDULength(acc.PeerMaxPDULength()).SendCap(defaultMaxPDULength)
	return sendMessage(ctx, acc.Conn(), acc.Machine(), pcID, rsp, identifier, ts, sendCap)
}

// refuseUnsupportedFind answers a C-FIND-RQ that reached a handler with no FindHandler capability: it
// writes a single terminal C-FIND-RSP carrying StatusSOPClassNotSupported (0x0122) and no dataset, so
// the peer learns the query service is unsupported rather than the SCP panicking or aborting
// (interface segregation, PRD §8.2). The query identifier the RQ carried has already been read from
// the wire by the dispatch loop and is discarded.
func refuseUnsupportedFind(ctx context.Context, acc *acse.Acceptor, cmd CommandSet, pcID uint8) error {
	return sendFindResponse(ctx, acc, cmd, pcID, StatusSOPClassNotSupported, nil)
}

// queryLevelFromIdentifier reads the Query/Retrieve Level (0008,0052) the SCU wrote into the
// identifier and maps it to a QueryLevel. The Modality Worklist model has no level element, so it is
// reported as QueryLevelStudy (the level is meaningless for worklist and the handler ignores it). For
// a levelled (Patient/Study Root) model a missing element or an unrecognised keyword is a malformed
// query: it returns a *ValidationError so the caller fails closed rather than guessing a level and
// running the wrong scope of matches (PRD §9.2 fail-closed).
func queryLevelFromIdentifier(identifier *dicom.DataSet, model dicom.SOPClassUID) (QueryLevel, error) {
	if model == modalityWorklistSOPClass {
		return QueryLevelStudy, nil
	}
	kw, ok := identifier.GetString(dicom.TagQueryRetrieveLevel)
	if !ok {
		return 0, &ValidationError{Detail: "C-FIND identifier has no Query/Retrieve Level (0008,0052) for a levelled model"}
	}
	return ParseQueryLevel(kw)
}
