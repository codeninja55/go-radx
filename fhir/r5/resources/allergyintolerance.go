package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeAllergyIntolerance is the FHIR resource type name for AllergyIntolerance.
const ResourceTypeAllergyIntolerance = "AllergyIntolerance"

// AllergyIntoleranceParticipant represents a FHIR BackboneElement for AllergyIntolerance.participant.
type AllergyIntoleranceParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of involvement
	Function *CodeableConcept `json:"function,omitempty"`
	// Who or what participated in the activities related to the allergy or intolerance
	Actor Reference `json:"actor"`
}

// AllergyIntoleranceReaction represents a FHIR BackboneElement for AllergyIntolerance.reaction.
type AllergyIntoleranceReaction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Specific substance or pharmaceutical product considered to be responsible for event
	Substance *CodeableConcept `json:"substance,omitempty"`
	// Clinical symptoms/signs associated with the Event
	Manifestation []CodeableReference `json:"manifestation,omitempty"`
	// Description of the event as a whole
	Description *string `json:"description,omitempty"`
	// Date(/time) when manifestations showed
	Onset *primitives.DateTime `json:"onset,omitempty"`
	// mild | moderate | severe (of event as a whole)
	Severity *string `json:"severity,omitempty"`
	// How the subject was exposed to the substance
	ExposureRoute *CodeableConcept `json:"exposureRoute,omitempty"`
	// Text about event not captured in other fields
	Note []Annotation `json:"note,omitempty"`
}

// AllergyIntolerance represents a FHIR AllergyIntolerance.
type AllergyIntolerance struct {
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
	// External ids for this item
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | inactive | resolved
	ClinicalStatus *CodeableConcept `json:"clinicalStatus,omitempty"`
	// unconfirmed | presumed | confirmed | refuted | entered-in-error
	VerificationStatus *CodeableConcept `json:"verificationStatus,omitempty"`
	// allergy | intolerance - Underlying mechanism (if known)
	Type *CodeableConcept `json:"type,omitempty"`
	// food | medication | environment | biologic
	Category []string `json:"category,omitempty"`
	// low | high | unable-to-assess
	Criticality *string `json:"criticality,omitempty"`
	// Code that identifies the allergy or intolerance
	Code *CodeableConcept `json:"code,omitempty"`
	// Who the allergy or intolerance is for
	Patient Reference `json:"patient"`
	// Encounter when the allergy or intolerance was asserted
	Encounter *Reference `json:"encounter,omitempty"`
	// When allergy or intolerance was identified
	Onset *any `json:"onset,omitempty"`
	// Date allergy or intolerance was first recorded
	RecordedDate *primitives.DateTime `json:"recordedDate,omitempty"`
	// Who or what participated in the activities related to the allergy or intolerance and how they were involved
	Participant []AllergyIntoleranceParticipant `json:"participant,omitempty"`
	// Date(/time) of last known occurrence of a reaction
	LastOccurrence *primitives.DateTime `json:"lastOccurrence,omitempty"`
	// Additional text not captured in other fields
	Note []Annotation `json:"note,omitempty"`
	// Adverse Reaction Events linked to exposure to substance
	Reaction []AllergyIntoleranceReaction `json:"reaction,omitempty"`
}
