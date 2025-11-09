package resources

// ResourceTypeIngredient is the FHIR resource type name for Ingredient.
const ResourceTypeIngredient = "Ingredient"

// IngredientManufacturer represents a FHIR BackboneElement for Ingredient.manufacturer.
type IngredientManufacturer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// allowed | possible | actual
	Role *string `json:"role,omitempty"`
	// An organization that manufactures this ingredient
	Manufacturer Reference `json:"manufacturer"`
}

// IngredientSubstanceStrengthReferenceStrength represents a FHIR BackboneElement for Ingredient.substance.strength.referenceStrength.
type IngredientSubstanceStrengthReferenceStrength struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Relevant reference substance
	Substance CodeableReference `json:"substance"`
	// Strength expressed in terms of a reference substance
	Strength any `json:"strength"`
	// When strength is measured at a particular point or distance
	MeasurementPoint *string `json:"measurementPoint,omitempty"`
	// Where the strength range applies
	Country []CodeableConcept `json:"country,omitempty"`
}

// IngredientSubstanceStrength represents a FHIR BackboneElement for Ingredient.substance.strength.
type IngredientSubstanceStrength struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The quantity of substance in the unit of presentation
	Presentation *any `json:"presentation,omitempty"`
	// Text of either the whole presentation strength or a part of it (rest being in Strength.presentation as a ratio)
	TextPresentation *string `json:"textPresentation,omitempty"`
	// The strength per unitary volume (or mass)
	Concentration *any `json:"concentration,omitempty"`
	// Text of either the whole concentration strength or a part of it (rest being in Strength.concentration as a ratio)
	TextConcentration *string `json:"textConcentration,omitempty"`
	// A code that indicates if the strength is, for example, based on the ingredient substance as stated or on the substance base (when the ingredient is a salt)
	Basis *CodeableConcept `json:"basis,omitempty"`
	// When strength is measured at a particular point or distance
	MeasurementPoint *string `json:"measurementPoint,omitempty"`
	// Where the strength range applies
	Country []CodeableConcept `json:"country,omitempty"`
	// Strength expressed in terms of a reference substance
	ReferenceStrength []IngredientSubstanceStrengthReferenceStrength `json:"referenceStrength,omitempty"`
}

// IngredientSubstance represents a FHIR BackboneElement for Ingredient.substance.
type IngredientSubstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A code or full resource that represents the ingredient substance
	Code CodeableReference `json:"code"`
	// The quantity of substance, per presentation, or per volume or mass, and type of quantity
	Strength []IngredientSubstanceStrength `json:"strength,omitempty"`
}

// Ingredient represents a FHIR Ingredient.
type Ingredient struct {
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
	// An identifier or code by which the ingredient can be referenced
	Identifier *Identifier `json:"identifier,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// The product which this ingredient is a constituent part of
	For []Reference `json:"for,omitempty"`
	// Purpose of the ingredient within the product, e.g. active, inactive
	Role CodeableConcept `json:"role"`
	// Precise action within the drug product, e.g. antioxidant, alkalizing agent
	Function []CodeableConcept `json:"function,omitempty"`
	// A classification of the ingredient according to where in the physical item it tends to be used, such the outer shell of a tablet, inner body or ink
	Group *CodeableConcept `json:"group,omitempty"`
	// If the ingredient is a known or suspected allergen
	AllergenicIndicator *bool `json:"allergenicIndicator,omitempty"`
	// A place for providing any notes that are relevant to the component, e.g. removed during process, adjusted for loss on drying
	Comment *string `json:"comment,omitempty"`
	// An organization that manufactures this ingredient
	Manufacturer []IngredientManufacturer `json:"manufacturer,omitempty"`
	// The substance that comprises this ingredient
	Substance IngredientSubstance `json:"substance"`
}
