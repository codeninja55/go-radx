package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeImmunizationEvaluation is the FHIR resource type name for ImmunizationEvaluation.
const ResourceTypeImmunizationEvaluation = "ImmunizationEvaluation"

// ImmunizationEvaluation represents a FHIR ImmunizationEvaluation.
type ImmunizationEvaluation struct {
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
	// completed | entered-in-error
	Status string `json:"status"`
	// Who this evaluation is for
	Patient Reference `json:"patient"`
	// Date evaluation was performed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Who is responsible for publishing the recommendations
	Authority *Reference `json:"authority,omitempty"`
	// The vaccine preventable disease schedule being evaluated
	TargetDisease CodeableConcept `json:"targetDisease"`
	// Immunization being evaluated
	ImmunizationEvent Reference `json:"immunizationEvent"`
	// Status of the dose relative to published recommendations
	DoseStatus CodeableConcept `json:"doseStatus"`
	// Reason why the doese is considered valid, invalid or some other status
	DoseStatusReason []CodeableConcept `json:"doseStatusReason,omitempty"`
	// Evaluation notes
	Description *string `json:"description,omitempty"`
	// Name of vaccine series
	Series *string `json:"series,omitempty"`
	// Dose number within series
	DoseNumber *string `json:"doseNumber,omitempty"`
	// Recommended number of doses for immunity
	SeriesDoses *string `json:"seriesDoses,omitempty"`
}
