package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeInvoice is the FHIR resource type name for Invoice.
const ResourceTypeInvoice = "Invoice"

// InvoiceParticipant represents a FHIR BackboneElement for Invoice.participant.
type InvoiceParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of involvement in creation of this Invoice
	Role *CodeableConcept `json:"role,omitempty"`
	// Individual who was involved
	Actor Reference `json:"actor"`
}

// InvoiceLineItem represents a FHIR BackboneElement for Invoice.lineItem.
type InvoiceLineItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Sequence number of line item
	Sequence *int `json:"sequence,omitempty"`
	// Service data or period
	Serviced *any `json:"serviced,omitempty"`
	// Reference to ChargeItem containing details of this line item or an inline billing code
	ChargeItem any `json:"chargeItem"`
	// Components of total line item price
	PriceComponent []MonetaryComponent `json:"priceComponent,omitempty"`
}

// Invoice represents a FHIR Invoice.
type Invoice struct {
	// Logical id of this artifact
	ID *string `json:"id,omitempty"`
	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty"`
	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty"`
	// Language of the resource content
	Language *string `json:"language,omitempty"`
	// Text summary of the resource, for human interpretation
	Text *Narrative `json:"text,omitempty"`
	// Contained, inline Resources
	Contained []any `json:"contained,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Business Identifier for item
	Identifier []Identifier `json:"identifier,omitempty"`
	// draft | issued | balanced | cancelled | entered-in-error
	Status string `json:"status"`
	// Reason for cancellation of this Invoice
	CancelledReason *string `json:"cancelledReason,omitempty"`
	// Type of Invoice
	Type *CodeableConcept `json:"type,omitempty"`
	// Recipient(s) of goods and services
	Subject *Reference `json:"subject,omitempty"`
	// Recipient of this invoice
	Recipient *Reference `json:"recipient,omitempty"`
	// DEPRICATED
	Date *primitives.DateTime `json:"date,omitempty"`
	// When posted
	Creation *primitives.DateTime `json:"creation,omitempty"`
	// Billing date or period
	Period *any `json:"period,omitempty"`
	// Participant in creation of this Invoice
	Participant []InvoiceParticipant `json:"participant,omitempty"`
	// Issuing Organization of Invoice
	Issuer *Reference `json:"issuer,omitempty"`
	// Account that is being balanced
	Account *Reference `json:"account,omitempty"`
	// Line items of this Invoice
	LineItem []InvoiceLineItem `json:"lineItem,omitempty"`
	// Components of Invoice total
	TotalPriceComponent []MonetaryComponent `json:"totalPriceComponent,omitempty"`
	// Net total of this Invoice
	TotalNet *Money `json:"totalNet,omitempty"`
	// Gross total of this Invoice
	TotalGross *Money `json:"totalGross,omitempty"`
	// Payment details
	PaymentTerms *string `json:"paymentTerms,omitempty"`
	// Comments made about the invoice
	Note []Annotation `json:"note,omitempty"`
}
