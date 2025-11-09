package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypePatient is the FHIR resource type name for Patient.
const ResourceTypePatient = "Patient"

// PatientContact represents a FHIR BackboneElement for Patient.contact.
type PatientContact struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The kind of relationship
	Relationship []CodeableConcept `json:"relationship,omitempty"`
	// A name associated with the contact person
	Name *HumanName `json:"name,omitempty"`
	// A contact detail for the person
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Address for the contact person
	Address *Address `json:"address,omitempty"`
	// male | female | other | unknown
	Gender *string `json:"gender,omitempty"`
	// Organization that is associated with the contact
	Organization *Reference `json:"organization,omitempty"`
	// The period during which this contact person or organization is valid to be contacted relating to this patient
	Period *Period `json:"period,omitempty"`
}

// PatientCommunication represents a FHIR BackboneElement for Patient.communication.
type PatientCommunication struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The language which can be used to communicate with the patient about his or her health
	Language CodeableConcept `json:"language"`
	// Language preference indicator
	Preferred *bool `json:"preferred,omitempty"`
}

// PatientLink represents a FHIR BackboneElement for Patient.link.
type PatientLink struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The other patient or related person resource that the link refers to
	Other Reference `json:"other"`
	// replaced-by | replaces | refer | seealso
	Type string `json:"type"`
}

// Patient represents a FHIR Patient.
type Patient struct {
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
	// An identifier for this patient
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this patient's record is in active use
	Active *bool `json:"active,omitempty"`
	// A name associated with the patient
	Name []HumanName `json:"name,omitempty"`
	// A contact detail for the individual
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// male | female | other | unknown
	Gender *string `json:"gender,omitempty"`
	// The date of birth for the individual
	BirthDate *primitives.Date `json:"birthDate,omitempty"`
	// Indicates if the individual is deceased or not
	Deceased *any `json:"deceased,omitempty"`
	// An address for the individual
	Address []Address `json:"address,omitempty"`
	// Marital (civil) status of a patient
	MaritalStatus *CodeableConcept `json:"maritalStatus,omitempty"`
	// Whether patient is part of a multiple birth
	MultipleBirth *any `json:"multipleBirth,omitempty"`
	// Image of the patient
	Photo []Attachment `json:"photo,omitempty"`
	// A contact party (e.g. guardian, partner, friend) for the patient
	Contact []PatientContact `json:"contact,omitempty"`
	// A language which may be used to communicate with the patient about his or her health
	Communication []PatientCommunication `json:"communication,omitempty"`
	// Patient's nominated primary care provider
	GeneralPractitioner []Reference `json:"generalPractitioner,omitempty"`
	// Organization that is the custodian of the patient record
	ManagingOrganization *Reference `json:"managingOrganization,omitempty"`
	// Link to another patient resource that concerns the same actual person
	Link []PatientLink `json:"link,omitempty"`
}
