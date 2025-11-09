package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSubstance is the FHIR resource type name for Substance.
const ResourceTypeSubstance = "Substance"

// SubstanceInstance represents a FHIR BackboneElement for Substance.instance.
type SubstanceInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifier of the package/container
	Identifier *Identifier `json:"identifier,omitempty"`
	// When no longer valid to use
	Expiry *primitives.DateTime `json:"expiry,omitempty"`
	// Amount of substance in the package
	Quantity *Quantity `json:"quantity,omitempty"`
}

// SubstanceIngredient represents a FHIR BackboneElement for Substance.ingredient.
type SubstanceIngredient struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Optional amount (concentration)
	Quantity *Ratio `json:"quantity,omitempty"`
	// A component of the substance
	Substance any `json:"substance"`
}

// Substance represents a FHIR Substance.
type Substance struct {
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
	// Unique identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | inactive | entered-in-error
	Status *string `json:"status,omitempty"`
	// What class/type of substance this is
	Category []CodeableConcept `json:"category,omitempty"`
	// What substance this is
	Code CodeableConcept `json:"code"`
	// Textual description of the substance, comments
	Description *string `json:"description,omitempty"`
	// If this describes a specific package/container of the substance
	Instance []SubstanceInstance `json:"instance,omitempty"`
	// Composition information about the substance
	Ingredient []SubstanceIngredient `json:"ingredient,omitempty"`
}
