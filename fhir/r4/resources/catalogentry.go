package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeCatalogEntry is the FHIR resource type name for CatalogEntry.
const ResourceTypeCatalogEntry = "CatalogEntry"

// CatalogEntryRelatedEntry represents a FHIR BackboneElement for CatalogEntry.relatedEntry.
type CatalogEntryRelatedEntry struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// triggers | is-replaced-by
	Relationtype string `json:"relationtype"`
	// The reference to the related item
	Item Reference `json:"item"`
}

// CatalogEntry represents a FHIR CatalogEntry.
type CatalogEntry struct {
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
	// Unique identifier of the catalog item
	Identifier []Identifier `json:"identifier,omitempty"`
	// The type of item - medication, device, service, protocol or other
	Type *CodeableConcept `json:"type,omitempty"`
	// Whether the entry represents an orderable item
	Orderable bool `json:"orderable"`
	// The item that is being defined
	ReferencedItem Reference `json:"referencedItem"`
	// Any additional identifier(s) for the catalog item, in the same granularity or concept
	AdditionalIdentifier []Identifier `json:"additionalIdentifier,omitempty"`
	// Classification (category or class) of the item entry
	Classification []CodeableConcept `json:"classification,omitempty"`
	// draft | active | retired | unknown
	Status *string `json:"status,omitempty"`
	// The time period in which this catalog entry is expected to be active
	ValidityPeriod *Period `json:"validityPeriod,omitempty"`
	// The date until which this catalog entry is expected to be active
	ValidTo *primitives.DateTime `json:"validTo,omitempty"`
	// When was this catalog last updated
	LastUpdated *primitives.DateTime `json:"lastUpdated,omitempty"`
	// Additional characteristics of the catalog entry
	AdditionalCharacteristic []CodeableConcept `json:"additionalCharacteristic,omitempty"`
	// Additional classification of the catalog entry
	AdditionalClassification []CodeableConcept `json:"additionalClassification,omitempty"`
	// An item that this catalog entry is related to
	RelatedEntry []CatalogEntryRelatedEntry `json:"relatedEntry,omitempty"`
}
