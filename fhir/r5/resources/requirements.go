package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeRequirements is the FHIR resource type name for Requirements.
const ResourceTypeRequirements = "Requirements"

// RequirementsStatement represents a FHIR BackboneElement for Requirements.statement.
type RequirementsStatement struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Key that identifies this statement
	Key string `json:"key"`
	// Short Human label for this statement
	Label *string `json:"label,omitempty"`
	// SHALL | SHOULD | MAY | SHOULD-NOT
	Conformance []string `json:"conformance,omitempty"`
	// Set to true if requirements statement is conditional
	Conditionality *bool `json:"conditionality,omitempty"`
	// The actual requirement
	Requirement string `json:"requirement"`
	// Another statement this clarifies/restricts ([url#]key)
	DerivedFrom *string `json:"derivedFrom,omitempty"`
	// A larger requirement that this requirement helps to refine and enable
	Parent *string `json:"parent,omitempty"`
	// Design artifact that satisfies this requirement
	SatisfiedBy []string `json:"satisfiedBy,omitempty"`
	// External artifact (rule/document etc. that) created this requirement
	Reference []string `json:"reference,omitempty"`
	// Who asked for this statement
	Source []Reference `json:"source,omitempty"`
}

// Requirements represents a FHIR Requirements.
type Requirements struct {
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
	// Canonical identifier for this Requirements, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the Requirements (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the Requirements
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this Requirements (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this Requirements (human friendly)
	Title *string `json:"title,omitempty"`
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
	// Natural language description of the requirements
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for Requirements (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this Requirements is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// Other set of Requirements this builds on
	DerivedFrom []string `json:"derivedFrom,omitempty"`
	// External artifact (rule/document etc. that) created this set of requirements
	Reference []string `json:"reference,omitempty"`
	// Actor for these requirements
	Actor []string `json:"actor,omitempty"`
	// Actual statement as markdown
	Statement []RequirementsStatement `json:"statement,omitempty"`
}
