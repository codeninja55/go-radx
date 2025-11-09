package resources

// ResourceTypeBodyStructure is the FHIR resource type name for BodyStructure.
const ResourceTypeBodyStructure = "BodyStructure"

// BodyStructure represents a FHIR BodyStructure.
type BodyStructure struct {
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
	// Bodystructure identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this record is in active use
	Active *bool `json:"active,omitempty"`
	// Kind of Structure
	Morphology *CodeableConcept `json:"morphology,omitempty"`
	// Body site
	Location *CodeableConcept `json:"location,omitempty"`
	// Body site modifier
	LocationQualifier []CodeableConcept `json:"locationQualifier,omitempty"`
	// Text description
	Description *string `json:"description,omitempty"`
	// Attached images
	Image []Attachment `json:"image,omitempty"`
	// Who this is about
	Patient Reference `json:"patient"`
}
