package resources

// CodeableReference represents a FHIR CodeableReference.
type CodeableReference struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Reference to a concept (by class)
	Concept *CodeableConcept `json:"concept,omitempty"`
	// Reference to a resource (by instance)
	Reference *Reference `json:"reference,omitempty"`
}
