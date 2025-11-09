package resources

// Narrative represents a FHIR Narrative.
type Narrative struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// generated | extensions | additional | empty
	Status string `json:"status"`
	// Limited xhtml content
	Div string `json:"div"`
}
