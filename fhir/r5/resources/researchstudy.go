package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeResearchStudy is the FHIR resource type name for ResearchStudy.
const ResourceTypeResearchStudy = "ResearchStudy"

// ResearchStudyLabel represents a FHIR BackboneElement for ResearchStudy.label.
type ResearchStudyLabel struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// primary | official | scientific | plain-language | subtitle | short-title | acronym | earlier-title | language | auto-translated | human-use | machine-use | duplicate-uid
	Type *CodeableConcept `json:"type,omitempty"`
	// The name
	Value *string `json:"value,omitempty"`
}

// ResearchStudyAssociatedParty represents a FHIR BackboneElement for ResearchStudy.associatedParty.
type ResearchStudyAssociatedParty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of associated party
	Name *string `json:"name,omitempty"`
	// sponsor | lead-sponsor | sponsor-investigator | primary-investigator | collaborator | funding-source | general-contact | recruitment-contact | sub-investigator | study-director | study-chair
	Role CodeableConcept `json:"role"`
	// When active in the role
	Period []Period `json:"period,omitempty"`
	// nih | fda | government | nonprofit | academic | industry
	Classifier []CodeableConcept `json:"classifier,omitempty"`
	// Individual or organization associated with study (use practitionerRole to specify their organisation)
	Party *Reference `json:"party,omitempty"`
}

// ResearchStudyProgressStatus represents a FHIR BackboneElement for ResearchStudy.progressStatus.
type ResearchStudyProgressStatus struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for status or state (e.g. recruitment status)
	State CodeableConcept `json:"state"`
	// Actual if true else anticipated
	Actual *bool `json:"actual,omitempty"`
	// Date range
	Period *Period `json:"period,omitempty"`
}

// ResearchStudyRecruitment represents a FHIR BackboneElement for ResearchStudy.recruitment.
type ResearchStudyRecruitment struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Estimated total number of participants to be enrolled
	TargetNumber *uint `json:"targetNumber,omitempty"`
	// Actual total number of participants enrolled in study
	ActualNumber *uint `json:"actualNumber,omitempty"`
	// Inclusion and exclusion criteria
	Eligibility *Reference `json:"eligibility,omitempty"`
	// Group of participants who were enrolled in study
	ActualGroup *Reference `json:"actualGroup,omitempty"`
}

// ResearchStudyComparisonGroup represents a FHIR BackboneElement for ResearchStudy.comparisonGroup.
type ResearchStudyComparisonGroup struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Allows the comparisonGroup for the study and the comparisonGroup for the subject to be linked easily
	LinkId *string `json:"linkId,omitempty"`
	// Label for study comparisonGroup
	Name string `json:"name"`
	// Categorization of study comparisonGroup
	Type *CodeableConcept `json:"type,omitempty"`
	// Short explanation of study path
	Description *string `json:"description,omitempty"`
	// Interventions or exposures in this comparisonGroup or cohort
	IntendedExposure []Reference `json:"intendedExposure,omitempty"`
	// Group of participants who were enrolled in study comparisonGroup
	ObservedGroup *Reference `json:"observedGroup,omitempty"`
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
	// Description of the objective
	Description *string `json:"description,omitempty"`
}

// ResearchStudyOutcomeMeasure represents a FHIR BackboneElement for ResearchStudy.outcomeMeasure.
type ResearchStudyOutcomeMeasure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for the outcome
	Name *string `json:"name,omitempty"`
	// primary | secondary | exploratory
	Type []CodeableConcept `json:"type,omitempty"`
	// Description of the outcome
	Description *string `json:"description,omitempty"`
	// Structured outcome definition
	Reference *Reference `json:"reference,omitempty"`
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
	// Canonical identifier for this study resource
	URL *string `json:"url,omitempty"`
	// Business Identifier for study
	Identifier []Identifier `json:"identifier,omitempty"`
	// The business version for the study record
	Version *string `json:"version,omitempty"`
	// Name for this study (computer friendly)
	Name *string `json:"name,omitempty"`
	// Human readable name of the study
	Title *string `json:"title,omitempty"`
	// Additional names for the study
	Label []ResearchStudyLabel `json:"label,omitempty"`
	// Steps followed in executing study
	Protocol []Reference `json:"protocol,omitempty"`
	// Part of larger study
	PartOf []Reference `json:"partOf,omitempty"`
	// References, URLs, and attachments
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Date the resource last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// treatment | prevention | diagnostic | supportive-care | screening | health-services-research | basic-science | device-feasibility
	PrimaryPurposeType *CodeableConcept `json:"primaryPurposeType,omitempty"`
	// n-a | early-phase-1 | phase-1 | phase-1-phase-2 | phase-2 | phase-2-phase-3 | phase-3 | phase-4
	Phase *CodeableConcept `json:"phase,omitempty"`
	// Classifications of the study design characteristics
	StudyDesign []CodeableConcept `json:"studyDesign,omitempty"`
	// Drugs, devices, etc. under study
	Focus []CodeableReference `json:"focus,omitempty"`
	// Condition being studied
	Condition []CodeableConcept `json:"condition,omitempty"`
	// Used to search for the study
	Keyword []CodeableConcept `json:"keyword,omitempty"`
	// Geographic area for the study
	Region []CodeableConcept `json:"region,omitempty"`
	// Brief text explaining the study
	DescriptionSummary *string `json:"descriptionSummary,omitempty"`
	// Detailed narrative of the study
	Description *string `json:"description,omitempty"`
	// When the study began and ended
	Period *Period `json:"period,omitempty"`
	// Facility where study activities are conducted
	Site []Reference `json:"site,omitempty"`
	// Comments made about the study
	Note []Annotation `json:"note,omitempty"`
	// Classification for the study
	Classifier []CodeableConcept `json:"classifier,omitempty"`
	// Sponsors, collaborators, and other parties
	AssociatedParty []ResearchStudyAssociatedParty `json:"associatedParty,omitempty"`
	// Status of study with time for that status
	ProgressStatus []ResearchStudyProgressStatus `json:"progressStatus,omitempty"`
	// accrual-goal-met | closed-due-to-toxicity | closed-due-to-lack-of-study-progress | temporarily-closed-per-study-design
	WhyStopped *CodeableConcept `json:"whyStopped,omitempty"`
	// Target or actual group of participants enrolled in study
	Recruitment *ResearchStudyRecruitment `json:"recruitment,omitempty"`
	// Defined path through the study for a subject
	ComparisonGroup []ResearchStudyComparisonGroup `json:"comparisonGroup,omitempty"`
	// A goal for the study
	Objective []ResearchStudyObjective `json:"objective,omitempty"`
	// A variable measured during the study
	OutcomeMeasure []ResearchStudyOutcomeMeasure `json:"outcomeMeasure,omitempty"`
	// Link to results generated during the study
	Result []Reference `json:"result,omitempty"`
}
