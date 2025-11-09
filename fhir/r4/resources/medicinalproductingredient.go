package resources

// ResourceTypeMedicinalProductIngredient is the FHIR resource type name for MedicinalProductIngredient.
const ResourceTypeMedicinalProductIngredient = "MedicinalProductIngredient"

// MedicinalProductIngredientSpecifiedSubstanceStrengthReferenceStrength represents a FHIR BackboneElement for MedicinalProductIngredient.specifiedSubstance.strength.referenceStrength.
type MedicinalProductIngredientSpecifiedSubstanceStrengthReferenceStrength struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Relevant reference substance
	Substance *CodeableConcept `json:"substance,omitempty"`
	// Strength expressed in terms of a reference substance
	Strength Ratio `json:"strength"`
	// Strength expressed in terms of a reference substance
	StrengthLowLimit *Ratio `json:"strengthLowLimit,omitempty"`
	// For when strength is measured at a particular point or distance
	MeasurementPoint *string `json:"measurementPoint,omitempty"`
	// The country or countries for which the strength range applies
	Country []CodeableConcept `json:"country,omitempty"`
}

// MedicinalProductIngredientSpecifiedSubstanceStrength represents a FHIR BackboneElement for MedicinalProductIngredient.specifiedSubstance.strength.
type MedicinalProductIngredientSpecifiedSubstanceStrength struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The quantity of substance in the unit of presentation, or in the volume (or mass) of the single pharmaceutical product or manufactured item
	Presentation Ratio `json:"presentation"`
	// A lower limit for the quantity of substance in the unit of presentation. For use when there is a range of strengths, this is the lower limit, with the presentation attribute becoming the upper limit
	PresentationLowLimit *Ratio `json:"presentationLowLimit,omitempty"`
	// The strength per unitary volume (or mass)
	Concentration *Ratio `json:"concentration,omitempty"`
	// A lower limit for the strength per unitary volume (or mass), for when there is a range. The concentration attribute then becomes the upper limit
	ConcentrationLowLimit *Ratio `json:"concentrationLowLimit,omitempty"`
	// For when strength is measured at a particular point or distance
	MeasurementPoint *string `json:"measurementPoint,omitempty"`
	// The country or countries for which the strength range applies
	Country []CodeableConcept `json:"country,omitempty"`
	// Strength expressed in terms of a reference substance
	ReferenceStrength []MedicinalProductIngredientSpecifiedSubstanceStrengthReferenceStrength `json:"referenceStrength,omitempty"`
}

// MedicinalProductIngredientSpecifiedSubstance represents a FHIR BackboneElement for MedicinalProductIngredient.specifiedSubstance.
type MedicinalProductIngredientSpecifiedSubstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The specified substance
	Code CodeableConcept `json:"code"`
	// The group of specified substance, e.g. group 1 to 4
	Group CodeableConcept `json:"group"`
	// Confidentiality level of the specified substance as the ingredient
	Confidentiality *CodeableConcept `json:"confidentiality,omitempty"`
	// Quantity of the substance or specified substance present in the manufactured item or pharmaceutical product
	Strength []MedicinalProductIngredientSpecifiedSubstanceStrength `json:"strength,omitempty"`
}

// MedicinalProductIngredientSubstanceStrength represents a FHIR BackboneElement for MedicinalProductIngredient.substance.strength.
type MedicinalProductIngredientSubstanceStrength struct {
}

// MedicinalProductIngredientSubstance represents a FHIR BackboneElement for MedicinalProductIngredient.substance.
type MedicinalProductIngredientSubstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The ingredient substance
	Code CodeableConcept `json:"code"`
	// Quantity of the substance or specified substance present in the manufactured item or pharmaceutical product
	Strength []MedicinalProductIngredientSubstanceStrength `json:"strength,omitempty"`
}

// MedicinalProductIngredient represents a FHIR MedicinalProductIngredient.
type MedicinalProductIngredient struct {
	// Logical id of this artifact
	ID *string `json:"id,omitempty"`
	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty"`
	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty"`
	// Language of the resource content
	Language *string `json:"language,omitempty"`
	// Text summary of the resource, for human interpretation
	Text *Narrative `json:"text,omitempty"`
	// Contained, inline Resources
	Contained []any `json:"contained,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifier for the ingredient
	Identifier *Identifier `json:"identifier,omitempty"`
	// Ingredient role e.g. Active ingredient, excipient
	Role CodeableConcept `json:"role"`
	// If the ingredient is a known or suspected allergen
	AllergenicIndicator *bool `json:"allergenicIndicator,omitempty"`
	// Manufacturer of this Ingredient
	Manufacturer []Reference `json:"manufacturer,omitempty"`
	// A specified substance that comprises this ingredient
	SpecifiedSubstance []MedicinalProductIngredientSpecifiedSubstance `json:"specifiedSubstance,omitempty"`
	// The ingredient substance
	Substance *MedicinalProductIngredientSubstance `json:"substance,omitempty"`
}
