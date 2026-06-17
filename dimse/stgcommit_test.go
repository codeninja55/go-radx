package dimse

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// stgCommitObservation records what the mock Storage Commitment SCP saw on the N-ACTION request
// association, for the unit tests to assert against without inspecting the wire directly.
type stgCommitObservation struct {
	mu sync.Mutex

	actionCmd CommandSet
	actionDS  *dicom.DataSet
	err       error
}

func (o *stgCommitObservation) snapshot() stgCommitObservation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return stgCommitObservation{actionCmd: o.actionCmd, actionDS: o.actionDS, err: o.err}
}

// startStgCommitSCP listens on loopback, accepts a Storage Commitment association, then serves one
// N-ACTION-RQ, recording the command and its action information data set. It replies with the
// N-ACTION-RSP carrying actionStatus. It models a dcm4chee Storage Commitment SCP that accepts the
// request synchronously and would report the result later on a separate association.
func startStgCommitSCP(t *testing.T, actionStatus uint16) (string, *stgCommitObservation) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	obs := &stgCommitObservation{}
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: "STGCMTSCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(storageCommitmentPushModelSOPClass),
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			_ = nc.Close()
			obs.mu.Lock()
			obs.err = perr
			obs.mu.Unlock()
			return
		}
		serveCannedStgCommitAction(ctx, acc, actionStatus, obs)
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String(), obs
}

// serveCannedStgCommitAction reads the N-ACTION-RQ and its action-information data set, then replies
// with an N-ACTION-RSP carrying actionStatus and no data set (the result follows asynchronously on a
// separate association in the push model).
func serveCannedStgCommitAction(ctx context.Context, acc *acse.Acceptor, actionStatus uint16, obs *stgCommitObservation) {
	m := acc.Machine()
	conn := acc.Conn()

	var pcID uint8
	ts := dicom.ImplicitVRLittleEndian
	for _, pc := range acc.AcceptedContexts() {
		if pc.Result == 0 {
			pcID = pc.ID
			ts = dicom.TransferSyntax(pc.TransferSyntax)
			break
		}
	}

	actionCmd, actionDS, _, rerr := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if rerr != nil {
		obs.mu.Lock()
		obs.err = rerr
		obs.mu.Unlock()
		return
	}
	obs.mu.Lock()
	obs.actionCmd = actionCmd
	obs.actionDS = actionDS
	obs.mu.Unlock()

	rsp := CommandSet{
		CommandField:              CommandNActionRSP,
		MessageIDBeingRespondedTo: actionCmd.MessageID,
		AffectedSOPClassUID:       actionCmd.RequestedSOPClassUID,
		AffectedSOPInstanceUID:    actionCmd.RequestedSOPInstanceUID,
		HasActionTypeID:           true,
		ActionTypeID:              actionCmd.ActionTypeID,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    actionStatus,
	}
	if serr := sendMessage(ctx, conn, m, pcID, rsp, nil, ts, MaxPDULength(16382)); serr != nil {
		obs.mu.Lock()
		obs.err = serr
		obs.mu.Unlock()
	}
}

// dialStgCommitSCU opens a Storage Commitment association to the mock SCP, proposing the Storage
// Commitment Push Model context and the SCP role for it so the role-reversed same-association
// N-EVENT-REPORT (the synchronous reporting model) is negotiated (PS3.7 D.3.3.4).
func dialStgCommitSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("STGCMTSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("STGCMTSCP"), StorageCommitmentContexts(),
		WithRoleSelection(RoleSelection{SOPClassUID: storageCommitmentPushModelSOPClass, SCURole: true, SCPRole: true}))
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// committedRefs is a small synthetic set of SOP instances to request commitment for. These UIDs are
// synthetic fixtures, never real patient data.
func committedRefs() []dicom.ReferencedSOPInstance {
	return []dicom.ReferencedSOPInstance{
		{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SOPInstanceUID: "1.2.826.0.1.3680043.8.498.40000001"},
		{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SOPInstanceUID: "1.2.826.0.1.3680043.8.498.40000002"},
	}
}

// TestStorageCommitmentRequestSendsNAction is the request-side gate: Request issues an N-ACTION-RQ
// against the Storage Commitment Push Model well-known SOP Instance with Action Type ID 1, carrying
// the Transaction UID and the Referenced SOP Sequence of the instances whose commitment is sought,
// and surfaces a typed Success status.
func TestStorageCommitmentRequestSendsNAction(t *testing.T) {
	addr, obs := startStgCommitSCP(t, StatusStorageCommitmentSuccess.Code)
	assoc, ctx, cancel := dialStgCommitSCU(t, addr)
	defer cancel()

	const transactionUID = "1.2.826.0.1.3680043.8.498.49000099"
	sc := assoc.StorageCommitment()

	status, err := sc.Request(ctx, transactionUID, committedRefs())
	if err != nil {
		t.Fatalf("Storage Commitment Request: %v", err)
	}
	if status.ServiceClass() != ServiceClassStorageCommitment {
		t.Errorf("Request status service class = %v, want StorageCommitment", status.ServiceClass())
	}
	if !status.IsSuccess() {
		t.Errorf("Request status = %s, want Success", status)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.err != nil {
		t.Fatalf("SCP error: %v", got.err)
	}
	if got.actionCmd.CommandField != CommandNActionRQ {
		t.Errorf("command field = %#04x, want N-ACTION-RQ", uint16(got.actionCmd.CommandField))
	}
	// N-ACTION references the well-known Storage Commitment Push Model SOP Instance by Requested SOP
	// Class/Instance UID, distinct from the Affected pair a C-service or N-CREATE carries.
	if got.actionCmd.RequestedSOPClassUID != dicom.UID(storageCommitmentPushModelSOPClass) {
		t.Errorf("Requested SOP Class UID = %q, want Storage Commitment Push Model", got.actionCmd.RequestedSOPClassUID)
	}
	if got.actionCmd.RequestedSOPInstanceUID != dicom.UID(storageCommitmentPushModelInstance) {
		t.Errorf("Requested SOP Instance UID = %q, want the well-known instance", got.actionCmd.RequestedSOPInstanceUID)
	}
	if got.actionCmd.AffectedSOPInstanceUID != "" {
		t.Errorf("N-ACTION should not carry an Affected SOP Instance UID, got %q", got.actionCmd.AffectedSOPInstanceUID)
	}
	if !got.actionCmd.HasActionTypeID || got.actionCmd.ActionTypeID != storageCommitmentActionType {
		t.Errorf("Action Type ID = %d (present=%v), want %d", got.actionCmd.ActionTypeID, got.actionCmd.HasActionTypeID, storageCommitmentActionType)
	}
	if got.actionDS == nil {
		t.Fatal("N-ACTION carried no action-information data set")
	}
	if tu, ok := got.actionDS.GetString(dicom.TagTransactionUID); !ok || tu != transactionUID {
		t.Errorf("Transaction UID in action info = %q, want %q", tu, transactionUID)
	}
	seq, ok := got.actionDS.GetSequence(dicom.TagReferencedSOPSequence)
	if !ok || seq.Len() != 2 {
		t.Fatalf("Referenced SOP Sequence len = %d (present=%v), want 2", seqLen(seq), ok)
	}
}

func seqLen(s *dicom.Sequence) int {
	if s == nil {
		return 0
	}
	return s.Len()
}

// TestStorageCommitmentRequestRejectsEmptyInputs confirms Request fails closed before any wire I/O
// on a missing Transaction UID or an empty reference set (PRD §9.2 fail-closed).
func TestStorageCommitmentRequestRejectsEmptyInputs(t *testing.T) {
	addr, _ := startStgCommitSCP(t, StatusStorageCommitmentSuccess.Code)
	assoc, ctx, cancel := dialStgCommitSCU(t, addr)
	defer cancel()

	if _, err := assoc.StorageCommitment().Request(ctx, "", committedRefs()); err == nil {
		t.Error("Request should reject an empty Transaction UID")
	}
	if _, err := assoc.StorageCommitment().Request(ctx, "1.2.3", nil); err == nil {
		t.Error("Request should reject an empty reference set")
	}
	_ = assoc.Release(ctx)
}

// TestStorageCommitmentOnUnestablishedAssociation confirms the SCU fails closed with a typed error
// rather than panicking on a nil/unestablished association.
func TestStorageCommitmentOnUnestablishedAssociation(t *testing.T) {
	var a *Association
	if _, err := a.StorageCommitment().Request(context.Background(), "1.2.3", committedRefs()); err == nil {
		t.Error("Request on a nil association should return a typed error, not panic")
	}
}

// buildCommitmentResult encodes an N-EVENT-REPORT action-information data set the way a Storage
// Commitment SCP reports a result (PS3.4 J.3.3): the Transaction UID, the Referenced SOP Sequence of
// committed instances, and (when any failed) the Failed SOP Sequence with each instance's Failure
// Reason (0008,1198). The UIDs are synthetic fixtures.
func buildCommitmentResult(transactionUID string, committed []dicom.ReferencedSOPInstance, failed []dicom.FailedSOPInstance) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagTransactionUID, transactionUID)
	if len(committed) > 0 {
		items := make([]*dicom.DataSet, 0, len(committed))
		for _, ref := range committed {
			item := dicom.NewDataSet()
			item.SetString(dicom.TagReferencedSOPClassUID, string(ref.SOPClassUID))
			item.SetString(dicom.TagReferencedSOPInstanceUID, string(ref.SOPInstanceUID))
			items = append(items, item)
		}
		ds.Set(dicom.Element{Tag: dicom.TagReferencedSOPSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence(items...))})
	}
	if len(failed) > 0 {
		items := make([]*dicom.DataSet, 0, len(failed))
		for _, f := range failed {
			item := dicom.NewDataSet()
			item.SetString(dicom.TagReferencedSOPClassUID, string(f.SOPClassUID))
			item.SetString(dicom.TagReferencedSOPInstanceUID, string(f.SOPInstanceUID))
			item.Set(dicom.Element{Tag: dicom.TagFailureReason, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, int64(f.FailureReason))})
			items = append(items, item)
		}
		ds.Set(dicom.Element{Tag: dicom.TagFailedSOPSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence(items...))})
	}
	return ds
}

// TestParseCommitmentResultAllCommitted parses an N-EVENT-REPORT result that committed every
// instance (Event Type ID 1) and confirms it reports success with no failures.
func TestParseCommitmentResultAllCommitted(t *testing.T) {
	const transactionUID = "1.2.826.0.1.3680043.8.498.49000100"
	committed := committedRefs()
	ds := buildCommitmentResult(transactionUID, committed, nil)

	result, err := parseCommitmentResult(StorageCommitmentEventComplete, ds)
	if err != nil {
		t.Fatalf("parseCommitmentResult: %v", err)
	}
	if result.TransactionUID != transactionUID {
		t.Errorf("TransactionUID = %q, want %q", result.TransactionUID, transactionUID)
	}
	if len(result.Committed) != len(committed) {
		t.Errorf("Committed count = %d, want %d", len(result.Committed), len(committed))
	}
	if len(result.Failed) != 0 {
		t.Errorf("Failed count = %d, want 0", len(result.Failed))
	}
	if result.HasFailures() {
		t.Error("HasFailures should be false when no instance failed commitment")
	}
	if result.EventType != StorageCommitmentEventComplete {
		t.Errorf("EventType = %v, want StorageCommitmentEventComplete", result.EventType)
	}
}

// TestParseCommitmentResultPartialFailure is the partial-failure honesty gate: an N-EVENT-REPORT
// result that failed one instance (Event Type ID 2) MUST surface the failure, never launder it into
// success. The failed instance carries its Failure Reason (0008,1198).
func TestParseCommitmentResultPartialFailure(t *testing.T) {
	const transactionUID = "1.2.826.0.1.3680043.8.498.49000200"
	committed := []dicom.ReferencedSOPInstance{
		{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SOPInstanceUID: "1.2.826.0.1.3680043.8.498.40000001"},
	}
	failed := []dicom.FailedSOPInstance{
		{
			ReferencedSOPInstance: dicom.ReferencedSOPInstance{
				SOPClassUID:    "1.2.840.10008.5.1.4.1.1.2",
				SOPInstanceUID: "1.2.826.0.1.3680043.8.498.40000002",
			},
			FailureReason: StorageCommitmentNoSuchObjectInstance,
		},
	}
	ds := buildCommitmentResult(transactionUID, committed, failed)

	result, err := parseCommitmentResult(StorageCommitmentEventFailures, ds)
	if err != nil {
		t.Fatalf("parseCommitmentResult: %v", err)
	}
	if !result.HasFailures() {
		t.Fatal("HasFailures must be true when at least one instance failed commitment")
	}
	if len(result.Failed) != 1 {
		t.Fatalf("Failed count = %d, want 1", len(result.Failed))
	}
	if result.Failed[0].FailureReason != StorageCommitmentNoSuchObjectInstance {
		t.Errorf("Failure Reason = %#04x, want %#04x (No Such Object Instance)",
			uint16(result.Failed[0].FailureReason), uint16(StorageCommitmentNoSuchObjectInstance))
	}
	if result.EventType != StorageCommitmentEventFailures {
		t.Errorf("EventType = %v, want StorageCommitmentEventFailures", result.EventType)
	}
	// Err() must report the partial failure as an error so a caller that treats commitment as a gate
	// fails closed rather than reading a success it did not get.
	if result.Err() == nil {
		t.Error("Result.Err() must be non-nil when there are failed instances")
	}
}

// TestParseCommitmentResultRequiresTransactionUID confirms a result data set with no Transaction UID
// is rejected (it cannot be correlated to a request), and a nil data set is rejected.
func TestParseCommitmentResultRequiresTransactionUID(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagReferencedSOPClassUID, "1.2.840.10008.5.1.4.1.1.2")
	if _, err := parseCommitmentResult(StorageCommitmentEventComplete, ds); err == nil {
		t.Error("parseCommitmentResult should reject a result with no Transaction UID")
	}
	if _, err := parseCommitmentResult(StorageCommitmentEventComplete, nil); err == nil {
		t.Error("parseCommitmentResult should reject a nil result data set")
	}
}

// startCommitmentReporter dials the SCU's receiver as a Storage Commitment SCP would on the separate
// reporting association, sends one N-EVENT-REPORT-RQ carrying the result data set with eventType, and
// reads back the N-EVENT-REPORT-RSP. It returns the response status the SCU acknowledged with.
func startCommitmentReporter(t *testing.T, addr string, eventType StorageCommitmentEventType, ds *dicom.DataSet) uint16 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	ae, err := NewAE(AETitle("STGCMTSCP"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("reporter NewAE: %v", err)
	}
	assoc, err := ae.Associate(ctx, addr, AETitle("STGCMTSCU"), StorageCommitmentContexts())
	if err != nil {
		t.Fatalf("reporter Associate: %v", err)
	}

	pcID, ts, ok := assoc.contextForQuery(storageCommitmentPushModelSOPClass)
	if !ok {
		t.Fatal("reporter could not find an accepted Storage Commitment context")
	}

	rq := CommandSet{
		CommandField:           CommandNEventReportRQ,
		MessageID:              1,
		AffectedSOPClassUID:    dicom.UID(storageCommitmentPushModelSOPClass),
		AffectedSOPInstanceUID: dicom.UID(storageCommitmentPushModelInstance),
		HasEventTypeID:         true,
		EventTypeID:            uint16(eventType),
		CommandDataSetType:     CommandDataSetPresent,
	}
	conn := assoc.requestor.Conn()
	m := assoc.requestor.Machine()
	if err := sendMessage(ctx, conn, m, pcID, rq, ds, ts, MaxPDULength(16382)); err != nil {
		t.Fatalf("reporter send N-EVENT-REPORT-RQ: %v", err)
	}
	rsp, _, _, err := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if err != nil {
		t.Fatalf("reporter receive N-EVENT-REPORT-RSP: %v", err)
	}
	_ = assoc.Release(ctx)
	if !rsp.HasStatus {
		t.Fatal("N-EVENT-REPORT-RSP carried no status")
	}
	return rsp.Status
}

// TestReceiveResultOnSeparateAssociation drives the supported separate-association reporting model:
// the SCU's CommitmentReceiver listens for an inbound association, a reporting peer connects and sends
// the N-EVENT-REPORT-RQ carrying the commitment result, and the receiver parses it, replies with the
// N-EVENT-REPORT-RSP, and surfaces the typed result through the registered handler.
func TestReceiveResultOnSeparateAssociation(t *testing.T) {
	const transactionUID = "1.2.826.0.1.3680043.8.498.49000300"
	committed := committedRefs()

	var (
		mu     sync.Mutex
		got    StorageCommitmentResult
		called bool
	)
	ae, err := NewAE(AETitle("STGCMTSCU"), WithACSETimeout(5*time.Second), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	recv := NewCommitmentReceiver(ae, WithCommitmentHandler(func(_ context.Context, info OpInfo, result StorageCommitmentResult) error {
		mu.Lock()
		defer mu.Unlock()
		called = true
		got = result
		_ = info
		return nil
	}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	recvCtx, recvCancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer recvCancel()
	recvErr := make(chan error, 1)
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			recvErr <- aerr
			return
		}
		recvErr <- recv.ServeConn(recvCtx, nc)
	}()

	ds := buildCommitmentResult(transactionUID, committed, nil)
	status := startCommitmentReporter(t, ln.Addr().String(), StorageCommitmentEventComplete, ds)
	if status != StatusSuccess.Code {
		t.Errorf("N-EVENT-REPORT-RSP status = %#04x, want Success (0x0000)", status)
	}

	if err := <-recvErr; err != nil {
		t.Fatalf("CommitmentReceiver.ServeConn: %v", err)
	}
	_ = ln.Close()

	mu.Lock()
	defer mu.Unlock()
	if !called {
		t.Fatal("commitment handler was not invoked for the inbound N-EVENT-REPORT")
	}
	if got.TransactionUID != transactionUID {
		t.Errorf("handler TransactionUID = %q, want %q", got.TransactionUID, transactionUID)
	}
	if len(got.Committed) != len(committed) {
		t.Errorf("handler Committed count = %d, want %d", len(got.Committed), len(committed))
	}
	if got.HasFailures() {
		t.Error("handler result should report no failures for an all-committed report")
	}
}

// TestCommitmentReceiverGrantsProviderSCPRole drives the inverted-role negotiation of the
// separate-association reporting model: the commitment provider opens the report association and
// proposes the Storage Commitment Push Model SCP role (it is the N-EVENT-REPORT SCP). The
// receiver's accept params must grant that SCP role so a strict peer is not refused; the test reads
// the grant back from the requestor's NegotiatedRoles, then completes a real report over the
// inverted-role association so the negotiation is exercised end to end.
func TestCommitmentReceiverGrantsProviderSCPRole(t *testing.T) {
	const transactionUID = "1.2.826.0.1.3680043.8.498.49000500"

	ae, err := NewAE(AETitle("STGCMTSCU"), WithACSETimeout(5*time.Second), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	recv := NewCommitmentReceiver(ae, WithCommitmentHandler(func(_ context.Context, _ OpInfo, _ StorageCommitmentResult) error {
		return nil
	}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	recvCtx, recvCancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer recvCancel()
	recvErr := make(chan error, 1)
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			recvErr <- aerr
			return
		}
		recvErr <- recv.ServeConn(recvCtx, nc)
	}()

	provider, err := NewAE(AETitle("STGCMTSCP"), WithACSETimeout(5*time.Second), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("provider NewAE: %v", err)
	}
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dialCancel()
	assoc, err := provider.Associate(dialCtx, ln.Addr().String(), AETitle("STGCMTSCU"), StorageCommitmentContexts(),
		WithRoleSelection(RoleSelection{SOPClassUID: storageCommitmentPushModelSOPClass, SCPRole: true}))
	if err != nil {
		t.Fatalf("provider Associate: %v", err)
	}

	roles := assoc.NegotiatedRoles()
	var granted *RoleSelection
	for i := range roles {
		if roles[i].SOPClassUID == storageCommitmentPushModelSOPClass {
			granted = &roles[i]
			break
		}
	}
	if granted == nil {
		t.Fatalf("no role-selection grant for the Storage Commitment Push Model SOP Class; got %+v", roles)
	}
	if !granted.SCPRole {
		t.Errorf("provider was not granted the SCP role for the report association; grant = %+v", *granted)
	}

	pcID, ts, ok := assoc.contextForQuery(storageCommitmentPushModelSOPClass)
	if !ok {
		t.Fatal("provider could not find an accepted Storage Commitment context")
	}
	rq := CommandSet{
		CommandField:           CommandNEventReportRQ,
		MessageID:              1,
		AffectedSOPClassUID:    dicom.UID(storageCommitmentPushModelSOPClass),
		AffectedSOPInstanceUID: dicom.UID(storageCommitmentPushModelInstance),
		HasEventTypeID:         true,
		EventTypeID:            uint16(StorageCommitmentEventComplete),
		CommandDataSetType:     CommandDataSetPresent,
	}
	conn := assoc.requestor.Conn()
	m := assoc.requestor.Machine()
	ds := buildCommitmentResult(transactionUID, committedRefs(), nil)
	if err := sendMessage(dialCtx, conn, m, pcID, rq, ds, ts, MaxPDULength(16382)); err != nil {
		t.Fatalf("provider send N-EVENT-REPORT-RQ: %v", err)
	}
	rsp, _, _, err := receiveMessage(dialCtx, conn, m, newMessageReassembler(ts))
	if err != nil {
		t.Fatalf("provider receive N-EVENT-REPORT-RSP: %v", err)
	}
	if !rsp.HasStatus || rsp.Status != StatusSuccess.Code {
		t.Errorf("N-EVENT-REPORT-RSP status = %#04x (present=%v), want Success", rsp.Status, rsp.HasStatus)
	}
	_ = assoc.Release(dialCtx)

	if err := <-recvErr; err != nil {
		t.Fatalf("CommitmentReceiver.ServeConn: %v", err)
	}
}

// TestReceiveRejectsMissingEventType is the event-type honesty gate: an N-EVENT-REPORT-RQ that omits
// the mandatory Event Type ID (PS3.4 J.3.3) must NOT be acknowledged as a clean success — a missing
// type leaves the command's Event Type ID zero, which would otherwise parse as a success with no
// failed items. The receiver must answer a failure status and surface a protocol error, and must not
// invoke the result handler.
func TestReceiveRejectsMissingEventType(t *testing.T) {
	const transactionUID = "1.2.826.0.1.3680043.8.498.49000400"

	var handlerCalled bool
	var mu sync.Mutex
	ae, err := NewAE(AETitle("STGCMTSCU"), WithACSETimeout(5*time.Second), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	recv := NewCommitmentReceiver(ae, WithCommitmentHandler(func(_ context.Context, _ OpInfo, _ StorageCommitmentResult) error {
		mu.Lock()
		defer mu.Unlock()
		handlerCalled = true
		return nil
	}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	recvCtx, recvCancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer recvCancel()
	recvErr := make(chan error, 1)
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			recvErr <- aerr
			return
		}
		recvErr <- recv.ServeConn(recvCtx, nc)
	}()

	ds := buildCommitmentResult(transactionUID, committedRefs(), nil)
	status := reportWithoutEventType(t, ln.Addr().String(), ds)

	// 0x0113 is "No Such Event Type" (PS3.7 §10.3.5): a clear failure category, never Success.
	if NewStatus(status, ServiceClassStorageCommitment).IsSuccess() {
		t.Errorf("N-EVENT-REPORT-RSP status = %#04x, want a failure for a missing Event Type ID", status)
	}
	if status != 0x0113 {
		t.Errorf("N-EVENT-REPORT-RSP status = %#04x, want 0x0113 (No Such Event Type)", status)
	}

	if err := <-recvErr; err == nil {
		t.Error("ServeConn should surface a protocol error for an N-EVENT-REPORT with no Event Type ID")
	}
	_ = ln.Close()

	mu.Lock()
	defer mu.Unlock()
	if handlerCalled {
		t.Error("the result handler must not be invoked for a report with no valid Event Type ID")
	}
}

// reportWithoutEventType dials the receiver as a provider would, sends one N-EVENT-REPORT-RQ that
// omits the mandatory Event Type ID, and returns the N-EVENT-REPORT-RSP status the receiver answered.
func reportWithoutEventType(t *testing.T, addr string, ds *dicom.DataSet) uint16 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	ae, err := NewAE(AETitle("STGCMTSCP"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("reporter NewAE: %v", err)
	}
	assoc, err := ae.Associate(ctx, addr, AETitle("STGCMTSCU"), StorageCommitmentContexts())
	if err != nil {
		t.Fatalf("reporter Associate: %v", err)
	}

	pcID, ts, ok := assoc.contextForQuery(storageCommitmentPushModelSOPClass)
	if !ok {
		t.Fatal("reporter could not find an accepted Storage Commitment context")
	}

	rq := CommandSet{
		CommandField:           CommandNEventReportRQ,
		MessageID:              1,
		AffectedSOPClassUID:    dicom.UID(storageCommitmentPushModelSOPClass),
		AffectedSOPInstanceUID: dicom.UID(storageCommitmentPushModelInstance),
		HasEventTypeID:         false, // the fault under test: the mandatory Event Type ID is absent
		CommandDataSetType:     CommandDataSetPresent,
	}
	conn := assoc.requestor.Conn()
	m := assoc.requestor.Machine()
	if err := sendMessage(ctx, conn, m, pcID, rq, ds, ts, MaxPDULength(16382)); err != nil {
		t.Fatalf("reporter send N-EVENT-REPORT-RQ: %v", err)
	}
	rsp, _, _, err := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if err != nil {
		t.Fatalf("reporter receive N-EVENT-REPORT-RSP: %v", err)
	}
	_ = assoc.Release(ctx)
	if !rsp.HasStatus {
		t.Fatal("N-EVENT-REPORT-RSP carried no status")
	}
	return rsp.Status
}

// TestStorageCommitmentEventTypeString pins the human labels each event type renders to, so a log or
// error never shows a bare integer.
func TestStorageCommitmentEventTypeString(t *testing.T) {
	cases := []struct {
		evt  StorageCommitmentEventType
		want string
	}{
		{StorageCommitmentEventComplete, "Storage Commitment Request Complete"},
		{StorageCommitmentEventFailures, "Storage Commitment Request Complete — Failures Exist"},
	}
	for _, tc := range cases {
		if got := tc.evt.String(); got != tc.want {
			t.Errorf("%d.String() = %q, want %q", uint16(tc.evt), got, tc.want)
		}
	}
}

// TestFailedSOPInstanceErrorIsTyped confirms a Result.Err over failed instances is a typed
// *CommitmentFailureError that errors.As recovers, carrying the failed-instance count.
func TestFailedSOPInstanceErrorIsTyped(t *testing.T) {
	result := StorageCommitmentResult{
		TransactionUID: "1.2.3",
		Failed: []dicom.FailedSOPInstance{
			{
				ReferencedSOPInstance: dicom.ReferencedSOPInstance{SOPClassUID: "1.2.840.10008.5.1.4.1.1.2", SOPInstanceUID: "1.2.3.4"},
				FailureReason:         StorageCommitmentProcessingFailure,
			},
		},
	}
	err := result.Err()
	if err == nil {
		t.Fatal("Err must be non-nil for a result with failed instances")
	}
	var cfe *CommitmentFailureError
	if !errors.As(err, &cfe) {
		t.Fatalf("Err = %T, want *CommitmentFailureError", err)
	}
	if cfe.FailedCount != 1 {
		t.Errorf("FailedCount = %d, want 1", cfe.FailedCount)
	}
}
