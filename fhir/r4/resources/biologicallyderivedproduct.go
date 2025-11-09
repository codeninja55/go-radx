package resources

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
	// Who is product from
	Source *Reference `json:"source,omitempty"`
	// Time of product collection
	Collected *any `json:"collected,omitempty"`
}

// BiologicallyDerivedProductProcessing represents a FHIR BackboneElement for BiologicallyDerivedProduct.processing.
type BiologicallyDerivedProductProcessing struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of of processing
	Description *string `json:"description,omitempty"`
	// Procesing code
	Procedure *CodeableConcept `json:"procedure,omitempty"`
	// Substance added during processing
	Additive *Reference `json:"additive,omitempty"`
	// Time of processing
	Time *any `json:"time,omitempty"`
}

// BiologicallyDerivedProductManipulation represents a FHIR BackboneElement for BiologicallyDerivedProduct.manipulation.
type BiologicallyDerivedProductManipulation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of manipulation
	Description *string `json:"description,omitempty"`
	// Time of manipulation
	Time *any `json:"time,omitempty"`
}

// BiologicallyDerivedProductStorage represents a FHIR BackboneElement for BiologicallyDerivedProduct.storage.
type BiologicallyDerivedProductStorage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of storage
	Description *string `json:"description,omitempty"`
	// Storage temperature
	Temperature *float64 `json:"temperature,omitempty"`
	// farenheit | celsius | kelvin
	Scale *string `json:"scale,omitempty"`
	// Storage timeperiod
	Duration *Period `json:"duration,omitempty"`
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
	// External ids for this item
	Identifier []Identifier `json:"identifier,omitempty"`
	// organ | tissue | fluid | cells | biologicalAgent
	ProductCategory *string `json:"productCategory,omitempty"`
	// What this biologically derived product is
	ProductCode *CodeableConcept `json:"productCode,omitempty"`
	// available | unavailable
	Status *string `json:"status,omitempty"`
	// Procedure request
	Request []Reference `json:"request,omitempty"`
	// The amount of this biologically derived product
	Quantity *int `json:"quantity,omitempty"`
	// BiologicallyDerivedProduct parent
	Parent []Reference `json:"parent,omitempty"`
	// How this product was collected
	Collection *BiologicallyDerivedProductCollection `json:"collection,omitempty"`
	// Any processing of the product during collection
	Processing []BiologicallyDerivedProductProcessing `json:"processing,omitempty"`
	// Any manipulation of product post-collection
	Manipulation *BiologicallyDerivedProductManipulation `json:"manipulation,omitempty"`
	// Product storage
	Storage []BiologicallyDerivedProductStorage `json:"storage,omitempty"`
}
