package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeStructureDefinition is the FHIR resource type name for StructureDefinition.
const ResourceTypeStructureDefinition = "StructureDefinition"

// StructureDefinitionMapping represents a FHIR BackboneElement for StructureDefinition.mapping.
type StructureDefinitionMapping struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Internal id when this mapping is used
	Identity string `json:"identity"`
	// Identifies what this mapping refers to
	URI *string `json:"uri,omitempty"`
	// Names what this mapping refers to
	Name *string `json:"name,omitempty"`
	// Versions, Issues, Scope limitations etc.
	Comment *string `json:"comment,omitempty"`
}

// StructureDefinitionContext represents a FHIR BackboneElement for StructureDefinition.context.
type StructureDefinitionContext struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// fhirpath | element | extension
	Type string `json:"type"`
	// Where the extension can be used in instances
	Expression string `json:"expression"`
}

// StructureDefinitionSnapshot represents a FHIR BackboneElement for StructureDefinition.snapshot.
type StructureDefinitionSnapshot struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Definition of elements in the resource (if no StructureDefinition)
	Element []ElementDefinition `json:"element,omitempty"`
}

// StructureDefinitionDifferential represents a FHIR BackboneElement for StructureDefinition.differential.
type StructureDefinitionDifferential struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Definition of elements in the resource (if no StructureDefinition)
	Element []ElementDefinition `json:"element,omitempty"`
}

// StructureDefinition represents a FHIR StructureDefinition.
type StructureDefinition struct {
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
	// Canonical identifier for this structure definition, represented as a URI (globally unique)
	URL string `json:"url"`
	// Additional identifier for the structure definition
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the structure definition
	Version *string `json:"version,omitempty"`
	// Name for this structure definition (computer friendly)
	Name string `json:"name"`
	// Name for this structure definition (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the structure definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for structure definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this structure definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Assist with indexing and finding
	Keyword []Coding `json:"keyword,omitempty"`
	// FHIR Version this StructureDefinition targets
	FhirVersion *string `json:"fhirVersion,omitempty"`
	// External specification that the content is mapped to
	Mapping []StructureDefinitionMapping `json:"mapping,omitempty"`
	// primitive-type | complex-type | resource | logical
	Kind string `json:"kind"`
	// Whether the structure is abstract
	Abstract bool `json:"abstract"`
	// If an extension, where it can be used in instances
	Context []StructureDefinitionContext `json:"context,omitempty"`
	// FHIRPath invariants - when the extension can be used
	ContextInvariant []string `json:"contextInvariant,omitempty"`
	// Type defined or constrained by this structure
	Type string `json:"type"`
	// Definition that this type is constrained/specialized from
	BaseDefinition *string `json:"baseDefinition,omitempty"`
	// specialization | constraint - How relates to base definition
	Derivation *string `json:"derivation,omitempty"`
	// Snapshot view of the structure
	Snapshot *StructureDefinitionSnapshot `json:"snapshot,omitempty"`
	// Differential view of the structure
	Differential *StructureDefinitionDifferential `json:"differential,omitempty"`
}
