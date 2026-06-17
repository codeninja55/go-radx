package dimse

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// nServiceFullHandler is a test SCP that implements ALL six DIMSE-N capabilities and records the
// request each serve-path handed it. It exercises the N-CREATE/N-SET/N-ACTION/N-EVENT-REPORT
// dispatch arms — the ones the high-level Association SCU API does not drive — which are reached
// here by sending raw command sets over the loopback association.
type nServiceFullHandler struct {
	mu sync.Mutex

	createReq      *NRequest
	setReq         *NRequest
	actionReq      *NRequest
	eventReportReq *NRequest

	createStatus   Status
	createInstance dicom.UID
	setStatus      Status
	actionStatus   Status
	eventStatus    Status
}

func (h *nServiceFullHandler) NCreate(_ context.Context, req NRequest) (Status, dicom.UID) {
	h.mu.Lock()
	r := req
	h.createReq = &r
	h.mu.Unlock()
	return h.createStatus, h.createInstance
}

func (h *nServiceFullHandler) NSet(_ context.Context, req NRequest) Status {
	h.mu.Lock()
	r := req
	h.setReq = &r
	h.mu.Unlock()
	return h.setStatus
}

func (h *nServiceFullHandler) NAction(_ context.Context, req NRequest) Status {
	h.mu.Lock()
	r := req
	h.actionReq = &r
	h.mu.Unlock()
	return h.actionStatus
}

func (h *nServiceFullHandler) NEventReport(_ context.Context, req NRequest) Status {
	h.mu.Lock()
	r := req
	h.eventReportReq = &r
	h.mu.Unlock()
	return h.eventStatus
}

func (h *nServiceFullHandler) snapshotFull() (create, set, action, event *NRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.createReq, h.setReq, h.actionReq, h.eventReportReq
}

// sendRawNRequest sends a hand-built N-service command set (optionally with a data set) over an
// already-established association's presentation context, then reads back the single RSP. It is the
// negotiation-respecting way to drive the N-CREATE/N-SET/N-ACTION/N-EVENT-REPORT serve paths the
// high-level Association SCU API does not expose.
func sendRawNRequest(t *testing.T, assoc *Association, ctx context.Context, rq CommandSet, ds *dicom.DataSet) CommandSet {
	t.Helper()
	pcID, ts, ok := assoc.contextForQuery(displaySystemSOPClass)
	if !ok {
		t.Fatal("no accepted Display System presentation context")
	}
	conn := assoc.requestor.Conn()
	machine := assoc.requestor.Machine()
	if ds != nil {
		rq.CommandDataSetType = CommandDataSetPresent
		sendCap := MaxPDULength(assoc.requestor.PeerMaxPDULength()).SendCap(defaultMaxPDULength)
		if err := sendMessage(ctx, conn, machine, pcID, rq, ds, ts, sendCap); err != nil {
			t.Fatalf("sendMessage: %v", err)
		}
	} else {
		rq.CommandDataSetType = CommandDataSetNotPresent
		if err := sendCommand(ctx, conn, machine, pcID, rq); err != nil {
			t.Fatalf("sendCommand: %v", err)
		}
	}
	rsp, _, _, err := receiveMessage(ctx, conn, machine, newMessageReassembler(ts))
	if err != nil {
		t.Fatalf("receiveMessage (RSP): %v", err)
	}
	return rsp
}

// TestServeNCreateLoopback drives an N-CREATE through the dispatch: the SCP returns Success and a
// freshly-assigned SOP Instance UID, which the N-CREATE-RSP must echo. It also confirms the handler
// received the Affected pair and the new object's attributes as the request data set.
func TestServeNCreateLoopback(t *testing.T) {
	const assigned = dicom.UID("1.2.840.10008.5.1.1.40.1.99")
	h := &nServiceFullHandler{createStatus: StatusNSuccess, createInstance: assigned}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	attrs := dicom.NewDataSet()
	attrs.SetString(dicom.NewTag(0x0008, 0x0070), "RADX")
	rq := CommandSet{
		CommandField:        CommandNCreateRQ,
		MessageID:           7,
		AffectedSOPClassUID: dicom.UID(displaySystemSOPClass),
		// No proposed instance UID: the SCP assigns one.
	}
	rsp := sendRawNRequest(t, assoc, ctx, rq, attrs)

	if rsp.CommandField != CommandNCreateRSP {
		t.Errorf("RSP command field = %#04x, want N-CREATE-RSP", uint16(rsp.CommandField))
	}
	if !rsp.HasStatus || rsp.Status != StatusNSuccess.Code {
		t.Errorf("RSP status = %#04x (has=%v), want Success", rsp.Status, rsp.HasStatus)
	}
	if rsp.AffectedSOPInstanceUID != assigned {
		t.Errorf("RSP Affected SOP Instance UID = %q, want the SCP-assigned %q", rsp.AffectedSOPInstanceUID, assigned)
	}
	if rsp.AffectedSOPClassUID != dicom.UID(displaySystemSOPClass) {
		t.Errorf("RSP Affected SOP Class UID = %q, want %q", rsp.AffectedSOPClassUID, displaySystemSOPClass)
	}
	_ = assoc.Release(ctx)

	create, _, _, _ := h.snapshotFull()
	if create == nil {
		t.Fatal("handler did not observe the N-CREATE request")
	}
	if create.AffectedSOPClassUID != dicom.UID(displaySystemSOPClass) {
		t.Errorf("handler Affected SOP Class UID = %q, want %q", create.AffectedSOPClassUID, displaySystemSOPClass)
	}
	if create.DataSet == nil {
		t.Fatal("handler N-CREATE request carried no data set")
	}
	if man, ok := create.DataSet.GetString(dicom.NewTag(0x0008, 0x0070)); !ok || man != "RADX" {
		t.Errorf("handler data set Manufacturer = %q (present=%v), want RADX", man, ok)
	}
}

// TestServeNCreateProposedInstanceEchoed confirms that when the SCU proposes an Affected SOP
// Instance UID and the handler returns an empty UID, the dispatch echoes the SCU's proposed UID
// (the serveNCreateMessage fallback branch).
func TestServeNCreateProposedInstanceEchoed(t *testing.T) {
	const proposed = dicom.UID("1.2.840.10008.5.1.1.40.1.42")
	h := &nServiceFullHandler{createStatus: StatusNSuccess} // returns "" instance UID
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	rq := CommandSet{
		CommandField:           CommandNCreateRQ,
		MessageID:              8,
		AffectedSOPClassUID:    dicom.UID(displaySystemSOPClass),
		AffectedSOPInstanceUID: proposed,
	}
	rsp := sendRawNRequest(t, assoc, ctx, rq, nil)
	if rsp.AffectedSOPInstanceUID != proposed {
		t.Errorf("RSP Affected SOP Instance UID = %q, want the SCU-proposed %q (handler assigned none)", rsp.AffectedSOPInstanceUID, proposed)
	}
	_ = assoc.Release(ctx)
}

// TestServeNSetLoopback drives an N-SET through the dispatch and asserts the RSP echoes the
// Requested pair into the Affected pair and carries the handler's status, and the handler received
// the updated attributes as the request data set.
func TestServeNSetLoopback(t *testing.T) {
	h := &nServiceFullHandler{setStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	attrs := dicom.NewDataSet()
	attrs.SetString(dicom.NewTag(0x0040, 0x0252), "IN PROGRESS") // Performed Procedure Step Status
	rq := CommandSet{
		CommandField:            CommandNSetRQ,
		MessageID:               11,
		RequestedSOPClassUID:    dicom.UID(displaySystemSOPClass),
		RequestedSOPInstanceUID: displaySystemInstance,
	}
	rsp := sendRawNRequest(t, assoc, ctx, rq, attrs)

	if rsp.CommandField != CommandNSetRSP {
		t.Errorf("RSP command field = %#04x, want N-SET-RSP", uint16(rsp.CommandField))
	}
	if !rsp.HasStatus || rsp.Status != StatusNSuccess.Code {
		t.Errorf("RSP status = %#04x, want Success", rsp.Status)
	}
	if rsp.AffectedSOPInstanceUID != displaySystemInstance {
		t.Errorf("RSP Affected SOP Instance UID = %q, want the Requested %q echoed", rsp.AffectedSOPInstanceUID, displaySystemInstance)
	}
	_ = assoc.Release(ctx)

	_, set, _, _ := h.snapshotFull()
	if set == nil {
		t.Fatal("handler did not observe the N-SET request")
	}
	if set.DataSet == nil {
		t.Fatal("handler N-SET request carried no data set")
	}
}

// TestServeNActionLoopback drives an N-ACTION through the dispatch and asserts the RSP echoes the
// Requested pair and the Action Type ID, and the handler saw the same Action Type ID.
func TestServeNActionLoopback(t *testing.T) {
	h := &nServiceFullHandler{actionStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	rq := CommandSet{
		CommandField:            CommandNActionRQ,
		MessageID:               12,
		RequestedSOPClassUID:    dicom.UID(displaySystemSOPClass),
		RequestedSOPInstanceUID: displaySystemInstance,
		HasActionTypeID:         true,
		ActionTypeID:            1,
	}
	rsp := sendRawNRequest(t, assoc, ctx, rq, nil)

	if rsp.CommandField != CommandNActionRSP {
		t.Errorf("RSP command field = %#04x, want N-ACTION-RSP", uint16(rsp.CommandField))
	}
	if !rsp.HasActionTypeID || rsp.ActionTypeID != 1 {
		t.Errorf("RSP Action Type ID = %d (has=%v), want 1", rsp.ActionTypeID, rsp.HasActionTypeID)
	}
	if rsp.AffectedSOPInstanceUID != displaySystemInstance {
		t.Errorf("RSP Affected SOP Instance UID = %q, want the Requested %q echoed", rsp.AffectedSOPInstanceUID, displaySystemInstance)
	}
	_ = assoc.Release(ctx)

	_, _, action, _ := h.snapshotFull()
	if action == nil {
		t.Fatal("handler did not observe the N-ACTION request")
	}
	if !action.HasActionTypeID || action.ActionTypeID != 1 {
		t.Errorf("handler Action Type ID = %d (has=%v), want 1", action.ActionTypeID, action.HasActionTypeID)
	}
}

// TestServeNEventReportLoopback drives an N-EVENT-REPORT through the dispatch and asserts the RSP
// echoes the Affected pair and the Event Type ID.
func TestServeNEventReportLoopback(t *testing.T) {
	h := &nServiceFullHandler{eventStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	rq := CommandSet{
		CommandField:           CommandNEventReportRQ,
		MessageID:              13,
		AffectedSOPClassUID:    dicom.UID(displaySystemSOPClass),
		AffectedSOPInstanceUID: displaySystemInstance,
		HasEventTypeID:         true,
		EventTypeID:            2,
	}
	rsp := sendRawNRequest(t, assoc, ctx, rq, nil)

	if rsp.CommandField != CommandNEventReportRSP {
		t.Errorf("RSP command field = %#04x, want N-EVENT-REPORT-RSP", uint16(rsp.CommandField))
	}
	if !rsp.HasEventTypeID || rsp.EventTypeID != 2 {
		t.Errorf("RSP Event Type ID = %d (has=%v), want 2", rsp.EventTypeID, rsp.HasEventTypeID)
	}
	if rsp.AffectedSOPInstanceUID != displaySystemInstance {
		t.Errorf("RSP Affected SOP Instance UID = %q, want %q echoed", rsp.AffectedSOPInstanceUID, displaySystemInstance)
	}
	_ = assoc.Release(ctx)

	_, _, _, event := h.snapshotFull()
	if event == nil {
		t.Fatal("handler did not observe the N-EVENT-REPORT request")
	}
	if !event.HasEventTypeID || event.EventTypeID != 2 {
		t.Errorf("handler Event Type ID = %d (has=%v), want 2", event.EventTypeID, event.HasEventTypeID)
	}
}

// TestRefuseUnsupportedNEchoesTypeIDs confirms that when an N-ACTION or N-EVENT-REPORT reaches an
// SCP with no matching capability, refuseUnsupportedN answers StatusSOPClassNotSupported and echoes
// the request's Action/Event Type ID so the peer can correlate the refusal — the type-ID echo
// branches the supported paths never exercise.
func TestRefuseUnsupportedNEchoesTypeIDs(t *testing.T) {
	// A handler implementing only N-GET/N-DELETE: an N-ACTION and an N-EVENT-REPORT must be refused.
	h := &nServiceHandler{getStatus: StatusNSuccess, deleteStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	action := CommandSet{
		CommandField:            CommandNActionRQ,
		MessageID:               21,
		RequestedSOPClassUID:    dicom.UID(displaySystemSOPClass),
		RequestedSOPInstanceUID: displaySystemInstance,
		HasActionTypeID:         true,
		ActionTypeID:            5,
	}
	rsp := sendRawNRequest(t, assoc, ctx, action, nil)
	if rsp.CommandField != CommandNActionRSP {
		t.Errorf("refused N-ACTION RSP command field = %#04x, want N-ACTION-RSP", uint16(rsp.CommandField))
	}
	if rsp.Status != StatusSOPClassNotSupported.Code {
		t.Errorf("refused N-ACTION status = %#04x, want SOP Class Not Supported (0x0122)", rsp.Status)
	}
	if !rsp.HasActionTypeID || rsp.ActionTypeID != 5 {
		t.Errorf("refused N-ACTION RSP Action Type ID = %d (has=%v), want 5 echoed", rsp.ActionTypeID, rsp.HasActionTypeID)
	}
	if rsp.AffectedSOPInstanceUID != displaySystemInstance {
		t.Errorf("refused N-ACTION RSP Affected SOP Instance UID = %q, want the Requested %q echoed", rsp.AffectedSOPInstanceUID, displaySystemInstance)
	}

	event := CommandSet{
		CommandField:           CommandNEventReportRQ,
		MessageID:              22,
		AffectedSOPClassUID:    dicom.UID(displaySystemSOPClass),
		AffectedSOPInstanceUID: displaySystemInstance,
		HasEventTypeID:         true,
		EventTypeID:            3,
	}
	rsp = sendRawNRequest(t, assoc, ctx, event, nil)
	if rsp.CommandField != CommandNEventReportRSP {
		t.Errorf("refused N-EVENT-REPORT RSP command field = %#04x, want N-EVENT-REPORT-RSP", uint16(rsp.CommandField))
	}
	if rsp.Status != StatusSOPClassNotSupported.Code {
		t.Errorf("refused N-EVENT-REPORT status = %#04x, want SOP Class Not Supported (0x0122)", rsp.Status)
	}
	if !rsp.HasEventTypeID || rsp.EventTypeID != 3 {
		t.Errorf("refused N-EVENT-REPORT RSP Event Type ID = %d (has=%v), want 3 echoed", rsp.EventTypeID, rsp.HasEventTypeID)
	}
	_ = assoc.Release(ctx)
}

// TestRefuseUnsupportedNSetCreateRefused confirms N-SET and N-CREATE reaching an SCP without the
// matching capability are refused (the affected/requested-pair echo branches of refuseUnsupportedN,
// without a type ID).
func TestRefuseUnsupportedNSetCreateRefused(t *testing.T) {
	h := &nServiceHandler{getStatus: StatusNSuccess, deleteStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	set := CommandSet{
		CommandField:            CommandNSetRQ,
		MessageID:               31,
		RequestedSOPClassUID:    dicom.UID(displaySystemSOPClass),
		RequestedSOPInstanceUID: displaySystemInstance,
	}
	rsp := sendRawNRequest(t, assoc, ctx, set, nil)
	if rsp.CommandField != CommandNSetRSP || rsp.Status != StatusSOPClassNotSupported.Code {
		t.Errorf("refused N-SET RSP = (field %#04x, status %#04x), want (N-SET-RSP, 0x0122)", uint16(rsp.CommandField), rsp.Status)
	}

	create := CommandSet{
		CommandField:        CommandNCreateRQ,
		MessageID:           32,
		AffectedSOPClassUID: dicom.UID(displaySystemSOPClass),
	}
	rsp = sendRawNRequest(t, assoc, ctx, create, nil)
	if rsp.CommandField != CommandNCreateRSP || rsp.Status != StatusSOPClassNotSupported.Code {
		t.Errorf("refused N-CREATE RSP = (field %#04x, status %#04x), want (N-CREATE-RSP, 0x0122)", uint16(rsp.CommandField), rsp.Status)
	}
	if rsp.AffectedSOPClassUID != dicom.UID(displaySystemSOPClass) {
		t.Errorf("refused N-CREATE RSP Affected SOP Class UID = %q, want %q echoed", rsp.AffectedSOPClassUID, displaySystemSOPClass)
	}
	_ = assoc.Release(ctx)
}

// TestNGetNDeleteOnReleasedAssociation confirms the SCU primitives fail closed with a typed
// AssociationError once the association has been released, before any wire I/O.
func TestNGetNDeleteOnReleasedAssociation(t *testing.T) {
	h := &nServiceHandler{getStatus: StatusNSuccess, deleteStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("Release: %v", err)
	}

	if _, err := assoc.NGet(ctx, displaySystemSOPClass, displaySystemInstance, nil); err == nil {
		t.Error("N-GET on a released association should return a typed error")
	}
	if _, err := assoc.NDelete(ctx, displaySystemSOPClass, displaySystemInstance); err == nil {
		t.Error("N-DELETE on a released association should return a typed error")
	}
}

// TestNGetNDeleteNoMatchingContext confirms the SCU primitives fail closed when no presentation
// context was negotiated for the Requested SOP Class, transmitting no request.
func TestNGetNDeleteNoMatchingContext(t *testing.T) {
	h := &nServiceHandler{getStatus: StatusNSuccess, deleteStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()
	defer func() { _ = assoc.Release(ctx) }()

	// otherNServiceSOPClass was never negotiated on this association.
	if _, err := assoc.NGet(ctx, otherNServiceSOPClass, displaySystemInstance, nil); err == nil {
		t.Error("N-GET with no negotiated context should fail closed")
	}
	if _, err := assoc.NDelete(ctx, otherNServiceSOPClass, displaySystemInstance); err == nil {
		t.Error("N-DELETE with no negotiated context should fail closed")
	}
}

// nMalformedAcceptParams accepts the Display System N-service context, mirroring nServiceContexts.
func nMalformedAcceptParams(called AETitle) acse.AcceptParams {
	return acse.AcceptParams{
		CalledAETitle: string(called),
		MaxPDULength:  16382,
		Supported: []acse.SupportedContext{{
			AbstractSyntax:   string(displaySystemSOPClass),
			TransferSyntaxes: []string{"1.2.840.10008.1.2.1", "1.2.840.10008.1.2"},
		}},
	}
}

// startMalformedNAcceptor serves a single N-service association that replies to whatever it reads
// with the caller-supplied RSP command set, then releases. It drives the NGet/NDelete SCU
// protocol-fault branches (wrong command field, missing Status) the loopback Server never produces.
func startMalformedNAcceptor(t *testing.T, rsp CommandSet) string {
	t.Helper()
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
		defer nc.Close()
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, nMalformedAcceptParams("RADX-NSCP"))
		if perr != nil {
			return
		}
		cmd, pcID, rerr := receiveCommand(ctx, acc.Conn(), acc.Machine())
		if rerr != nil {
			return
		}
		out := rsp
		out.MessageIDBeingRespondedTo = cmd.MessageID
		_ = sendCommand(ctx, acc.Conn(), acc.Machine(), pcID, out)
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String()
}

// TestNGetWrongResponseCommandField confirms NGet faults with a *ProtocolError when the peer answers
// with a command field other than N-GET-RSP.
func TestNGetWrongResponseCommandField(t *testing.T) {
	addr := startMalformedNAcceptor(t, CommandSet{
		CommandField:       CommandNDeleteRSP, // wrong: NGet expects N-GET-RSP
		HasStatus:          true,
		Status:             StatusNSuccess.Code,
		CommandDataSetType: CommandDataSetNotPresent,
	})
	assoc, ctx, cancel := dialNSCU(t, addr)
	defer cancel()
	defer func() { _ = assoc.Release(ctx) }()

	if _, err := assoc.NGet(ctx, displaySystemSOPClass, displaySystemInstance, nil); err == nil {
		t.Error("N-GET should fault on a non-N-GET-RSP response command field")
	}
}

// TestNGetMissingStatus confirms NGet faults with a *ProtocolError when the N-GET-RSP omits the
// mandatory Status element.
func TestNGetMissingStatus(t *testing.T) {
	addr := startMalformedNAcceptor(t, CommandSet{
		CommandField:       CommandNGetRSP,
		HasStatus:          false, // the malformed bit
		CommandDataSetType: CommandDataSetNotPresent,
	})
	assoc, ctx, cancel := dialNSCU(t, addr)
	defer cancel()
	defer func() { _ = assoc.Release(ctx) }()

	if _, err := assoc.NGet(ctx, displaySystemSOPClass, displaySystemInstance, nil); err == nil {
		t.Error("N-GET should fault on an N-GET-RSP missing the Status element")
	}
}

// TestNDeleteWrongResponseCommandField confirms NDelete faults when the peer answers with a command
// field other than N-DELETE-RSP.
func TestNDeleteWrongResponseCommandField(t *testing.T) {
	addr := startMalformedNAcceptor(t, CommandSet{
		CommandField:       CommandNGetRSP, // wrong: NDelete expects N-DELETE-RSP
		HasStatus:          true,
		Status:             StatusNSuccess.Code,
		CommandDataSetType: CommandDataSetNotPresent,
	})
	assoc, ctx, cancel := dialNSCU(t, addr)
	defer cancel()
	defer func() { _ = assoc.Release(ctx) }()

	if _, err := assoc.NDelete(ctx, displaySystemSOPClass, displaySystemInstance); err == nil {
		t.Error("N-DELETE should fault on a non-N-DELETE-RSP response command field")
	}
}

// TestNDeleteMissingStatus confirms NDelete faults when the N-DELETE-RSP omits the mandatory Status.
func TestNDeleteMissingStatus(t *testing.T) {
	addr := startMalformedNAcceptor(t, CommandSet{
		CommandField:       CommandNDeleteRSP,
		HasStatus:          false,
		CommandDataSetType: CommandDataSetNotPresent,
	})
	assoc, ctx, cancel := dialNSCU(t, addr)
	defer cancel()
	defer func() { _ = assoc.Release(ctx) }()

	if _, err := assoc.NDelete(ctx, displaySystemSOPClass, displaySystemInstance); err == nil {
		t.Error("N-DELETE should fault on an N-DELETE-RSP missing the Status element")
	}
}
