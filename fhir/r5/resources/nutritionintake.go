package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeNutritionIntake is the FHIR resource type name for NutritionIntake.
const ResourceTypeNutritionIntake = "NutritionIntake"

// NutritionIntakeConsumedItem represents a FHIR BackboneElement for NutritionIntake.consumedItem.
type NutritionIntakeConsumedItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of food or fluid product
	Type CodeableConcept `json:"type"`
	// Code that identifies the food or fluid product that was consumed
	NutritionProduct CodeableReference `json:"nutritionProduct"`
	// Scheduled frequency of consumption
	Schedule *Timing `json:"schedule,omitempty"`
	// Quantity of the specified food
	Amount *Quantity `json:"amount,omitempty"`
	// Rate at which enteral feeding was administered
	Rate *Quantity `json:"rate,omitempty"`
	// Flag to indicate if the food or fluid item was refused or otherwise not consumed
	NotConsumed *bool `json:"notConsumed,omitempty"`
	// Reason food or fluid was not consumed
	NotConsumedReason *CodeableConcept `json:"notConsumedReason,omitempty"`
}

// NutritionIntakeIngredientLabel represents a FHIR BackboneElement for NutritionIntake.ingredientLabel.
type NutritionIntakeIngredientLabel struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Total nutrient consumed
	Nutrient CodeableReference `json:"nutrient"`
	// Total amount of nutrient consumed
	Amount Quantity `json:"amount"`
}

// NutritionIntakePerformer represents a FHIR BackboneElement for NutritionIntake.performer.
type NutritionIntakePerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of performer
	Function *CodeableConcept `json:"function,omitempty"`
	// Who performed the intake
	Actor Reference `json:"actor"`
}

// NutritionIntake represents a FHIR NutritionIntake.
type NutritionIntake struct {
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
	// External identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// Fulfils plan, proposal or order
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// preparation | in-progress | not-done | on-hold | stopped | completed | entered-in-error | unknown
	Status string `json:"status"`
	// Reason for current status
	StatusReason []CodeableConcept `json:"statusReason,omitempty"`
	// Code representing an overall type of nutrition intake
	Code *CodeableConcept `json:"code,omitempty"`
	// Who is/was consuming the food or fluid
	Subject Reference `json:"subject"`
	// Encounter associated with NutritionIntake
	Encounter *Reference `json:"encounter,omitempty"`
	// The date/time or interval when the food or fluid is/was consumed
	Occurrence *any `json:"occurrence,omitempty"`
	// When the intake was recorded
	Recorded *primitives.DateTime `json:"recorded,omitempty"`
	// Person or organization that provided the information about the consumption of this food or fluid
	Reported *any `json:"reported,omitempty"`
	// What food or fluid product or item was consumed
	ConsumedItem []NutritionIntakeConsumedItem `json:"consumedItem,omitempty"`
	// Total nutrient for the whole meal, product, serving
	IngredientLabel []NutritionIntakeIngredientLabel `json:"ingredientLabel,omitempty"`
	// Who was performed in the intake
	Performer []NutritionIntakePerformer `json:"performer,omitempty"`
	// Where the intake occurred
	Location *Reference `json:"location,omitempty"`
	// Additional supporting information
	DerivedFrom []Reference `json:"derivedFrom,omitempty"`
	// Reason for why the food or fluid is /was consumed
	Reason []CodeableReference `json:"reason,omitempty"`
	// Further information about the consumption
	Note []Annotation `json:"note,omitempty"`
}
