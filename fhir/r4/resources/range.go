package resources

// Range represents a FHIR Range.
type Range struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Low limit
	Low *Quantity `json:"low,omitempty"`
	// High limit
	High *Quantity `json:"high,omitempty"`
}
