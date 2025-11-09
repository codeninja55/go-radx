package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeAdverseEvent is the FHIR resource type name for AdverseEvent.
const ResourceTypeAdverseEvent = "AdverseEvent"

// AdverseEventParticipant represents a FHIR BackboneElement for AdverseEvent.participant.
type AdverseEventParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of involvement
	Function *CodeableConcept `json:"function,omitempty"`
	// Who was involved in the adverse event or the potential adverse event
	Actor Reference `json:"actor"`
}

// AdverseEventSuspectEntityCausality represents a FHIR BackboneElement for AdverseEvent.suspectEntity.causality.
type AdverseEventSuspectEntityCausality struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Method of evaluating the relatedness of the suspected entity to the event
	AssessmentMethod *CodeableConcept `json:"assessmentMethod,omitempty"`
	// Result of the assessment regarding the relatedness of the suspected entity to the event
	EntityRelatedness *CodeableConcept `json:"entityRelatedness,omitempty"`
	// Author of the information on the possible cause of the event
	Author *Reference `json:"author,omitempty"`
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
	Instance any `json:"instance"`
	// Information on the possible cause of the event
	Causality *AdverseEventSuspectEntityCausality `json:"causality,omitempty"`
}

// AdverseEventContributingFactor represents a FHIR BackboneElement for AdverseEvent.contributingFactor.
type AdverseEventContributingFactor struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Item suspected to have increased the probability or severity of the adverse event
	Item any `json:"item"`
}

// AdverseEventPreventiveAction represents a FHIR BackboneElement for AdverseEvent.preventiveAction.
type AdverseEventPreventiveAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Action that contributed to avoiding the adverse event
	Item any `json:"item"`
}

// AdverseEventMitigatingAction represents a FHIR BackboneElement for AdverseEvent.mitigatingAction.
type AdverseEventMitigatingAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Ameliorating action taken after the adverse event occured in order to reduce the extent of harm
	Item any `json:"item"`
}

// AdverseEventSupportingInfo represents a FHIR BackboneElement for AdverseEvent.supportingInfo.
type AdverseEventSupportingInfo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Subject medical history or document relevant to this adverse event
	Item any `json:"item"`
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
	Identifier []Identifier `json:"identifier,omitempty"`
	// in-progress | completed | entered-in-error | unknown
	Status string `json:"status"`
	// actual | potential
	Actuality string `json:"actuality"`
	// wrong-patient | procedure-mishap | medication-mishap | device | unsafe-physical-environment | hospital-aquired-infection | wrong-body-site
	Category []CodeableConcept `json:"category,omitempty"`
	// Event or incident that occurred or was averted
	Code *CodeableConcept `json:"code,omitempty"`
	// Subject impacted by event
	Subject Reference `json:"subject"`
	// The Encounter associated with the start of the AdverseEvent
	Encounter *Reference `json:"encounter,omitempty"`
	// When the event occurred
	Occurrence *any `json:"occurrence,omitempty"`
	// When the event was detected
	Detected *primitives.DateTime `json:"detected,omitempty"`
	// When the event was recorded
	RecordedDate *primitives.DateTime `json:"recordedDate,omitempty"`
	// Effect on the subject due to this event
	ResultingEffect []Reference `json:"resultingEffect,omitempty"`
	// Location where adverse event occurred
	Location *Reference `json:"location,omitempty"`
	// Seriousness or gravity of the event
	Seriousness *CodeableConcept `json:"seriousness,omitempty"`
	// Type of outcome from the adverse event
	Outcome []CodeableConcept `json:"outcome,omitempty"`
	// Who recorded the adverse event
	Recorder *Reference `json:"recorder,omitempty"`
	// Who was involved in the adverse event or the potential adverse event and what they did
	Participant []AdverseEventParticipant `json:"participant,omitempty"`
	// Research study that the subject is enrolled in
	Study []Reference `json:"study,omitempty"`
	// Considered likely or probable or anticipated in the research study
	ExpectedInResearchStudy *bool `json:"expectedInResearchStudy,omitempty"`
	// The suspected agent causing the adverse event
	SuspectEntity []AdverseEventSuspectEntity `json:"suspectEntity,omitempty"`
	// Contributing factors suspected to have increased the probability or severity of the adverse event
	ContributingFactor []AdverseEventContributingFactor `json:"contributingFactor,omitempty"`
	// Preventive actions that contributed to avoiding the adverse event
	PreventiveAction []AdverseEventPreventiveAction `json:"preventiveAction,omitempty"`
	// Ameliorating actions taken after the adverse event occured in order to reduce the extent of harm
	MitigatingAction []AdverseEventMitigatingAction `json:"mitigatingAction,omitempty"`
	// Supporting information relevant to the event
	SupportingInfo []AdverseEventSupportingInfo `json:"supportingInfo,omitempty"`
	// Comment on adverse event
	Note []Annotation `json:"note,omitempty"`
}
