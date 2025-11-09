package resources

// ResourceTypeMedicinalProductUndesirableEffect is the FHIR resource type name for MedicinalProductUndesirableEffect.
const ResourceTypeMedicinalProductUndesirableEffect = "MedicinalProductUndesirableEffect"

// MedicinalProductUndesirableEffect represents a FHIR MedicinalProductUndesirableEffect.
type MedicinalProductUndesirableEffect struct {
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
	// The medication for which this is an indication
	Subject []Reference `json:"subject,omitempty"`
	// The symptom, condition or undesirable effect
	SymptomConditionEffect *CodeableConcept `json:"symptomConditionEffect,omitempty"`
	// Classification of the effect
	Classification *CodeableConcept `json:"classification,omitempty"`
	// The frequency of occurrence of the effect
	FrequencyOfOccurrence *CodeableConcept `json:"frequencyOfOccurrence,omitempty"`
	// The population group to which this applies
	Population []Population `json:"population,omitempty"`
}
