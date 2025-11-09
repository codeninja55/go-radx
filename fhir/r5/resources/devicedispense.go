package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDeviceDispense is the FHIR resource type name for DeviceDispense.
const ResourceTypeDeviceDispense = "DeviceDispense"

// DeviceDispensePerformer represents a FHIR BackboneElement for DeviceDispense.performer.
type DeviceDispensePerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Who performed the dispense and what they did
	Function *CodeableConcept `json:"function,omitempty"`
	// Individual who was performing
	Actor Reference `json:"actor"`
}

// DeviceDispense represents a FHIR DeviceDispense.
type DeviceDispense struct {
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
	// Business identifier for this dispensation
	Identifier []Identifier `json:"identifier,omitempty"`
	// The order or request that this dispense is fulfilling
	BasedOn []Reference `json:"basedOn,omitempty"`
	// The bigger event that this dispense is a part of
	PartOf []Reference `json:"partOf,omitempty"`
	// preparation | in-progress | cancelled | on-hold | completed | entered-in-error | stopped | declined | unknown
	Status string `json:"status"`
	// Why a dispense was or was not performed
	StatusReason *CodeableReference `json:"statusReason,omitempty"`
	// Type of device dispense
	Category []CodeableConcept `json:"category,omitempty"`
	// What device was supplied
	Device CodeableReference `json:"device"`
	// Who the dispense is for
	Subject Reference `json:"subject"`
	// Who collected the device or where the medication was delivered
	Receiver *Reference `json:"receiver,omitempty"`
	// Encounter associated with event
	Encounter *Reference `json:"encounter,omitempty"`
	// Information that supports the dispensing of the device
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
	// Who performed event
	Performer []DeviceDispensePerformer `json:"performer,omitempty"`
	// Where the dispense occurred
	Location *Reference `json:"location,omitempty"`
	// Trial fill, partial fill, emergency fill, etc
	Type *CodeableConcept `json:"type,omitempty"`
	// Amount dispensed
	Quantity *Quantity `json:"quantity,omitempty"`
	// When product was packaged and reviewed
	PreparedDate *primitives.DateTime `json:"preparedDate,omitempty"`
	// When product was given out
	WhenHandedOver *primitives.DateTime `json:"whenHandedOver,omitempty"`
	// Where the device was sent or should be sent
	Destination *Reference `json:"destination,omitempty"`
	// Information about the dispense
	Note []Annotation `json:"note,omitempty"`
	// Full representation of the usage instructions
	UsageInstruction *string `json:"usageInstruction,omitempty"`
	// A list of relevant lifecycle events
	EventHistory []Reference `json:"eventHistory,omitempty"`
}
