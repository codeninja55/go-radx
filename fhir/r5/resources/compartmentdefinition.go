package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCompartmentDefinition is the FHIR resource type name for CompartmentDefinition.
const ResourceTypeCompartmentDefinition = "CompartmentDefinition"

// CompartmentDefinitionResource represents a FHIR BackboneElement for CompartmentDefinition.resource.
type CompartmentDefinitionResource struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of resource type
	Code string `json:"code"`
	// Search Parameter Name, or chained parameters
	Param []string `json:"param,omitempty"`
	// Additional documentation about the resource and compartment
	Documentation *string `json:"documentation,omitempty"`
	// Search Param for interpreting $everything.start
	StartParam *string `json:"startParam,omitempty"`
	// Search Param for interpreting $everything.end
	EndParam *string `json:"endParam,omitempty"`
}

// CompartmentDefinition represents a FHIR CompartmentDefinition.
type CompartmentDefinition struct {
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
	// Canonical identifier for this compartment definition, represented as a URI (globally unique)
	URL string `json:"url"`
	// Business version of the compartment definition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this compartment definition (computer friendly)
	Name string `json:"name"`
	// Name for this compartment definition (human friendly)
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
	// Natural language description of the compartment definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Why this compartment definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Patient | Encounter | RelatedPerson | Practitioner | Device | EpisodeOfCare
	Code string `json:"code"`
	// Whether the search syntax is supported
	Search bool `json:"search"`
	// How a resource is related to the compartment
	Resource []CompartmentDefinitionResource `json:"resource,omitempty"`
}
