package resources

// ResourceTypeDeviceAssociation is the FHIR resource type name for DeviceAssociation.
const ResourceTypeDeviceAssociation = "DeviceAssociation"

// DeviceAssociationOperation represents a FHIR BackboneElement for DeviceAssociation.operation.
type DeviceAssociationOperation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Device operational condition
	Status CodeableConcept `json:"status"`
	// The individual performing the action enabled by the device
	Operator []Reference `json:"operator,omitempty"`
	// Begin and end dates and times for the device's operation
	Period *Period `json:"period,omitempty"`
}

// DeviceAssociation represents a FHIR DeviceAssociation.
type DeviceAssociation struct {
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
	// Instance identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Reference to the devices associated with the patient or group
	Device Reference `json:"device"`
	// Describes the relationship between the device and subject
	Category []CodeableConcept `json:"category,omitempty"`
	// implanted | explanted | attached | entered-in-error | unknown
	Status CodeableConcept `json:"status"`
	// The reasons given for the current association status
	StatusReason []CodeableConcept `json:"statusReason,omitempty"`
	// The individual, group of individuals or device that the device is on or associated with
	Subject *Reference `json:"subject,omitempty"`
	// Current anatomical location of the device in/on subject
	BodyStructure *Reference `json:"bodyStructure,omitempty"`
	// Begin and end dates and times for the device association
	Period *Period `json:"period,omitempty"`
	// The details about the device when it is in use to describe its operation
	Operation []DeviceAssociationOperation `json:"operation,omitempty"`
}
