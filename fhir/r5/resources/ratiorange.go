package resources

// RatioRange represents a FHIR RatioRange.
type RatioRange struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Low Numerator limit
	LowNumerator *Quantity `json:"lowNumerator,omitempty"`
	// High Numerator limit
	HighNumerator *Quantity `json:"highNumerator,omitempty"`
	// Denominator value
	Denominator *Quantity `json:"denominator,omitempty"`
}
