package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeNutritionProduct is the FHIR resource type name for NutritionProduct.
const ResourceTypeNutritionProduct = "NutritionProduct"

// NutritionProductNutrient represents a FHIR BackboneElement for NutritionProduct.nutrient.
type NutritionProductNutrient struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The (relevant) nutrients in the product
	Item *CodeableReference `json:"item,omitempty"`
	// The amount of nutrient expressed in one or more units: X per pack / per serving / per dose
	Amount []Ratio `json:"amount,omitempty"`
}

// NutritionProductIngredient represents a FHIR BackboneElement for NutritionProduct.ingredient.
type NutritionProductIngredient struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The ingredient contained in the product
	Item CodeableReference `json:"item"`
	// The amount of ingredient that is in the product
	Amount []Ratio `json:"amount,omitempty"`
}

// NutritionProductCharacteristic represents a FHIR BackboneElement for NutritionProduct.characteristic.
type NutritionProductCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code specifying the type of characteristic
	Type CodeableConcept `json:"type"`
	// The value of the characteristic
	Value any `json:"value"`
}

// NutritionProductInstance represents a FHIR BackboneElement for NutritionProduct.instance.
type NutritionProductInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The amount of items or instances
	Quantity *Quantity `json:"quantity,omitempty"`
	// The identifier for the physical instance, typically a serial number or manufacturer number
	Identifier []Identifier `json:"identifier,omitempty"`
	// The name for the specific product
	Name *string `json:"name,omitempty"`
	// The identification of the batch or lot of the product
	LotNumber *string `json:"lotNumber,omitempty"`
	// The expiry date or date and time for the product
	Expiry *primitives.DateTime `json:"expiry,omitempty"`
	// The date until which the product is expected to be good for consumption
	UseBy *primitives.DateTime `json:"useBy,omitempty"`
	// An identifier that supports traceability to the event during which material in this product from one or more biological entities was obtained or pooled
	BiologicalSourceEvent *Identifier `json:"biologicalSourceEvent,omitempty"`
}

// NutritionProduct represents a FHIR NutritionProduct.
type NutritionProduct struct {
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
	// A code that can identify the detailed nutrients and ingredients in a specific food product
	Code *CodeableConcept `json:"code,omitempty"`
	// active | inactive | entered-in-error
	Status string `json:"status"`
	// Broad product groups or categories used to classify the product, such as Legume and Legume Products, Beverages, or Beef Products
	Category []CodeableConcept `json:"category,omitempty"`
	// Manufacturer, representative or officially responsible for the product
	Manufacturer []Reference `json:"manufacturer,omitempty"`
	// The product's nutritional information expressed by the nutrients
	Nutrient []NutritionProductNutrient `json:"nutrient,omitempty"`
	// Ingredients contained in this product
	Ingredient []NutritionProductIngredient `json:"ingredient,omitempty"`
	// Known or suspected allergens that are a part of this product
	KnownAllergen []CodeableReference `json:"knownAllergen,omitempty"`
	// Specifies descriptive properties of the nutrition product
	Characteristic []NutritionProductCharacteristic `json:"characteristic,omitempty"`
	// One or several physical instances or occurrences of the nutrition product
	Instance []NutritionProductInstance `json:"instance,omitempty"`
	// Comments made about the product
	Note []Annotation `json:"note,omitempty"`
}
