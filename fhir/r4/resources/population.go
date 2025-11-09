package resources

// Population represents a FHIR Population.
type Population struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The age of the specific population
	Age *any `json:"age,omitempty"`
	// The gender of the specific population
	Gender *CodeableConcept `json:"gender,omitempty"`
	// Race of the specific population
	Race *CodeableConcept `json:"race,omitempty"`
	// The existing physiological conditions of the specific population to which this applies
	PhysiologicalCondition *CodeableConcept `json:"physiologicalCondition,omitempty"`
}
