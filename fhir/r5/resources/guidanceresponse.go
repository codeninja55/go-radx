package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeGuidanceResponse is the FHIR resource type name for GuidanceResponse.
const ResourceTypeGuidanceResponse = "GuidanceResponse"

// GuidanceResponse represents a FHIR GuidanceResponse.
type GuidanceResponse struct {
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
	// The identifier of the request associated with this response, if any
	RequestIdentifier *Identifier `json:"requestIdentifier,omitempty"`
	// Business identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// What guidance was requested
	Module any `json:"module"`
	// success | data-requested | data-required | in-progress | failure | entered-in-error
	Status string `json:"status"`
	// Patient the request was performed for
	Subject *Reference `json:"subject,omitempty"`
	// Encounter during which the response was returned
	Encounter *Reference `json:"encounter,omitempty"`
	// When the guidance response was processed
	OccurrenceDateTime *primitives.DateTime `json:"occurrenceDateTime,omitempty"`
	// Device returning the guidance
	Performer *Reference `json:"performer,omitempty"`
	// Why guidance is needed
	Reason []CodeableReference `json:"reason,omitempty"`
	// Additional notes about the response
	Note []Annotation `json:"note,omitempty"`
	// Messages resulting from the evaluation of the artifact or artifacts
	EvaluationMessage *Reference `json:"evaluationMessage,omitempty"`
	// The output parameters of the evaluation, if any
	OutputParameters *Reference `json:"outputParameters,omitempty"`
	// Proposed actions, if any
	Result []Reference `json:"result,omitempty"`
	// Additional required data
	DataRequirement []DataRequirement `json:"dataRequirement,omitempty"`
}
