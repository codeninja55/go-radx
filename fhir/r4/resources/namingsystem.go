package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeNamingSystem is the FHIR resource type name for NamingSystem.
const ResourceTypeNamingSystem = "NamingSystem"

// NamingSystemUniqueId represents a FHIR BackboneElement for NamingSystem.uniqueId.
type NamingSystemUniqueId struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// oid | uuid | uri | other
	Type string `json:"type"`
	// The unique identifier
	Value string `json:"value"`
	// Is this the id that should be used for this type
	Preferred *bool `json:"preferred,omitempty"`
	// Notes about identifier usage
	Comment *string `json:"comment,omitempty"`
	// When is identifier valid?
	Period *Period `json:"period,omitempty"`
}

// NamingSystem represents a FHIR NamingSystem.
type NamingSystem struct {
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
	// Name for this naming system (computer friendly)
	Name string `json:"name"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// codesystem | identifier | root
	Kind string `json:"kind"`
	// Date last changed
	Date primitives.DateTime `json:"date"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Who maintains system namespace?
	Responsible *string `json:"responsible,omitempty"`
	// e.g. driver,  provider,  patient, bank etc.
	Type *CodeableConcept `json:"type,omitempty"`
	// Natural language description of the naming system
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for naming system (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// How/where is it used
	Usage *string `json:"usage,omitempty"`
	// Unique identifiers used for system
	UniqueId []NamingSystemUniqueId `json:"uniqueId,omitempty"`
}
