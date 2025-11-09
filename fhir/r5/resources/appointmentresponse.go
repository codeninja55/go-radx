package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeAppointmentResponse is the FHIR resource type name for AppointmentResponse.
const ResourceTypeAppointmentResponse = "AppointmentResponse"

// AppointmentResponse represents a FHIR AppointmentResponse.
type AppointmentResponse struct {
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
	// Appointment this response relates to
	Appointment Reference `json:"appointment"`
	// Indicator for a counter proposal
	ProposedNewTime *bool `json:"proposedNewTime,omitempty"`
	// Time from appointment, or requested new start time
	Start *primitives.Instant `json:"start,omitempty"`
	// Time from appointment, or requested new end time
	End *primitives.Instant `json:"end,omitempty"`
	// Role of participant in the appointment
	ParticipantType []CodeableConcept `json:"participantType,omitempty"`
	// Person(s), Location, HealthcareService, or Device
	Actor *Reference `json:"actor,omitempty"`
	// accepted | declined | tentative | needs-action | entered-in-error
	ParticipantStatus string `json:"participantStatus"`
	// Additional comments
	Comment *string `json:"comment,omitempty"`
	// This response is for all occurrences in a recurring request
	Recurring *bool `json:"recurring,omitempty"`
	// Original date within a recurring request
	OccurrenceDate *primitives.Date `json:"occurrenceDate,omitempty"`
	// The recurrence ID of the specific recurring request
	RecurrenceId *int `json:"recurrenceId,omitempty"`
}
