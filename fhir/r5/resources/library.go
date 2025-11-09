package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeLibrary is the FHIR resource type name for Library.
const ResourceTypeLibrary = "Library"

// Library represents a FHIR Library.
type Library struct {
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
	// Canonical identifier for this library, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the library
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the library
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this library (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this library (human friendly)
	Title *string `json:"title,omitempty"`
	// Subordinate title of the library
	Subtitle *string `json:"subtitle,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// logic-library | model-definition | asset-collection | module-definition
	Type CodeableConcept `json:"type"`
	// Type of individual the library content is focused on
	Subject *any `json:"subject,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the library
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for library (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this library is defined
	Purpose *string `json:"purpose,omitempty"`
	// Describes the clinical usage of the library
	Usage *string `json:"usage,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the library was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the library was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the library is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// E.g. Education, Treatment, Assessment, etc
	Topic []CodeableConcept `json:"topic,omitempty"`
	// Who authored the content
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the content
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the content
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the content
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations, etc
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Parameters defined by the library
	Parameter []ParameterDefinition `json:"parameter,omitempty"`
	// What data is referenced by this library
	DataRequirement []DataRequirement `json:"dataRequirement,omitempty"`
	// Contents of the library, either embedded or referenced
	Content []Attachment `json:"content,omitempty"`
}
