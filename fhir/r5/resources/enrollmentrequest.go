package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeEnrollmentRequest is the FHIR resource type name for EnrollmentRequest.
const ResourceTypeEnrollmentRequest = "EnrollmentRequest"

// EnrollmentRequest represents a FHIR EnrollmentRequest.
type EnrollmentRequest struct {
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
	// Business Identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | cancelled | draft | entered-in-error
	Status *string `json:"status,omitempty"`
	// Creation date
	Created *primitives.DateTime `json:"created,omitempty"`
	// Target
	Insurer *Reference `json:"insurer,omitempty"`
	// Responsible practitioner
	Provider *Reference `json:"provider,omitempty"`
	// The subject to be enrolled
	Candidate *Reference `json:"candidate,omitempty"`
	// Insurance information
	Coverage *Reference `json:"coverage,omitempty"`
}
