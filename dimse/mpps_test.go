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

// mppsObservation records what the mock MPPS SCP saw, for the unit tests to assert against without
// inspecting the wire directly.
type mppsObservation struct {
	mu sync.Mutex

	createCmd CommandSet
	createDS  *dicom.DataSet
	setCmd    CommandSet
	setDS     *dicom.DataSet
	err       error
}

func (o *mppsObservation) snapshot() mppsObservation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return mppsObservation{
		createCmd: o.createCmd,
		createDS:  o.createDS,
		setCmd:    o.setCmd,
		setDS:     o.setDS,
		err:       o.err,
	}
}

// mppsSCPConfig tunes the mock SCP's responses so a test can drive the failure and instance-UID
// assignment paths.
type mppsSCPConfig struct {
	// createStatus is the status the N-CREATE-RSP returns (default MPPS Success).
	createStatus uint16
	// setStatus is the status the N-SET-RSP returns (default MPPS Success).
	setStatus uint16
	// assignInstanceUID, when non-empty, is the Affected SOP Instance UID the SCP assigns in the
	// N-CREATE-RSP, modelling an SCP that allocates the instance UID rather than echoing the SCU's.
	assignInstanceUID dicom.UID
}

// startMPPSSCP listens on loopback, accepts an MPPS association, then serves one N-CREATE-RQ
// followed by one N-SET-RQ, recording each command and data set. It models the dcm4chee MPPS SCP
// behaviour the interop test exercises: the N-CREATE opens the step, the N-SET advances it.
func startMPPSSCP(t *testing.T, cfg mppsSCPConfig) (string, *mppsObservation) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	obs := &mppsObservation{}
	go func() {
		nc, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		conn := dul.NewConn(nc, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		acc, perr := acse.Accept(ctx, conn, acse.AcceptParams{
			CalledAETitle: "MPPSSCP",
			MaxPDULength:  16382,
			Supported: []acse.SupportedContext{{
				AbstractSyntax:   string(modalityPerformedStepSOPClass),
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
		serveCannedMPPS(ctx, acc, cfg, obs)
		_ = acc.ServeRelease(ctx)
	}()
	return ln.Addr().String(), obs
}

// serveCannedMPPS reads the N-CREATE-RQ and its data set, replies with an N-CREATE-RSP, then reads
// the N-SET-RQ and its data set and replies with an N-SET-RSP. It echoes the SCU's Affected SOP
// Instance UID unless the config tells it to assign its own.
func serveCannedMPPS(ctx context.Context, acc *acse.Acceptor, cfg mppsSCPConfig, obs *mppsObservation) {
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

	// N-CREATE.
	createCmd, createDS, _, rerr := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if rerr != nil {
		obs.mu.Lock()
		obs.err = rerr
		obs.mu.Unlock()
		return
	}
	obs.mu.Lock()
	obs.createCmd = createCmd
	obs.createDS = createDS
	obs.mu.Unlock()

	assigned := createCmd.AffectedSOPInstanceUID
	if cfg.assignInstanceUID != "" {
		assigned = cfg.assignInstanceUID
	}
	createRSP := CommandSet{
		CommandField:              CommandNCreateRSP,
		MessageIDBeingRespondedTo: createCmd.MessageID,
		AffectedSOPClassUID:       createCmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    assigned,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    cfg.createStatus,
	}
	if serr := sendMessage(ctx, conn, m, pcID, createRSP, nil, ts, MaxPDULength(16382)); serr != nil {
		obs.mu.Lock()
		obs.err = serr
		obs.mu.Unlock()
		return
	}
	if !NewStatus(cfg.createStatus, ServiceClassProcedureStep).IsSuccess() {
		return // a failed N-CREATE ends the canned sequence; the SCU never sends the N-SET
	}

	// N-SET.
	setCmd, setDS, _, rerr := receiveMessage(ctx, conn, m, newMessageReassembler(ts))
	if rerr != nil {
		obs.mu.Lock()
		obs.err = rerr
		obs.mu.Unlock()
		return
	}
	obs.mu.Lock()
	obs.setCmd = setCmd
	obs.setDS = setDS
	obs.mu.Unlock()

	setRSP := CommandSet{
		CommandField:              CommandNSetRSP,
		MessageIDBeingRespondedTo: setCmd.MessageID,
		AffectedSOPClassUID:       createCmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    assigned,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    cfg.setStatus,
	}
	if serr := sendMessage(ctx, conn, m, pcID, setRSP, nil, ts, MaxPDULength(16382)); serr != nil {
		obs.mu.Lock()
		obs.err = serr
		obs.mu.Unlock()
	}
}

// dialMPPSSCU opens an MPPS association to the mock SCP, proposing the MPPS context.
func dialMPPSSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("MPPSSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("MPPSSCP"), ModalityPerformedContexts())
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// beginStep builds an N-CREATE attribute set for a procedure step the SCU opens, carrying the
// synthetic identifiers a test uses. These are synthetic fixtures, never real patient data.
func beginStep(instanceUID string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagPerformedProcedureStepID, "RADX-PPS-1")
	if instanceUID != "" {
		ds.SetString(dicom.TagSOPInstanceUID, instanceUID)
	}
	return ds
}

// TestMPPSCreateThenSetDrivesInProgressToCompleted is the acceptance gate: N-CREATE opens the
// procedure step IN PROGRESS, then N-SET advances it to COMPLETED, with a typed Success status
// surfaced on each step.
func TestMPPSCreateThenSetDrivesInProgressToCompleted(t *testing.T) {
	addr, obs := startMPPSSCP(t, mppsSCPConfig{
		createStatus: StatusMPPSSuccess.Code,
		setStatus:    StatusMPPSSuccess.Code,
	})
	assoc, ctx, cancel := dialMPPSSCU(t, addr)
	defer cancel()

	const stepUID = "1.2.826.0.1.3680043.8.498.10000099"
	mpps := assoc.MPPS()

	instanceUID, status, err := mpps.Create(ctx, beginStep(stepUID))
	if err != nil {
		t.Fatalf("MPPS Create: %v", err)
	}
	if status.ServiceClass() != ServiceClassProcedureStep {
		t.Errorf("Create status service class = %v, want ProcedureStep", status.ServiceClass())
	}
	if !status.IsSuccess() {
		t.Errorf("Create status = %s, want Success", status)
	}
	if instanceUID != dicom.UID(stepUID) {
		t.Errorf("Create returned instance UID %q, want %q", instanceUID, stepUID)
	}

	endDS := dicom.NewDataSet()
	endDS.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepCompleted.String())
	status, err = mpps.Set(ctx, instanceUID, endDS)
	if err != nil {
		t.Fatalf("MPPS Set: %v", err)
	}
	if !status.IsSuccess() {
		t.Errorf("Set status = %s, want Success", status)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.err != nil {
		t.Fatalf("SCP error: %v", got.err)
	}

	// The N-CREATE must carry the MPPS Affected SOP Class UID and the IN PROGRESS state in its data
	// set; the N-SET must reference the step by Requested SOP Class/Instance UID, not Affected.
	if got.createCmd.CommandField != CommandNCreateRQ {
		t.Errorf("N-CREATE command field = %#04x, want N-CREATE-RQ", uint16(got.createCmd.CommandField))
	}
	if got.createCmd.AffectedSOPClassUID != dicom.UID(modalityPerformedStepSOPClass) {
		t.Errorf("N-CREATE Affected SOP Class UID = %q, want MPPS", got.createCmd.AffectedSOPClassUID)
	}
	if got.createDS == nil {
		t.Fatal("N-CREATE carried no attribute data set")
	}
	if pps, ok := got.createDS.GetString(dicom.TagPerformedProcedureStepStatus); !ok || pps != ProcedureStepInProgress.String() {
		t.Errorf("N-CREATE Performed Procedure Step Status = %q, want %q", pps, ProcedureStepInProgress.String())
	}

	if got.setCmd.CommandField != CommandNSetRQ {
		t.Errorf("N-SET command field = %#04x, want N-SET-RQ", uint16(got.setCmd.CommandField))
	}
	if got.setCmd.RequestedSOPClassUID != dicom.UID(modalityPerformedStepSOPClass) {
		t.Errorf("N-SET Requested SOP Class UID = %q, want MPPS", got.setCmd.RequestedSOPClassUID)
	}
	if got.setCmd.RequestedSOPInstanceUID != dicom.UID(stepUID) {
		t.Errorf("N-SET Requested SOP Instance UID = %q, want the created step UID", got.setCmd.RequestedSOPInstanceUID)
	}
	if got.setCmd.AffectedSOPInstanceUID != "" {
		t.Errorf("N-SET should not carry an Affected SOP Instance UID, got %q", got.setCmd.AffectedSOPInstanceUID)
	}
	if got.setDS == nil {
		t.Fatal("N-SET carried no attribute data set")
	}
	if pps, ok := got.setDS.GetString(dicom.TagPerformedProcedureStepStatus); !ok || pps != ProcedureStepCompleted.String() {
		t.Errorf("N-SET Performed Procedure Step Status = %q, want %q", pps, ProcedureStepCompleted.String())
	}
}

// TestMPPSCreateAssignsInProgressWhenAbsent confirms Create writes the IN PROGRESS state into a
// copy of the caller's attribute set when the caller did not set it, leaving the caller's dataset
// untouched.
func TestMPPSCreateAssignsInProgressWhenAbsent(t *testing.T) {
	addr, obs := startMPPSSCP(t, mppsSCPConfig{createStatus: StatusMPPSSuccess.Code, setStatus: StatusMPPSSuccess.Code})
	assoc, ctx, cancel := dialMPPSSCU(t, addr)
	defer cancel()

	begin := beginStep("1.2.826.0.1.3680043.8.498.10000100")
	if _, ok := begin.GetString(dicom.TagPerformedProcedureStepStatus); ok {
		t.Fatal("test precondition: begin dataset should not carry a Performed Procedure Step Status")
	}

	mpps := assoc.MPPS()
	if _, _, err := mpps.Create(ctx, begin); err != nil {
		t.Fatalf("MPPS Create: %v", err)
	}
	_ = assoc.Release(ctx)

	// The caller's dataset must be untouched (Create copies before writing the state).
	if _, ok := begin.GetString(dicom.TagPerformedProcedureStepStatus); ok {
		t.Error("Create mutated the caller's dataset by writing the Performed Procedure Step Status into it")
	}

	got := obs.snapshot()
	if got.createDS == nil {
		t.Fatal("N-CREATE carried no attribute data set")
	}
	if pps, ok := got.createDS.GetString(dicom.TagPerformedProcedureStepStatus); !ok || pps != ProcedureStepInProgress.String() {
		t.Errorf("N-CREATE state = %q, want IN PROGRESS (Create should default it)", pps)
	}
}

// TestMPPSCreateAdoptsAssignedInstanceUID confirms Create returns the Affected SOP Instance UID the
// SCP assigned in the N-CREATE-RSP, and that the subsequent N-SET references that assigned UID.
func TestMPPSCreateAdoptsAssignedInstanceUID(t *testing.T) {
	const assigned = "1.2.826.0.1.3680043.8.498.20000200"
	addr, obs := startMPPSSCP(t, mppsSCPConfig{
		createStatus:      StatusMPPSSuccess.Code,
		setStatus:         StatusMPPSSuccess.Code,
		assignInstanceUID: assigned,
	})
	assoc, ctx, cancel := dialMPPSSCU(t, addr)
	defer cancel()

	mpps := assoc.MPPS()
	instanceUID, _, err := mpps.Create(ctx, beginStep(""))
	if err != nil {
		t.Fatalf("MPPS Create: %v", err)
	}
	if instanceUID != dicom.UID(assigned) {
		t.Errorf("Create returned instance UID %q, want the SCP-assigned %q", instanceUID, assigned)
	}

	endDS := dicom.NewDataSet()
	endDS.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepCompleted.String())
	if _, err := mpps.Set(ctx, instanceUID, endDS); err != nil {
		t.Fatalf("MPPS Set: %v", err)
	}
	_ = assoc.Release(ctx)

	got := obs.snapshot()
	if got.setCmd.RequestedSOPInstanceUID != dicom.UID(assigned) {
		t.Errorf("N-SET Requested SOP Instance UID = %q, want the SCP-assigned %q", got.setCmd.RequestedSOPInstanceUID, assigned)
	}
}

// TestMPPSSetSurfacesFailureStatus confirms a Failure-category N-SET status (the step may no longer
// be updated) is surfaced as in-band data, not a Go error, and is never laundered to success.
func TestMPPSSetSurfacesFailureStatus(t *testing.T) {
	addr, _ := startMPPSSCP(t, mppsSCPConfig{
		createStatus: StatusMPPSSuccess.Code,
		setStatus:    StatusMPPSMayNoLongerBeUpdated.Code,
	})
	assoc, ctx, cancel := dialMPPSSCU(t, addr)
	defer cancel()

	mpps := assoc.MPPS()
	instanceUID, _, err := mpps.Create(ctx, beginStep("1.2.826.0.1.3680043.8.498.30000300"))
	if err != nil {
		t.Fatalf("MPPS Create: %v", err)
	}

	endDS := dicom.NewDataSet()
	endDS.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepCompleted.String())
	status, err := mpps.Set(ctx, instanceUID, endDS)
	if err != nil {
		t.Fatalf("MPPS Set returned a Go error for an in-band failure status: %v", err)
	}
	if !status.IsFailure() {
		t.Errorf("Set status = %s, want a Failure category", status)
	}
	if status.IsSuccess() {
		t.Error("Set Failure status must never report IsSuccess")
	}
	_ = assoc.Release(ctx)
}

// TestMPPSCreateRejectsEmptyAttributes confirms Create fails closed on a nil attribute set before
// any wire I/O (an N-CREATE-RQ requires a data set).
func TestMPPSCreateRejectsEmptyAttributes(t *testing.T) {
	addr, _ := startMPPSSCP(t, mppsSCPConfig{createStatus: StatusMPPSSuccess.Code})
	assoc, ctx, cancel := dialMPPSSCU(t, addr)
	defer cancel()

	if _, _, err := assoc.MPPS().Create(ctx, nil); err == nil {
		t.Error("MPPS Create should reject a nil attribute set")
	}
	// A non-nil but empty data set carries none of the required MPPS attributes; it must be
	// rejected before any N-CREATE reaches the peer, not sent as an empty object.
	if _, _, err := assoc.MPPS().Create(ctx, dicom.NewDataSet()); err == nil {
		t.Error("MPPS Create should reject an empty attribute set")
	}
	_ = assoc.Release(ctx)
}

// TestMPPSSetRejectsEmptyInstanceUID confirms Set fails closed when no instance UID is supplied (an
// N-SET-RQ must reference the target by Requested SOP Instance UID).
func TestMPPSSetRejectsEmptyInstanceUID(t *testing.T) {
	addr, _ := startMPPSSCP(t, mppsSCPConfig{createStatus: StatusMPPSSuccess.Code})
	assoc, ctx, cancel := dialMPPSSCU(t, addr)
	defer cancel()

	endDS := dicom.NewDataSet()
	endDS.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepCompleted.String())
	if _, err := assoc.MPPS().Set(ctx, "", endDS); err == nil {
		t.Error("MPPS Set should reject an empty instance UID")
	}
	_ = assoc.Release(ctx)
}

// TestMPPSOnUnestablishedAssociation confirms the SCU fails closed with a typed error rather than
// panicking on a nil/unestablished association (Codex DIMSE-017 discipline).
func TestMPPSOnUnestablishedAssociation(t *testing.T) {
	var a *Association
	if _, _, err := a.MPPS().Create(context.Background(), beginStep("1.2.3")); err == nil {
		t.Error("MPPS Create on a nil association should return a typed error, not panic")
	}
	if _, err := a.MPPS().Set(context.Background(), "1.2.3", dicom.NewDataSet()); err == nil {
		t.Error("MPPS Set on a nil association should return a typed error, not panic")
	}
}

// TestProcedureStepStateString pins the CS keywords the procedure-step states render to (PS3.3
// C.4.14: "IN PROGRESS", "COMPLETED", "DISCONTINUED").
func TestProcedureStepStateString(t *testing.T) {
	cases := []struct {
		state ProcedureStepState
		want  string
	}{
		{ProcedureStepInProgress, "IN PROGRESS"},
		{ProcedureStepCompleted, "COMPLETED"},
		{ProcedureStepDiscontinued, "DISCONTINUED"},
	}
	for _, tc := range cases {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("%v.String() = %q, want %q", tc.state, got, tc.want)
		}
	}
}
