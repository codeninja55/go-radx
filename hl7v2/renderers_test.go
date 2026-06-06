package hl7v2

import (
	"testing"
)

func TestMSHSegmentRoundTrip(t *testing.T) {
	enc := DefaultEncoding()
	dt, _ := ParseDTM("202605311230")
	h := MSH{
		SendingApplication:   HD{NamespaceID: "RADIS"},
		SendingFacility:      HD{NamespaceID: "HOSP"},
		ReceivingApplication: HD{NamespaceID: "PACS"},
		ReceivingFacility:    HD{NamespaceID: "HOSP"},
		DateTime:             dt,
		MessageType:          MessageType{Code: "ORM", TriggerEvent: "O01"},
		ControlID:            "MSG00001",
		ProcessingID:         "P",
		VersionID:            "2.5",
	}
	seg := h.Segment(enc)
	if seg.ID() != "MSH" {
		t.Fatalf("rendered segment ID = %q, want MSH", seg.ID())
	}
	got, err := ParseMSH(seg)
	if err != nil {
		t.Fatalf("ParseMSH(rendered) error = %v", err)
	}
	if got.SendingApplication != h.SendingApplication ||
		got.ReceivingApplication != h.ReceivingApplication ||
		got.MessageType != h.MessageType ||
		got.ControlID != h.ControlID ||
		got.ProcessingID != h.ProcessingID ||
		got.VersionID != h.VersionID {
		t.Errorf("MSH round-trip = %+v, want %+v", got, h)
	}
	if got.DateTime.String() != h.DateTime.String() {
		t.Errorf("MSH-7 round-trip = %q, want %q", got.DateTime.String(), h.DateTime.String())
	}
}

// TestMSHSegmentEncodingHeader proves the rendered MSH carries the delimiter and
// encoding-character fields verbatim so the line re-parses with the same
// EncodingCharacters and the message round-trips.
func TestMSHSegmentEncodingHeader(t *testing.T) {
	enc := DefaultEncoding()
	h := MSH{MessageType: MessageType{Code: "ACK", Structure: "ACK"}, ControlID: "X1"}
	seg := h.Segment(enc)

	m := &Message{Segments: []Segment{seg}, Enc: enc}
	out, _ := m.MarshalText()
	if len(out) < 9 || string(out[:9]) != "MSH|^~\\&|" {
		t.Fatalf("rendered MSH header = %q, want it to start with MSH|^~\\&|", out)
	}
	if _, err := Parse(out); err != nil {
		t.Fatalf("re-parse of rendered MSH error = %v\nrendered = %q", err, out)
	}
}

func TestPIDSegmentRoundTrip(t *testing.T) {
	enc := DefaultEncoding()
	dob, _ := ParseDTM("19620320")
	p := PID{
		SetID:       "1",
		PatientID:   CX{ID: "555-44-4444", AssigningAuthority: HD{NamespaceID: "HOSP"}, IdentifierTypeCode: "MR"},
		PatientName: XPN{Family: "EVERYWOMAN", Given: "EVE", Middle: "E", NameTypeCode: "L"},
		BirthDate:   dob,
		Sex:         "F",
		Address:     XAD{Street: "123 MAIN ST", City: "METROPOLIS", State: "NY", Zip: "10001"},
	}
	got, err := ParsePID(p.Segment(enc))
	if err != nil {
		t.Fatalf("ParsePID(rendered) error = %v", err)
	}
	if got.SetID != p.SetID || got.Sex != p.Sex ||
		got.PatientID.ID != p.PatientID.ID || got.PatientID.IdentifierTypeCode != p.PatientID.IdentifierTypeCode ||
		got.PatientName != p.PatientName || got.Address != p.Address {
		t.Errorf("PID round-trip = %+v, want %+v", got, p)
	}
	if got.BirthDate.String() != p.BirthDate.String() {
		t.Errorf("PID-7 round-trip = %q, want %q", got.BirthDate.String(), p.BirthDate.String())
	}
}

// TestPIDSegmentPreservesAllPatientIDs proves PID-3 renders every identifier in
// AllPatientIDs as a '~'-separated repetition list, so a value such as "MRN~SSN"
// keeps its alternate identifier through a typed round-trip rather than dropping
// AllPatientIDs[1:].
func TestPIDSegmentPreservesAllPatientIDs(t *testing.T) {
	enc := DefaultEncoding()
	ids := []CX{
		{ID: "555-44-4444", AssigningAuthority: HD{NamespaceID: "HOSP"}, IdentifierTypeCode: "MR"},
		{ID: "123-45-6789", AssigningAuthority: HD{NamespaceID: "SSA"}, IdentifierTypeCode: "SS"},
	}
	p := PID{SetID: "1", PatientID: ids[0], AllPatientIDs: ids, Sex: "F"}

	got, err := ParsePID(p.Segment(enc))
	if err != nil {
		t.Fatalf("ParsePID(rendered) error = %v", err)
	}
	if len(got.AllPatientIDs) != 2 {
		t.Fatalf("round-trip AllPatientIDs len = %d, want 2 (alternate identifier dropped)", len(got.AllPatientIDs))
	}
	if got.AllPatientIDs[0].ID != "555-44-4444" || got.AllPatientIDs[0].IdentifierTypeCode != "MR" {
		t.Errorf("PID-3 rep 1 = %+v, want MRN", got.AllPatientIDs[0])
	}
	if got.AllPatientIDs[1].ID != "123-45-6789" || got.AllPatientIDs[1].IdentifierTypeCode != "SS" {
		t.Errorf("PID-3 rep 2 = %+v, want SSN", got.AllPatientIDs[1])
	}
}

// TestPIDSegmentFallsBackToPatientID proves a PID built without AllPatientIDs
// still renders its primary identifier from PatientID, so the empty-slice path
// does not drop PID-3 entirely.
func TestPIDSegmentFallsBackToPatientID(t *testing.T) {
	enc := DefaultEncoding()
	p := PID{SetID: "1", PatientID: CX{ID: "555-44-4444", IdentifierTypeCode: "MR"}}

	got, err := ParsePID(p.Segment(enc))
	if err != nil {
		t.Fatalf("ParsePID(rendered) error = %v", err)
	}
	if got.PatientID.ID != "555-44-4444" || got.PatientID.IdentifierTypeCode != "MR" {
		t.Errorf("PID-3 fallback = %+v, want the single PatientID", got.PatientID)
	}
}

func TestORCSegmentRoundTrip(t *testing.T) {
	enc := DefaultEncoding()
	dt, _ := ParseDTM("202605311230")
	o := ORC{
		OrderControl:      "NW",
		PlacerOrderNumber: "PLACER123",
		FillerOrderNumber: "FILLER456",
		OrderStatus:       "SC",
		DateTime:          dt,
	}
	got, err := ParseORC(o.Segment(enc))
	if err != nil {
		t.Fatalf("ParseORC(rendered) error = %v", err)
	}
	if got.OrderControl != o.OrderControl || got.PlacerOrderNumber != o.PlacerOrderNumber ||
		got.FillerOrderNumber != o.FillerOrderNumber || got.OrderStatus != o.OrderStatus {
		t.Errorf("ORC round-trip = %+v, want %+v", got, o)
	}
	if got.DateTime.String() != o.DateTime.String() {
		t.Errorf("ORC-9 round-trip = %q, want %q", got.DateTime.String(), o.DateTime.String())
	}
}

func TestOBRSegmentRoundTrip(t *testing.T) {
	enc := DefaultEncoding()
	dt, _ := ParseDTM("202605311231")
	o := OBR{
		SetID:               "1",
		PlacerOrderNumber:   "PLACER123",
		FillerOrderNumber:   "FILLER456",
		UniversalServiceID:  CWE{Code: "36643-5", Text: "CHEST XRAY", CodingSystem: "LN"},
		ObservationDateTime: dt,
	}
	got, err := ParseOBR(o.Segment(enc))
	if err != nil {
		t.Fatalf("ParseOBR(rendered) error = %v", err)
	}
	if got.SetID != o.SetID || got.PlacerOrderNumber != o.PlacerOrderNumber ||
		got.FillerOrderNumber != o.FillerOrderNumber || got.UniversalServiceID != o.UniversalServiceID {
		t.Errorf("OBR round-trip = %+v, want %+v", got, o)
	}
	if got.ObservationDateTime.String() != o.ObservationDateTime.String() {
		t.Errorf("OBR-7 round-trip = %q, want %q", got.ObservationDateTime.String(), o.ObservationDateTime.String())
	}
}

// TestRendererEscapesDelimiters proves escape-on-write: a leaf value carrying an
// in-band delimiter is emitted as its §2.10 escape sequence so it cannot fracture
// the composite on re-parse. The faithful inverse is the unescaping string
// accessor Get; the typed read path deliberately returns the raw escaped bytes,
// matching the established contract (a typed accessor never unescapes; only Get
// does).
func TestRendererEscapesDelimiters(t *testing.T) {
	enc := DefaultEncoding()
	p := PID{PatientName: XPN{Family: "SMITH & SONS", Given: "A^B"}}
	seg := p.Segment(enc)

	m := &Message{Segments: []Segment{(MSH{MessageType: MessageType{Code: "ADT", TriggerEvent: "A01"}, ControlID: "1"}).Segment(enc), seg}, Enc: enc}
	out, _ := m.MarshalText()

	reparsed, err := Parse(out)
	if err != nil {
		t.Fatalf("re-parse error = %v\nrendered = %q", err, out)
	}

	// The structure is intact: the in-band '&' and '^' did not split the name into
	// extra subcomponents/components.
	got, ok := reparsed.PID()
	if !ok {
		t.Fatal("re-parsed message has no PID")
	}
	if got.PatientName.Middle != "" {
		t.Errorf("escaped given fractured into XPN-3 = %q, want empty", got.PatientName.Middle)
	}

	// Through the unescaping accessor the original values come back exactly.
	if family, _ := reparsed.Get("PID-5-1-1"); family != "SMITH & SONS" {
		t.Errorf("Get(PID-5-1-1) = %q, want %q", family, "SMITH & SONS")
	}
	if given, _ := reparsed.Get("PID-5-1-2"); given != "A^B" {
		t.Errorf("Get(PID-5-1-2) = %q, want %q", given, "A^B")
	}
}

func TestNewMessageConstruction(t *testing.T) {
	enc := DefaultEncoding()
	m := NewMessage(enc)

	dt, _ := ParseDTM("202605311230")
	m.SetMSH(MSH{
		SendingApplication:   HD{NamespaceID: "RADIS"},
		SendingFacility:      HD{NamespaceID: "HOSP"},
		ReceivingApplication: HD{NamespaceID: "PACS"},
		ReceivingFacility:    HD{NamespaceID: "HOSP"},
		DateTime:             dt,
		MessageType:          MessageType{Code: "ORM", TriggerEvent: "O01"},
		ControlID:            "MSG00001",
		ProcessingID:         "P",
		VersionID:            "2.5",
	})
	dob, _ := ParseDTM("19620320")
	m.AppendSegment((PID{
		PatientID:   CX{ID: "555-44-4444"},
		PatientName: XPN{Family: "EVERYWOMAN", Given: "EVE"},
		BirthDate:   dob,
		Sex:         "F",
	}).Segment(enc))
	m.AppendSegment((ORC{OrderControl: "NW", PlacerOrderNumber: "PLACER123"}).Segment(enc))

	out, err := m.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error = %v", err)
	}

	reparsed, err := Parse(out)
	if err != nil {
		t.Fatalf("re-parse of constructed message error = %v\nrendered = %q", err, out)
	}
	orm, ok := AsORM(reparsed)
	if !ok {
		t.Fatalf("constructed message is not an ORM\nrendered = %q", out)
	}
	if pid, ok := orm.PID(); !ok || pid.PatientName.Family != "EVERYWOMAN" {
		t.Errorf("constructed PID = %+v ok=%v, want family EVERYWOMAN", pid, ok)
	}
	var orders int
	for range orm.Orders() {
		orders++
	}
	if orders != 1 {
		t.Errorf("constructed message has %d orders, want 1", orders)
	}
}

// TestSetMSHReplaces proves SetMSH replaces an existing MSH rather than appending
// a second header, keeping exactly one MSH at the front.
func TestSetMSHReplaces(t *testing.T) {
	enc := DefaultEncoding()
	m := NewMessage(enc)
	m.SetMSH(MSH{MessageType: MessageType{Code: "ACK", Structure: "ACK"}, ControlID: "A"})
	m.SetMSH(MSH{MessageType: MessageType{Code: "ACK", Structure: "ACK"}, ControlID: "B"})

	if got := len(m.AllSegments("MSH")); got != 1 {
		t.Fatalf("MSH count = %d, want 1", got)
	}
	if m.Segments[0].ID() != "MSH" {
		t.Errorf("first segment = %q, want MSH", m.Segments[0].ID())
	}
	h, _ := m.MSH()
	if h.ControlID != "B" {
		t.Errorf("MSH-10 = %q, want B (the replacement)", h.ControlID)
	}
}
