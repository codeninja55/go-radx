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
	// Participation period of the actor
	Period *Period `json:"period,omitempty"`
	// The individual, device, location, or service participating in the appointment
	Actor *Reference `json:"actor,omitempty"`
	// The participant is required to attend (optional when false)
	Required *bool `json:"required,omitempty"`
	// accepted | declined | tentative | needs-action
	Status string `json:"status"`
}

// AppointmentRecurrenceTemplateWeeklyTemplate represents a FHIR BackboneElement for Appointment.recurrenceTemplate.weeklyTemplate.
type AppointmentRecurrenceTemplateWeeklyTemplate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Recurs on Mondays
	Monday *bool `json:"monday,omitempty"`
	// Recurs on Tuesday
	Tuesday *bool `json:"tuesday,omitempty"`
	// Recurs on Wednesday
	Wednesday *bool `json:"wednesday,omitempty"`
	// Recurs on Thursday
	Thursday *bool `json:"thursday,omitempty"`
	// Recurs on Friday
	Friday *bool `json:"friday,omitempty"`
	// Recurs on Saturday
	Saturday *bool `json:"saturday,omitempty"`
	// Recurs on Sunday
	Sunday *bool `json:"sunday,omitempty"`
	// Recurs every nth week
	WeekInterval *int `json:"weekInterval,omitempty"`
}

// AppointmentRecurrenceTemplateMonthlyTemplate represents a FHIR BackboneElement for Appointment.recurrenceTemplate.monthlyTemplate.
type AppointmentRecurrenceTemplateMonthlyTemplate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Recurs on a specific day of the month
	DayOfMonth *int `json:"dayOfMonth,omitempty"`
	// Indicates which week of the month the appointment should occur
	NthWeekOfMonth *Coding `json:"nthWeekOfMonth,omitempty"`
	// Indicates which day of the week the appointment should occur
	DayOfWeek *Coding `json:"dayOfWeek,omitempty"`
	// Recurs every nth month
	MonthInterval int `json:"monthInterval"`
}

// AppointmentRecurrenceTemplateYearlyTemplate represents a FHIR BackboneElement for Appointment.recurrenceTemplate.yearlyTemplate.
type AppointmentRecurrenceTemplateYearlyTemplate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Recurs every nth year
	YearInterval int `json:"yearInterval"`
}

// AppointmentRecurrenceTemplate represents a FHIR BackboneElement for Appointment.recurrenceTemplate.
type AppointmentRecurrenceTemplate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The timezone of the occurrences
	Timezone *CodeableConcept `json:"timezone,omitempty"`
	// The frequency of the recurrence
	RecurrenceType CodeableConcept `json:"recurrenceType"`
	// The date when the recurrence should end
	LastOccurrenceDate *primitives.Date `json:"lastOccurrenceDate,omitempty"`
	// The number of planned occurrences
	OccurrenceCount *int `json:"occurrenceCount,omitempty"`
	// Specific dates for a recurring set of appointments (no template)
	OccurrenceDate []primitives.Date `json:"occurrenceDate,omitempty"`
	// Information about weekly recurring appointments
	WeeklyTemplate *AppointmentRecurrenceTemplateWeeklyTemplate `json:"weeklyTemplate,omitempty"`
	// Information about monthly recurring appointments
	MonthlyTemplate *AppointmentRecurrenceTemplateMonthlyTemplate `json:"monthlyTemplate,omitempty"`
	// Information about yearly recurring appointments
	YearlyTemplate *AppointmentRecurrenceTemplateYearlyTemplate `json:"yearlyTemplate,omitempty"`
	// Any dates that should be excluded from the series
	ExcludingDate []primitives.Date `json:"excludingDate,omitempty"`
	// Any recurrence IDs that should be excluded from the recurrence
	ExcludingRecurrenceId []int `json:"excludingRecurrenceId,omitempty"`
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
	CancellationReason *CodeableConcept `json:"cancellationReason,omitempty"`
	// Classification when becoming an encounter
	Class []CodeableConcept `json:"class,omitempty"`
	// A broad categorization of the service that is to be performed during this appointment
	ServiceCategory []CodeableConcept `json:"serviceCategory,omitempty"`
	// The specific service that is to be performed during this appointment
	ServiceType []CodeableReference `json:"serviceType,omitempty"`
	// The specialty of a practitioner that would be required to perform the service requested in this appointment
	Specialty []CodeableConcept `json:"specialty,omitempty"`
	// The style of appointment or patient that has been booked in the slot (not service type)
	AppointmentType *CodeableConcept `json:"appointmentType,omitempty"`
	// Reason this appointment is scheduled
	Reason []CodeableReference `json:"reason,omitempty"`
	// Used to make informed decisions if needing to re-prioritize
	Priority *CodeableConcept `json:"priority,omitempty"`
	// Shown on a subject line in a meeting request, or appointment list
	Description *string `json:"description,omitempty"`
	// Appointment replaced by this Appointment
	Replaces []Reference `json:"replaces,omitempty"`
	// Connection details of a virtual service (e.g. conference call)
	VirtualService []VirtualServiceDetail `json:"virtualService,omitempty"`
	// Additional information to support the appointment
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
	// The previous appointment in a series
	PreviousAppointment *Reference `json:"previousAppointment,omitempty"`
	// The originating appointment in a recurring set of appointments
	OriginatingAppointment *Reference `json:"originatingAppointment,omitempty"`
	// When appointment is to take place
	Start *primitives.Instant `json:"start,omitempty"`
	// When appointment is to conclude
	End *primitives.Instant `json:"end,omitempty"`
	// Can be less than start/end (e.g. estimate)
	MinutesDuration *int `json:"minutesDuration,omitempty"`
	// Potential date/time interval(s) requested to allocate the appointment within
	RequestedPeriod []Period `json:"requestedPeriod,omitempty"`
	// The slots that this appointment is filling
	Slot []Reference `json:"slot,omitempty"`
	// The set of accounts that may be used for billing for this Appointment
	Account []Reference `json:"account,omitempty"`
	// The date that this appointment was initially created
	Created *primitives.DateTime `json:"created,omitempty"`
	// When the appointment was cancelled
	CancellationDate *primitives.DateTime `json:"cancellationDate,omitempty"`
	// Additional comments
	Note []Annotation `json:"note,omitempty"`
	// Detailed information and instructions for the patient
	PatientInstruction []CodeableReference `json:"patientInstruction,omitempty"`
	// The request this appointment is allocated to assess
	BasedOn []Reference `json:"basedOn,omitempty"`
	// The patient or group associated with the appointment
	Subject *Reference `json:"subject,omitempty"`
	// Participants involved in appointment
	Participant []AppointmentParticipant `json:"participant,omitempty"`
	// The sequence number in the recurrence
	RecurrenceId *int `json:"recurrenceId,omitempty"`
	// Indicates that this appointment varies from a recurrence pattern
	OccurrenceChanged *bool `json:"occurrenceChanged,omitempty"`
	// Details of the recurrence pattern/template used to generate occurrences
	RecurrenceTemplate []AppointmentRecurrenceTemplate `json:"recurrenceTemplate,omitempty"`
}
