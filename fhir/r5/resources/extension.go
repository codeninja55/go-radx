package resources

// Extension represents a FHIR Extension.
type Extension struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// identifies the meaning of the extension
	URL string `json:"url"`
	// Value of extension
	Value *any `json:"value,omitempty"`
}
