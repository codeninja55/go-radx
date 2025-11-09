package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCondition is the FHIR resource type name for Condition.
const ResourceTypeCondition = "Condition"

// ConditionParticipant represents a FHIR BackboneElement for Condition.participant.
type ConditionParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of involvement
	Function *CodeableConcept `json:"function,omitempty"`
	// Who or what participated in the activities related to the condition
	Actor Reference `json:"actor"`
}

// ConditionStage represents a FHIR BackboneElement for Condition.stage.
type ConditionStage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Simple summary (disease specific)
	Summary *CodeableConcept `json:"summary,omitempty"`
	// Formal record of assessment
	Assessment []Reference `json:"assessment,omitempty"`
	// Kind of staging
	Type *CodeableConcept `json:"type,omitempty"`
}

// Condition represents a FHIR Condition.
type Condition struct {
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
	// External Ids for this condition
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | recurrence | relapse | inactive | remission | resolved | unknown
	ClinicalStatus CodeableConcept `json:"clinicalStatus"`
	// unconfirmed | provisional | differential | confirmed | refuted | entered-in-error
	VerificationStatus *CodeableConcept `json:"verificationStatus,omitempty"`
	// problem-list-item | encounter-diagnosis
	Category []CodeableConcept `json:"category,omitempty"`
	// Subjective severity of condition
	Severity *CodeableConcept `json:"severity,omitempty"`
	// Identification of the condition, problem or diagnosis
	Code *CodeableConcept `json:"code,omitempty"`
	// Anatomical location, if relevant
	BodySite []CodeableConcept `json:"bodySite,omitempty"`
	// Who has the condition?
	Subject Reference `json:"subject"`
	// The Encounter during which this Condition was created
	Encounter *Reference `json:"encounter,omitempty"`
	// Estimated or actual date,  date-time, or age
	Onset *any `json:"onset,omitempty"`
	// When in resolution/remission
	Abatement *any `json:"abatement,omitempty"`
	// Date condition was first recorded
	RecordedDate *primitives.DateTime `json:"recordedDate,omitempty"`
	// Who or what participated in the activities related to the condition and how they were involved
	Participant []ConditionParticipant `json:"participant,omitempty"`
	// Stage/grade, usually assessed formally
	Stage []ConditionStage `json:"stage,omitempty"`
	// Supporting evidence for the verification status
	Evidence []CodeableReference `json:"evidence,omitempty"`
	// Additional information about the Condition
	Note []Annotation `json:"note,omitempty"`
}
