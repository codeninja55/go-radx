package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCarePlan is the FHIR resource type name for CarePlan.
const ResourceTypeCarePlan = "CarePlan"

// CarePlanActivity represents a FHIR BackboneElement for CarePlan.activity.
type CarePlanActivity struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Results of the activity (concept, or Appointment, Encounter, Procedure, etc.)
	PerformedActivity []CodeableReference `json:"performedActivity,omitempty"`
	// Comments about the activity status/progress
	Progress []Annotation `json:"progress,omitempty"`
	// Activity that is intended to be part of the care plan
	PlannedActivityReference *Reference `json:"plannedActivityReference,omitempty"`
}

// CarePlan represents a FHIR CarePlan.
type CarePlan struct {
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
	// External Ids for this plan
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// Fulfills plan, proposal or order
	BasedOn []Reference `json:"basedOn,omitempty"`
	// CarePlan replaced by this CarePlan
	Replaces []Reference `json:"replaces,omitempty"`
	// Part of referenced CarePlan
	PartOf []Reference `json:"partOf,omitempty"`
	// draft | active | on-hold | revoked | completed | entered-in-error | unknown
	Status string `json:"status"`
	// proposal | plan | order | option | directive
	Intent string `json:"intent"`
	// Type of plan
	Category []CodeableConcept `json:"category,omitempty"`
	// Human-friendly name for the care plan
	Title *string `json:"title,omitempty"`
	// Summary of nature of plan
	Description *string `json:"description,omitempty"`
	// Who the care plan is for
	Subject Reference `json:"subject"`
	// The Encounter during which this CarePlan was created
	Encounter *Reference `json:"encounter,omitempty"`
	// Time period plan covers
	Period *Period `json:"period,omitempty"`
	// Date record was first recorded
	Created *primitives.DateTime `json:"created,omitempty"`
	// Who is the designated responsible party
	Custodian *Reference `json:"custodian,omitempty"`
	// Who provided the content of the care plan
	Contributor []Reference `json:"contributor,omitempty"`
	// Who's involved in plan?
	CareTeam []Reference `json:"careTeam,omitempty"`
	// Health issues this plan addresses
	Addresses []CodeableReference `json:"addresses,omitempty"`
	// Information considered as part of plan
	SupportingInfo []Reference `json:"supportingInfo,omitempty"`
	// Desired outcome of plan
	Goal []Reference `json:"goal,omitempty"`
	// Action to occur or has occurred as part of plan
	Activity []CarePlanActivity `json:"activity,omitempty"`
	// Comments about the plan
	Note []Annotation `json:"note,omitempty"`
}
