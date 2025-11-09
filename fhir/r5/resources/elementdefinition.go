package resources

// ElementDefinitionSlicingDiscriminator represents a FHIR BackboneElement for ElementDefinition.slicing.discriminator.
type ElementDefinitionSlicingDiscriminator struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// value | exists | type | profile | position
	Type string `json:"type"`
	// Path to element value
	Path string `json:"path"`
}

// ElementDefinitionSlicing represents a FHIR BackboneElement for ElementDefinition.slicing.
type ElementDefinitionSlicing struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Element values that are used to distinguish the slices
	Discriminator []ElementDefinitionSlicingDiscriminator `json:"discriminator,omitempty"`
	// Text description of how slicing works (or not)
	Description *string `json:"description,omitempty"`
	// If elements must be in same order as slices
	Ordered *bool `json:"ordered,omitempty"`
	// closed | open | openAtEnd
	Rules string `json:"rules"`
}

// ElementDefinitionBase represents a FHIR BackboneElement for ElementDefinition.base.
type ElementDefinitionBase struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Path that identifies the base element
	Path string `json:"path"`
	// Min cardinality of the base element
	Min uint `json:"min"`
	// Max cardinality of the base element
	Max string `json:"max"`
}

// ElementDefinitionType represents a FHIR BackboneElement for ElementDefinition.type.
type ElementDefinitionType struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Data type or Resource (reference to definition)
	Code string `json:"code"`
	// Profiles (StructureDefinition or IG) - one must apply
	Profile []string `json:"profile,omitempty"`
	// Profile (StructureDefinition or IG) on the Reference/canonical target - one must apply
	TargetProfile []string `json:"targetProfile,omitempty"`
	// contained | referenced | bundled - how aggregated
	Aggregation []string `json:"aggregation,omitempty"`
	// either | independent | specific
	Versioning *string `json:"versioning,omitempty"`
}

// ElementDefinitionExample represents a FHIR BackboneElement for ElementDefinition.example.
type ElementDefinitionExample struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Describes the purpose of this example
	Label string `json:"label"`
	// Value of Example (one of allowed types)
	Value any `json:"value"`
}

// ElementDefinitionConstraint represents a FHIR BackboneElement for ElementDefinition.constraint.
type ElementDefinitionConstraint struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Target of 'condition' reference above
	Key string `json:"key"`
	// Why this constraint is necessary or appropriate
	Requirements *string `json:"requirements,omitempty"`
	// error | warning
	Severity string `json:"severity"`
	// Suppress warning or hint in profile
	Suppress *bool `json:"suppress,omitempty"`
	// Human description of constraint
	Human string `json:"human"`
	// FHIRPath expression of constraint
	Expression *string `json:"expression,omitempty"`
	// Reference to original source of constraint
	Source *string `json:"source,omitempty"`
}

// ElementDefinitionBindingAdditional represents a FHIR BackboneElement for ElementDefinition.binding.additional.
type ElementDefinitionBindingAdditional struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// maximum | minimum | required | extensible | candidate | current | preferred | ui | starter | component
	Purpose string `json:"purpose"`
	// The value set for the additional binding
	ValueSet string `json:"valueSet"`
	// Documentation of the purpose of use of the binding
	Documentation *string `json:"documentation,omitempty"`
	// Concise documentation - for summary tables
	ShortDoco *string `json:"shortDoco,omitempty"`
	// Qualifies the usage - jurisdiction, gender, workflow status etc.
	Usage []UsageContext `json:"usage,omitempty"`
	// Whether binding can applies to all repeats, or just one
	Any *bool `json:"any,omitempty"`
}

// ElementDefinitionBinding represents a FHIR BackboneElement for ElementDefinition.binding.
type ElementDefinitionBinding struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// required | extensible | preferred | example
	Strength string `json:"strength"`
	// Intended use of codes in the bound value set
	Description *string `json:"description,omitempty"`
	// Source of value set
	ValueSet *string `json:"valueSet,omitempty"`
	// Additional Bindings - more rules about the binding
	Additional []ElementDefinitionBindingAdditional `json:"additional,omitempty"`
}

// ElementDefinitionMapping represents a FHIR BackboneElement for ElementDefinition.mapping.
type ElementDefinitionMapping struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Reference to mapping declaration
	Identity string `json:"identity"`
	// Computable language of mapping
	Language *string `json:"language,omitempty"`
	// Details of the mapping
	Map string `json:"map"`
	// Comments about the mapping or its use
	Comment *string `json:"comment,omitempty"`
}

// ElementDefinition represents a FHIR ElementDefinition.
type ElementDefinition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Path of the element in the hierarchy of elements
	Path string `json:"path"`
	// xmlAttr | xmlText | typeAttr | cdaText | xhtml
	Representation []string `json:"representation,omitempty"`
	// Name for this particular element (in a set of slices)
	SliceName *string `json:"sliceName,omitempty"`
	// If this slice definition constrains an inherited slice definition (or not)
	SliceIsConstraining *bool `json:"sliceIsConstraining,omitempty"`
	// Name for element to display with or prompt for element
	Label *string `json:"label,omitempty"`
	// Corresponding codes in terminologies
	Code []Coding `json:"code,omitempty"`
	// This element is sliced - slices follow
	Slicing *ElementDefinitionSlicing `json:"slicing,omitempty"`
	// Concise definition for space-constrained presentation
	Short *string `json:"short,omitempty"`
	// Full formal definition as narrative text
	Definition *string `json:"definition,omitempty"`
	// Comments about the use of this element
	Comment *string `json:"comment,omitempty"`
	// Why this resource has been created
	Requirements *string `json:"requirements,omitempty"`
	// Other names
	Alias []string `json:"alias,omitempty"`
	// Minimum Cardinality
	Min *uint `json:"min,omitempty"`
	// Maximum Cardinality (a number or *)
	Max *string `json:"max,omitempty"`
	// Base definition information for tools
	Base *ElementDefinitionBase `json:"base,omitempty"`
	// Reference to definition of content for the element
	ContentReference *string `json:"contentReference,omitempty"`
	// Data type and Profile for this element
	Type []ElementDefinitionType `json:"type,omitempty"`
	// Specified value if missing from instance
	DefaultValue *any `json:"defaultValue,omitempty"`
	// Implicit meaning when this element is missing
	MeaningWhenMissing *string `json:"meaningWhenMissing,omitempty"`
	// What the order of the elements means
	OrderMeaning *string `json:"orderMeaning,omitempty"`
	// Value must be exactly this
	Fixed *any `json:"fixed,omitempty"`
	// Value must have at least these property values
	Pattern *any `json:"pattern,omitempty"`
	// Example value (as defined for type)
	Example []ElementDefinitionExample `json:"example,omitempty"`
	// Minimum Allowed Value (for some types)
	MinValue *any `json:"minValue,omitempty"`
	// Maximum Allowed Value (for some types)
	MaxValue *any `json:"maxValue,omitempty"`
	// Max length for string type data
	MaxLength *int `json:"maxLength,omitempty"`
	// Reference to invariant about presence
	Condition []string `json:"condition,omitempty"`
	// Condition that must evaluate to true
	Constraint []ElementDefinitionConstraint `json:"constraint,omitempty"`
	// For primitives, that a value must be present - not replaced by an extension
	MustHaveValue *bool `json:"mustHaveValue,omitempty"`
	// Extensions that are allowed to replace a primitive value
	ValueAlternatives []string `json:"valueAlternatives,omitempty"`
	// If the element must be supported (discouraged - see obligations)
	MustSupport *bool `json:"mustSupport,omitempty"`
	// If this modifies the meaning of other elements
	IsModifier *bool `json:"isModifier,omitempty"`
	// Reason that this element is marked as a modifier
	IsModifierReason *string `json:"isModifierReason,omitempty"`
	// Include when _summary = true?
	IsSummary *bool `json:"isSummary,omitempty"`
	// ValueSet details if this is coded
	Binding *ElementDefinitionBinding `json:"binding,omitempty"`
	// Map element to another set of definitions
	Mapping []ElementDefinitionMapping `json:"mapping,omitempty"`
}
