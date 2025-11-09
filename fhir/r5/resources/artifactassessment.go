package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeArtifactAssessment is the FHIR resource type name for ArtifactAssessment.
const ResourceTypeArtifactAssessment = "ArtifactAssessment"

// ArtifactAssessmentContentComponent represents a FHIR BackboneElement for ArtifactAssessment.content.component.
type ArtifactAssessmentContentComponent struct {
}

// ArtifactAssessmentContent represents a FHIR BackboneElement for ArtifactAssessment.content.
type ArtifactAssessmentContent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// comment | classifier | rating | container | response | change-request
	InformationType *string `json:"informationType,omitempty"`
	// Brief summary of the content
	Summary *string `json:"summary,omitempty"`
	// What type of content
	Type *CodeableConcept `json:"type,omitempty"`
	// Rating, classifier, or assessment
	Classifier []CodeableConcept `json:"classifier,omitempty"`
	// Quantitative rating
	Quantity *Quantity `json:"quantity,omitempty"`
	// Who authored the content
	Author *Reference `json:"author,omitempty"`
	// What the comment is directed to
	Path []string `json:"path,omitempty"`
	// Additional information
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Acceptable to publicly share the resource content
	FreeToShare *bool `json:"freeToShare,omitempty"`
	// Contained content
	Component []ArtifactAssessmentContentComponent `json:"component,omitempty"`
}

// ArtifactAssessment represents a FHIR ArtifactAssessment.
type ArtifactAssessment struct {
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
	// Additional identifier for the artifact assessment
	Identifier []Identifier `json:"identifier,omitempty"`
	// A short title for the assessment for use in displaying and selecting
	Title *string `json:"title,omitempty"`
	// How to cite the comment or rating
	CiteAs *any `json:"citeAs,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the artifact assessment was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the artifact assessment was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// The artifact assessed, commented upon or rated
	Artifact any `json:"artifact"`
	// Comment, classifier, or rating content
	Content []ArtifactAssessmentContent `json:"content,omitempty"`
	// submitted | triaged | waiting-for-input | resolved-no-change | resolved-change-required | deferred | duplicate | applied | published | entered-in-error
	WorkflowStatus *string `json:"workflowStatus,omitempty"`
	// unresolved | not-persuasive | persuasive | persuasive-with-modification | not-persuasive-with-modification
	Disposition *string `json:"disposition,omitempty"`
}
