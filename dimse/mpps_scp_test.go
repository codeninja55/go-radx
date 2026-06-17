package dimse

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// startMPPSServer serves an MPPSProvider over a Modality Performed Procedure Step presentation context
// on loopback, mirroring startNServer but with the MPPS contexts and a Modality Performed Procedure
// Step SCP AE. It returns the running server.
func startMPPSServer(t *testing.T, h any) *Server {
	t.Helper()
	ae, err := NewAE(AETitle("MPPSSCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := NewServer(ae, ModalityPerformedContexts(), h)

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

// mppsInProgressAttrs builds a minimal valid N-CREATE attribute list carrying the SCU-assigned SOP
// Instance UID and the IN PROGRESS Performed Procedure Step Status.
func mppsInProgressAttrs(instanceUID dicom.UID) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPInstanceUID, string(instanceUID))
	ds.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepInProgress.String())
	ds.SetString(dicom.NewTag(0x0040, 0x0253), "PPS-001") // Performed Procedure Step ID
	return ds
}

// TestMPPSSCPCreateThenSetCompletedRoundTrip is the acceptance gate for the MPPS SCP: an in-process
// MPPS SCU opens a Performed Procedure Step IN PROGRESS with N-CREATE against the MPPSProvider, then
// advances it to COMPLETED with N-SET. Both return Success, and the provider's store holds the
// completed step (PS3.4 F.7.1).
func TestMPPSSCPCreateThenSetCompletedRoundTrip(t *testing.T) {
	const stepUID = dicom.UID("1.2.840.10008.3.1.2.3.3.1.77")
	store := NewMemoryMPPSStore()
	srv := startMPPSServer(t, NewMPPSProvider(store))
	assoc, ctx, cancel := dialMPPSSCU(t, srv.Addr().String())
	defer cancel()

	createUID, status, err := assoc.MPPS().Create(ctx, mppsInProgressAttrs(stepUID))
	if err != nil {
		t.Fatalf("MPPS Create transport error: %v", err)
	}
	if !status.IsSuccess() {
		t.Errorf("N-CREATE status = %s, want Success", status)
	}
	if createUID != stepUID {
		t.Errorf("N-CREATE returned instance UID = %q, want the SCU-assigned %q", createUID, stepUID)
	}

	mods := dicom.NewDataSet()
	mods.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepCompleted.String())
	setStatus, err := assoc.MPPS().Set(ctx, createUID, mods)
	if err != nil {
		t.Fatalf("MPPS Set transport error: %v", err)
	}
	if !setStatus.IsSuccess() {
		t.Errorf("N-SET status = %s, want Success", setStatus)
	}
	_ = assoc.Release(ctx)

	step, ok := store.LookupStep(context.Background(), stepUID)
	if !ok {
		t.Fatal("store does not hold the created step")
	}
	if got, _ := step.GetString(dicom.TagPerformedProcedureStepStatus); !equalsStepState(got, ProcedureStepCompleted) {
		t.Errorf("stored step status = %q, want COMPLETED", got)
	}
}

// TestMPPSSCPRejectsSetOnFinalisedStep confirms the SCP rejects an N-SET against a step that has
// already reached a final state — the step may no longer be updated (PS3.4 F.7.1, F.8.1). The first
// N-SET to COMPLETED succeeds; a second N-SET is refused with the procedure-step-specific
// "may no longer be updated" failure (0x0110), never laundered into success.
func TestMPPSSCPRejectsSetOnFinalisedStep(t *testing.T) {
	const stepUID = dicom.UID("1.2.840.10008.3.1.2.3.3.1.88")
	store := NewMemoryMPPSStore()
	srv := startMPPSServer(t, NewMPPSProvider(store))
	assoc, ctx, cancel := dialMPPSSCU(t, srv.Addr().String())
	defer cancel()

	if _, status, err := assoc.MPPS().Create(ctx, mppsInProgressAttrs(stepUID)); err != nil || !status.IsSuccess() {
		t.Fatalf("MPPS Create = (%s, %v), want Success", status, err)
	}

	completed := dicom.NewDataSet()
	completed.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepCompleted.String())
	if status, err := assoc.MPPS().Set(ctx, stepUID, completed); err != nil || !status.IsSuccess() {
		t.Fatalf("first MPPS Set to COMPLETED = (%s, %v), want Success", status, err)
	}

	// A second N-SET against the now-final step must be refused.
	reopen := dicom.NewDataSet()
	reopen.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepDiscontinued.String())
	status, err := assoc.MPPS().Set(ctx, stepUID, reopen)
	if err != nil {
		t.Fatalf("second MPPS Set transport error: %v", err)
	}
	if status.IsSuccess() {
		t.Errorf("N-SET against a finalised step returned Success; must be refused")
	}
	if status.Code != StatusMPPSMayNoLongerBeUpdated.Code {
		t.Errorf("N-SET against a finalised step status = %s, want 0x0110 (may no longer be updated)", status)
	}
	_ = assoc.Release(ctx)
}

// TestMPPSSCPRejectsSetOnUnknownStep confirms an N-SET against a step the SCP never created is refused
// with No Such SOP Instance (0x0112), failing closed rather than fabricating a step (pynetdicom MPPS
// SCP handle_set).
func TestMPPSSCPRejectsSetOnUnknownStep(t *testing.T) {
	srv := startMPPSServer(t, NewMPPSProvider(NewMemoryMPPSStore()))
	assoc, ctx, cancel := dialMPPSSCU(t, srv.Addr().String())
	defer cancel()

	mods := dicom.NewDataSet()
	mods.SetString(dicom.TagPerformedProcedureStepStatus, ProcedureStepCompleted.String())
	status, err := assoc.MPPS().Set(ctx, dicom.UID("1.2.3.4.5.6.7.8.never"), mods)
	if err != nil {
		t.Fatalf("MPPS Set transport error: %v", err)
	}
	if status.IsSuccess() {
		t.Error("N-SET against an unknown step returned Success; must be refused")
	}
	if status.Code != StatusMPPSNoSuchInstance.Code {
		t.Errorf("N-SET against an unknown step status = %s, want 0x0112 (No Such SOP Instance)", status)
	}
	_ = assoc.Release(ctx)
}

// TestMPPSProviderNCreateValidation pins the N-CREATE attribute validation the provider performs
// directly (unit-level), independent of the loopback: missing instance UID, missing status, and a
// non-IN-PROGRESS status are each refused with the pynetdicom-matching status code, and the store is
// never touched on a refusal.
func TestMPPSProviderNCreateValidation(t *testing.T) {
	const stepUID = dicom.UID("1.2.840.10008.3.1.2.3.3.1.99")
	mpps := dicom.UID(modalityPerformedStepSOPClass)

	withStatus := func(s string) *dicom.DataSet {
		ds := dicom.NewDataSet()
		ds.SetString(dicom.TagPerformedProcedureStepStatus, s)
		return ds
	}

	for _, tc := range []struct {
		name string
		req  NRequest
		want uint16
	}{
		{
			name: "missing instance UID",
			req:  NRequest{AffectedSOPClassUID: mpps, DataSet: withStatus("IN PROGRESS")},
			want: StatusMPPSInvalidAttributeValue.Code,
		},
		{
			name: "missing status attribute",
			req: NRequest{AffectedSOPClassUID: mpps, AffectedSOPInstanceUID: stepUID, DataSet: func() *dicom.DataSet {
				ds := dicom.NewDataSet()
				ds.SetString(dicom.NewTag(0x0008, 0x0070), "RADX")
				return ds
			}()},
			want: StatusMPPSMissingAttribute.Code,
		},
		{
			name: "status not IN PROGRESS",
			req:  NRequest{AffectedSOPClassUID: mpps, AffectedSOPInstanceUID: stepUID, DataSet: withStatus("COMPLETED")},
			want: StatusMPPSInvalidAttributeValue.Code,
		},
		{
			name: "wrong SOP class",
			req:  NRequest{AffectedSOPClassUID: dicom.UID(otherNServiceSOPClass), AffectedSOPInstanceUID: stepUID, DataSet: withStatus("IN PROGRESS")},
			want: StatusSOPClassNotSupported.Code,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMemoryMPPSStore()
			p := NewMPPSProvider(store)
			status, uid := p.NCreate(context.Background(), tc.req)
			if status.Code != tc.want {
				t.Errorf("NCreate status = %s, want %#04x", status, tc.want)
			}
			if status.IsSuccess() {
				t.Error("a refused N-CREATE must not report Success")
			}
			if uid != "" {
				t.Errorf("a refused N-CREATE returned instance UID %q, want empty", uid)
			}
			if _, ok := store.LookupStep(context.Background(), stepUID); ok {
				t.Error("a refused N-CREATE must not persist a step")
			}
		})
	}
}
