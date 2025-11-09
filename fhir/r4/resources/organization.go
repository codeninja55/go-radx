package resources

// ResourceTypeOrganization is the FHIR resource type name for Organization.
const ResourceTypeOrganization = "Organization"

// OrganizationContact represents a FHIR BackboneElement for Organization.contact.
type OrganizationContact struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of contact
	Purpose *CodeableConcept `json:"purpose,omitempty"`
	// A name associated with the contact
	Name *HumanName `json:"name,omitempty"`
	// Contact details (telephone, email, etc.)  for a contact
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Visiting or postal addresses for the contact
	Address *Address `json:"address,omitempty"`
}

// Organization represents a FHIR Organization.
type Organization struct {
	// Logical id of this artifact
	ID *string `json:"id,omitempty"`
	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty"`
	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty"`
	// Language of the resource content
	Language *string `json:"language,omitempty"`
	// Text summary of the resource, for human interpretation
	Text *Narrative `json:"text,omitempty"`
	// Contained, inline Resources
	Contained []any `json:"contained,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifies this organization  across multiple systems
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether the organization's record is still in active use
	Active *bool `json:"active,omitempty"`
	// Kind of organization
	Type []CodeableConcept `json:"type,omitempty"`
	// Name used for the organization
	Name *string `json:"name,omitempty"`
	// A list of alternate names that the organization is known as, or was known as in the past
	Alias []string `json:"alias,omitempty"`
	// A contact detail for the organization
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// An address for the organization
	Address []Address `json:"address,omitempty"`
	// The organization of which this organization forms a part
	PartOf *Reference `json:"partOf,omitempty"`
	// Contact for the organization for a certain purpose
	Contact []OrganizationContact `json:"contact,omitempty"`
	// Technical endpoints providing access to services operated for the organization
	Endpoint []Reference `json:"endpoint,omitempty"`
}
