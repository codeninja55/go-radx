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
	// oid | uuid | uri | iri-stem | v2csmnemonic | other
	Type string `json:"type"`
	// The unique identifier
	Value string `json:"value"`
	// Is this the id that should be used for this type
	Preferred *bool `json:"preferred,omitempty"`
	// Notes about identifier usage
	Comment *string `json:"comment,omitempty"`
	// When is identifier valid?
	Period *Period `json:"period,omitempty"`
	// Whether the identifier is authoritative
	Authoritative *bool `json:"authoritative,omitempty"`
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
	// Canonical identifier for this naming system, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the naming system (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the naming system
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this naming system (computer friendly)
	Name string `json:"name"`
	// Title for this naming system (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// codesystem | identifier | root
	Kind string `json:"kind"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date primitives.DateTime `json:"date"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Who maintains system namespace?
	Responsible *string `json:"responsible,omitempty"`
	// e.g. driver,  provider,  patient, bank etc
	Type *CodeableConcept `json:"type,omitempty"`
	// Natural language description of the naming system
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for naming system (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this naming system is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the NamingSystem was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the NamingSystem was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the NamingSystem is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// E.g. Education, Treatment, Assessment, etc
	Topic []CodeableConcept `json:"topic,omitempty"`
	// Who authored the CodeSystem
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the NamingSystem
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the NamingSystem
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the NamingSystem
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations, etc
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// How/where is it used
	Usage *string `json:"usage,omitempty"`
	// Unique identifiers used for system
	UniqueId []NamingSystemUniqueId `json:"uniqueId,omitempty"`
}
