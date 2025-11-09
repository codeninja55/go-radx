package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypePractitionerRole is the FHIR resource type name for PractitionerRole.
const ResourceTypePractitionerRole = "PractitionerRole"

// PractitionerRoleAvailableTime represents a FHIR BackboneElement for PractitionerRole.availableTime.
type PractitionerRoleAvailableTime struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// mon | tue | wed | thu | fri | sat | sun
	DaysOfWeek []string `json:"daysOfWeek,omitempty"`
	// Always available? e.g. 24 hour service
	AllDay *bool `json:"allDay,omitempty"`
	// Opening time of day (ignored if allDay = true)
	AvailableStartTime *primitives.Time `json:"availableStartTime,omitempty"`
	// Closing time of day (ignored if allDay = true)
	AvailableEndTime *primitives.Time `json:"availableEndTime,omitempty"`
}

// PractitionerRoleNotAvailable represents a FHIR BackboneElement for PractitionerRole.notAvailable.
type PractitionerRoleNotAvailable struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reason presented to the user explaining why time not available
	Description string `json:"description"`
	// Service not available from this date
	During *Period `json:"during,omitempty"`
}

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
	// Business Identifiers that are specific to a role/location
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this practitioner role record is in active use
	Active *bool `json:"active,omitempty"`
	// The period during which the practitioner is authorized to perform in these role(s)
	Period *Period `json:"period,omitempty"`
	// Practitioner that is able to provide the defined services for the organization
	Practitioner *Reference `json:"practitioner,omitempty"`
	// Organization where the roles are available
	Organization *Reference `json:"organization,omitempty"`
	// Roles which this practitioner may perform
	Code []CodeableConcept `json:"code,omitempty"`
	// Specific specialty of the practitioner
	Specialty []CodeableConcept `json:"specialty,omitempty"`
	// The location(s) at which this practitioner provides care
	Location []Reference `json:"location,omitempty"`
	// The list of healthcare services that this worker provides for this role's Organization/Location(s)
	HealthcareService []Reference `json:"healthcareService,omitempty"`
	// Contact details that are specific to the role/location/service
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Times the Service Site is available
	AvailableTime []PractitionerRoleAvailableTime `json:"availableTime,omitempty"`
	// Not available during this time due to provided reason
	NotAvailable []PractitionerRoleNotAvailable `json:"notAvailable,omitempty"`
	// Description of availability exceptions
	AvailabilityExceptions *string `json:"availabilityExceptions,omitempty"`
	// Technical endpoints providing access to services operated for the practitioner with this role
	Endpoint []Reference `json:"endpoint,omitempty"`
}
