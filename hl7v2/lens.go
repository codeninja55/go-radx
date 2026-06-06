package hl7v2

import "iter"

// ResultGroup is one OBR with the OBX rows that follow it, the grouping an ORU
// message expresses through segment order. ORU.Results yields these in order.
type ResultGroup struct {
	Order        OBR
	Observations []OBX
}

// ADT — admission/discharge/transfer, a typed lens over a *Message. It does not
// copy the tree. The trigger event (A01, A04, ...) is MSH-9.2.
type ADT struct{ *Message }

// AsADT verifies MSH-9.1 is "ADT" and returns the typed view; the bool is false
// when MSH-9.1 does not match or the message has no MSH.
func AsADT(m *Message) (ADT, bool) {
	h, ok := m.MSH()
	if !ok || h.MessageType.Code != "ADT" {
		return ADT{}, false
	}
	return ADT{m}, true
}

// Event returns the ADT trigger event (MSH-9.2), e.g. "A01".
func (a ADT) Event() string {
	h, _ := a.MSH()
	return h.MessageType.TriggerEvent
}

// EVN returns the typed event segment and true, or false when absent.
func (a ADT) EVN() (EVN, bool) { return a.Message.EVN() }

// PID returns the typed patient identification and true, or false when absent.
func (a ADT) PID() (PID, bool) { return a.Message.PID() }

// PV1 returns the typed patient visit and true, or false when absent.
func (a ADT) PV1() (PV1, bool) { return a.Message.PV1() }

// ORU — observation result, a typed lens over a *Message. It does not copy the
// tree. Results groups each OBR with the OBX rows that follow it.
type ORU struct{ *Message }

// AsORU verifies MSH-9.1 is "ORU" and returns the typed view; the bool is false
// when MSH-9.1 does not match or the message has no MSH.
func AsORU(m *Message) (ORU, bool) {
	h, ok := m.MSH()
	if !ok || h.MessageType.Code != "ORU" {
		return ORU{}, false
	}
	return ORU{m}, true
}

// PID returns the typed patient identification and true, or false when absent.
func (o ORU) PID() (PID, bool) { return o.Message.PID() }

// Results yields each OBR with the OBX rows that follow it in segment order. A
// trailing OBR with no following OBX yields a ResultGroup with an empty
// Observations slice; OBX segments before the first OBR are not yielded (a
// well-formed ORU opens each result group with an OBR).
func (o ORU) Results() iter.Seq[ResultGroup] {
	return func(yield func(ResultGroup) bool) {
		var group *ResultGroup
		for _, seg := range o.Segments {
			switch seg.ID() {
			case "OBR":
				if group != nil && !yield(*group) {
					return
				}
				obr, err := ParseOBR(seg)
				if err != nil {
					group = nil
					continue
				}
				group = &ResultGroup{Order: obr}
			case "OBX":
				if group == nil {
					continue
				}
				if obx, err := ParseOBX(seg); err == nil {
					group.Observations = append(group.Observations, obx)
				}
			}
		}
		if group != nil {
			yield(*group)
		}
	}
}

// ACK — acknowledgement, a typed lens over a *Message (MSH + MSA, optional ERR).
// It does not copy the tree.
type ACK struct{ *Message }

// AsACK verifies MSH-9.1 is "ACK" and returns the typed view; the bool is false
// when MSH-9.1 does not match or the message has no MSH.
func AsACK(m *Message) (ACK, bool) {
	h, ok := m.MSH()
	if !ok || h.MessageType.Code != "ACK" {
		return ACK{}, false
	}
	return ACK{m}, true
}

// MSA returns the typed message-acknowledgement segment and true, or false when
// absent or when MSA-1 is not a recognised acknowledgement code.
func (a ACK) MSA() (MSA, bool) {
	seg, ok := a.Segment("MSA")
	if !ok {
		return MSA{}, false
	}
	msa, err := ParseMSA(seg)
	if err != nil {
		return MSA{}, false
	}
	return msa, true
}

// Errors returns every ERR segment in document order, skipping any malformed
// one. An ACK in original acknowledgement mode carries no ERR; an enhanced-mode
// negative acknowledgement may carry one per detected error.
func (a ACK) Errors() []ERR {
	var out []ERR
	for _, seg := range a.AllSegments("ERR") {
		if e, err := ParseERR(seg); err == nil {
			out = append(out, e)
		}
	}
	return out
}
