package resources

// ResourceTypeDeviceDefinition is the FHIR resource type name for DeviceDefinition.
const ResourceTypeDeviceDefinition = "DeviceDefinition"

// DeviceDefinitionUdiDeviceIdentifier represents a FHIR BackboneElement for DeviceDefinition.udiDeviceIdentifier.
type DeviceDefinitionUdiDeviceIdentifier struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The identifier that is to be associated with every Device that references this DeviceDefintiion for the issuer and jurisdication porvided in the DeviceDefinition.udiDeviceIdentifier
	DeviceIdentifier string `json:"deviceIdentifier"`
	// The organization that assigns the identifier algorithm
	Issuer string `json:"issuer"`
	// The jurisdiction to which the deviceIdentifier applies
	Jurisdiction string `json:"jurisdiction"`
}

// DeviceDefinitionDeviceName represents a FHIR BackboneElement for DeviceDefinition.deviceName.
type DeviceDefinitionDeviceName struct {
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

// DeviceDefinitionSpecialization represents a FHIR BackboneElement for DeviceDefinition.specialization.
type DeviceDefinitionSpecialization struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The standard that is used to operate and communicate
	SystemType string `json:"systemType"`
	// The version of the standard that is used to operate and communicate
	Version *string `json:"version,omitempty"`
}

// DeviceDefinitionCapability represents a FHIR BackboneElement for DeviceDefinition.capability.
type DeviceDefinitionCapability struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of capability
	Type CodeableConcept `json:"type"`
	// Description of capability
	Description []CodeableConcept `json:"description,omitempty"`
}

// DeviceDefinitionProperty represents a FHIR BackboneElement for DeviceDefinition.property.
type DeviceDefinitionProperty struct {
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

// DeviceDefinitionMaterial represents a FHIR BackboneElement for DeviceDefinition.material.
type DeviceDefinitionMaterial struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The substance
	Substance CodeableConcept `json:"substance"`
	// Indicates an alternative material of the device
	Alternate *bool `json:"alternate,omitempty"`
	// Whether the substance is a known or suspected allergen
	AllergenicIndicator *bool `json:"allergenicIndicator,omitempty"`
}

// DeviceDefinition represents a FHIR DeviceDefinition.
type DeviceDefinition struct {
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
	// Unique Device Identifier (UDI) Barcode string
	UdiDeviceIdentifier []DeviceDefinitionUdiDeviceIdentifier `json:"udiDeviceIdentifier,omitempty"`
	// Name of device manufacturer
	Manufacturer *any `json:"manufacturer,omitempty"`
	// A name given to the device to identify it
	DeviceName []DeviceDefinitionDeviceName `json:"deviceName,omitempty"`
	// The model number for the device
	ModelNumber *string `json:"modelNumber,omitempty"`
	// What kind of device or device system this is
	Type *CodeableConcept `json:"type,omitempty"`
	// The capabilities supported on a  device, the standards to which the device conforms for a particular purpose, and used for the communication
	Specialization []DeviceDefinitionSpecialization `json:"specialization,omitempty"`
	// Available versions
	Version []string `json:"version,omitempty"`
	// Safety characteristics of the device
	Safety []CodeableConcept `json:"safety,omitempty"`
	// Shelf Life and storage information
	ShelfLifeStorage []ProductShelfLife `json:"shelfLifeStorage,omitempty"`
	// Dimensions, color etc.
	PhysicalCharacteristics *ProdCharacteristic `json:"physicalCharacteristics,omitempty"`
	// Language code for the human-readable text strings produced by the device (all supported)
	LanguageCode []CodeableConcept `json:"languageCode,omitempty"`
	// Device capabilities
	Capability []DeviceDefinitionCapability `json:"capability,omitempty"`
	// The actual configuration settings of a device as it actually operates, e.g., regulation status, time properties
	Property []DeviceDefinitionProperty `json:"property,omitempty"`
	// Organization responsible for device
	Owner *Reference `json:"owner,omitempty"`
	// Details for human/organization for support
	Contact []ContactPoint `json:"contact,omitempty"`
	// Network address to contact device
	URL *string `json:"url,omitempty"`
	// Access to on-line information
	OnlineInformation *string `json:"onlineInformation,omitempty"`
	// Device notes and comments
	Note []Annotation `json:"note,omitempty"`
	// The quantity of the device present in the packaging (e.g. the number of devices present in a pack, or the number of devices in the same package of the medicinal product)
	Quantity *Quantity `json:"quantity,omitempty"`
	// The parent device it can be part of
	ParentDevice *Reference `json:"parentDevice,omitempty"`
	// A substance used to create the material(s) of which the device is made
	Material []DeviceDefinitionMaterial `json:"material,omitempty"`
}
