package resources

// HumanName represents a FHIR HumanName.
type HumanName struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// usual | official | temp | nickname | anonymous | old | maiden
	Use *string `json:"use,omitempty"`
	// Text representation of the full name
	Text *string `json:"text,omitempty"`
	// Family name (often called 'Surname')
	Family *string `json:"family,omitempty"`
	// Given names (not always 'first'). Includes middle names
	Given []string `json:"given,omitempty"`
	// Parts that come before the name
	Prefix []string `json:"prefix,omitempty"`
	// Parts that come after the name
	Suffix []string `json:"suffix,omitempty"`
	// Time period when name was/is in use
	Period *Period `json:"period,omitempty"`
}
