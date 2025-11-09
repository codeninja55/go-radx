package resources

// Address represents a FHIR Address.
type Address struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// home | work | temp | old | billing - purpose of this address
	Use *string `json:"use,omitempty"`
	// postal | physical | both
	Type *string `json:"type,omitempty"`
	// Text representation of the address
	Text *string `json:"text,omitempty"`
	// Street name, number, direction & P.O. Box etc.
	Line []string `json:"line,omitempty"`
	// Name of city, town etc.
	City *string `json:"city,omitempty"`
	// District name (aka county)
	District *string `json:"district,omitempty"`
	// Sub-unit of country (abbreviations ok)
	State *string `json:"state,omitempty"`
	// Postal code for area
	PostalCode *string `json:"postalCode,omitempty"`
	// Country (e.g. may be ISO 3166 2 or 3 letter code)
	Country *string `json:"country,omitempty"`
	// Time period when address was/is in use
	Period *Period `json:"period,omitempty"`
}
