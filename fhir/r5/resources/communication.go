package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCommunication is the FHIR resource type name for Communication.
const ResourceTypeCommunication = "Communication"

// CommunicationPayload represents a FHIR BackboneElement for Communication.payload.
type CommunicationPayload struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Message part content
	Content any `json:"content"`
}

// Communication represents a FHIR Communication.
type Communication struct {
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
	// Unique identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// Request fulfilled by this communication
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Part of referenced event (e.g. Communication, Procedure)
	PartOf []Reference `json:"partOf,omitempty"`
	// Reply to
	InResponseTo []Reference `json:"inResponseTo,omitempty"`
	// preparation | in-progress | not-done | on-hold | stopped | completed | entered-in-error | unknown
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// Message category
	Category []CodeableConcept `json:"category,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// A channel of communication
	Medium []CodeableConcept `json:"medium,omitempty"`
	// Focus of message
	Subject *Reference `json:"subject,omitempty"`
	// Description of the purpose/content
	Topic *CodeableConcept `json:"topic,omitempty"`
	// Resources that pertain to this communication
	About []Reference `json:"about,omitempty"`
	// The Encounter during which this Communication was created
	Encounter *Reference `json:"encounter,omitempty"`
	// When sent
	Sent *primitives.DateTime `json:"sent,omitempty"`
	// When received
	Received *primitives.DateTime `json:"received,omitempty"`
	// Who the information is shared with
	Recipient []Reference `json:"recipient,omitempty"`
	// Who shares the information
	Sender *Reference `json:"sender,omitempty"`
	// Indication for message
	Reason []CodeableReference `json:"reason,omitempty"`
	// Message payload
	Payload []CommunicationPayload `json:"payload,omitempty"`
	// Comments made about the communication
	Note []Annotation `json:"note,omitempty"`
}
