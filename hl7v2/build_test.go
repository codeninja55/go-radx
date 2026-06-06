package hl7v2

import (
	"errors"
	"testing"
)

// fixedControlID is a deterministic control-ID source for the BuildACK tests so
// the minted MSH-10 is reproducible without a clock or randomness.
func fixedControlID(id string) func() string { return func() string { return id } }

func TestBuildACKFieldSwap(t *testing.T) {
	src, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	ack, err := src.BuildACK(AckAccept, WithControlIDSource(fixedControlID("ACK999")))
	if err != nil {
		t.Fatalf("BuildACK error = %v", err)
	}

	h, ok := ack.MSH()
	if !ok {
		t.Fatal("ack MSH() = false, want true")
	}
	srcMSH, _ := src.MSH()

	// Sending and receiving application/facility are swapped from the source.
	if h.SendingApplication != srcMSH.ReceivingApplication {
		t.Errorf("MSH-3 = %+v, want source MSH-5 %+v", h.SendingApplication, srcMSH.ReceivingApplication)
	}
	if h.SendingFacility != srcMSH.ReceivingFacility {
		t.Errorf("MSH-4 = %+v, want source MSH-6 %+v", h.SendingFacility, srcMSH.ReceivingFacility)
	}
	if h.ReceivingApplication != srcMSH.SendingApplication {
		t.Errorf("MSH-5 = %+v, want source MSH-3 %+v", h.ReceivingApplication, srcMSH.SendingApplication)
	}
	if h.ReceivingFacility != srcMSH.SendingFacility {
		t.Errorf("MSH-6 = %+v, want source MSH-4 %+v", h.ReceivingFacility, srcMSH.SendingFacility)
	}

	// MSH-9 is ACK with the echoed inbound trigger event: ACK^<trigger>^ACK.
	if h.MessageType.Code != "ACK" || h.MessageType.TriggerEvent != "O01" || h.MessageType.Structure != "ACK" {
		t.Errorf("MSH-9 = %+v, want {ACK O01 ACK}", h.MessageType)
	}

	// MSH-10 is freshly minted from the injected source.
	if h.ControlID != "ACK999" {
		t.Errorf("MSH-10 = %q, want ACK999", h.ControlID)
	}

	// MSA-1 is the chosen code; MSA-2 echoes the source MSH-10.
	typed, ok := AsACK(ack)
	if !ok {
		t.Fatal("AsACK on built ack = false, want true")
	}
	msa, ok := typed.MSA()
	if !ok {
		t.Fatal("built ack MSA() = false, want true")
	}
	if msa.AckCode != AckAccept {
		t.Errorf("MSA-1 = %q, want AA", msa.AckCode)
	}
	if msa.ControlID != "MSG00001" {
		t.Errorf("MSA-2 = %q, want source MSH-10 MSG00001", msa.ControlID)
	}
}

func TestBuildACKEnhancedMode(t *testing.T) {
	src, _ := Parse([]byte(canonicalORM))
	ack, err := src.BuildACK(AckCommitAccept, WithControlIDSource(fixedControlID("ACK002")))
	if err != nil {
		t.Fatalf("BuildACK error = %v", err)
	}
	typed, _ := AsACK(ack)
	msa, ok := typed.MSA()
	if !ok {
		t.Fatal("MSA() = false, want true")
	}
	if msa.AckCode != AckCommitAccept || !msa.AckCode.IsPositive() {
		t.Errorf("MSA-1 = %q, want CA", msa.AckCode)
	}
	// Enhanced and original mode share the same field-swap and trigger echo; only
	// the MSA-1 code differs.
	h, _ := ack.MSH()
	if h.MessageType.Code != "ACK" || h.MessageType.TriggerEvent != "O01" {
		t.Errorf("MSH-9 = %+v, want {ACK O01 ...}", h.MessageType)
	}
}

func TestBuildACKErrorText(t *testing.T) {
	src, _ := Parse([]byte(canonicalORM))
	ack, err := src.BuildACK(AckError,
		WithControlIDSource(fixedControlID("ACK003")),
		WithACKText("OBX-5 failed datatype validation"))
	if err != nil {
		t.Fatalf("BuildACK error = %v", err)
	}
	typed, _ := AsACK(ack)
	msa, _ := typed.MSA()
	if msa.AckCode != AckError {
		t.Errorf("MSA-1 = %q, want AE", msa.AckCode)
	}
	if msa.TextMessage != "OBX-5 failed datatype validation" {
		t.Errorf("MSA-3 = %q, want the supplied text", msa.TextMessage)
	}
}

func TestBuildACKSendingOverride(t *testing.T) {
	src, _ := Parse([]byte(canonicalORM))
	app := HD{NamespaceID: "GORADX"}
	fac := HD{NamespaceID: "SITE1"}
	ack, err := src.BuildACK(AckAccept,
		WithControlIDSource(fixedControlID("ACK004")),
		WithACKSendingApplication(app),
		WithACKSendingFacility(fac))
	if err != nil {
		t.Fatalf("BuildACK error = %v", err)
	}
	h, _ := ack.MSH()
	if h.SendingApplication != app {
		t.Errorf("MSH-3 = %+v, want override %+v", h.SendingApplication, app)
	}
	if h.SendingFacility != fac {
		t.Errorf("MSH-4 = %+v, want override %+v", h.SendingFacility, fac)
	}
	// The receiver is still the source's sender even when the responder app is
	// overridden, so the reply is routed back to the originator.
	srcMSH, _ := src.MSH()
	if h.ReceivingApplication != srcMSH.SendingApplication {
		t.Errorf("MSH-5 = %+v, want source MSH-3 %+v", h.ReceivingApplication, srcMSH.SendingApplication)
	}
}

func TestBuildACKMissingMSH(t *testing.T) {
	_, err := (&Message{}).BuildACK(AckAccept)
	var se *SegmentError
	if !errors.As(err, &se) {
		t.Fatalf("BuildACK on MSH-less message error = %v, want *SegmentError", err)
	}
	if se.Segment != "MSH" {
		t.Errorf("error names segment %q, want MSH", se.Segment)
	}
}

// TestBuildACKDefaultControlID confirms the default control-ID source produces a
// non-empty, distinct ID without an injected source, so a caller that omits the
// option still gets a usable MSH-10.
func TestBuildACKDefaultControlID(t *testing.T) {
	src, _ := Parse([]byte(canonicalORM))
	a, err := src.BuildACK(AckAccept)
	if err != nil {
		t.Fatalf("BuildACK error = %v", err)
	}
	b, _ := src.BuildACK(AckAccept)
	ha, _ := a.MSH()
	hb, _ := b.MSH()
	if ha.ControlID == "" {
		t.Error("default MSH-10 is empty, want a generated value")
	}
	if ha.ControlID == hb.ControlID {
		t.Errorf("two default control IDs collide (%q); want distinct", ha.ControlID)
	}
}

// TestBuildACKNoPHIInControlID guards against a control-ID generator that copies
// patient-derived bytes: the minted MSH-10 must not equal the source PID-3.
func TestBuildACKDoesNotEchoPatientID(t *testing.T) {
	src, _ := Parse([]byte(canonicalORM))
	ack, _ := src.BuildACK(AckAccept)
	h, _ := ack.MSH()
	if h.ControlID == "555-44-4444" {
		t.Error("minted control ID echoes the patient identifier; must be synthetic")
	}
}

func TestBuildACKRoundTrips(t *testing.T) {
	src, _ := Parse([]byte(canonicalORM))
	ack, err := src.BuildACK(AckAccept, WithControlIDSource(fixedControlID("ACK500")))
	if err != nil {
		t.Fatalf("BuildACK error = %v", err)
	}
	out, err := ack.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}
	reparsed, err := Parse(out)
	if err != nil {
		t.Fatalf("re-parse of built ACK error = %v\nrendered = %q", err, out)
	}
	typed, ok := AsACK(reparsed)
	if !ok {
		t.Fatal("re-parsed ACK is not an ACK lens")
	}
	if msa, ok := typed.MSA(); !ok || msa.AckCode != AckAccept || msa.ControlID != "MSG00001" {
		t.Errorf("re-parsed MSA = %+v ok=%v, want code AA control MSG00001", msa, ok)
	}
}
