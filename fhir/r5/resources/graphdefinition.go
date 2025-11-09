package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeGraphDefinition is the FHIR resource type name for GraphDefinition.
const ResourceTypeGraphDefinition = "GraphDefinition"

// GraphDefinitionNode represents a FHIR BackboneElement for GraphDefinition.node.
type GraphDefinitionNode struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Internal ID - target for link references
	NodeId string `json:"nodeId"`
	// Why this node is specified
	Description *string `json:"description,omitempty"`
	// Type of resource this link refers to
	Type string `json:"type"`
	// Profile for the target resource
	Profile *string `json:"profile,omitempty"`
}

// GraphDefinitionLinkCompartment represents a FHIR BackboneElement for GraphDefinition.link.compartment.
type GraphDefinitionLinkCompartment struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// where | requires
	Use string `json:"use"`
	// identical | matching | different | custom
	Rule string `json:"rule"`
	// Patient | Encounter | RelatedPerson | Practitioner | Device | EpisodeOfCare
	Code string `json:"code"`
	// Custom rule, as a FHIRPath expression
	Expression *string `json:"expression,omitempty"`
	// Documentation for FHIRPath expression
	Description *string `json:"description,omitempty"`
}

// GraphDefinitionLink represents a FHIR BackboneElement for GraphDefinition.link.
type GraphDefinitionLink struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Why this link is specified
	Description *string `json:"description,omitempty"`
	// Minimum occurrences for this link
	Min *int `json:"min,omitempty"`
	// Maximum occurrences for this link
	Max *string `json:"max,omitempty"`
	// Source Node for this link
	SourceId string `json:"sourceId"`
	// Path in the resource that contains the link
	Path *string `json:"path,omitempty"`
	// Which slice (if profiled)
	SliceName *string `json:"sliceName,omitempty"`
	// Target Node for this link
	TargetId string `json:"targetId"`
	// Criteria for reverse lookup
	Params *string `json:"params,omitempty"`
	// Compartment Consistency Rules
	Compartment []GraphDefinitionLinkCompartment `json:"compartment,omitempty"`
}

// GraphDefinition represents a FHIR GraphDefinition.
type GraphDefinition struct {
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
	// Canonical identifier for this graph definition, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the GraphDefinition (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the graph definition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this graph definition (computer friendly)
	Name string `json:"name"`
	// Name for this graph definition (human friendly)
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
	// Natural language description of the graph definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for graph definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this graph definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// Starting Node
	Start *string `json:"start,omitempty"`
	// Potential target for the link
	Node []GraphDefinitionNode `json:"node,omitempty"`
	// Links this graph makes rules about
	Link []GraphDefinitionLink `json:"link,omitempty"`
}
