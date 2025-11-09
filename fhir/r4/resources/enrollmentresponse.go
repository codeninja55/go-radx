package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeEnrollmentResponse is the FHIR resource type name for EnrollmentResponse.
const ResourceTypeEnrollmentResponse = "EnrollmentResponse"

// EnrollmentResponse represents a FHIR EnrollmentResponse.
type EnrollmentResponse struct {
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
	// Claim reference
	Request *Reference `json:"request,omitempty"`
	// queued | complete | error | partial
	Outcome *string `json:"outcome,omitempty"`
	// Disposition Message
	Disposition *string `json:"disposition,omitempty"`
	// Creation date
	Created *primitives.DateTime `json:"created,omitempty"`
	// Insurer
	Organization *Reference `json:"organization,omitempty"`
	// Responsible practitioner
	RequestProvider *Reference `json:"requestProvider,omitempty"`
}
