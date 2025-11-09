package resources

// ResourceTypeMedicinalProductIndication is the FHIR resource type name for MedicinalProductIndication.
const ResourceTypeMedicinalProductIndication = "MedicinalProductIndication"

// MedicinalProductIndicationOtherTherapy represents a FHIR BackboneElement for MedicinalProductIndication.otherTherapy.
type MedicinalProductIndicationOtherTherapy struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of relationship between the medicinal product indication or contraindication and another therapy
	TherapyRelationshipType CodeableConcept `json:"therapyRelationshipType"`
	// Reference to a specific medication (active substance, medicinal product or class of products) as part of an indication or contraindication
	Medication any `json:"medication"`
}

// MedicinalProductIndication represents a FHIR MedicinalProductIndication.
type MedicinalProductIndication struct {
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
	// The disease, symptom or procedure that is the indication for treatment
	DiseaseSymptomProcedure *CodeableConcept `json:"diseaseSymptomProcedure,omitempty"`
	// The status of the disease or symptom for which the indication applies
	DiseaseStatus *CodeableConcept `json:"diseaseStatus,omitempty"`
	// Comorbidity (concurrent condition) or co-infection as part of the indication
	Comorbidity []CodeableConcept `json:"comorbidity,omitempty"`
	// The intended effect, aim or strategy to be achieved by the indication
	IntendedEffect *CodeableConcept `json:"intendedEffect,omitempty"`
	// Timing or duration information as part of the indication
	Duration *Quantity `json:"duration,omitempty"`
	// Information about the use of the medicinal product in relation to other therapies described as part of the indication
	OtherTherapy []MedicinalProductIndicationOtherTherapy `json:"otherTherapy,omitempty"`
	// Describe the undesirable effects of the medicinal product
	UndesirableEffect []Reference `json:"undesirableEffect,omitempty"`
	// The population group to which this applies
	Population []Population `json:"population,omitempty"`
}
