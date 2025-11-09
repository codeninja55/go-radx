package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedicinalProduct is the FHIR resource type name for MedicinalProduct.
const ResourceTypeMedicinalProduct = "MedicinalProduct"

// MedicinalProductNameNamePart represents a FHIR BackboneElement for MedicinalProduct.name.namePart.
type MedicinalProductNameNamePart struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A fragment of a product name
	Part string `json:"part"`
	// Idenifying type for this part of the name (e.g. strength part)
	Type Coding `json:"type"`
}

// MedicinalProductNameCountryLanguage represents a FHIR BackboneElement for MedicinalProduct.name.countryLanguage.
type MedicinalProductNameCountryLanguage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Country code for where this name applies
	Country CodeableConcept `json:"country"`
	// Jurisdiction code for where this name applies
	Jurisdiction *CodeableConcept `json:"jurisdiction,omitempty"`
	// Language code for this name
	Language CodeableConcept `json:"language"`
}

// MedicinalProductName represents a FHIR BackboneElement for MedicinalProduct.name.
type MedicinalProductName struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The full product name
	ProductName string `json:"productName"`
	// Coding words or phrases of the name
	NamePart []MedicinalProductNameNamePart `json:"namePart,omitempty"`
	// Country where the name applies
	CountryLanguage []MedicinalProductNameCountryLanguage `json:"countryLanguage,omitempty"`
}

// MedicinalProductManufacturingBusinessOperation represents a FHIR BackboneElement for MedicinalProduct.manufacturingBusinessOperation.
type MedicinalProductManufacturingBusinessOperation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of manufacturing operation
	OperationType *CodeableConcept `json:"operationType,omitempty"`
	// Regulatory authorization reference number
	AuthorisationReferenceNumber *Identifier `json:"authorisationReferenceNumber,omitempty"`
	// Regulatory authorization date
	EffectiveDate *primitives.DateTime `json:"effectiveDate,omitempty"`
	// To indicate if this proces is commercially confidential
	ConfidentialityIndicator *CodeableConcept `json:"confidentialityIndicator,omitempty"`
	// The manufacturer or establishment associated with the process
	Manufacturer []Reference `json:"manufacturer,omitempty"`
	// A regulator which oversees the operation
	Regulator *Reference `json:"regulator,omitempty"`
}

// MedicinalProductSpecialDesignation represents a FHIR BackboneElement for MedicinalProduct.specialDesignation.
type MedicinalProductSpecialDesignation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifier for the designation, or procedure number
	Identifier []Identifier `json:"identifier,omitempty"`
	// The type of special designation, e.g. orphan drug, minor use
	Type *CodeableConcept `json:"type,omitempty"`
	// The intended use of the product, e.g. prevention, treatment
	IntendedUse *CodeableConcept `json:"intendedUse,omitempty"`
	// Condition for which the medicinal use applies
	Indication *any `json:"indication,omitempty"`
	// For example granted, pending, expired or withdrawn
	Status *CodeableConcept `json:"status,omitempty"`
	// Date when the designation was granted
	Date *primitives.DateTime `json:"date,omitempty"`
	// Animal species for which this applies
	Species *CodeableConcept `json:"species,omitempty"`
}

// MedicinalProduct represents a FHIR MedicinalProduct.
type MedicinalProduct struct {
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
	// Business identifier for this product. Could be an MPID
	Identifier []Identifier `json:"identifier,omitempty"`
	// Regulatory type, e.g. Investigational or Authorized
	Type *CodeableConcept `json:"type,omitempty"`
	// If this medicine applies to human or veterinary uses
	Domain *Coding `json:"domain,omitempty"`
	// The dose form for a single part product, or combined form of a multiple part product
	CombinedPharmaceuticalDoseForm *CodeableConcept `json:"combinedPharmaceuticalDoseForm,omitempty"`
	// The legal status of supply of the medicinal product as classified by the regulator
	LegalStatusOfSupply *CodeableConcept `json:"legalStatusOfSupply,omitempty"`
	// Whether the Medicinal Product is subject to additional monitoring for regulatory reasons
	AdditionalMonitoringIndicator *CodeableConcept `json:"additionalMonitoringIndicator,omitempty"`
	// Whether the Medicinal Product is subject to special measures for regulatory reasons
	SpecialMeasures []string `json:"specialMeasures,omitempty"`
	// If authorised for use in children
	PaediatricUseIndicator *CodeableConcept `json:"paediatricUseIndicator,omitempty"`
	// Allows the product to be classified by various systems
	ProductClassification []CodeableConcept `json:"productClassification,omitempty"`
	// Marketing status of the medicinal product, in contrast to marketing authorizaton
	MarketingStatus []MarketingStatus `json:"marketingStatus,omitempty"`
	// Pharmaceutical aspects of product
	PharmaceuticalProduct []Reference `json:"pharmaceuticalProduct,omitempty"`
	// Package representation for the product
	PackagedMedicinalProduct []Reference `json:"packagedMedicinalProduct,omitempty"`
	// Supporting documentation, typically for regulatory submission
	AttachedDocument []Reference `json:"attachedDocument,omitempty"`
	// A master file for to the medicinal product (e.g. Pharmacovigilance System Master File)
	MasterFile []Reference `json:"masterFile,omitempty"`
	// A product specific contact, person (in a role), or an organization
	Contact []Reference `json:"contact,omitempty"`
	// Clinical trials or studies that this product is involved in
	ClinicalTrial []Reference `json:"clinicalTrial,omitempty"`
	// The product's name, including full name and possibly coded parts
	Name []MedicinalProductName `json:"name,omitempty"`
	// Reference to another product, e.g. for linking authorised to investigational product
	CrossReference []Identifier `json:"crossReference,omitempty"`
	// An operation applied to the product, for manufacturing or adminsitrative purpose
	ManufacturingBusinessOperation []MedicinalProductManufacturingBusinessOperation `json:"manufacturingBusinessOperation,omitempty"`
	// Indicates if the medicinal product has an orphan designation for the treatment of a rare disease
	SpecialDesignation []MedicinalProductSpecialDesignation `json:"specialDesignation,omitempty"`
}
