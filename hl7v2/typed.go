package hl7v2

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
