package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeNutritionOrder is the FHIR resource type name for NutritionOrder.
const ResourceTypeNutritionOrder = "NutritionOrder"

// NutritionOrderOralDietNutrient represents a FHIR BackboneElement for NutritionOrder.oralDiet.nutrient.
type NutritionOrderOralDietNutrient struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of nutrient that is being modified
	Modifier *CodeableConcept `json:"modifier,omitempty"`
	// Quantity of the specified nutrient
	Amount *Quantity `json:"amount,omitempty"`
}

// NutritionOrderOralDietTexture represents a FHIR BackboneElement for NutritionOrder.oralDiet.texture.
type NutritionOrderOralDietTexture struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code to indicate how to alter the texture of the foods, e.g. pureed
	Modifier *CodeableConcept `json:"modifier,omitempty"`
	// Concepts that are used to identify an entity that is ingested for nutritional purposes
	FoodType *CodeableConcept `json:"foodType,omitempty"`
}

// NutritionOrderOralDiet represents a FHIR BackboneElement for NutritionOrder.oralDiet.
type NutritionOrderOralDiet struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of oral diet or diet restrictions that describe what can be consumed orally
	Type []CodeableConcept `json:"type,omitempty"`
	// Scheduled frequency of diet
	Schedule []Timing `json:"schedule,omitempty"`
	// Required  nutrient modifications
	Nutrient []NutritionOrderOralDietNutrient `json:"nutrient,omitempty"`
	// Required  texture modifications
	Texture []NutritionOrderOralDietTexture `json:"texture,omitempty"`
	// The required consistency of fluids and liquids provided to the patient
	FluidConsistencyType []CodeableConcept `json:"fluidConsistencyType,omitempty"`
	// Instructions or additional information about the oral diet
	Instruction *string `json:"instruction,omitempty"`
}

// NutritionOrderSupplement represents a FHIR BackboneElement for NutritionOrder.supplement.
type NutritionOrderSupplement struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of supplement product requested
	Type *CodeableConcept `json:"type,omitempty"`
	// Product or brand name of the nutritional supplement
	ProductName *string `json:"productName,omitempty"`
	// Scheduled frequency of supplement
	Schedule []Timing `json:"schedule,omitempty"`
	// Amount of the nutritional supplement
	Quantity *Quantity `json:"quantity,omitempty"`
	// Instructions or additional information about the oral supplement
	Instruction *string `json:"instruction,omitempty"`
}

// NutritionOrderEnteralFormulaAdministration represents a FHIR BackboneElement for NutritionOrder.enteralFormula.administration.
type NutritionOrderEnteralFormulaAdministration struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Scheduled frequency of enteral feeding
	Schedule *Timing `json:"schedule,omitempty"`
	// The volume of formula to provide
	Quantity *Quantity `json:"quantity,omitempty"`
	// Speed with which the formula is provided per period of time
	Rate *any `json:"rate,omitempty"`
}

// NutritionOrderEnteralFormula represents a FHIR BackboneElement for NutritionOrder.enteralFormula.
type NutritionOrderEnteralFormula struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of enteral or infant formula
	BaseFormulaType *CodeableConcept `json:"baseFormulaType,omitempty"`
	// Product or brand name of the enteral or infant formula
	BaseFormulaProductName *string `json:"baseFormulaProductName,omitempty"`
	// Type of modular component to add to the feeding
	AdditiveType *CodeableConcept `json:"additiveType,omitempty"`
	// Product or brand name of the modular additive
	AdditiveProductName *string `json:"additiveProductName,omitempty"`
	// Amount of energy per specified volume that is required
	CaloricDensity *Quantity `json:"caloricDensity,omitempty"`
	// How the formula should enter the patient's gastrointestinal tract
	RouteofAdministration *CodeableConcept `json:"routeofAdministration,omitempty"`
	// Formula feeding instruction as structured data
	Administration []NutritionOrderEnteralFormulaAdministration `json:"administration,omitempty"`
	// Upper limit on formula volume per unit of time
	MaxVolumeToDeliver *Quantity `json:"maxVolumeToDeliver,omitempty"`
	// Formula feeding instructions expressed as text
	AdministrationInstruction *string `json:"administrationInstruction,omitempty"`
}

// NutritionOrder represents a FHIR NutritionOrder.
type NutritionOrder struct {
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
	// Identifiers assigned to this order
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// Instantiates protocol or definition
	Instantiates []string `json:"instantiates,omitempty"`
	// draft | active | on-hold | revoked | completed | entered-in-error | unknown
	Status string `json:"status"`
	// proposal | plan | directive | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// The person who requires the diet, formula or nutritional supplement
	Patient Reference `json:"patient"`
	// The encounter associated with this nutrition order
	Encounter *Reference `json:"encounter,omitempty"`
	// Date and time the nutrition order was requested
	DateTime primitives.DateTime `json:"dateTime"`
	// Who ordered the diet, formula or nutritional supplement
	Orderer *Reference `json:"orderer,omitempty"`
	// List of the patient's food and nutrition-related allergies and intolerances
	AllergyIntolerance []Reference `json:"allergyIntolerance,omitempty"`
	// Order-specific modifier about the type of food that should be given
	FoodPreferenceModifier []CodeableConcept `json:"foodPreferenceModifier,omitempty"`
	// Order-specific modifier about the type of food that should not be given
	ExcludeFoodModifier []CodeableConcept `json:"excludeFoodModifier,omitempty"`
	// Oral diet components
	OralDiet *NutritionOrderOralDiet `json:"oralDiet,omitempty"`
	// Supplement components
	Supplement []NutritionOrderSupplement `json:"supplement,omitempty"`
	// Enteral formula components
	EnteralFormula *NutritionOrderEnteralFormula `json:"enteralFormula,omitempty"`
	// Comments
	Note []Annotation `json:"note,omitempty"`
}
