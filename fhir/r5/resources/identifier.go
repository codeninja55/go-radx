package resources

// Identifier represents a FHIR Identifier.
type Identifier struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// usual | official | temp | secondary | old (If known)
	Use *string `json:"use,omitempty"`
	// Description of identifier
	Type *CodeableConcept `json:"type,omitempty"`
	// The namespace for the identifier value
	System *string `json:"system,omitempty"`
	// The value that is unique
	Value *string `json:"value,omitempty"`
	// Time period when id is/was valid for use
	Period *Period `json:"period,omitempty"`
	// Organization that issued id (may be just text)
	Assigner *Reference `json:"assigner,omitempty"`
}
