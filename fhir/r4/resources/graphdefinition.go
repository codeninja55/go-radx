package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeGraphDefinition is the FHIR resource type name for GraphDefinition.
const ResourceTypeGraphDefinition = "GraphDefinition"

// GraphDefinitionLinkTargetCompartment represents a FHIR BackboneElement for GraphDefinition.link.target.compartment.
type GraphDefinitionLinkTargetCompartment struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// condition | requirement
	Use string `json:"use"`
	// Patient | Encounter | RelatedPerson | Practitioner | Device
	Code string `json:"code"`
	// identical | matching | different | custom
	Rule string `json:"rule"`
	// Custom rule, as a FHIRPath expression
	Expression *string `json:"expression,omitempty"`
	// Documentation for FHIRPath expression
	Description *string `json:"description,omitempty"`
}

// GraphDefinitionLinkTargetLink represents a FHIR BackboneElement for GraphDefinition.link.target.link.
type GraphDefinitionLinkTargetLink struct {
}

// GraphDefinitionLinkTarget represents a FHIR BackboneElement for GraphDefinition.link.target.
type GraphDefinitionLinkTarget struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of resource this link refers to
	Type string `json:"type"`
	// Criteria for reverse lookup
	Params *string `json:"params,omitempty"`
	// Profile for the target resource
	Profile *string `json:"profile,omitempty"`
	// Compartment Consistency Rules
	Compartment []GraphDefinitionLinkTargetCompartment `json:"compartment,omitempty"`
	// Additional links from target resource
	Link []GraphDefinitionLinkTargetLink `json:"link,omitempty"`
}

// GraphDefinitionLink represents a FHIR BackboneElement for GraphDefinition.link.
type GraphDefinitionLink struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Path in the resource that contains the link
	Path *string `json:"path,omitempty"`
	// Which slice (if profiled)
	SliceName *string `json:"sliceName,omitempty"`
	// Minimum occurrences for this link
	Min *int `json:"min,omitempty"`
	// Maximum occurrences for this link
	Max *string `json:"max,omitempty"`
	// Why this link is specified
	Description *string `json:"description,omitempty"`
	// Potential target for the link
	Target []GraphDefinitionLinkTarget `json:"target,omitempty"`
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
	// Business version of the graph definition
	Version *string `json:"version,omitempty"`
	// Name for this graph definition (computer friendly)
	Name string `json:"name"`
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
	// Natural language description of the graph definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for graph definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this graph definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Type of resource at which the graph starts
	Start string `json:"start"`
	// Profile on base resource
	Profile *string `json:"profile,omitempty"`
	// Links this graph makes rules about
	Link []GraphDefinitionLink `json:"link,omitempty"`
}
