package convert

import (
	"fmt"

	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/hl7v2"
)

// ORMToServiceRequestR4 converts an HL7 v2 order message (ORM^O01 or OMG^O19) to a
// FHIR R4 ServiceRequest, the R4 twin of ORMToServiceRequestR5. The HL7 reading,
// the single-order-per-call boundary, status/priority/occurrence mapping, and the
// identity rule are identical. The R4 output differs in two load-bearing ways the
// R4 resource model imposes:
//
//   - ServiceRequest.code is a CodeableConcept in R4 (CodeableReference in R5).
//   - R4 has no CodeableReference: where R5 carries the reason as
//     ServiceRequest.reason (CodeableReference), R4 splits it into reasonCode
//     (CodeableConcept) and reasonReference (Reference); the OBR-31 reason for
//     study becomes a reasonCode.
//
// intent has no HL7 source and is defaulted to "order". The subject carries the
// PID-3 patient identity logically (Reference.identifier), or the WithSubjectR4
// reference when supplied — never a fabricated Reference.reference URL.
func ORMToServiceRequestR4(msg *hl7v2.Message, opts ...Option) (*r4.ServiceRequest, *Report, error) {
	cfg := newConfig(opts...)
	report := &Report{}

	if msg == nil {
		return nil, nil, fmt.Errorf("%w: message is nil", ErrMalformedSource)
	}

	orm, ok := hl7v2.AsORM(msg)
	if !ok {
		return nil, nil, fmt.Errorf("%w: MSH-9.1 is not ORM or OMG", ErrUnsupportedSource)
	}
	if h, hasMSH := msg.MSH(); hasMSH && !supportedOrderTrigger(h.MessageType) {
		return nil, nil, fmt.Errorf("%w: order trigger %s^%s is not ORM^O01 or OMG^O19",
			ErrUnsupportedSource, h.MessageType.Code, h.MessageType.TriggerEvent)
	}

	groups := collectOrderGroups(orm)
	switch {
	case len(groups) == 0:
		return nil, nil, fmt.Errorf("%w: message carries no order group", ErrMalformedSource)
	case len(groups) > 1:
		return nil, nil, fmt.Errorf("%w: message carries %d order groups, v1 maps one per call",
			ErrUnsupportedSource, len(groups))
	}

	group := groups[0]
	sr := &r4.ServiceRequest{}

	appendIdentifierR4(&sr.Identifier, group.Common.PlacerOrderNumber)
	appendIdentifierR4(&sr.Identifier, group.Common.FillerOrderNumber)

	status := orderStatusR4(group.Common)
	sr.Status = &status

	intent := r4.RequestIntentOrder
	sr.Intent = &intent
	report.defaulted("ServiceRequest.intent", "order", "ORM has no intent field; defaulted per convert.md")

	if len(group.Requests) > 0 {
		obr := group.Requests[0]
		if code := serviceCodeR4(obr.UniversalServiceID); code != nil {
			sr.Code = code
		}
		for i := range group.Requests[1:] {
			report.dropped(
				fmt.Sprintf("OBR-4 UniversalServiceID (request %d)", i+2),
				"v1 ServiceRequest carries a single code; extra OBR requests are not mapped",
			)
		}
	}

	if authored := hl7DateTimeToFHIR(group.Common.DateTime, report, "ServiceRequest.authoredOn"); authored != "" {
		sr.AuthoredOn = &authored
	}

	if priority, ok := orderPriorityR4(msg, report); ok {
		sr.Priority = &priority
	}

	if subject := patientSubjectR4(cfg, msg, report, "ServiceRequest.subject"); subject != nil {
		sr.Subject = subject
	}

	if encounter := encounterReferenceR4(msg); encounter != nil {
		sr.Encounter = encounter
	}

	if requester := requesterReferenceR4(msg); requester != nil {
		sr.Requester = requester
	}

	sr.ReasonCode = append(sr.ReasonCode, reasonForStudyR4(msg)...)

	if occ := orderOccurrence(msg, report); occ != "" {
		sr.SetOccurrenceDateTime(r4.FHIRDateTime(occ))
	}

	rep, err := cfg.finalize(report)
	return sr, rep, err
}

// orderStatusR4 maps ORC-1 Order Control / ORC-5 Order Status to an R4
// ServiceRequest.status, the R4 twin of orderStatus. The mapping is identical; the
// RequestStatus enum lives in the R4 sub-package.
func orderStatusR4(orc hl7v2.ORC) r4.RequestStatus {
	switch orc.OrderControl {
	case "NW", "XO":
		return r4.RequestStatusActive
	case "CA":
		return r4.RequestStatusRevoked
	case "CM":
		return r4.RequestStatusCompleted
	}
	switch orc.OrderStatus {
	case "CM":
		return r4.RequestStatusCompleted
	case "CA":
		return r4.RequestStatusRevoked
	}
	return r4.RequestStatusActive
}

// serviceCodeR4 maps an OBR-4 CWE to an R4 CodeableConcept, the R4 twin of
// serviceCode. The Coding is carried verbatim.
func serviceCodeR4(cwe hl7v2.CWE) *r4.CodeableConcept {
	if cwe.Code == "" && cwe.Text == "" {
		return nil
	}
	coding := r4.Coding{}
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
	cc := &r4.CodeableConcept{Coding: []r4.Coding{coding}}
	if cwe.Text != "" {
		text := cwe.Text
		cc.Text = &text
	}
	return cc
}

// appendIdentifierR4 appends an R4 Identifier built from an HL7 EI order number,
// the R4 twin of appendIdentifier.
func appendIdentifierR4(ids *[]r4.Identifier, value string) {
	if value == "" {
		return
	}
	v := value
	*ids = append(*ids, r4.Identifier{Value: &v})
}

// orderPriorityR4 maps the HL7 v2 Quantity/Timing priority to an R4
// RequestPriority, the R4 twin of orderPriority. The mapping and the no-PHI drop
// rule are identical.
func orderPriorityR4(msg *hl7v2.Message, report *Report) (r4.RequestPriority, bool) {
	code := firstNonEmpty(getField(msg, "ORC-7-1-6"), getField(msg, "OBR-27-1-6"))
	if code == "" {
		return "", false
	}
	switch code {
	case "S":
		return r4.RequestPriorityStat, true
	case "A":
		return r4.RequestPriorityAsap, true
	case "R", "P":
		return r4.RequestPriorityRoutine, true
	default:
		report.dropped("ORC-7 / OBR-27 priority",
			"the Quantity/Timing priority code is not in HL7 Table 0027; ServiceRequest.priority needs a bound code, so it was dropped")
		return "", false
	}
}

// encounterReferenceR4 builds the ServiceRequest.encounter Reference from the
// PV1-19 Visit Number, the R4 twin of encounterReferenceR5. The visit number is
// carried as a logical Reference.identifier — never a fabricated URL.
func encounterReferenceR4(msg *hl7v2.Message) *r4.Reference {
	pv1, ok := msg.PV1()
	if !ok || pv1.VisitNumber.ID == "" {
		return nil
	}
	id := cxToIdentifierR4(pv1.VisitNumber)
	refType := encounterReferenceType
	return &r4.Reference{Type: &refType, Identifier: &id}
}

// requesterReferenceR4 builds the ServiceRequest.requester Reference from the
// ORC-12 Ordering Provider, the R4 twin of requesterReferenceR5. The provider's ID
// becomes a logical Reference.identifier and the name the display — never a
// fabricated URL.
func requesterReferenceR4(msg *hl7v2.Message) *r4.Reference {
	id := getField(msg, "ORC-12-1-1")
	family := getField(msg, "ORC-12-1-2")
	given := getField(msg, "ORC-12-1-3")
	if id == "" && family == "" && given == "" {
		return nil
	}

	refType := requesterReferenceType
	ref := &r4.Reference{Type: &refType}
	if id != "" {
		value := id
		ref.Identifier = &r4.Identifier{Value: &value}
	}
	if display := providerDisplay(family, given); display != "" {
		ref.Display = &display
	}
	return ref
}

// reasonForStudyR4 maps OBR-31 Reason for Study (a CWE) to an R4
// ServiceRequest.reasonCode CodeableConcept, the R4 twin of reasonForStudy. R4 has
// no CodeableReference, so the reason becomes a reasonCode rather than the single
// reason CodeableReference R5 uses.
func reasonForStudyR4(msg *hl7v2.Message) []r4.CodeableConcept {
	code := getField(msg, "OBR-31-1-1")
	display := getField(msg, "OBR-31-1-2")
	system := getField(msg, "OBR-31-1-3")
	if code == "" && display == "" {
		return nil
	}
	coding := r4.Coding{}
	if code != "" {
		c := code
		coding.Code = &c
	}
	if display != "" {
		d := display
		coding.Display = &d
	}
	if system != "" {
		s := system
		coding.System = &s
	}
	cc := r4.CodeableConcept{Coding: []r4.Coding{coding}}
	if display != "" {
		t := display
		cc.Text = &t
	}
	return []r4.CodeableConcept{cc}
}
