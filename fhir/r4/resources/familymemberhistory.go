package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeFamilyMemberHistory is the FHIR resource type name for FamilyMemberHistory.
const ResourceTypeFamilyMemberHistory = "FamilyMemberHistory"

// FamilyMemberHistoryCondition represents a FHIR BackboneElement for FamilyMemberHistory.condition.
type FamilyMemberHistoryCondition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Condition suffered by relation
	Code CodeableConcept `json:"code"`
	// deceased | permanent disability | etc.
	Outcome *CodeableConcept `json:"outcome,omitempty"`
	// Whether the condition contributed to the cause of death
	ContributedToDeath *bool `json:"contributedToDeath,omitempty"`
	// When condition first manifested
	Onset *any `json:"onset,omitempty"`
	// Extra information about condition
	Note []Annotation `json:"note,omitempty"`
}

// FamilyMemberHistory represents a FHIR FamilyMemberHistory.
type FamilyMemberHistory struct {
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
	// External Id(s) for this record
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// partial | completed | entered-in-error | health-unknown
	Status string `json:"status"`
	// subject-unknown | withheld | unable-to-obtain | deferred
	DataAbsentReason *CodeableConcept `json:"dataAbsentReason,omitempty"`
	// Patient history is about
	Patient Reference `json:"patient"`
	// When history was recorded or last updated
	Date *primitives.DateTime `json:"date,omitempty"`
	// The family member described
	Name *string `json:"name,omitempty"`
	// Relationship to the subject
	Relationship CodeableConcept `json:"relationship"`
	// male | female | other | unknown
	Sex *CodeableConcept `json:"sex,omitempty"`
	// (approximate) date of birth
	Born *any `json:"born,omitempty"`
	// (approximate) age
	Age *any `json:"age,omitempty"`
	// Age is estimated?
	EstimatedAge *bool `json:"estimatedAge,omitempty"`
	// Dead? How old/when?
	Deceased *any `json:"deceased,omitempty"`
	// Why was family member history performed?
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Why was family member history performed?
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// General note about related person
	Note []Annotation `json:"note,omitempty"`
	// Condition that the related person had
	Condition []FamilyMemberHistoryCondition `json:"condition,omitempty"`
}
