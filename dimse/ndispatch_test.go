package dimse

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// otherNServiceSOPClass is a second, NEVER-negotiated N-service abstract syntax (a synthetic UID).
// The negotiated-context regression sends an N-service request carrying this SOP Class on the
// presentation context negotiated for displaySystemSOPClass, which must be rejected.
const otherNServiceSOPClass dicom.SOPClassUID = "1.2.840.10008.5.1.1.40.99"

// displaySystemSOPClass is the Display System SOP Class UID (PS3.4 EE), the canonical N-GET target:
// a Display System Management SCU reads the configuration of a display device with N-GET. It is a
// synthetic test fixture for the loopback, negotiated as an arbitrary N-service abstract syntax.
const displaySystemSOPClass dicom.SOPClassUID = "1.2.840.10008.5.1.1.40"

// displaySystemInstance is the well-known Display System SOP Instance UID (PS3.4 EE.1).
const displaySystemInstance dicom.UID = "1.2.840.10008.5.1.1.40.1"

// nServiceHandler is a test SCP that records the N-GET and N-DELETE requests it serves and returns
// configurable statuses, mirroring the C-service serverTestHandler. It implements ONLY the
// NGetHandler and NDeleteHandler capabilities (interface segregation): the dispatch must route the
// two N-services to it and leave the other four N-services / the C-services refused.
type nServiceHandler struct {
	mu sync.Mutex

	getReq    *NRequest
	deleteReq *NRequest

	// getStatus / deleteStatus are the statuses the handler returns (default Success).
	getStatus    Status
	deleteStatus Status
	// getAttrs is the attribute set the N-GET handler returns on a success status.
	getAttrs *dicom.DataSet
}

func (h *nServiceHandler) NGet(_ context.Context, req NRequest) (Status, *dicom.DataSet) {
	h.mu.Lock()
	r := req
	h.getReq = &r
	h.mu.Unlock()
	if h.getStatus.IsSuccess() {
		return h.getStatus, h.getAttrs
	}
	return h.getStatus, nil
}

func (h *nServiceHandler) NDelete(_ context.Context, req NRequest) Status {
	h.mu.Lock()
	r := req
	h.deleteReq = &r
	h.mu.Unlock()
	return h.deleteStatus
}

func (h *nServiceHandler) snapshot() (getReq, deleteReq *NRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.getReq, h.deleteReq
}

// nServiceContexts builds the proposal/supported list for the loopback: a single Display System
// context the N-GET and N-DELETE both run over.
func nServiceContexts() []PresentationContext {
	return []PresentationContext{NewPresentationContext(1, displaySystemSOPClass)}
}

// startNServer serves an SCP hosting the N-service handler on loopback and returns it and its SCU
// AE. It mirrors startServer but with the N-service presentation context.
func startNServer(t *testing.T, h any) (*Server, *AE) {
	t.Helper()
	ae, err := NewAE(AETitle("RADX-NSCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := NewServer(ae, nServiceContexts(), h)

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
	return srv, ae
}

// dialNSCU opens an association to the N-service SCP, proposing the Display System context.
func dialNSCU(t *testing.T, addr string) (*Association, context.Context, context.CancelFunc) {
	t.Helper()
	scu, err := NewAE(AETitle("RADX-NSCU"), WithDIMSETimeout(3*time.Second))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	assoc, err := scu.Associate(ctx, addr, AETitle("RADX-NSCP"), nServiceContexts())
	if err != nil {
		cancel()
		t.Fatalf("Associate: %v", err)
	}
	return assoc, ctx, cancel
}

// TestNGetLoopbackReturnsAttributes is the acceptance gate for N-GET through the new SCP dispatch
// substrate: an in-process SCU runs N-GET against the Server, the NGetHandler answers with a Success
// status and an attribute set, and the SCU receives the attributes back. It also asserts the
// handler saw the Requested reference pair and the Attribute Identifier List the SCU named.
func TestNGetLoopbackReturnsAttributes(t *testing.T) {
	attrs := dicom.NewDataSet()
	attrs.SetString(dicom.NewTag(0x0008, 0x0070), "RADX") // Manufacturer
	attrs.SetString(dicom.NewTag(0x0018, 0x1020), "1.0")  // Software Versions

	h := &nServiceHandler{getStatus: StatusNSuccess, getAttrs: attrs}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	requested := []dicom.Tag{dicom.NewTag(0x0008, 0x0070), dicom.NewTag(0x0018, 0x1020)}
	result, err := assoc.NGet(ctx, displaySystemSOPClass, displaySystemInstance, requested)
	if err != nil {
		t.Fatalf("N-GET transport error: %v", err)
	}
	if !result.Status.IsSuccess() {
		t.Errorf("N-GET status = %s, want Success", result.Status)
	}
	if result.Attributes == nil {
		t.Fatal("N-GET returned no attributes on a Success status")
	}
	if man, ok := result.Attributes.GetString(dicom.NewTag(0x0008, 0x0070)); !ok || man != "RADX" {
		t.Errorf("returned Manufacturer = %q (present=%v), want RADX", man, ok)
	}
	if result.AffectedSOPInstanceUID != displaySystemInstance {
		t.Errorf("N-GET-RSP Affected SOP Instance UID = %q, want %q", result.AffectedSOPInstanceUID, displaySystemInstance)
	}
	_ = assoc.Release(ctx)

	getReq, _ := h.snapshot()
	if getReq == nil {
		t.Fatal("handler did not observe the N-GET request")
	}
	if getReq.RequestedSOPClassUID != dicom.UID(displaySystemSOPClass) {
		t.Errorf("handler Requested SOP Class UID = %q, want %q", getReq.RequestedSOPClassUID, displaySystemSOPClass)
	}
	if getReq.RequestedSOPInstanceUID != displaySystemInstance {
		t.Errorf("handler Requested SOP Instance UID = %q, want %q", getReq.RequestedSOPInstanceUID, displaySystemInstance)
	}
	if len(getReq.AttributeIdentifierList) != len(requested) {
		t.Fatalf("handler Attribute Identifier List length = %d, want %d", len(getReq.AttributeIdentifierList), len(requested))
	}
	for i, want := range requested {
		if getReq.AttributeIdentifierList[i] != want {
			t.Errorf("handler Attribute Identifier List[%d] = %v, want %v", i, getReq.AttributeIdentifierList[i], want)
		}
	}
}

// TestNGetLoopbackSurfacesFailureStatus confirms a Failure-category N-GET status is surfaced as
// in-band data (not a Go error) and carries no attributes — a missing object is never laundered into
// a success with an empty result.
func TestNGetLoopbackSurfacesFailureStatus(t *testing.T) {
	h := &nServiceHandler{getStatus: StatusNoSuchSOPInstance}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	result, err := assoc.NGet(ctx, displaySystemSOPClass, displaySystemInstance, nil)
	if err != nil {
		t.Fatalf("N-GET returned a Go error for an in-band failure status: %v", err)
	}
	if !result.Status.IsFailure() {
		t.Errorf("N-GET status = %s, want a Failure category", result.Status)
	}
	if result.Status.IsSuccess() {
		t.Error("a No Such SOP Instance status must never report IsSuccess")
	}
	if result.Attributes != nil {
		t.Error("a failure N-GET must not surface attributes")
	}
	_ = assoc.Release(ctx)
}

// TestNDeleteLoopback is the acceptance gate for N-DELETE through the SCP dispatch substrate: an
// in-process SCU runs N-DELETE against the Server, the NDeleteHandler answers with Success, and the
// handler saw the Requested reference pair.
func TestNDeleteLoopback(t *testing.T) {
	h := &nServiceHandler{deleteStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	status, err := assoc.NDelete(ctx, displaySystemSOPClass, displaySystemInstance)
	if err != nil {
		t.Fatalf("N-DELETE transport error: %v", err)
	}
	if !status.IsSuccess() {
		t.Errorf("N-DELETE status = %s, want Success", status)
	}
	_ = assoc.Release(ctx)

	_, deleteReq := h.snapshot()
	if deleteReq == nil {
		t.Fatal("handler did not observe the N-DELETE request")
	}
	if deleteReq.RequestedSOPClassUID != dicom.UID(displaySystemSOPClass) {
		t.Errorf("handler Requested SOP Class UID = %q, want %q", deleteReq.RequestedSOPClassUID, displaySystemSOPClass)
	}
	if deleteReq.RequestedSOPInstanceUID != displaySystemInstance {
		t.Errorf("handler Requested SOP Instance UID = %q, want %q", deleteReq.RequestedSOPInstanceUID, displaySystemInstance)
	}
}

// TestNGetRefusedWhenUnsupported confirms the dispatch refuses an N-GET reaching an SCP with no
// NGetHandler capability with a StatusSOPClassNotSupported N-GET-RSP rather than panicking — the
// interface-segregation contract, symmetric with the C-service refusal paths.
func TestNGetRefusedWhenUnsupported(t *testing.T) {
	// A handler that implements only NDeleteHandler — an N-GET routed to it must be refused.
	h := &deleteOnlyHandler{status: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	result, err := assoc.NGet(ctx, displaySystemSOPClass, displaySystemInstance, nil)
	if err != nil {
		t.Fatalf("N-GET transport error: %v", err)
	}
	if result.Status.Code != StatusSOPClassNotSupported.Code {
		t.Errorf("N-GET status = %s, want Refused: SOP Class Not Supported (0x0122)", result.Status)
	}
	if result.Status.IsSuccess() {
		t.Error("a refused N-GET must never report IsSuccess")
	}
	_ = assoc.Release(ctx)
}

// deleteOnlyHandler implements ONLY NDeleteHandler — the interface-segregation case for the
// N-service dispatch (no NGetHandler method).
type deleteOnlyHandler struct {
	status Status
}

func (h *deleteOnlyHandler) NDelete(_ context.Context, _ NRequest) Status { return h.status }

// TestNGetNDeleteOnUnestablishedAssociation confirms the SCU primitives fail closed with a typed
// error rather than panicking on a nil/unestablished association (Codex DIMSE-017 discipline).
func TestNGetNDeleteOnUnestablishedAssociation(t *testing.T) {
	var a *Association
	if _, err := a.NGet(context.Background(), displaySystemSOPClass, displaySystemInstance, nil); err == nil {
		t.Error("N-GET on a nil association should return a typed error, not panic")
	}
	if _, err := a.NDelete(context.Background(), displaySystemSOPClass, displaySystemInstance); err == nil {
		t.Error("N-DELETE on a nil association should return a typed error, not panic")
	}
}

// TestNGetNDeleteRejectEmptyReferences confirms the SCU primitives fail closed before any wire I/O on
// a missing SOP Class or instance UID.
func TestNGetNDeleteRejectEmptyReferences(t *testing.T) {
	h := &nServiceHandler{getStatus: StatusNSuccess, deleteStatus: StatusNSuccess}
	srv, _ := startNServer(t, h)
	assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
	defer cancel()

	if _, err := assoc.NGet(ctx, "", displaySystemInstance, nil); err == nil {
		t.Error("N-GET should reject an empty SOP Class UID")
	}
	if _, err := assoc.NGet(ctx, displaySystemSOPClass, "", nil); err == nil {
		t.Error("N-GET should reject an empty instance UID")
	}
	if _, err := assoc.NDelete(ctx, "", displaySystemInstance); err == nil {
		t.Error("N-DELETE should reject an empty SOP Class UID")
	}
	if _, err := assoc.NDelete(ctx, displaySystemSOPClass, ""); err == nil {
		t.Error("N-DELETE should reject an empty instance UID")
	}
	_ = assoc.Release(ctx)
}

// TestValidateNContext is the unit regression for the DIMSE-N negotiation guard (the symmetry with
// validateStoreContext): an N-service request must arrive on a presentation context whose negotiated
// abstract syntax equals the SOP Class the request names, else it is a protocol fault — a peer
// cannot run an N-service outside the negotiated/accepted SOP Class. It also pins which SOP Class UID
// field each N-primitive carries per PS3.7 §10: the reference-pair operations (N-GET/N-SET/N-ACTION/
// N-DELETE) carry the Requested SOP Class UID, N-CREATE/N-EVENT-REPORT the Affected SOP Class UID.
func TestValidateNContext(t *testing.T) {
	abstractFor := func(pcID uint8) (dicom.SOPClassUID, bool) {
		if pcID == 1 {
			return displaySystemSOPClass, true
		}
		return "", false
	}

	// Reference-pair operations validate the Requested SOP Class UID against the negotiated context.
	for _, tc := range []struct {
		name  string
		field CommandField
	}{
		{"N-GET", CommandNGetRQ},
		{"N-SET", CommandNSetRQ},
		{"N-ACTION", CommandNActionRQ},
		{"N-DELETE", CommandNDeleteRQ},
	} {
		match := CommandSet{CommandField: tc.field, RequestedSOPClassUID: dicom.UID(displaySystemSOPClass)}
		if err := validateNContext(match, 1, abstractFor, Sta6); err != nil {
			t.Errorf("%s: matching Requested SOP Class on the negotiated context rejected: %v", tc.name, err)
		}
		mismatch := CommandSet{CommandField: tc.field, RequestedSOPClassUID: dicom.UID(otherNServiceSOPClass)}
		if err := validateNContext(mismatch, 1, abstractFor, Sta6); err == nil {
			t.Errorf("%s: mismatched Requested SOP Class on the negotiated context = nil error, want a protocol fault", tc.name)
		} else {
			if _, ok := errors.AsType[*ProtocolError](err); !ok {
				t.Errorf("%s: error = %T, want *ProtocolError", tc.name, err)
			}
		}
	}

	// Affected-pair operations validate the Affected SOP Class UID against the negotiated context.
	for _, tc := range []struct {
		name  string
		field CommandField
	}{
		{"N-CREATE", CommandNCreateRQ},
		{"N-EVENT-REPORT", CommandNEventReportRQ},
	} {
		match := CommandSet{CommandField: tc.field, AffectedSOPClassUID: dicom.UID(displaySystemSOPClass)}
		if err := validateNContext(match, 1, abstractFor, Sta6); err != nil {
			t.Errorf("%s: matching Affected SOP Class on the negotiated context rejected: %v", tc.name, err)
		}
		mismatch := CommandSet{CommandField: tc.field, AffectedSOPClassUID: dicom.UID(otherNServiceSOPClass)}
		if err := validateNContext(mismatch, 1, abstractFor, Sta6); err == nil {
			t.Errorf("%s: mismatched Affected SOP Class on the negotiated context = nil error, want a protocol fault", tc.name)
		}
	}

	// An N-service on a context that was never negotiated is a protocol fault.
	never := CommandSet{CommandField: CommandNDeleteRQ, RequestedSOPClassUID: dicom.UID(displaySystemSOPClass)}
	if err := validateNContext(never, 9, abstractFor, Sta6); err == nil {
		t.Error("N-service on an unknown presentation context = nil error, want a protocol fault")
	}
}

// TestNServiceMismatchedSOPClassRefused is the loopback regression for the negotiated-context boundary
// (BUG: the N-service dispatch arms routed to the handler WITHOUT the presentation-context SOP-class
// validation the C-service paths perform). It negotiates SOP Class A (displaySystemSOPClass), then
// sends an N-DELETE and an N-CREATE carrying SOP Class B (otherNServiceSOPClass) on the context
// negotiated for A, and asserts the handler is NEVER invoked and the request fails — the destructive
// N-DELETE must not run for a SOP Class never accepted on the association (PS3.7 §9.1).
func TestNServiceMismatchedSOPClassRefused(t *testing.T) {
	for _, tc := range []struct {
		name   string
		field  CommandField
		setSOP func(*CommandSet)
	}{
		{
			name:  "N-DELETE",
			field: CommandNDeleteRQ,
			setSOP: func(c *CommandSet) {
				c.RequestedSOPClassUID = dicom.UID(otherNServiceSOPClass)
				c.RequestedSOPInstanceUID = displaySystemInstance
			},
		},
		{
			name:  "N-CREATE",
			field: CommandNCreateRQ,
			setSOP: func(c *CommandSet) {
				c.AffectedSOPClassUID = dicom.UID(otherNServiceSOPClass)
				c.AffectedSOPInstanceUID = displaySystemInstance
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &nServiceHandler{getStatus: StatusNSuccess, deleteStatus: StatusNSuccess}
			srv, _ := startNServer(t, h)
			assoc, ctx, cancel := dialNSCU(t, srv.Addr().String())
			defer cancel()

			// Resolve the presentation context negotiated for SOP Class A, then craft an N-service
			// request carrying SOP Class B on that same context — the negotiation-bypass attempt the
			// high-level SCU primitives (which pick the context by SOP Class) cannot construct.
			pcID, _, ok := assoc.contextForQuery(displaySystemSOPClass)
			if !ok {
				t.Fatal("no accepted Display System presentation context")
			}
			rq := CommandSet{
				CommandField:       tc.field,
				MessageID:          1,
				CommandDataSetType: CommandDataSetNotPresent,
			}
			tc.setSOP(&rq)
			if err := sendCommand(ctx, assoc.requestor.Conn(), assoc.requestor.Machine(), pcID, rq); err != nil {
				t.Fatalf("send mismatched-context %s: %v", tc.name, err)
			}

			// The SCP must fault the association rather than answer: the SCU's read returns a transport/
			// protocol error, never a Success RSP.
			rsp, _, err := receiveCommand(ctx, assoc.requestor.Conn(), assoc.requestor.Machine())
			if err == nil && rsp.HasStatus && rsp.Status == StatusNSuccess.Code {
				t.Fatalf("%s on a mismatched context returned a Success RSP, want a fault", tc.name)
			}

			getReq, deleteReq := h.snapshot()
			if deleteReq != nil {
				t.Errorf("%s on a mismatched context invoked the N-DELETE handler — the negotiated-context boundary was bypassed", tc.name)
			}
			if getReq != nil {
				t.Errorf("%s on a mismatched context invoked the N-GET handler — the negotiated-context boundary was bypassed", tc.name)
			}
		})
	}
}
