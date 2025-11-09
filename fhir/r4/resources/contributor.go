package resources

// Contributor represents a FHIR Contributor.
type Contributor struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// author | editor | reviewer | endorser
	Type string `json:"type"`
	// Who contributed the content
	Name string `json:"name"`
	// Contact details of the contributor
	Contact []ContactDetail `json:"contact,omitempty"`
}
