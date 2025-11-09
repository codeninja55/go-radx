package resources

// ResourceTypeDeviceDefinition is the FHIR resource type name for DeviceDefinition.
const ResourceTypeDeviceDefinition = "DeviceDefinition"

// DeviceDefinitionUdiDeviceIdentifierMarketDistribution represents a FHIR BackboneElement for DeviceDefinition.udiDeviceIdentifier.marketDistribution.
type DeviceDefinitionUdiDeviceIdentifierMarketDistribution struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Begin and end dates for the commercial distribution of the device
	MarketPeriod Period `json:"marketPeriod"`
	// National state or territory where the device is commercialized
	SubJurisdiction string `json:"subJurisdiction"`
}

// DeviceDefinitionUdiDeviceIdentifier represents a FHIR BackboneElement for DeviceDefinition.udiDeviceIdentifier.
type DeviceDefinitionUdiDeviceIdentifier struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The identifier that is to be associated with every Device that references this DeviceDefintiion for the issuer and jurisdiction provided in the DeviceDefinition.udiDeviceIdentifier
	DeviceIdentifier string `json:"deviceIdentifier"`
	// The organization that assigns the identifier algorithm
	Issuer string `json:"issuer"`
	// The jurisdiction to which the deviceIdentifier applies
	Jurisdiction string `json:"jurisdiction"`
	// Indicates whether and when the device is available on the market
	MarketDistribution []DeviceDefinitionUdiDeviceIdentifierMarketDistribution `json:"marketDistribution,omitempty"`
}

// DeviceDefinitionRegulatoryIdentifier represents a FHIR BackboneElement for DeviceDefinition.regulatoryIdentifier.
type DeviceDefinitionRegulatoryIdentifier struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// basic | master | license
	Type string `json:"type"`
	// The identifier itself
	DeviceIdentifier string `json:"deviceIdentifier"`
	// The organization that issued this identifier
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
	// A name that is used to refer to the device
	Name string `json:"name"`
	// registered-name | user-friendly-name | patient-reported-name
	Type string `json:"type"`
}

// DeviceDefinitionClassification represents a FHIR BackboneElement for DeviceDefinition.classification.
type DeviceDefinitionClassification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A classification or risk class of the device model
	Type CodeableConcept `json:"type"`
	// Further information qualifying this classification of the device model
	Justification []RelatedArtifact `json:"justification,omitempty"`
}

// DeviceDefinitionConformsTo represents a FHIR BackboneElement for DeviceDefinition.conformsTo.
type DeviceDefinitionConformsTo struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Describes the common type of the standard, specification, or formal guidance
	Category *CodeableConcept `json:"category,omitempty"`
	// Identifies the standard, specification, or formal guidance that the device adheres to the Device Specification type
	Specification CodeableConcept `json:"specification"`
	// The specific form or variant of the standard, specification or formal guidance
	Version []string `json:"version,omitempty"`
	// Standard, regulation, certification, or guidance website, document, or other publication, or similar, supporting the conformance
	Source []RelatedArtifact `json:"source,omitempty"`
}

// DeviceDefinitionHasPart represents a FHIR BackboneElement for DeviceDefinition.hasPart.
type DeviceDefinitionHasPart struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to the part
	Reference Reference `json:"reference"`
	// Number of occurrences of the part
	Count *int `json:"count,omitempty"`
}

// DeviceDefinitionPackagingDistributor represents a FHIR BackboneElement for DeviceDefinition.packaging.distributor.
type DeviceDefinitionPackagingDistributor struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Distributor's human-readable name
	Name *string `json:"name,omitempty"`
	// Distributor as an Organization resource
	OrganizationReference []Reference `json:"organizationReference,omitempty"`
}

// DeviceDefinitionPackagingUdiDeviceIdentifier represents a FHIR BackboneElement for DeviceDefinition.packaging.udiDeviceIdentifier.
type DeviceDefinitionPackagingUdiDeviceIdentifier struct {
}

// DeviceDefinitionPackagingPackaging represents a FHIR BackboneElement for DeviceDefinition.packaging.packaging.
type DeviceDefinitionPackagingPackaging struct {
}

// DeviceDefinitionPackaging represents a FHIR BackboneElement for DeviceDefinition.packaging.
type DeviceDefinitionPackaging struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Business identifier of the packaged medication
	Identifier *Identifier `json:"identifier,omitempty"`
	// A code that defines the specific type of packaging
	Type *CodeableConcept `json:"type,omitempty"`
	// The number of items contained in the package (devices or sub-packages)
	Count *int `json:"count,omitempty"`
	// An organization that distributes the packaged device
	Distributor []DeviceDefinitionPackagingDistributor `json:"distributor,omitempty"`
	// Unique Device Identifier (UDI) Barcode string on the packaging
	UdiDeviceIdentifier []DeviceDefinitionPackagingUdiDeviceIdentifier `json:"udiDeviceIdentifier,omitempty"`
	// Allows packages within packages
	Packaging []DeviceDefinitionPackagingPackaging `json:"packaging,omitempty"`
}

// DeviceDefinitionVersion represents a FHIR BackboneElement for DeviceDefinition.version.
type DeviceDefinitionVersion struct {
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
	// The version text
	Value string `json:"value"`
}

// DeviceDefinitionProperty represents a FHIR BackboneElement for DeviceDefinition.property.
type DeviceDefinitionProperty struct {
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

// DeviceDefinitionLink represents a FHIR BackboneElement for DeviceDefinition.link.
type DeviceDefinitionLink struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type indicates the relationship of the related device to the device instance
	Relation Coding `json:"relation"`
	// A reference to the linked device
	RelatedDevice CodeableReference `json:"relatedDevice"`
}

// DeviceDefinitionMaterial represents a FHIR BackboneElement for DeviceDefinition.material.
type DeviceDefinitionMaterial struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A relevant substance that the device contains, may contain, or is made of
	Substance CodeableConcept `json:"substance"`
	// Indicates an alternative material of the device
	Alternate *bool `json:"alternate,omitempty"`
	// Whether the substance is a known or suspected allergen
	AllergenicIndicator *bool `json:"allergenicIndicator,omitempty"`
}

// DeviceDefinitionGuideline represents a FHIR BackboneElement for DeviceDefinition.guideline.
type DeviceDefinitionGuideline struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The circumstances that form the setting for using the device
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Detailed written and visual directions for the user on how to use the device
	UsageInstruction *string `json:"usageInstruction,omitempty"`
	// A source of information or reference for this guideline
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// A clinical condition for which the device was designed to be used
	Indication []CodeableConcept `json:"indication,omitempty"`
	// A specific situation when a device should not be used because it may cause harm
	Contraindication []CodeableConcept `json:"contraindication,omitempty"`
	// Specific hazard alert information that a user needs to know before using the device
	Warning []CodeableConcept `json:"warning,omitempty"`
	// A description of the general purpose or medical use of the device or its function
	IntendedUse *string `json:"intendedUse,omitempty"`
}

// DeviceDefinitionCorrectiveAction represents a FHIR BackboneElement for DeviceDefinition.correctiveAction.
type DeviceDefinitionCorrectiveAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Whether the corrective action was a recall
	Recall bool `json:"recall"`
	// model | lot-numbers | serial-numbers
	Scope *string `json:"scope,omitempty"`
	// Start and end dates of the  corrective action
	Period Period `json:"period"`
}

// DeviceDefinitionChargeItem represents a FHIR BackboneElement for DeviceDefinition.chargeItem.
type DeviceDefinitionChargeItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The code or reference for the charge item
	ChargeItemCode CodeableReference `json:"chargeItemCode"`
	// Coefficient applicable to the billing code
	Count Quantity `json:"count"`
	// A specific time period in which this charge item applies
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// The context to which this charge item applies
	UseContext []UsageContext `json:"useContext,omitempty"`
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
	// Additional information to describe the device
	Description *string `json:"description,omitempty"`
	// Instance identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Unique Device Identifier (UDI) Barcode string
	UdiDeviceIdentifier []DeviceDefinitionUdiDeviceIdentifier `json:"udiDeviceIdentifier,omitempty"`
	// Regulatory identifier(s) associated with this device
	RegulatoryIdentifier []DeviceDefinitionRegulatoryIdentifier `json:"regulatoryIdentifier,omitempty"`
	// The part number or catalog number of the device
	PartNumber *string `json:"partNumber,omitempty"`
	// Name of device manufacturer
	Manufacturer *Reference `json:"manufacturer,omitempty"`
	// The name or names of the device as given by the manufacturer
	DeviceName []DeviceDefinitionDeviceName `json:"deviceName,omitempty"`
	// The catalog or model number for the device for example as defined by the manufacturer
	ModelNumber *string `json:"modelNumber,omitempty"`
	// What kind of device or device system this is
	Classification []DeviceDefinitionClassification `json:"classification,omitempty"`
	// Identifies the standards, specifications, or formal guidances for the capabilities supported by the device
	ConformsTo []DeviceDefinitionConformsTo `json:"conformsTo,omitempty"`
	// A device, part of the current one
	HasPart []DeviceDefinitionHasPart `json:"hasPart,omitempty"`
	// Information about the packaging of the device, i.e. how the device is packaged
	Packaging []DeviceDefinitionPackaging `json:"packaging,omitempty"`
	// The version of the device or software
	Version []DeviceDefinitionVersion `json:"version,omitempty"`
	// Safety characteristics of the device
	Safety []CodeableConcept `json:"safety,omitempty"`
	// Shelf Life and storage information
	ShelfLifeStorage []ProductShelfLife `json:"shelfLifeStorage,omitempty"`
	// Language code for the human-readable text strings produced by the device (all supported)
	LanguageCode []CodeableConcept `json:"languageCode,omitempty"`
	// Inherent, essentially fixed, characteristics of this kind of device, e.g., time properties, size, etc
	Property []DeviceDefinitionProperty `json:"property,omitempty"`
	// Organization responsible for device
	Owner *Reference `json:"owner,omitempty"`
	// Details for human/organization for support
	Contact []ContactPoint `json:"contact,omitempty"`
	// An associated device, attached to, used with, communicating with or linking a previous or new device model to the focal device
	Link []DeviceDefinitionLink `json:"link,omitempty"`
	// Device notes and comments
	Note []Annotation `json:"note,omitempty"`
	// A substance used to create the material(s) of which the device is made
	Material []DeviceDefinitionMaterial `json:"material,omitempty"`
	// lot-number | manufactured-date | serial-number | expiration-date | biological-source | software-version
	ProductionIdentifierInUDI []string `json:"productionIdentifierInUDI,omitempty"`
	// Information aimed at providing directions for the usage of this model of device
	Guideline *DeviceDefinitionGuideline `json:"guideline,omitempty"`
	// Tracking of latest field safety corrective action
	CorrectiveAction *DeviceDefinitionCorrectiveAction `json:"correctiveAction,omitempty"`
	// Billing code or reference associated with the device
	ChargeItem []DeviceDefinitionChargeItem `json:"chargeItem,omitempty"`
}
