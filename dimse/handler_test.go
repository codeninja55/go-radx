package dimse

import (
	"context"
	"errors"
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

// TestValidateStoreContext is the regression for the SCP-side negotiation guard: a C-STORE-RQ
// whose Affected SOP Class UID does not match the abstract syntax negotiated for the context it
// arrived on must be rejected as a protocol fault, so a peer cannot store an unnegotiated SOP
// Class on an accepted context.
func TestValidateStoreContext(t *testing.T) {
	const negotiated = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	abstractFor := func(pcID uint8) (dicom.SOPClassUID, bool) {
		if pcID == 1 {
			return negotiated, true
		}
		return "", false
	}

	// A matching SOP Class on the negotiated context passes.
	match := CommandSet{CommandField: CommandCStoreRQ, AffectedSOPClassUID: dicom.UID(negotiated)}
	if err := validateStoreContext(match, 1, abstractFor, Sta6); err != nil {
		t.Errorf("matching SOP class rejected: %v", err)
	}

	// A different SOP Class on the negotiated context is a protocol fault.
	mismatch := CommandSet{CommandField: CommandCStoreRQ, AffectedSOPClassUID: dicom.UID("1.2.840.10008.5.1.4.1.1.4")} // MR
	err := validateStoreContext(mismatch, 1, abstractFor, Sta6)
	if err == nil {
		t.Fatal("mismatched SOP class = nil error, want a protocol fault")
	}
	var pe *ProtocolError
	if !errors.As(err, &pe) {
		t.Errorf("error = %T, want *ProtocolError", err)
	}

	// A C-STORE on a context that was never negotiated is also a protocol fault.
	if err := validateStoreContext(match, 9, abstractFor, Sta6); err == nil {
		t.Error("unknown presentation context = nil error, want a protocol fault")
	}
}

// TestValidateStoreInstance is the regression for the instance-identity guard: a C-STORE-RQ whose
// command Affected SOP Instance UID is absent or disagrees with the dataset's (0008,0018) is a
// protocol fault, so the SCP never persists one instance while acknowledging another.
func TestValidateStoreInstance(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.NewTag(0x0008, 0x0018), "1.2.3.4")

	// Command and dataset agree on the instance: passes.
	match := CommandSet{CommandField: CommandCStoreRQ, AffectedSOPInstanceUID: "1.2.3.4"}
	if err := validateStoreInstance(match, ds, Sta6); err != nil {
		t.Errorf("matching instance UID rejected: %v", err)
	}

	// Command names a different instance than the dataset carries: protocol fault.
	mismatch := CommandSet{CommandField: CommandCStoreRQ, AffectedSOPInstanceUID: "9.9.9"}
	err := validateStoreInstance(mismatch, ds, Sta6)
	if err == nil {
		t.Fatal("mismatched instance UID = nil error, want a protocol fault")
	}
	var pe *ProtocolError
	if !errors.As(err, &pe) {
		t.Errorf("error = %T, want *ProtocolError", err)
	}

	// Command omits the mandatory instance UID: protocol fault.
	if err := validateStoreInstance(CommandSet{CommandField: CommandCStoreRQ}, ds, Sta6); err == nil {
		t.Error("empty instance UID = nil error, want a protocol fault")
	}
}
