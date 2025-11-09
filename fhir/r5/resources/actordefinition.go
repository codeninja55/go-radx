package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeActorDefinition is the FHIR resource type name for ActorDefinition.
const ResourceTypeActorDefinition = "ActorDefinition"

// ActorDefinition represents a FHIR ActorDefinition.
type ActorDefinition struct {
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
	// Canonical identifier for this actor definition, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the actor definition (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the actor definition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this actor definition (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this actor definition (human friendly)
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
	// Natural language description of the actor
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for actor definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this actor definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// person | system
	Type string `json:"type"`
	// Functionality associated with the actor
	Documentation *string `json:"documentation,omitempty"`
	// Reference to more information about the actor
	Reference []string `json:"reference,omitempty"`
	// CapabilityStatement for the actor (if applicable)
	Capabilities *string `json:"capabilities,omitempty"`
	// Definition of this actor in another context / IG
	DerivedFrom []string `json:"derivedFrom,omitempty"`
}
