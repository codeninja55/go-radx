package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeResearchSubject is the FHIR resource type name for ResearchSubject.
const ResourceTypeResearchSubject = "ResearchSubject"

// ResearchSubjectProgress represents a FHIR BackboneElement for ResearchSubject.progress.
type ResearchSubjectProgress struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// state | milestone
	Type *CodeableConcept `json:"type,omitempty"`
	// candidate | eligible | follow-up | ineligible | not-registered | off-study | on-study | on-study-intervention | on-study-observation | pending-on-study | potential-candidate | screening | withdrawn
	SubjectState *CodeableConcept `json:"subjectState,omitempty"`
	// SignedUp | Screened | Randomized
	Milestone *CodeableConcept `json:"milestone,omitempty"`
	// State change reason
	Reason *CodeableConcept `json:"reason,omitempty"`
	// State change date
	StartDate *primitives.DateTime `json:"startDate,omitempty"`
	// State change date
	EndDate *primitives.DateTime `json:"endDate,omitempty"`
}

// ResearchSubject represents a FHIR ResearchSubject.
type ResearchSubject struct {
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
	// Business Identifier for research subject in a study
	Identifier []Identifier `json:"identifier,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// Subject status
	Progress []ResearchSubjectProgress `json:"progress,omitempty"`
	// Start and end of participation
	Period *Period `json:"period,omitempty"`
	// Study subject is part of
	Study Reference `json:"study"`
	// Who or what is part of study
	Subject Reference `json:"subject"`
	// What path should be followed
	AssignedComparisonGroup *string `json:"assignedComparisonGroup,omitempty"`
	// What path was followed
	ActualComparisonGroup *string `json:"actualComparisonGroup,omitempty"`
	// Agreement to participate in study
	Consent []Reference `json:"consent,omitempty"`
}
