package dimse

import (
	"context"
	"errors"
	"iter"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// TestValidateFindContext is the regression for the C-FIND negotiation guard (the symmetry with
// validateStoreContext/validateEchoContext): a C-FIND must arrive on a context whose negotiated
// abstract syntax is a Query/Retrieve FIND information model AND whose Affected SOP Class matches that
// model, else it is a protocol fault — a peer cannot run a query outside the negotiated/declared SOP
// Class.
func TestValidateFindContext(t *testing.T) {
	abstractFor := func(pcID uint8) (dicom.SOPClassUID, bool) {
		switch pcID {
		case 1:
			return studyRootFindSOPClass, true // a FIND context
		case 3:
			return studyRootMoveSOPClass, true // a MOVE context (not FIND)
		default:
			return "", false
		}
	}

	// A C-FIND on the negotiated FIND context with the matching Affected SOP Class passes.
	match := CommandSet{CommandField: CommandCFindRQ, AffectedSOPClassUID: dicom.UID(studyRootFindSOPClass)}
	if err := validateFindContext(match, 1, abstractFor, Sta6); err != nil {
		t.Errorf("matching FIND context rejected: %v", err)
	}

	// A C-FIND on a context whose negotiated abstract syntax is a MOVE model is a protocol fault.
	onMove := CommandSet{CommandField: CommandCFindRQ, AffectedSOPClassUID: dicom.UID(studyRootMoveSOPClass)}
	if err := validateFindContext(onMove, 3, abstractFor, Sta6); err == nil {
		t.Error("C-FIND on a non-FIND (MOVE) context = nil error, want a protocol fault")
	} else {
		if _, ok := errors.AsType[*ProtocolError](err); !ok {
			t.Errorf("error = %T, want *ProtocolError", err)
		}
	}

	// A C-FIND whose Affected SOP Class disagrees with the negotiated FIND context is a protocol fault.
	mismatch := CommandSet{CommandField: CommandCFindRQ, AffectedSOPClassUID: dicom.UID(patientRootFindSOPClass)}
	if err := validateFindContext(mismatch, 1, abstractFor, Sta6); err == nil {
		t.Error("mismatched FIND SOP class on a FIND context = nil error, want a protocol fault")
	}

	// A C-FIND on a context that was never negotiated is a protocol fault.
	if err := validateFindContext(match, 9, abstractFor, Sta6); err == nil {
		t.Error("unknown presentation context = nil error, want a protocol fault")
	}
}

// cannedFindHandler is a FindHandler that yields a fixed list of (status, identifier) results, so a
// drain test can assert the dispatcher turns each Pending yield into one Pending C-FIND-RSP plus a
// terminal RSP. It records the query, level, and OpInfo it was called with and whether its context
// was observed cancelled (the C-CANCEL path).
type cannedFindHandler struct {
	results []findResponse

	mu          sync.Mutex
	calledQuery *dicom.DataSet
	calledLevel QueryLevel
	calledInfo  OpInfo
}

func (h *cannedFindHandler) Find(ctx context.Context, query *dicom.DataSet, level QueryLevel, info OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	h.mu.Lock()
	h.calledQuery = query
	h.calledLevel = level
	h.calledInfo = info
	h.mu.Unlock()
	return func(yield func(Status, *dicom.DataSet) bool) {
		for _, r := range h.results {
			status := NewStatus(r.status, ServiceClassFind)
			if !yield(status, r.identifier) {
				return
			}
		}
	}
}

// blockingFindHandler yields one Pending match then blocks until its context is cancelled, so the
// C-CANCEL path can be exercised: a C-CANCEL-RQ arriving mid-drain must cancel the handler's context
// and produce a terminal Cancel RSP.
type blockingFindHandler struct {
	first *dicom.DataSet

	mu          sync.Mutex
	ctxCanceled bool
}

func (h *blockingFindHandler) Find(ctx context.Context, _ *dicom.DataSet, _ QueryLevel, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {
		if !yield(StatusFindPending, h.first) {
			return
		}
		<-ctx.Done()
		h.mu.Lock()
		h.ctxCanceled = true
		h.mu.Unlock()
	}
}

func (h *blockingFindHandler) wasCanceled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ctxCanceled
}

// signalingBlockingFindHandler yields one Pending match, signals it has been entered, then blocks on
// its context — the lever the cooperative-shutdown regression uses to park a C-FIND drain mid-query
// while Server.Shutdown is invoked.
type signalingBlockingFindHandler struct {
	first   *dicom.DataSet
	entered chan struct{}

	mu          sync.Mutex
	ctxCanceled bool
	enteredOnce sync.Once
}

func (h *signalingBlockingFindHandler) Find(ctx context.Context, _ *dicom.DataSet, _ QueryLevel, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {
		if !yield(StatusFindPending, h.first) {
			return
		}
		h.enteredOnce.Do(func() { close(h.entered) })
		<-ctx.Done()
		h.mu.Lock()
		h.ctxCanceled = true
		h.mu.Unlock()
	}
}

func (h *signalingBlockingFindHandler) wasCanceled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ctxCanceled
}

// findScpHandler bundles a FindHandler so a Server hosts only the C-FIND capability (interface
// segregation: it implements FindHandler alone, no Echo/Store).
type findScpHandler struct {
	h FindHandler
}

func (s *findScpHandler) Find(ctx context.Context, query *dicom.DataSet, level QueryLevel, info OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return s.h.Find(ctx, query, level, info)
}

// startFindServer stands up a go-radx Server hosting the given handler with the Query/Retrieve
// contexts on loopback, returning the bound address. Shutdown is registered on the test.
func startFindServer(t *testing.T, h any) string {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := NewServer(ae, QueryRetrieveContexts(), h)
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Error("ListenAndServe did not return after Shutdown")
		}
	})
	return srv.Addr().String()
}

// dialFindServerSCU associates to a go-radx Server proposing the Query/Retrieve contexts.
func dialFindServerSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("FINDSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("RADX-SCP"), QueryRetrieveContexts())
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// TestServerAnswersCFind drives a go-radx SCU C-FIND against a go-radx Server hosting a FindHandler
// that yields two matches then Success. The SCU iterator must surface two Pending matches with their
// identifiers and one terminal Success — the in-process SCU↔SCP round-trip.
func TestServerAnswersCFind(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	match2 := dicom.NewDataSet()
	match2.SetString(dicom.TagStudyInstanceUID, "1.2.3.2")

	canned := &cannedFindHandler{results: []findResponse{
		{status: StatusFindPending.Code, identifier: match1},
		{status: StatusFindPending.Code, identifier: match2},
		{status: StatusFindSuccess.Code},
	}}
	addr := startFindServer(t, &findScpHandler{h: canned})

	assoc, ctx, cancel := dialFindServerSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagPatientID, "PID-1")
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var pendingUIDs []string
	var terminal Status
	var terminalSeen bool
	for status, ds := range assoc.Find(ctx, query, QueryLevelStudy) {
		switch {
		case status.IsPending():
			if ds == nil {
				t.Error("Pending response yielded a nil dataset; want a matching identifier")
				continue
			}
			uid, _ := ds.GetString(dicom.TagStudyInstanceUID)
			pendingUIDs = append(pendingUIDs, uid)
		default:
			terminal = status
			terminalSeen = true
			if ds != nil {
				t.Error("terminal response yielded a non-nil dataset; want nil")
			}
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if len(pendingUIDs) != 2 || pendingUIDs[0] != "1.2.3.1" || pendingUIDs[1] != "1.2.3.2" {
		t.Errorf("pending matches = %v, want [1.2.3.1 1.2.3.2]", pendingUIDs)
	}
	if !terminalSeen || !terminal.IsSuccess() {
		t.Errorf("terminal status = %s (seen=%v), want Success", terminal, terminalSeen)
	}

	// The handler must have seen the requested level and the query identifier (with the level the SCU
	// wrote into it).
	canned.mu.Lock()
	gotLevel := canned.calledLevel
	gotQuery := canned.calledQuery
	gotInfo := canned.calledInfo
	canned.mu.Unlock()
	if gotLevel != QueryLevelStudy {
		t.Errorf("handler called with level %v, want QueryLevelStudy", gotLevel)
	}
	if gotQuery == nil {
		t.Fatal("handler received a nil query")
	}
	if lvl, _ := gotQuery.GetString(dicom.TagQueryRetrieveLevel); lvl != "STUDY" {
		t.Errorf("handler query Query/Retrieve Level = %q, want STUDY", lvl)
	}
	if gotInfo.CallingAETitle != "FINDSCU" || gotInfo.CalledAETitle != "RADX-SCP" {
		t.Errorf("handler OpInfo AE titles = %+v, want calling FINDSCU / called RADX-SCP", gotInfo)
	}
	if gotInfo.SOPClassUID != studyRootFindSOPClass {
		t.Errorf("handler OpInfo SOP Class = %q, want Study Root FIND", gotInfo.SOPClassUID)
	}
}

// TestServeFindZeroMatchesSendsTerminalSuccess is the no-hang requirement: a handler that yields zero
// matches (only Success) still produces a single terminal Success RSP, so the SCU's range loop ends
// with one terminal status and no hang.
func TestServeFindZeroMatchesSendsTerminalSuccess(t *testing.T) {
	canned := &cannedFindHandler{results: []findResponse{
		{status: StatusFindSuccess.Code},
	}}
	addr := startFindServer(t, &findScpHandler{h: canned})

	assoc, ctx, cancel := dialFindServerSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var statuses []Status
	for status, ds := range assoc.Find(ctx, query, QueryLevelStudy) {
		statuses = append(statuses, status)
		if ds != nil {
			t.Error("zero-match terminal yielded a non-nil dataset")
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if len(statuses) != 1 || !statuses[0].IsSuccess() {
		t.Fatalf("zero-match Find yielded %v, want exactly one terminal Success", statuses)
	}
}

// TestServeFindUnsupported is the interface-segregation regression: a C-FIND-RQ routed to a handler
// with no FindHandler capability (a store-only handler) is refused with a terminal C-FIND-RSP
// carrying StatusSOPClassNotSupported (0x0122), never a panic and never an accepted query.
func TestServeFindUnsupported(t *testing.T) {
	addr := startFindServer(t, &storeOnlyHandler{status: StatusStoreSuccess})

	assoc, ctx, cancel := dialFindServerSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var statuses []Status
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		statuses = append(statuses, status)
	}
	_ = assoc.Release(ctx)

	if len(statuses) != 1 {
		t.Fatalf("C-FIND to a store-only handler yielded %d statuses, want 1 terminal refusal", len(statuses))
	}
	if statuses[0].Code != StatusSOPClassNotSupported.Code {
		t.Errorf("terminal status = %s, want 0x%04X (Refused: SOP Class Not Supported)",
			statuses[0], StatusSOPClassNotSupported.Code)
	}
	if err := assoc.LastError(); err != nil {
		t.Errorf("LastError = %v, want nil (a graceful refusal is data, not a transport fault)", err)
	}
}

// TestServeFindCancelStopsHandler verifies a C-CANCEL-RQ arriving mid-drain cancels the handler's
// context (stopping its iterator) and the dispatcher sends a terminal 0xFE00 Cancel RSP. The SCU
// breaks the range loop after the first match, which sends the C-CANCEL; the handler — blocked on its
// context after that first yield — is woken and the iterator stops.
func TestServeFindCancelStopsHandler(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	handler := &blockingFindHandler{first: match1}
	addr := startFindServer(t, &findScpHandler{h: handler})

	assoc, ctx, cancel := dialFindServerSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	count := 0
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		if status.IsPending() {
			count++
			break // stop after the first match — sends a C-CANCEL-RQ
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError after cancel-drain = %v, want nil", err)
	}
	if count != 1 {
		t.Errorf("consumed %d pending matches before break, want 1", count)
	}

	// The dispatcher must cancel the handler's context so its iterator stops (no goroutine leak).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !handler.wasCanceled() {
		time.Sleep(10 * time.Millisecond)
	}
	if !handler.wasCanceled() {
		t.Error("C-CANCEL did not cancel the handler's context; the handler iterator was not stopped")
	}
	_ = assoc.Release(ctx)
}

// reusableFindHandler yields a fixed list of results on EVERY call, so an association can run more
// than one C-FIND against it (the reuse-after-cancel regression needs a second query to succeed).
type reusableFindHandler struct {
	results []findResponse
}

func (h *reusableFindHandler) Find(_ context.Context, _ *dicom.DataSet, _ QueryLevel, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {
		for _, r := range h.results {
			if !yield(NewStatus(r.status, ServiceClassFind), r.identifier) {
				return
			}
		}
	}
}

// TestServeFindLateCancelDoesNotPoisonAssociation is the regression for the trailing-C-CANCEL race
// (Codex round-2 finding): when an SCU breaks a C-FIND after a Pending and the query terminates at
// about the same moment, the SCU's C-CANCEL can reach the SCP after its drain already returned. A
// standalone C-CANCEL at the top-level dispatch must be IGNORED (PS3.7 §9.3.2.3), not faulted, so the
// association survives and a subsequent C-FIND on it still works. The test breaks the first query
// after one match, then runs a SECOND C-FIND on the same association and asserts it completes — which
// is only possible if the trailing cancel did not close the association.
func TestServeFindLateCancelDoesNotPoisonAssociation(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	handler := &reusableFindHandler{results: []findResponse{
		{status: StatusFindPending.Code, identifier: match1},
		{status: StatusFindSuccess.Code},
	}}
	addr := startFindServer(t, &findScpHandler{h: handler})

	assoc, ctx, cancel := dialFindServerSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	// First query: break after the first Pending, sending a C-CANCEL the SCP may receive late.
	first := 0
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		if status.IsPending() {
			first++
			break
		}
	}
	if first != 1 {
		t.Fatalf("first query consumed %d pending before break, want 1", first)
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("first query LastError after break = %v, want nil", err)
	}

	// Give any trailing C-CANCEL time to land at the SCP's top-level dispatch.
	time.Sleep(200 * time.Millisecond)

	// Second query on the SAME association: it must succeed, proving the trailing cancel did not
	// poison/close the association.
	var secondPending int
	var secondTerminalSuccess bool
	for status, ds := range assoc.Find(ctx, query, QueryLevelStudy) {
		switch {
		case status.IsPending():
			secondPending++
			_ = ds
		case status.IsSuccess():
			secondTerminalSuccess = true
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("second query LastError = %v, want nil (the association must survive the late cancel)", err)
	}
	if secondPending != 1 || !secondTerminalSuccess {
		t.Errorf("second query: pending=%d success=%v, want 1 pending then Success", secondPending, secondTerminalSuccess)
	}
	if err := assoc.Release(ctx); err != nil {
		t.Errorf("Release after reuse = %v, want nil", err)
	}
}

// TestServeFindReleaseAfterTerminalSucceeds is the regression for the terminal-response watcher race
// (Codex round-3 finding): the cancel watcher must stop its blocking read BEFORE the terminal RSP is
// sent, so the A-RELEASE-RQ the SCU sends immediately after receiving the terminal reaches the main
// dispatch loop rather than being swallowed by the watcher. It runs a C-FIND to completion then
// releases at once, and asserts the release succeeds — a swallowed A-RELEASE-RQ would hang or fault
// it. A follow-up query proves the association also remains reusable.
func TestServeFindReleaseAfterTerminalSucceeds(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	handler := &reusableFindHandler{results: []findResponse{
		{status: StatusFindPending.Code, identifier: match1},
		{status: StatusFindSuccess.Code},
	}}
	addr := startFindServer(t, &findScpHandler{h: handler})

	assoc, ctx, cancel := dialFindServerSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	// Run the first query to its terminal, consuming all responses.
	for range assoc.Find(ctx, query, QueryLevelStudy) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("first query LastError = %v, want nil", err)
	}

	// A second query on the same association must still complete (the watcher did not poison reuse).
	var pending int
	var success bool
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		if status.IsPending() {
			pending++
		}
		if status.IsSuccess() {
			success = true
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("second query LastError = %v, want nil", err)
	}
	if pending != 1 || !success {
		t.Errorf("reuse query: pending=%d success=%v, want 1 pending then Success", pending, success)
	}

	// Immediate release after the terminal must succeed — the watcher must not have consumed the
	// A-RELEASE-RQ.
	if err := assoc.Release(ctx); err != nil {
		t.Errorf("Release immediately after terminal C-FIND-RSP = %v, want nil", err)
	}
}

// sendRawFindRQ associates to addr proposing the Query/Retrieve contexts, then sends a hand-built
// C-FIND-RQ — optionally with no identifier or an identifier missing the level — and returns the
// first response status it reads. It is the malformed-request driver the fail-closed SCP regressions
// use, since the production Find SCU always sends a well-formed identifier with a level.
func sendRawFindRQ(t *testing.T, addr string, withIdentifier bool, writeLevel bool) Status {
	t.Helper()
	ae, err := NewAE(AETitle("FINDSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, addr, AETitle("RADX-SCP"), QueryRetrieveContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Abort(ctx) }()

	pcID, ts, ok := assoc.contextForQuery(studyRootFindSOPClass)
	if !ok {
		t.Fatal("no accepted Study Root FIND context")
	}
	conn := assoc.requestor.Conn()
	m := assoc.requestor.Machine()

	rq := CommandSet{
		CommandField:        CommandCFindRQ,
		MessageID:           1,
		AffectedSOPClassUID: dicom.UID(studyRootFindSOPClass),
		HasPriority:         true,
		Priority:            PriorityMedium,
		CommandDataSetType:  CommandDataSetNotPresent,
	}
	var identifier *dicom.DataSet
	if withIdentifier {
		rq.CommandDataSetType = CommandDataSetPresent
		identifier = dicom.NewDataSet()
		identifier.SetEmpty(dicom.TagStudyInstanceUID)
		if writeLevel {
			identifier.SetString(dicom.TagQueryRetrieveLevel, "STUDY")
		}
	}
	if err := sendMessage(ctx, conn, m, pcID, rq, identifier, ts, assoc.sendCap()); err != nil {
		t.Fatalf("send raw C-FIND-RQ: %v", err)
	}
	rsp, _, _, err := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if err != nil {
		t.Fatalf("read C-FIND-RSP: %v", err)
	}
	if rsp.CommandField != CommandCFindRSP || !rsp.HasStatus {
		t.Fatalf("response is not a C-FIND-RSP with a status: %+v", rsp)
	}
	return NewStatus(rsp.Status, ServiceClassFind)
}

// TestServeFindRejectsMissingIdentifier is the fail-closed regression for a C-FIND-RQ that declares
// no identifier (Codex round-3 finding): the SCP must refuse it with a terminal Failure rather than
// fabricating an empty identifier a handler could treat as a match-everything wildcard.
func TestServeFindRejectsMissingIdentifier(t *testing.T) {
	handler := &cannedFindHandler{results: []findResponse{{status: StatusFindSuccess.Code}}}
	addr := startFindServer(t, &findScpHandler{h: handler})

	status := sendRawFindRQ(t, addr, false /* no identifier */, false)
	if !status.IsFailure() {
		t.Errorf("C-FIND with no identifier got terminal %s, want a Failure (fail-closed)", status)
	}
	// The handler must never have been invoked for a malformed request.
	handler.mu.Lock()
	called := handler.calledQuery != nil
	handler.mu.Unlock()
	if called {
		t.Error("the FindHandler was invoked for a C-FIND with no identifier; want fail-closed before dispatch")
	}
}

// TestServeFindRejectsMissingLevel is the fail-closed regression for a Patient/Study Root C-FIND-RQ
// whose identifier omits the Query/Retrieve Level (Codex round-3 finding): the SCP must refuse it
// with a terminal Failure rather than silently defaulting to Study level and running the wrong scope.
func TestServeFindRejectsMissingLevel(t *testing.T) {
	handler := &cannedFindHandler{results: []findResponse{{status: StatusFindSuccess.Code}}}
	addr := startFindServer(t, &findScpHandler{h: handler})

	status := sendRawFindRQ(t, addr, true /* identifier present */, false /* no level written */)
	if !status.IsFailure() {
		t.Errorf("C-FIND with no Query/Retrieve Level got terminal %s, want a Failure (fail-closed)", status)
	}
	handler.mu.Lock()
	called := handler.calledQuery != nil
	handler.mu.Unlock()
	if called {
		t.Error("the FindHandler was invoked for a C-FIND with no level; want fail-closed before dispatch")
	}
}

// dialFindServerSCUUnlimited associates to a go-radx Server advertising an UNLIMITED max PDU length
// (WithMaxPDULength(0)), the condition the large-identifier fragmentation regression needs.
func dialFindServerSCUUnlimited(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("FINDSCU"), WithDIMSETimeout(5*time.Second), WithMaxPDULength(0))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("RADX-SCP"), QueryRetrieveContexts())
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// TestServeFindFragmentsLargeIdentifierForUnlimitedPeer is the regression for the unlimited-peer
// max-PDU fragmentation defect (Codex round-2 finding): when the SCU advertises max PDU length 0
// (unlimited), the SCP must NOT pass that raw 0 into the fragmenter — which would place a whole match
// identifier in one P-DATA-TF and risk exceeding the PDU ceiling on a large identifier. The SCP
// resolves the send cap to the local default for an unlimited peer, so a large identifier still
// fragments and reassembles intact on the SCU.
func TestServeFindFragmentsLargeIdentifierForUnlimitedPeer(t *testing.T) {
	// A match identifier whose value comfortably exceeds the default 16382-byte PDV cap, so it can
	// only round-trip if the SCP fragmented it into several P-DATA-TF PDUs.
	bigValue := make([]byte, 40000)
	for i := range bigValue {
		bigValue[i] = 'A'
	}
	match := dicom.NewDataSet()
	match.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	match.SetString(dicom.TagStudyDescription, string(bigValue))

	handler := &reusableFindHandler{results: []findResponse{
		{status: StatusFindPending.Code, identifier: match},
		{status: StatusFindSuccess.Code},
	}}
	addr := startFindServer(t, &findScpHandler{h: handler})

	assoc, ctx, cancel := dialFindServerSCUUnlimited(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)
	query.SetEmpty(dicom.TagStudyDescription)

	var gotDesc string
	var pending int
	var success bool
	for status, ds := range assoc.Find(ctx, query, QueryLevelStudy) {
		switch {
		case status.IsPending():
			pending++
			if ds != nil {
				gotDesc, _ = ds.GetString(dicom.TagStudyDescription)
			}
		case status.IsSuccess():
			success = true
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError = %v, want nil (the large identifier must fragment and reassemble)", err)
	}
	_ = assoc.Release(ctx)

	if pending != 1 || !success {
		t.Fatalf("unlimited-peer large-identifier Find: pending=%d success=%v, want 1 pending then Success", pending, success)
	}
	if len(gotDesc) != len(bigValue) {
		t.Errorf("reassembled Study Description length = %d, want %d (fragmentation/reassembly lost bytes)", len(gotDesc), len(bigValue))
	}
}

// TestFindResponseSendCapBoundsUnlimitedPeer is the deterministic unit regression for the
// unlimited-peer fragmentation fix (Codex round-2 finding): the send cap the SCP resolves for a peer
// advertising max PDU 0 (unlimited) must be the bounded local default, NOT unlimited, so a large
// match identifier fragments into more than one P-DATA-TF. The loopback round-trip above proves the
// path works end to end but cannot prove fragmentation occurred (loopback enforces no PDU ceiling);
// this asserts the resolved cap directly against the fragmenter, the exact value serveFindMessage
// passes to sendMessage.
func TestFindResponseSendCapBoundsUnlimitedPeer(t *testing.T) {
	const unlimitedPeer = MaxPDULength(0)
	sendCap := unlimitedPeer.SendCap(defaultMaxPDULength)
	if sendCap.IsUnlimited() {
		t.Fatal("unlimited peer max resolved to an unlimited send cap; a large identifier would be one oversized PDU")
	}
	if sendCap != defaultMaxPDULength {
		t.Errorf("resolved send cap = %d, want the local default %d", sendCap, defaultMaxPDULength)
	}

	// A match identifier larger than the resolved cap must fragment into more than one PDU.
	bigValue := make([]byte, 40000)
	for i := range bigValue {
		bigValue[i] = 'A'
	}
	identifier := dicom.NewDataSet()
	identifier.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	identifier.SetString(dicom.TagStudyDescription, string(bigValue))

	rsp := CommandSet{
		CommandField:              CommandCFindRSP,
		MessageIDBeingRespondedTo: 1,
		AffectedSOPClassUID:       dicom.UID(studyRootFindSOPClass),
		HasStatus:                 true,
		Status:                    StatusFindPending.Code,
		CommandDataSetType:        CommandDataSetPresent,
	}
	pdus, err := fragmentMessage(rsp, identifier, dicom.ExplicitVRLittleEndian, 1, sendCap)
	if err != nil {
		t.Fatalf("fragmentMessage: %v", err)
	}
	if len(pdus) < 2 {
		t.Errorf("large identifier produced %d PDU(s) under the resolved cap, want fragmentation into >= 2", len(pdus))
	}
}

// TestServeFindReturnsOnContextCancelWhileHandlerBlocked is the cooperative-shutdown regression for
// the C-FIND drain (Codex round-1 finding): when the parent context is cancelled while the SCP is
// parked waiting for the next C-FIND event — the handler blocked between yields, the peer holding the
// connection OPEN (no inbound C-CANCEL, no transport close) — serveFindMessage must return promptly
// with the context error, NOT wedge. Without the ctx.Done() arm in the drain's select, neither helper
// goroutine sends (the cancel watcher suppresses its context error and the match pump takes its own
// ctx.Done() branch), the select blocks forever, and the association goroutine outlives the deadline.
//
// It drives serveFindMessage directly over a loopback acceptor so the connection stays open after the
// SCU reads the first Pending — the precise condition Server.Shutdown's connection-close otherwise
// masks. A peer that holds its socket open while a query is in flight is a real misbehaving SCU.
func TestServeFindReturnsOnContextCancelWhileHandlerBlocked(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	handler := &signalingBlockingFindHandler{first: match1, entered: make(chan struct{})}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	// parentCtx is the SCP's per-operation context; cancelling it simulates Server.Shutdown's handler
	// cancel WITHOUT closing the connection.
	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()
	scpDone := make(chan error, 1)

	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			scpDone <- aerr
			return
		}
		conn := dul.NewConn(nc, 0)
		acceptCtx, acceptCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer acceptCancel()
		acc, perr := acse.Accept(acceptCtx, conn, acse.AcceptParams{
			CalledAETitle: "RADX-SCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(studyRootFindSOPClass),
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			_ = nc.Close()
			scpDone <- perr
			return
		}
		var pcID uint8
		ts := dicom.ImplicitVRLittleEndian
		for _, pc := range acc.AcceptedContexts() {
			if pc.Result == 0 {
				pcID = pc.ID
				ts = dicom.TransferSyntax(pc.TransferSyntax)
				break
			}
		}
		cmd, identifier, _, rerr := receiveMessage(acceptCtx, acc.Conn(), acc.Machine(), newMessageReassembler(ts))
		if rerr != nil {
			scpDone <- rerr
			return
		}
		// Serve under the cancellable parent context; this is the call under test.
		info := OpInfo{CallingAETitle: AETitle("FINDSCU"), CalledAETitle: AETitle("RADX-SCP")}
		scpDone <- serveFindMessage(parentCtx, acc, handler, cmd, identifier, pcID, info)
	}()

	// SCU side: associate and send a C-FIND-RQ, read the first Pending, then HOLD the connection open.
	assoc, ctx, cancel := dialFindServerSCU(t, ln.Addr().String())
	defer cancel()
	t.Cleanup(func() { _ = assoc.Abort(context.Background()) })

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	firstSeen := make(chan struct{})
	go func() {
		var once sync.Once
		for status := range assoc.Find(ctx, query, QueryLevelStudy) {
			if status.IsPending() {
				once.Do(func() { close(firstSeen) })
			}
		}
	}()

	select {
	case <-firstSeen:
	case <-time.After(5 * time.Second):
		t.Fatal("SCU never received the first Pending; cannot park the SCP drain")
	}
	// The handler is now blocked between yields and the SCP drain is waiting on its select with the
	// connection open. Cancel the parent context and assert serveFindMessage returns promptly.
	select {
	case <-handler.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("C-FIND handler never parked between yields")
	}
	start := time.Now()
	cancelParent()

	select {
	case err := <-scpDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("serveFindMessage returned %v, want context.Canceled (the prompt ctx-cancel path)", err)
		}
		if elapsed := time.Since(start); elapsed >= time.Second {
			t.Fatalf("serveFindMessage took %s to return after ctx cancel (>= 1s); the drain wedged", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serveFindMessage did not return after its context was cancelled; the C-FIND drain wedged")
	}

	// The handler must have observed its cancelled context (cooperative wake), proving the
	// handler-context cancel reached it rather than the goroutine being abandoned.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !handler.wasCanceled() {
		time.Sleep(10 * time.Millisecond)
	}
	if !handler.wasCanceled() {
		t.Error("the parked C-FIND handler never observed its cancelled context")
	}
}

// TestFindCancelWatcherQueuesResultBeforeOnCancel deterministically pins the watcher's
// queue-then-cancel ordering contract (the cancel-signal race): at the moment onCancel is
// invoked for a matching C-CANCEL-RQ, the cancel result must ALREADY be queued on the buffered
// result channel. onCancel is what cancels the C-MOVE drain's handler context and thereby
// unblocks an in-flight sub-operation store; were the result queued after, the drain's
// non-blocking re-check could miss the cancel, miscount the interrupted store as a destination
// failure, and emit a Pending after the cancel. The natural interleaving almost never exposes
// this (the store's cancellation unwind is far slower than the watcher's next instruction), so
// the ordering is asserted directly.
func TestFindCancelWatcherQueuesResultBeforeOnCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	accepted := make(chan *acse.Acceptor, 1)
	acceptErr := make(chan error, 1)
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			acceptErr <- aerr
			return
		}
		conn := dul.NewConn(nc, 0)
		acceptCtx, acceptCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer acceptCancel()
		acc, perr := acse.Accept(acceptCtx, conn, acse.AcceptParams{
			CalledAETitle: "RADX-SCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(studyRootFindSOPClass),
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			_ = nc.Close()
			acceptErr <- perr
			return
		}
		accepted <- acc
	}()

	scuAE, err := NewAE(AETitle("CANCELSCU"), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := scuAE.Associate(ctx, ln.Addr().String(), AETitle("RADX-SCP"), QueryRetrieveContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	t.Cleanup(func() { _ = assoc.Abort(context.Background()) })

	var acc *acse.Acceptor
	select {
	case acc = <-accepted:
	case aerr := <-acceptErr:
		t.Fatalf("accept: %v", aerr)
	case <-time.After(5 * time.Second):
		t.Fatal("acceptor never established")
	}

	const msgID = uint16(7)
	var w *findCancelWatcher
	ready := make(chan struct{})
	queuedAtCancel := make(chan int, 1)
	w = newFindCancelWatcher(context.Background(), acc.Conn(), acc.Machine(), msgID, func() {
		// ready orders the test goroutine's write of w before this read; the watcher goroutine
		// blocks here until the constructor's return value has been assigned.
		<-ready
		queuedAtCancel <- len(w.result)
	})
	close(ready)
	t.Cleanup(w.stop)

	pcID, _, ok := assoc.contextForQuery(studyRootFindSOPClass)
	if !ok {
		t.Fatal("no accepted Study Root FIND presentation context")
	}
	cancelRQ := CommandSet{
		CommandField:              CommandCCancelRQ,
		MessageIDBeingRespondedTo: msgID,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	if err := sendCommand(ctx, assoc.requestor.Conn(), assoc.requestor.Machine(), pcID, cancelRQ); err != nil {
		t.Fatalf("send C-CANCEL-RQ: %v", err)
	}

	select {
	case n := <-queuedAtCancel:
		if n != 1 {
			t.Fatalf("at onCancel time the watcher held %d queued result(s), want 1: the cancel must be visible before the context is cancelled", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("onCancel was never invoked for the matching C-CANCEL-RQ")
	}

	// The queued result is still consumable by the drain afterwards.
	select {
	case res := <-w.result:
		if res.err != nil {
			t.Fatalf("cancel result carried error %v, want a clean cancel", res.err)
		}
	case <-time.After(time.Second):
		t.Fatal("the cancel result was never delivered on the result channel")
	}
}
