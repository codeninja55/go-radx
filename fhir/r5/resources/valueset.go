package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeValueSet is the FHIR resource type name for ValueSet.
const ResourceTypeValueSet = "ValueSet"

// ValueSetComposeIncludeConceptDesignation represents a FHIR BackboneElement for ValueSet.compose.include.concept.designation.
type ValueSetComposeIncludeConceptDesignation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Human language of the designation
	Language *string `json:"language,omitempty"`
	// Types of uses of designations
	Use *Coding `json:"use,omitempty"`
	// Additional ways how this designation would be used
	AdditionalUse []Coding `json:"additionalUse,omitempty"`
	// The text value for this designation
	Value string `json:"value"`
}

// ValueSetComposeIncludeConcept represents a FHIR BackboneElement for ValueSet.compose.include.concept.
type ValueSetComposeIncludeConcept struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code or expression from system
	Code string `json:"code"`
	// Text to display for this code for this value set in this valueset
	Display *string `json:"display,omitempty"`
	// Additional representations for this concept
	Designation []ValueSetComposeIncludeConceptDesignation `json:"designation,omitempty"`
}

// ValueSetComposeIncludeFilter represents a FHIR BackboneElement for ValueSet.compose.include.filter.
type ValueSetComposeIncludeFilter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A property/filter defined by the code system
	Property string `json:"property"`
	// = | is-a | descendent-of | is-not-a | regex | in | not-in | generalizes | child-of | descendent-leaf | exists
	Op string `json:"op"`
	// Code from the system, or regex criteria, or boolean value for exists
	Value string `json:"value"`
}

// ValueSetComposeInclude represents a FHIR BackboneElement for ValueSet.compose.include.
type ValueSetComposeInclude struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The system the codes come from
	System *string `json:"system,omitempty"`
	// Specific version of the code system referred to
	Version *string `json:"version,omitempty"`
	// A concept defined in the system
	Concept []ValueSetComposeIncludeConcept `json:"concept,omitempty"`
	// Select codes/concepts by their properties (including relationships)
	Filter []ValueSetComposeIncludeFilter `json:"filter,omitempty"`
	// Select the contents included in this value set
	ValueSet []string `json:"valueSet,omitempty"`
	// A copyright statement for the specific code system included in the value set
	Copyright *string `json:"copyright,omitempty"`
}

// ValueSetComposeExclude represents a FHIR BackboneElement for ValueSet.compose.exclude.
type ValueSetComposeExclude struct {
}

// ValueSetCompose represents a FHIR BackboneElement for ValueSet.compose.
type ValueSetCompose struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Fixed date for references with no specified version (transitive)
	LockedDate *primitives.Date `json:"lockedDate,omitempty"`
	// Whether inactive codes are in the value set
	Inactive *bool `json:"inactive,omitempty"`
	// Include one or more codes from a code system or other value set(s)
	Include []ValueSetComposeInclude `json:"include,omitempty"`
	// Explicitly exclude codes from a code system or other value sets
	Exclude []ValueSetComposeExclude `json:"exclude,omitempty"`
	// Property to return if client doesn't override
	Property []string `json:"property,omitempty"`
}

// ValueSetExpansionParameter represents a FHIR BackboneElement for ValueSet.expansion.parameter.
type ValueSetExpansionParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name as assigned by the client or server
	Name string `json:"name"`
	// Value of the named parameter
	Value *any `json:"value,omitempty"`
}

// ValueSetExpansionProperty represents a FHIR BackboneElement for ValueSet.expansion.property.
type ValueSetExpansionProperty struct {
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
}

// ValueSetExpansionContainsDesignation represents a FHIR BackboneElement for ValueSet.expansion.contains.designation.
type ValueSetExpansionContainsDesignation struct {
}

// ValueSetExpansionContainsPropertySubProperty represents a FHIR BackboneElement for ValueSet.expansion.contains.property.subProperty.
type ValueSetExpansionContainsPropertySubProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to ValueSet.expansion.property.code
	Code string `json:"code"`
	// Value of the subproperty for this concept
	Value any `json:"value"`
}

// ValueSetExpansionContainsProperty represents a FHIR BackboneElement for ValueSet.expansion.contains.property.
type ValueSetExpansionContainsProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to ValueSet.expansion.property.code
	Code string `json:"code"`
	// Value of the property for this concept
	Value any `json:"value"`
	// SubProperty value for the concept
	SubProperty []ValueSetExpansionContainsPropertySubProperty `json:"subProperty,omitempty"`
}

// ValueSetExpansionContainsContains represents a FHIR BackboneElement for ValueSet.expansion.contains.contains.
type ValueSetExpansionContainsContains struct {
}

// ValueSetExpansionContains represents a FHIR BackboneElement for ValueSet.expansion.contains.
type ValueSetExpansionContains struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// System value for the code
	System *string `json:"system,omitempty"`
	// If user cannot select this entry
	Abstract *bool `json:"abstract,omitempty"`
	// If concept is inactive in the code system
	Inactive *bool `json:"inactive,omitempty"`
	// Version in which this code/display is defined
	Version *string `json:"version,omitempty"`
	// Code - if blank, this is not a selectable code
	Code *string `json:"code,omitempty"`
	// User display for the concept
	Display *string `json:"display,omitempty"`
	// Additional representations for this item
	Designation []ValueSetExpansionContainsDesignation `json:"designation,omitempty"`
	// Property value for the concept
	Property []ValueSetExpansionContainsProperty `json:"property,omitempty"`
	// Codes contained under this entry
	Contains []ValueSetExpansionContainsContains `json:"contains,omitempty"`
}

// ValueSetExpansion represents a FHIR BackboneElement for ValueSet.expansion.
type ValueSetExpansion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifies the value set expansion (business identifier)
	Identifier *string `json:"identifier,omitempty"`
	// Opaque urls for paging through expansion results
	Next *string `json:"next,omitempty"`
	// Time ValueSet expansion happened
	Timestamp primitives.DateTime `json:"timestamp"`
	// Total number of codes in the expansion
	Total *int `json:"total,omitempty"`
	// Offset at which this resource starts
	Offset *int `json:"offset,omitempty"`
	// Parameter that controlled the expansion process
	Parameter []ValueSetExpansionParameter `json:"parameter,omitempty"`
	// Additional information supplied about each concept
	Property []ValueSetExpansionProperty `json:"property,omitempty"`
	// Codes in the value set
	Contains []ValueSetExpansionContains `json:"contains,omitempty"`
}

// ValueSetScope represents a FHIR BackboneElement for ValueSet.scope.
type ValueSetScope struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Criteria describing which concepts or codes should be included and why
	InclusionCriteria *string `json:"inclusionCriteria,omitempty"`
	// Criteria describing which concepts or codes should be excluded and why
	ExclusionCriteria *string `json:"exclusionCriteria,omitempty"`
}

// ValueSet represents a FHIR ValueSet.
type ValueSet struct {
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
	// Canonical identifier for this value set, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the value set (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the value set
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this value set (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this value set (human friendly)
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
	// Natural language description of the value set
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for value set (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Indicates whether or not any change to the content logical definition may occur
	Immutable *bool `json:"immutable,omitempty"`
	// Why this value set is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the ValueSet was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the ValueSet was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the ValueSet is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// E.g. Education, Treatment, Assessment, etc
	Topic []CodeableConcept `json:"topic,omitempty"`
	// Who authored the ValueSet
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the ValueSet
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the ValueSet
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the ValueSet
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations, etc
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Content logical definition of the value set (CLD)
	Compose *ValueSetCompose `json:"compose,omitempty"`
	// Used when the value set is "expanded"
	Expansion *ValueSetExpansion `json:"expansion,omitempty"`
	// Description of the semantic space the Value Set Expansion is intended to cover and should further clarify the text in ValueSet.description
	Scope *ValueSetScope `json:"scope,omitempty"`
}
