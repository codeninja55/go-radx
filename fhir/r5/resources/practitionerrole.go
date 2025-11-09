package resources

// ResourceTypePractitionerRole is the FHIR resource type name for PractitionerRole.
const ResourceTypePractitionerRole = "PractitionerRole"

// PractitionerRole represents a FHIR PractitionerRole.
type PractitionerRole struct {
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
	// Identifiers for a role/location
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this practitioner role record is in active use
	Active *bool `json:"active,omitempty"`
	// The period during which the practitioner is authorized to perform in these role(s)
	Period *Period `json:"period,omitempty"`
	// Practitioner that provides services for the organization
	Practitioner *Reference `json:"practitioner,omitempty"`
	// Organization where the roles are available
	Organization *Reference `json:"organization,omitempty"`
	// Roles which this practitioner may perform
	Code []CodeableConcept `json:"code,omitempty"`
	// Specific specialty of the practitioner
	Specialty []CodeableConcept `json:"specialty,omitempty"`
	// Location(s) where the practitioner provides care
	Location []Reference `json:"location,omitempty"`
	// Healthcare services provided for this role's Organization/Location(s)
	HealthcareService []Reference `json:"healthcareService,omitempty"`
	// Official contact details relating to this PractitionerRole
	Contact []ExtendedContactDetail `json:"contact,omitempty"`
	// Collection of characteristics (attributes)
	Characteristic []CodeableConcept `json:"characteristic,omitempty"`
	// A language the practitioner (in this role) can use in patient communication
	Communication []CodeableConcept `json:"communication,omitempty"`
	// Times the Practitioner is available at this location and/or healthcare service (including exceptions)
	Availability []Availability `json:"availability,omitempty"`
	// Endpoints for interacting with the practitioner in this role
	Endpoint []Reference `json:"endpoint,omitempty"`
}
