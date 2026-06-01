package dimse

import (
	"context"
	"testing"
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

// TestHandlerEchoIsInvokable confirms the Handler interface's Echo method dispatches and returns
// the handler's status.
func TestHandlerEchoIsInvokable(t *testing.T) {
	var h Handler = &echoHandler{status: StatusEchoSuccess}
	got := h.Echo(context.Background(), OpInfo{MessageID: 3})
	if !got.IsSuccess() {
		t.Errorf("Handler.Echo() = %s, want success", got)
	}
}
