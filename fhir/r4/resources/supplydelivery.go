package resources

// ResourceTypeSupplyDelivery is the FHIR resource type name for SupplyDelivery.
const ResourceTypeSupplyDelivery = "SupplyDelivery"

// SupplyDeliverySuppliedItem represents a FHIR BackboneElement for SupplyDelivery.suppliedItem.
type SupplyDeliverySuppliedItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Amount dispensed
	Quantity *Quantity `json:"quantity,omitempty"`
	// Medication, Substance, or Device supplied
	Item *any `json:"item,omitempty"`
}

// SupplyDelivery represents a FHIR SupplyDelivery.
type SupplyDelivery struct {
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
	// External identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Fulfills plan, proposal or order
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// in-progress | completed | abandoned | entered-in-error
	Status *string `json:"status,omitempty"`
	// Patient for whom the item is supplied
	Patient *Reference `json:"patient,omitempty"`
	// Category of dispense event
	Type *CodeableConcept `json:"type,omitempty"`
	// The item that is delivered or supplied
	SuppliedItem *SupplyDeliverySuppliedItem `json:"suppliedItem,omitempty"`
	// When event occurred
	Occurrence *any `json:"occurrence,omitempty"`
	// Dispenser
	Supplier *Reference `json:"supplier,omitempty"`
	// Where the Supply was sent
	Destination *Reference `json:"destination,omitempty"`
	// Who collected the Supply
	Receiver []Reference `json:"receiver,omitempty"`
}
