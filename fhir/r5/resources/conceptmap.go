package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeConceptMap is the FHIR resource type name for ConceptMap.
const ResourceTypeConceptMap = "ConceptMap"

// ConceptMapProperty represents a FHIR BackboneElement for ConceptMap.property.
type ConceptMapProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifies the property on the mappings, and when referred to in the $translate operation
	Code string `json:"code"`
	// Formal identifier for the property
	URI *string `json:"uri,omitempty"`
	// Why the property is defined, and/or what it conveys
	Description *string `json:"description,omitempty"`
	// Coding | string | integer | boolean | dateTime | decimal | code
	Type string `json:"type"`
	// The CodeSystem from which code values come
	System *string `json:"system,omitempty"`
}

// ConceptMapAdditionalAttribute represents a FHIR BackboneElement for ConceptMap.additionalAttribute.
type ConceptMapAdditionalAttribute struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifies this additional attribute through this resource
	Code string `json:"code"`
	// Formal identifier for the data element referred to in this attribte
	URI *string `json:"uri,omitempty"`
	// Why the additional attribute is defined, and/or what the data element it refers to is
	Description *string `json:"description,omitempty"`
	// code | Coding | string | boolean | Quantity
	Type string `json:"type"`
}

// ConceptMapGroupElementTargetProperty represents a FHIR BackboneElement for ConceptMap.group.element.target.property.
type ConceptMapGroupElementTargetProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to ConceptMap.property.code
	Code string `json:"code"`
	// Value of the property for this concept
	Value any `json:"value"`
}

// ConceptMapGroupElementTargetDependsOn represents a FHIR BackboneElement for ConceptMap.group.element.target.dependsOn.
type ConceptMapGroupElementTargetDependsOn struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A reference to a mapping attribute defined in ConceptMap.additionalAttribute
	Attribute string `json:"attribute"`
	// Value of the referenced data element
	Value *any `json:"value,omitempty"`
	// The mapping depends on a data element with a value from this value set
	ValueSet *string `json:"valueSet,omitempty"`
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
	// Identifies the set of target concepts
	ValueSet *string `json:"valueSet,omitempty"`
	// related-to | equivalent | source-is-narrower-than-target | source-is-broader-than-target | not-related-to
	Relationship string `json:"relationship"`
	// Description of status/issues in mapping
	Comment *string `json:"comment,omitempty"`
	// Property value for the source -> target mapping
	Property []ConceptMapGroupElementTargetProperty `json:"property,omitempty"`
	// Other properties required for this mapping
	DependsOn []ConceptMapGroupElementTargetDependsOn `json:"dependsOn,omitempty"`
	// Other data elements that this mapping also produces
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
	// Identifies the set of concepts being mapped
	ValueSet *string `json:"valueSet,omitempty"`
	// No mapping to a target concept for this source concept
	NoMap *bool `json:"noMap,omitempty"`
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
	// use-source-code | fixed | other-map
	Mode string `json:"mode"`
	// Fixed code when mode = fixed
	Code *string `json:"code,omitempty"`
	// Display for the code
	Display *string `json:"display,omitempty"`
	// Fixed code set when mode = fixed
	ValueSet *string `json:"valueSet,omitempty"`
	// related-to | equivalent | source-is-narrower-than-target | source-is-broader-than-target | not-related-to
	Relationship *string `json:"relationship,omitempty"`
	// canonical reference to an additional ConceptMap to use for mapping if the source concept is unmapped
	OtherMap *string `json:"otherMap,omitempty"`
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
	// Target system that the concepts are to be mapped to
	Target *string `json:"target,omitempty"`
	// Mappings for a concept from the source set
	Element []ConceptMapGroupElement `json:"element,omitempty"`
	// What to do when there is no mapping target for the source concept and ConceptMap.group.element.noMap is not true
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
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the concept map
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
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
	// Name of the publisher/steward (organization or individual)
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
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the ConceptMap was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the ConceptMap was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the ConceptMap is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// E.g. Education, Treatment, Assessment, etc
	Topic []CodeableConcept `json:"topic,omitempty"`
	// Who authored the ConceptMap
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the ConceptMap
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the ConceptMap
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the ConceptMap
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations, etc
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Additional properties of the mapping
	Property []ConceptMapProperty `json:"property,omitempty"`
	// Definition of an additional attribute to act as a data source or target
	AdditionalAttribute []ConceptMapAdditionalAttribute `json:"additionalAttribute,omitempty"`
	// The source value set that contains the concepts that are being mapped
	SourceScope *any `json:"sourceScope,omitempty"`
	// The target value set which provides context for the mappings
	TargetScope *any `json:"targetScope,omitempty"`
	// Same source and target systems
	Group []ConceptMapGroup `json:"group,omitempty"`
}
