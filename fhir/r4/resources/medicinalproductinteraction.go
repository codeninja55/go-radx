package resources

// ResourceTypeMedicinalProductInteraction is the FHIR resource type name for MedicinalProductInteraction.
const ResourceTypeMedicinalProductInteraction = "MedicinalProductInteraction"

// MedicinalProductInteractionInteractant represents a FHIR BackboneElement for MedicinalProductInteraction.interactant.
type MedicinalProductInteractionInteractant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The specific medication, food or laboratory test that interacts
	Item any `json:"item"`
}

// MedicinalProductInteraction represents a FHIR MedicinalProductInteraction.
type MedicinalProductInteraction struct {
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
	// The medication for which this is a described interaction
	Subject []Reference `json:"subject,omitempty"`
	// The interaction described
	Description *string `json:"description,omitempty"`
	// The specific medication, food or laboratory test that interacts
	Interactant []MedicinalProductInteractionInteractant `json:"interactant,omitempty"`
	// The type of the interaction e.g. drug-drug interaction, drug-food interaction, drug-lab test interaction
	Type *CodeableConcept `json:"type,omitempty"`
	// The effect of the interaction, for example "reduced gastric absorption of primary medication"
	Effect *CodeableConcept `json:"effect,omitempty"`
	// The incidence of the interaction, e.g. theoretical, observed
	Incidence *CodeableConcept `json:"incidence,omitempty"`
	// Actions for managing the interaction
	Management *CodeableConcept `json:"management,omitempty"`
}
