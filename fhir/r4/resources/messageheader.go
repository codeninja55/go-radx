package resources

// ResourceTypeMessageHeader is the FHIR resource type name for MessageHeader.
const ResourceTypeMessageHeader = "MessageHeader"

// MessageHeaderDestination represents a FHIR BackboneElement for MessageHeader.destination.
type MessageHeaderDestination struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of system
	Name *string `json:"name,omitempty"`
	// Particular delivery destination within the destination
	Target *Reference `json:"target,omitempty"`
	// Actual destination address or id
	Endpoint string `json:"endpoint"`
	// Intended "real-world" recipient for the data
	Receiver *Reference `json:"receiver,omitempty"`
}

// MessageHeaderSource represents a FHIR BackboneElement for MessageHeader.source.
type MessageHeaderSource struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of system
	Name *string `json:"name,omitempty"`
	// Name of software running the system
	Software *string `json:"software,omitempty"`
	// Version of software running
	Version *string `json:"version,omitempty"`
	// Human contact for problems
	Contact *ContactPoint `json:"contact,omitempty"`
	// Actual message source address or id
	Endpoint string `json:"endpoint"`
}

// MessageHeaderResponse represents a FHIR BackboneElement for MessageHeader.response.
type MessageHeaderResponse struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Id of original message
	Identifier string `json:"identifier"`
	// ok | transient-error | fatal-error
	Code string `json:"code"`
	// Specific list of hints/warnings/errors
	Details *Reference `json:"details,omitempty"`
}

// MessageHeader represents a FHIR MessageHeader.
type MessageHeader struct {
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
	// Code for the event this message represents or link to event definition
	Event any `json:"event"`
	// Message destination application(s)
	Destination []MessageHeaderDestination `json:"destination,omitempty"`
	// Real world sender of the message
	Sender *Reference `json:"sender,omitempty"`
	// The source of the data entry
	Enterer *Reference `json:"enterer,omitempty"`
	// The source of the decision
	Author *Reference `json:"author,omitempty"`
	// Message source application
	Source MessageHeaderSource `json:"source"`
	// Final responsibility for event
	Responsible *Reference `json:"responsible,omitempty"`
	// Cause of event
	Reason *CodeableConcept `json:"reason,omitempty"`
	// If this is a reply to prior message
	Response *MessageHeaderResponse `json:"response,omitempty"`
	// The actual content of the message
	Focus []Reference `json:"focus,omitempty"`
	// Link to the definition for this message
	Definition *string `json:"definition,omitempty"`
}
