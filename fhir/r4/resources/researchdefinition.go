package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeResearchDefinition is the FHIR resource type name for ResearchDefinition.
const ResourceTypeResearchDefinition = "ResearchDefinition"

// ResearchDefinition represents a FHIR ResearchDefinition.
type ResearchDefinition struct {
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
	// Canonical identifier for this research definition, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the research definition
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the research definition
	Version *string `json:"version,omitempty"`
	// Name for this research definition (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this research definition (human friendly)
	Title *string `json:"title,omitempty"`
	// Title for use in informal contexts
	ShortTitle *string `json:"shortTitle,omitempty"`
	// Subordinate title of the ResearchDefinition
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
	// Natural language description of the research definition
	Description *string `json:"description,omitempty"`
	// Used for footnotes or explanatory notes
	Comment []string `json:"comment,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for research definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this research definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Describes the clinical usage of the ResearchDefinition
	Usage *string `json:"usage,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the research definition was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the research definition was last reviewed
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the research definition is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// The category of the ResearchDefinition, such as Education, Treatment, Assessment, etc.
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
	// Logic used by the ResearchDefinition
	Library []string `json:"library,omitempty"`
	// What population?
	Population Reference `json:"population"`
	// What exposure?
	Exposure *Reference `json:"exposure,omitempty"`
	// What alternative exposure state?
	ExposureAlternative *Reference `json:"exposureAlternative,omitempty"`
	// What outcome?
	Outcome *Reference `json:"outcome,omitempty"`
}
