package resources

// ResourceTypeMedicinalProductPharmaceutical is the FHIR resource type name for MedicinalProductPharmaceutical.
const ResourceTypeMedicinalProductPharmaceutical = "MedicinalProductPharmaceutical"

// MedicinalProductPharmaceuticalCharacteristics represents a FHIR BackboneElement for MedicinalProductPharmaceutical.characteristics.
type MedicinalProductPharmaceuticalCharacteristics struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A coded characteristic
	Code CodeableConcept `json:"code"`
	// The status of characteristic e.g. assigned or pending
	Status *CodeableConcept `json:"status,omitempty"`
}

// MedicinalProductPharmaceuticalRouteOfAdministrationTargetSpeciesWithdrawalPeriod represents a FHIR BackboneElement for MedicinalProductPharmaceutical.routeOfAdministration.targetSpecies.withdrawalPeriod.
type MedicinalProductPharmaceuticalRouteOfAdministrationTargetSpeciesWithdrawalPeriod struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Coded expression for the type of tissue for which the withdrawal period applues, e.g. meat, milk
	Tissue CodeableConcept `json:"tissue"`
	// A value for the time
	Value Quantity `json:"value"`
	// Extra information about the withdrawal period
	SupportingInformation *string `json:"supportingInformation,omitempty"`
}

// MedicinalProductPharmaceuticalRouteOfAdministrationTargetSpecies represents a FHIR BackboneElement for MedicinalProductPharmaceutical.routeOfAdministration.targetSpecies.
type MedicinalProductPharmaceuticalRouteOfAdministrationTargetSpecies struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Coded expression for the species
	Code CodeableConcept `json:"code"`
	// A species specific time during which consumption of animal product is not appropriate
	WithdrawalPeriod []MedicinalProductPharmaceuticalRouteOfAdministrationTargetSpeciesWithdrawalPeriod `json:"withdrawalPeriod,omitempty"`
}

// MedicinalProductPharmaceuticalRouteOfAdministration represents a FHIR BackboneElement for MedicinalProductPharmaceutical.routeOfAdministration.
type MedicinalProductPharmaceuticalRouteOfAdministration struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Coded expression for the route
	Code CodeableConcept `json:"code"`
	// The first dose (dose quantity) administered in humans can be specified, for a product under investigation, using a numerical value and its unit of measurement
	FirstDose *Quantity `json:"firstDose,omitempty"`
	// The maximum single dose that can be administered as per the protocol of a clinical trial can be specified using a numerical value and its unit of measurement
	MaxSingleDose *Quantity `json:"maxSingleDose,omitempty"`
	// The maximum dose per day (maximum dose quantity to be administered in any one 24-h period) that can be administered as per the protocol referenced in the clinical trial authorisation
	MaxDosePerDay *Quantity `json:"maxDosePerDay,omitempty"`
	// The maximum dose per treatment period that can be administered as per the protocol referenced in the clinical trial authorisation
	MaxDosePerTreatmentPeriod *Ratio `json:"maxDosePerTreatmentPeriod,omitempty"`
	// The maximum treatment period during which an Investigational Medicinal Product can be administered as per the protocol referenced in the clinical trial authorisation
	MaxTreatmentPeriod *Duration `json:"maxTreatmentPeriod,omitempty"`
	// A species for which this route applies
	TargetSpecies []MedicinalProductPharmaceuticalRouteOfAdministrationTargetSpecies `json:"targetSpecies,omitempty"`
}

// MedicinalProductPharmaceutical represents a FHIR MedicinalProductPharmaceutical.
type MedicinalProductPharmaceutical struct {
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
	// An identifier for the pharmaceutical medicinal product
	Identifier []Identifier `json:"identifier,omitempty"`
	// The administrable dose form, after necessary reconstitution
	AdministrableDoseForm CodeableConcept `json:"administrableDoseForm"`
	// Todo
	UnitOfPresentation *CodeableConcept `json:"unitOfPresentation,omitempty"`
	// Ingredient
	Ingredient []Reference `json:"ingredient,omitempty"`
	// Accompanying device
	Device []Reference `json:"device,omitempty"`
	// Characteristics e.g. a products onset of action
	Characteristics []MedicinalProductPharmaceuticalCharacteristics `json:"characteristics,omitempty"`
	// The path by which the pharmaceutical product is taken into or makes contact with the body
	RouteOfAdministration []MedicinalProductPharmaceuticalRouteOfAdministration `json:"routeOfAdministration,omitempty"`
}
