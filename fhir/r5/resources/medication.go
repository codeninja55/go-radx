package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedication is the FHIR resource type name for Medication.
const ResourceTypeMedication = "Medication"

// MedicationIngredient represents a FHIR BackboneElement for Medication.ingredient.
type MedicationIngredient struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The ingredient (substance or medication) that the ingredient.strength relates to
	Item CodeableReference `json:"item"`
	// Active ingredient indicator
	IsActive *bool `json:"isActive,omitempty"`
	// Quantity of ingredient present
	Strength *any `json:"strength,omitempty"`
}

// MedicationBatch represents a FHIR BackboneElement for Medication.batch.
type MedicationBatch struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifier assigned to batch
	LotNumber *string `json:"lotNumber,omitempty"`
	// When batch will expire
	ExpirationDate *primitives.DateTime `json:"expirationDate,omitempty"`
}

// Medication represents a FHIR Medication.
type Medication struct {
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
	// Business identifier for this medication
	Identifier []Identifier `json:"identifier,omitempty"`
	// Codes that identify this medication
	Code *CodeableConcept `json:"code,omitempty"`
	// active | inactive | entered-in-error
	Status *string `json:"status,omitempty"`
	// Organization that has authorization to market medication
	MarketingAuthorizationHolder *Reference `json:"marketingAuthorizationHolder,omitempty"`
	// powder | tablets | capsule +
	DoseForm *CodeableConcept `json:"doseForm,omitempty"`
	// When the specified product code does not infer a package size, this is the specific amount of drug in the product
	TotalVolume *Quantity `json:"totalVolume,omitempty"`
	// Active or inactive ingredient
	Ingredient []MedicationIngredient `json:"ingredient,omitempty"`
	// Details about packaged medications
	Batch *MedicationBatch `json:"batch,omitempty"`
	// Knowledge about this medication
	Definition *Reference `json:"definition,omitempty"`
}
