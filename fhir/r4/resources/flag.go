package resources

// ResourceTypeFlag is the FHIR resource type name for Flag.
const ResourceTypeFlag = "Flag"

// Flag represents a FHIR Flag.
type Flag struct {
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
	// Business identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | inactive | entered-in-error
	Status string `json:"status"`
	// Clinical, administrative, etc.
	Category []CodeableConcept `json:"category,omitempty"`
	// Coded or textual message to display to user
	Code CodeableConcept `json:"code"`
	// Who/What is flag about?
	Subject Reference `json:"subject"`
	// Time period when flag is active
	Period *Period `json:"period,omitempty"`
	// Alert relevant during encounter
	Encounter *Reference `json:"encounter,omitempty"`
	// Flag creator
	Author *Reference `json:"author,omitempty"`
}
