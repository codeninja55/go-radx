package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeStructureMap is the FHIR resource type name for StructureMap.
const ResourceTypeStructureMap = "StructureMap"

// StructureMapStructure represents a FHIR BackboneElement for StructureMap.structure.
type StructureMapStructure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Canonical reference to structure definition
	URL string `json:"url"`
	// source | queried | target | produced
	Mode string `json:"mode"`
	// Name for type in this map
	Alias *string `json:"alias,omitempty"`
	// Documentation on use of structure
	Documentation *string `json:"documentation,omitempty"`
}

// StructureMapGroupInput represents a FHIR BackboneElement for StructureMap.group.input.
type StructureMapGroupInput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name for this instance of data
	Name string `json:"name"`
	// Type for this instance of data
	Type *string `json:"type,omitempty"`
	// source | target
	Mode string `json:"mode"`
	// Documentation for this instance of data
	Documentation *string `json:"documentation,omitempty"`
}

// StructureMapGroupRuleSource represents a FHIR BackboneElement for StructureMap.group.rule.source.
type StructureMapGroupRuleSource struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type or variable this rule applies to
	Context string `json:"context"`
	// Specified minimum cardinality
	Min *int `json:"min,omitempty"`
	// Specified maximum cardinality (number or *)
	Max *string `json:"max,omitempty"`
	// Rule only applies if source has this type
	Type *string `json:"type,omitempty"`
	// Default value if no value exists
	DefaultValue *any `json:"defaultValue,omitempty"`
	// Optional field for this source
	Element *string `json:"element,omitempty"`
	// first | not_first | last | not_last | only_one
	ListMode *string `json:"listMode,omitempty"`
	// Named context for field, if a field is specified
	Variable *string `json:"variable,omitempty"`
	// FHIRPath expression  - must be true or the rule does not apply
	Condition *string `json:"condition,omitempty"`
	// FHIRPath expression  - must be true or the mapping engine throws an error instead of completing
	Check *string `json:"check,omitempty"`
	// Message to put in log if source exists (FHIRPath)
	LogMessage *string `json:"logMessage,omitempty"`
}

// StructureMapGroupRuleTargetParameter represents a FHIR BackboneElement for StructureMap.group.rule.target.parameter.
type StructureMapGroupRuleTargetParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Parameter value - variable or literal
	Value any `json:"value"`
}

// StructureMapGroupRuleTarget represents a FHIR BackboneElement for StructureMap.group.rule.target.
type StructureMapGroupRuleTarget struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type or variable this rule applies to
	Context *string `json:"context,omitempty"`
	// type | variable
	ContextType *string `json:"contextType,omitempty"`
	// Field to create in the context
	Element *string `json:"element,omitempty"`
	// Named context for field, if desired, and a field is specified
	Variable *string `json:"variable,omitempty"`
	// first | share | last | collate
	ListMode []string `json:"listMode,omitempty"`
	// Internal rule reference for shared list items
	ListRuleId *string `json:"listRuleId,omitempty"`
	// create | copy +
	Transform *string `json:"transform,omitempty"`
	// Parameters to the transform
	Parameter []StructureMapGroupRuleTargetParameter `json:"parameter,omitempty"`
}

// StructureMapGroupRuleRule represents a FHIR BackboneElement for StructureMap.group.rule.rule.
type StructureMapGroupRuleRule struct {
}

// StructureMapGroupRuleDependent represents a FHIR BackboneElement for StructureMap.group.rule.dependent.
type StructureMapGroupRuleDependent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of a rule or group to apply
	Name string `json:"name"`
	// Variable to pass to the rule or group
	Variable []string `json:"variable,omitempty"`
}

// StructureMapGroupRule represents a FHIR BackboneElement for StructureMap.group.rule.
type StructureMapGroupRule struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of the rule for internal references
	Name string `json:"name"`
	// Source inputs to the mapping
	Source []StructureMapGroupRuleSource `json:"source,omitempty"`
	// Content to create because of this mapping rule
	Target []StructureMapGroupRuleTarget `json:"target,omitempty"`
	// Rules contained in this rule
	Rule []StructureMapGroupRuleRule `json:"rule,omitempty"`
	// Which other rules to apply in the context of this rule
	Dependent []StructureMapGroupRuleDependent `json:"dependent,omitempty"`
	// Documentation for this instance of data
	Documentation *string `json:"documentation,omitempty"`
}

// StructureMapGroup represents a FHIR BackboneElement for StructureMap.group.
type StructureMapGroup struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Human-readable label
	Name string `json:"name"`
	// Another group that this group adds rules to
	Extends *string `json:"extends,omitempty"`
	// none | types | type-and-types
	TypeMode string `json:"typeMode"`
	// Additional description/explanation for group
	Documentation *string `json:"documentation,omitempty"`
	// Named instance provided when invoking the map
	Input []StructureMapGroupInput `json:"input,omitempty"`
	// Transform Rule from source to target
	Rule []StructureMapGroupRule `json:"rule,omitempty"`
}

// StructureMap represents a FHIR StructureMap.
type StructureMap struct {
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
	// Canonical identifier for this structure map, represented as a URI (globally unique)
	URL string `json:"url"`
	// Additional identifier for the structure map
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the structure map
	Version *string `json:"version,omitempty"`
	// Name for this structure map (computer friendly)
	Name string `json:"name"`
	// Name for this structure map (human friendly)
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
	// Natural language description of the structure map
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for structure map (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this structure map is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Structure Definition used by this map
	Structure []StructureMapStructure `json:"structure,omitempty"`
	// Other maps used by this map (canonical URLs)
	Import []string `json:"import,omitempty"`
	// Named sections for reader convenience
	Group []StructureMapGroup `json:"group,omitempty"`
}
