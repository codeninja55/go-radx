package hl7v2

import (
	"strconv"
	"sync/atomic"
	"time"
)

// NewMessage starts an empty message with the given encoding characters. It has
// no segments yet; SetMSH installs the header and AppendSegment adds the body.
// Construction produces canonical output: '\r' segment terminators and the
// supplied EncodingCharacters.
func NewMessage(enc EncodingCharacters) *Message {
	return &Message{Enc: enc}
}

// SetMSH installs h as the message header, rendered against the message's
// encoding characters. An existing MSH is replaced in place so the message keeps
// exactly one header at the front; otherwise the new MSH is inserted first.
func (m *Message) SetMSH(h MSH) {
	seg := h.Segment(m.Enc)
	for i := range m.Segments {
		if m.Segments[i].ID() == "MSH" {
			m.Segments[i] = seg
			return
		}
	}
	m.Segments = append([]Segment{seg}, m.Segments...)
}

// AppendSegment appends a generic segment to the message body. Build it from a
// typed segment via its Segment method, or construct it directly. The segment's
// terminator is normalised to '\r' so a constructed message is canonical.
func (m *Message) AppendSegment(s Segment) {
	s.term = "\r"
	m.Segments = append(m.Segments, s)
}

// Segment renders the MSH back to a generic Segment in the wire layout the parser
// expects: Fields[1] is the field separator byte and Fields[2] is the encoding
// characters, both verbatim, so the rendered line carries the delimiters and
// re-parses with the same EncodingCharacters. MSH-3 onward are escaped on write.
func (h MSH) Segment(enc EncodingCharacters) Segment {
	fields := make([]Field, 13)
	fields[0] = leaf("MSH")
	fields[1] = leaf(string(enc.Field))
	fields[2] = leaf(encodingChars(enc))
	fields[3] = escapedRepetitionField(h.SendingApplication.repetition(), enc)    // MSH-3
	fields[4] = escapedRepetitionField(h.SendingFacility.repetition(), enc)       // MSH-4
	fields[5] = escapedRepetitionField(h.ReceivingApplication.repetition(), enc)  // MSH-5
	fields[6] = escapedRepetitionField(h.ReceivingFacility.repetition(), enc)     // MSH-6
	fields[7] = escapedLeaf(h.DateTime.String(), enc)                             // MSH-7
	fields[8] = leaf("")                                                          // MSH-8 (security)
	fields[9] = escapedRepetitionField(messageTypeRepetition(h.MessageType), enc) // MSH-9
	fields[10] = escapedLeaf(h.ControlID, enc)                                    // MSH-10
	fields[11] = escapedLeaf(h.ProcessingID, enc)                                 // MSH-11
	fields[12] = escapedLeaf(h.VersionID, enc)                                    // MSH-12
	return Segment{Fields: trimTrailingEmpty(fields), term: "\r"}
}

// Segment renders the PID back to a generic Segment, escaping its leaf values on
// write so an in-band delimiter survives the round-trip. PID-3 carries every
// patient identifier as a '~'-separated repetition list (MRN, SSN, ...), so an
// alternate identifier is not dropped on a typed round-trip.
func (p PID) Segment(enc EncodingCharacters) Segment {
	fields := newFields("PID", 11)
	fields[1] = escapedLeaf(p.SetID, enc)                               // PID-1
	fields[3] = escapedPatientIDField(p, enc)                           // PID-3
	fields[5] = escapedRepetitionField(p.PatientName.repetition(), enc) // PID-5
	fields[7] = escapedLeaf(p.BirthDate.String(), enc)                  // PID-7
	fields[8] = escapedLeaf(p.Sex, enc)                                 // PID-8
	fields[11] = escapedRepetitionField(p.Address.repetition(), enc)    // PID-11
	return Segment{Fields: trimTrailingEmpty(fields), term: "\r"}
}

// Segment renders the ORC back to a generic Segment, placing the transaction
// date/time at ORC-9 so the rendered line re-parses equal. Leaf values are
// escaped on write.
func (o ORC) Segment(enc EncodingCharacters) Segment {
	fields := newFields("ORC", 9)
	fields[1] = escapedLeaf(o.OrderControl, enc)      // ORC-1
	fields[2] = escapedLeaf(o.PlacerOrderNumber, enc) // ORC-2
	fields[3] = escapedLeaf(o.FillerOrderNumber, enc) // ORC-3
	fields[5] = escapedLeaf(o.OrderStatus, enc)       // ORC-5
	fields[9] = escapedLeaf(o.DateTime.String(), enc) // ORC-9
	return Segment{Fields: trimTrailingEmpty(fields), term: "\r"}
}

// Segment renders the OBR back to a generic Segment, placing the universal
// service identifier at OBR-4 and the observation date/time at OBR-7. Leaf values
// are escaped on write.
func (o OBR) Segment(enc EncodingCharacters) Segment {
	fields := newFields("OBR", 7)
	fields[1] = escapedLeaf(o.SetID, enc)                                      // OBR-1
	fields[2] = escapedLeaf(o.PlacerOrderNumber, enc)                          // OBR-2
	fields[3] = escapedLeaf(o.FillerOrderNumber, enc)                          // OBR-3
	fields[4] = escapedRepetitionField(o.UniversalServiceID.repetition(), enc) // OBR-4
	fields[7] = escapedLeaf(o.ObservationDateTime.String(), enc)               // OBR-7
	return Segment{Fields: trimTrailingEmpty(fields), term: "\r"}
}

// encodingChars renders the four MSH-2 characters in spec order
// (component/repetition/escape/subcomponent) so the rendered header re-derives
// the same EncodingCharacters on parse.
func encodingChars(enc EncodingCharacters) string {
	return string([]byte{enc.Component, enc.Repetition, enc.Escape, enc.Subcomponent})
}

// messageTypeRepetition renders a MessageType as the MSH-9 composite, trimming
// trailing empty components so a two-part type renders as "ACK^O01" rather than
// "ACK^O01^".
func messageTypeRepetition(t MessageType) Repetition {
	return componentsToRepetition([][]string{
		{t.Code},
		{t.TriggerEvent},
		{t.Structure},
	})
}

// escapedLeaf builds a flat single-value Field carrying the §2.10-escaped form of
// v, so a leaf value containing an in-band delimiter or a segment terminator is
// emitted as its escape sequence and re-parses to the original value.
func escapedLeaf(v string, enc EncodingCharacters) Field {
	return leaf(Escape(v, enc))
}

// escapedPatientIDField renders PID-3 from the full AllPatientIDs repetition list
// so a value such as "MRN~SSN" keeps every identifier through a typed round-trip,
// mirroring how ParsePID populated AllPatientIDs from allCX(field 3). When
// AllPatientIDs is empty it falls back to the single PatientID, so a PID built
// without the slice still renders its primary identifier. Each CX is escaped on
// write like the other composite renderers.
func escapedPatientIDField(p PID, enc EncodingCharacters) Field {
	ids := p.AllPatientIDs
	if len(ids) == 0 {
		return escapedRepetitionField(p.PatientID.repetition(), enc)
	}
	reps := make([]Repetition, 0, len(ids))
	for i := range ids {
		f := escapedRepetitionField(ids[i].repetition(), enc)
		reps = append(reps, f.Repetitions...)
	}
	return Field{Repetitions: reps}
}

// escapedRepetitionField builds a single-repetition Field from a composite
// repetition, escaping each leaf subcomponent against enc so an in-band delimiter
// inside a component (e.g. an '&' in a name) does not fracture the composite on
// re-parse. The component and subcomponent structure is preserved; only the leaf
// bytes are escaped.
func escapedRepetitionField(r Repetition, enc EncodingCharacters) Field {
	comps := make([]Component, len(r.Components))
	for i := range r.Components {
		subs := make([]string, len(r.Components[i].Subcomponents))
		for j, s := range r.Components[i].Subcomponents {
			subs[j] = Escape(s, enc)
		}
		comps[i] = Component{Subcomponents: subs}
	}
	return Field{Repetitions: []Repetition{{Components: comps}}}
}

// trimTrailingEmpty drops trailing wholly-empty fields so a rendered segment
// matches what the parser produces for the same value, keeping a constructed
// segment byte-stable through render->parse->render. The segment ID at index 0
// is always kept.
func trimTrailingEmpty(fields []Field) []Field {
	end := len(fields)
	for end > 1 && fieldIsEmpty(fields[end-1]) {
		end--
	}
	return fields[:end]
}

// ackConfig holds the resolved BuildACK options.
type ackConfig struct {
	controlID  func() string
	now        func() time.Time
	text       string
	sendingApp *HD
	sendingFac *HD
}

// ACKOption overrides a BuildACK default: the minted control-ID source, the ACK
// creation clock, the MSA-3 text, or the responder's sending application/facility.
type ACKOption func(*ackConfig)

// WithControlIDSource sets the source of the fresh MSH-10 control ID for the
// acknowledgement. It is injectable so tests are deterministic; the default is a
// monotonic synthetic generator that never derives the ID from message content.
func WithControlIDSource(src func() string) ACKOption {
	return func(c *ackConfig) {
		if src != nil {
			c.controlID = src
		}
	}
}

// WithACKClock sets the source of the acknowledgement's own creation time
// (MSH-7). It is injectable so tests are deterministic; the default is the wall
// clock. MSH-7 is the date/time the ACK is built, so it is taken from this clock
// rather than copied from the source message, whose MSH-7 may be older.
func WithACKClock(now func() time.Time) ACKOption {
	return func(c *ackConfig) {
		if now != nil {
			c.now = now
		}
	}
}

// WithACKText sets MSA-3, the human-readable acknowledgement text. It must not
// carry PHI; it is a diagnostic such as "OBX-5 failed datatype validation".
func WithACKText(text string) ACKOption {
	return func(c *ackConfig) { c.text = text }
}

// WithACKSendingApplication overrides MSH-3 of the acknowledgement (the
// responder's application) instead of echoing the source's receiving application.
func WithACKSendingApplication(app HD) ACKOption {
	return func(c *ackConfig) { c.sendingApp = &app }
}

// WithACKSendingFacility overrides MSH-4 of the acknowledgement (the responder's
// facility) instead of echoing the source's receiving facility.
func WithACKSendingFacility(fac HD) ACKOption {
	return func(c *ackConfig) { c.sendingFac = &fac }
}

// ackControlIDCounter backs the default control-ID generator. It is process-wide
// and monotonic so two acknowledgements built in the same nanosecond still get
// distinct IDs.
var ackControlIDCounter atomic.Uint64

// defaultControlID mints a synthetic, locally-unique MSH-10 from the wall clock
// and a monotonic counter. It never reads message content, so the minted ID
// cannot leak PHI.
func defaultControlID() string {
	n := ackControlIDCounter.Add(1)
	return "ACK" + strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.FormatUint(n, 36)
}

// dtmFromTime renders t as a second-precision HL7 DTM (YYYYMMDDHHMMSS), the form
// the package stamps a generated date/time at, and parses it back so MSH-7 stays
// a DTM whose lexical form round-trips through render and parse.
func dtmFromTime(t time.Time) DTM {
	d, _ := ParseDTM(t.UTC().Format("20060102150405"))
	return d
}

// BuildACK constructs a spec-correct acknowledgement for m per HL7 Chapter 2
// §2.9.2, following the field-swap logic of python-hl7's create_ack: the sending
// and receiving application/facility are swapped from the source MSH (MSH-3 with
// MSH-5, MSH-4 with MSH-6), MSH-9 becomes ACK^<inbound trigger>^ACK, a fresh
// MSH-10 is minted, and MSA-2 echoes the source MSH-10. MSA-1 is set to code,
// which selects original mode (AA/AE/AR) or enhanced mode (CA/CE/CR). A source
// message with no MSH returns a *SegmentError rather than producing a malformed
// reply.
func (m *Message) BuildACK(code AckCode, opts ...ACKOption) (*Message, error) {
	src, ok := m.MSH()
	if !ok {
		return nil, &SegmentError{Segment: "MSH", Reason: "cannot acknowledge a message without an MSH header"}
	}

	cfg := ackConfig{controlID: defaultControlID, now: time.Now}
	for _, opt := range opts {
		opt(&cfg)
	}

	sendingApp := src.ReceivingApplication
	sendingFac := src.ReceivingFacility
	if cfg.sendingApp != nil {
		sendingApp = *cfg.sendingApp
	}
	if cfg.sendingFac != nil {
		sendingFac = *cfg.sendingFac
	}

	ackHeader := MSH{
		SendingApplication:   sendingApp,
		SendingFacility:      sendingFac,
		ReceivingApplication: src.SendingApplication,
		ReceivingFacility:    src.SendingFacility,
		DateTime:             dtmFromTime(cfg.now()),
		MessageType:          MessageType{Code: "ACK", TriggerEvent: src.MessageType.TriggerEvent, Structure: "ACK"},
		ControlID:            cfg.controlID(),
		ProcessingID:         src.ProcessingID,
		VersionID:            src.VersionID,
	}

	ack := NewMessage(m.Enc)
	ack.SetMSH(ackHeader)
	ack.AppendSegment((MSA{
		AckCode:     code,
		ControlID:   src.ControlID,
		TextMessage: cfg.text,
	}).Segment(m.Enc))

	return ack, nil
}
