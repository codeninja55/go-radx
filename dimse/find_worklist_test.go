package dimse

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// startWorklistSCP listens on loopback, accepts a Modality Worklist Information Model — FIND
// association, reads the C-FIND-RQ plus its identifier, and returns the canned responses in order.
// It mirrors startFindSCP but negotiates the worklist abstract syntax so an MWL C-FIND is accepted.
func startWorklistSCP(t *testing.T, responses []findResponse) (string, *findSCPObservation) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	obs := &findSCPObservation{}
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: "MWLSCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(modalityWorklistSOPClass),
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
		serveCannedFind(ctx, acc, responses, obs)
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String(), obs
}

// dialWorklistSCU opens an association proposing the Modality Worklist context to the mock SCP.
func dialWorklistSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("MWLSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("MWLSCP"), BasicWorklistContexts())
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// TestNewWorklistQueryHasEmptyScheduledStepSequence verifies the worklist query skeleton carries a
// single empty Scheduled Procedure Step Sequence (0040,0100) item and no Query/Retrieve Level.
func TestNewWorklistQueryHasEmptyScheduledStepSequence(t *testing.T) {
	query := NewWorklistQuery()

	if _, present := query.GetString(dicom.TagQueryRetrieveLevel); present {
		t.Error("worklist query carries a Query/Retrieve Level (0008,0052); the worklist model is flat")
	}
	seq, ok := query.GetSequence(dicom.TagScheduledProcedureStepSequence)
	if !ok {
		t.Fatal("worklist query has no Scheduled Procedure Step Sequence (0040,0100)")
	}
	if seq.Len() != 1 {
		t.Fatalf("Scheduled Procedure Step Sequence item count = %d, want 1", seq.Len())
	}
	for item := range seq.Items() {
		if item.DataSet.Len() != 0 {
			t.Errorf("Scheduled Procedure Step item element count = %d, want 0 (empty universal match)", item.DataSet.Len())
		}
	}
}

// TestFindWorklistSuppressesQueryLevel is the worklist-level-semantics gate: a Modality Worklist
// C-FIND must NOT send a Query/Retrieve Level (0008,0052), unlike Patient/Study Root C-FIND.
func TestFindWorklistSuppressesQueryLevel(t *testing.T) {
	addr, obs := startWorklistSCP(t, []findResponse{{status: StatusWorklistSuccess.Code}})
	assoc, ctx, cancel := dialWorklistSCU(t, addr)
	defer cancel()

	query := NewWorklistQuery()

	for range assoc.FindWorklist(ctx, query) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("FindWorklist LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.err != nil {
		t.Fatalf("SCP error: %v", got.err)
	}
	if got.identifier == nil {
		t.Fatal("SCP received no identifier dataset")
	}
	if _, present := got.identifier.GetString(dicom.TagQueryRetrieveLevel); present {
		t.Error("MWL C-FIND sent a Query/Retrieve Level (0008,0052); the worklist model is flat and carries no level")
	}
	if got.requestCmd.AffectedSOPClassUID != dicom.UID(modalityWorklistSOPClass) {
		t.Errorf("C-FIND-RQ Affected SOP Class = %q, want Modality Worklist %q",
			got.requestCmd.AffectedSOPClassUID, modalityWorklistSOPClass)
	}
}

// TestFindWorklistYieldsScheduledSteps drives the iterator over two Pending worklist matches then
// Success, asserting each Pending carries a Scheduled Procedure Step Sequence the modality reads.
func TestFindWorklistYieldsScheduledSteps(t *testing.T) {
	match1 := worklistMatch("SPS-1", "CT")
	match2 := worklistMatch("SPS-2", "MR")

	addr, _ := startWorklistSCP(t, []findResponse{
		{status: StatusWorklistPending.Code, identifier: match1},
		{status: StatusWorklistPending.Code, identifier: match2},
		{status: StatusWorklistSuccess.Code},
	})
	assoc, ctx, cancel := dialWorklistSCU(t, addr)
	defer cancel()

	var stepIDs []string
	var terminal Status
	var terminalSeen bool
	for status, ds := range assoc.FindWorklist(ctx, NewWorklistQuery()) {
		switch {
		case status.IsPending():
			if ds == nil {
				t.Error("Pending worklist response yielded a nil dataset; want a matching identifier")
				continue
			}
			seq, ok := ds.GetSequence(dicom.TagScheduledProcedureStepSequence)
			if !ok || seq.Len() == 0 {
				t.Error("Pending worklist match has no Scheduled Procedure Step Sequence item")
				continue
			}
			for item := range seq.Items() {
				if id, has := item.DataSet.GetString(dicom.TagScheduledProcedureStepID); has {
					stepIDs = append(stepIDs, id)
				}
			}
		default:
			terminal = status
			terminalSeen = true
			if ds != nil {
				t.Error("terminal worklist response yielded a non-nil dataset; want nil")
			}
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("FindWorklist LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if len(stepIDs) != 2 || stepIDs[0] != "SPS-1" || stepIDs[1] != "SPS-2" {
		t.Errorf("scheduled procedure step IDs = %v, want [SPS-1 SPS-2]", stepIDs)
	}
	if !terminalSeen {
		t.Fatal("iterator never yielded a terminal status")
	}
	if !terminal.IsSuccess() {
		t.Errorf("terminal status = %s, want Success", terminal)
	}
}

// TestFindWorklistOnNilQuerySendsNoLevel verifies a nil worklist query sends an empty identifier
// that still carries no Query/Retrieve Level — Find must not write a level for the worklist model.
func TestFindWorklistOnNilQuerySendsNoLevel(t *testing.T) {
	addr, obs := startWorklistSCP(t, []findResponse{{status: StatusWorklistSuccess.Code}})
	assoc, ctx, cancel := dialWorklistSCU(t, addr)
	defer cancel()

	for range assoc.FindWorklist(ctx, nil) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("FindWorklist LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.identifier == nil {
		t.Fatal("SCP received no identifier dataset for a nil worklist query")
	}
	if _, present := got.identifier.GetString(dicom.TagQueryRetrieveLevel); present {
		t.Error("nil-query MWL C-FIND sent a Query/Retrieve Level; the worklist model is flat")
	}
}

// TestFindWorklistNoMatchingContext verifies FindWorklist fails closed when no worklist context was
// negotiated: it yields one terminal failure and sets LastError, transmitting no C-FIND-RQ.
func TestFindWorklistNoMatchingContext(t *testing.T) {
	addr := startEchoAcceptor(t) // Verification only — no worklist context
	ae, _ := NewAE(AETitle("MWLSCU"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, addr, AETitle("ACCEPTOR"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	var statuses []Status
	for status := range assoc.FindWorklist(ctx, NewWorklistQuery()) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("FindWorklist with no worklist context yielded %v, want one terminal failure", statuses)
	}
	var assocErr *AssociationError
	if !errors.As(assoc.LastError(), &assocErr) {
		t.Errorf("LastError() = %T, want *AssociationError", assoc.LastError())
	}
}

// TestFindWorklistIgnoresWithQueryModel verifies a caller-supplied WithQueryModel naming a Q/R model
// does not divert FindWorklist away from the worklist model: the worklist model is always sent, and
// no Query/Retrieve Level is written.
func TestFindWorklistIgnoresWithQueryModel(t *testing.T) {
	addr, obs := startWorklistSCP(t, []findResponse{{status: StatusWorklistSuccess.Code}})
	assoc, ctx, cancel := dialWorklistSCU(t, addr)
	defer cancel()

	for range assoc.FindWorklist(ctx, NewWorklistQuery(), WithQueryModel(studyRootFindSOPClass)) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("FindWorklist LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.requestCmd.AffectedSOPClassUID != dicom.UID(modalityWorklistSOPClass) {
		t.Errorf("C-FIND-RQ Affected SOP Class = %q, want Modality Worklist %q (override must not win)",
			got.requestCmd.AffectedSOPClassUID, modalityWorklistSOPClass)
	}
	if _, present := got.identifier.GetString(dicom.TagQueryRetrieveLevel); present {
		t.Error("FindWorklist wrote a Query/Retrieve Level despite a Q/R-model override; worklist stays flat")
	}
}

// worklistMatch builds a single-item worklist match identifier carrying a Scheduled Procedure Step
// Sequence (0040,0100) whose item holds a Scheduled Procedure Step ID and Modality, the minimal
// shape a returned MWL match takes. The values are synthetic test fixtures, not PHI.
func worklistMatch(stepID, modality string) *dicom.DataSet {
	step := dicom.NewDataSet()
	step.SetString(dicom.TagScheduledProcedureStepID, stepID)
	step.SetString(dicom.TagModality, modality)
	match := dicom.NewDataSet()
	match.Set(dicom.Element{
		Tag:   dicom.TagScheduledProcedureStepSequence,
		VR:    dicom.VRSQ,
		Value: dicom.NewSequenceValue(dicom.NewSequence(step)),
	})
	return match
}
