package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypePerson is the FHIR resource type name for Person.
const ResourceTypePerson = "Person"

// PersonCommunication represents a FHIR BackboneElement for Person.communication.
type PersonCommunication struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The language which can be used to communicate with the person about his or her health
	Language CodeableConcept `json:"language"`
	// Language preference indicator
	Preferred *bool `json:"preferred,omitempty"`
}

// PersonLink represents a FHIR BackboneElement for Person.link.
type PersonLink struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The resource to which this actual person is associated
	Target Reference `json:"target"`
	// level1 | level2 | level3 | level4
	Assurance *string `json:"assurance,omitempty"`
}

// Person represents a FHIR Person.
type Person struct {
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
	// This person's record is in active use
	Active *bool `json:"active,omitempty"`
	// A name associated with the person
	Name []HumanName `json:"name,omitempty"`
	// A contact detail for the person
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// male | female | other | unknown
	Gender *string `json:"gender,omitempty"`
	// The date on which the person was born
	BirthDate *primitives.Date `json:"birthDate,omitempty"`
	// Indicates if the individual is deceased or not
	Deceased *any `json:"deceased,omitempty"`
	// One or more addresses for the person
	Address []Address `json:"address,omitempty"`
	// Marital (civil) status of a person
	MaritalStatus *CodeableConcept `json:"maritalStatus,omitempty"`
	// Image of the person
	Photo []Attachment `json:"photo,omitempty"`
	// A language which may be used to communicate with the person about his or her health
	Communication []PersonCommunication `json:"communication,omitempty"`
	// The organization that is the custodian of the person record
	ManagingOrganization *Reference `json:"managingOrganization,omitempty"`
	// Link to a resource that concerns the same actual person
	Link []PersonLink `json:"link,omitempty"`
}
