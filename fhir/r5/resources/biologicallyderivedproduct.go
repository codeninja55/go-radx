package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeBiologicallyDerivedProduct is the FHIR resource type name for BiologicallyDerivedProduct.
const ResourceTypeBiologicallyDerivedProduct = "BiologicallyDerivedProduct"

// BiologicallyDerivedProductCollection represents a FHIR BackboneElement for BiologicallyDerivedProduct.collection.
type BiologicallyDerivedProductCollection struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Individual performing collection
	Collector *Reference `json:"collector,omitempty"`
	// The patient who underwent the medical procedure to collect the product or the organization that facilitated the collection
	Source *Reference `json:"source,omitempty"`
	// Time of product collection
	Collected *any `json:"collected,omitempty"`
}

// BiologicallyDerivedProductProperty represents a FHIR BackboneElement for BiologicallyDerivedProduct.property.
type BiologicallyDerivedProductProperty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code that specifies the property
	Type CodeableConcept `json:"type"`
	// Property values
	Value any `json:"value"`
}

// BiologicallyDerivedProduct represents a FHIR BiologicallyDerivedProduct.
type BiologicallyDerivedProduct struct {
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
	// organ | tissue | fluid | cells | biologicalAgent
	ProductCategory *Coding `json:"productCategory,omitempty"`
	// A code that identifies the kind of this biologically derived product
	ProductCode *CodeableConcept `json:"productCode,omitempty"`
	// The parent biologically-derived product
	Parent []Reference `json:"parent,omitempty"`
	// Request to obtain and/or infuse this product
	Request []Reference `json:"request,omitempty"`
	// Instance identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// An identifier that supports traceability to the event during which material in this product from one or more biological entities was obtained or pooled
	BiologicalSourceEvent *Identifier `json:"biologicalSourceEvent,omitempty"`
	// Processing facilities responsible for the labeling and distribution of this biologically derived product
	ProcessingFacility []Reference `json:"processingFacility,omitempty"`
	// A unique identifier for an aliquot of a product
	Division *string `json:"division,omitempty"`
	// available | unavailable
	ProductStatus *Coding `json:"productStatus,omitempty"`
	// Date, and where relevant time, of expiration
	ExpirationDate *primitives.DateTime `json:"expirationDate,omitempty"`
	// How this product was collected
	Collection *BiologicallyDerivedProductCollection `json:"collection,omitempty"`
	// Product storage temperature requirements
	StorageTempRequirements *Range `json:"storageTempRequirements,omitempty"`
	// A property that is specific to this BiologicallyDerviedProduct instance
	Property []BiologicallyDerivedProductProperty `json:"property,omitempty"`
}
