package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDeviceRequest is the FHIR resource type name for DeviceRequest.
const ResourceTypeDeviceRequest = "DeviceRequest"

// DeviceRequestParameter represents a FHIR BackboneElement for DeviceRequest.parameter.
type DeviceRequestParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Device detail
	Code *CodeableConcept `json:"code,omitempty"`
	// Value of detail
	Value *any `json:"value,omitempty"`
}

// DeviceRequest represents a FHIR DeviceRequest.
type DeviceRequest struct {
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
	// External Request identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// What request fulfills
	BasedOn []Reference `json:"basedOn,omitempty"`
	// What request replaces
	PriorRequest []Reference `json:"priorRequest,omitempty"`
	// Identifier of composite request
	GroupIdentifier *Identifier `json:"groupIdentifier,omitempty"`
	// draft | active | on-hold | revoked | completed | entered-in-error | unknown
	Status *string `json:"status,omitempty"`
	// proposal | plan | directive | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// Device requested
	Code any `json:"code"`
	// Device details
	Parameter []DeviceRequestParameter `json:"parameter,omitempty"`
	// Focus of request
	Subject Reference `json:"subject"`
	// Encounter motivating request
	Encounter *Reference `json:"encounter,omitempty"`
	// Desired time or schedule for use
	Occurrence *any `json:"occurrence,omitempty"`
	// When recorded
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Who/what is requesting diagnostics
	Requester *Reference `json:"requester,omitempty"`
	// Filler role
	PerformerType *CodeableConcept `json:"performerType,omitempty"`
	// Requested Filler
	Performer *Reference `json:"performer,omitempty"`
	// Coded Reason for request
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Linked Reason for request
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Associated insurance coverage
	Insurance []Reference `json:"insurance,omitempty"`
	// Additional clinical information
	SupportingInfo []Reference `json:"supportingInfo,omitempty"`
	// Notes or comments
	Note []Annotation `json:"note,omitempty"`
	// Request provenance
	RelevantHistory []Reference `json:"relevantHistory,omitempty"`
}
