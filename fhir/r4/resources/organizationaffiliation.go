package resources

// ResourceTypeOrganizationAffiliation is the FHIR resource type name for OrganizationAffiliation.
const ResourceTypeOrganizationAffiliation = "OrganizationAffiliation"

// OrganizationAffiliation represents a FHIR OrganizationAffiliation.
type OrganizationAffiliation struct {
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
	// Business identifiers that are specific to this role
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this organization affiliation record is in active use
	Active *bool `json:"active,omitempty"`
	// The period during which the participatingOrganization is affiliated with the primary organization
	Period *Period `json:"period,omitempty"`
	// Organization where the role is available
	Organization *Reference `json:"organization,omitempty"`
	// Organization that provides/performs the role (e.g. providing services or is a member of)
	ParticipatingOrganization *Reference `json:"participatingOrganization,omitempty"`
	// Health insurance provider network in which the participatingOrganization provides the role's services (if defined) at the indicated locations (if defined)
	Network []Reference `json:"network,omitempty"`
	// Definition of the role the participatingOrganization plays
	Code []CodeableConcept `json:"code,omitempty"`
	// Specific specialty of the participatingOrganization in the context of the role
	Specialty []CodeableConcept `json:"specialty,omitempty"`
	// The location(s) at which the role occurs
	Location []Reference `json:"location,omitempty"`
	// Healthcare services provided through the role
	HealthcareService []Reference `json:"healthcareService,omitempty"`
	// Contact details at the participatingOrganization relevant to this Affiliation
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Technical endpoints providing access to services operated for this role
	Endpoint []Reference `json:"endpoint,omitempty"`
}
