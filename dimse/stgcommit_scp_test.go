package dimse

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// startStgCommitServer serves a StorageCommitmentProvider over a Storage Commitment Push Model
// presentation context on loopback. It returns the running server.
func startStgCommitServer(t *testing.T, h any) *Server {
	t.Helper()
	ae, err := NewAE(AETitle("STGCMTSCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := NewServer(ae, StorageCommitmentContexts(), h)

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
	return srv
}

// receiveSameAssociationReport drives the SCU side of the synchronous Storage Commitment reporting
// model: after the N-ACTION-RSP, the SCP sends an N-EVENT-REPORT-RQ on the SAME association. This
// reads it, parses the result, sends the mandatory N-EVENT-REPORT-RSP back, and returns the parsed
// result. The high-level SCU Request reads only the N-ACTION-RSP, so the test drives this read itself.
func receiveSameAssociationReport(t *testing.T, assoc *Association, ctx context.Context) StorageCommitmentResult {
	t.Helper()
	conn := assoc.requestor.Conn()
	machine := assoc.requestor.Machine()
	pcID, ts, ok := assoc.contextForQuery(storageCommitmentPushModelSOPClass)
	if !ok {
		t.Fatal("no accepted Storage Commitment presentation context")
	}

	cmd, ds, _, err := receiveMessage(ctx, conn, machine, newMessageReassembler(ts))
	if err != nil {
		t.Fatalf("receive N-EVENT-REPORT-RQ: %v", err)
	}
	if cmd.CommandField != CommandNEventReportRQ {
		t.Fatalf("expected an N-EVENT-REPORT-RQ, got command field %#04x", uint16(cmd.CommandField))
	}

	rsp := CommandSet{
		CommandField:              CommandNEventReportRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.AffectedSOPInstanceUID,
		HasEventTypeID:            cmd.HasEventTypeID,
		EventTypeID:               cmd.EventTypeID,
		HasStatus:                 true,
		Status:                    StatusSuccess.Code,
		CommandDataSetType:        CommandDataSetNotPresent,
	}
	if err := sendCommand(ctx, conn, machine, pcID, rsp); err != nil {
		t.Fatalf("send N-EVENT-REPORT-RSP: %v", err)
	}

	result, err := parseCommitmentResult(StorageCommitmentEventType(cmd.EventTypeID), ds)
	if err != nil {
		t.Fatalf("parse commitment result: %v", err)
	}
	return result
}

func ref(instance string) dicom.ReferencedSOPInstance {
	return dicom.ReferencedSOPInstance{
		SOPClassUID:    dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2"), // CT Image Storage
		SOPInstanceUID: dicom.SOPInstanceUID(instance),
	}
}

// TestStgCommitSCPReportsSuccess is the acceptance gate for the Storage Commitment SCP success path:
// the SCU requests commitment of two instances the SCP holds, the SCP accepts the N-ACTION and then
// reports event type 1 (complete) with both instances in the Referenced SOP Sequence on the same
// association (PS3.4 J.3.3).
func TestStgCommitSCPReportsSuccess(t *testing.T) {
	const txn = "1.2.840.10008.3.1.2.3.3.txn.1"
	present := map[dicom.SOPInstanceUID]bool{"1.2.3.1": true, "1.2.3.2": true}
	srv := startStgCommitServer(t, NewStorageCommitmentProvider(CommitAllPresent(present)))
	assoc, ctx, cancel := dialStgCommitSCU(t, srv.Addr().String())
	defer cancel()

	refs := []dicom.ReferencedSOPInstance{ref("1.2.3.1"), ref("1.2.3.2")}
	status, err := assoc.StorageCommitment().Request(ctx, txn, refs)
	if err != nil {
		t.Fatalf("N-ACTION transport error: %v", err)
	}
	if !status.IsSuccess() {
		t.Errorf("N-ACTION status = %s, want Success (request accepted)", status)
	}

	result := receiveSameAssociationReport(t, assoc, ctx)
	if result.TransactionUID != txn {
		t.Errorf("report Transaction UID = %q, want %q", result.TransactionUID, txn)
	}
	if result.EventType != StorageCommitmentEventComplete {
		t.Errorf("report event type = %s, want Complete (1)", result.EventType)
	}
	if result.HasFailures() {
		t.Errorf("report has %d failures, want none", len(result.Failed))
	}
	if len(result.Committed) != 2 {
		t.Errorf("report committed %d instances, want 2", len(result.Committed))
	}
	if err := result.Err(); err != nil {
		t.Errorf("clean-success result Err() = %v, want nil", err)
	}
	_ = assoc.Release(ctx)
}

// TestStgCommitSCPReportsPartialFailure confirms the partial-failure path: the SCU requests commitment
// of one instance the SCP holds and one it does not; the SCP reports event type 2 (failures exist)
// with the held instance committed and the missing one in the Failed SOP Sequence carrying No Such
// Object Instance (0x0112) — a failed instance is never laundered into the committed set (PS3.4
// J.3.3).
func TestStgCommitSCPReportsPartialFailure(t *testing.T) {
	const txn = "1.2.840.10008.3.1.2.3.3.txn.2"
	present := map[dicom.SOPInstanceUID]bool{"1.2.3.1": true} // 1.2.3.9 is absent
	srv := startStgCommitServer(t, NewStorageCommitmentProvider(CommitAllPresent(present)))
	assoc, ctx, cancel := dialStgCommitSCU(t, srv.Addr().String())
	defer cancel()

	refs := []dicom.ReferencedSOPInstance{ref("1.2.3.1"), ref("1.2.3.9")}
	if status, err := assoc.StorageCommitment().Request(ctx, txn, refs); err != nil || !status.IsSuccess() {
		t.Fatalf("N-ACTION = (%s, %v), want Success", status, err)
	}

	result := receiveSameAssociationReport(t, assoc, ctx)
	if result.EventType != StorageCommitmentEventFailures {
		t.Errorf("report event type = %s, want Failures (2)", result.EventType)
	}
	if !result.HasFailures() {
		t.Fatal("partial-failure report has no failures")
	}
	if len(result.Committed) != 1 {
		t.Errorf("report committed %d instances, want 1", len(result.Committed))
	}
	if len(result.Failed) != 1 {
		t.Fatalf("report failed %d instances, want 1", len(result.Failed))
	}
	if result.Failed[0].SOPInstanceUID != "1.2.3.9" {
		t.Errorf("failed instance = %q, want 1.2.3.9", result.Failed[0].SOPInstanceUID)
	}
	if result.Failed[0].FailureReason != StorageCommitmentNoSuchObjectInstance {
		t.Errorf("failure reason = %#04x, want No Such Object Instance (0x0112)", result.Failed[0].FailureReason)
	}
	if result.Err() == nil {
		t.Error("a partial-failure result Err() = nil, want a CommitmentFailureError")
	}
	_ = assoc.Release(ctx)
}

// TestStgCommitProviderNActionValidation pins the N-ACTION validation the provider performs directly:
// a wrong SOP class, a wrong target instance, a wrong action type, and a missing Transaction UID are
// each refused, and a refused N-ACTION returns a non-success status (so no report follows).
func TestStgCommitProviderNActionValidation(t *testing.T) {
	p := NewStorageCommitmentProvider(CommitAllPresent(nil))
	pushClass := dicom.UID(storageCommitmentPushModelSOPClass)

	withTxn := func() *dicom.DataSet {
		ds := dicom.NewDataSet()
		ds.SetString(dicom.TagTransactionUID, "1.2.3.txn")
		return ds
	}
	withTxnAndRefs := func() *dicom.DataSet {
		ds := withTxn()
		item := dicom.NewDataSet()
		item.SetString(dicom.TagReferencedSOPClassUID, "1.2.840.10008.5.1.4.1.1.2")
		item.SetString(dicom.TagReferencedSOPInstanceUID, "1.2.3.1")
		ds.Set(dicom.Element{Tag: dicom.TagReferencedSOPSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence(item))})
		return ds
	}
	withTxnEmptyRefs := func() *dicom.DataSet {
		ds := withTxn()
		ds.Set(dicom.Element{Tag: dicom.TagReferencedSOPSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence())})
		return ds
	}

	for _, tc := range []struct {
		name string
		req  NRequest
		want uint16
	}{
		{
			name: "wrong SOP class",
			req:  NRequest{RequestedSOPClassUID: dicom.UID(otherNServiceSOPClass), RequestedSOPInstanceUID: storageCommitmentPushModelInstance, HasActionTypeID: true, ActionTypeID: 1, DataSet: withTxn()},
			want: StatusSOPClassNotSupported.Code,
		},
		{
			name: "wrong target instance",
			req:  NRequest{RequestedSOPClassUID: pushClass, RequestedSOPInstanceUID: "1.2.3.not.the.well.known", HasActionTypeID: true, ActionTypeID: 1, DataSet: withTxn()},
			want: StatusNoSuchSOPInstance.Code,
		},
		{
			name: "wrong action type",
			req:  NRequest{RequestedSOPClassUID: pushClass, RequestedSOPInstanceUID: storageCommitmentPushModelInstance, HasActionTypeID: true, ActionTypeID: 9, DataSet: withTxn()},
			want: StorageCommitmentNoSuchAction.Code,
		},
		{
			name: "missing transaction UID",
			req:  NRequest{RequestedSOPClassUID: pushClass, RequestedSOPInstanceUID: storageCommitmentPushModelInstance, HasActionTypeID: true, ActionTypeID: 1, DataSet: dicom.NewDataSet()},
			want: 0x0114,
		},
		{
			name: "absent referenced SOP sequence",
			req:  NRequest{RequestedSOPClassUID: pushClass, RequestedSOPInstanceUID: storageCommitmentPushModelInstance, HasActionTypeID: true, ActionTypeID: 1, DataSet: withTxn()},
			want: 0x0120,
		},
		{
			name: "empty referenced SOP sequence",
			req:  NRequest{RequestedSOPClassUID: pushClass, RequestedSOPInstanceUID: storageCommitmentPushModelInstance, HasActionTypeID: true, ActionTypeID: 1, DataSet: withTxnEmptyRefs()},
			want: 0x0120,
		},
		{
			name: "valid request with one referenced instance",
			req:  NRequest{RequestedSOPClassUID: pushClass, RequestedSOPInstanceUID: storageCommitmentPushModelInstance, HasActionTypeID: true, ActionTypeID: 1, DataSet: withTxnAndRefs()},
			want: StatusStorageCommitmentSuccess.Code,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status := p.NAction(context.Background(), tc.req)
			if status.Code != tc.want {
				t.Errorf("NAction status = %s, want %#04x", status, tc.want)
			}
			if tc.want != StatusStorageCommitmentSuccess.Code && status.IsSuccess() {
				t.Error("a refused N-ACTION must not report Success")
			}
		})
	}
}
