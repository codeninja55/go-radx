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

// TestValidateMoveContext is the regression for the C-MOVE negotiation guard (the symmetry with
// validateFindContext): a C-MOVE must arrive on a context whose negotiated abstract syntax is a
// Query/Retrieve MOVE information model AND whose Affected SOP Class matches that model, else it is a
// protocol fault — a peer cannot run a retrieve outside the negotiated/declared SOP Class.
func TestValidateMoveContext(t *testing.T) {
	abstractFor := func(pcID uint8) (dicom.SOPClassUID, bool) {
		switch pcID {
		case 1:
			return studyRootMoveSOPClass, true // a MOVE context
		case 3:
			return studyRootFindSOPClass, true // a FIND context (not MOVE)
		default:
			return "", false
		}
	}

	match := CommandSet{CommandField: CommandCMoveRQ, AffectedSOPClassUID: dicom.UID(studyRootMoveSOPClass)}
	if err := validateMoveContext(match, 1, abstractFor, Sta6); err != nil {
		t.Errorf("matching MOVE context rejected: %v", err)
	}

	onFind := CommandSet{CommandField: CommandCMoveRQ, AffectedSOPClassUID: dicom.UID(studyRootFindSOPClass)}
	if err := validateMoveContext(onFind, 3, abstractFor, Sta6); err == nil {
		t.Error("C-MOVE on a non-MOVE (FIND) context = nil error, want a protocol fault")
	} else {
		var pe *ProtocolError
		if !errors.As(err, &pe) {
			t.Errorf("error = %T, want *ProtocolError", err)
		}
	}

	mismatch := CommandSet{CommandField: CommandCMoveRQ, AffectedSOPClassUID: dicom.UID(patientRootMoveSOPClass)}
	if err := validateMoveContext(mismatch, 1, abstractFor, Sta6); err == nil {
		t.Error("mismatched MOVE SOP class on a MOVE context = nil error, want a protocol fault")
	}

	if err := validateMoveContext(match, 9, abstractFor, Sta6); err == nil {
		t.Error("unknown presentation context = nil error, want a protocol fault")
	}
}

// movingFindHandler yields a fixed list of matched instance datasets (each as a Pending with the
// instance dataset) then a terminal Success, so a C-MOVE drain test can assert the runtime C-STOREs
// each matched instance to the destination AE. It satisfies the full Handler union (Echo/Store/Find
// are unused for the move path but the Handler interface requires them).
type movingFindHandler struct {
	instances []*dicom.DataSet

	mu          sync.Mutex
	calledQuery *dicom.DataSet
	calledDest  AETitle
}

func (h *movingFindHandler) Echo(context.Context, OpInfo) Status { return StatusEchoSuccess }
func (h *movingFindHandler) Store(context.Context, *dicom.DataSet, OpInfo) Status {
	return StatusStoreSuccess
}

func (h *movingFindHandler) Find(_ context.Context, _ *dicom.DataSet, _ QueryLevel, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return func(func(Status, *dicom.DataSet) bool) {}
}

func (h *movingFindHandler) Move(_ context.Context, query *dicom.DataSet, _ QueryLevel, dest AETitle, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	h.mu.Lock()
	h.calledQuery = query
	h.calledDest = dest
	h.mu.Unlock()
	return func(yield func(Status, *dicom.DataSet) bool) {
		for _, ds := range h.instances {
			if !yield(StatusMovePending, ds) {
				return
			}
		}
		yield(StatusMoveSuccess, nil)
	}
}

// recordingDestination is a StoreHandler used as the C-MOVE destination AE: it records the SOP
// Instance UID, the Message ID, and the Move Originator AE Title of each sub-operation C-STORE it
// receives, so the move tests can assert the instances arrived, the DIMSE-016 distinct-Message-ID
// rule holds, and the originator is the original C-MOVE requestor.
type recordingDestination struct {
	mu          sync.Mutex
	instances   []string
	msgIDs      []uint16
	originators []AETitle
}

func (d *recordingDestination) Store(_ context.Context, ds *dicom.DataSet, info OpInfo) Status {
	sopInstance, _ := ds.GetString(tagSOPInstanceUID)
	d.mu.Lock()
	d.instances = append(d.instances, sopInstance)
	d.msgIDs = append(d.msgIDs, info.MessageID)
	d.originators = append(d.originators, info.MoveOriginatorAETitle)
	d.mu.Unlock()
	return StatusStoreSuccess
}

func (d *recordingDestination) snapshot() ([]string, []uint16) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ins := append([]string(nil), d.instances...)
	ids := append([]uint16(nil), d.msgIDs...)
	return ins, ids
}

func (d *recordingDestination) originatorSnapshot() []AETitle {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]AETitle(nil), d.originators...)
}

// instanceDataset builds a minimal storable dataset carrying the SOP Class/Instance UID a C-STORE
// reads to select a context and build the RQ. MR Image Storage is in the validated Storage set.
func instanceDataset(sopInstanceUID string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(tagSOPClassUID, "1.2.840.10008.5.1.4.1.1.4") // MR Image Storage
	ds.SetString(tagSOPInstanceUID, sopInstanceUID)
	return ds
}

// startDestinationSCP stands up a go-radx Store SCP (the C-MOVE destination AE) on loopback,
// returning its AE title, address, and the recording handler.
func startDestinationSCP(t *testing.T) (AETitle, string, *recordingDestination) {
	t.Helper()
	const destTitle = AETitle("RADX-DEST")
	ae, err := NewAE(destTitle)
	if err != nil {
		t.Fatalf("NewAE destination: %v", err)
	}
	dest := &recordingDestination{}
	srv := NewServer(ae, StorageContexts(), dest)
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})
	return destTitle, srv.Addr().String(), dest
}

// startMoveServer stands up a go-radx Move SCP hosting the given handler, with the Query/Retrieve
// contexts and the supplied move-destination table, on loopback. It also advertises the Storage
// contexts on its OUTBOUND associations to the destination (the Server's AE drives those), so the
// sub-operation C-STOREs negotiate a Storage context.
func startMoveServer(t *testing.T, h any, dests map[AETitle]string) string {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-MOVESCP"))
	if err != nil {
		t.Fatalf("NewAE move SCP: %v", err)
	}
	srv := NewServer(ae, QueryRetrieveContexts(), h, WithMoveDestinations(dests))
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})
	return srv.Addr().String()
}

// dialMoveServerSCU associates to a go-radx Move SCP proposing the Query/Retrieve contexts.
func dialMoveServerSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("MOVESCU"), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("RADX-MOVESCP"), QueryRetrieveContexts())
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// TestServerAnswersCMove is the in-process C-MOVE round-trip: a go-radx Move SCU drives a C-MOVE
// against a go-radx Move SCP, which opens an outbound association to a THIRD go-radx Store SCP (the
// destination AE) and C-STOREs the matched instances there. The SCU must surface a terminal Success
// and the destination must receive both instances.
func TestServerAnswersCMove(t *testing.T) {
	destTitle, destAddr, dest := startDestinationSCP(t)

	handler := &movingFindHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"),
	}}
	moveAddr := startMoveServer(t, handler, map[AETitle]string{destTitle: destAddr})

	assoc, ctx, cancel := dialMoveServerSCU(t, moveAddr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var terminal Status
	var terminalSeen bool
	for status := range assoc.Move(ctx, query, QueryLevelStudy, destTitle) {
		if !status.IsPending() {
			terminal = status
			terminalSeen = true
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if !terminalSeen || !terminal.IsSuccess() {
		t.Errorf("terminal status = %s (seen=%v), want Success", terminal, terminalSeen)
	}

	instances, _ := dest.snapshot()
	if len(instances) != 2 {
		t.Fatalf("destination received %v, want 2 instances", instances)
	}
	want := map[string]bool{"1.2.3.1": true, "1.2.3.2": true}
	for _, got := range instances {
		if !want[got] {
			t.Errorf("destination received unexpected instance %q", got)
		}
	}

	final := assoc.SubOperationCounts()
	if final.Completed != 2 {
		t.Errorf("final Completed count = %d, want 2", final.Completed)
	}
	if final.Failed != 0 {
		t.Errorf("final Failed count = %d, want 0", final.Failed)
	}

	handler.mu.Lock()
	gotDest := handler.calledDest
	handler.mu.Unlock()
	if gotDest != destTitle {
		t.Errorf("handler called with destination %q, want %q", gotDest, destTitle)
	}

	// Each sub-operation C-STORE must carry the Move Originator AE Title of the AE that INVOKED the
	// C-MOVE (the calling SCU "MOVESCU"), not the Move SCP's own title (PS3.7 §9.1.1).
	for _, orig := range dest.originatorSnapshot() {
		if orig != AETitle("MOVESCU") {
			t.Errorf("sub-operation Move Originator AE Title = %q, want the C-MOVE requestor MOVESCU", orig)
		}
	}
}

// TestMoveSubOperationsUseDistinctMessageIDs is the DIMSE-016 regression: the SCP's sub-operation
// C-STOREs to the destination must each carry a distinct, non-zero Message ID (the prototype used
// MessageID 0 and read exactly one P-DATA-TF, miscounting failures and hanging against compliant
// peers).
func TestMoveSubOperationsUseDistinctMessageIDs(t *testing.T) {
	destTitle, destAddr, dest := startDestinationSCP(t)

	handler := &movingFindHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"),
		instanceDataset("1.2.3.3"),
	}}
	moveAddr := startMoveServer(t, handler, map[AETitle]string{destTitle: destAddr})

	assoc, ctx, cancel := dialMoveServerSCU(t, moveAddr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	for range assoc.Move(ctx, query, QueryLevelStudy, destTitle) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	instances, msgIDs := dest.snapshot()
	if len(instances) != 3 {
		t.Fatalf("destination received %d instances, want 3", len(instances))
	}
	if len(msgIDs) != 3 {
		t.Fatalf("destination recorded %d Message IDs, want 3", len(msgIDs))
	}
	seen := make(map[uint16]bool)
	for i, id := range msgIDs {
		if id == 0 {
			t.Errorf("sub-operation %d used Message ID 0 (DIMSE-016: sub-operations need a distinct non-zero ID)", i)
		}
		if seen[id] {
			t.Errorf("Message ID %d reused across sub-operations (DIMSE-016: each needs a distinct ID)", id)
		}
		seen[id] = true
	}
}

// failingDestination is a StoreHandler that fails (returns a Failure status) for one named SOP
// Instance UID and succeeds otherwise, so the SCP's terminal status can be checked when one
// sub-operation fails.
type failingDestination struct {
	failUID string

	mu        sync.Mutex
	instances []string
}

func (d *failingDestination) Store(_ context.Context, ds *dicom.DataSet, _ OpInfo) Status {
	sopInstance, _ := ds.GetString(tagSOPInstanceUID)
	d.mu.Lock()
	d.instances = append(d.instances, sopInstance)
	d.mu.Unlock()
	if sopInstance == d.failUID {
		return StatusStoreCannotUnderstand // 0xC000 Failure
	}
	return StatusStoreSuccess
}

// startFailingDestinationSCP stands up a destination Store SCP that fails one named instance.
func startFailingDestinationSCP(t *testing.T, failUID string) (AETitle, string, *failingDestination) {
	t.Helper()
	const destTitle = AETitle("RADX-DEST")
	ae, err := NewAE(destTitle)
	if err != nil {
		t.Fatalf("NewAE destination: %v", err)
	}
	dest := &failingDestination{failUID: failUID}
	srv := NewServer(ae, StorageContexts(), dest)
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})
	return destTitle, srv.Addr().String(), dest
}

// TestServeMoveTerminalWarningOnSubOpFailure verifies that when one sub-operation C-STORE fails at
// the destination, the SCP reports the terminal 0xB000 "Sub-operations Complete — One or More
// Failures" Warning (not Success, not all-failed), and the failed count reaches the SCU.
func TestServeMoveTerminalWarningOnSubOpFailure(t *testing.T) {
	destTitle, destAddr, _ := startFailingDestinationSCP(t, "1.2.3.2")

	handler := &movingFindHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"), // this one fails at the destination
		instanceDataset("1.2.3.3"),
	}}
	moveAddr := startMoveServer(t, handler, map[AETitle]string{destTitle: destAddr})

	assoc, ctx, cancel := dialMoveServerSCU(t, moveAddr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var terminal Status
	var sawSuccess bool
	for status := range assoc.Move(ctx, query, QueryLevelStudy, destTitle) {
		if status.IsSuccess() {
			sawSuccess = true
		}
		if !status.IsPending() {
			terminal = status
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if sawSuccess {
		t.Error("a partial-failure move was laundered into a Success status (PRD §9.2)")
	}
	if !terminal.IsWarning() {
		t.Errorf("terminal status = %s, want a Warning (0xB000 one or more sub-operations failed)", terminal)
	}
	final := assoc.SubOperationCounts()
	if final.Failed != 1 {
		t.Errorf("final Failed count = %d, want 1", final.Failed)
	}
	if final.Completed != 2 {
		t.Errorf("final Completed count = %d, want 2", final.Completed)
	}
}

// warningDestination is a StoreHandler that returns a Storage Warning (Coercion of Data Elements,
// 0xB000) for every instance — the instance is stored, but with a warning — so the SCP's terminal
// status can be checked when sub-operations warn but none fail.
type warningDestination struct{}

func (d *warningDestination) Store(context.Context, *dicom.DataSet, OpInfo) Status {
	return StatusStoreCoercionOfDataElements // 0xB000 Storage Warning
}

// startWarningDestinationSCP stands up a destination Store SCP that warns on every instance.
func startWarningDestinationSCP(t *testing.T) (AETitle, string) {
	t.Helper()
	const destTitle = AETitle("RADX-DEST")
	ae, err := NewAE(destTitle)
	if err != nil {
		t.Fatalf("NewAE destination: %v", err)
	}
	srv := NewServer(ae, StorageContexts(), &warningDestination{})
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})
	return destTitle, srv.Addr().String()
}

// TestServeMoveTerminalWarningOnSubOpWarning verifies that when every sub-operation C-STORE returns a
// Warning (stored with a warning) and none fails, the SCP reports the terminal 0xB000 Warning, NOT
// Success — a warning is not laundered into Success (PRD §9.2; the Codex round-1 finding).
func TestServeMoveTerminalWarningOnSubOpWarning(t *testing.T) {
	destTitle, destAddr := startWarningDestinationSCP(t)

	handler := &movingFindHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"),
	}}
	moveAddr := startMoveServer(t, handler, map[AETitle]string{destTitle: destAddr})

	assoc, ctx, cancel := dialMoveServerSCU(t, moveAddr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var terminal Status
	var sawSuccess bool
	for status := range assoc.Move(ctx, query, QueryLevelStudy, destTitle) {
		if status.IsSuccess() {
			sawSuccess = true
		}
		if !status.IsPending() {
			terminal = status
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if sawSuccess {
		t.Error("a warning-only move was laundered into a Success status (PRD §9.2)")
	}
	if !terminal.IsWarning() {
		t.Errorf("terminal status = %s, want a Warning (0xB000 sub-operations complete with warnings)", terminal)
	}
	final := assoc.SubOperationCounts()
	if final.Warning != 2 {
		t.Errorf("final Warning count = %d, want 2", final.Warning)
	}
	if final.Failed != 0 {
		t.Errorf("final Failed count = %d, want 0", final.Failed)
	}
}

// TestServeMoveDestinationUnknown verifies that a C-MOVE naming a destination AE the SCP cannot
// resolve answers with the terminal 0xA801 "Move Destination Unknown" Failure, never a panic, and
// opens no outbound association.
func TestServeMoveDestinationUnknown(t *testing.T) {
	handler := &movingFindHandler{instances: []*dicom.DataSet{instanceDataset("1.2.3.1")}}
	// No destinations configured, so any destination is unresolvable.
	moveAddr := startMoveServer(t, handler, map[AETitle]string{})

	assoc, ctx, cancel := dialMoveServerSCU(t, moveAddr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var statuses []Status
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("UNKNOWN-AE")) {
		statuses = append(statuses, status)
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("C-MOVE to an unknown destination yielded %v, want one terminal Failure", statuses)
	}
	if statuses[0].Code != StatusMoveDestinationUnknown.Code {
		t.Errorf("terminal status = %s, want 0xA801 Move Destination Unknown", statuses[0])
	}
	// The handler must not have been invoked: resolution fails before any match is requested.
	handler.mu.Lock()
	called := handler.calledQuery != nil
	handler.mu.Unlock()
	if called {
		t.Error("the Move handler was invoked for an unresolvable destination; want fail-closed before dispatch")
	}
}

// blockingMoveHandler yields one matched instance, signals it has been entered, then blocks on its
// context — the lever the cooperative-shutdown regression uses to park a C-MOVE drain mid-retrieve.
type blockingMoveHandler struct {
	first   *dicom.DataSet
	entered chan struct{}

	mu          sync.Mutex
	ctxCanceled bool
	enteredOnce sync.Once
}

func (h *blockingMoveHandler) Echo(context.Context, OpInfo) Status { return StatusEchoSuccess }
func (h *blockingMoveHandler) Store(context.Context, *dicom.DataSet, OpInfo) Status {
	return StatusStoreSuccess
}
func (h *blockingMoveHandler) Find(context.Context, *dicom.DataSet, QueryLevel, OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return func(func(Status, *dicom.DataSet) bool) {}
}

func (h *blockingMoveHandler) Move(ctx context.Context, _ *dicom.DataSet, _ QueryLevel, _ AETitle, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {
		if !yield(StatusMovePending, h.first) {
			return
		}
		h.enteredOnce.Do(func() { close(h.entered) })
		<-ctx.Done()
		h.mu.Lock()
		h.ctxCanceled = true
		h.mu.Unlock()
	}
}

func (h *blockingMoveHandler) wasCanceled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ctxCanceled
}

// TestServeMoveReturnsOnContextCancelWhileHandlerBlocked is the cooperative-shutdown regression for
// the C-MOVE drain: when the dispatch context is cancelled (Server.Shutdown) while the move handler
// is parked between yields, serveMoveMessage must return promptly with the context error and the
// handler must observe its cancelled context (so its iterator stops and the destination association
// is torn down, no goroutine leak — PRD §9.4). It drives serveMoveMessage directly over a loopback
// acceptor, mirroring the C-FIND cooperative-shutdown test.
func TestServeMoveReturnsOnContextCancelWhileHandlerBlocked(t *testing.T) {
	destTitle, destAddr, _ := startDestinationSCP(t)
	handler := &blockingMoveHandler{first: instanceDataset("1.2.3.1"), entered: make(chan struct{})}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	moveAE, err := NewAE(AETitle("RADX-MOVESCP"))
	if err != nil {
		t.Fatalf("NewAE move SCP: %v", err)
	}
	move := moveSupport{ae: moveAE, destinations: map[AETitle]string{destTitle: destAddr}}

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
			CalledAETitle: "RADX-MOVESCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(studyRootMoveSOPClass),
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
		info := OpInfo{CallingAETitle: AETitle("MOVESCU"), CalledAETitle: AETitle("RADX-MOVESCP")}
		scpDone <- serveMoveMessage(parentCtx, acc, handler, move, cmd, identifier, pcID, info)
	}()

	assoc, ctx, cancel := dialMoveServerSCU(t, ln.Addr().String())
	defer cancel()
	t.Cleanup(func() { _ = assoc.Abort(context.Background()) })

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	firstSeen := make(chan struct{})
	go func() {
		var once sync.Once
		for status := range assoc.Move(ctx, query, QueryLevelStudy, destTitle) {
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
	select {
	case <-handler.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("C-MOVE handler never parked between yields")
	}

	start := time.Now()
	cancelParent()
	select {
	case err := <-scpDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("serveMoveMessage returned %v, want context.Canceled (the prompt ctx-cancel path)", err)
		}
		if elapsed := time.Since(start); elapsed >= time.Second {
			t.Fatalf("serveMoveMessage took %s to return after ctx cancel (>= 1s); the drain wedged", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serveMoveMessage did not return after its context was cancelled; the C-MOVE drain wedged")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !handler.wasCanceled() {
		time.Sleep(10 * time.Millisecond)
	}
	if !handler.wasCanceled() {
		t.Error("the parked C-MOVE handler never observed its cancelled context")
	}
}

// TestServeMoveTerminalCancelOnInboundCancel is the C-CANCEL-during-C-MOVE regression (PS3.4
// C.4.2.3): when the SCU sends a C-CANCEL-RQ mid-retrieve, the SCP's inbound cancel watcher must
// read it, stop sub-operation dispatch (cancelling the parked handler), and answer the terminal
// Cancel status with the counts accumulated so far — a clean cancellation, never a protocol fault
// that poisons the association. It mirrors the C-GET cancel regression
// (TestServeGetTerminalCancelOnInboundCancel) over the three-AE move topology.
func TestServeMoveTerminalCancelOnInboundCancel(t *testing.T) {
	destTitle, destAddr, dest := startDestinationSCP(t)
	handler := &blockingMoveHandler{first: instanceDataset("1.2.3.1"), entered: make(chan struct{})}
	moveAddr := startMoveServer(t, handler, map[AETitle]string{destTitle: destAddr})

	assoc, ctx, cancel := dialMoveServerSCU(t, moveAddr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	// Break out of the SCU range loop after the first Pending response: the iterator's break sends a
	// C-CANCEL-RQ on the same association and drains to the terminal, which the SCP must answer with
	// the Cancel status while its handler is parked between yields.
	for status := range assoc.Move(ctx, query, QueryLevelStudy, destTitle) {
		if status.IsPending() {
			break
		}
	}

	// The SCU's cancel-drain leaves the association clean: no transport/protocol fault is recorded,
	// so the association is reusable rather than poisoned or wedged awaiting a terminal that never
	// comes (pre-fix the SCP only noticed the cancel after the move completed).
	if err := assoc.LastError(); err != nil {
		var pe *ProtocolError
		if errors.As(err, &pe) {
			t.Fatalf("inbound C-CANCEL was treated as a protocol fault, poisoning the association: %v", err)
		}
		t.Fatalf("Move LastError = %v, want nil after a clean cancellation", err)
	}

	// The cancel must stop sub-operation dispatch: the parked handler observes its cancelled context.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !handler.wasCanceled() {
		time.Sleep(10 * time.Millisecond)
	}
	if !handler.wasCanceled() {
		t.Error("the parked C-MOVE handler never observed the C-CANCEL; sub-operation dispatch was not stopped")
	}

	// Only the sub-operation dispatched before the cancel reached the destination.
	instances, _ := dest.snapshot()
	if len(instances) != 1 || instances[0] != "1.2.3.1" {
		t.Errorf("destination received %v, want exactly the pre-cancel instance 1.2.3.1", instances)
	}

	_ = assoc.Release(ctx)
}

// slowDestination is a StoreHandler whose Store parks (up to hold) once entered, signalling entry,
// so the cancel-latency regression can issue a C-CANCEL while a sub-operation C-STORE is in flight
// at the destination.
type slowDestination struct {
	entered     chan struct{}
	enteredOnce sync.Once
	hold        time.Duration
}

func (d *slowDestination) Store(ctx context.Context, _ *dicom.DataSet, _ OpInfo) Status {
	d.enteredOnce.Do(func() { close(d.entered) })
	select {
	case <-ctx.Done():
	case <-time.After(d.hold):
	}
	return StatusStoreSuccess
}

// TestServeMoveCancelAbortsInFlightSubOperation is the cancel-latency regression (PS3.4 C.4.2.2.3
// "as soon as possible"): a C-CANCEL-RQ arriving while a sub-operation C-STORE is in flight at a
// SLOW destination must abort that store promptly (the store rides the watcher-cancelled handler
// context), and the next protocol message the SCU receives must be the terminal Cancel (0xFE00)
// with the accumulated counts — never a Pending after the cancel was issued. The SCU is hand-rolled
// over the association internals so the test observes the raw response sequence and its timing.
func TestServeMoveCancelAbortsInFlightSubOperation(t *testing.T) {
	const destTitle = AETitle("RADX-DEST")
	destAE, err := NewAE(destTitle)
	if err != nil {
		t.Fatalf("NewAE destination: %v", err)
	}
	dest := &slowDestination{entered: make(chan struct{}), hold: 2 * time.Second}
	destSrv := NewServer(destAE, StorageContexts(), dest)
	destServed := make(chan error, 1)
	go func() { destServed <- destSrv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	waitForAddr(t, destSrv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = destSrv.Shutdown(ctx)
		<-destServed
	})

	handler := &movingFindHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"),
	}}
	moveAddr := startMoveServer(t, handler, map[AETitle]string{destTitle: destSrv.Addr().String()})

	assoc, ctx, cancel := dialMoveServerSCU(t, moveAddr)
	defer cancel()
	t.Cleanup(func() { _ = assoc.Abort(context.Background()) })

	pcID, ts, ok := assoc.contextForQuery(studyRootMoveSOPClass)
	if !ok {
		t.Fatal("no accepted Study Root MOVE presentation context")
	}
	conn := assoc.requestor.Conn()
	m := assoc.requestor.Machine()

	identifier := dicom.NewDataSet()
	identifier.SetString(dicom.TagStudyInstanceUID, "1.2.3")
	identifier.SetString(dicom.TagQueryRetrieveLevel, QueryLevelStudy.String())

	const msgID = uint16(7)
	rq := CommandSet{
		CommandField:        CommandCMoveRQ,
		MessageID:           msgID,
		AffectedSOPClassUID: dicom.UID(studyRootMoveSOPClass),
		HasPriority:         true,
		Priority:            PriorityMedium,
		MoveDestination:     destTitle,
		CommandDataSetType:  CommandDataSetPresent,
	}
	if err := sendMessage(ctx, conn, m, pcID, rq, identifier, ts, assoc.sendCap()); err != nil {
		t.Fatalf("send C-MOVE-RQ: %v", err)
	}

	// Wait until the first sub-operation C-STORE is in flight (parked) at the destination, THEN
	// cancel. No Pending has been sent yet: the SCP reports a Pending only after each store ends.
	select {
	case <-dest.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the destination never entered the sub-operation store; cannot park the retrieve")
	}

	cancelRQ := CommandSet{
		CommandField:              CommandCCancelRQ,
		MessageIDBeingRespondedTo: msgID,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	issued := time.Now()
	if err := sendCommand(ctx, conn, m, pcID, cancelRQ); err != nil {
		t.Fatalf("send C-CANCEL-RQ: %v", err)
	}

	var statuses []Status
	var terminal CommandSet
	for {
		rsp, _, _, rerr := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
		if rerr != nil {
			t.Fatalf("read C-MOVE-RSP after the cancel: %v", rerr)
		}
		if rsp.CommandField != CommandCMoveRSP || !rsp.HasStatus {
			t.Fatalf("unexpected response after the cancel: command %#04x", uint16(rsp.CommandField))
		}
		st := NewStatus(rsp.Status, ServiceClassMove)
		statuses = append(statuses, st)
		if !st.IsPending() {
			terminal = rsp
			break
		}
	}
	elapsed := time.Since(issued)

	for _, st := range statuses {
		if st.IsPending() {
			t.Errorf("a Pending C-MOVE-RSP (%s) arrived after the C-CANCEL was issued; the next message must be the terminal Cancel", st)
		}
	}
	if got := statuses[len(statuses)-1]; got.Code != StatusMoveCancel.Code {
		t.Errorf("terminal status = %s, want 0xFE00 Cancel", got)
	}
	// The in-flight store must be ABORTED, not awaited: the terminal Cancel arrives well before the
	// destination's 2s hold elapses.
	if elapsed >= 1500*time.Millisecond {
		t.Errorf("terminal Cancel took %s after the C-CANCEL; the in-flight sub-operation store was awaited, not aborted", elapsed)
	}
	// The interrupted store is neither completed nor a destination failure: the accumulated counts
	// the Cancel carries report only sub-operations that actually finished.
	if terminal.CompletedSubOperations != 0 || terminal.FailedSubOperations != 0 || terminal.WarningSubOperations != 0 {
		t.Errorf("Cancel counts = completed %d / failed %d / warning %d, want all zero (the interrupted store is not counted)",
			terminal.CompletedSubOperations, terminal.FailedSubOperations, terminal.WarningSubOperations)
	}

	_ = assoc.Release(ctx)
}

// TestServeMoveUnsupported is the interface-segregation regression: a C-MOVE-RQ routed to a handler
// with no Move capability (a store-only handler) is refused with a terminal C-MOVE-RSP carrying
// StatusSOPClassNotSupported, never a panic.
func TestServeMoveUnsupported(t *testing.T) {
	moveAddr := startMoveServer(t, &storeOnlyHandler{status: StatusStoreSuccess}, map[AETitle]string{})

	assoc, ctx, cancel := dialMoveServerSCU(t, moveAddr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var statuses []Status
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("DEST-AE")) {
		statuses = append(statuses, status)
	}
	_ = assoc.Release(ctx)

	if len(statuses) != 1 {
		t.Fatalf("C-MOVE to a store-only handler yielded %d statuses, want 1 terminal refusal", len(statuses))
	}
	if statuses[0].Code != StatusSOPClassNotSupported.Code {
		t.Errorf("terminal status = %s, want 0x%04X (Refused: SOP Class Not Supported)",
			statuses[0], StatusSOPClassNotSupported.Code)
	}
	if err := assoc.LastError(); err != nil {
		t.Errorf("LastError = %v, want nil (a graceful refusal is data, not a transport fault)", err)
	}
}
