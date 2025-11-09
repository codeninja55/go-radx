package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypePaymentReconciliation is the FHIR resource type name for PaymentReconciliation.
const ResourceTypePaymentReconciliation = "PaymentReconciliation"

// PaymentReconciliationAllocation represents a FHIR BackboneElement for PaymentReconciliation.allocation.
type PaymentReconciliationAllocation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Business identifier of the payment detail
	Identifier *Identifier `json:"identifier,omitempty"`
	// Business identifier of the prior payment detail
	Predecessor *Identifier `json:"predecessor,omitempty"`
	// Subject of the payment
	Target *Reference `json:"target,omitempty"`
	// Sub-element of the subject
	TargetItem *any `json:"targetItem,omitempty"`
	// Applied-to encounter
	Encounter *Reference `json:"encounter,omitempty"`
	// Applied-to account
	Account *Reference `json:"account,omitempty"`
	// Category of payment
	Type *CodeableConcept `json:"type,omitempty"`
	// Submitter of the request
	Submitter *Reference `json:"submitter,omitempty"`
	// Response committing to a payment
	Response *Reference `json:"response,omitempty"`
	// Date of commitment to pay
	Date *primitives.Date `json:"date,omitempty"`
	// Contact for the response
	Responsible *Reference `json:"responsible,omitempty"`
	// Recipient of the payment
	Payee *Reference `json:"payee,omitempty"`
	// Amount allocated to this payable
	Amount *Money `json:"amount,omitempty"`
}

// PaymentReconciliationProcessNote represents a FHIR BackboneElement for PaymentReconciliation.processNote.
type PaymentReconciliationProcessNote struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// display | print | printoper
	Type *string `json:"type,omitempty"`
	// Note explanatory text
	Text *string `json:"text,omitempty"`
}

// PaymentReconciliation represents a FHIR PaymentReconciliation.
type PaymentReconciliation struct {
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
	// Business Identifier for a payment reconciliation
	Identifier []Identifier `json:"identifier,omitempty"`
	// Category of payment
	Type CodeableConcept `json:"type"`
	// active | cancelled | draft | entered-in-error
	Status string `json:"status"`
	// Workflow originating payment
	Kind *CodeableConcept `json:"kind,omitempty"`
	// Period covered
	Period *Period `json:"period,omitempty"`
	// Creation date
	Created primitives.DateTime `json:"created"`
	// Who entered the payment
	Enterer *Reference `json:"enterer,omitempty"`
	// Nature of the source
	IssuerType *CodeableConcept `json:"issuerType,omitempty"`
	// Party generating payment
	PaymentIssuer *Reference `json:"paymentIssuer,omitempty"`
	// Reference to requesting resource
	Request *Reference `json:"request,omitempty"`
	// Responsible practitioner
	Requestor *Reference `json:"requestor,omitempty"`
	// queued | complete | error | partial
	Outcome *string `json:"outcome,omitempty"`
	// Disposition message
	Disposition *string `json:"disposition,omitempty"`
	// When payment issued
	Date primitives.Date `json:"date"`
	// Where payment collected
	Location *Reference `json:"location,omitempty"`
	// Payment instrument
	Method *CodeableConcept `json:"method,omitempty"`
	// Type of card
	CardBrand *string `json:"cardBrand,omitempty"`
	// Digits for verification
	AccountNumber *string `json:"accountNumber,omitempty"`
	// Expiration year-month
	ExpirationDate *primitives.Date `json:"expirationDate,omitempty"`
	// Processor name
	Processor *string `json:"processor,omitempty"`
	// Check number or payment reference
	ReferenceNumber *string `json:"referenceNumber,omitempty"`
	// Authorization number
	Authorization *string `json:"authorization,omitempty"`
	// Amount offered by the issuer
	TenderedAmount *Money `json:"tenderedAmount,omitempty"`
	// Amount returned by the receiver
	ReturnedAmount *Money `json:"returnedAmount,omitempty"`
	// Total amount of Payment
	Amount Money `json:"amount"`
	// Business identifier for the payment
	PaymentIdentifier *Identifier `json:"paymentIdentifier,omitempty"`
	// Settlement particulars
	Allocation []PaymentReconciliationAllocation `json:"allocation,omitempty"`
	// Printed form identifier
	FormCode *CodeableConcept `json:"formCode,omitempty"`
	// Note concerning processing
	ProcessNote []PaymentReconciliationProcessNote `json:"processNote,omitempty"`
}
