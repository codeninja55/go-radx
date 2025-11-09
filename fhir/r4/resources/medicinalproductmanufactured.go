package resources

// ResourceTypeMedicinalProductManufactured is the FHIR resource type name for MedicinalProductManufactured.
const ResourceTypeMedicinalProductManufactured = "MedicinalProductManufactured"

// MedicinalProductManufactured represents a FHIR MedicinalProductManufactured.
type MedicinalProductManufactured struct {
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
	// Dose form as manufactured and before any transformation into the pharmaceutical product
	ManufacturedDoseForm CodeableConcept `json:"manufacturedDoseForm"`
	// The “real world” units in which the quantity of the manufactured item is described
	UnitOfPresentation *CodeableConcept `json:"unitOfPresentation,omitempty"`
	// The quantity or "count number" of the manufactured item
	Quantity Quantity `json:"quantity"`
	// Manufacturer of the item (Note that this should be named "manufacturer" but it currently causes technical issues)
	Manufacturer []Reference `json:"manufacturer,omitempty"`
	// Ingredient
	Ingredient []Reference `json:"ingredient,omitempty"`
	// Dimensions, color etc.
	PhysicalCharacteristics *ProdCharacteristic `json:"physicalCharacteristics,omitempty"`
	// Other codeable characteristics
	OtherCharacteristics []CodeableConcept `json:"otherCharacteristics,omitempty"`
}
