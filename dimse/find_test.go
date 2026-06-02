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

// TestQueryLevelString verifies QueryLevel renders the DICOM keyword written into (0008,0052).
func TestQueryLevelString(t *testing.T) {
	cases := map[QueryLevel]string{
		QueryLevelPatient: "PATIENT",
		QueryLevelStudy:   "STUDY",
		QueryLevelSeries:  "SERIES",
		QueryLevelImage:   "IMAGE",
	}
	for level, want := range cases {
		if got := level.String(); got != want {
			t.Errorf("QueryLevel(%d).String() = %q, want %q", level, got, want)
		}
	}
}

// TestParseQueryLevel verifies the inverse mapping and that an unknown keyword is a typed error.
func TestParseQueryLevel(t *testing.T) {
	for s, want := range map[string]QueryLevel{
		"PATIENT": QueryLevelPatient,
		"STUDY":   QueryLevelStudy,
		"SERIES":  QueryLevelSeries,
		"IMAGE":   QueryLevelImage,
	} {
		got, err := ParseQueryLevel(s)
		if err != nil {
			t.Errorf("ParseQueryLevel(%q) error = %v, want nil", s, err)
		}
		if got != want {
			t.Errorf("ParseQueryLevel(%q) = %v, want %v", s, got, want)
		}
	}
	if _, err := ParseQueryLevel("BOGUS"); err == nil {
		t.Error("ParseQueryLevel(\"BOGUS\") = nil error, want a *ValidationError")
	}
}

// findResponse is a canned C-FIND-RSP the mock SCP returns: a status code and, on a Pending,
// an identifier dataset.
type findResponse struct {
	status     uint16
	identifier *dicom.DataSet
}

// findSCPObservation is what the mock C-FIND SCP captured from the SCU's request.
type findSCPObservation struct {
	mu          sync.Mutex
	requestCmd  CommandSet
	identifier  *dicom.DataSet
	cancelSeen  bool
	cancelMsgID uint16
	err         error
}

func (o *findSCPObservation) snapshot() findSCPObservation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return findSCPObservation{
		requestCmd:  o.requestCmd,
		identifier:  o.identifier,
		cancelSeen:  o.cancelSeen,
		cancelMsgID: o.cancelMsgID,
		err:         o.err,
	}
}

// startFindSCP listens on loopback, accepts a Study Root FIND association, reads the C-FIND-RQ
// plus its identifier, returns the canned responses in order (each a Pending or terminal RSP),
// and records whether a C-CANCEL-RQ arrives. After the C-FIND-RQ identifier is read it watches
// for an interleaved C-CANCEL while sending responses, so an early-break cancel is captured.
func startFindSCP(t *testing.T, responses []findResponse) (string, *findSCPObservation) {
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
			CalledAETitle: "FINDSCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(studyRootFindSOPClass),
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

// serveCannedFind reads the C-FIND-RQ and identifier, then sends the canned responses. While
// sending Pending responses it polls (non-blocking, via a short read deadline) for an inbound
// C-CANCEL-RQ so an early break by the SCU is observed.
func serveCannedFind(ctx context.Context, acc *acse.Acceptor, responses []findResponse, obs *findSCPObservation) {
	m := acc.Machine()
	conn := acc.Conn()

	// Resolve the negotiated transfer syntax for the accepted FIND context.
	var pcID uint8
	ts := dicom.ImplicitVRLittleEndian
	for _, pc := range acc.AcceptedContexts() {
		if pc.Result == 0 { // acceptance
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
			CommandField:              CommandCFindRSP,
			MessageIDBeingRespondedTo: rqCmd.MessageID,
			AffectedSOPClassUID:       rqCmd.AffectedSOPClassUID,
			HasStatus:                 true,
			Status:                    r.status,
		}
		if r.identifier != nil {
			rsp.CommandDataSetType = CommandDataSetPresent
		} else {
			rsp.CommandDataSetType = CommandDataSetNotPresent
		}
		if serr := sendMessage(ctx, conn, m, pcID, rsp, r.identifier, ts, MaxPDULength(16382)); serr != nil {
			obs.mu.Lock()
			obs.err = serr
			obs.mu.Unlock()
			return
		}
		// After each Pending, give the SCU a moment to break and send a C-CANCEL. Poll once
		// with a short deadline; on a C-CANCEL-RQ, answer with a terminal Cancel (0xFE00) RSP so the
		// SCU's drain reaches the terminal status, then end the canned sequence — mirroring a real
		// SCP that stops matching and replies with a Cancel terminal (PS3.4 C.4.1.3).
		if NewStatus(r.status, ServiceClassFind).IsPending() {
			if pollForCancel(ctx, conn, m, obs) {
				cancelTerminal := CommandSet{
					CommandField:              CommandCFindRSP,
					MessageIDBeingRespondedTo: rqCmd.MessageID,
					AffectedSOPClassUID:       rqCmd.AffectedSOPClassUID,
					HasStatus:                 true,
					Status:                    StatusCancel.Code, // 0xFE00
					CommandDataSetType:        CommandDataSetNotPresent,
				}
				_ = sendMessage(ctx, conn, m, pcID, cancelTerminal, nil, ts, MaxPDULength(16382))
				return
			}
		}
	}
}

// pollForCancel does one short-deadline inbound read looking for a C-CANCEL-RQ. It returns true
// if a cancel was seen (so the caller stops sending). A timeout means no cancel yet — continue.
func pollForCancel(ctx context.Context, conn *dul.Conn, m *dul.StateMachine, obs *findSCPObservation) bool {
	pollCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	cmd, _, _, err := receiveMessage(pollCtx, conn, m, newMessageReassembler(dicom.ImplicitVRLittleEndian))
	if err != nil {
		return false
	}
	if cmd.CommandField == CommandCCancelRQ {
		obs.mu.Lock()
		obs.cancelSeen = true
		obs.cancelMsgID = cmd.MessageIDBeingRespondedTo
		obs.mu.Unlock()
		return true
	}
	return false
}

// dialFindSCU opens an association proposing the Query/Retrieve contexts to the mock SCP. The DIMSE
// timeout is kept short so a best-effort cancel-drain cannot block the test if the mock stops
// responding.
func dialFindSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("FINDSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("FINDSCP"), QueryRetrieveContexts())
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// TestFindWritesQueryLevel is the DIMSE-015 regression: Find writes the requested QueryLevel into
// (0008,0052) of the sent identifier even when the caller's query omits it.
func TestFindWritesQueryLevel(t *testing.T) {
	addr, obs := startFindSCP(t, []findResponse{{status: StatusFindSuccess.Code}})
	assoc, ctx, cancel := dialFindSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagPatientID, "PID-1")
	query.SetEmpty(dicom.TagStudyInstanceUID)

	for range assoc.Find(ctx, query, QueryLevelStudy) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError = %v, want nil", err)
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
	if !ok {
		t.Fatal("sent identifier has no Query/Retrieve Level (0008,0052) — DIMSE-015 regression")
	}
	if level != "STUDY" {
		t.Errorf("Query/Retrieve Level = %q, want %q", level, "STUDY")
	}
	// The caller's original query must be untouched (Find writes into a copy).
	if _, present := query.GetString(dicom.TagQueryRetrieveLevel); present {
		t.Error("Find mutated the caller's query dataset; it must write into a copy")
	}
}

// TestFindYieldsPendingThenTerminal drives the iterator over two Pending matches then Success:
// it yields two (Pending, ds) pairs and one terminal (Success, nil).
func TestFindYieldsPendingThenTerminal(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	match2 := dicom.NewDataSet()
	match2.SetString(dicom.TagStudyInstanceUID, "1.2.3.2")

	addr, _ := startFindSCP(t, []findResponse{
		{status: StatusFindPending.Code, identifier: match1},
		{status: StatusFindPending.Code, identifier: match2},
		{status: StatusFindSuccess.Code},
	})
	assoc, ctx, cancel := dialFindSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
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
	if !terminalSeen {
		t.Fatal("iterator never yielded a terminal status")
	}
	if !terminal.IsSuccess() {
		t.Errorf("terminal status = %s, want Success", terminal)
	}
}

// TestFindBreakSendsCancel verifies that breaking out of the range loop after the first match
// sends a C-CANCEL-RQ for the operation's Message ID.
func TestFindBreakSendsCancel(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	match2 := dicom.NewDataSet()
	match2.SetString(dicom.TagStudyInstanceUID, "1.2.3.2")

	addr, obs := startFindSCP(t, []findResponse{
		{status: StatusFindPending.Code, identifier: match1},
		{status: StatusFindPending.Code, identifier: match2},
		{status: StatusFindSuccess.Code},
	})
	assoc, ctx, cancel := dialFindSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	count := 0
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		if status.IsPending() {
			count++
			break // stop after the first match
		}
	}
	// Give the SCP a moment to observe the cancel before releasing.
	time.Sleep(300 * time.Millisecond)
	_ = assoc.Abort(ctx)

	if count != 1 {
		t.Errorf("consumed %d pending matches before break, want 1", count)
	}
	got := obs.snapshot()
	if !got.cancelSeen {
		t.Fatal("breaking the range loop did not send a C-CANCEL-RQ")
	}
	if got.cancelMsgID != got.requestCmd.MessageID {
		t.Errorf("C-CANCEL Message ID Being Responded To = %d, want the request's Message ID %d",
			got.cancelMsgID, got.requestCmd.MessageID)
	}
}

// TestFindTransportFaultSetsLastError verifies that a peer aborting mid-query ends the iterator
// and surfaces the fault via LastError(), not a panic, and not laundered into Success.
func TestFindTransportFaultSetsLastError(t *testing.T) {
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
			CalledAETitle: "FINDSCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(studyRootFindSOPClass),
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			_ = nc.Close()
			return
		}
		// Read the C-FIND-RQ, then abort instead of answering.
		_, _, _, _ = receiveMessage(ctx, acc.Conn(), acc.Machine(), newMessageReassembler(dicom.ExplicitVRLittleEndian))
		_ = acc.Abort(ctx)
	}()

	assoc, ctx, cancel := dialFindSCU(t, ln.Addr().String())
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var sawSuccess bool
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		if status.IsSuccess() {
			sawSuccess = true
		}
	}
	if sawSuccess {
		t.Error("a transport fault was laundered into a Success status")
	}
	if assoc.LastError() == nil {
		t.Fatal("Find LastError() = nil after a mid-query abort, want a typed transport error")
	}
}

// TestFindOnReleasedAssociation is the DIMSE-017 carry-forward: Find on a released association
// yields a single terminal failure status and sets a typed error, never panicking.
func TestFindOnReleasedAssociation(t *testing.T) {
	addr := startEchoAcceptor(t)
	ae, _ := NewAE(AETitle("FINDSCU"))
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
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var statuses []Status
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 {
		t.Fatalf("Find on released association yielded %d statuses, want 1 terminal failure", len(statuses))
	}
	if !statuses[0].IsFailure() {
		t.Errorf("terminal status = %s, want a Failure category", statuses[0])
	}
	var assocErr *AssociationError
	if !errors.As(assoc.LastError(), &assocErr) {
		t.Errorf("LastError() = %T, want *AssociationError", assoc.LastError())
	}
}

// TestFindNoMatchingContext verifies Find fails closed when no Q/R context was negotiated: it
// yields one terminal failure and sets LastError, transmitting no C-FIND-RQ.
func TestFindNoMatchingContext(t *testing.T) {
	addr := startEchoAcceptor(t) // Verification only
	ae, _ := NewAE(AETitle("FINDSCU"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Negotiate only Verification, so no FIND context is accepted.
	assoc, err := ae.Associate(ctx, addr, AETitle("ACCEPTOR"), VerificationContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var statuses []Status
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Find with no FIND context yielded %v, want one terminal failure", statuses)
	}
	if assoc.LastError() == nil {
		t.Error("LastError() = nil, want a typed pre-flight error")
	}
}

// TestFindOnNilAssociation verifies Find on a nil *Association yields a single terminal failure and
// never panics (DIMSE-017). LastError() is nil-safe and returns nil (no association to carry it).
func TestFindOnNilAssociation(t *testing.T) {
	var a *Association
	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var statuses []Status
	for status := range a.Find(context.Background(), query, QueryLevelStudy) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Find on nil association yielded %v, want one terminal failure", statuses)
	}
	if a.LastError() != nil {
		t.Errorf("LastError() on nil association = %v, want nil", a.LastError())
	}
}

// TestFindNilQueryWritesLevel verifies a nil query is treated as an empty identifier carrying only
// the Query/Retrieve Level — Find does not panic on a nil dataset, and (0008,0052) still reaches
// the wire (DIMSE-015 holds even with no caller keys).
func TestFindNilQueryWritesLevel(t *testing.T) {
	addr, obs := startFindSCP(t, []findResponse{{status: StatusFindSuccess.Code}})
	assoc, ctx, cancel := dialFindSCU(t, addr)
	defer cancel()

	for range assoc.Find(ctx, nil, QueryLevelSeries) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.err != nil {
		t.Fatalf("SCP error: %v", got.err)
	}
	if got.identifier == nil {
		t.Fatal("SCP received no identifier dataset for a nil query")
	}
	level, ok := got.identifier.GetString(dicom.TagQueryRetrieveLevel)
	if !ok || level != "SERIES" {
		t.Errorf("Query/Retrieve Level = %q (present=%v), want %q", level, ok, "SERIES")
	}
}

// TestFindContextCancelSendsCancel verifies that cancelling ctx while the iterator is blocked
// awaiting the next response sends a C-CANCEL-RQ for the operation's Message ID (PRD §9.4), then
// ends the iterator. The mock SCP sends one Pending, never the terminal, so the SCU blocks until
// the test cancels ctx.
func TestFindContextCancelSendsCancel(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")

	// Only one Pending and then nothing: the SCU blocks on the second read until ctx is cancelled.
	addr, obs := startFindSCP(t, []findResponse{
		{status: StatusFindPending.Code, identifier: match1},
	})
	assoc, _, dialCancel := dialFindSCU(t, addr)
	defer dialCancel()

	opCtx, opCancel := context.WithCancel(context.Background())

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	seen := 0
	for status := range assoc.Find(opCtx, query, QueryLevelStudy) {
		if status.IsPending() {
			seen++
			// Cancel the operation context; the iterator's next read unblocks with ctx.Canceled and
			// must send a C-CANCEL before ending.
			opCancel()
		}
	}
	opCancel()

	// The SCP's poll-after-Pending observes the C-CANCEL. Give it a moment, then confirm.
	time.Sleep(300 * time.Millisecond)
	_ = assoc.Abort(context.Background())

	if seen != 1 {
		t.Errorf("consumed %d pending matches, want 1", seen)
	}
	got := obs.snapshot()
	if !got.cancelSeen {
		t.Fatal("cancelling ctx mid-query did not send a C-CANCEL-RQ")
	}
	if got.cancelMsgID != got.requestCmd.MessageID {
		t.Errorf("C-CANCEL Message ID Being Responded To = %d, want %d", got.cancelMsgID, got.requestCmd.MessageID)
	}
}

// TestFindRejectsNonFindModel verifies WithQueryModel naming a MOVE/GET (non-FIND) SOP Class fails
// closed: Find yields one terminal failure and sets a typed error, transmitting no C-FIND-RQ whose
// Affected SOP Class is not a query information model.
func TestFindRejectsNonFindModel(t *testing.T) {
	addr, obs := startFindSCP(t, []findResponse{{status: StatusFindSuccess.Code}})
	assoc, ctx, cancel := dialFindSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var statuses []Status
	for status := range assoc.Find(ctx, query, QueryLevelStudy, WithQueryModel(studyRootMoveSOPClass)) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Find with a MOVE model yielded %v, want one terminal failure", statuses)
	}
	var ve *ValidationError
	if !errors.As(assoc.LastError(), &ve) {
		t.Errorf("LastError() = %T, want *ValidationError", assoc.LastError())
	}
	_ = assoc.Abort(ctx)

	// The SCP must never have received a C-FIND-RQ (nothing was transmitted).
	got := obs.snapshot()
	if got.identifier != nil {
		t.Error("a C-FIND-RQ was transmitted for a non-FIND model; want fail-closed before any wire I/O")
	}
}

// TestFindRejectsUnknownLevel verifies a QueryLevel outside the four defined constants fails closed
// rather than writing "UNKNOWN" into (0008,0052) and sending a standards-invalid C-FIND.
func TestFindRejectsUnknownLevel(t *testing.T) {
	addr, obs := startFindSCP(t, []findResponse{{status: StatusFindSuccess.Code}})
	assoc, ctx, cancel := dialFindSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	const bogusLevel = QueryLevel(99)
	var statuses []Status
	for status := range assoc.Find(ctx, query, bogusLevel) {
		statuses = append(statuses, status)
	}
	if len(statuses) != 1 || !statuses[0].IsFailure() {
		t.Fatalf("Find with an unknown level yielded %v, want one terminal failure", statuses)
	}
	var ve *ValidationError
	if !errors.As(assoc.LastError(), &ve) {
		t.Errorf("LastError() = %T, want *ValidationError", assoc.LastError())
	}
	_ = assoc.Abort(ctx)

	got := obs.snapshot()
	if got.identifier != nil {
		t.Error("a C-FIND-RQ was transmitted for an unknown level; want fail-closed before any wire I/O")
	}
}

// TestFindStudyLevelDefaultsToStudyRoot verifies that a study-level Find with no WithQueryModel
// sends the Study Root FIND Information Model SOP Class (the default for non-patient levels).
func TestFindStudyLevelDefaultsToStudyRoot(t *testing.T) {
	addr, obs := startFindSCP(t, []findResponse{{status: StatusFindSuccess.Code}})
	assoc, ctx, cancel := dialFindSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	for range assoc.Find(ctx, query, QueryLevelStudy) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.requestCmd.AffectedSOPClassUID != dicom.UID(studyRootFindSOPClass) {
		t.Errorf("C-FIND-RQ Affected SOP Class = %q, want Study Root FIND %q",
			got.requestCmd.AffectedSOPClassUID, studyRootFindSOPClass)
	}
}

// startPatientRootFindSCP is startFindSCP negotiating the Patient Root FIND model, used to prove a
// patient-level Find defaults to Patient Root (Study Root has no Patient level).
func startPatientRootFindSCP(t *testing.T, responses []findResponse) (string, *findSCPObservation) {
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
			CalledAETitle: "FINDSCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(patientRootFindSOPClass),
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

// TestFindPatientLevelDefaultsToPatientRoot verifies a patient-level Find with no WithQueryModel
// negotiates and sends the Patient Root FIND model against a Patient-Root-only peer (the Codex
// round-3 finding: a patient-level query is only valid in the Patient Root model).
func TestFindPatientLevelDefaultsToPatientRoot(t *testing.T) {
	addr, obs := startPatientRootFindSCP(t, []findResponse{{status: StatusFindSuccess.Code}})
	assoc, ctx, cancel := dialFindSCU(t, addr)
	defer cancel()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagPatientID, "PID-1")

	for range assoc.Find(ctx, query, QueryLevelPatient) {
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError = %v, want nil (Patient Root context was negotiated)", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.err != nil {
		t.Fatalf("SCP error: %v", got.err)
	}
	if got.requestCmd.AffectedSOPClassUID != dicom.UID(patientRootFindSOPClass) {
		t.Errorf("C-FIND-RQ Affected SOP Class = %q, want Patient Root FIND %q",
			got.requestCmd.AffectedSOPClassUID, patientRootFindSOPClass)
	}
	level, ok := got.identifier.GetString(dicom.TagQueryRetrieveLevel)
	if !ok || level != "PATIENT" {
		t.Errorf("Query/Retrieve Level = %q (present=%v), want PATIENT", level, ok)
	}
}

// TestFindDIMSETimeoutSetsLastError verifies that when the association's DIMSE timeout fires while
// the caller's ctx is still live (a stalled peer), the iterator ends with a terminal failure and a
// non-nil LastError — never a silent end mistaken for cancellation.
func TestFindDIMSETimeoutSetsLastError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: "FINDSCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(studyRootFindSOPClass),
				TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
			}},
		})
		if perr != nil {
			_ = nc.Close()
			return
		}
		// Read the C-FIND-RQ, then go silent — never answer, simulating a stalled peer.
		_, _, _, _ = receiveMessage(ctx, acc.Conn(), acc.Machine(), newMessageReassembler(dicom.ExplicitVRLittleEndian))
		<-stop
	}()

	// A SHORT DIMSE timeout, with a generous (still-live) parent ctx, so the DIMSE timeout — not the
	// parent — fires while awaiting the response.
	ae, err := NewAE(AETitle("FINDSCU"), WithDIMSETimeout(400*time.Millisecond))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, ln.Addr().String(), AETitle("FINDSCP"), QueryRetrieveContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	query := dicom.NewDataSet()
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var sawSuccess bool
	var terminalSeen bool
	for status := range assoc.Find(ctx, query, QueryLevelStudy) {
		terminalSeen = true
		if status.IsSuccess() {
			sawSuccess = true
		}
	}
	if sawSuccess {
		t.Error("a DIMSE timeout was laundered into a Success status")
	}
	if !terminalSeen {
		t.Error("a stalled C-FIND ended without yielding any terminal status")
	}
	if assoc.LastError() == nil {
		t.Fatal("LastError() = nil after a DIMSE timeout; a stalled peer must surface a fault")
	}
}
