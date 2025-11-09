package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDeviceUsage is the FHIR resource type name for DeviceUsage.
const ResourceTypeDeviceUsage = "DeviceUsage"

// DeviceUsageAdherence represents a FHIR BackboneElement for DeviceUsage.adherence.
type DeviceUsageAdherence struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// always | never | sometimes
	Code CodeableConcept `json:"code"`
	// lost | stolen | prescribed | broken | burned | forgot
	Reason []CodeableConcept `json:"reason,omitempty"`
}

// DeviceUsage represents a FHIR DeviceUsage.
type DeviceUsage struct {
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
	// External identifier for this record
	Identifier []Identifier `json:"identifier,omitempty"`
	// Fulfills plan, proposal or order
	BasedOn []Reference `json:"basedOn,omitempty"`
	// active | completed | not-done | entered-in-error +
	Status string `json:"status"`
	// The category of the statement - classifying how the statement is made
	Category []CodeableConcept `json:"category,omitempty"`
	// Patient using device
	Patient Reference `json:"patient"`
	// Supporting information
	DerivedFrom []Reference `json:"derivedFrom,omitempty"`
	// The encounter or episode of care that establishes the context for this device use statement
	Context *Reference `json:"context,omitempty"`
	// How often  the device was used
	Timing *any `json:"timing,omitempty"`
	// When the statement was made (and recorded)
	DateAsserted *primitives.DateTime `json:"dateAsserted,omitempty"`
	// The status of the device usage, for example always, sometimes, never. This is not the same as the status of the statement
	UsageStatus *CodeableConcept `json:"usageStatus,omitempty"`
	// The reason for asserting the usage status - for example forgot, lost, stolen, broken
	UsageReason []CodeableConcept `json:"usageReason,omitempty"`
	// How device is being used
	Adherence *DeviceUsageAdherence `json:"adherence,omitempty"`
	// Who made the statement
	InformationSource *Reference `json:"informationSource,omitempty"`
	// Code or Reference to device used
	Device CodeableReference `json:"device"`
	// Why device was used
	Reason []CodeableReference `json:"reason,omitempty"`
	// Target body site
	BodySite *CodeableReference `json:"bodySite,omitempty"`
	// Addition details (comments, instructions)
	Note []Annotation `json:"note,omitempty"`
}
