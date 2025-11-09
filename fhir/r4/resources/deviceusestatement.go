package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDeviceUseStatement is the FHIR resource type name for DeviceUseStatement.
const ResourceTypeDeviceUseStatement = "DeviceUseStatement"

// DeviceUseStatement represents a FHIR DeviceUseStatement.
type DeviceUseStatement struct {
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
	// active | completed | entered-in-error +
	Status string `json:"status"`
	// Patient using device
	Subject Reference `json:"subject"`
	// Supporting information
	DerivedFrom []Reference `json:"derivedFrom,omitempty"`
	// How often  the device was used
	Timing *any `json:"timing,omitempty"`
	// When statement was recorded
	RecordedOn *primitives.DateTime `json:"recordedOn,omitempty"`
	// Who made the statement
	Source *Reference `json:"source,omitempty"`
	// Reference to device used
	Device Reference `json:"device"`
	// Why device was used
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Why was DeviceUseStatement performed?
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Target body site
	BodySite *CodeableConcept `json:"bodySite,omitempty"`
	// Addition details (comments, instructions)
	Note []Annotation `json:"note,omitempty"`
}
