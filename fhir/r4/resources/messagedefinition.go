package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMessageDefinition is the FHIR resource type name for MessageDefinition.
const ResourceTypeMessageDefinition = "MessageDefinition"

// MessageDefinitionFocus represents a FHIR BackboneElement for MessageDefinition.focus.
type MessageDefinitionFocus struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of resource
	Code string `json:"code"`
	// Profile that must be adhered to by focus
	Profile *string `json:"profile,omitempty"`
	// Minimum number of focuses of this type
	Min uint `json:"min"`
	// Maximum number of focuses of this type
	Max *string `json:"max,omitempty"`
}

// MessageDefinitionAllowedResponse represents a FHIR BackboneElement for MessageDefinition.allowedResponse.
type MessageDefinitionAllowedResponse struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to allowed message definition response
	Message string `json:"message"`
	// When should this response be used
	Situation *string `json:"situation,omitempty"`
}

// MessageDefinition represents a FHIR MessageDefinition.
type MessageDefinition struct {
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
	// Business Identifier for a given MessageDefinition
	URL *string `json:"url,omitempty"`
	// Primary key for the message definition on a given server
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the message definition
	Version *string `json:"version,omitempty"`
	// Name for this message definition (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this message definition (human friendly)
	Title *string `json:"title,omitempty"`
	// Takes the place of
	Replaces []string `json:"replaces,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date primitives.DateTime `json:"date"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the message definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for message definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this message definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Definition this one is based on
	Base *string `json:"base,omitempty"`
	// Protocol/workflow this is part of
	Parent []string `json:"parent,omitempty"`
	// Event code  or link to the EventDefinition
	Event any `json:"event"`
	// consequence | currency | notification
	Category *string `json:"category,omitempty"`
	// Resource(s) that are the subject of the event
	Focus []MessageDefinitionFocus `json:"focus,omitempty"`
	// always | on-error | never | on-success
	ResponseRequired *string `json:"responseRequired,omitempty"`
	// Responses to this message
	AllowedResponse []MessageDefinitionAllowedResponse `json:"allowedResponse,omitempty"`
	// Canonical reference to a GraphDefinition
	Graph []string `json:"graph,omitempty"`
}
