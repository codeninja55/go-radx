package resources

// ResourceTypeManufacturedItemDefinition is the FHIR resource type name for ManufacturedItemDefinition.
const ResourceTypeManufacturedItemDefinition = "ManufacturedItemDefinition"

// ManufacturedItemDefinitionProperty represents a FHIR BackboneElement for ManufacturedItemDefinition.property.
type ManufacturedItemDefinitionProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A code expressing the type of characteristic
	Type CodeableConcept `json:"type"`
	// A value for the characteristic
	Value *any `json:"value,omitempty"`
}

// ManufacturedItemDefinitionComponentConstituent represents a FHIR BackboneElement for ManufacturedItemDefinition.component.constituent.
type ManufacturedItemDefinitionComponentConstituent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The measurable amount of the substance, expressable in different ways (e.g. by mass or volume)
	Amount []Quantity `json:"amount,omitempty"`
	// The physical location of the constituent/ingredient within the component
	Location []CodeableConcept `json:"location,omitempty"`
	// The function of this constituent within the component e.g. binder
	Function []CodeableConcept `json:"function,omitempty"`
	// The ingredient that is the constituent of the given component
	HasIngredient []CodeableReference `json:"hasIngredient,omitempty"`
}

// ManufacturedItemDefinitionComponentProperty represents a FHIR BackboneElement for ManufacturedItemDefinition.component.property.
type ManufacturedItemDefinitionComponentProperty struct {
}

// ManufacturedItemDefinitionComponentComponent represents a FHIR BackboneElement for ManufacturedItemDefinition.component.component.
type ManufacturedItemDefinitionComponentComponent struct {
}

// ManufacturedItemDefinitionComponent represents a FHIR BackboneElement for ManufacturedItemDefinition.component.
type ManufacturedItemDefinitionComponent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Defining type of the component e.g. shell, layer, ink
	Type CodeableConcept `json:"type"`
	// The function of this component within the item e.g. delivers active ingredient, masks taste
	Function []CodeableConcept `json:"function,omitempty"`
	// The measurable amount of total quantity of all substances in the component, expressable in different ways (e.g. by mass or volume)
	Amount []Quantity `json:"amount,omitempty"`
	// A reference to a constituent of the manufactured item as a whole, linked here so that its component location within the item can be indicated. This not where the item's ingredient are primarily stated (for which see Ingredient.for or ManufacturedItemDefinition.ingredient)
	Constituent []ManufacturedItemDefinitionComponentConstituent `json:"constituent,omitempty"`
	// General characteristics of this component
	Property []ManufacturedItemDefinitionComponentProperty `json:"property,omitempty"`
	// A component that this component contains or is made from
	Component []ManufacturedItemDefinitionComponentComponent `json:"component,omitempty"`
}

// ManufacturedItemDefinition represents a FHIR ManufacturedItemDefinition.
type ManufacturedItemDefinition struct {
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
	// draft | active | retired | unknown
	Status string `json:"status"`
	// A descriptive name applied to this item
	Name *string `json:"name,omitempty"`
	// Dose form as manufactured (before any necessary transformation)
	ManufacturedDoseForm CodeableConcept `json:"manufacturedDoseForm"`
	// The “real-world” units in which the quantity of the item is described
	UnitOfPresentation *CodeableConcept `json:"unitOfPresentation,omitempty"`
	// Manufacturer of the item, one of several possible
	Manufacturer []Reference `json:"manufacturer,omitempty"`
	// Allows specifying that an item is on the market for sale, or that it is not available, and the dates and locations associated
	MarketingStatus []MarketingStatus `json:"marketingStatus,omitempty"`
	// The ingredients of this manufactured item. Only needed if these are not specified by incoming references from the Ingredient resource
	Ingredient []CodeableConcept `json:"ingredient,omitempty"`
	// General characteristics of this item
	Property []ManufacturedItemDefinitionProperty `json:"property,omitempty"`
	// Physical parts of the manufactured item, that it is intrisically made from. This is distinct from the ingredients that are part of its chemical makeup
	Component []ManufacturedItemDefinitionComponent `json:"component,omitempty"`
}
