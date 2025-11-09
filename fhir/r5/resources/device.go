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
	DeviceIdentifier string `json:"deviceIdentifier"`
	// UDI Issuing Organization
	Issuer string `json:"issuer"`
	// Regional UDI authority
	Jurisdiction *string `json:"jurisdiction,omitempty"`
	// UDI Machine Readable Barcode String
	CarrierAIDC *string `json:"carrierAIDC,omitempty"`
	// UDI Human Readable Barcode String
	CarrierHRF *string `json:"carrierHRF,omitempty"`
	// barcode | rfid | manual | card | self-reported | electronic-transmission | unknown
	EntryType *string `json:"entryType,omitempty"`
}

// DeviceName represents a FHIR BackboneElement for Device.name.
type DeviceName struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The term that names the device
	Value string `json:"value"`
	// registered-name | user-friendly-name | patient-reported-name
	Type string `json:"type"`
	// The preferred device name
	Display *bool `json:"display,omitempty"`
}

// DeviceVersion represents a FHIR BackboneElement for Device.version.
type DeviceVersion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of the device version, e.g. manufacturer, approved, internal
	Type *CodeableConcept `json:"type,omitempty"`
	// The hardware or software module of the device to which the version applies
	Component *Identifier `json:"component,omitempty"`
	// The date the version was installed on the device
	InstallDate *primitives.DateTime `json:"installDate,omitempty"`
	// The version text
	Value string `json:"value"`
}

// DeviceConformsTo represents a FHIR BackboneElement for Device.conformsTo.
type DeviceConformsTo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Describes the common type of the standard, specification, or formal guidance.  communication | performance | measurement
	Category *CodeableConcept `json:"category,omitempty"`
	// Identifies the standard, specification, or formal guidance that the device adheres to
	Specification CodeableConcept `json:"specification"`
	// Specific form or variant of the standard
	Version *string `json:"version,omitempty"`
}

// DeviceProperty represents a FHIR BackboneElement for Device.property.
type DeviceProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code that specifies the property being represented
	Type CodeableConcept `json:"type"`
	// Value of the property
	Value any `json:"value"`
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
	// The name used to display by default when the device is referenced
	DisplayName *string `json:"displayName,omitempty"`
	// The reference to the definition for the device
	Definition *CodeableReference `json:"definition,omitempty"`
	// Unique Device Identifier (UDI) Barcode string
	UdiCarrier []DeviceUdiCarrier `json:"udiCarrier,omitempty"`
	// active | inactive | entered-in-error
	Status *string `json:"status,omitempty"`
	// lost | damaged | destroyed | available
	AvailabilityStatus *CodeableConcept `json:"availabilityStatus,omitempty"`
	// An identifier that supports traceability to the event during which material in this product from one or more biological entities was obtained or pooled
	BiologicalSourceEvent *Identifier `json:"biologicalSourceEvent,omitempty"`
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
	// The name or names of the device as known to the manufacturer and/or patient
	Name []DeviceName `json:"name,omitempty"`
	// The manufacturer's model number for the device
	ModelNumber *string `json:"modelNumber,omitempty"`
	// The part number or catalog number of the device
	PartNumber *string `json:"partNumber,omitempty"`
	// Indicates a high-level grouping of the device
	Category []CodeableConcept `json:"category,omitempty"`
	// The kind or type of device
	Type []CodeableConcept `json:"type,omitempty"`
	// The actual design of the device or software version running on the device
	Version []DeviceVersion `json:"version,omitempty"`
	// Identifies the standards, specifications, or formal guidances for the capabilities supported by the device
	ConformsTo []DeviceConformsTo `json:"conformsTo,omitempty"`
	// Inherent, essentially fixed, characteristics of the device.  e.g., time properties, size, material, etc.
	Property []DeviceProperty `json:"property,omitempty"`
	// The designated condition for performing a task
	Mode *CodeableConcept `json:"mode,omitempty"`
	// The series of occurrences that repeats during the operation of the device
	Cycle *Count `json:"cycle,omitempty"`
	// A measurement of time during the device's operation (e.g., days, hours, mins, etc.)
	Duration *Duration `json:"duration,omitempty"`
	// Organization responsible for device
	Owner *Reference `json:"owner,omitempty"`
	// Details for human/organization for support
	Contact []ContactPoint `json:"contact,omitempty"`
	// Where the device is found
	Location *Reference `json:"location,omitempty"`
	// Network address to contact device
	URL *string `json:"url,omitempty"`
	// Technical endpoints providing access to electronic services provided by the device
	Endpoint []Reference `json:"endpoint,omitempty"`
	// Linked device acting as a communication/data collector, translator or controller
	Gateway []CodeableReference `json:"gateway,omitempty"`
	// Device notes and comments
	Note []Annotation `json:"note,omitempty"`
	// Safety Characteristics of Device
	Safety []CodeableConcept `json:"safety,omitempty"`
	// The higher level or encompassing device that this device is a logical part of
	Parent *Reference `json:"parent,omitempty"`
}
