package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeChargeItemDefinition is the FHIR resource type name for ChargeItemDefinition.
const ResourceTypeChargeItemDefinition = "ChargeItemDefinition"

// ChargeItemDefinitionApplicability represents a FHIR BackboneElement for ChargeItemDefinition.applicability.
type ChargeItemDefinitionApplicability struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Natural language description of the condition
	Description *string `json:"description,omitempty"`
	// Language of the expression
	Language *string `json:"language,omitempty"`
	// Boolean-valued expression
	Expression *string `json:"expression,omitempty"`
}

// ChargeItemDefinitionPropertyGroupApplicability represents a FHIR BackboneElement for ChargeItemDefinition.propertyGroup.applicability.
type ChargeItemDefinitionPropertyGroupApplicability struct {
}

// ChargeItemDefinitionPropertyGroupPriceComponent represents a FHIR BackboneElement for ChargeItemDefinition.propertyGroup.priceComponent.
type ChargeItemDefinitionPropertyGroupPriceComponent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// base | surcharge | deduction | discount | tax | informational
	Type string `json:"type"`
	// Code identifying the specific component
	Code *CodeableConcept `json:"code,omitempty"`
	// Factor used for calculating this component
	Factor *float64 `json:"factor,omitempty"`
	// Monetary amount associated with this component
	Amount *Money `json:"amount,omitempty"`
}

// ChargeItemDefinitionPropertyGroup represents a FHIR BackboneElement for ChargeItemDefinition.propertyGroup.
type ChargeItemDefinitionPropertyGroup struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Conditions under which the priceComponent is applicable
	Applicability []ChargeItemDefinitionPropertyGroupApplicability `json:"applicability,omitempty"`
	// Components of total line item price
	PriceComponent []ChargeItemDefinitionPropertyGroupPriceComponent `json:"priceComponent,omitempty"`
}

// ChargeItemDefinition represents a FHIR ChargeItemDefinition.
type ChargeItemDefinition struct {
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
	// Canonical identifier for this charge item definition, represented as a URI (globally unique)
	URL string `json:"url"`
	// Additional identifier for the charge item definition
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the charge item definition
	Version *string `json:"version,omitempty"`
	// Name for this charge item definition (human friendly)
	Title *string `json:"title,omitempty"`
	// Underlying externally-defined charge item definition
	DerivedFromUri []string `json:"derivedFromUri,omitempty"`
	// A larger definition of which this particular definition is a component or step
	PartOf []string `json:"partOf,omitempty"`
	// Completed or terminated request(s) whose function is taken by this new request
	Replaces []string `json:"replaces,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the charge item definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for charge item definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the charge item definition was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the charge item definition was last reviewed
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the charge item definition is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// Billing codes or product types this definition applies to
	Code *CodeableConcept `json:"code,omitempty"`
	// Instances this definition applies to
	Instance []Reference `json:"instance,omitempty"`
	// Whether or not the billing code is applicable
	Applicability []ChargeItemDefinitionApplicability `json:"applicability,omitempty"`
	// Group of properties which are applicable under the same conditions
	PropertyGroup []ChargeItemDefinitionPropertyGroup `json:"propertyGroup,omitempty"`
}
