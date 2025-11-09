package resources

// Reference represents a FHIR Reference.
type Reference struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Literal reference, Relative, internal or absolute URL
	Reference *string `json:"reference,omitempty"`
	// Type the reference refers to (e.g. "Patient") - must be a resource in resources
	Type *string `json:"type,omitempty"`
	// Logical reference, when literal reference is not known
	Identifier *Identifier `json:"identifier,omitempty"`
	// Text alternative for the resource
	Display *string `json:"display,omitempty"`
}
