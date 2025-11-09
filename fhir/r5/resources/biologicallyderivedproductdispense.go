package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeBiologicallyDerivedProductDispense is the FHIR resource type name for BiologicallyDerivedProductDispense.
const ResourceTypeBiologicallyDerivedProductDispense = "BiologicallyDerivedProductDispense"

// BiologicallyDerivedProductDispensePerformer represents a FHIR BackboneElement for BiologicallyDerivedProductDispense.performer.
type BiologicallyDerivedProductDispensePerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifies the function of the performer during the dispense
	Function *CodeableConcept `json:"function,omitempty"`
	// Who performed the action
	Actor Reference `json:"actor"`
}

// BiologicallyDerivedProductDispense represents a FHIR BiologicallyDerivedProductDispense.
type BiologicallyDerivedProductDispense struct {
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
	// Business identifier for this dispense
	Identifier []Identifier `json:"identifier,omitempty"`
	// The order or request that this dispense is fulfilling
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Short description
	PartOf []Reference `json:"partOf,omitempty"`
	// preparation | in-progress | allocated | issued | unfulfilled | returned | entered-in-error | unknown
	Status string `json:"status"`
	// Relationship between the donor and intended recipient
	OriginRelationshipType *CodeableConcept `json:"originRelationshipType,omitempty"`
	// The BiologicallyDerivedProduct that is dispensed
	Product Reference `json:"product"`
	// The intended recipient of the dispensed product
	Patient Reference `json:"patient"`
	// Indicates the type of matching associated with the dispense
	MatchStatus *CodeableConcept `json:"matchStatus,omitempty"`
	// Indicates who or what performed an action
	Performer []BiologicallyDerivedProductDispensePerformer `json:"performer,omitempty"`
	// Where the dispense occurred
	Location *Reference `json:"location,omitempty"`
	// Amount dispensed
	Quantity *Quantity `json:"quantity,omitempty"`
	// When product was selected/matched
	PreparedDate *primitives.DateTime `json:"preparedDate,omitempty"`
	// When the product was dispatched
	WhenHandedOver *primitives.DateTime `json:"whenHandedOver,omitempty"`
	// Where the product was dispatched to
	Destination *Reference `json:"destination,omitempty"`
	// Additional notes
	Note []Annotation `json:"note,omitempty"`
	// Specific instructions for use
	UsageInstruction *string `json:"usageInstruction,omitempty"`
}
