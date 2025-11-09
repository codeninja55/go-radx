package resources

// Ratio represents a FHIR Ratio.
type Ratio struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Numerator value
	Numerator *Quantity `json:"numerator,omitempty"`
	// Denominator value
	Denominator *Quantity `json:"denominator,omitempty"`
}
