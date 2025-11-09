package resources

// ResourceTypeMedicinalProductContraindication is the FHIR resource type name for MedicinalProductContraindication.
const ResourceTypeMedicinalProductContraindication = "MedicinalProductContraindication"

// MedicinalProductContraindicationOtherTherapy represents a FHIR BackboneElement for MedicinalProductContraindication.otherTherapy.
type MedicinalProductContraindicationOtherTherapy struct {
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

// MedicinalProductContraindication represents a FHIR MedicinalProductContraindication.
type MedicinalProductContraindication struct {
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
	// The disease, symptom or procedure for the contraindication
	Disease *CodeableConcept `json:"disease,omitempty"`
	// The status of the disease or symptom for the contraindication
	DiseaseStatus *CodeableConcept `json:"diseaseStatus,omitempty"`
	// A comorbidity (concurrent condition) or coinfection
	Comorbidity []CodeableConcept `json:"comorbidity,omitempty"`
	// Information about the use of the medicinal product in relation to other therapies as part of the indication
	TherapeuticIndication []Reference `json:"therapeuticIndication,omitempty"`
	// Information about the use of the medicinal product in relation to other therapies described as part of the indication
	OtherTherapy []MedicinalProductContraindicationOtherTherapy `json:"otherTherapy,omitempty"`
	// The population group to which this applies
	Population []Population `json:"population,omitempty"`
}
