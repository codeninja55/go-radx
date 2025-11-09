package resources

// ParameterDefinition represents a FHIR ParameterDefinition.
type ParameterDefinition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Name used to access the parameter value
	Name *string `json:"name,omitempty"`
	// in | out
	Use string `json:"use"`
	// Minimum cardinality
	Min *int `json:"min,omitempty"`
	// Maximum cardinality (a number of *)
	Max *string `json:"max,omitempty"`
	// A brief description of the parameter
	Documentation *string `json:"documentation,omitempty"`
	// What type of value
	Type string `json:"type"`
	// What profile the value is expected to be
	Profile *string `json:"profile,omitempty"`
}
