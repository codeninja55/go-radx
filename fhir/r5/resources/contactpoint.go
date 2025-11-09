package resources

// ContactPoint represents a FHIR ContactPoint.
type ContactPoint struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// phone | fax | email | pager | url | sms | other
	System *string `json:"system,omitempty"`
	// The actual contact point details
	Value *string `json:"value,omitempty"`
	// home | work | temp | old | mobile - purpose of this contact point
	Use *string `json:"use,omitempty"`
	// Specify preferred order of use (1 = highest)
	Rank *int `json:"rank,omitempty"`
	// Time period when the contact point was/is in use
	Period *Period `json:"period,omitempty"`
}
