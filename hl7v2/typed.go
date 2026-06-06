package hl7v2

import "bytes"

// MessageType is the MSH-9 composite: code ^ trigger event ^ structure.
type MessageType struct {
	Code         string // MSH-9.1, e.g. "ORM"
	TriggerEvent string // MSH-9.2, e.g. "O01"
	Structure    string // MSH-9.3, e.g. "ORM_O01"
}

// MSH — message header. MSH-1 and MSH-2 are the encoding characters and are
// exposed through the message's Encoding(), not as string fields. Only the
// fields the M2 converters read are modelled.
type MSH struct {
	SendingApplication   HD          // MSH-3
	SendingFacility      HD          // MSH-4
	ReceivingApplication HD          // MSH-5
	ReceivingFacility    HD          // MSH-6
	DateTime             DTM         // MSH-7
	MessageType          MessageType // MSH-9
	ControlID            string      // MSH-10 (locally unique, not a UID)
	ProcessingID         string      // MSH-11
	VersionID            string      // MSH-12
}

// ParseMSH builds a typed MSH view from a generic Segment, validating its ID.
func ParseMSH(s Segment) (MSH, error) {
	if s.ID() != "MSH" {
		return MSH{}, &SegmentError{Segment: s.ID(), Reason: "not an MSH segment"}
	}
	dt, err := ParseDTM(s.field(7).raw())
	if err != nil {
		return MSH{}, &SegmentError{Segment: "MSH", Reason: "MSH-7 is not a valid timestamp"}
	}
	return MSH{
		SendingApplication:   firstHD(s.field(3)),
		SendingFacility:      firstHD(s.field(4)),
		ReceivingApplication: firstHD(s.field(5)),
		ReceivingFacility:    firstHD(s.field(6)),
		DateTime:             dt,
		MessageType:          parseMessageType(s.field(9)),
		ControlID:            s.field(10).raw(),
		ProcessingID:         s.field(11).raw(),
		VersionID:            s.field(12).raw(),
	}, nil
}

// PID — patient identification. Only the fields the converters read are modelled.
type PID struct {
	SetID         string // PID-1
	PatientID     CX     // PID-3 (first repetition; AllPatientIDs for the rest)
	AllPatientIDs []CX   // PID-3 full repetition list
	PatientName   XPN    // PID-5
	BirthDate     DTM    // PID-7
	Sex           string // PID-8
	Address       XAD    // PID-11
}

// ParsePID builds a typed PID view from a generic Segment, validating its ID.
func ParsePID(s Segment) (PID, error) {
	if s.ID() != "PID" {
		return PID{}, &SegmentError{Segment: s.ID(), Reason: "not a PID segment"}
	}
	dob, err := ParseDTM(s.field(7).raw())
	if err != nil {
		return PID{}, &SegmentError{Segment: "PID", Reason: "PID-7 is not a valid birth date"}
	}
	ids := allCX(s.field(3))
	var first CX
	if len(ids) > 0 {
		first = ids[0]
	}
	return PID{
		SetID:         s.field(1).raw(),
		PatientID:     first,
		AllPatientIDs: ids,
		PatientName:   firstXPN(s.field(5)),
		BirthDate:     dob,
		Sex:           s.field(8).raw(),
		Address:       firstXAD(s.field(11)),
	}, nil
}

// ORC — common order. Only the fields the ORM converter reads are modelled.
type ORC struct {
	OrderControl      string // ORC-1, e.g. "NW" new, "OK" accepted
	PlacerOrderNumber string // ORC-2
	FillerOrderNumber string // ORC-3
	OrderStatus       string // ORC-5
	DateTime          DTM    // ORC-9 Date/Time of Transaction
}

// ParseORC builds a typed ORC view from a generic Segment, validating its ID.
func ParseORC(s Segment) (ORC, error) {
	if s.ID() != "ORC" {
		return ORC{}, &SegmentError{Segment: s.ID(), Reason: "not an ORC segment"}
	}
	dt, err := ParseDTM(s.field(9).raw())
	if err != nil {
		return ORC{}, &SegmentError{Segment: "ORC", Reason: "ORC-9 is not a valid timestamp"}
	}
	return ORC{
		OrderControl:      s.field(1).raw(),
		PlacerOrderNumber: s.field(2).raw(),
		FillerOrderNumber: s.field(3).raw(),
		OrderStatus:       s.field(5).raw(),
		DateTime:          dt,
	}, nil
}

// OBR — observation request. Only the fields the converters read are modelled.
type OBR struct {
	SetID               string // OBR-1
	PlacerOrderNumber   string // OBR-2
	FillerOrderNumber   string // OBR-3
	UniversalServiceID  CWE    // OBR-4 (the ordered procedure)
	ObservationDateTime DTM    // OBR-7
}

// ParseOBR builds a typed OBR view from a generic Segment, validating its ID.
func ParseOBR(s Segment) (OBR, error) {
	if s.ID() != "OBR" {
		return OBR{}, &SegmentError{Segment: s.ID(), Reason: "not an OBR segment"}
	}
	dt, err := ParseDTM(s.field(7).raw())
	if err != nil {
		return OBR{}, &SegmentError{Segment: "OBR", Reason: "OBR-7 is not a valid timestamp"}
	}
	return OBR{
		SetID:               s.field(1).raw(),
		PlacerOrderNumber:   s.field(2).raw(),
		FillerOrderNumber:   s.field(3).raw(),
		UniversalServiceID:  firstCWE(s.field(4)),
		ObservationDateTime: dt,
	}, nil
}

// EVN — event type. Carried by ADT messages; read by the encounter converter.
type EVN struct {
	EventTypeCode    string // EVN-1, a deprecated mirror of MSH-9.2
	RecordedDateTime DTM    // EVN-2
	EventReasonCode  string // EVN-4
}

// ParseEVN builds a typed EVN view from a generic Segment, validating its ID.
func ParseEVN(s Segment) (EVN, error) {
	if s.ID() != "EVN" {
		return EVN{}, &SegmentError{Segment: s.ID(), Reason: "not an EVN segment"}
	}
	dt, err := ParseDTM(s.field(2).raw())
	if err != nil {
		return EVN{}, &SegmentError{Segment: "EVN", Reason: "EVN-2 is not a valid timestamp"}
	}
	return EVN{
		EventTypeCode:    s.field(1).raw(),
		RecordedDateTime: dt,
		EventReasonCode:  s.field(4).raw(),
	}, nil
}

// Segment renders the EVN back to a generic Segment so a constructed message
// round-trips.
func (e EVN) Segment(enc EncodingCharacters) Segment {
	return buildSegment(enc, "EVN",
		leaf(e.EventTypeCode),             // EVN-1
		leaf(e.RecordedDateTime.String()), // EVN-2
		leaf(""),                          // EVN-3
		leaf(e.EventReasonCode),           // EVN-4
	)
}

// PV1 — patient visit. Only the fields the encounter converter reads are
// modelled. VisitNumber is at PV1-19 (not PV1-18 — a common off-by-one).
type PV1 struct {
	SetID            string // PV1-1
	PatientClass     string // PV1-2, e.g. "I" inpatient, "O" outpatient
	AssignedLocation string // PV1-3 (PL, rendered)
	AttendingDoctor  XPN    // PV1-7
	VisitNumber      CX     // PV1-19
}

// ParsePV1 builds a typed PV1 view from a generic Segment, validating its ID.
func ParsePV1(s Segment) (PV1, error) {
	if s.ID() != "PV1" {
		return PV1{}, &SegmentError{Segment: s.ID(), Reason: "not a PV1 segment"}
	}
	return PV1{
		SetID:            s.field(1).raw(),
		PatientClass:     s.field(2).raw(),
		AssignedLocation: renderField(s.field(3)),
		AttendingDoctor:  firstXPN(s.field(7)),
		VisitNumber:      firstCX(s.field(19)),
	}, nil
}

// Segment renders the PV1 back to a generic Segment, placing VisitNumber at
// PV1-19 so the rendered line re-parses equal.
func (p PV1) Segment(enc EncodingCharacters) Segment {
	fields := newFields("PV1", 19)
	fields[1] = leaf(p.SetID)                                                    // PV1-1
	fields[2] = leaf(p.PatientClass)                                             // PV1-2
	fields[3] = rendered(p.AssignedLocation)                                     // PV1-3
	fields[7] = Field{Repetitions: []Repetition{p.AttendingDoctor.repetition()}} // PV1-7
	fields[19] = Field{Repetitions: []Repetition{p.VisitNumber.repetition()}}    // PV1-19
	return Segment{Fields: fields, term: "\r"}
}

// OBX — observation/result. OBX-5 holds one or more raw value repetitions to be
// interpreted per OBX-2 (ValueType); OBX-8 holds the abnormal-flag repetitions.
type OBX struct {
	SetID          string   // OBX-1
	ValueType      string   // OBX-2, e.g. "NM", "ST", "CWE", "SN", "TX"
	ObservationID  CWE      // OBX-3 (what was observed)
	Value          []string // OBX-5 (raw repetitions; interpret per ValueType)
	Units          CWE      // OBX-6
	ReferenceRange string   // OBX-7
	AbnormalFlags  []string // OBX-8
	ResultStatus   string   // OBX-11, e.g. "F" final
}

// ParseOBX builds a typed OBX view from a generic Segment, validating its ID.
func ParseOBX(s Segment) (OBX, error) {
	if s.ID() != "OBX" {
		return OBX{}, &SegmentError{Segment: s.ID(), Reason: "not an OBX segment"}
	}
	return OBX{
		SetID:          s.field(1).raw(),
		ValueType:      s.field(2).raw(),
		ObservationID:  firstCWE(s.field(3)),
		Value:          repetitionValues(s.field(5)),
		Units:          firstCWE(s.field(6)),
		ReferenceRange: s.field(7).raw(),
		AbnormalFlags:  repetitionValues(s.field(8)),
		ResultStatus:   s.field(11).raw(),
	}, nil
}

// Segment renders the OBX back to a generic Segment, with OBX-5 and OBX-8 as
// repetition lists so a multi-valued result round-trips.
func (o OBX) Segment(enc EncodingCharacters) Segment {
	fields := newFields("OBX", 11)
	fields[1] = leaf(o.SetID)                                                  // OBX-1
	fields[2] = leaf(o.ValueType)                                              // OBX-2
	fields[3] = Field{Repetitions: []Repetition{o.ObservationID.repetition()}} // OBX-3
	fields[5] = valueField(o.Value)                                            // OBX-5
	fields[6] = Field{Repetitions: []Repetition{o.Units.repetition()}}         // OBX-6
	fields[7] = leaf(o.ReferenceRange)                                         // OBX-7
	fields[8] = valueField(o.AbnormalFlags)                                    // OBX-8
	fields[11] = leaf(o.ResultStatus)                                          // OBX-11
	return Segment{Fields: fields, term: "\r"}
}

// MSA — message acknowledgement. MSA-1 carries the typed AckCode (HL7 Table
// 0008); a negative acknowledgement is an MSA with a rejecting AckCode, not a
// distinct message type.
type MSA struct {
	AckCode     AckCode // MSA-1
	ControlID   string  // MSA-2 (the control ID of the message being acked)
	TextMessage string  // MSA-3
}

// ParseMSA builds a typed MSA view from a generic Segment, validating its ID and
// the MSA-1 acknowledgement code.
func ParseMSA(s Segment) (MSA, error) {
	if s.ID() != "MSA" {
		return MSA{}, &SegmentError{Segment: s.ID(), Reason: "not an MSA segment"}
	}
	code, err := ParseAckCode(s.field(1).raw())
	if err != nil {
		return MSA{}, &SegmentError{Segment: "MSA", Reason: "MSA-1 is not a recognised acknowledgement code"}
	}
	return MSA{
		AckCode:     code,
		ControlID:   s.field(2).raw(),
		TextMessage: s.field(3).raw(),
	}, nil
}

// Segment renders the MSA back to a generic Segment, escaping its leaf values on
// write like the other renderers. MSA-3 is free diagnostic text, so an in-band
// delimiter or a CR/LF in it is emitted as its escape sequence rather than
// splitting the field, truncating the line, or forging a spurious segment break.
func (m MSA) Segment(enc EncodingCharacters) Segment {
	return buildSegment(enc, "MSA",
		leaf(Escape(string(m.AckCode), enc)), // MSA-1
		leaf(Escape(m.ControlID, enc)),       // MSA-2
		leaf(Escape(m.TextMessage, enc)),     // MSA-3
	)
}

// ERR — error. The HL7 v2.5 layout: ERR-2 the error location, ERR-3 the HL7
// error code (CWE, Table 0357), ERR-4 the severity (Table 0516). The location is
// kept in its rendered form for diagnostics rather than decomposed.
type ERR struct {
	Location string // ERR-2 (ERL, rendered)
	Code     CWE    // ERR-3 (HL7 Table 0357)
	Severity string // ERR-4, e.g. "E" error, "W" warning
}

// ParseERR builds a typed ERR view from a generic Segment, validating its ID.
func ParseERR(s Segment) (ERR, error) {
	if s.ID() != "ERR" {
		return ERR{}, &SegmentError{Segment: s.ID(), Reason: "not an ERR segment"}
	}
	return ERR{
		Location: renderField(s.field(2)),
		Code:     firstCWE(s.field(3)),
		Severity: s.field(4).raw(),
	}, nil
}

// Segment renders the ERR back to a generic Segment.
func (e ERR) Segment(enc EncodingCharacters) Segment {
	fields := newFields("ERR", 4)
	fields[2] = rendered(e.Location)                                  // ERR-2
	fields[3] = Field{Repetitions: []Repetition{e.Code.repetition()}} // ERR-3
	fields[4] = leaf(e.Severity)                                      // ERR-4
	return Segment{Fields: fields, term: "\r"}
}

// MSH returns the typed MSH header and true, or false when the message has no
// MSH (which a parsed Message always does, but the bool keeps the accessor
// uniform with the optional segments).
func (m *Message) MSH() (MSH, bool) {
	seg, ok := m.Segment("MSH")
	if !ok {
		return MSH{}, false
	}
	h, err := ParseMSH(seg)
	if err != nil {
		return MSH{}, false
	}
	return h, true
}

// PID returns the typed PID and true, or false when absent.
func (m *Message) PID() (PID, bool) {
	seg, ok := m.Segment("PID")
	if !ok {
		return PID{}, false
	}
	p, err := ParsePID(seg)
	if err != nil {
		return PID{}, false
	}
	return p, true
}

// EVN returns the typed EVN and true, or false when absent.
func (m *Message) EVN() (EVN, bool) {
	seg, ok := m.Segment("EVN")
	if !ok {
		return EVN{}, false
	}
	e, err := ParseEVN(seg)
	if err != nil {
		return EVN{}, false
	}
	return e, true
}

// PV1 returns the typed PV1 and true, or false when absent.
func (m *Message) PV1() (PV1, bool) {
	seg, ok := m.Segment("PV1")
	if !ok {
		return PV1{}, false
	}
	p, err := ParsePV1(seg)
	if err != nil {
		return PV1{}, false
	}
	return p, true
}

// AllOBX returns every OBX in document order, skipping any malformed one. An
// absent OBX yields an empty slice, never an error.
func (m *Message) AllOBX() []OBX {
	var out []OBX
	for _, seg := range m.AllSegments("OBX") {
		if o, err := ParseOBX(seg); err == nil {
			out = append(out, o)
		}
	}
	return out
}

// parseMessageType reads the MSH-9 composite from its field.
func parseMessageType(f Field) MessageType {
	return MessageType{
		Code:         f.component(1),
		TriggerEvent: f.component(2),
		Structure:    f.component(3),
	}
}

// firstHD parses the first repetition of f as an HD.
func firstHD(f Field) HD {
	if len(f.Repetitions) == 0 {
		return HD{}
	}
	return parseHD(f.Repetitions[0])
}

// firstXPN parses the first repetition of f as an XPN.
func firstXPN(f Field) XPN {
	if len(f.Repetitions) == 0 {
		return XPN{}
	}
	return parseXPN(f.Repetitions[0])
}

// firstCWE parses the first repetition of f as a CWE.
func firstCWE(f Field) CWE {
	if len(f.Repetitions) == 0 {
		return CWE{}
	}
	return parseCWE(f.Repetitions[0])
}

// allCX parses every non-empty repetition of f as a CX (the PID-3 identifier
// list). An absent PID-3 is one empty repetition in the generic tree; it yields
// nil rather than a slice holding a blank CX, so callers that check the length
// or iterate the list never mistake absence for a present-but-empty identifier.
func allCX(f Field) []CX {
	var out []CX
	for i := range f.Repetitions {
		cx := parseCX(f.Repetitions[i])
		if cx == (CX{}) {
			continue
		}
		out = append(out, cx)
	}
	return out
}

// firstCX parses the first repetition of f as a CX.
func firstCX(f Field) CX {
	if len(f.Repetitions) == 0 {
		return CX{}
	}
	return parseCX(f.Repetitions[0])
}

// firstXAD parses the first repetition of f as an XAD.
func firstXAD(f Field) XAD {
	if len(f.Repetitions) == 0 {
		return XAD{}
	}
	return parseXAD(f.Repetitions[0])
}

// repetitionValues returns the delimited value of every repetition of f, the
// form a multi-valued field (OBX-5, OBX-8) carries. Each repetition is rendered
// with the canonical component/subcomponent separators, so a componentised value
// such as a CWE-typed OBX-5 ("123^text^LN") is preserved whole and interpreted
// per OBX-2 by the caller, rather than collapsed to its first component. An
// absent field yields nil so a caller never mistakes absence for a
// present-but-empty value.
func repetitionValues(f Field) []string {
	if len(f.Repetitions) == 0 {
		return nil
	}
	if len(f.Repetitions) == 1 && repetitionString(f.Repetitions[0]) == "" {
		return nil
	}
	out := make([]string, len(f.Repetitions))
	for i := range f.Repetitions {
		out[i] = repetitionString(f.Repetitions[i])
	}
	return out
}

// buildSegment builds a generic Segment from an ID and its 1-based fields. The
// first field of the result is the segment ID, so fields[0] becomes HL7 field 1;
// trailing wholly-empty fields are trimmed so the rendered line matches what the
// parser produces for the same value (a segment round-trips byte-stably).
func buildSegment(enc EncodingCharacters, id string, fields ...Field) Segment {
	_ = enc // reserved for future escape-on-render; the field separator is enc.Field
	end := len(fields)
	for end > 0 && fieldIsEmpty(fields[end-1]) {
		end--
	}
	out := make([]Field, end+1)
	out[0] = leaf(id)
	for i := 0; i < end; i++ {
		out[i+1] = fields[i]
	}
	return Segment{Fields: out, term: "\r"}
}

// newFields allocates a field slice for a segment whose highest populated field
// is hi (1-based), with index 0 holding the segment ID and the rest empty. The
// caller assigns the populated positions by their HL7 field number, so a
// renderer reads exactly like the field layout.
func newFields(id string, hi int) []Field {
	fields := make([]Field, hi+1)
	fields[0] = leaf(id)
	for i := 1; i <= hi; i++ {
		fields[i] = leaf("")
	}
	return fields
}

// leaf builds a flat single-value Field carrying v.
func leaf(v string) Field {
	return Field{Repetitions: []Repetition{{Components: []Component{{Subcomponents: []string{v}}}}}}
}

// rendered builds a single-repetition Field from a canonical delimited string
// (a "rendered" PL/ERL value such as PV1-3 or ERR-2). The value is always parsed
// with the canonical separators it was rendered with by renderField, never the
// in-force message encoding, so the component structure survives even when the
// message uses non-standard delimiters; the generic renderer then re-emits the
// components with the message's separators.
func rendered(v string) Field {
	return Field{Repetitions: []Repetition{parseRepetition([]byte(v), DefaultEncoding())}}
}

// renderField renders the first repetition of f to its canonical delimited
// string form, the stored representation of a "rendered" PL/ERL field (PV1-3,
// ERR-2). It uses the standard component/subcomponent separators regardless of
// the source delimiters, so the stored string is delimiter-independent and the
// rendered(...) inverse reconstructs the same component structure.
func renderField(f Field) string {
	if len(f.Repetitions) == 0 {
		return ""
	}
	return repetitionString(f.Repetitions[0])
}

// repetitionString renders one repetition to its canonical component/
// subcomponent-delimited string. It is the shared inverse of parsing a
// canonical value string back into a repetition (rendered, valueField).
//
// The canonical separators are used on both the read and the render side so the
// component structure of a string-typed field (PV1-3, ERR-2, an OBX-5/OBX-8
// value) is delimiter-independent. The one ambiguous case is a value that
// contains a literal canonical separator (e.g. a '^' in free-text OBX data)
// inside a message that uses that byte as data rather than a separator; such a
// literal is re-split on the canonical round-trip. Real messages escape an
// in-band delimiter (Chapter 2 §2.10, M5 Increment 5), so this is bounded and
// does not affect the byte-exact generic round-trip, which never flattens.
func repetitionString(r Repetition) string {
	var buf bytes.Buffer
	r.render(&buf, DefaultEncoding())
	return buf.String()
}

// valueField builds a Field from a list of canonical value-repetition strings
// (OBX-5, OBX-8). Each value is parsed with the canonical separators it was
// rendered with, so a componentised value re-parses into its components rather
// than a single flat subcomponent. An empty list renders as one empty repetition
// so the field position is preserved.
func valueField(values []string) Field {
	if len(values) == 0 {
		return leaf("")
	}
	reps := make([]Repetition, len(values))
	for i, v := range values {
		reps[i] = parseRepetition([]byte(v), DefaultEncoding())
	}
	return Field{Repetitions: reps}
}

// fieldIsEmpty reports whether f carries no value (used to trim trailing empty
// fields when rendering a segment).
func fieldIsEmpty(f Field) bool {
	for i := range f.Repetitions {
		if f.Repetitions[i].raw() != "" {
			return false
		}
		for _, c := range f.Repetitions[i].Components {
			for _, s := range c.Subcomponents {
				if s != "" {
					return false
				}
			}
		}
	}
	return true
}
