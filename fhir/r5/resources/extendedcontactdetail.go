package resources

// ExtendedContactDetail represents a FHIR ExtendedContactDetail.
type ExtendedContactDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// The type of contact
	Purpose *CodeableConcept `json:"purpose,omitempty"`
	// Name of an individual to contact
	Name []HumanName `json:"name,omitempty"`
	// Contact details (e.g.phone/fax/url)
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Address for the contact
	Address *Address `json:"address,omitempty"`
	// This contact detail is handled/monitored by a specific organization
	Organization *Reference `json:"organization,omitempty"`
	// Period that this contact was valid for usage
	Period *Period `json:"period,omitempty"`
}
