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
	// Boolean-valued expression
	Condition *Expression `json:"condition,omitempty"`
	// When the charge item definition is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// Reference to / quotation of the external source of the group of properties
	RelatedArtifact *RelatedArtifact `json:"relatedArtifact,omitempty"`
}

// ChargeItemDefinitionPropertyGroupApplicability represents a FHIR BackboneElement for ChargeItemDefinition.propertyGroup.applicability.
type ChargeItemDefinitionPropertyGroupApplicability struct {
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
	PriceComponent []MonetaryComponent `json:"priceComponent,omitempty"`
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
	URL *string `json:"url,omitempty"`
	// Additional identifier for the charge item definition
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the charge item definition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this charge item definition (computer friendly)
	Name *string `json:"name,omitempty"`
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
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the charge item definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for charge item definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this charge item definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the charge item definition was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the charge item definition was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// Billing code or product type this definition applies to
	Code *CodeableConcept `json:"code,omitempty"`
	// Instances this definition applies to
	Instance []Reference `json:"instance,omitempty"`
	// Whether or not the billing code is applicable
	Applicability []ChargeItemDefinitionApplicability `json:"applicability,omitempty"`
	// Group of properties which are applicable under the same conditions
	PropertyGroup []ChargeItemDefinitionPropertyGroup `json:"propertyGroup,omitempty"`
}
