package resources

// SubstanceAmountReferenceRange represents a FHIR BackboneElement for SubstanceAmount.referenceRange.
type SubstanceAmountReferenceRange struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Lower limit possible or expected
	LowLimit *Quantity `json:"lowLimit,omitempty"`
	// Upper limit possible or expected
	HighLimit *Quantity `json:"highLimit,omitempty"`
}

// SubstanceAmount represents a FHIR SubstanceAmount.
type SubstanceAmount struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Used to capture quantitative values for a variety of elements. If only limits are given, the arithmetic mean would be the average. If only a single definite value for a given element is given, it would be captured in this field
	Amount *any `json:"amount,omitempty"`
	// Most elements that require a quantitative value will also have a field called amount type. Amount type should always be specified because the actual value of the amount is often dependent on it. EXAMPLE: In capturing the actual relative amounts of substances or molecular fragments it is essential to indicate whether the amount refers to a mole ratio or weight ratio. For any given element an effort should be made to use same the amount type for all related definitional elements
	AmountType *CodeableConcept `json:"amountType,omitempty"`
	// A textual comment on a numeric value
	AmountText *string `json:"amountText,omitempty"`
	// Reference range of possible or expected values
	ReferenceRange *SubstanceAmountReferenceRange `json:"referenceRange,omitempty"`
}
