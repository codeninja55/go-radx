package resources

// ResourceTypeSchedule is the FHIR resource type name for Schedule.
const ResourceTypeSchedule = "Schedule"

// Schedule represents a FHIR Schedule.
type Schedule struct {
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
	// External Ids for this item
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this schedule is in active use
	Active *bool `json:"active,omitempty"`
	// High-level category
	ServiceCategory []CodeableConcept `json:"serviceCategory,omitempty"`
	// Specific service
	ServiceType []CodeableConcept `json:"serviceType,omitempty"`
	// Type of specialty needed
	Specialty []CodeableConcept `json:"specialty,omitempty"`
	// Resource(s) that availability information is being provided for
	Actor []Reference `json:"actor,omitempty"`
	// Period of time covered by schedule
	PlanningHorizon *Period `json:"planningHorizon,omitempty"`
	// Comments on availability
	Comment *string `json:"comment,omitempty"`
}
