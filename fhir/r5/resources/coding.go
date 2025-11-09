package resources

// Coding represents a FHIR Coding.
type Coding struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Identity of the terminology system
	System *string `json:"system,omitempty"`
	// Version of the system - if relevant
	Version *string `json:"version,omitempty"`
	// Symbol in syntax defined by the system
	Code *string `json:"code,omitempty"`
	// Representation defined by the system
	Display *string `json:"display,omitempty"`
	// If this coding was chosen directly by the user
	UserSelected *bool `json:"userSelected,omitempty"`
}
