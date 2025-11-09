package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCommunicationRequest is the FHIR resource type name for CommunicationRequest.
const ResourceTypeCommunicationRequest = "CommunicationRequest"

// CommunicationRequestPayload represents a FHIR BackboneElement for CommunicationRequest.payload.
type CommunicationRequestPayload struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Message part content
	Content any `json:"content"`
}

// CommunicationRequest represents a FHIR CommunicationRequest.
type CommunicationRequest struct {
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
	// Fulfills plan or proposal
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Request(s) replaced by this request
	Replaces []Reference `json:"replaces,omitempty"`
	// Composite request this is part of
	GroupIdentifier *Identifier `json:"groupIdentifier,omitempty"`
	// draft | active | on-hold | revoked | completed | entered-in-error | unknown
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// proposal | plan | directive | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// Message category
	Category []CodeableConcept `json:"category,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// True if request is prohibiting action
	DoNotPerform *bool `json:"doNotPerform,omitempty"`
	// A channel of communication
	Medium []CodeableConcept `json:"medium,omitempty"`
	// Focus of message
	Subject *Reference `json:"subject,omitempty"`
	// Resources that pertain to this communication request
	About []Reference `json:"about,omitempty"`
	// The Encounter during which this CommunicationRequest was created
	Encounter *Reference `json:"encounter,omitempty"`
	// Message payload
	Payload []CommunicationRequestPayload `json:"payload,omitempty"`
	// When scheduled
	Occurrence *any `json:"occurrence,omitempty"`
	// When request transitioned to being actionable
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Who asks for the information to be shared
	Requester *Reference `json:"requester,omitempty"`
	// Who to share the information with
	Recipient []Reference `json:"recipient,omitempty"`
	// Who should share the information
	InformationProvider []Reference `json:"informationProvider,omitempty"`
	// Why is communication needed?
	Reason []CodeableReference `json:"reason,omitempty"`
	// Comments made about communication request
	Note []Annotation `json:"note,omitempty"`
}
