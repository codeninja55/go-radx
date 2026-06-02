package hl7v2

import (
	"bytes"
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
	if _, ok := msg.EVN(); ok {
		t.Error("EVN() = true on a message with no EVN, want false")
	}
	if _, ok := msg.PV1(); ok {
		t.Error("PV1() = true on a message with no PV1, want false")
	}
	if obx := msg.AllOBX(); len(obx) != 0 {
		t.Errorf("AllOBX() = %d, want 0 on a message with no OBX", len(obx))
	}
}

func TestParseEVN(t *testing.T) {
	msg := corpusMessage(t, "adt-a01")
	e, ok := msg.EVN()
	if !ok {
		t.Fatal("EVN() = false, want true")
	}
	if e.EventTypeCode != "A01" {
		t.Errorf("EVN-1 EventTypeCode = %q, want A01", e.EventTypeCode)
	}
	if e.RecordedDateTime.String() != "20260531084500" {
		t.Errorf("EVN-2 RecordedDateTime = %q, want 20260531084500", e.RecordedDateTime.String())
	}
}

func TestParsePV1(t *testing.T) {
	// Field positions verified against HL7 v2.5: PatientClass at PV1-2, attending
	// doctor at PV1-7, VisitNumber at PV1-19 (NOT PV1-18 — the Inc 0 off-by-one).
	// The attending doctor is modelled as an XPN, so its first component is the
	// family name (the reference doc commits XPN, not the wire XCN).
	seg := parseTestSegment("PV1|1|I|ICU^101^A||||DOE^JANE^^^^^DR||||||||||||V123")
	p, err := ParsePV1(seg)
	if err != nil {
		t.Fatalf("ParsePV1 error = %v", err)
	}
	if p.PatientClass != "I" {
		t.Errorf("PV1-2 PatientClass = %q, want I", p.PatientClass)
	}
	if p.AssignedLocation != "ICU^101^A" {
		t.Errorf("PV1-3 AssignedLocation = %q, want ICU^101^A", p.AssignedLocation)
	}
	if p.AttendingDoctor.Family != "DOE" || p.AttendingDoctor.Given != "JANE" {
		t.Errorf("PV1-7 AttendingDoctor = %+v, want family DOE given JANE", p.AttendingDoctor)
	}
	if p.VisitNumber.ID != "V123" {
		t.Errorf("PV1-19 VisitNumber = %q, want V123", p.VisitNumber.ID)
	}
}

func TestParseOBX(t *testing.T) {
	// OBX-2 ValueType, OBX-3 ObservationID, OBX-5 Value, OBX-6 Units,
	// OBX-7 ReferenceRange, OBX-8 AbnormalFlags, OBX-11 ResultStatus.
	seg := parseTestSegment("OBX|1|NM|2345-7^GLUCOSE^LN||182|mg/dL|70-105|H|||F")
	o, err := ParseOBX(seg)
	if err != nil {
		t.Fatalf("ParseOBX error = %v", err)
	}
	if o.ValueType != "NM" {
		t.Errorf("OBX-2 ValueType = %q, want NM", o.ValueType)
	}
	if o.ObservationID.Code != "2345-7" || o.ObservationID.Text != "GLUCOSE" {
		t.Errorf("OBX-3 ObservationID = %+v, want {2345-7 GLUCOSE ...}", o.ObservationID)
	}
	if len(o.Value) != 1 || o.Value[0] != "182" {
		t.Errorf("OBX-5 Value = %v, want [182]", o.Value)
	}
	if o.Units.Code != "mg/dL" {
		t.Errorf("OBX-6 Units = %+v, want code mg/dL", o.Units)
	}
	if o.ReferenceRange != "70-105" {
		t.Errorf("OBX-7 ReferenceRange = %q, want 70-105", o.ReferenceRange)
	}
	if len(o.AbnormalFlags) != 1 || o.AbnormalFlags[0] != "H" {
		t.Errorf("OBX-8 AbnormalFlags = %v, want [H]", o.AbnormalFlags)
	}
	if o.ResultStatus != "F" {
		t.Errorf("OBX-11 ResultStatus = %q, want F", o.ResultStatus)
	}
}

func TestParseOBXRepeatedValue(t *testing.T) {
	// A repeated OBX-5 (a~b) reads as a two-element Value slice.
	seg := parseTestSegment("OBX|1|TX|36643-5^CHEST XRAY^LN||No acute process.~Heart size normal.|||N|||F")
	o, err := ParseOBX(seg)
	if err != nil {
		t.Fatalf("ParseOBX error = %v", err)
	}
	want := []string{"No acute process.", "Heart size normal."}
	if len(o.Value) != 2 || o.Value[0] != want[0] || o.Value[1] != want[1] {
		t.Errorf("OBX-5 Value = %v, want %v", o.Value, want)
	}
}

func TestParseOBXComponentValue(t *testing.T) {
	// A CWE-typed OBX-5 carries components; the raw repetition value must be
	// preserved whole so the caller can interpret it per OBX-2, not collapsed to
	// the first component.
	seg := parseTestSegment("OBX|1|CWE|664-3^Tumour^LN||123^Some text^LN|||N|||F")
	o, err := ParseOBX(seg)
	if err != nil {
		t.Fatalf("ParseOBX error = %v", err)
	}
	if len(o.Value) != 1 || o.Value[0] != "123^Some text^LN" {
		t.Errorf("OBX-5 Value = %v, want [123^Some text^LN]", o.Value)
	}
	// The componentised value round-trips through the renderer.
	got, err := ParseOBX(o.Segment(DefaultEncoding()))
	if err != nil {
		t.Fatalf("ParseOBX(rendered) error = %v", err)
	}
	if len(got.Value) != 1 || got.Value[0] != "123^Some text^LN" {
		t.Errorf("OBX-5 round-trip Value = %v, want [123^Some text^LN]", got.Value)
	}
}

func TestPV1RenderNonStandardDelimiters(t *testing.T) {
	// A PV1 rendered with non-standard delimiters must emit PV1-3's components
	// using the message's component separator, not literal canonical carets.
	enc := EncodingCharacters{Field: '#', Component: '@', Repetition: '+', Escape: '$', Subcomponent: '%'}
	pv1 := PV1{PatientClass: "I", AssignedLocation: "ICU^101^A"}
	seg := pv1.Segment(enc)
	var buf bytes.Buffer
	seg.render(&buf, enc)
	// PV1-3 should appear as ICU@101@A (the '@' component separator), never with
	// the canonical '^'.
	if !bytes.Contains(buf.Bytes(), []byte("ICU@101@A")) {
		t.Errorf("PV1 render = %q, want PV1-3 components joined with '@'", buf.String())
	}
	if bytes.Contains(buf.Bytes(), []byte("ICU^101^A")) {
		t.Errorf("PV1 render = %q, leaked canonical '^' separator", buf.String())
	}
}

func TestParseMSA(t *testing.T) {
	msg := corpusMessage(t, "ack")
	seg, _ := msg.Segment("MSA")
	m, err := ParseMSA(seg)
	if err != nil {
		t.Fatalf("ParseMSA error = %v", err)
	}
	if m.AckCode != AckAccept {
		t.Errorf("MSA-1 AckCode = %q, want %q", m.AckCode, AckAccept)
	}
	if !m.AckCode.IsPositive() {
		t.Error("MSA-1 AckCode.IsPositive() = false, want true")
	}
	if m.ControlID != "MSGORU0001" {
		t.Errorf("MSA-2 ControlID = %q, want MSGORU0001", m.ControlID)
	}
	if m.TextMessage != "Message accepted" {
		t.Errorf("MSA-3 TextMessage = %q, want Message accepted", m.TextMessage)
	}
}

func TestParseMSAUnknownAckCode(t *testing.T) {
	seg := parseTestSegment("MSA|ZZ|MSG0001")
	_, err := ParseMSA(seg)
	var se *SegmentError
	if !errors.As(err, &se) {
		t.Fatalf("ParseMSA(bad code) error = %v, want *SegmentError", err)
	}
}

func TestParseERR(t *testing.T) {
	// HL7 v2.5 ERR: ERR-2 location, ERR-3 HL7 error code (CWE), ERR-4 severity.
	seg := parseTestSegment("ERR||PID^1^11|207^Application internal error^HL70357|E")
	e, err := ParseERR(seg)
	if err != nil {
		t.Fatalf("ParseERR error = %v", err)
	}
	if e.Code.Code != "207" || e.Code.Text != "Application internal error" {
		t.Errorf("ERR-3 Code = %+v, want {207 Application internal error ...}", e.Code)
	}
	if e.Severity != "E" {
		t.Errorf("ERR-4 Severity = %q, want E", e.Severity)
	}
}

func TestParsePIDAddress(t *testing.T) {
	// PID-11 carries the patient address as an XAD.
	msg := corpusMessage(t, "adt-a01")
	p, ok := msg.PID()
	if !ok {
		t.Fatal("PID() = false, want true")
	}
	if p.Address.Street != "100 FICTION ST" {
		t.Errorf("PID-11 Address.Street = %q, want 100 FICTION ST", p.Address.Street)
	}
	if p.Address.City != "SPRINGFIELD" || p.Address.State != "IL" {
		t.Errorf("PID-11 Address = %+v, want city SPRINGFIELD state IL", p.Address)
	}
}

func TestAllOBX(t *testing.T) {
	// The ORU fixture has three OBX in document order across two OBR groups.
	msg := corpusMessage(t, "oru-r01")
	obx := msg.AllOBX()
	if len(obx) != 3 {
		t.Fatalf("AllOBX() = %d, want 3", len(obx))
	}
	if obx[0].ObservationID.Code != "2345-7" {
		t.Errorf("AllOBX()[0] ObservationID = %q, want 2345-7", obx[0].ObservationID.Code)
	}
	if obx[2].ValueType != "TX" {
		t.Errorf("AllOBX()[2] ValueType = %q, want TX", obx[2].ValueType)
	}
}

func TestTypedSegmentWrongID(t *testing.T) {
	msg := corpusMessage(t, "adt-a01")
	pid, _ := msg.Segment("PID")
	for name, fn := range map[string]func(Segment) error{
		"ParseEVN": func(s Segment) error { _, err := ParseEVN(s); return err },
		"ParsePV1": func(s Segment) error { _, err := ParsePV1(s); return err },
		"ParseOBX": func(s Segment) error { _, err := ParseOBX(s); return err },
		"ParseMSA": func(s Segment) error { _, err := ParseMSA(s); return err },
		"ParseERR": func(s Segment) error { _, err := ParseERR(s); return err },
	} {
		var se *SegmentError
		if err := fn(pid); !errors.As(err, &se) {
			t.Errorf("%s(PID) error = %v, want *SegmentError", name, err)
		} else if se.Segment != "PID" {
			t.Errorf("%s(PID) SegmentError.Segment = %q, want PID", name, se.Segment)
		}
	}
}

func TestTypedSegmentRoundTrip(t *testing.T) {
	// Each typed segment's Segment(enc) renderer must produce a line that
	// re-parses to an equal typed value (typed round-trip).
	enc := DefaultEncoding()

	evn := EVN{EventTypeCode: "A01", EventReasonCode: "ADM"}
	if got, err := ParseEVN(evn.Segment(enc)); err != nil || got != evn {
		t.Errorf("EVN round-trip = %+v (err %v), want %+v", got, err, evn)
	}

	pv1 := PV1{SetID: "1", PatientClass: "I", AssignedLocation: "ICU^101^A",
		AttendingDoctor: XPN{Family: "DOE", Given: "JANE", NameTypeCode: "L"},
		VisitNumber:     CX{ID: "V123", IdentifierTypeCode: "VN"}}
	if got, err := ParsePV1(pv1.Segment(enc)); err != nil || got != pv1 {
		t.Errorf("PV1 round-trip = %+v (err %v), want %+v", got, err, pv1)
	}

	obx := OBX{SetID: "1", ValueType: "NM", ObservationID: CWE{Code: "2345-7", Text: "GLUCOSE", CodingSystem: "LN"},
		Value: []string{"182"}, Units: CWE{Code: "mg/dL"}, ReferenceRange: "70-105",
		AbnormalFlags: []string{"H"}, ResultStatus: "F"}
	got, err := ParseOBX(obx.Segment(enc))
	if err != nil {
		t.Fatalf("ParseOBX(rendered) error = %v", err)
	}
	if got.ValueType != obx.ValueType || got.ResultStatus != obx.ResultStatus ||
		len(got.Value) != 1 || got.Value[0] != "182" || len(got.AbnormalFlags) != 1 || got.AbnormalFlags[0] != "H" ||
		got.ObservationID != obx.ObservationID || got.Units != obx.Units || got.ReferenceRange != obx.ReferenceRange {
		t.Errorf("OBX round-trip = %+v, want %+v", got, obx)
	}

	msa := MSA{AckCode: AckError, ControlID: "MSG0001", TextMessage: "bad"}
	if got, err := ParseMSA(msa.Segment(enc)); err != nil || got != msa {
		t.Errorf("MSA round-trip = %+v (err %v), want %+v", got, err, msa)
	}

	errSeg := ERR{Location: "PID^1^11", Code: CWE{Code: "207", Text: "internal", CodingSystem: "HL70357"}, Severity: "E"}
	if got, err := ParseERR(errSeg.Segment(enc)); err != nil || got != errSeg {
		t.Errorf("ERR round-trip = %+v (err %v), want %+v", got, err, errSeg)
	}
}

// parseTestSegment parses a single CR-terminated segment line and returns its
// generic Segment, the input the ParseXxx constructors take.
func parseTestSegment(line string) Segment {
	seg, _ := parseSegment([]byte(line), "\r", DefaultEncoding(), 0)
	return seg
}
