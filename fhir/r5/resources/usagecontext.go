package resources

// UsageContext represents a FHIR UsageContext.
type UsageContext struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Type of context being specified
	Code Coding `json:"code"`
	// Value that defines the context
	Value any `json:"value"`
}
