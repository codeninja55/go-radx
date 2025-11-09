package resources

// MonetaryComponent represents a FHIR MonetaryComponent.
type MonetaryComponent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// base | surcharge | deduction | discount | tax | informational
	Type string `json:"type"`
	// Codes may be used to differentiate between kinds of taxes, surcharges, discounts etc.
	Code *CodeableConcept `json:"code,omitempty"`
	// Factor used for calculating this component
	Factor *float64 `json:"factor,omitempty"`
	// Explicit value amount to be used
	Amount *Money `json:"amount,omitempty"`
}
