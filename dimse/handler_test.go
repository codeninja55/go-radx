package dimse

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/pdu"
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

func (h *storeRecorder) Find(_ context.Context, _ *dicom.DataSet, _ QueryLevel, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {}
}

func (h *storeRecorder) Move(_ context.Context, _ *dicom.DataSet, _ QueryLevel, _ AETitle, _ OpInfo) iter.Seq2[Status, *dicom.DataSet] {
	return func(yield func(Status, *dicom.DataSet) bool) {}
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

// TestServeEchoRejectsNonVerificationContext is the regression for the C-ECHO negotiation guard
// (the symmetry with validateStoreContext — P2 review): a C-ECHO arriving on a presentation
// context whose negotiated abstract syntax is not the Verification SOP Class must be rejected as a
// protocol fault, so a peer that negotiated only Storage contexts cannot run Verification on one.
func TestServeEchoRejectsNonVerificationContext(t *testing.T) {
	const storage = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	abstractFor := func(pcID uint8) (dicom.SOPClassUID, bool) {
		switch pcID {
		case 1:
			return verificationSOPClass, true
		case 3:
			return storage, true
		default:
			return "", false
		}
	}

	verifyRQ := CommandSet{CommandField: CommandCEchoRQ, AffectedSOPClassUID: dicom.UID(verificationSOPClass)}

	// A C-ECHO on the Verification context, naming the Verification SOP Class, passes.
	if err := validateEchoContext(verifyRQ, 1, abstractFor, Sta6); err != nil {
		t.Errorf("C-ECHO on the Verification context rejected: %v", err)
	}

	// A C-ECHO on a Storage context is a protocol fault (context check).
	err := validateEchoContext(verifyRQ, 3, abstractFor, Sta6)
	if err == nil {
		t.Fatal("C-ECHO on a Storage context = nil error, want a protocol fault")
	}
	var pe *ProtocolError
	if !errors.As(err, &pe) {
		t.Errorf("error = %T, want *ProtocolError", err)
	}

	// A C-ECHO on a context that was never negotiated is also a protocol fault.
	if err := validateEchoContext(verifyRQ, 9, abstractFor, Sta6); err == nil {
		t.Error("C-ECHO on an unknown presentation context = nil error, want a protocol fault")
	}

	// A C-ECHO on the Verification context but naming a NON-Verification SOP Class in the command
	// is a protocol fault (command check — the new symmetry with validateStoreContext).
	wrongClass := CommandSet{CommandField: CommandCEchoRQ, AffectedSOPClassUID: dicom.UID(storage)}
	if err := validateEchoContext(wrongClass, 1, abstractFor, Sta6); err == nil {
		t.Error("C-ECHO-RQ with a non-Verification Affected SOP Class = nil error, want a protocol fault")
	}
}

// TestAbstractSyntaxForRequiresAcceptance is the regression for the abstract-syntax resolver
// hardening (P2 adversarial review): resolving a presentation context's abstract syntax must
// require the context was ACCEPTED (Result 0, PS3.8 §9.3.3.2), not merely proposed. A peer that
// negotiated a context that the acceptor REJECTED must not be able to operate (C-ECHO/C-STORE) on
// that rejected context's ID, bypassing negotiation.
func TestAbstractSyntaxForRequiresAcceptance(t *testing.T) {
	const ctImageStorage = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.2")

	requested := []pdu.PresentationContextRQ{
		{ID: 1, AbstractSyntax: string(verificationSOPClass)},
		{ID: 3, AbstractSyntax: string(ctImageStorage)},
		{ID: 5, AbstractSyntax: string(ctImageStorage)},
	}
	accepted := []pdu.PresentationContextAC{
		{ID: 1, Result: pdu.PresentationContextAcceptance},
		{ID: 3, Result: pdu.PresentationContextAbstractSyntaxNotSupported}, // rejected during negotiation
	}

	// A context accepted (Result 0) with a matching requested abstract syntax resolves it.
	if abstract, ok := abstractSyntaxFor(requested, accepted, 1); !ok || abstract != verificationSOPClass {
		t.Errorf("accepted context 1 = (%q, %v), want (%q, true)", abstract, ok, verificationSOPClass)
	}

	// A context that is present but was REJECTED (Result != 0) must not resolve, even though it
	// has a requested abstract syntax — this is the bypass the fix closes.
	if abstract, ok := abstractSyntaxFor(requested, accepted, 3); ok {
		t.Errorf("rejected context 3 = (%q, true), want empty/false", abstract)
	}

	// A context accepted but absent from the requested set cannot have its abstract syntax
	// resolved (no abstract syntax to map to).
	acceptedOrphan := append(accepted, pdu.PresentationContextAC{ID: 7, Result: pdu.PresentationContextAcceptance})
	if abstract, ok := abstractSyntaxFor(requested, acceptedOrphan, 7); ok {
		t.Errorf("accepted-but-unrequested context 7 = (%q, true), want empty/false", abstract)
	}

	// An unknown presentation context ID resolves to nothing.
	if abstract, ok := abstractSyntaxFor(requested, accepted, 9); ok {
		t.Errorf("unknown context 9 = (%q, true), want empty/false", abstract)
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
