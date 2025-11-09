package resources

// ContactDetail represents a FHIR ContactDetail.
type ContactDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Name of an individual to contact
	Name *string `json:"name,omitempty"`
	// Contact details for individual or organization
	Telecom []ContactPoint `json:"telecom,omitempty"`
}
