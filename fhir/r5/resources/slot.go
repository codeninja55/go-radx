package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSlot is the FHIR resource type name for Slot.
const ResourceTypeSlot = "Slot"

// Slot represents a FHIR Slot.
type Slot struct {
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
	// A broad categorization of the service that is to be performed during this appointment
	ServiceCategory []CodeableConcept `json:"serviceCategory,omitempty"`
	// The type of appointments that can be booked into this slot (ideally this would be an identifiable service - which is at a location, rather than the location itself). If provided then this overrides the value provided on the Schedule resource
	ServiceType []CodeableReference `json:"serviceType,omitempty"`
	// The specialty of a practitioner that would be required to perform the service requested in this appointment
	Specialty []CodeableConcept `json:"specialty,omitempty"`
	// The style of appointment or patient that may be booked in the slot (not service type)
	AppointmentType []CodeableConcept `json:"appointmentType,omitempty"`
	// The schedule resource that this slot defines an interval of status information
	Schedule Reference `json:"schedule"`
	// busy | free | busy-unavailable | busy-tentative | entered-in-error
	Status string `json:"status"`
	// Date/Time that the slot is to begin
	Start primitives.Instant `json:"start"`
	// Date/Time that the slot is to conclude
	End primitives.Instant `json:"end"`
	// This slot has already been overbooked, appointments are unlikely to be accepted for this time
	Overbooked *bool `json:"overbooked,omitempty"`
	// Comments on the slot to describe any extended information. Such as custom constraints on the slot
	Comment *string `json:"comment,omitempty"`
}
