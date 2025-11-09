package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeResearchElementDefinition is the FHIR resource type name for ResearchElementDefinition.
const ResourceTypeResearchElementDefinition = "ResearchElementDefinition"

// ResearchElementDefinitionCharacteristic represents a FHIR BackboneElement for ResearchElementDefinition.characteristic.
type ResearchElementDefinitionCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What code or expression defines members?
	Definition any `json:"definition"`
	// What code/value pairs define members?
	UsageContext []UsageContext `json:"usageContext,omitempty"`
	// Whether the characteristic includes or excludes members
	Exclude *bool `json:"exclude,omitempty"`
	// What unit is the outcome described in?
	UnitOfMeasure *CodeableConcept `json:"unitOfMeasure,omitempty"`
	// What time period does the study cover
	StudyEffectiveDescription *string `json:"studyEffectiveDescription,omitempty"`
	// What time period does the study cover
	StudyEffective *any `json:"studyEffective,omitempty"`
	// Observation time from study start
	StudyEffectiveTimeFromStart *Duration `json:"studyEffectiveTimeFromStart,omitempty"`
	// mean | median | mean-of-mean | mean-of-median | median-of-mean | median-of-median
	StudyEffectiveGroupMeasure *string `json:"studyEffectiveGroupMeasure,omitempty"`
	// What time period do participants cover
	ParticipantEffectiveDescription *string `json:"participantEffectiveDescription,omitempty"`
	// What time period do participants cover
	ParticipantEffective *any `json:"participantEffective,omitempty"`
	// Observation time from study start
	ParticipantEffectiveTimeFromStart *Duration `json:"participantEffectiveTimeFromStart,omitempty"`
	// mean | median | mean-of-mean | mean-of-median | median-of-mean | median-of-median
	ParticipantEffectiveGroupMeasure *string `json:"participantEffectiveGroupMeasure,omitempty"`
}

// ResearchElementDefinition represents a FHIR ResearchElementDefinition.
type ResearchElementDefinition struct {
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
	// Canonical identifier for this research element definition, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the research element definition
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the research element definition
	Version *string `json:"version,omitempty"`
	// Name for this research element definition (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this research element definition (human friendly)
	Title *string `json:"title,omitempty"`
	// Title for use in informal contexts
	ShortTitle *string `json:"shortTitle,omitempty"`
	// Subordinate title of the ResearchElementDefinition
	Subtitle *string `json:"subtitle,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// E.g. Patient, Practitioner, RelatedPerson, Organization, Location, Device
	Subject *any `json:"subject,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the research element definition
	Description *string `json:"description,omitempty"`
	// Used for footnotes or explanatory notes
	Comment []string `json:"comment,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for research element definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this research element definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Describes the clinical usage of the ResearchElementDefinition
	Usage *string `json:"usage,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the research element definition was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the research element definition was last reviewed
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the research element definition is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// The category of the ResearchElementDefinition, such as Education, Treatment, Assessment, etc.
	Topic []CodeableConcept `json:"topic,omitempty"`
	// Who authored the content
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the content
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the content
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the content
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations, etc.
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Logic used by the ResearchElementDefinition
	Library []string `json:"library,omitempty"`
	// population | exposure | outcome
	Type string `json:"type"`
	// dichotomous | continuous | descriptive
	VariableType *string `json:"variableType,omitempty"`
	// What defines the members of the research element
	Characteristic []ResearchElementDefinitionCharacteristic `json:"characteristic,omitempty"`
}
