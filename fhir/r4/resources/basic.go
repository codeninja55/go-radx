package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeBasic is the FHIR resource type name for Basic.
const ResourceTypeBasic = "Basic"

// Basic represents a FHIR Basic.
type Basic struct {
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
	// Kind of Resource
	Code CodeableConcept `json:"code"`
	// Identifies the focus of this resource
	Subject *Reference `json:"subject,omitempty"`
	// When created
	Created *primitives.Date `json:"created,omitempty"`
	// Who created
	Author *Reference `json:"author,omitempty"`
}
