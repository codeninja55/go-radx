package dimse

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// The SCP serve paths must enforce the per-model Query/Retrieve Level table (validateModelLevel) on
// the INBOUND request, symmetric with the SCU preflight. queryLevelFromIdentifier accepts any known
// keyword, so without the inbound check a peer could send a Composite Instance Root C-GET/C-MOVE at
// STUDY, or a Patient/Study Only C-FIND at SERIES, and the application handler would run instead of
// the malformed-identifier failure being returned. The production SCU helpers (Get/Move/Find) refuse
// such combinations at their own preflight, so these regressions hand-build the RQ to drive the SCP
// boundary directly (the pattern sendRawFindRQ established), exercising the extended models the
// validateModelLevel table newly admits. The failure status the SCP returns for a malformed identifier
// is 0xA900 "Identifier Does Not Match SOP Class" (PS3.4 Annex C, PS3.7).

// sendRawQRRequest associates to addr (with calledAE) proposing contexts, hand-builds a
// C-FIND/C-GET/C-MOVE-RQ for model with an identifier carrying levelKeyword, and returns the first
// response status read. It bypasses the SCU preflight (which would itself refuse an invalid
// model/level), so it can drive the SCP-side validateModelLevel boundary for both invalid and valid
// combinations. command is one of CommandCFindRQ/CommandCGetRQ/CommandCMoveRQ; for a C-MOVE moveDest
// names the destination AE (empty for the other commands).
func sendRawQRRequest(t *testing.T, addr string, calledAE AETitle, contexts []PresentationContext,
	command CommandField, model dicom.SOPClassUID, levelKeyword string, moveDest AETitle) Status {
	t.Helper()
	ae, err := NewAE(AETitle("QRSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, addr, calledAE, contexts)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Abort(ctx) }()

	pcID, ts, ok := assoc.contextForQuery(model)
	if !ok {
		t.Fatalf("no accepted presentation context for model %s", model)
	}
	conn := assoc.requestor.Conn()
	m := assoc.requestor.Machine()

	identifier := dicom.NewDataSet()
	identifier.SetString(dicom.TagStudyInstanceUID, "1.2.3")
	identifier.SetString(dicom.TagQueryRetrieveLevel, levelKeyword)

	rq := CommandSet{
		CommandField:        command,
		MessageID:           1,
		AffectedSOPClassUID: dicom.UID(model),
		HasPriority:         true,
		Priority:            PriorityMedium,
		CommandDataSetType:  CommandDataSetPresent,
	}
	var svc ServiceClass
	var rspCommand CommandField
	switch command {
	case CommandCFindRQ:
		svc, rspCommand = serviceClassForQueryModel(model), CommandCFindRSP
	case CommandCGetRQ:
		svc, rspCommand = ServiceClassGet, CommandCGetRSP
	case CommandCMoveRQ:
		svc, rspCommand = ServiceClassMove, CommandCMoveRSP
		rq.MoveDestination = moveDest
	default:
		t.Fatalf("unsupported command %#04x", uint16(command))
	}

	if err := sendMessage(ctx, conn, m, pcID, rq, identifier, ts, assoc.sendCap()); err != nil {
		t.Fatalf("send raw RQ: %v", err)
	}
	rsp, _, _, err := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if err != nil {
		t.Fatalf("read RSP: %v", err)
	}
	if rsp.CommandField != rspCommand || !rsp.HasStatus {
		t.Fatalf("response is not a %#04x with a status: %+v", uint16(rspCommand), rsp)
	}
	return NewStatus(rsp.Status, svc)
}

// TestServeFindRejectsInvalidModelLevel is the inbound-boundary regression for C-FIND: a Patient/Study
// Only FIND carrying SERIES (a level the model does not admit, PS3.4 C.6.3) must be refused with the
// 0xA900 malformed-identifier Failure BEFORE the FindHandler runs. A valid level (STUDY) still works.
func TestServeFindRejectsInvalidModelLevel(t *testing.T) {
	handler := &cannedFindHandler{results: []findResponse{{status: StatusFindSuccess.Code}}}
	addr := startExtendedFindServer(t, &findScpHandler{h: handler})
	contexts := ExtendedQueryRetrieveContexts()

	// Invalid: Patient/Study Only admits only PATIENT/STUDY, not SERIES.
	status := sendRawQRRequest(t, addr, AETitle("RADX-SCP"), contexts, CommandCFindRQ, patientStudyOnlyFindSOPClass, "SERIES", "")
	if !status.IsFailure() || status.Code != 0xA900 {
		t.Errorf("invalid model/level C-FIND got terminal %s, want 0xA900 Failure (fail-closed)", status)
	}
	handler.mu.Lock()
	called := handler.calledQuery != nil
	handler.mu.Unlock()
	if called {
		t.Error("the FindHandler was invoked for an invalid model/level C-FIND; want fail-closed before dispatch")
	}

	// Valid: STUDY is admissible for Patient/Study Only — the handler must run and the SCP returns Success.
	valid := sendRawQRRequest(t, addr, AETitle("RADX-SCP"), contexts, CommandCFindRQ, patientStudyOnlyFindSOPClass, "STUDY", "")
	if !valid.IsSuccess() {
		t.Errorf("valid model/level C-FIND got terminal %s, want Success", valid)
	}
	handler.mu.Lock()
	calledValid := handler.calledQuery != nil
	handler.mu.Unlock()
	if !calledValid {
		t.Error("the FindHandler was not invoked for a valid model/level C-FIND")
	}
}

// TestServeGetRejectsInvalidModelLevel is the inbound-boundary regression for C-GET: a Composite
// Instance Root GET carrying STUDY (the model admits only IMAGE/FRAME, PS3.4 C.6.5) must be refused
// with the 0xA900 malformed-identifier Failure BEFORE the GetHandler runs. A valid level (IMAGE) still
// reaches the handler (which here yields zero matches → a terminal Success with no sub-operations).
func TestServeGetRejectsInvalidModelLevel(t *testing.T) {
	handler := &gettingHandler{} // zero matches: a valid request terminates with Success, no stores
	addr := startExtendedGetServer(t, handler)
	contexts := extendedQueryRetrieveWithStorageContexts()

	// Invalid: Composite Instance Root admits only IMAGE/FRAME, not STUDY.
	status := sendRawQRRequest(t, addr, AETitle("RADX-GETSCP"), contexts, CommandCGetRQ, compositeInstanceRootGetSOPClass, "STUDY", "")
	if !status.IsFailure() || status.Code != 0xA900 {
		t.Errorf("invalid model/level C-GET got terminal %s, want 0xA900 Failure (fail-closed)", status)
	}
	handler.mu.Lock()
	called := handler.calledQuery != nil
	handler.mu.Unlock()
	if called {
		t.Error("the GetHandler was invoked for an invalid model/level C-GET; want fail-closed before dispatch")
	}

	// Valid: IMAGE is admissible for Composite Instance Root — the handler runs (zero matches → Success).
	valid := sendRawQRRequest(t, addr, AETitle("RADX-GETSCP"), contexts, CommandCGetRQ, compositeInstanceRootGetSOPClass, "IMAGE", "")
	if !valid.IsSuccess() {
		t.Errorf("valid model/level C-GET got terminal %s, want Success", valid)
	}
	handler.mu.Lock()
	calledValid := handler.calledQuery != nil
	handler.mu.Unlock()
	if !calledValid {
		t.Error("the GetHandler was not invoked for a valid model/level C-GET")
	}
}

// TestServeMoveRejectsInvalidModelLevel is the inbound-boundary regression for C-MOVE: a Composite
// Instance Root MOVE carrying STUDY (the model admits only IMAGE/FRAME, PS3.4 C.6.5) must be refused
// with the 0xA900 malformed-identifier Failure BEFORE the MoveHandler runs and before the Move
// Destination is even resolved. A valid level (IMAGE) reaches the handler.
func TestServeMoveRejectsInvalidModelLevel(t *testing.T) {
	destTitle, destAddr, _ := startDestinationSCP(t)
	handler := &movingFindHandler{} // zero matches: a valid request terminates with Success
	addr := startMoveServerExtended(t, handler, map[AETitle]string{destTitle: destAddr})
	contexts := ExtendedQueryRetrieveContexts()

	// Invalid: Composite Instance Root admits only IMAGE/FRAME, not STUDY.
	status := sendRawQRRequest(t, addr, AETitle("RADX-MOVESCP"), contexts, CommandCMoveRQ, compositeInstanceRootMoveSOPClass, "STUDY", destTitle)
	if !status.IsFailure() || status.Code != 0xA900 {
		t.Errorf("invalid model/level C-MOVE got terminal %s, want 0xA900 Failure (fail-closed)", status)
	}
	handler.mu.Lock()
	called := handler.calledQuery != nil
	handler.mu.Unlock()
	if called {
		t.Error("the MoveHandler was invoked for an invalid model/level C-MOVE; want fail-closed before dispatch")
	}

	// Valid: IMAGE is admissible for Composite Instance Root — the handler runs (zero matches → Success).
	valid := sendRawQRRequest(t, addr, AETitle("RADX-MOVESCP"), contexts, CommandCMoveRQ, compositeInstanceRootMoveSOPClass, "IMAGE", destTitle)
	if !valid.IsSuccess() {
		t.Errorf("valid model/level C-MOVE got terminal %s, want Success", valid)
	}
	handler.mu.Lock()
	calledValid := handler.calledQuery != nil
	handler.mu.Unlock()
	if !calledValid {
		t.Error("the MoveHandler was not invoked for a valid model/level C-MOVE")
	}
}

// startMoveServerExtended stands up a C-MOVE SCP advertising the EXTENDED Q/R contexts (so the
// Composite Instance Root MOVE model negotiates) with the supplied destination table, on loopback.
func startMoveServerExtended(t *testing.T, h any, dests map[AETitle]string) string {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-MOVESCP"))
	if err != nil {
		t.Fatalf("NewAE move SCP: %v", err)
	}
	srv := NewServer(ae, ExtendedQueryRetrieveContexts(), h, WithMoveDestinations(dests))
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
