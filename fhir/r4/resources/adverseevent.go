package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeAdverseEvent is the FHIR resource type name for AdverseEvent.
const ResourceTypeAdverseEvent = "AdverseEvent"

// AdverseEventSuspectEntityCausality represents a FHIR BackboneElement for AdverseEvent.suspectEntity.causality.
type AdverseEventSuspectEntityCausality struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Assessment of if the entity caused the event
	Assessment *CodeableConcept `json:"assessment,omitempty"`
	// AdverseEvent.suspectEntity.causalityProductRelatedness
	ProductRelatedness *string `json:"productRelatedness,omitempty"`
	// AdverseEvent.suspectEntity.causalityAuthor
	Author *Reference `json:"author,omitempty"`
	// ProbabilityScale | Bayesian | Checklist
	Method *CodeableConcept `json:"method,omitempty"`
}

// AdverseEventSuspectEntity represents a FHIR BackboneElement for AdverseEvent.suspectEntity.
type AdverseEventSuspectEntity struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Refers to the specific entity that caused the adverse event
	Instance Reference `json:"instance"`
	// Information on the possible cause of the event
	Causality []AdverseEventSuspectEntityCausality `json:"causality,omitempty"`
}

// AdverseEvent represents a FHIR AdverseEvent.
type AdverseEvent struct {
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
	// Business identifier for the event
	Identifier *Identifier `json:"identifier,omitempty"`
	// actual | potential
	Actuality string `json:"actuality"`
	// product-problem | product-quality | product-use-error | wrong-dose | incorrect-prescribing-information | wrong-technique | wrong-route-of-administration | wrong-rate | wrong-duration | wrong-time | expired-drug | medical-device-use-error | problem-different-manufacturer | unsafe-physical-environment
	Category []CodeableConcept `json:"category,omitempty"`
	// Type of the event itself in relation to the subject
	Event *CodeableConcept `json:"event,omitempty"`
	// Subject impacted by event
	Subject Reference `json:"subject"`
	// Encounter created as part of
	Encounter *Reference `json:"encounter,omitempty"`
	// When the event occurred
	Date *primitives.DateTime `json:"date,omitempty"`
	// When the event was detected
	Detected *primitives.DateTime `json:"detected,omitempty"`
	// When the event was recorded
	RecordedDate *primitives.DateTime `json:"recordedDate,omitempty"`
	// Effect on the subject due to this event
	ResultingCondition []Reference `json:"resultingCondition,omitempty"`
	// Location where adverse event occurred
	Location *Reference `json:"location,omitempty"`
	// Seriousness of the event
	Seriousness *CodeableConcept `json:"seriousness,omitempty"`
	// mild | moderate | severe
	Severity *CodeableConcept `json:"severity,omitempty"`
	// resolved | recovering | ongoing | resolvedWithSequelae | fatal | unknown
	Outcome *CodeableConcept `json:"outcome,omitempty"`
	// Who recorded the adverse event
	Recorder *Reference `json:"recorder,omitempty"`
	// Who  was involved in the adverse event or the potential adverse event
	Contributor []Reference `json:"contributor,omitempty"`
	// The suspected agent causing the adverse event
	SuspectEntity []AdverseEventSuspectEntity `json:"suspectEntity,omitempty"`
	// AdverseEvent.subjectMedicalHistory
	SubjectMedicalHistory []Reference `json:"subjectMedicalHistory,omitempty"`
	// AdverseEvent.referenceDocument
	ReferenceDocument []Reference `json:"referenceDocument,omitempty"`
	// AdverseEvent.study
	Study []Reference `json:"study,omitempty"`
}
