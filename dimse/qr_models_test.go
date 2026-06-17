package dimse

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// TestValidateModelLevel is the per-model level-handling regression (PS3.4 Annex C). Each
// information model admits only the Query/Retrieve Levels its annex defines; a level outside that set
// is a *ValidationError so the SCU preflight fails closed before any wire I/O. A flat (level-less)
// model is exempt.
func TestValidateModelLevel(t *testing.T) {
	tests := []struct {
		name    string
		model   dicom.SOPClassUID
		level   QueryLevel
		wantErr bool
	}{
		{"study root accepts IMAGE", studyRootGetSOPClass, QueryLevelImage, false},
		{"study root rejects PATIENT", studyRootFindSOPClass, QueryLevelPatient, true},
		{"study root rejects FRAME", studyRootGetSOPClass, QueryLevelFrame, true},
		{"patient root accepts PATIENT", patientRootFindSOPClass, QueryLevelPatient, false},
		{"patient/study only accepts STUDY", patientStudyOnlyFindSOPClass, QueryLevelStudy, false},
		{"patient/study only rejects SERIES", patientStudyOnlyMoveSOPClass, QueryLevelSeries, true},
		{"patient/study only rejects IMAGE", patientStudyOnlyGetSOPClass, QueryLevelImage, true},
		{"composite instance root accepts IMAGE", compositeInstanceRootGetSOPClass, QueryLevelImage, false},
		{"composite instance root accepts FRAME", compositeInstanceRootMoveSOPClass, QueryLevelFrame, false},
		{"composite instance root rejects STUDY", compositeInstanceRootGetSOPClass, QueryLevelStudy, true},
		{"without bulk data accepts IMAGE", compositeInstanceRetrieveWithoutBulkGetSOPClass, QueryLevelImage, false},
		{"without bulk data rejects FRAME", compositeInstanceRetrieveWithoutBulkGetSOPClass, QueryLevelFrame, true},
		{"worklist model is level-exempt", modalityWorklistSOPClass, QueryLevelImage, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModelLevel(tt.model, tt.level)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateModelLevel(%s, %s) = nil, want a *ValidationError", tt.model, tt.level)
				}
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Errorf("error = %T, want *ValidationError", err)
				}
				return
			}
			if err != nil {
				t.Errorf("validateModelLevel(%s, %s) = %v, want nil", tt.model, tt.level, err)
			}
		})
	}
}

// TestFrameQueryLevelKeyword pins the wire keyword for the new FRAME retrieve level (PS3.4 C.6.5):
// it round-trips through String/ParseQueryLevel as "FRAME".
func TestFrameQueryLevelKeyword(t *testing.T) {
	if got := QueryLevelFrame.String(); got != "FRAME" {
		t.Errorf("QueryLevelFrame.String() = %q, want FRAME", got)
	}
	got, err := ParseQueryLevel("FRAME")
	if err != nil {
		t.Fatalf("ParseQueryLevel(FRAME) = %v, want nil", err)
	}
	if got != QueryLevelFrame {
		t.Errorf("ParseQueryLevel(FRAME) = %v, want QueryLevelFrame", got)
	}
}

// TestMovePreflightRejectsInvalidModelLevel is the fail-closed property for a C-MOVE on the Composite
// Instance Root Retrieve model at an invalid (STUDY) level: the preflight refuses it without touching
// the wire. Move surfaces one terminal Failure and records the *ValidationError in LastError.
func TestMovePreflightRejectsInvalidModelLevel(t *testing.T) {
	err := (&Association{}).movePreflight(compositeInstanceRootMoveSOPClass, QueryLevelStudy, AETitle("DEST"))
	if err == nil {
		t.Fatal("movePreflight on Composite Instance Root at STUDY level = nil, want a *ValidationError")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("error = %T, want *ValidationError", err)
	}
}

// extendedQueryRetrieveWithStorageContexts builds, in a single contiguous-ID pass, the extended Q/R
// model contexts (so a C-GET-RQ on a Composite Instance Root / Patient-Study-Only / without-bulk-data
// model has an accepted context) plus the validated Storage contexts (so the SCP's same-association
// sub-operation C-STOREs land on an accepted context). Combining the two preset slices directly would
// collide their IDs, so this unions them under one ID sequence — the same shape as
// QueryRetrieveWithStorageContexts but over the extended models.
func extendedQueryRetrieveWithStorageContexts() []PresentationContext {
	combined := make([]dicom.SOPClassUID, 0, len(extendedQueryRetrieveSOPClasses)+len(validatedStorageSOPClasses))
	combined = append(combined, extendedQueryRetrieveSOPClasses...)
	combined = append(combined, validatedStorageSOPClasses...)
	return contextsFor(combined)
}

// startExtendedFindServer stands up a C-FIND SCP advertising the extended Q/R model contexts.
func startExtendedFindServer(t *testing.T, h any) string {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := NewServer(ae, ExtendedQueryRetrieveContexts(), h)
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

// TestServerAnswersCFindPatientStudyOnly drives a C-FIND against the Patient/Study Only Q/R model
// (PS3.4 C.6.3): the SCU names the model in WithQueryModel and queries at the STUDY level. The handler
// yields two matches then Success; the SCU must surface both Pending matches, one terminal Success,
// and the handler must observe the Patient/Study Only model SOP Class in its OpInfo.
func TestServerAnswersCFindPatientStudyOnly(t *testing.T) {
	match1 := dicom.NewDataSet()
	match1.SetString(dicom.TagStudyInstanceUID, "1.2.3.1")
	match2 := dicom.NewDataSet()
	match2.SetString(dicom.TagStudyInstanceUID, "1.2.3.2")

	canned := &cannedFindHandler{results: []findResponse{
		{status: StatusFindPending.Code, identifier: match1},
		{status: StatusFindPending.Code, identifier: match2},
		{status: StatusFindSuccess.Code},
	}}
	addr := startExtendedFindServer(t, &findScpHandler{h: canned})

	ae, err := NewAE(AETitle("FINDSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assoc, err := ae.Associate(ctx, addr, AETitle("RADX-SCP"), ExtendedQueryRetrieveContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}

	query := dicom.NewDataSet()
	query.SetString(dicom.TagPatientID, "PID-1")
	query.SetEmpty(dicom.TagStudyInstanceUID)

	var pendingUIDs []string
	var terminal Status
	for status, ds := range assoc.Find(ctx, query, QueryLevelStudy,
		WithQueryModel(PatientStudyOnlyQueryRetrieveInformationModelFind)) {
		if status.IsPending() {
			uid, _ := ds.GetString(dicom.TagStudyInstanceUID)
			pendingUIDs = append(pendingUIDs, uid)
			continue
		}
		terminal = status
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Find LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if len(pendingUIDs) != 2 || pendingUIDs[0] != "1.2.3.1" || pendingUIDs[1] != "1.2.3.2" {
		t.Errorf("pending matches = %v, want [1.2.3.1 1.2.3.2]", pendingUIDs)
	}
	if !terminal.IsSuccess() {
		t.Errorf("terminal status = %s, want Success", terminal)
	}

	canned.mu.Lock()
	gotInfo := canned.calledInfo
	canned.mu.Unlock()
	if gotInfo.SOPClassUID != patientStudyOnlyFindSOPClass {
		t.Errorf("handler OpInfo SOP Class = %q, want Patient/Study Only FIND", gotInfo.SOPClassUID)
	}
}

// startExtendedGetServer stands up a C-GET SCP advertising the extended Q/R model contexts and the
// Storage contexts (for the same-association sub-operation C-STOREs), granting the Storage SCP role.
func startExtendedGetServer(t *testing.T, h any) string {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-GETSCP"))
	if err != nil {
		t.Fatalf("NewAE get SCP: %v", err)
	}
	srv := NewServer(ae, extendedQueryRetrieveWithStorageContexts(), h,
		WithGetStorageRoles(validatedStorageSOPClasses...))
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

// dialExtendedGetServerSCU associates proposing the extended Q/R + Storage contexts and the Storage
// SCP role selection, so a same-association C-GET on a Composite Instance Root model can run.
func dialExtendedGetServerSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	ae, err := NewAE(AETitle("GETSCU"), WithDIMSETimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	assoc, err := ae.Associate(ctx, addr, AETitle("RADX-GETSCP"), extendedQueryRetrieveWithStorageContexts(),
		WithRoleSelection(RoleSelection{SOPClassUID: getStorageSOPClass, SCURole: true, SCPRole: true}))
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// levelRecordingGetHandler is a GetHandler that records the Query/Retrieve Level it was invoked with,
// then C-STOREs back a fixed set of instances. It lets the instance/frame-level test assert the SCP
// observed the IMAGE or FRAME level the SCU requested.
type levelRecordingGetHandler struct {
	instances   []*dicom.DataSet
	gotLevel    QueryLevel
	gotLevelKey string
}

func (h *levelRecordingGetHandler) Get(_ context.Context, query *dicom.DataSet, level QueryLevel, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	h.gotLevel = level
	if query != nil {
		h.gotLevelKey, _ = query.GetString(dicom.TagQueryRetrieveLevel)
	}
	return func(yield func(Status, *dicom.DataSet) bool) {
		for _, ds := range h.instances {
			if !yield(StatusGetPending, ds) {
				return
			}
		}
		yield(StatusGetSuccess, nil)
	}
}

// TestServerAnswersCGetCompositeInstanceRootImageLevel is the IMAGE-level retrieve-granularity round
// trip on the Composite Instance Root Retrieve Information Model (PS3.4 C.6.5): the SCU names the
// model in WithQueryModel and retrieves at QueryLevelImage with a SOP Instance UID. The SCP C-STOREs
// the matched instance back on the same association; the SCU's sink must receive it and the SCP must
// have observed the IMAGE level.
func TestServerAnswersCGetCompositeInstanceRootImageLevel(t *testing.T) {
	handler := &levelRecordingGetHandler{instances: []*dicom.DataSet{instanceDataset("1.2.3.99")}}
	addr := startExtendedGetServer(t, handler)

	assoc, ctx, cancel := dialExtendedGetServerSCU(t, addr)
	defer cancel()

	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(tagSOPInstanceUID, "1.2.3.99")

	var terminal Status
	for status := range assoc.Get(ctx, query, QueryLevelImage, sink,
		WithQueryModel(CompositeInstanceRootRetrieveInformationModelGet)) {
		if !status.IsPending() {
			terminal = status
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if !terminal.IsSuccess() {
		t.Errorf("terminal status = %s, want Success", terminal)
	}
	instances, _, _ := sink.snapshot()
	if len(instances) != 1 || instances[0] != "1.2.3.99" {
		t.Fatalf("sink received %v, want [1.2.3.99]", instances)
	}
	if handler.gotLevel != QueryLevelImage || handler.gotLevelKey != "IMAGE" {
		t.Errorf("SCP observed level %v / key %q, want QueryLevelImage / IMAGE", handler.gotLevel, handler.gotLevelKey)
	}
}

// TestServerAnswersCGetCompositeInstanceRootFrameLevel is the FRAME-level retrieve-granularity round
// trip on the Composite Instance Root Retrieve Information Model (PS3.4 C.6.5): the SCU retrieves at
// QueryLevelFrame, carrying a Simple Frame List (0008,1161). The SCP must observe the FRAME level
// keyword in the identifier and complete the retrieve.
func TestServerAnswersCGetCompositeInstanceRootFrameLevel(t *testing.T) {
	handler := &levelRecordingGetHandler{instances: []*dicom.DataSet{instanceDataset("1.2.3.framed")}}
	addr := startExtendedGetServer(t, handler)

	assoc, ctx, cancel := dialExtendedGetServerSCU(t, addr)
	defer cancel()

	sink := &recordingGetSink{}
	query := dicom.NewDataSet()
	query.SetString(tagSOPInstanceUID, "1.2.3.framed")
	// Frame selection at the FRAME level (PS3.4 C.6.5): retrieve frames 1 and 2 via Simple Frame
	// List (0008,1161), VR UL.
	query.Set(dicom.Element{Tag: dicom.TagSimpleFrameList, VR: dicom.VRUL, Value: dicom.NewInts(dicom.VRUL, 1, 2)})

	var terminal Status
	for status := range assoc.Get(ctx, query, QueryLevelFrame, sink,
		WithQueryModel(CompositeInstanceRootRetrieveInformationModelGet)) {
		if !status.IsPending() {
			terminal = status
		}
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get LastError = %v, want nil", err)
	}
	_ = assoc.Release(ctx)

	if !terminal.IsSuccess() {
		t.Errorf("terminal status = %s, want Success", terminal)
	}
	if handler.gotLevel != QueryLevelFrame || handler.gotLevelKey != "FRAME" {
		t.Errorf("SCP observed level %v / key %q, want QueryLevelFrame / FRAME", handler.gotLevel, handler.gotLevelKey)
	}
}
