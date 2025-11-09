package resources

// ProdCharacteristic represents a FHIR ProdCharacteristic.
type ProdCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Where applicable, the height can be specified using a numerical value and its unit of measurement The unit of measurement shall be specified in accordance with ISO 11240 and the resulting terminology The symbol and the symbol identifier shall be used
	Height *Quantity `json:"height,omitempty"`
	// Where applicable, the width can be specified using a numerical value and its unit of measurement The unit of measurement shall be specified in accordance with ISO 11240 and the resulting terminology The symbol and the symbol identifier shall be used
	Width *Quantity `json:"width,omitempty"`
	// Where applicable, the depth can be specified using a numerical value and its unit of measurement The unit of measurement shall be specified in accordance with ISO 11240 and the resulting terminology The symbol and the symbol identifier shall be used
	Depth *Quantity `json:"depth,omitempty"`
	// Where applicable, the weight can be specified using a numerical value and its unit of measurement The unit of measurement shall be specified in accordance with ISO 11240 and the resulting terminology The symbol and the symbol identifier shall be used
	Weight *Quantity `json:"weight,omitempty"`
	// Where applicable, the nominal volume can be specified using a numerical value and its unit of measurement The unit of measurement shall be specified in accordance with ISO 11240 and the resulting terminology The symbol and the symbol identifier shall be used
	NominalVolume *Quantity `json:"nominalVolume,omitempty"`
	// Where applicable, the external diameter can be specified using a numerical value and its unit of measurement The unit of measurement shall be specified in accordance with ISO 11240 and the resulting terminology The symbol and the symbol identifier shall be used
	ExternalDiameter *Quantity `json:"externalDiameter,omitempty"`
	// Where applicable, the shape can be specified An appropriate controlled vocabulary shall be used The term and the term identifier shall be used
	Shape *string `json:"shape,omitempty"`
	// Where applicable, the color can be specified An appropriate controlled vocabulary shall be used The term and the term identifier shall be used
	Color []string `json:"color,omitempty"`
	// Where applicable, the imprint can be specified as text
	Imprint []string `json:"imprint,omitempty"`
	// Where applicable, the image can be provided The format of the image attachment shall be specified by regional implementations
	Image []Attachment `json:"image,omitempty"`
	// Where applicable, the scoring can be specified An appropriate controlled vocabulary shall be used The term and the term identifier shall be used
	Scoring *CodeableConcept `json:"scoring,omitempty"`
}
