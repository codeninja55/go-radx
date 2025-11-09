package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedicinalProductDefinition is the FHIR resource type name for MedicinalProductDefinition.
const ResourceTypeMedicinalProductDefinition = "MedicinalProductDefinition"

// MedicinalProductDefinitionContact represents a FHIR BackboneElement for MedicinalProductDefinition.contact.
type MedicinalProductDefinitionContact struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Allows the contact to be classified, for example QPPV, Pharmacovigilance Enquiry Information
	Type *CodeableConcept `json:"type,omitempty"`
	// A product specific contact, person (in a role), or an organization
	Contact Reference `json:"contact"`
}

// MedicinalProductDefinitionNamePart represents a FHIR BackboneElement for MedicinalProductDefinition.name.part.
type MedicinalProductDefinitionNamePart struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A fragment of a product name
	Part string `json:"part"`
	// Identifying type for this part of the name (e.g. strength part)
	Type CodeableConcept `json:"type"`
}

// MedicinalProductDefinitionNameUsage represents a FHIR BackboneElement for MedicinalProductDefinition.name.usage.
type MedicinalProductDefinitionNameUsage struct {
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

// MedicinalProductDefinitionName represents a FHIR BackboneElement for MedicinalProductDefinition.name.
type MedicinalProductDefinitionName struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The full product name
	ProductName string `json:"productName"`
	// Type of product name, such as rINN, BAN, Proprietary, Non-Proprietary
	Type *CodeableConcept `json:"type,omitempty"`
	// Coding words or phrases of the name
	Part []MedicinalProductDefinitionNamePart `json:"part,omitempty"`
	// Country and jurisdiction where the name applies
	Usage []MedicinalProductDefinitionNameUsage `json:"usage,omitempty"`
}

// MedicinalProductDefinitionCrossReference represents a FHIR BackboneElement for MedicinalProductDefinition.crossReference.
type MedicinalProductDefinitionCrossReference struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Reference to another product, e.g. for linking authorised to investigational product
	Product CodeableReference `json:"product"`
	// The type of relationship, for instance branded to generic or virtual to actual product
	Type *CodeableConcept `json:"type,omitempty"`
}

// MedicinalProductDefinitionOperation represents a FHIR BackboneElement for MedicinalProductDefinition.operation.
type MedicinalProductDefinitionOperation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of manufacturing operation e.g. manufacturing itself, re-packaging
	Type *CodeableReference `json:"type,omitempty"`
	// Date range of applicability
	EffectiveDate *Period `json:"effectiveDate,omitempty"`
	// The organization responsible for the particular process, e.g. the manufacturer or importer
	Organization []Reference `json:"organization,omitempty"`
	// Specifies whether this process is considered proprietary or confidential
	ConfidentialityIndicator *CodeableConcept `json:"confidentialityIndicator,omitempty"`
}

// MedicinalProductDefinitionCharacteristic represents a FHIR BackboneElement for MedicinalProductDefinition.characteristic.
type MedicinalProductDefinitionCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A code expressing the type of characteristic
	Type CodeableConcept `json:"type"`
	// A value for the characteristic
	Value *any `json:"value,omitempty"`
}

// MedicinalProductDefinition represents a FHIR MedicinalProductDefinition.
type MedicinalProductDefinition struct {
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
	Domain *CodeableConcept `json:"domain,omitempty"`
	// A business identifier relating to a specific version of the product
	Version *string `json:"version,omitempty"`
	// The status within the lifecycle of this product record
	Status *CodeableConcept `json:"status,omitempty"`
	// The date at which the given status became applicable
	StatusDate *primitives.DateTime `json:"statusDate,omitempty"`
	// General description of this product
	Description *string `json:"description,omitempty"`
	// The dose form for a single part product, or combined form of a multiple part product
	CombinedPharmaceuticalDoseForm *CodeableConcept `json:"combinedPharmaceuticalDoseForm,omitempty"`
	// The path by which the product is taken into or makes contact with the body
	Route []CodeableConcept `json:"route,omitempty"`
	// Description of indication(s) for this product, used when structured indications are not required
	Indication *string `json:"indication,omitempty"`
	// The legal status of supply of the medicinal product as classified by the regulator
	LegalStatusOfSupply *CodeableConcept `json:"legalStatusOfSupply,omitempty"`
	// Whether the Medicinal Product is subject to additional monitoring for regulatory reasons
	AdditionalMonitoringIndicator *CodeableConcept `json:"additionalMonitoringIndicator,omitempty"`
	// Whether the Medicinal Product is subject to special measures for regulatory reasons
	SpecialMeasures []CodeableConcept `json:"specialMeasures,omitempty"`
	// If authorised for use in children
	PediatricUseIndicator *CodeableConcept `json:"pediatricUseIndicator,omitempty"`
	// Allows the product to be classified by various systems
	Classification []CodeableConcept `json:"classification,omitempty"`
	// Marketing status of the medicinal product, in contrast to marketing authorization
	MarketingStatus []MarketingStatus `json:"marketingStatus,omitempty"`
	// Package type for the product
	PackagedMedicinalProduct []CodeableConcept `json:"packagedMedicinalProduct,omitempty"`
	// Types of medicinal manufactured items and/or devices that this product consists of, such as tablets, capsule, or syringes
	ComprisedOf []Reference `json:"comprisedOf,omitempty"`
	// The ingredients of this medicinal product - when not detailed in other resources
	Ingredient []CodeableConcept `json:"ingredient,omitempty"`
	// Any component of the drug product which is not the chemical entity defined as the drug substance, or an excipient in the drug product
	Impurity []CodeableReference `json:"impurity,omitempty"`
	// Additional documentation about the medicinal product
	AttachedDocument []Reference `json:"attachedDocument,omitempty"`
	// A master file for the medicinal product (e.g. Pharmacovigilance System Master File)
	MasterFile []Reference `json:"masterFile,omitempty"`
	// A product specific contact, person (in a role), or an organization
	Contact []MedicinalProductDefinitionContact `json:"contact,omitempty"`
	// Clinical trials or studies that this product is involved in
	ClinicalTrial []Reference `json:"clinicalTrial,omitempty"`
	// A code that this product is known by, within some formal terminology
	Code []Coding `json:"code,omitempty"`
	// The product's name, including full name and possibly coded parts
	Name []MedicinalProductDefinitionName `json:"name,omitempty"`
	// Reference to another product, e.g. for linking authorised to investigational product
	CrossReference []MedicinalProductDefinitionCrossReference `json:"crossReference,omitempty"`
	// A manufacturing or administrative process for the medicinal product
	Operation []MedicinalProductDefinitionOperation `json:"operation,omitempty"`
	// Key product features such as "sugar free", "modified release"
	Characteristic []MedicinalProductDefinitionCharacteristic `json:"characteristic,omitempty"`
}
