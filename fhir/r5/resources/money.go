package resources

// Money represents a FHIR Money.
type Money struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Numerical value (with implicit precision)
	Value *float64 `json:"value,omitempty"`
	// ISO 4217 Currency Code
	Currency *string `json:"currency,omitempty"`
}
