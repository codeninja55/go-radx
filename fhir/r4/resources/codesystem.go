package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCodeSystem is the FHIR resource type name for CodeSystem.
const ResourceTypeCodeSystem = "CodeSystem"

// CodeSystemFilter represents a FHIR BackboneElement for CodeSystem.filter.
type CodeSystemFilter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code that identifies the filter
	Code string `json:"code"`
	// How or why the filter is used
	Description *string `json:"description,omitempty"`
	// = | is-a | descendent-of | is-not-a | regex | in | not-in | generalizes | exists
	Operator []string `json:"operator,omitempty"`
	// What to use for the value
	Value string `json:"value"`
}

// CodeSystemProperty represents a FHIR BackboneElement for CodeSystem.property.
type CodeSystemProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifies the property on the concepts, and when referred to in operations
	Code string `json:"code"`
	// Formal identifier for the property
	URI *string `json:"uri,omitempty"`
	// Why the property is defined, and/or what it conveys
	Description *string `json:"description,omitempty"`
	// code | Coding | string | integer | boolean | dateTime | decimal
	Type string `json:"type"`
}

// CodeSystemConceptDesignation represents a FHIR BackboneElement for CodeSystem.concept.designation.
type CodeSystemConceptDesignation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Human language of the designation
	Language *string `json:"language,omitempty"`
	// Details how this designation would be used
	Use *Coding `json:"use,omitempty"`
	// The text value for this designation
	Value string `json:"value"`
}

// CodeSystemConceptProperty represents a FHIR BackboneElement for CodeSystem.concept.property.
type CodeSystemConceptProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to CodeSystem.property.code
	Code string `json:"code"`
	// Value of the property for this concept
	Value any `json:"value"`
}

// CodeSystemConceptConcept represents a FHIR BackboneElement for CodeSystem.concept.concept.
type CodeSystemConceptConcept struct {
}

// CodeSystemConcept represents a FHIR BackboneElement for CodeSystem.concept.
type CodeSystemConcept struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code that identifies concept
	Code string `json:"code"`
	// Text to display to the user
	Display *string `json:"display,omitempty"`
	// Formal definition
	Definition *string `json:"definition,omitempty"`
	// Additional representations for the concept
	Designation []CodeSystemConceptDesignation `json:"designation,omitempty"`
	// Property value for the concept
	Property []CodeSystemConceptProperty `json:"property,omitempty"`
	// Child Concepts (is-a/contains/categorizes)
	Concept []CodeSystemConceptConcept `json:"concept,omitempty"`
}

// CodeSystem represents a FHIR CodeSystem.
type CodeSystem struct {
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
	// Canonical identifier for this code system, represented as a URI (globally unique) (Coding.system)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the code system (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the code system (Coding.version)
	Version *string `json:"version,omitempty"`
	// Name for this code system (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this code system (human friendly)
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
	// Natural language description of the code system
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for code system (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this code system is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// If code comparison is case sensitive
	CaseSensitive *bool `json:"caseSensitive,omitempty"`
	// Canonical reference to the value set with entire code system
	ValueSet *string `json:"valueSet,omitempty"`
	// grouped-by | is-a | part-of | classified-with
	HierarchyMeaning *string `json:"hierarchyMeaning,omitempty"`
	// If code system defines a compositional grammar
	Compositional *bool `json:"compositional,omitempty"`
	// If definitions are not stable
	VersionNeeded *bool `json:"versionNeeded,omitempty"`
	// not-present | example | fragment | complete | supplement
	Content string `json:"content"`
	// Canonical URL of Code System this adds designations and properties to
	Supplements *string `json:"supplements,omitempty"`
	// Total concepts in the code system
	Count *uint `json:"count,omitempty"`
	// Filter that can be used in a value set
	Filter []CodeSystemFilter `json:"filter,omitempty"`
	// Additional information supplied about each concept
	Property []CodeSystemProperty `json:"property,omitempty"`
	// Concepts in the code system
	Concept []CodeSystemConcept `json:"concept,omitempty"`
}
