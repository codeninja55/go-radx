package hl7v2

import (
	"errors"
	"testing"
)

func TestParseMSH(t *testing.T) {
	msg, err := Parse([]byte(canonicalORM))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	h, ok := msg.MSH()
	if !ok {
		t.Fatal("MSH() = false, want true")
	}
	if h.SendingApplication.NamespaceID != "RADIS" {
		t.Errorf("SendingApplication = %q, want RADIS", h.SendingApplication.NamespaceID)
	}
	if h.MessageType.Code != "ORM" || h.MessageType.TriggerEvent != "O01" {
		t.Errorf("MessageType = %+v, want {ORM O01}", h.MessageType)
	}
	if h.ControlID != "MSG00001" {
		t.Errorf("ControlID = %q, want MSG00001", h.ControlID)
	}
	if h.VersionID != "2.4" {
		t.Errorf("VersionID = %q, want 2.4", h.VersionID)
	}
	if h.DateTime.String() != "202605311230" {
		t.Errorf("DateTime = %q, want 202605311230", h.DateTime.String())
	}
}

func TestParsePID(t *testing.T) {
	msg, _ := Parse([]byte(canonicalORM))
	p, ok := msg.PID()
	if !ok {
		t.Fatal("PID() = false, want true")
	}
	if p.PatientID.ID != "555-44-4444" {
		t.Errorf("PatientID.ID = %q, want 555-44-4444", p.PatientID.ID)
	}
	if p.PatientName.Family != "EVERYWOMAN" || p.PatientName.Given != "EVE" {
		t.Errorf("PatientName = %+v, want family EVERYWOMAN given EVE", p.PatientName)
	}
	if p.Sex != "F" {
		t.Errorf("Sex = %q, want F", p.Sex)
	}
	if p.BirthDate.String() != "19620320" {
		t.Errorf("BirthDate = %q, want 19620320", p.BirthDate.String())
	}
}

func TestParseORCAndOBR(t *testing.T) {
	msg, _ := Parse([]byte(canonicalORM))
	orc, _ := msg.Segment("ORC")
	obr, _ := msg.Segment("OBR")

	o, err := ParseORC(orc)
	if err != nil {
		t.Fatalf("ParseORC error = %v", err)
	}
	if o.OrderControl != "NW" || o.PlacerOrderNumber != "PLACER123" || o.FillerOrderNumber != "FILLER456" {
		t.Errorf("ORC = %+v", o)
	}
	if o.DateTime.String() != "202605311230" {
		t.Errorf("ORC-9 DateTime = %q, want 202605311230", o.DateTime.String())
	}

	r, err := ParseOBR(obr)
	if err != nil {
		t.Fatalf("ParseOBR error = %v", err)
	}
	if r.UniversalServiceID.Code != "36643-5" || r.UniversalServiceID.Text != "CHEST XRAY" {
		t.Errorf("OBR-4 = %+v, want {36643-5 CHEST XRAY ...}", r.UniversalServiceID)
	}
	if r.ObservationDateTime.String() != "202605311231" {
		t.Errorf("OBR-7 = %q, want 202605311231", r.ObservationDateTime.String())
	}
}

func TestParseSegmentWrongID(t *testing.T) {
	msg, _ := Parse([]byte(canonicalORM))
	pid, _ := msg.Segment("PID")

	_, err := ParseORC(pid)
	var se *SegmentError
	if !errors.As(err, &se) {
		t.Fatalf("ParseORC(PID) error = %v, want *SegmentError", err)
	}
	if se.Segment != "PID" {
		t.Errorf("SegmentError.Segment = %q, want PID", se.Segment)
	}
}

func TestPIDEmptyPatientIDList(t *testing.T) {
	// PID with an absent PID-3 must yield a nil AllPatientIDs, not a slice
	// holding a blank CX.
	msg, err := Parse([]byte(
		"MSH|^~\\&|A|B|C|D|202605311230||ORM^O01|M1|P|2.4\r" +
			"PID|1||||DOE^JOHN||19800101|M\r"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	p, ok := msg.PID()
	if !ok {
		t.Fatal("PID() = false, want true")
	}
	if len(p.AllPatientIDs) != 0 {
		t.Errorf("AllPatientIDs = %d entries, want 0 for an absent PID-3", len(p.AllPatientIDs))
	}
	if p.PatientID != (CX{}) {
		t.Errorf("PatientID = %+v, want zero CX for an absent PID-3", p.PatientID)
	}
}

func TestTypedAccessorsAbsent(t *testing.T) {
	// A minimal message with only MSH: PID() is false, not an error.
	msg, err := Parse([]byte("MSH|^~\\&|A|B|C|D|202605311230||ORM^O01|M1|P|2.4\r"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if _, ok := msg.PID(); ok {
		t.Error("PID() = true on a message with no PID, want false")
	}
}
