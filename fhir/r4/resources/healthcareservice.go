package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeHealthcareService is the FHIR resource type name for HealthcareService.
const ResourceTypeHealthcareService = "HealthcareService"

// HealthcareServiceEligibility represents a FHIR BackboneElement for HealthcareService.eligibility.
type HealthcareServiceEligibility struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Coded value for the eligibility
	Code *CodeableConcept `json:"code,omitempty"`
	// Describes the eligibility conditions for the service
	Comment *string `json:"comment,omitempty"`
}

// HealthcareServiceAvailableTime represents a FHIR BackboneElement for HealthcareService.availableTime.
type HealthcareServiceAvailableTime struct {
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

// HealthcareServiceNotAvailable represents a FHIR BackboneElement for HealthcareService.notAvailable.
type HealthcareServiceNotAvailable struct {
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

// HealthcareService represents a FHIR HealthcareService.
type HealthcareService struct {
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
	// External identifiers for this item
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this HealthcareService record is in active use
	Active *bool `json:"active,omitempty"`
	// Organization that provides this service
	ProvidedBy *Reference `json:"providedBy,omitempty"`
	// Broad category of service being performed or delivered
	Category []CodeableConcept `json:"category,omitempty"`
	// Type of service that may be delivered or performed
	Type []CodeableConcept `json:"type,omitempty"`
	// Specialties handled by the HealthcareService
	Specialty []CodeableConcept `json:"specialty,omitempty"`
	// Location(s) where service may be provided
	Location []Reference `json:"location,omitempty"`
	// Description of service as presented to a consumer while searching
	Name *string `json:"name,omitempty"`
	// Additional description and/or any specific issues not covered elsewhere
	Comment *string `json:"comment,omitempty"`
	// Extra details about the service that can't be placed in the other fields
	ExtraDetails *string `json:"extraDetails,omitempty"`
	// Facilitates quick identification of the service
	Photo *Attachment `json:"photo,omitempty"`
	// Contacts related to the healthcare service
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Location(s) service is intended for/available to
	CoverageArea []Reference `json:"coverageArea,omitempty"`
	// Conditions under which service is available/offered
	ServiceProvisionCode []CodeableConcept `json:"serviceProvisionCode,omitempty"`
	// Specific eligibility requirements required to use the service
	Eligibility []HealthcareServiceEligibility `json:"eligibility,omitempty"`
	// Programs that this service is applicable to
	Program []CodeableConcept `json:"program,omitempty"`
	// Collection of characteristics (attributes)
	Characteristic []CodeableConcept `json:"characteristic,omitempty"`
	// The language that this service is offered in
	Communication []CodeableConcept `json:"communication,omitempty"`
	// Ways that the service accepts referrals
	ReferralMethod []CodeableConcept `json:"referralMethod,omitempty"`
	// If an appointment is required for access to this service
	AppointmentRequired *bool `json:"appointmentRequired,omitempty"`
	// Times the Service Site is available
	AvailableTime []HealthcareServiceAvailableTime `json:"availableTime,omitempty"`
	// Not available during this time due to provided reason
	NotAvailable []HealthcareServiceNotAvailable `json:"notAvailable,omitempty"`
	// Description of availability exceptions
	AvailabilityExceptions *string `json:"availabilityExceptions,omitempty"`
	// Technical endpoints providing access to electronic services operated for the healthcare service
	Endpoint []Reference `json:"endpoint,omitempty"`
}
