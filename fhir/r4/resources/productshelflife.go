package resources

// ProductShelfLife represents a FHIR ProductShelfLife.
type ProductShelfLife struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Unique identifier for the packaged Medicinal Product
	Identifier *Identifier `json:"identifier,omitempty"`
	// This describes the shelf life, taking into account various scenarios such as shelf life of the packaged Medicinal Product itself, shelf life after transformation where necessary and shelf life after the first opening of a bottle, etc. The shelf life type shall be specified using an appropriate controlled vocabulary The controlled term and the controlled term identifier shall be specified
	Type CodeableConcept `json:"type"`
	// The shelf life time period can be specified using a numerical value for the period of time and its unit of time measurement The unit of measurement shall be specified in accordance with ISO 11240 and the resulting terminology The symbol and the symbol identifier shall be used
	Period Quantity `json:"period"`
	// Special precautions for storage, if any, can be specified using an appropriate controlled vocabulary The controlled term and the controlled term identifier shall be specified
	SpecialPrecautionsForStorage []CodeableConcept `json:"specialPrecautionsForStorage,omitempty"`
}
