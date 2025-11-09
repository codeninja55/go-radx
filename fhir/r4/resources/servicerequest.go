package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeServiceRequest is the FHIR resource type name for ServiceRequest.
const ResourceTypeServiceRequest = "ServiceRequest"

// ServiceRequest represents a FHIR ServiceRequest.
type ServiceRequest struct {
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
	// Identifiers assigned to this order
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// What request fulfills
	BasedOn []Reference `json:"basedOn,omitempty"`
	// What request replaces
	Replaces []Reference `json:"replaces,omitempty"`
	// Composite Request ID
	Requisition *Identifier `json:"requisition,omitempty"`
	// draft | active | on-hold | revoked | completed | entered-in-error | unknown
	Status string `json:"status"`
	// proposal | plan | directive | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// Classification of service
	Category []CodeableConcept `json:"category,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// True if service/procedure should not be performed
	DoNotPerform *bool `json:"doNotPerform,omitempty"`
	// What is being requested/ordered
	Code *CodeableConcept `json:"code,omitempty"`
	// Additional order information
	OrderDetail []CodeableConcept `json:"orderDetail,omitempty"`
	// Service amount
	Quantity *any `json:"quantity,omitempty"`
	// Individual or Entity the service is ordered for
	Subject Reference `json:"subject"`
	// Encounter in which the request was created
	Encounter *Reference `json:"encounter,omitempty"`
	// When service should occur
	Occurrence *any `json:"occurrence,omitempty"`
	// Preconditions for service
	AsNeeded *any `json:"asNeeded,omitempty"`
	// Date request signed
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Who/what is requesting service
	Requester *Reference `json:"requester,omitempty"`
	// Performer role
	PerformerType *CodeableConcept `json:"performerType,omitempty"`
	// Requested performer
	Performer []Reference `json:"performer,omitempty"`
	// Requested location
	LocationCode []CodeableConcept `json:"locationCode,omitempty"`
	// Requested location
	LocationReference []Reference `json:"locationReference,omitempty"`
	// Explanation/Justification for procedure or service
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Explanation/Justification for service or service
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Associated insurance coverage
	Insurance []Reference `json:"insurance,omitempty"`
	// Additional clinical information
	SupportingInfo []Reference `json:"supportingInfo,omitempty"`
	// Procedure Samples
	Specimen []Reference `json:"specimen,omitempty"`
	// Location on Body
	BodySite []CodeableConcept `json:"bodySite,omitempty"`
	// Comments
	Note []Annotation `json:"note,omitempty"`
	// Patient or consumer-oriented instructions
	PatientInstruction *string `json:"patientInstruction,omitempty"`
	// Request provenance
	RelevantHistory []Reference `json:"relevantHistory,omitempty"`
}
