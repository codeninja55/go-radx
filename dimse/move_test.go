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

// moveResponse is a canned C-MOVE-RSP the mock SCP returns: a status code and the four
// sub-operation counts the RSP carries (Remaining/Completed/Failed/Warning).
type moveResponse struct {
	status    uint16
	remaining uint16
	completed uint16
	failed    uint16
	warning   uint16
}

// moveSCPObservation is what the mock C-MOVE SCP captured from the SCU's request.
type moveSCPObservation struct {
	mu         sync.Mutex
	requestCmd CommandSet
	identifier *dicom.DataSet
	err        error
}

func (o *moveSCPObservation) snapshot() moveSCPObservation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return moveSCPObservation{
		requestCmd: o.requestCmd,
		identifier: o.identifier,
		err:        o.err,
	}
}

// startMoveSCP listens on loopback, accepts a Study Root MOVE association, reads the C-MOVE-RQ plus
// its identifier, and returns the canned C-MOVE-RSP responses in order (each carrying the four
// sub-operation counts, the terminal one last). It mirrors startFindSCP but for the MOVE model.
func startMoveSCP(t *testing.T, responses []moveResponse) (string, *moveSCPObservation) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	obs := &moveSCPObservation{}
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: "MOVESCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(studyRootMoveSOPClass),
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
		serveCannedMove(ctx, acc, responses, obs)
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String(), obs
}

// serveCannedMove reads the C-MOVE-RQ and identifier, then sends the canned C-MOVE-RSP responses
// (each carrying the four sub-operation counts). No C-MOVE-RSP carries a dataset.
func serveCannedMove(ctx context.Context, acc *acse.Acceptor, responses []moveResponse, obs *moveSCPObservation) {
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

	rqCmd, identifier, _, rerr := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if rerr != nil {
		obs.mu.Lock()
		obs.err = rerr
		obs.mu.Unlock()
		return
	}
	obs.mu.Lock()
	obs.requestCmd = rqCmd
	obs.identifier = identifier
	obs.mu.Unlock()

	for _, r := range responses {
		rsp := CommandSet{
			CommandField:              CommandCMoveRSP,
			MessageIDBeingRespondedTo: rqCmd.MessageID,
			AffectedSOPClassUID:       rqCmd.AffectedSOPClassUID,
			HasStatus:                 true,
			Status:                    r.status,
			CommandDataSetType:        CommandDataSetNotPresent,
			HasSubOpCounts:            true,
			RemainingSubOperations:    r.remaining,
			CompletedSubOperations:    r.completed,
			FailedSubOperations:       r.failed,
			WarningSubOperations:      r.warning,
		}
		if serr := sendCommand(ctx, conn, m, pcID, rsp); serr != nil {
			obs.mu.Lock()
			obs.err = serr
			obs.mu.Unlock()
			return
		}
	}
}

// dialMoveSCU opens an association proposing the Query/Retrieve contexts to the mock MOVE SCP.
func dialMoveSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("MOVESCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("MOVESCP"), QueryRetrieveContexts())
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// TestMoveWritesQueryLevel is the DIMSE-015 carry-forward: Move writes the requested QueryLevel into
// (0008,0052) of the sent identifier and carries the Move Destination AE Title in (0000,0600).
func TestMoveWritesQueryLevel(t *testing.T) {
	addr, obs := startMoveSCP(t, []moveResponse{{status: StatusMoveSuccess.Code}})
	assoc, ctx, cancel := dialMoveSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3.99")

	for range assoc.Move(ctx, query, QueryLevelStudy, AETitle("DEST-AE")) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.err != nil {
		t.Fatalf("SCP error: %v", got.err)
	}
	if got.identifier == nil {
		t.Fatal("SCP received no identifier dataset")
	}
	level, ok := got.identifier.GetString(dicom.TagQueryRetrieveLevel)
	if !ok || level != "STUDY" {
		t.Errorf("Query/Retrieve Level = %q (present=%v), want STUDY (DIMSE-015)", level, ok)
	}
	if got.requestCmd.MoveDestination != AETitle("DEST-AE") {
		t.Errorf("C-MOVE-RQ Move Destination = %q, want DEST-AE", got.requestCmd.MoveDestination)
	}
	if got.requestCmd.AffectedSOPClassUID != dicom.UID(studyRootMoveSOPClass) {
		t.Errorf("C-MOVE-RQ Affected SOP Class = %q, want Study Root MOVE", got.requestCmd.AffectedSOPClassUID)
	}
	// The caller's original query must be untouched (Move writes into a copy).
	if _, present := query.GetString(dicom.TagQueryRetrieveLevel); present {
		t.Error("Move mutated the caller's query dataset; it must write into a copy")
	}
}

// TestMoveSCUReportsCounts drives the iterator over two Pending RSPs carrying decreasing Remaining
// counts then a terminal Success, and asserts the SCU surfaces the four sub-operation counts of the
// most recently yielded response via SubOperationCounts().
func TestMoveSCUReportsCounts(t *testing.T) {
	addr, _ := startMoveSCP(t, []moveResponse{
		{status: StatusMovePending.Code, remaining: 2, completed: 0},
		{status: StatusMovePending.Code, remaining: 1, completed: 1},
		{status: StatusMoveSuccess.Code, remaining: 0, completed: 2},
	})
	assoc, ctx, cancel := dialMoveSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")

	var counts []SubOperationCounts
	var terminal Status
	var terminalSeen bool
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("DEST-AE")) {
		switch {
		case status.IsPending():
			counts = append(counts, assoc.SubOperationCounts())
		default:
			terminal = status
			terminalSeen = true
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if len(counts) != 2 {
		t.Fatalf("saw %d Pending responses, want 2", len(counts))
	}
	if counts[0].Remaining != 2 || counts[0].Completed != 0 {
		t.Errorf("first Pending counts = %+v, want Remaining 2, Completed 0", counts[0])
	}
	if counts[1].Remaining != 1 || counts[1].Completed != 1 {
		t.Errorf("second Pending counts = %+v, want Remaining 1, Completed 1", counts[1])
	}
	if !terminalSeen || !terminal.IsSuccess() {
		t.Errorf("terminal status = %s (seen=%v), want Success", terminal, terminalSeen)
	}
	// After the terminal, the counts reflect the final RSP.
	final := assoc.SubOperationCounts()
	if final.Completed != 2 || final.Remaining != 0 {
		t.Errorf("final counts = %+v, want Completed 2, Remaining 0", final)
	}
}

// TestMoveTerminalWarningOnSubOpFailure verifies a 0xB000 terminal (one or more sub-operations
// failed) is surfaced faithfully as a Warning, never laundered into Success (PRD §9.2).
func TestMoveTerminalWarningOnSubOpFailure(t *testing.T) {
	addr, _ := startMoveSCP(t, []moveResponse{
		{status: StatusMovePending.Code, remaining: 1, completed: 0},
		{status: StatusMoveSubOpsCompleteWithFailures.Code, completed: 1, failed: 1},
	})
	assoc, ctx, cancel := dialMoveSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")

	var terminal Status
	var sawSuccess bool
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("DEST-AE")) {
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
		t.Error("a sub-operation-failure warning (0xB000) was laundered into a Success status")
	}
	if !terminal.IsWarning() {
		t.Errorf("terminal status = %s, want a Warning (0xB000 sub-operations complete with failures)", terminal)
	}
	if final := assoc.SubOperationCounts(); final.Failed != 1 {
		t.Errorf("final Failed count = %d, want 1", final.Failed)
	}
}

// TestMoveDestinationUnknownIsFailureData verifies a 0xA801 "Move Destination Unknown" terminal is
// surfaced as a Failure-category status (data the caller inspects), never a panic and never laundered.
func TestMoveDestinationUnknownIsFailureData(t *testing.T) {
	addr, _ := startMoveSCP(t, []moveResponse{
		{status: StatusMoveDestinationUnknown.Code},
	})
	assoc, ctx, cancel := dialMoveSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")

	var statuses []Status
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("NOPE")) {
		statuses = append(statuses, status)
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil (a Failure status is in-band data, not a transport fault)", err)
	}
	_ = assoc.Release(ctx)

	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Move to an unknown destination yielded %v, want one terminal Failure", statuses)
	}
	if statuses[0].Code != StatusMoveDestinationUnknown.Code {
		t.Errorf("terminal status = %s, want 0xA801 Move Destination Unknown", statuses[0])
	}
}

// TestMovePatientLevelDefaultsToPatientRoot verifies a patient-level Move with no WithQueryModel
// negotiates and sends the Patient Root MOVE model against a Patient-Root-only peer.
func TestMovePatientLevelDefaultsToPatientRoot(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	obs := &moveSCPObservation{}
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: "MOVESCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(patientRootMoveSOPClass),
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
		serveCannedMove(ctx, acc, []moveResponse{{status: StatusMoveSuccess.Code}}, obs)
		_ = acc.ServeRelease(ctx)
	}()

	assoc, ctx, cancel := dialMoveSCU(t, ln.Addr().String())
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagPatientID, "PID-1")

	for range assoc.Move(ctx, query, QueryLevelPatient, AETitle("DEST-AE")) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move LastError = %v, want nil (Patient Root MOVE context was negotiated)", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.requestCmd.AffectedSOPClassUID != dicom.UID(patientRootMoveSOPClass) {
		t.Errorf("C-MOVE-RQ Affected SOP Class = %q, want Patient Root MOVE", got.requestCmd.AffectedSOPClassUID)
	}
}

// TestMoveRejectsNonMoveModel verifies WithQueryModel naming a FIND (non-MOVE) SOP Class fails
// closed: Move yields one terminal failure and sets a typed *ValidationError, transmitting nothing.
func TestMoveRejectsNonMoveModel(t *testing.T) {
	addr, obs := startMoveSCP(t, []moveResponse{{status: StatusMoveSuccess.Code}})
	assoc, ctx, cancel := dialMoveSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")

	var statuses []Status
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("DEST-AE"), WithQueryModel(studyRootFindSOPClass)) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Move with a FIND model yielded %v, want one terminal failure", statuses)
	}
	var ve *ValidationError
	if !errors.As(assoc.LastError(), &ve) {
		t.Errorf("LastError() = %T, want *ValidationError", assoc.LastError())
	}
	_ = assoc.Abort(ctx)

	got := obs.snapshot()
	if got.identifier != nil {
		t.Error("a C-MOVE-RQ was transmitted for a non-MOVE model; want fail-closed before any wire I/O")
	}
}

// TestMoveRequiresDestination verifies Move with an empty destination AE Title fails closed: it
// yields one terminal failure and sets a typed error, transmitting no C-MOVE-RQ (a C-MOVE-RQ
// without a Move Destination is malformed).
func TestMoveRequiresDestination(t *testing.T) {
	addr, obs := startMoveSCP(t, []moveResponse{{status: StatusMoveSuccess.Code}})
	assoc, ctx, cancel := dialMoveSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")

	var statuses []Status
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("")) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Move with an empty destination yielded %v, want one terminal failure", statuses)
	}
	var ve *ValidationError
	if !errors.As(assoc.LastError(), &ve) {
		t.Errorf("LastError() = %T, want *ValidationError", assoc.LastError())
	}
	_ = assoc.Abort(ctx)

	got := obs.snapshot()
	if got.identifier != nil {
		t.Error("a C-MOVE-RQ was transmitted with no Move Destination; want fail-closed before any wire I/O")
	}
}

// TestMoveOnReleasedAssociation is the DIMSE-017 carry-forward: Move on a released association yields
// a single terminal failure status and sets a typed error, never panicking.
func TestMoveOnReleasedAssociation(t *testing.T) {
	addr := startEchoAcceptor(t)
	ae, _ := NewAE(AETitle("MOVESCU"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, addr, AETitle("ACCEPTOR"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")

	var statuses []Status
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("DEST-AE")) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Move on released association yielded %d statuses, want 1 terminal failure", len(statuses))
	}
	var assocErr *AssociationError
	if !errors.As(assoc.LastError(), &assocErr) {
		t.Errorf("LastError() = %T, want *AssociationError", assoc.LastError())
	}
}

// TestMoveTransportFaultSetsLastError verifies that a peer aborting mid-move ends the iterator and
// surfaces the fault via LastError(), not a panic, and not laundered into Success.
func TestMoveTransportFaultSetsLastError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: "MOVESCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(studyRootMoveSOPClass),
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			_ = nc.Close()
			return
		}
		// Read the C-MOVE-RQ, then abort instead of answering.
		_, _, _, _ = receiveMessage(ctx, acc.Conn(), acc.Machine(), newMessageReassembler(dicom.ExplicitVRLittleEndian))
		_ = acc.Abort(ctx)
	}()

	assoc, ctx, cancel := dialMoveSCU(t, ln.Addr().String())
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")

	var sawSuccess bool
	for status := range assoc.Move(ctx, query, QueryLevelStudy, AETitle("DEST-AE")) {
		if status.IsSuccess() {
			sawSuccess = true
		}
	}
	if sawSuccess {
		t.Error("a transport fault was laundered into a Success status")
	}
	if assoc.LastError() == nil {
		t.Fatal("Move LastError() = nil after a mid-move abort, want a typed transport error")
	}
}
