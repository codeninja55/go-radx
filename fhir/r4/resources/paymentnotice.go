package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypePaymentNotice is the FHIR resource type name for PaymentNotice.
const ResourceTypePaymentNotice = "PaymentNotice"

// PaymentNotice represents a FHIR PaymentNotice.
type PaymentNotice struct {
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
	// Business Identifier for the payment noctice
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | cancelled | draft | entered-in-error
	Status string `json:"status"`
	// Request reference
	Request *Reference `json:"request,omitempty"`
	// Response reference
	Response *Reference `json:"response,omitempty"`
	// Creation date
	Created primitives.DateTime `json:"created"`
	// Responsible practitioner
	Provider *Reference `json:"provider,omitempty"`
	// Payment reference
	Payment Reference `json:"payment"`
	// Payment or clearing date
	PaymentDate *primitives.Date `json:"paymentDate,omitempty"`
	// Party being paid
	Payee *Reference `json:"payee,omitempty"`
	// Party being notified
	Recipient Reference `json:"recipient"`
	// Monetary amount of the payment
	Amount Money `json:"amount"`
	// Issued or cleared Status of the payment
	PaymentStatus *CodeableConcept `json:"paymentStatus,omitempty"`
}
