package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDevice is the FHIR resource type name for Device.
const ResourceTypeDevice = "Device"

// DeviceUdiCarrier represents a FHIR BackboneElement for Device.udiCarrier.
type DeviceUdiCarrier struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Mandatory fixed portion of UDI
	DeviceIdentifier *string `json:"deviceIdentifier,omitempty"`
	// UDI Issuing Organization
	Issuer *string `json:"issuer,omitempty"`
	// Regional UDI authority
	Jurisdiction *string `json:"jurisdiction,omitempty"`
	// UDI Machine Readable Barcode String
	CarrierAIDC *string `json:"carrierAIDC,omitempty"`
	// UDI Human Readable Barcode String
	CarrierHRF *string `json:"carrierHRF,omitempty"`
	// barcode | rfid | manual +
	EntryType *string `json:"entryType,omitempty"`
}

// DeviceDeviceName represents a FHIR BackboneElement for Device.deviceName.
type DeviceDeviceName struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The name of the device
	Name string `json:"name"`
	// udi-label-name | user-friendly-name | patient-reported-name | manufacturer-name | model-name | other
	Type string `json:"type"`
}

// DeviceSpecialization represents a FHIR BackboneElement for Device.specialization.
type DeviceSpecialization struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The standard that is used to operate and communicate
	SystemType CodeableConcept `json:"systemType"`
	// The version of the standard that is used to operate and communicate
	Version *string `json:"version,omitempty"`
}

// DeviceVersion represents a FHIR BackboneElement for Device.version.
type DeviceVersion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of the device version
	Type *CodeableConcept `json:"type,omitempty"`
	// A single component of the device version
	Component *Identifier `json:"component,omitempty"`
	// The version text
	Value string `json:"value"`
}

// DeviceProperty represents a FHIR BackboneElement for Device.property.
type DeviceProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code that specifies the property DeviceDefinitionPropetyCode (Extensible)
	Type CodeableConcept `json:"type"`
	// Property value as a quantity
	ValueQuantity []Quantity `json:"valueQuantity,omitempty"`
	// Property value as a code, e.g., NTP4 (synced to NTP)
	ValueCode []CodeableConcept `json:"valueCode,omitempty"`
}

// Device represents a FHIR Device.
type Device struct {
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
	// The reference to the definition for the device
	Definition *Reference `json:"definition,omitempty"`
	// Unique Device Identifier (UDI) Barcode string
	UdiCarrier []DeviceUdiCarrier `json:"udiCarrier,omitempty"`
	// active | inactive | entered-in-error | unknown
	Status *string `json:"status,omitempty"`
	// online | paused | standby | offline | not-ready | transduc-discon | hw-discon | off
	StatusReason []CodeableConcept `json:"statusReason,omitempty"`
	// The distinct identification string
	DistinctIdentifier *string `json:"distinctIdentifier,omitempty"`
	// Name of device manufacturer
	Manufacturer *string `json:"manufacturer,omitempty"`
	// Date when the device was made
	ManufactureDate *primitives.DateTime `json:"manufactureDate,omitempty"`
	// Date and time of expiry of this device (if applicable)
	ExpirationDate *primitives.DateTime `json:"expirationDate,omitempty"`
	// Lot number of manufacture
	LotNumber *string `json:"lotNumber,omitempty"`
	// Serial number assigned by the manufacturer
	SerialNumber *string `json:"serialNumber,omitempty"`
	// The name of the device as given by the manufacturer
	DeviceName []DeviceDeviceName `json:"deviceName,omitempty"`
	// The model number for the device
	ModelNumber *string `json:"modelNumber,omitempty"`
	// The part number of the device
	PartNumber *string `json:"partNumber,omitempty"`
	// The kind or type of device
	Type *CodeableConcept `json:"type,omitempty"`
	// The capabilities supported on a  device, the standards to which the device conforms for a particular purpose, and used for the communication
	Specialization []DeviceSpecialization `json:"specialization,omitempty"`
	// The actual design of the device or software version running on the device
	Version []DeviceVersion `json:"version,omitempty"`
	// The actual configuration settings of a device as it actually operates, e.g., regulation status, time properties
	Property []DeviceProperty `json:"property,omitempty"`
	// Patient to whom Device is affixed
	Patient *Reference `json:"patient,omitempty"`
	// Organization responsible for device
	Owner *Reference `json:"owner,omitempty"`
	// Details for human/organization for support
	Contact []ContactPoint `json:"contact,omitempty"`
	// Where the device is found
	Location *Reference `json:"location,omitempty"`
	// Network address to contact device
	URL *string `json:"url,omitempty"`
	// Device notes and comments
	Note []Annotation `json:"note,omitempty"`
	// Safety Characteristics of Device
	Safety []CodeableConcept `json:"safety,omitempty"`
	// The parent device
	Parent *Reference `json:"parent,omitempty"`
}
