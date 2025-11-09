package resources

// ResourceTypeResearchSubject is the FHIR resource type name for ResearchSubject.
const ResourceTypeResearchSubject = "ResearchSubject"

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
	// candidate | eligible | follow-up | ineligible | not-registered | off-study | on-study | on-study-intervention | on-study-observation | pending-on-study | potential-candidate | screening | withdrawn
	Status string `json:"status"`
	// Start and end of participation
	Period *Period `json:"period,omitempty"`
	// Study subject is part of
	Study Reference `json:"study"`
	// Who is part of study
	Individual Reference `json:"individual"`
	// What path should be followed
	AssignedArm *string `json:"assignedArm,omitempty"`
	// What path was followed
	ActualArm *string `json:"actualArm,omitempty"`
	// Agreement to participate in study
	Consent *Reference `json:"consent,omitempty"`
}
