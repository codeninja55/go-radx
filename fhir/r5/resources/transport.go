package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeTransport is the FHIR resource type name for Transport.
const ResourceTypeTransport = "Transport"

// TransportRestriction represents a FHIR BackboneElement for Transport.restriction.
type TransportRestriction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// How many times to repeat
	Repetitions *int `json:"repetitions,omitempty"`
	// When fulfillment sought
	Period *Period `json:"period,omitempty"`
	// For whom is fulfillment sought?
	Recipient []Reference `json:"recipient,omitempty"`
}

// TransportInput represents a FHIR BackboneElement for Transport.input.
type TransportInput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for the input
	Type CodeableConcept `json:"type"`
	// Content to use in performing the transport
	Value any `json:"value"`
}

// TransportOutput represents a FHIR BackboneElement for Transport.output.
type TransportOutput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for output
	Type CodeableConcept `json:"type"`
	// Result of output
	Value any `json:"value"`
}

// Transport represents a FHIR Transport.
type Transport struct {
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
	// External identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Formal definition of transport
	InstantiatesCanonical *string `json:"instantiatesCanonical,omitempty"`
	// Formal definition of transport
	InstantiatesUri *string `json:"instantiatesUri,omitempty"`
	// Request fulfilled by this transport
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Requisition or grouper id
	GroupIdentifier *Identifier `json:"groupIdentifier,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// in-progress | completed | abandoned | cancelled | planned | entered-in-error
	Status *string `json:"status,omitempty"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// unknown | proposal | plan | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// Transport Type
	Code *CodeableConcept `json:"code,omitempty"`
	// Human-readable explanation of transport
	Description *string `json:"description,omitempty"`
	// What transport is acting on
	Focus *Reference `json:"focus,omitempty"`
	// Beneficiary of the Transport
	For *Reference `json:"for,omitempty"`
	// Healthcare event during which this transport originated
	Encounter *Reference `json:"encounter,omitempty"`
	// Completion time of the event (the occurrence)
	CompletionTime *primitives.DateTime `json:"completionTime,omitempty"`
	// Transport Creation Date
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Transport Last Modified Date
	LastModified *primitives.DateTime `json:"lastModified,omitempty"`
	// Who is asking for transport to be done
	Requester *Reference `json:"requester,omitempty"`
	// Requested performer
	PerformerType []CodeableConcept `json:"performerType,omitempty"`
	// Responsible individual
	Owner *Reference `json:"owner,omitempty"`
	// Where transport occurs
	Location *Reference `json:"location,omitempty"`
	// Associated insurance coverage
	Insurance []Reference `json:"insurance,omitempty"`
	// Comments made about the transport
	Note []Annotation `json:"note,omitempty"`
	// Key events in history of the Transport
	RelevantHistory []Reference `json:"relevantHistory,omitempty"`
	// Constraints on fulfillment transports
	Restriction *TransportRestriction `json:"restriction,omitempty"`
	// Information used to perform transport
	Input []TransportInput `json:"input,omitempty"`
	// Information produced as part of transport
	Output []TransportOutput `json:"output,omitempty"`
	// The desired location
	RequestedLocation Reference `json:"requestedLocation"`
	// The entity current location
	CurrentLocation Reference `json:"currentLocation"`
	// Why transport is needed
	Reason *CodeableReference `json:"reason,omitempty"`
	// Parent (or preceding) transport
	History *Reference `json:"history,omitempty"`
}
