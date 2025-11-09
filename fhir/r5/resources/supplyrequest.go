package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSupplyRequest is the FHIR resource type name for SupplyRequest.
const ResourceTypeSupplyRequest = "SupplyRequest"

// SupplyRequestParameter represents a FHIR BackboneElement for SupplyRequest.parameter.
type SupplyRequestParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Item detail
	Code *CodeableConcept `json:"code,omitempty"`
	// Value of detail
	Value *any `json:"value,omitempty"`
}

// SupplyRequest represents a FHIR SupplyRequest.
type SupplyRequest struct {
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
	// Business Identifier for SupplyRequest
	Identifier []Identifier `json:"identifier,omitempty"`
	// draft | active | suspended +
	Status *string `json:"status,omitempty"`
	// What other request is fulfilled by this supply request
	BasedOn []Reference `json:"basedOn,omitempty"`
	// The kind of supply (central, non-stock, etc.)
	Category *CodeableConcept `json:"category,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// The patient for who the supply request is for
	DeliverFor *Reference `json:"deliverFor,omitempty"`
	// Medication, Substance, or Device requested to be supplied
	Item CodeableReference `json:"item"`
	// The requested amount of the item indicated
	Quantity Quantity `json:"quantity"`
	// Ordered item details
	Parameter []SupplyRequestParameter `json:"parameter,omitempty"`
	// When the request should be fulfilled
	Occurrence *any `json:"occurrence,omitempty"`
	// When the request was made
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Individual making the request
	Requester *Reference `json:"requester,omitempty"`
	// Who is intended to fulfill the request
	Supplier []Reference `json:"supplier,omitempty"`
	// The reason why the supply item was requested
	Reason []CodeableReference `json:"reason,omitempty"`
	// The origin of the supply
	DeliverFrom *Reference `json:"deliverFrom,omitempty"`
	// The destination of the supply
	DeliverTo *Reference `json:"deliverTo,omitempty"`
}
