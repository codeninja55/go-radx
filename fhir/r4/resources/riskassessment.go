package resources

// ResourceTypeRiskAssessment is the FHIR resource type name for RiskAssessment.
const ResourceTypeRiskAssessment = "RiskAssessment"

// RiskAssessmentPrediction represents a FHIR BackboneElement for RiskAssessment.prediction.
type RiskAssessmentPrediction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Possible outcome for the subject
	Outcome *CodeableConcept `json:"outcome,omitempty"`
	// Likelihood of specified outcome
	Probability *any `json:"probability,omitempty"`
	// Likelihood of specified outcome as a qualitative value
	QualitativeRisk *CodeableConcept `json:"qualitativeRisk,omitempty"`
	// Relative likelihood
	RelativeRisk *float64 `json:"relativeRisk,omitempty"`
	// Timeframe or age range
	When *any `json:"when,omitempty"`
	// Explanation of prediction
	Rationale *string `json:"rationale,omitempty"`
}

// RiskAssessment represents a FHIR RiskAssessment.
type RiskAssessment struct {
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
	// Unique identifier for the assessment
	Identifier []Identifier `json:"identifier,omitempty"`
	// Request fulfilled by this assessment
	BasedOn *Reference `json:"basedOn,omitempty"`
	// Part of this occurrence
	Parent *Reference `json:"parent,omitempty"`
	// registered | preliminary | final | amended +
	Status string `json:"status"`
	// Evaluation mechanism
	Method *CodeableConcept `json:"method,omitempty"`
	// Type of assessment
	Code *CodeableConcept `json:"code,omitempty"`
	// Who/what does assessment apply to?
	Subject Reference `json:"subject"`
	// Where was assessment performed?
	Encounter *Reference `json:"encounter,omitempty"`
	// When was assessment made?
	Occurrence *any `json:"occurrence,omitempty"`
	// Condition assessed
	Condition *Reference `json:"condition,omitempty"`
	// Who did assessment?
	Performer *Reference `json:"performer,omitempty"`
	// Why the assessment was necessary?
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Why the assessment was necessary?
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Information used in assessment
	Basis []Reference `json:"basis,omitempty"`
	// Outcome predicted
	Prediction []RiskAssessmentPrediction `json:"prediction,omitempty"`
	// How to reduce risk
	Mitigation *string `json:"mitigation,omitempty"`
	// Comments on the risk assessment
	Note []Annotation `json:"note,omitempty"`
}
