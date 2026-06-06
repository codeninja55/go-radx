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

// TestInteropDcm4cheeMWLCFind is the M3 Modality Worklist interop gate against dcm4chee-arc: a
// scheduled procedure step is seeded into the archive's MWL SCP via REST, then a go-radx SCU drives
// an MWL C-FIND (Association.FindWorklist) against the archive and asserts the seeded step is
// returned as a Pending match with a terminal Success. It proves the streaming MWL C-FIND SCU
// interoperates with a third-party MWL SCP and that the flat worklist model is honoured (no
// Query/Retrieve Level on the wire). This is step 2 of the radiology workflow — the modality queries
// its worklist.
//
// The query and the seeded step values are synthetic test fixtures, never real patient data, and
// they are never logged.
func TestInteropDcm4cheeMWLCFind(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	arc := startDcm4chee(ctx, t)

	const (
		seededStudyUID  = "1.2.826.0.1.3680043.8.498.10000001"
		seededStepID    = "RADX-SPS-1"
		seededStationAE = "RADX-MOD"
		seededModality  = "CT"
	)

	// Seed a worklist item so the MWL C-FIND has a scheduled procedure step to match.
	if err := arc.CreateWorklistItem(ctx, dcm4chee.WorklistItem{
		StudyInstanceUID:         seededStudyUID,
		AccessionNumber:          "RADX-ACC-1",
		RequestedProcedureID:     "RADX-RP-1",
		ScheduledStationAETitle:  seededStationAE,
		ScheduledProcedureStepID: seededStepID,
		Modality:                 seededModality,
		ScheduledStartDate:       "20260606",
		ScheduledStartTime:       "090000",
	}); err != nil {
		t.Fatalf("seed MWL item into dcm4chee: %v", err)
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

	mwlAssoc, err := ae.Associate(ctx, arc.DICOMAddr(), called, dimse.BasicWorklistContexts())
	if err != nil {
		t.Fatalf("associate for MWL C-FIND: %v", err)
	}
	defer func() { _ = mwlAssoc.Release(ctx) }()

	// Build a worklist query that requests the scheduled-step return keys and constrains the match to
	// the seeded scheduled station AE Title, the modality-side filter a modality applies.
	query := dimse.NewWorklistQuery()
	if seq, ok := query.GetSequence(dicom.TagScheduledProcedureStepSequence); ok {
		for item := range seq.Items() {
			item.DataSet.SetString(dicom.TagScheduledStationAETitle, seededStationAE)
			item.DataSet.SetEmpty(dicom.TagScheduledProcedureStepID)
			item.DataSet.SetEmpty(dicom.TagModality)
			item.DataSet.SetEmpty(dicom.TagScheduledProcedureStepStartDate)
		}
	}

	matched := false
	terminalSuccess := false
	for st, match := range mwlAssoc.FindWorklist(ctx, query) {
		switch {
		case st.IsPending():
			if match == nil {
				t.Error("Pending MWL match had a nil identifier")
				continue
			}
			seq, ok := match.GetSequence(dicom.TagScheduledProcedureStepSequence)
			if !ok {
				t.Error("MWL match had no Scheduled Procedure Step Sequence (0040,0100)")
				continue
			}
			for item := range seq.Items() {
				if id, has := item.DataSet.GetString(dicom.TagScheduledProcedureStepID); has && id == seededStepID {
					matched = true
				}
			}
		case st.IsSuccess():
			terminalSuccess = true
		case st.IsFailure():
			t.Errorf("MWL C-FIND terminal status category = Failure, want Success")
		}
	}
	if err := mwlAssoc.LastError(); err != nil {
		t.Fatalf("MWL C-FIND transport error: %v", err)
	}
	if !matched {
		t.Error("MWL C-FIND did not return the seeded scheduled procedure step")
	}
	if !terminalSuccess {
		t.Error("MWL C-FIND did not end with a terminal Success status")
	}
}
