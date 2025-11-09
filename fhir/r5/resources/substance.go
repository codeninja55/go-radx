package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSubstance is the FHIR resource type name for Substance.
const ResourceTypeSubstance = "Substance"

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
	// Is this an instance of a substance or a kind of one
	Instance bool `json:"instance"`
	// active | inactive | entered-in-error
	Status *string `json:"status,omitempty"`
	// What class/type of substance this is
	Category []CodeableConcept `json:"category,omitempty"`
	// What substance this is
	Code CodeableReference `json:"code"`
	// Textual description of the substance, comments
	Description *string `json:"description,omitempty"`
	// When no longer valid to use
	Expiry *primitives.DateTime `json:"expiry,omitempty"`
	// Amount of substance in the package
	Quantity *Quantity `json:"quantity,omitempty"`
	// Composition information about the substance
	Ingredient []SubstanceIngredient `json:"ingredient,omitempty"`
}
