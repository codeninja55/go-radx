package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeRelatedPerson is the FHIR resource type name for RelatedPerson.
const ResourceTypeRelatedPerson = "RelatedPerson"

// RelatedPersonCommunication represents a FHIR BackboneElement for RelatedPerson.communication.
type RelatedPersonCommunication struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The language which can be used to communicate with the related person about the patient's health
	Language CodeableConcept `json:"language"`
	// Language preference indicator
	Preferred *bool `json:"preferred,omitempty"`
}

// RelatedPerson represents a FHIR RelatedPerson.
type RelatedPerson struct {
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
	// A human identifier for this person
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this related person's record is in active use
	Active *bool `json:"active,omitempty"`
	// The patient this person is related to
	Patient Reference `json:"patient"`
	// The relationship of the related person to the patient
	Relationship []CodeableConcept `json:"relationship,omitempty"`
	// A name associated with the person
	Name []HumanName `json:"name,omitempty"`
	// A contact detail for the person
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// male | female | other | unknown
	Gender *string `json:"gender,omitempty"`
	// The date on which the related person was born
	BirthDate *primitives.Date `json:"birthDate,omitempty"`
	// Address where the related person can be contacted or visited
	Address []Address `json:"address,omitempty"`
	// Image of the person
	Photo []Attachment `json:"photo,omitempty"`
	// Period of time that this relationship is considered valid
	Period *Period `json:"period,omitempty"`
	// A language which may be used to communicate with the related person about the patient's health
	Communication []RelatedPersonCommunication `json:"communication,omitempty"`
}
