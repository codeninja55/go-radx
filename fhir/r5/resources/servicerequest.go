package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeServiceRequest is the FHIR resource type name for ServiceRequest.
const ResourceTypeServiceRequest = "ServiceRequest"

// ServiceRequestOrderDetailParameter represents a FHIR BackboneElement for ServiceRequest.orderDetail.parameter.
type ServiceRequestOrderDetailParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The detail of the order being requested
	Code CodeableConcept `json:"code"`
	// The value for the order detail
	Value any `json:"value"`
}

// ServiceRequestOrderDetail represents a FHIR BackboneElement for ServiceRequest.orderDetail.
type ServiceRequestOrderDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The context of the order details by reference
	ParameterFocus *CodeableReference `json:"parameterFocus,omitempty"`
	// The parameter details for the service being requested
	Parameter []ServiceRequestOrderDetailParameter `json:"parameter,omitempty"`
}

// ServiceRequestPatientInstruction represents a FHIR BackboneElement for ServiceRequest.patientInstruction.
type ServiceRequestPatientInstruction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Patient or consumer-oriented instructions
	Instruction *any `json:"instruction,omitempty"`
}

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
	// proposal | plan | directive | order +
	Intent string `json:"intent"`
	// Classification of service
	Category []CodeableConcept `json:"category,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// True if service/procedure should not be performed
	DoNotPerform *bool `json:"doNotPerform,omitempty"`
	// What is being requested/ordered
	Code *CodeableReference `json:"code,omitempty"`
	// Additional order information
	OrderDetail []ServiceRequestOrderDetail `json:"orderDetail,omitempty"`
	// Service amount
	Quantity *any `json:"quantity,omitempty"`
	// Individual or Entity the service is ordered for
	Subject Reference `json:"subject"`
	// What the service request is about, when it is not about the subject of record
	Focus []Reference `json:"focus,omitempty"`
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
	Location []CodeableReference `json:"location,omitempty"`
	// Explanation/Justification for procedure or service
	Reason []CodeableReference `json:"reason,omitempty"`
	// Associated insurance coverage
	Insurance []Reference `json:"insurance,omitempty"`
	// Additional clinical information
	SupportingInfo []CodeableReference `json:"supportingInfo,omitempty"`
	// Procedure Samples
	Specimen []Reference `json:"specimen,omitempty"`
	// Coded location on Body
	BodySite []CodeableConcept `json:"bodySite,omitempty"`
	// BodyStructure-based location on the body
	BodyStructure *Reference `json:"bodyStructure,omitempty"`
	// Comments
	Note []Annotation `json:"note,omitempty"`
	// Patient or consumer-oriented instructions
	PatientInstruction []ServiceRequestPatientInstruction `json:"patientInstruction,omitempty"`
	// Request provenance
	RelevantHistory []Reference `json:"relevantHistory,omitempty"`
}
