package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir/r5"
	"github.com/codeninja55/go-radx/hl7v2"
)

// ORMToServiceRequestR5 converts an HL7 v2 order message (ORM^O01 or OMG^O19) to
// a FHIR R5 ServiceRequest. One hl7v2.OrderGroup (the common ORC with its OBR
// requests) becomes one ServiceRequest. A message carrying multiple order groups
// is rejected with ErrUnsupportedSource: the single-order-per-call limit is the
// documented v1 boundary, and the converter fails closed rather than guess which
// order to map (docs/reference/convert.md "ORMToServiceRequest"). Split the
// message into its OrderGroups upstream and call once per group.
//
// intent has no HL7 source and is defaulted to "order", recorded in
// Report.Defaulted. The subject carries the PID-3 patient identity logically
// (Reference.identifier), or the WithSubjectR5 reference when supplied — never a
// fabricated Reference.reference URL (the identity rule).
func ORMToServiceRequestR5(msg *hl7v2.Message, opts ...Option) (*r5.ServiceRequest, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if msg == nil {
		return nil, nil, fmt.Errorf("%w: message is nil", ErrMalformedSource)
	}

	orm, ok := hl7v2.AsORM(msg)
	if !ok {
		return nil, nil, fmt.Errorf("%w: MSH-9.1 is not ORM or OMG", ErrUnsupportedSource)
	}
	// AsORM accepts any ORM/OMG trigger; v1 supports only ORM^O01 and OMG^O19. A
	// different trigger structure maps order data differently, so reject it fail-
	// closed rather than guess (docs/reference/convert.md scope).
	if h, hasMSH := msg.MSH(); hasMSH && !supportedOrderTrigger(h.MessageType) {
		return nil, nil, fmt.Errorf("%w: order trigger %s^%s is not ORM^O01 or OMG^O19",
			ErrUnsupportedSource, h.MessageType.Code, h.MessageType.TriggerEvent)
	}

	groups := collectOrderGroups(orm)
	switch {
	case len(groups) == 0:
		return nil, nil, fmt.Errorf("%w: message carries no order group", ErrMalformedSource)
	case len(groups) > 1:
		// Fail closed: v1 maps one order per call. Mapping the first and dropping
		// the rest would silently lose orders, so reject the whole message.
		return nil, nil, fmt.Errorf("%w: message carries %d order groups, v1 maps one per call",
			ErrUnsupportedSource, len(groups))
	}

	group := groups[0]
	sr := &r5.ServiceRequest{}

	appendIdentifier(&sr.Identifier, group.Common.PlacerOrderNumber)
	appendIdentifier(&sr.Identifier, group.Common.FillerOrderNumber)

	sr.Status = orderStatus(group.Common)

	// intent is required by FHIR and has no HL7 source; default it and record the
	// default so the mapping decision is auditable.
	sr.Intent = "order"
	report.defaulted("ServiceRequest.intent", "order", "ORM has no intent field; defaulted per convert.md")

	if len(group.Requests) > 0 {
		obr := group.Requests[0]
		if code := serviceCode(obr.UniversalServiceID); code != nil {
			sr.Code = &r5.CodeableReference{Concept: code}
		}
		for i := range group.Requests[1:] {
			// Name the structural locus only; the ordered-procedure code is
			// clinical data and must not appear in a Report (PRD §9.1 no-PHI).
			report.dropped(
				fmt.Sprintf("OBR-4 UniversalServiceID (request %d)", i+2),
				"v1 ServiceRequest carries a single code; extra OBR requests are not mapped",
			)
		}
	}

	if authored := hl7DateTimeToFHIR(group.Common.DateTime, report, "ServiceRequest.authoredOn"); authored != "" {
		sr.AuthoredOn = &authored
	}

	if subject := patientSubjectR5(cfg, msg, report, "ServiceRequest.subject"); subject != nil {
		sr.Subject = subject
	}

	rep, err := cfg.finalize(report)
	return sr, rep, err
}

// supportedOrderTrigger reports whether the message type is one of the v1-scoped
// order triggers: ORM^O01 or OMG^O19.
func supportedOrderTrigger(mt hl7v2.MessageType) bool {
	switch {
	case mt.Code == "ORM" && mt.TriggerEvent == "O01":
		return true
	case mt.Code == "OMG" && mt.TriggerEvent == "O19":
		return true
	default:
		return false
	}
}

// collectOrderGroups materialises the message's order groups so the count can be
// checked before mapping (the iterator is consumed once).
func collectOrderGroups(orm hl7v2.ORM) []hl7v2.OrderGroup {
	var groups []hl7v2.OrderGroup
	for g := range orm.Orders() {
		groups = append(groups, g)
	}
	return groups
}

// orderStatus maps ORC-1 Order Control / ORC-5 Order Status to a FHIR
// ServiceRequest.status. ORC-1 is the authoritative control code; ORC-5 is a
// secondary signal. An unrecognised pair defaults to "active".
func orderStatus(orc hl7v2.ORC) string {
	switch orc.OrderControl {
	case "NW", "XO":
		return "active"
	case "CA":
		return "revoked"
	case "CM":
		return "completed"
	}
	switch orc.OrderStatus {
	case "CM":
		return "completed"
	case "CA":
		return "revoked"
	}
	return "active"
}

// serviceCode maps an OBR-4 CWE to a FHIR CodeableConcept, or nil when the code
// is empty. The Coding is carried across verbatim — go-radx does not translate
// between code systems.
func serviceCode(cwe hl7v2.CWE) *r5.CodeableConcept {
	if cwe.Code == "" && cwe.Text == "" {
		return nil
	}
	coding := r5.Coding{}
	if cwe.Code != "" {
		code := cwe.Code
		coding.Code = &code
	}
	if cwe.Text != "" {
		display := cwe.Text
		coding.Display = &display
	}
	if cwe.CodingSystem != "" {
		system := cwe.CodingSystem
		coding.System = &system
	}
	cc := &r5.CodeableConcept{Coding: []r5.Coding{coding}}
	if cwe.Text != "" {
		text := cwe.Text
		cc.Text = &text
	}
	return cc
}

// appendIdentifier appends a FHIR Identifier built from an HL7 EI order number,
// skipping an empty value. The order number carries no assigning-authority
// namespace in the typed ORC view, so only the value is set.
func appendIdentifier(ids *[]r5.Identifier, value string) {
	if value == "" {
		return
	}
	v := value
	*ids = append(*ids, r5.Identifier{Value: &v})
}
