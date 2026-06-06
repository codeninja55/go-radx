//go:build interop

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/dimse/integration/dcm4chee"
)

// TestInteropDcm4cheeMPPS is the M3 Modality Performed Procedure Step interop gate against
// dcm4chee-arc: a go-radx MPPS SCU drives an N-CREATE that opens a procedure step IN PROGRESS, then
// an N-SET that advances it to COMPLETED, asserting a success-class status on each leg. It proves
// the normalised N-CREATE/N-SET SCU interoperates with a third-party MPPS SCP — this is step 3.5 of
// the radiology workflow, the modality reporting its procedure step.
//
// dcm4chee-arc is an MPPS SCP by default, so the presentation context should negotiate; if the
// archive rejects the MPPS context (abstract-syntax-not-supported), the live leg SKIPS with a
// documented reason rather than failing. The MPPS SCU's real correctness gate is the in-process
// mock-SCP unit tests in the dimse package.
//
// The procedure-step identifiers are synthetic test fixtures, never real patient data, and are
// never logged.
func TestInteropDcm4cheeMPPS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	arc := startDcm4chee(ctx, t)

	const (
		seededPatientID  = "RADX-PID-MPPS"
		seededStudyUID   = "1.2.826.0.1.3680043.8.498.30000001"
		performedStepUID = "1.2.826.0.1.3680043.8.498.30000002"
		modality         = "CT"
	)

	// dcm4chee-arc rejects an MPPS that references an unknown patient, so register one first.
	if err := arc.CreatePatient(ctx, seededPatientID, "DOE^JOHN"); err != nil {
		t.Fatalf("create patient in dcm4chee: %v", err)
	}

	calling, err := dimse.ParseAETitle("RADX-SCU")
	if err != nil {
		t.Fatalf("parse calling AE title: %v", err)
	}
	called, err := dimse.ParseAETitle(dcm4chee.AETitle)
	if err != nil {
		t.Fatalf("parse called AE title: %v", err)
	}
	ae, err := dimse.NewAE(calling)
	if err != nil {
		t.Fatalf("new AE: %v", err)
	}

	assoc, err := ae.Associate(ctx, arc.DICOMAddr(), called, dimse.ModalityPerformedContexts())
	if err != nil {
		t.Fatalf("associate for MPPS: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	// Skip the live leg unless the archive accepted the MPPS presentation context (interop-reality
	// guard): a peer that does not advertise the MPPS SOP Class as SCP rejects the context as
	// abstract-syntax-not-supported, and the in-process mock-SCP unit tests remain the real gate.
	mppsAccepted := false
	for _, pc := range assoc.AcceptedContexts() {
		if pc.AbstractSyntax == dimse.ModalityPerformedProcedureStepSOPClass && pc.Result == dimse.ContextAccepted {
			mppsAccepted = true
		}
	}
	if !mppsAccepted {
		t.Skipf("dcm4chee %s AE did not accept the Modality Performed Procedure Step presentation context "+
			"(abstract-syntax-not-supported); MPPS interop pending archive MPPS-SCP configuration", dcm4chee.AETitle)
	}

	mpps := assoc.MPPS()

	// N-CREATE: open the step IN PROGRESS. Build the mandatory Type 1/2 attributes dcm4chee expects
	// for a Modality Performed Procedure Step (PS3.4 F.7.1, PS3.3 C.4.14).
	begin := dicom.NewDataSet()
	begin.SetString(dicom.TagSOPInstanceUID, performedStepUID)
	begin.SetString(dicom.TagPatientID, seededPatientID)
	begin.SetString(dicom.TagStudyInstanceUID, seededStudyUID)
	begin.SetString(dicom.TagModality, modality)
	begin.SetString(dicom.TagPerformedProcedureStepID, "RADX-PPS-1")
	begin.SetString(dicom.TagPerformedProcedureStepStartDate, "20260607")
	begin.SetString(dicom.TagPerformedProcedureStepStartTime, "090000")

	instanceUID, status, err := mpps.Create(ctx, begin)
	if err != nil {
		t.Fatalf("MPPS N-CREATE transport error: %v", err)
	}
	if !status.IsSuccess() {
		t.Fatalf("MPPS N-CREATE status = %s, want a success category", status)
	}
	if instanceUID == "" {
		t.Fatal("MPPS N-CREATE returned an empty procedure-step instance UID")
	}

	// N-SET: advance the step to COMPLETED.
	end := dicom.NewDataSet()
	end.SetString(dicom.TagPerformedProcedureStepStatus, dimse.ProcedureStepCompleted.String())
	end.SetString(dicom.TagPerformedProcedureStepEndDate, "20260607")
	end.SetString(dicom.TagPerformedProcedureStepEndTime, "093000")

	status, err = mpps.Set(ctx, instanceUID, end)
	if err != nil {
		t.Fatalf("MPPS N-SET transport error: %v", err)
	}
	if !status.IsSuccess() {
		t.Fatalf("MPPS N-SET status = %s, want a success category", status)
	}
}
