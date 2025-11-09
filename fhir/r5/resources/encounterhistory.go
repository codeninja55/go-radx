package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeEncounterHistory is the FHIR resource type name for EncounterHistory.
const ResourceTypeEncounterHistory = "EncounterHistory"

// EncounterHistoryLocation represents a FHIR BackboneElement for EncounterHistory.location.
type EncounterHistoryLocation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Location the encounter takes place
	Location Reference `json:"location"`
	// The physical type of the location (usually the level in the location hierarchy - bed, room, ward, virtual etc.)
	Form *CodeableConcept `json:"form,omitempty"`
}

// EncounterHistory represents a FHIR EncounterHistory.
type EncounterHistory struct {
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
	// The Encounter associated with this set of historic values
	Encounter *Reference `json:"encounter,omitempty"`
	// Identifier(s) by which this encounter is known
	Identifier []Identifier `json:"identifier,omitempty"`
	// planned | in-progress | on-hold | discharged | completed | cancelled | discontinued | entered-in-error | unknown
	Status string `json:"status"`
	// Classification of patient encounter
	Class CodeableConcept `json:"class"`
	// Specific type of encounter
	Type []CodeableConcept `json:"type,omitempty"`
	// Specific type of service
	ServiceType []CodeableReference `json:"serviceType,omitempty"`
	// The patient or group related to this encounter
	Subject *Reference `json:"subject,omitempty"`
	// The current status of the subject in relation to the Encounter
	SubjectStatus *CodeableConcept `json:"subjectStatus,omitempty"`
	// The actual start and end time associated with this set of values associated with the encounter
	ActualPeriod *Period `json:"actualPeriod,omitempty"`
	// The planned start date/time (or admission date) of the encounter
	PlannedStartDate *primitives.DateTime `json:"plannedStartDate,omitempty"`
	// The planned end date/time (or discharge date) of the encounter
	PlannedEndDate *primitives.DateTime `json:"plannedEndDate,omitempty"`
	// Actual quantity of time the encounter lasted (less time absent)
	Length *Duration `json:"length,omitempty"`
	// Location of the patient at this point in the encounter
	Location []EncounterHistoryLocation `json:"location,omitempty"`
}
