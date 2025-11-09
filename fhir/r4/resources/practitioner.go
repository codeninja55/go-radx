package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypePractitioner is the FHIR resource type name for Practitioner.
const ResourceTypePractitioner = "Practitioner"

// PractitionerQualification represents a FHIR BackboneElement for Practitioner.qualification.
type PractitionerQualification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// An identifier for this qualification for the practitioner
	Identifier []Identifier `json:"identifier,omitempty"`
	// Coded representation of the qualification
	Code CodeableConcept `json:"code"`
	// Period during which the qualification is valid
	Period *Period `json:"period,omitempty"`
	// Organization that regulates and issues the qualification
	Issuer *Reference `json:"issuer,omitempty"`
}

// Practitioner represents a FHIR Practitioner.
type Practitioner struct {
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
	// An identifier for the person as this agent
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this practitioner's record is in active use
	Active *bool `json:"active,omitempty"`
	// The name(s) associated with the practitioner
	Name []HumanName `json:"name,omitempty"`
	// A contact detail for the practitioner (that apply to all roles)
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Address(es) of the practitioner that are not role specific (typically home address)
	Address []Address `json:"address,omitempty"`
	// male | female | other | unknown
	Gender *string `json:"gender,omitempty"`
	// The date  on which the practitioner was born
	BirthDate *primitives.Date `json:"birthDate,omitempty"`
	// Image of the person
	Photo []Attachment `json:"photo,omitempty"`
	// Certification, licenses, or training pertaining to the provision of care
	Qualification []PractitionerQualification `json:"qualification,omitempty"`
	// A language the practitioner can use in patient communication
	Communication []CodeableConcept `json:"communication,omitempty"`
}
