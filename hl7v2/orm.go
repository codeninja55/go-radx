package hl7v2

import "iter"

// OrderGroup is one ORC with the OBR requests that follow it, the grouping an
// ORM/OMG message expresses through segment order. ORM.Orders yields these.
type OrderGroup struct {
	Common   ORC
	Requests []OBR
}

// ORM is an order message (ORC + OBR groups), a typed lens over a *Message. It
// does not copy the tree.
type ORM struct{ *Message }

// AsORM verifies MSH-9.1 is "ORM" (or "OMG", the imaging-order variant) and
// returns the typed view; the bool is false when MSH-9.1 does not match.
func AsORM(m *Message) (ORM, bool) {
	h, ok := m.MSH()
	if !ok {
		return ORM{}, false
	}
	switch h.MessageType.Code {
	case "ORM", "OMG":
		return ORM{m}, true
	default:
		return ORM{}, false
	}
}

// OMG is a general clinical order message (OMG^O19, the imaging-order variant
// of ORM introduced in v2.4), a typed lens over a *Message. It does not copy
// the tree. AsORM also admits OMG codes; this dedicated lens is for callers
// that must distinguish the OMG variant from a classic ORM.
type OMG struct{ *Message }

// AsOMG verifies MSH-9.1 is "OMG" and returns the typed view; the bool is
// false when MSH-9.1 does not match or the message has no MSH.
func AsOMG(m *Message) (OMG, bool) {
	h, ok := m.MSH()
	if !ok || h.MessageType.Code != "OMG" {
		return OMG{}, false
	}
	return OMG{m}, true
}

// Orders yields each ORC with the OBR requests that follow it in segment
// order, the same grouping ORM.Orders applies: an OMG expresses its orders
// through the identical ORC+OBR segment structure.
func (o OMG) Orders() iter.Seq[OrderGroup] { return ORM(o).Orders() }

// Orders yields each ORC with the OBR requests that follow it in segment order.
// A trailing ORC with no following OBR yields an OrderGroup with an empty
// Requests slice; OBR segments before the first ORC are not yielded (a
// well-formed ORM opens each order with an ORC).
func (o ORM) Orders() iter.Seq[OrderGroup] {
	return func(yield func(OrderGroup) bool) {
		var group *OrderGroup
		for _, seg := range o.Segments {
			switch seg.ID() {
			case "ORC":
				if group != nil && !yield(*group) {
					return
				}
				orc, err := ParseORC(seg)
				if err != nil {
					group = nil
					continue
				}
				group = &OrderGroup{Common: orc}
			case "OBR":
				if group == nil {
					continue
				}
				if obr, err := ParseOBR(seg); err == nil {
					group.Requests = append(group.Requests, obr)
				}
			}
		}
		if group != nil {
			yield(*group)
		}
	}
}
