package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeAppointment is the FHIR resource type name for Appointment.
const ResourceTypeAppointment = "Appointment"

// AppointmentParticipant represents a FHIR BackboneElement for Appointment.participant.
type AppointmentParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Role of participant in the appointment
	Type []CodeableConcept `json:"type,omitempty"`
	// Person, Location/HealthcareService or Device
	Actor *Reference `json:"actor,omitempty"`
	// required | optional | information-only
	Required *string `json:"required,omitempty"`
	// accepted | declined | tentative | needs-action
	Status string `json:"status"`
	// Participation period of the actor
	Period *Period `json:"period,omitempty"`
}

// Appointment represents a FHIR Appointment.
type Appointment struct {
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
	// External Ids for this item
	Identifier []Identifier `json:"identifier,omitempty"`
	// proposed | pending | booked | arrived | fulfilled | cancelled | noshow | entered-in-error | checked-in | waitlist
	Status string `json:"status"`
	// The coded reason for the appointment being cancelled
	CancelationReason *CodeableConcept `json:"cancelationReason,omitempty"`
	// A broad categorization of the service that is to be performed during this appointment
	ServiceCategory []CodeableConcept `json:"serviceCategory,omitempty"`
	// The specific service that is to be performed during this appointment
	ServiceType []CodeableConcept `json:"serviceType,omitempty"`
	// The specialty of a practitioner that would be required to perform the service requested in this appointment
	Specialty []CodeableConcept `json:"specialty,omitempty"`
	// The style of appointment or patient that has been booked in the slot (not service type)
	AppointmentType *CodeableConcept `json:"appointmentType,omitempty"`
	// Coded reason this appointment is scheduled
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Reason the appointment is to take place (resource)
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Used to make informed decisions if needing to re-prioritize
	Priority *uint `json:"priority,omitempty"`
	// Shown on a subject line in a meeting request, or appointment list
	Description *string `json:"description,omitempty"`
	// Additional information to support the appointment
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
	// When appointment is to take place
	Start *primitives.Instant `json:"start,omitempty"`
	// When appointment is to conclude
	End *primitives.Instant `json:"end,omitempty"`
	// Can be less than start/end (e.g. estimate)
	MinutesDuration *int `json:"minutesDuration,omitempty"`
	// The slots that this appointment is filling
	Slot []Reference `json:"slot,omitempty"`
	// The date that this appointment was initially created
	Created *primitives.DateTime `json:"created,omitempty"`
	// Additional comments
	Comment *string `json:"comment,omitempty"`
	// Detailed information and instructions for the patient
	PatientInstruction *string `json:"patientInstruction,omitempty"`
	// The service request this appointment is allocated to assess
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Participants involved in appointment
	Participant []AppointmentParticipant `json:"participant,omitempty"`
	// Potential date/time interval(s) requested to allocate the appointment within
	RequestedPeriod []Period `json:"requestedPeriod,omitempty"`
}
