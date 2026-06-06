package dimse

import (
	"context"
	"errors"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// TestValidateGetContext is the regression for the C-GET negotiation guard (the symmetry with
// validateMoveContext): a C-GET must arrive on a context whose negotiated abstract syntax is a
// Query/Retrieve GET information model AND whose Affected SOP Class matches that model, else it is a
// protocol fault — a peer cannot run a retrieve outside the negotiated/declared SOP Class.
func TestValidateGetContext(t *testing.T) {
	abstractFor := func(pcID uint8) (dicom.SOPClassUID, bool) {
		switch pcID {
		case 1:
			return studyRootGetSOPClass, true // a GET context
		case 3:
			return studyRootMoveSOPClass, true // a MOVE context (not GET)
		default:
			return "", false
		}
	}

	match := CommandSet{CommandField: CommandCGetRQ, AffectedSOPClassUID: dicom.UID(studyRootGetSOPClass)}
	if err := validateGetContext(match, 1, abstractFor, Sta6); err != nil {
		t.Errorf("matching GET context rejected: %v", err)
	}

	onMove := CommandSet{CommandField: CommandCGetRQ, AffectedSOPClassUID: dicom.UID(studyRootMoveSOPClass)}
	if err := validateGetContext(onMove, 3, abstractFor, Sta6); err == nil {
		t.Error("C-GET on a non-GET (MOVE) context = nil error, want a protocol fault")
	} else {
		var pe *ProtocolError
		if !errors.As(err, &pe) {
			t.Errorf("error = %T, want *ProtocolError", err)
		}
	}

	mismatch := CommandSet{CommandField: CommandCGetRQ, AffectedSOPClassUID: dicom.UID(patientRootGetSOPClass)}
	if err := validateGetContext(mismatch, 1, abstractFor, Sta6); err == nil {
		t.Error("mismatched GET SOP class on a GET context = nil error, want a protocol fault")
	}

	if err := validateGetContext(match, 9, abstractFor, Sta6); err == nil {
		t.Error("unknown presentation context = nil error, want a protocol fault")
	}
}

// TestGetTerminalStatus pins the C-GET terminal-status resolver: a clean Success only when nothing
// failed or warned, the all-failed 0xA702, and the 0xB000 partial-failure Warning otherwise. It is
// the unit guard for the highest-consequence rule — a partial failure must never read as Success.
func TestGetTerminalStatus(t *testing.T) {
	tests := []struct {
		name   string
		counts SubOperationCounts
		want   StatusCategory
		code   uint16
	}{
		{"all completed", SubOperationCounts{Completed: 3}, StatusCategorySuccess, 0x0000},
		{"one failed", SubOperationCounts{Completed: 2, Failed: 1}, StatusCategoryWarning, 0xB000},
		{"one warned", SubOperationCounts{Completed: 2, Warning: 1}, StatusCategoryWarning, 0xB000},
		{"all failed", SubOperationCounts{Failed: 2}, StatusCategoryFailure, 0xA702},
		{"failed and warned, not all failed", SubOperationCounts{Failed: 1, Warning: 1}, StatusCategoryWarning, 0xB000},
		{"nothing attempted", SubOperationCounts{}, StatusCategorySuccess, 0x0000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTerminalStatus(tt.counts)
			if got.Category() != tt.want {
				t.Errorf("category = %s, want %s", got.Category(), tt.want)
			}
			if got.Code != tt.code {
				t.Errorf("code = %#04x, want %#04x", got.Code, tt.code)
			}
			if tt.want == StatusCategorySuccess && !got.IsSuccess() {
				t.Error("a clean retrieve did not read as Success")
			}
			if (tt.counts.Failed > 0 || tt.counts.Warning > 0) && got.IsSuccess() {
				t.Error("a partial-failure retrieve was laundered into a Success status (PRD §9.2)")
			}
		})
	}
}

// gettingHandler yields a fixed list of matched instance datasets (each as a Pending with the
// instance dataset) then a terminal Success, so a C-GET drain test can assert the runtime C-STOREs
// each matched instance back to the requestor. It implements GetHandler alone (interface segregation).
type gettingHandler struct {
	instances []*dicom.DataSet

	mu          sync.Mutex
	calledQuery *dicom.DataSet
}

func (h *gettingHandler) Get(_ context.Context, query *dicom.DataSet, _ QueryLevel, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	h.mu.Lock()
	h.calledQuery = query
	h.mu.Unlock()
	return func(yield func(Status, *dicom.DataSet) bool) {
		for _, ds := range h.instances {
			if !yield(StatusGetPending, ds) {
				return
			}
		}
		yield(StatusGetSuccess, nil)
	}
}

// recordingGetSink is the StoreHandler sink the C-GET SCU passes to receive the sub-operation
// instances. It records each SOP Instance UID, the sub-operation Message ID, and the Move Originator
// AE Title, and can be configured to fail or warn on one named SOP Instance UID so the partial-failure
// path can be exercised.
type recordingGetSink struct {
	failUID string
	warnUID string

	mu          sync.Mutex
	instances   []string
	msgIDs      []uint16
	originators []AETitle
}

func (d *recordingGetSink) Store(_ context.Context, ds *dicom.DataSet, info OpInfo) Status {
	sopInstance, _ := ds.GetString(tagSOPInstanceUID)
	d.mu.Lock()
	d.instances = append(d.instances, sopInstance)
	d.msgIDs = append(d.msgIDs, info.MessageID)
	d.originators = append(d.originators, info.MoveOriginatorAETitle)
	d.mu.Unlock()
	switch sopInstance {
	case d.failUID:
		return StatusStoreCannotUnderstand // 0xC000 Failure
	case d.warnUID:
		return StatusStoreCoercionOfDataElements // 0xB000 Warning
	}
	return StatusStoreSuccess
}

func (d *recordingGetSink) snapshot() ([]string, []uint16, []AETitle) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.instances...),
		append([]uint16(nil), d.msgIDs...),
		append([]AETitle(nil), d.originators...)
}

// getStorageSOPClass is the Storage SOP Class the C-GET sub-operations carry (MR Image Storage, in the
// validated Storage set). instanceDataset (move_scp_test.go) builds instances with this class.
const getStorageSOPClass = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.4")

// startGetServer stands up a go-radx C-GET SCP hosting the handler, advertising the Query/Retrieve and
// Storage contexts and granting the Storage SCP role for the C-GET sub-operation SOP Class, on
// loopback.
func startGetServer(t *testing.T, h any) string {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-GETSCP"))
	if err != nil {
		t.Fatalf("NewAE get SCP: %v", err)
	}
	srv := NewServer(ae, QueryRetrieveWithStorageContexts(), h, WithGetStorageRoles(validatedStorageSOPClasses...))
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

// dialGetServerSCU associates to a go-radx C-GET SCP proposing the retrieve contexts and the Storage
// SCP role selection so the same-association sub-operation C-STOREs are granted.
func dialGetServerSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("GETSCU"), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("RADX-GETSCP"), QueryRetrieveWithStorageContexts(),
		WithRoleSelection(RoleSelection{SOPClassUID: getStorageSOPClass, SCURole: true, SCPRole: true}))
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	// The acceptor must have granted the requestor the Storage SCP role; without it the requestor
	// cannot receive the same-association sub-operation C-STOREs.
	granted := false
	for _, r := range assoc.NegotiatedRoles() {
		if r.SOPClassUID == getStorageSOPClass && r.SCPRole {
			granted = true
		}
	}
	if !granted {
		cancel()
		t.Fatalf("acceptor did not grant the Storage SCP role for %s; same-association C-GET cannot proceed", getStorageSOPClass)
	}
	return assoc, ctx, cancel
}

// TestServerAnswersCGet is the in-process C-GET round-trip: a go-radx C-GET SCU drives a C-GET against
// a go-radx C-GET SCP, which C-STOREs the matched instances back to the SCU on the SAME association.
// The SCU must surface a terminal Success and its sink must receive both instances with the C-GET
// requestor as the Move Originator.
func TestServerAnswersCGet(t *testing.T) {
	handler := &gettingHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"),
	}}
	getAddr := startGetServer(t, handler)

	assoc, ctx, cancel := dialGetServerSCU(t, getAddr)
	defer cancel()

	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var terminal Status
	var terminalSeen bool
	for status := range assoc.Get(ctx, query, QueryLevelStudy, sink) {
		if !status.IsPending() {
			terminal = status
			terminalSeen = true
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if !terminalSeen || !terminal.IsSuccess() {
		t.Errorf("terminal status = %s (seen=%v), want Success", terminal, terminalSeen)
	}

	instances, _, originators := sink.snapshot()
	if len(instances) != 2 {
		t.Fatalf("sink received %v, want 2 instances", instances)
	}
	want := map[string]bool{"1.2.3.1": true, "1.2.3.2": true}
	for _, got := range instances {
		if !want[got] {
			t.Errorf("sink received unexpected instance %q", got)
		}
	}

	final := assoc.SubOperationCounts()
	if final.Completed != 2 {
		t.Errorf("final Completed count = %d, want 2", final.Completed)
	}
	if final.Failed != 0 {
		t.Errorf("final Failed count = %d, want 0", final.Failed)
	}

	// Each sub-operation C-STORE must carry the Move Originator AE Title of the AE that INVOKED the
	// C-GET (the calling SCU "GETSCU"), not the C-GET SCP's own title (PS3.7 §9.1.1).
	for _, orig := range originators {
		if orig != AETitle("GETSCU") {
			t.Errorf("sub-operation Move Originator AE Title = %q, want the C-GET requestor GETSCU", orig)
		}
	}
}

// TestGetSubOperationsUseDistinctMessageIDs is the DIMSE-016 regression for C-GET: the SCP's
// sub-operation C-STOREs back to the requestor must each carry a distinct, non-zero Message ID.
func TestGetSubOperationsUseDistinctMessageIDs(t *testing.T) {
	handler := &gettingHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"),
		instanceDataset("1.2.3.3"),
	}}
	getAddr := startGetServer(t, handler)

	assoc, ctx, cancel := dialGetServerSCU(t, getAddr)
	defer cancel()

	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	for range assoc.Get(ctx, query, QueryLevelStudy, sink) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	instances, msgIDs, _ := sink.snapshot()
	if len(instances) != 3 {
		t.Fatalf("sink received %d instances, want 3", len(instances))
	}
	seen := make(map[uint16]bool)
	for i, id := range msgIDs {
		if id == 0 {
			t.Errorf("sub-operation %d used Message ID 0 (DIMSE-016: sub-operations need a distinct non-zero ID)", i)
		}
		if seen[id] {
			t.Errorf("Message ID %d reused across sub-operations (DIMSE-016)", id)
		}
		seen[id] = true
	}
}

// TestServeGetTerminalWarningOnSubOpRejection is the highest-consequence C-GET correctness property:
// when the requestor REJECTS one sub-operation (its sink returns a Failure C-STORE-RSP status), the
// SCP must report the terminal 0xB000 "Sub-operations Complete — One or More Failures or Warnings"
// Warning, NOT Success, and the failed count must reach the SCU. A partial failure must never be
// laundered into success (PRD §9.2).
func TestServeGetTerminalWarningOnSubOpRejection(t *testing.T) {
	handler := &gettingHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"), // the SCU's sink rejects this one
		instanceDataset("1.2.3.3"),
	}}
	getAddr := startGetServer(t, handler)

	assoc, ctx, cancel := dialGetServerSCU(t, getAddr)
	defer cancel()

	sink := &recordingGetSink{failUID: "1.2.3.2"}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var terminal Status
	var sawSuccess bool
	for status := range assoc.Get(ctx, query, QueryLevelStudy, sink) {
		if status.IsSuccess() {
			sawSuccess = true
		}
		if !status.IsPending() {
			terminal = status
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if sawSuccess {
		t.Error("a partial-failure C-GET was laundered into a Success status (PRD §9.2)")
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

// TestServeGetTerminalWarningOnSubOpWarning verifies that when a sub-operation C-STORE-RSP is a Warning
// (stored with a warning) and none fails, the SCP reports the terminal 0xB000 Warning, NOT Success — a
// warning is not laundered into Success (PRD §9.2).
func TestServeGetTerminalWarningOnSubOpWarning(t *testing.T) {
	handler := &gettingHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"),
	}}
	getAddr := startGetServer(t, handler)

	assoc, ctx, cancel := dialGetServerSCU(t, getAddr)
	defer cancel()

	sink := &recordingGetSink{warnUID: "1.2.3.1"}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var terminal Status
	var sawSuccess bool
	for status := range assoc.Get(ctx, query, QueryLevelStudy, sink) {
		if status.IsSuccess() {
			sawSuccess = true
		}
		if !status.IsPending() {
			terminal = status
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if sawSuccess {
		t.Error("a warning C-GET was laundered into a Success status (PRD §9.2)")
	}
	if !terminal.IsWarning() {
		t.Errorf("terminal status = %s, want a Warning (0xB000 sub-operations complete with warnings)", terminal)
	}
	final := assoc.SubOperationCounts()
	if final.Warning != 1 {
		t.Errorf("final Warning count = %d, want 1", final.Warning)
	}
	if final.Completed != 1 {
		t.Errorf("final Completed count = %d, want 1", final.Completed)
	}
}

// TestServeGetUnsupported is the interface-segregation regression: a C-GET-RQ routed to a handler with
// no Get capability (a store-only handler) is refused with a terminal C-GET-RSP carrying
// StatusSOPClassNotSupported, never a panic.
func TestServeGetUnsupported(t *testing.T) {
	getAddr := startGetServer(t, &storeOnlyHandler{status: StatusStoreSuccess})

	assoc, ctx, cancel := dialGetServerSCU(t, getAddr)
	defer cancel()

	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var statuses []Status
	for status := range assoc.Get(ctx, query, QueryLevelStudy, sink) {
		statuses = append(statuses, status)
	}
	_ = assoc.Release(ctx)

	if len(statuses) != 1 {
		t.Fatalf("C-GET to a store-only handler yielded %d statuses, want 1 terminal refusal", len(statuses))
	}
	if statuses[0].Code != StatusSOPClassNotSupported.Code {
		t.Errorf("terminal status = %s, want 0x%04X (Refused: SOP Class Not Supported)",
			statuses[0], StatusSOPClassNotSupported.Code)
	}
	if err := assoc.LastError(); err != nil {
		t.Errorf("LastError = %v, want nil (a graceful refusal is data, not a transport fault)", err)
	}
}

// ctStorageSOPClass is a Storage SOP Class in the validated set (CT Image Storage) distinct from
// getStorageSOPClass. The role-gating test grants the requestor the Storage SCP role for MR but NOT
// for CT, so a CT sub-operation has an accepted context yet no granted SCP role.
const ctStorageSOPClass = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2")

// instanceDatasetWithClass builds a storable dataset for an explicit Storage SOP Class, so a test can
// produce a sub-operation instance whose class the SCP did or did not grant the Storage SCP role for.
func instanceDatasetWithClass(sopClass dicom.SOPClassUID, sopInstanceUID string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(tagSOPClassUID, string(sopClass))
	ds.SetString(tagSOPInstanceUID, sopInstanceUID)
	return ds
}

// startGetServerWithRoles stands up a C-GET SCP hosting the handler, advertising the Query/Retrieve
// and Storage contexts but granting the Storage SCP role ONLY for the named SOP Classes, so a test can
// reproduce a same-association C-GET where one matched instance's context lacks the granted SCP role.
func startGetServerWithRoles(t *testing.T, h any, grantRolesFor ...dicom.SOPClassUID) string {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-GETSCP"))
	if err != nil {
		t.Fatalf("NewAE get SCP: %v", err)
	}
	srv := NewServer(ae, QueryRetrieveWithStorageContexts(), h, WithGetStorageRoles(grantRolesFor...))
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

// TestGetPreflightFailsClosedWithoutStorageSCPRole is the FIX 2 (SCU fail-closed) regression: a C-GET
// must not reach the wire when the association negotiated no Storage SCP role. The requestor cannot act
// as the Storage SCP to receive the sub-operation C-STOREs, so a real peer would reject or abort; the
// preflight must instead yield a single terminal Failure and set LastError to a typed ValidationError
// BEFORE any wire I/O, never sending a C-GET-RQ that cannot complete (PRD §9.2 fail-closed).
func TestGetPreflightFailsClosedWithoutStorageSCPRole(t *testing.T) {
	getAddr := startGetServer(t, &gettingHandler{instances: []*dicom.DataSet{instanceDataset("1.2.3.1")}})

	// Associate proposing the retrieve+Storage contexts but NO role selection, so the acceptor grants
	// no Storage SCP role: the requestor cannot be the Storage SCP for the sub-operations.
	ae, err := NewAE(AETitle("GETSCU"), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, getAddr, AETitle("RADX-GETSCP"), QueryRetrieveWithStorageContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	if assoc.NegotiatedRoles() != nil {
		t.Fatalf("expected no negotiated roles without WithRoleSelection, got %v", assoc.NegotiatedRoles())
	}

	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var statuses []Status
	for status := range assoc.Get(ctx, query, QueryLevelStudy, sink) {
		statuses = append(statuses, status)
	}

	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("preflight without a granted Storage SCP role yielded %v, want one terminal Failure", statuses)
	}
	var ve *ValidationError
	if err := assoc.LastError(); !errors.As(err, &ve) {
		t.Fatalf("LastError = %v (%T), want a *ValidationError for the missing Storage SCP role", err, err)
	}
	// Nothing was stored: the C-GET never reached the wire (no sub-operations were received).
	if instances, _, _ := sink.snapshot(); len(instances) != 0 {
		t.Errorf("sink received %v, want none (the C-GET must not have been sent)", instances)
	}
}

// TestServeGetSubOpFailsWithoutGrantedStorageSCPRole is the FIX 3 (SCP role-gating) regression: when a
// matched instance's Storage SOP Class has an accepted context but role selection did NOT grant the
// requestor the Storage SCP role for it, the SCP must NOT send a C-STORE-RQ in the wrong (default SCU)
// role. It counts that sub-operation as FAILED so the terminal status reflects failure, never laundered
// into Success (PRD §9.2). Here the SCP grants the SCP role for MR Image Storage only, the SCU proposes
// it (so preflight passes), but the single matched instance is CT Image Storage: an accepted context
// with no granted SCP role.
func TestServeGetSubOpFailsWithoutGrantedStorageSCPRole(t *testing.T) {
	handler := &gettingHandler{instances: []*dicom.DataSet{
		instanceDatasetWithClass(ctStorageSOPClass, "1.2.3.1"),
	}}
	getAddr := startGetServerWithRoles(t, handler, getStorageSOPClass) // grant SCP role for MR only

	assoc, ctx, cancel := dialGetServerSCU(t, getAddr) // proposes the MR Storage SCP role
	defer cancel()

	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var terminal Status
	var sawSuccess bool
	for status := range assoc.Get(ctx, query, QueryLevelStudy, sink) {
		if status.IsSuccess() {
			sawSuccess = true
		}
		if !status.IsPending() {
			terminal = status
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get LastError = %v, want nil (an ungranted-role sub-operation is in-band failure, not a fault)", err)
	}
	_ = assoc.Release(ctx)

	if sawSuccess {
		t.Error("a sub-operation sent in an ungranted role was laundered into a Success status (PRD §9.2)")
	}
	if terminal.IsSuccess() || terminal.IsPending() {
		t.Errorf("terminal status = %s, want a Failure/Warning (the only sub-operation could not be sent in a granted role)", terminal)
	}
	final := assoc.SubOperationCounts()
	if final.Completed != 0 {
		t.Errorf("final Completed count = %d, want 0 (nothing was storable in a granted role)", final.Completed)
	}
	if final.Failed != 1 {
		t.Errorf("final Failed count = %d, want 1 (the ungranted-role instance is a failed sub-operation)", final.Failed)
	}
	// The instance must NOT have been stored: the SCP must not transmit a C-STORE-RQ in the wrong role.
	if instances, _, _ := sink.snapshot(); len(instances) != 0 {
		t.Errorf("sink received %v, want none (no C-STORE may be sent in an ungranted role)", instances)
	}
}

// TestServeGetTerminalCancelOnInboundCancel is the FIX 4 (graceful C-CANCEL) regression: when the SCU
// sends a C-CANCEL-GET-RQ mid-retrieve in place of a C-STORE-RSP, the SCP must recognize it as a clean
// cancellation — stop the sub-operation loop and report the terminal Cancel status (0xFE00) with the
// accumulated counts — rather than treating the unexpected command as a protocol fault that poisons
// the association.
func TestServeGetTerminalCancelOnInboundCancel(t *testing.T) {
	handler := &gettingHandler{instances: []*dicom.DataSet{
		instanceDataset("1.2.3.1"),
		instanceDataset("1.2.3.2"),
		instanceDataset("1.2.3.3"),
	}}
	getAddr := startGetServer(t, handler)

	assoc, ctx, cancel := dialGetServerSCU(t, getAddr)
	defer cancel()

	// Break out of the SCU range loop after the first Pending response: the iterator's break sends a
	// C-CANCEL-RQ on the same association, which the SCP must read (in place of the next C-STORE-RSP)
	// and treat as a clean cancellation.
	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var seen int
	for range assoc.Get(ctx, query, QueryLevelStudy, sink) {
		seen++
		if seen == 1 {
			break
		}
	}

	// The SCU's cancel-drain leaves the association clean: no transport/protocol fault is recorded, so
	// the association is reusable rather than poisoned by a spurious ProtocolError on the SCP side.
	if err := assoc.LastError(); err != nil {
		var pe *ProtocolError
		if errors.As(err, &pe) {
			t.Fatalf("inbound C-CANCEL was treated as a protocol fault, poisoning the association: %v", err)
		}
		t.Fatalf("Get LastError = %v, want nil after a clean cancellation", err)
	}
	_ = assoc.Release(ctx)
}

// TestServeGetZeroMatchesSuccess verifies a C-GET whose handler yields no match still terminates with a
// single Success RSP (the no-hang contract), with all-zero sub-operation counts.
func TestServeGetZeroMatchesSuccess(t *testing.T) {
	getAddr := startGetServer(t, &gettingHandler{})

	assoc, ctx, cancel := dialGetServerSCU(t, getAddr)
	defer cancel()

	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	var statuses []Status
	for status := range assoc.Get(ctx, query, QueryLevelStudy, sink) {
		statuses = append(statuses, status)
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if len(statuses) != 1 || !statuses[0].IsSuccess() {
		t.Fatalf("zero-match C-GET yielded %v, want one terminal Success", statuses)
	}
	instances, _, _ := sink.snapshot()
	if len(instances) != 0 {
		t.Errorf("sink received %v, want none", instances)
	}
}
