package resources

// Expression represents a FHIR Expression.
type Expression struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Natural language description of the condition
	Description *string `json:"description,omitempty"`
	// Short name assigned to expression for reuse
	Name *string `json:"name,omitempty"`
	// text/cql | text/fhirpath | application/x-fhir-query | etc.
	Language *string `json:"language,omitempty"`
	// Expression in specified language
	Expression *string `json:"expression,omitempty"`
	// Where the expression is found
	Reference *string `json:"reference,omitempty"`
}
