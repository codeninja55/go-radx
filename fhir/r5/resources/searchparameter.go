package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSearchParameter is the FHIR resource type name for SearchParameter.
const ResourceTypeSearchParameter = "SearchParameter"

// SearchParameterComponent represents a FHIR BackboneElement for SearchParameter.component.
type SearchParameterComponent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Defines how the part works
	Definition string `json:"definition"`
	// Subexpression relative to main expression
	Expression string `json:"expression"`
}

// SearchParameter represents a FHIR SearchParameter.
type SearchParameter struct {
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
	// Canonical identifier for this search parameter, represented as a URI (globally unique)
	URL string `json:"url"`
	// Additional identifier for the search parameter (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the search parameter
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this search parameter (computer friendly)
	Name string `json:"name"`
	// Name for this search parameter (human friendly)
	Title *string `json:"title,omitempty"`
	// Original definition for the search parameter
	DerivedFrom *string `json:"derivedFrom,omitempty"`
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
	// Natural language description of the search parameter
	Description string `json:"description"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for search parameter (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this search parameter is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// Recommended name for parameter in search url
	Code string `json:"code"`
	// The resource type(s) this search parameter applies to
	Base []string `json:"base,omitempty"`
	// number | date | string | token | reference | composite | quantity | uri | special
	Type string `json:"type"`
	// FHIRPath expression that extracts the values
	Expression *string `json:"expression,omitempty"`
	// normal | phonetic | other
	ProcessingMode *string `json:"processingMode,omitempty"`
	// FHIRPath expression that constraints the usage of this SearchParamete
	Constraint *string `json:"constraint,omitempty"`
	// Types of resource (if a resource reference)
	Target []string `json:"target,omitempty"`
	// Allow multiple values per parameter (or)
	MultipleOr *bool `json:"multipleOr,omitempty"`
	// Allow multiple parameters (and)
	MultipleAnd *bool `json:"multipleAnd,omitempty"`
	// eq | ne | gt | lt | ge | le | sa | eb | ap
	Comparator []string `json:"comparator,omitempty"`
	// missing | exact | contains | not | text | in | not-in | below | above | type | identifier | of-type | code-text | text-advanced | iterate
	Modifier []string `json:"modifier,omitempty"`
	// Chained names supported
	Chain []string `json:"chain,omitempty"`
	// For Composite resources to define the parts
	Component []SearchParameterComponent `json:"component,omitempty"`
}
