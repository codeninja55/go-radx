package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeConditionDefinition is the FHIR resource type name for ConditionDefinition.
const ResourceTypeConditionDefinition = "ConditionDefinition"

// ConditionDefinitionObservation represents a FHIR BackboneElement for ConditionDefinition.observation.
type ConditionDefinitionObservation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Category that is relevant
	Category *CodeableConcept `json:"category,omitempty"`
	// Code for relevant Observation
	Code *CodeableConcept `json:"code,omitempty"`
}

// ConditionDefinitionMedication represents a FHIR BackboneElement for ConditionDefinition.medication.
type ConditionDefinitionMedication struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Category that is relevant
	Category *CodeableConcept `json:"category,omitempty"`
	// Code for relevant Medication
	Code *CodeableConcept `json:"code,omitempty"`
}

// ConditionDefinitionPrecondition represents a FHIR BackboneElement for ConditionDefinition.precondition.
type ConditionDefinitionPrecondition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// sensitive | specific
	Type string `json:"type"`
	// Code for relevant Observation
	Code CodeableConcept `json:"code"`
	// Value of Observation
	Value *any `json:"value,omitempty"`
}

// ConditionDefinitionQuestionnaire represents a FHIR BackboneElement for ConditionDefinition.questionnaire.
type ConditionDefinitionQuestionnaire struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// preadmit | diff-diagnosis | outcome
	Purpose string `json:"purpose"`
	// Specific Questionnaire
	Reference Reference `json:"reference"`
}

// ConditionDefinitionPlan represents a FHIR BackboneElement for ConditionDefinition.plan.
type ConditionDefinitionPlan struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Use for the plan
	Role *CodeableConcept `json:"role,omitempty"`
	// The actual plan
	Reference Reference `json:"reference"`
}

// ConditionDefinition represents a FHIR ConditionDefinition.
type ConditionDefinition struct {
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
	// Canonical identifier for this condition definition, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the condition definition
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the condition definition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this condition definition (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this condition definition (human friendly)
	Title *string `json:"title,omitempty"`
	// Subordinate title of the event definition
	Subtitle *string `json:"subtitle,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the condition definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for condition definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Identification of the condition, problem or diagnosis
	Code CodeableConcept `json:"code"`
	// Subjective severity of condition
	Severity *CodeableConcept `json:"severity,omitempty"`
	// Anatomical location, if relevant
	BodySite *CodeableConcept `json:"bodySite,omitempty"`
	// Stage/grade, usually assessed formally
	Stage *CodeableConcept `json:"stage,omitempty"`
	// Whether Severity is appropriate
	HasSeverity *bool `json:"hasSeverity,omitempty"`
	// Whether bodySite is appropriate
	HasBodySite *bool `json:"hasBodySite,omitempty"`
	// Whether stage is appropriate
	HasStage *bool `json:"hasStage,omitempty"`
	// Formal Definition for the condition
	Definition []string `json:"definition,omitempty"`
	// Observations particularly relevant to this condition
	Observation []ConditionDefinitionObservation `json:"observation,omitempty"`
	// Medications particularly relevant for this condition
	Medication []ConditionDefinitionMedication `json:"medication,omitempty"`
	// Observation that suggets this condition
	Precondition []ConditionDefinitionPrecondition `json:"precondition,omitempty"`
	// Appropriate team for this condition
	Team []Reference `json:"team,omitempty"`
	// Questionnaire for this condition
	Questionnaire []ConditionDefinitionQuestionnaire `json:"questionnaire,omitempty"`
	// Plan that is appropriate
	Plan []ConditionDefinitionPlan `json:"plan,omitempty"`
}
