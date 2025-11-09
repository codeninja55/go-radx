package resources

// ResourceTypeParameters is the FHIR resource type name for Parameters.
const ResourceTypeParameters = "Parameters"

// ParametersParameterPart represents a FHIR BackboneElement for Parameters.parameter.part.
type ParametersParameterPart struct {
}

// ParametersParameter represents a FHIR BackboneElement for Parameters.parameter.
type ParametersParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name from the definition
	Name string `json:"name"`
	// If parameter is a data type
	Value *any `json:"value,omitempty"`
	// If parameter is a whole resource
	Resource *any `json:"resource,omitempty"`
	// Named part of a multi-part parameter
	Part []ParametersParameterPart `json:"part,omitempty"`
}

// Parameters represents a FHIR Parameters.
type Parameters struct {
	// Logical id of this artifact
	ID *string `json:"id,omitempty"`
	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty"`
	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty"`
	// Language of the resource content
	Language *string `json:"language,omitempty"`
	// Operation Parameter
	Parameter []ParametersParameter `json:"parameter,omitempty"`
}
