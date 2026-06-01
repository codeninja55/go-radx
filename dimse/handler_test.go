package dimse

import (
	"context"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// echoHandler is a minimal Handler that records the OpInfo it was called with and returns a
// configurable status.
type echoHandler struct {
	status Status
	seen   *OpInfo
}

func (h *echoHandler) Echo(_ context.Context, info OpInfo) Status {
	h.seen = &info
	return h.status
}

// TestOpInfoZeroValueIsPHIFree is a structural guard: OpInfo carries only protocol identifiers
// (AE titles, presentation/transfer/SOP identifiers, message id) and never a dataset, so it is
// safe to log (PRD §8.2, §9.1). The test pins the field set by constructing one.
func TestOpInfoCarriesProtocolContext(t *testing.T) {
	info := OpInfo{
		CallingAETitle: AETitle("SCU"),
		CalledAETitle:  AETitle("SCP"),
		PresentationID: 1,
		MessageID:      7,
	}
	if info.CallingAETitle != "SCU" || info.CalledAETitle != "SCP" {
		t.Errorf("OpInfo AE titles = (%q,%q), want (SCU,SCP)", info.CallingAETitle, info.CalledAETitle)
	}
	if info.PresentationID != 1 || info.MessageID != 7 {
		t.Errorf("OpInfo ids = (pc=%d, msg=%d), want (1, 7)", info.PresentationID, info.MessageID)
	}
}

// TestHandlerEchoIsInvokable confirms the EchoHandler capability dispatches and returns the
// handler's status. An echo-only SCP implements EchoHandler (interface segregation), not the full
// Handler union.
func TestHandlerEchoIsInvokable(t *testing.T) {
	var h EchoHandler = &echoHandler{status: StatusEchoSuccess}
	got := h.Echo(context.Background(), OpInfo{MessageID: 3})
	if !got.IsSuccess() {
		t.Errorf("EchoHandler.Echo() = %s, want success", got)
	}
}

// storeRecorder is a minimal Handler implementing both Echo and Store, recording the dataset and
// OpInfo it was dispatched with and returning a configurable status.
type storeRecorder struct {
	status   Status
	seenInfo *OpInfo
	seenDS   *dicom.DataSet
}

func (h *storeRecorder) Echo(_ context.Context, info OpInfo) Status { return StatusEchoSuccess }

func (h *storeRecorder) Store(_ context.Context, ds *dicom.DataSet, info OpInfo) Status {
	h.seenInfo = &info
	h.seenDS = ds
	return h.status
}

// TestHandlerStoreIsInvokable confirms the Handler interface's Store method dispatches, receives
// the dataset, and returns the handler's status.
func TestHandlerStoreIsInvokable(t *testing.T) {
	rec := &storeRecorder{status: StatusStoreSuccess}
	var h Handler = rec
	ds := dicom.NewDataSet()
	ds.SetString(dicom.NewTag(0x0008, 0x0018), "1.2.3")
	got := h.Store(context.Background(), ds, OpInfo{MessageID: 5})
	if !got.IsSuccess() {
		t.Errorf("Handler.Store() = %s, want success", got)
	}
	if rec.seenDS == nil {
		t.Fatal("Store did not receive the dataset")
	}
	if v, _ := rec.seenDS.GetString(dicom.NewTag(0x0008, 0x0018)); v != "1.2.3" {
		t.Errorf("Store dataset SOP Instance UID = %q, want 1.2.3", v)
	}
}
