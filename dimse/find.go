package dimse

import (
	"context"
	"errors"
	"iter"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// QueryLevel is the Query/Retrieve Level (0008,0052) for C-FIND/C-GET/C-MOVE (PS3.4 C.3). It
// selects the level of the information model a query addresses and is always written into the
// identifier before sending (the DIMSE-015 fix); the Modality Worklist model is the one exception,
// which carries no level.
type QueryLevel uint8

const (
	// QueryLevelPatient queries at the Patient level (the Patient Root model root).
	QueryLevelPatient QueryLevel = iota
	// QueryLevelStudy queries at the Study level.
	QueryLevelStudy
	// QueryLevelSeries queries at the Series level.
	QueryLevelSeries
	// QueryLevelImage queries at the Image (SOP Instance) level.
	QueryLevelImage
)

// queryLevelKeywords maps each level to the DICOM keyword written into Query/Retrieve Level
// (0008,0052), VR CS (PS3.4 C.3.1): "PATIENT", "STUDY", "SERIES", "IMAGE".
var queryLevelKeywords = map[QueryLevel]string{
	QueryLevelPatient: "PATIENT",
	QueryLevelStudy:   "STUDY",
	QueryLevelSeries:  "SERIES",
	QueryLevelImage:   "IMAGE",
}

// String renders the DICOM keyword written into Query/Retrieve Level (0008,0052).
func (l QueryLevel) String() string {
	if kw, ok := queryLevelKeywords[l]; ok {
		return kw
	}
	return "UNKNOWN"
}

// ParseQueryLevel maps a Query/Retrieve Level keyword back to a QueryLevel. An unrecognised keyword
// is a *ValidationError naming the constraint, never a silent default.
func ParseQueryLevel(s string) (QueryLevel, error) {
	for level, kw := range queryLevelKeywords {
		if kw == s {
			return level, nil
		}
	}
	return 0, &ValidationError{Detail: "unknown Query/Retrieve Level keyword " + s}
}

// queryConfig holds the resolved QueryOptions for a C-FIND/C-GET/C-MOVE.
type queryConfig struct {
	priority Priority
	// model overrides the default Q/R Information Model SOP Class chosen from the operation. Empty
	// means "use the operation's default" (Study Root FIND for Find).
	model dicom.SOPClassUID
}

// QueryOption configures a C-FIND/C-GET/C-MOVE query/retrieve.
type QueryOption func(*queryConfig)

// WithQueryPriority sets the query/retrieve operation priority (default medium).
func WithQueryPriority(p Priority) QueryOption {
	return func(c *queryConfig) { c.priority = p }
}

// WithQueryModel overrides the Query/Retrieve Information Model SOP Class the query negotiates a
// presentation context for (default chosen from the operation — Study Root FIND for Find). Pass it
// to select the Patient Root model, or the Modality Worklist model for an MWL query.
func WithQueryModel(sopClass dicom.SOPClassUID) QueryOption {
	return func(c *queryConfig) { c.model = sopClass }
}

// Find issues a C-FIND and yields (Status, identifier) for each response (PS3.4 C.4.1). The SCU
// sends a C-FIND-RQ carrying the query identifier, then yields each Pending (0xFF00/0xFF01) match
// with its identifier dataset; the terminal Success/Warning/Failure/Cancel status yields a nil
// dataset and ends iteration. The requested QueryLevel is written into Query/Retrieve Level
// (0008,0052) of a copy of the identifier before sending — the level the prototype silently dropped
// (Codex DIMSE-015).
//
// Breaking out of the range loop, or cancelling ctx, sends a C-CANCEL-RQ for the operation's
// Message ID on the same presentation context. A transport or protocol fault that ends iteration
// before a clean terminal status is exposed via Association.LastError(), read immediately after the
// loop; a terminal Failure/Cancel Status is in-band data, not an error. A pre-flight fault (an
// unestablished or released association, no negotiated Q/R context) yields a single terminal
// Failure status and sets LastError, never panicking (Codex DIMSE-017).
//
// An Association is not safe for concurrent queries: LastError() is per-association. Run one
// Find/Get/Move iterator per association at a time.
func (a *Association) Find(
	ctx context.Context,
	query *dicom.DataSet,
	level QueryLevel,
	opts ...QueryOption,
) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {
		cfg := queryConfig{priority: PriorityMedium}
		for _, opt := range opts {
			opt(&cfg)
		}

		model := cfg.model
		if model == "" {
			model = defaultFindModel(level)
		}

		svc := serviceClassForQueryModel(model)

		// Pre-flight is checked BEFORE touching any per-association state, so a nil/unestablished
		// association yields a typed terminal Failure rather than panicking dereferencing a.mu
		// (Codex DIMSE-017). It validates the model is a C-FIND information model, that an accepted
		// context for it was negotiated, and — for a levelled model — that the level is one of the
		// four defined constants. Any failure is a fail-closed fault: the iterator surfaces one
		// terminal Failure plus LastError and transmits nothing (PRD §9.2; Open question 1).
		if err := a.findPreflight(model, level); err != nil {
			a.setLastError(err)
			yield(NewStatus(0xC000, svc), nil)
			return
		}
		a.clearLastError()

		pcID, ts, _ := a.contextForQuery(model)

		// Write the Query/Retrieve Level into a COPY of the identifier so the caller's dataset is
		// untouched; a nil query becomes an empty identifier carrying only the level. The Modality
		// Worklist model carries no level element (suppressed for the worklist model).
		identifier := dicom.NewDataSet()
		if query != nil {
			identifier = query.Clone()
		}
		if model != modalityWorklistSOPClass {
			identifier.SetString(dicom.TagQueryRetrieveLevel, level.String())
		}

		conn := a.requestor.Conn()
		m := a.requestor.Machine()

		opCtx, cancel := a.dimseContext(ctx)
		defer cancel()

		msgID := a.nextMessageID()
		rq := CommandSet{
			CommandField:        CommandCFindRQ,
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

		// Read responses until the terminal status. A Pending yields its matching identifier; the
		// terminal yields a nil dataset and ends iteration. An early break, or a ctx cancellation
		// interrupting the read, sends a C-CANCEL-RQ for this operation's Message ID and then drains
		// the outstanding responses through to the terminal status, so the association is left clean
		// for reuse (the SCP keeps sending queued Pendings plus a terminal after the cancel — they
		// must not corrupt the next operation).
		for {
			rsp, ds, _, err := receiveMessage(opCtx, conn, m, newMessageReassembler(ts))
			if err != nil {
				// Distinguish caller cancellation from the internal DIMSE timeout. A cancelled or
				// expired PARENT ctx is the cancellation contract: send a C-CANCEL and drain, then
				// end promptly (PRD §9.4). A deadline that fired only on opCtx (the association's
				// DIMSE timeout, with the parent ctx still live) is a stalled-peer FAULT surfaced
				// via LastError, never a silent end — otherwise a stalled C-FIND would finish with
				// neither a terminal status nor an error.
				if (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) && ctx.Err() != nil {
					a.cancelOperation(ctx, conn, m, pcID, ts, msgID, svc)
					return
				}
				a.setLastError(err)
				yield(NewStatus(0xC000, svc), nil)
				return
			}

			if rsp.CommandField != CommandCFindRSP {
				a.setLastError(&ProtocolError{
					State:  m.CurrentState(),
					Detail: "expected a C-FIND-RSP command field in the response",
				})
				yield(NewStatus(0xC000, svc), nil)
				return
			}
			if !rsp.HasStatus {
				a.setLastError(&ProtocolError{
					State:  m.CurrentState(),
					Detail: "C-FIND-RSP missing the mandatory Status element",
				})
				yield(NewStatus(0xC000, svc), nil)
				return
			}

			status := NewStatus(rsp.Status, svc)
			if status.IsPending() {
				if !yield(status, ds) {
					// The caller broke out: cancel the in-flight operation and drain to terminal.
					a.cancelOperation(ctx, conn, m, pcID, ts, msgID, svc)
					return
				}
				continue
			}
			// Terminal status (Success/Warning/Cancel/Failure): yield it last with no dataset.
			yield(status, nil)
			return
		}
	}
}

// defaultFindModel chooses the C-FIND Information Model SOP Class for a level when the caller did
// not override it with WithQueryModel. A Patient-level query is only valid in the Patient Root
// model (Study Root has no Patient level), so QueryLevelPatient defaults to Patient Root FIND;
// every other level defaults to Study Root FIND, the common radiology entry point (PS3.4 C.6).
func defaultFindModel(level QueryLevel) dicom.SOPClassUID {
	if level == QueryLevelPatient {
		return patientRootFindSOPClass
	}
	return studyRootFindSOPClass
}

// serviceClassForQueryModel selects the status-categorisation service class for a C-FIND model:
// the Modality Worklist model categorises against the worklist table (FIND-shaped but with the
// worklist-specific meanings), every other Q/R FIND model against the FIND table.
func serviceClassForQueryModel(model dicom.SOPClassUID) ServiceClass {
	if model == modalityWorklistSOPClass {
		return ServiceClassWorklist
	}
	return ServiceClassFind
}

// findModels is the set of C-FIND information-model SOP Classes Find may negotiate: the Patient
// Root and Study Root FIND models and the Modality Worklist FIND model. A WithQueryModel naming a
// MOVE/GET (or any non-FIND) class is rejected so Find never sends a C-FIND-RQ whose Affected SOP
// Class is not a valid query information model (which a compliant peer would reject or abort).
var findModels = map[dicom.SOPClassUID]struct{}{
	patientRootFindSOPClass:  {},
	studyRootFindSOPClass:    {},
	modalityWorklistSOPClass: {},
}

// findPreflight validates the association can run a C-FIND for the given model and level: the model
// is a C-FIND information model, the association is established and not released, an accepted
// presentation context for the model exists, and (for a levelled model) the level is one of the
// four defined constants. It returns a typed error on any failure so Find can fail closed before
// any wire I/O (PRD §9.2).
func (a *Association) findPreflight(model dicom.SOPClassUID, level QueryLevel) error {
	if _, ok := findModels[model]; !ok {
		return &ValidationError{Detail: "query model " + string(model) + " is not a C-FIND information model"}
	}
	if model != modalityWorklistSOPClass {
		if _, ok := queryLevelKeywords[level]; !ok {
			return &ValidationError{Detail: "unknown Query/Retrieve Level for a C-FIND"}
		}
	}
	if a == nil || a.requestor == nil {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Find on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Find on a released association"}
	}
	if _, _, ok := a.contextForQuery(model); !ok {
		return &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for Query/Retrieve model " + string(model),
		}
	}
	return nil
}

// cancelOperation sends a C-CANCEL-RQ for the in-flight operation msgID on the same presentation
// context, then drains the outstanding C-FIND responses to their terminal status, used when a
// query/retrieve caller stops iterating early (breaks the range loop) or cancels the operation ctx
// (PS3.7 §9.3.2.3). The cancel and drain ride a fresh context bounded by the DIMSE timeout so a
// cancelled or expired operation ctx does not also block them; both are best-effort — failures are
// not propagated, since the operation is ending. The drain leaves the association clean for reuse:
// the SCP keeps sending already-queued Pending responses and a terminal RSP after the cancel, and
// those must be consumed so they do not corrupt the next operation on the association.
func (a *Association) cancelOperation(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, pcID uint8, ts dicom.TransferSyntax, msgID uint16, svc ServiceClass) {
	cancelCtx, cancel := a.cancelContext(ctx)
	defer cancel()
	cmd := CommandSet{
		CommandField:              CommandCCancelRQ,
		MessageIDBeingRespondedTo: msgID,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	if err := sendCommand(cancelCtx, conn, m, pcID, cmd); err != nil {
		// The cancel could not be sent (already-broken conversation); record the fault so a caller
		// inspecting LastError after the loop sees it, and stop — there is nothing to drain.
		a.setLastError(err)
		return
	}
	a.drainToTerminal(cancelCtx, conn, m, ts, svc)
}

// drainToTerminal reads C-FIND responses until a terminal (non-pending) status, discarding the
// matching identifiers, so a cancelled query leaves the association stream clean for the next
// operation. It is best-effort and bounded by ctx; a read fault records LastError and stops.
func (a *Association) drainToTerminal(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, ts dicom.TransferSyntax, svc ServiceClass) {
	for {
		rsp, _, _, err := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
		if err != nil {
			a.setLastError(err)
			return
		}
		if rsp.CommandField != CommandCFindRSP || !rsp.HasStatus {
			// A response that is not a well-formed C-FIND-RSP ends the drain; the stream is in an
			// unexpected state, recorded for LastError.
			a.setLastError(&ProtocolError{
				State:  m.CurrentState(),
				Detail: "draining cancelled C-FIND: unexpected response while awaiting the terminal status",
			})
			return
		}
		if !NewStatus(rsp.Status, svc).IsPending() {
			return // terminal status reached: the stream is drained
		}
	}
}

// cancelContext derives a short, fresh context for a best-effort C-CANCEL send. It does NOT inherit
// the operation's (possibly already-cancelled) ctx, so the cancel can still be written when the
// caller cancelled ctx to trigger it; it is bounded by the DIMSE timeout, defaulting to a short
// fixed bound when none is configured.
func (a *Association) cancelContext(context.Context) (context.Context, context.CancelFunc) {
	d := a.dimseTimeout
	if d <= 0 {
		d = 5 * time.Second
	}
	return context.WithTimeout(context.Background(), d)
}

// Cancel sends a C-CANCEL-RQ for an in-flight operation identified by its Message ID, on the first
// accepted presentation context. It is a low-level escape hatch; everyday callers express
// cancellation by cancelling the ctx passed to Find/Get/Move or by breaking the range loop, which
// send the cancel automatically. It returns a typed error on an unestablished/released association,
// never a panic.
func (a *Association) Cancel(ctx context.Context, msgID uint16) error {
	if a == nil || a.requestor == nil {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Cancel on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Cancel on a released association"}
	}
	pcID, ok := a.firstAcceptedContext()
	if !ok {
		return &AssociationError{Kind: AssociationNotEstablished, Detail: "Cancel: no accepted presentation context"}
	}
	cancelCtx, cancel := a.cancelContext(ctx)
	defer cancel()
	cmd := CommandSet{
		CommandField:              CommandCCancelRQ,
		MessageIDBeingRespondedTo: msgID,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	return sendCommand(cancelCtx, a.requestor.Conn(), a.requestor.Machine(), pcID, cmd)
}

// firstAcceptedContext returns the presentation context ID of the first accepted context, or false
// when none was accepted.
func (a *Association) firstAcceptedContext() (uint8, bool) {
	for _, pc := range a.accepted {
		if pc.Result == ContextAccepted {
			return pc.ID, true
		}
	}
	return 0, false
}

// contextForQuery returns the accepted presentation context ID and its negotiated transfer syntax
// for the Query/Retrieve Information Model SOP Class, or false when none was accepted (so Find
// fails closed rather than guessing a context). It mirrors contextForStorage: selection is by
// abstract syntax, and the first accepted matching context wins.
func (a *Association) contextForQuery(model dicom.SOPClassUID) (uint8, dicom.TransferSyntax, bool) {
	for _, pc := range a.accepted {
		if pc.Result != ContextAccepted {
			continue
		}
		if pc.AbstractSyntax != model {
			continue
		}
		if len(pc.TransferSyntaxes) == 0 {
			continue
		}
		return pc.ID, pc.TransferSyntaxes[0], true
	}
	return 0, "", false
}
