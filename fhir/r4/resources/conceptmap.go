package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeConceptMap is the FHIR resource type name for ConceptMap.
const ResourceTypeConceptMap = "ConceptMap"

// ConceptMapGroupElementTargetDependsOn represents a FHIR BackboneElement for ConceptMap.group.element.target.dependsOn.
type ConceptMapGroupElementTargetDependsOn struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to property mapping depends on
	Property string `json:"property"`
	// Code System (if necessary)
	System *string `json:"system,omitempty"`
	// Value of the referenced element
	Value string `json:"value"`
	// Display for the code (if value is a code)
	Display *string `json:"display,omitempty"`
}

// ConceptMapGroupElementTargetProduct represents a FHIR BackboneElement for ConceptMap.group.element.target.product.
type ConceptMapGroupElementTargetProduct struct {
}

// ConceptMapGroupElementTarget represents a FHIR BackboneElement for ConceptMap.group.element.target.
type ConceptMapGroupElementTarget struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code that identifies the target element
	Code *string `json:"code,omitempty"`
	// Display for the code
	Display *string `json:"display,omitempty"`
	// relatedto | equivalent | equal | wider | subsumes | narrower | specializes | inexact | unmatched | disjoint
	Equivalence string `json:"equivalence"`
	// Description of status/issues in mapping
	Comment *string `json:"comment,omitempty"`
	// Other elements required for this mapping (from context)
	DependsOn []ConceptMapGroupElementTargetDependsOn `json:"dependsOn,omitempty"`
	// Other concepts that this mapping also produces
	Product []ConceptMapGroupElementTargetProduct `json:"product,omitempty"`
}

// ConceptMapGroupElement represents a FHIR BackboneElement for ConceptMap.group.element.
type ConceptMapGroupElement struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifies element being mapped
	Code *string `json:"code,omitempty"`
	// Display for the code
	Display *string `json:"display,omitempty"`
	// Concept in target system for element
	Target []ConceptMapGroupElementTarget `json:"target,omitempty"`
}

// ConceptMapGroupUnmapped represents a FHIR BackboneElement for ConceptMap.group.unmapped.
type ConceptMapGroupUnmapped struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// provided | fixed | other-map
	Mode string `json:"mode"`
	// Fixed code when mode = fixed
	Code *string `json:"code,omitempty"`
	// Display for the code
	Display *string `json:"display,omitempty"`
	// canonical reference to an additional ConceptMap to use for mapping if the source concept is unmapped
	URL *string `json:"url,omitempty"`
}

// ConceptMapGroup represents a FHIR BackboneElement for ConceptMap.group.
type ConceptMapGroup struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Source system where concepts to be mapped are defined
	Source *string `json:"source,omitempty"`
	// Specific version of the  code system
	SourceVersion *string `json:"sourceVersion,omitempty"`
	// Target system that the concepts are to be mapped to
	Target *string `json:"target,omitempty"`
	// Specific version of the  code system
	TargetVersion *string `json:"targetVersion,omitempty"`
	// Mappings for a concept from the source set
	Element []ConceptMapGroupElement `json:"element,omitempty"`
	// What to do when there is no mapping for the source concept
	Unmapped *ConceptMapGroupUnmapped `json:"unmapped,omitempty"`
}

// ConceptMap represents a FHIR ConceptMap.
type ConceptMap struct {
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
	// Canonical identifier for this concept map, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the concept map
	Identifier *Identifier `json:"identifier,omitempty"`
	// Business version of the concept map
	Version *string `json:"version,omitempty"`
	// Name for this concept map (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this concept map (human friendly)
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
	// Natural language description of the concept map
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for concept map (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this concept map is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// The source value set that contains the concepts that are being mapped
	Source *any `json:"source,omitempty"`
	// The target value set which provides context for the mappings
	Target *any `json:"target,omitempty"`
	// Same source and target systems
	Group []ConceptMapGroup `json:"group,omitempty"`
}
