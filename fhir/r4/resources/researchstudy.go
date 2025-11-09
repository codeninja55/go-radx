package resources

// ResourceTypeResearchStudy is the FHIR resource type name for ResearchStudy.
const ResourceTypeResearchStudy = "ResearchStudy"

// ResearchStudyArm represents a FHIR BackboneElement for ResearchStudy.arm.
type ResearchStudyArm struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for study arm
	Name string `json:"name"`
	// Categorization of study arm
	Type *CodeableConcept `json:"type,omitempty"`
	// Short explanation of study path
	Description *string `json:"description,omitempty"`
}

// ResearchStudyObjective represents a FHIR BackboneElement for ResearchStudy.objective.
type ResearchStudyObjective struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for the objective
	Name *string `json:"name,omitempty"`
	// primary | secondary | exploratory
	Type *CodeableConcept `json:"type,omitempty"`
}

// ResearchStudy represents a FHIR ResearchStudy.
type ResearchStudy struct {
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
	// Business Identifier for study
	Identifier []Identifier `json:"identifier,omitempty"`
	// Name for this study
	Title *string `json:"title,omitempty"`
	// Steps followed in executing study
	Protocol []Reference `json:"protocol,omitempty"`
	// Part of larger study
	PartOf []Reference `json:"partOf,omitempty"`
	// active | administratively-completed | approved | closed-to-accrual | closed-to-accrual-and-intervention | completed | disapproved | in-review | temporarily-closed-to-accrual | temporarily-closed-to-accrual-and-intervention | withdrawn
	Status string `json:"status"`
	// treatment | prevention | diagnostic | supportive-care | screening | health-services-research | basic-science | device-feasibility
	PrimaryPurposeType *CodeableConcept `json:"primaryPurposeType,omitempty"`
	// n-a | early-phase-1 | phase-1 | phase-1-phase-2 | phase-2 | phase-2-phase-3 | phase-3 | phase-4
	Phase *CodeableConcept `json:"phase,omitempty"`
	// Classifications for the study
	Category []CodeableConcept `json:"category,omitempty"`
	// Drugs, devices, etc. under study
	Focus []CodeableConcept `json:"focus,omitempty"`
	// Condition being studied
	Condition []CodeableConcept `json:"condition,omitempty"`
	// Contact details for the study
	Contact []ContactDetail `json:"contact,omitempty"`
	// References and dependencies
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Used to search for the study
	Keyword []CodeableConcept `json:"keyword,omitempty"`
	// Geographic region(s) for study
	Location []CodeableConcept `json:"location,omitempty"`
	// What this is study doing
	Description *string `json:"description,omitempty"`
	// Inclusion & exclusion criteria
	Enrollment []Reference `json:"enrollment,omitempty"`
	// When the study began and ended
	Period *Period `json:"period,omitempty"`
	// Organization that initiates and is legally responsible for the study
	Sponsor *Reference `json:"sponsor,omitempty"`
	// Researcher who oversees multiple aspects of the study
	PrincipalInvestigator *Reference `json:"principalInvestigator,omitempty"`
	// Facility where study activities are conducted
	Site []Reference `json:"site,omitempty"`
	// accrual-goal-met | closed-due-to-toxicity | closed-due-to-lack-of-study-progress | temporarily-closed-per-study-design
	ReasonStopped *CodeableConcept `json:"reasonStopped,omitempty"`
	// Comments made about the study
	Note []Annotation `json:"note,omitempty"`
	// Defined path through the study for a subject
	Arm []ResearchStudyArm `json:"arm,omitempty"`
	// A goal for the study
	Objective []ResearchStudyObjective `json:"objective,omitempty"`
}
