package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeOperationDefinition is the FHIR resource type name for OperationDefinition.
const ResourceTypeOperationDefinition = "OperationDefinition"

// OperationDefinitionParameterBinding represents a FHIR BackboneElement for OperationDefinition.parameter.binding.
type OperationDefinitionParameterBinding struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// required | extensible | preferred | example
	Strength string `json:"strength"`
	// Source of value set
	ValueSet string `json:"valueSet"`
}

// OperationDefinitionParameterReferencedFrom represents a FHIR BackboneElement for OperationDefinition.parameter.referencedFrom.
type OperationDefinitionParameterReferencedFrom struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Referencing parameter
	Source string `json:"source"`
	// Element id of reference
	SourceId *string `json:"sourceId,omitempty"`
}

// OperationDefinitionParameterPart represents a FHIR BackboneElement for OperationDefinition.parameter.part.
type OperationDefinitionParameterPart struct {
}

// OperationDefinitionParameter represents a FHIR BackboneElement for OperationDefinition.parameter.
type OperationDefinitionParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name in Parameters.parameter.name or in URL
	Name string `json:"name"`
	// in | out
	Use string `json:"use"`
	// instance | type | system
	Scope []string `json:"scope,omitempty"`
	// Minimum Cardinality
	Min int `json:"min"`
	// Maximum Cardinality (a number or *)
	Max string `json:"max"`
	// Description of meaning/use
	Documentation *string `json:"documentation,omitempty"`
	// What type this parameter has
	Type *string `json:"type,omitempty"`
	// Allowed sub-type this parameter can have (if type is abstract)
	AllowedType []string `json:"allowedType,omitempty"`
	// If type is Reference | canonical, allowed targets. If type is 'Resource', then this constrains the allowed resource types
	TargetProfile []string `json:"targetProfile,omitempty"`
	// number | date | string | token | reference | composite | quantity | uri | special
	SearchType *string `json:"searchType,omitempty"`
	// ValueSet details if this is coded
	Binding *OperationDefinitionParameterBinding `json:"binding,omitempty"`
	// References to this parameter
	ReferencedFrom []OperationDefinitionParameterReferencedFrom `json:"referencedFrom,omitempty"`
	// Parts of a nested Parameter
	Part []OperationDefinitionParameterPart `json:"part,omitempty"`
}

// OperationDefinitionOverload represents a FHIR BackboneElement for OperationDefinition.overload.
type OperationDefinitionOverload struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of parameter to include in overload
	ParameterName []string `json:"parameterName,omitempty"`
	// Comments to go on overload
	Comment *string `json:"comment,omitempty"`
}

// OperationDefinition represents a FHIR OperationDefinition.
type OperationDefinition struct {
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
	// Canonical identifier for this operation definition, represented as an absolute URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the implementation guide (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the operation definition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this operation definition (computer friendly)
	Name string `json:"name"`
	// Name for this operation definition (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// operation | query
	Kind string `json:"kind"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the operation definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for operation definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this operation definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// Whether content is changed by the operation
	AffectsState *bool `json:"affectsState,omitempty"`
	// Recommended name for operation in search url
	Code string `json:"code"`
	// Additional information about use
	Comment *string `json:"comment,omitempty"`
	// Marks this as a profile of the base
	Base *string `json:"base,omitempty"`
	// Types this operation applies to
	Resource []string `json:"resource,omitempty"`
	// Invoke at the system level?
	System bool `json:"system"`
	// Invoke at the type level?
	Type bool `json:"type"`
	// Invoke on an instance?
	Instance bool `json:"instance"`
	// Validation information for in parameters
	InputProfile *string `json:"inputProfile,omitempty"`
	// Validation information for out parameters
	OutputProfile *string `json:"outputProfile,omitempty"`
	// Parameters for the operation/query
	Parameter []OperationDefinitionParameter `json:"parameter,omitempty"`
	// Define overloaded variants for when  generating code
	Overload []OperationDefinitionOverload `json:"overload,omitempty"`
}
