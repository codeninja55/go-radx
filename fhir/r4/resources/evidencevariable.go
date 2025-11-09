package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeEvidenceVariable is the FHIR resource type name for EvidenceVariable.
const ResourceTypeEvidenceVariable = "EvidenceVariable"

// EvidenceVariableCharacteristic represents a FHIR BackboneElement for EvidenceVariable.characteristic.
type EvidenceVariableCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Natural language description of the characteristic
	Description *string `json:"description,omitempty"`
	// What code or expression defines members?
	Definition any `json:"definition"`
	// What code/value pairs define members?
	UsageContext []UsageContext `json:"usageContext,omitempty"`
	// Whether the characteristic includes or excludes members
	Exclude *bool `json:"exclude,omitempty"`
	// What time period do participants cover
	ParticipantEffective *any `json:"participantEffective,omitempty"`
	// Observation time from study start
	TimeFromStart *Duration `json:"timeFromStart,omitempty"`
	// mean | median | mean-of-mean | mean-of-median | median-of-mean | median-of-median
	GroupMeasure *string `json:"groupMeasure,omitempty"`
}

// EvidenceVariable represents a FHIR EvidenceVariable.
type EvidenceVariable struct {
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
	// Canonical identifier for this evidence variable, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the evidence variable
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the evidence variable
	Version *string `json:"version,omitempty"`
	// Name for this evidence variable (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this evidence variable (human friendly)
	Title *string `json:"title,omitempty"`
	// Title for use in informal contexts
	ShortTitle *string `json:"shortTitle,omitempty"`
	// Subordinate title of the EvidenceVariable
	Subtitle *string `json:"subtitle,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the evidence variable
	Description *string `json:"description,omitempty"`
	// Used for footnotes or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for evidence variable (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the evidence variable was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the evidence variable was last reviewed
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the evidence variable is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// The category of the EvidenceVariable, such as Education, Treatment, Assessment, etc.
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
	// dichotomous | continuous | descriptive
	Type *string `json:"type,omitempty"`
	// What defines the members of the evidence element
	Characteristic []EvidenceVariableCharacteristic `json:"characteristic,omitempty"`
}
