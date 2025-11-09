package resources

// ResourceTypeMedicinalProductPackaged is the FHIR resource type name for MedicinalProductPackaged.
const ResourceTypeMedicinalProductPackaged = "MedicinalProductPackaged"

// MedicinalProductPackagedBatchIdentifier represents a FHIR BackboneElement for MedicinalProductPackaged.batchIdentifier.
type MedicinalProductPackagedBatchIdentifier struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A number appearing on the outer packaging of a specific batch
	OuterPackaging Identifier `json:"outerPackaging"`
	// A number appearing on the immediate packaging (and not the outer packaging)
	ImmediatePackaging *Identifier `json:"immediatePackaging,omitempty"`
}

// MedicinalProductPackagedPackageItemPackageItem represents a FHIR BackboneElement for MedicinalProductPackaged.packageItem.packageItem.
type MedicinalProductPackagedPackageItemPackageItem struct {
}

// MedicinalProductPackagedPackageItem represents a FHIR BackboneElement for MedicinalProductPackaged.packageItem.
type MedicinalProductPackagedPackageItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Including possibly Data Carrier Identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// The physical type of the container of the medicine
	Type CodeableConcept `json:"type"`
	// The quantity of this package in the medicinal product, at the current level of packaging. The outermost is always 1
	Quantity Quantity `json:"quantity"`
	// Material type of the package item
	Material []CodeableConcept `json:"material,omitempty"`
	// A possible alternate material for the packaging
	AlternateMaterial []CodeableConcept `json:"alternateMaterial,omitempty"`
	// A device accompanying a medicinal product
	Device []Reference `json:"device,omitempty"`
	// The manufactured item as contained in the packaged medicinal product
	ManufacturedItem []Reference `json:"manufacturedItem,omitempty"`
	// Allows containers within containers
	PackageItem []MedicinalProductPackagedPackageItemPackageItem `json:"packageItem,omitempty"`
	// Dimensions, color etc.
	PhysicalCharacteristics *ProdCharacteristic `json:"physicalCharacteristics,omitempty"`
	// Other codeable characteristics
	OtherCharacteristics []CodeableConcept `json:"otherCharacteristics,omitempty"`
	// Shelf Life and storage information
	ShelfLifeStorage []ProductShelfLife `json:"shelfLifeStorage,omitempty"`
	// Manufacturer of this Package Item
	Manufacturer []Reference `json:"manufacturer,omitempty"`
}

// MedicinalProductPackaged represents a FHIR MedicinalProductPackaged.
type MedicinalProductPackaged struct {
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
	// Unique identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// The product with this is a pack for
	Subject []Reference `json:"subject,omitempty"`
	// Textual description
	Description *string `json:"description,omitempty"`
	// The legal status of supply of the medicinal product as classified by the regulator
	LegalStatusOfSupply *CodeableConcept `json:"legalStatusOfSupply,omitempty"`
	// Marketing information
	MarketingStatus []MarketingStatus `json:"marketingStatus,omitempty"`
	// Manufacturer of this Package Item
	MarketingAuthorization *Reference `json:"marketingAuthorization,omitempty"`
	// Manufacturer of this Package Item
	Manufacturer []Reference `json:"manufacturer,omitempty"`
	// Batch numbering
	BatchIdentifier []MedicinalProductPackagedBatchIdentifier `json:"batchIdentifier,omitempty"`
	// A packaging item, as a contained for medicine, possibly with other packaging items within
	PackageItem []MedicinalProductPackagedPackageItem `json:"packageItem,omitempty"`
}
