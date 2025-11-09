package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCondition is the FHIR resource type name for Condition.
const ResourceTypeCondition = "Condition"

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

// ConditionEvidence represents a FHIR BackboneElement for Condition.evidence.
type ConditionEvidence struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Manifestation/symptom
	Code []CodeableConcept `json:"code,omitempty"`
	// Supporting information found elsewhere
	Detail []Reference `json:"detail,omitempty"`
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
	// active | recurrence | relapse | inactive | remission | resolved
	ClinicalStatus *CodeableConcept `json:"clinicalStatus,omitempty"`
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
	// Encounter created as part of
	Encounter *Reference `json:"encounter,omitempty"`
	// Estimated or actual date,  date-time, or age
	Onset *any `json:"onset,omitempty"`
	// When in resolution/remission
	Abatement *any `json:"abatement,omitempty"`
	// Date record was first recorded
	RecordedDate *primitives.DateTime `json:"recordedDate,omitempty"`
	// Who recorded the condition
	Recorder *Reference `json:"recorder,omitempty"`
	// Person who asserts this condition
	Asserter *Reference `json:"asserter,omitempty"`
	// Stage/grade, usually assessed formally
	Stage []ConditionStage `json:"stage,omitempty"`
	// Supporting evidence
	Evidence []ConditionEvidence `json:"evidence,omitempty"`
	// Additional information about the Condition
	Note []Annotation `json:"note,omitempty"`
}
