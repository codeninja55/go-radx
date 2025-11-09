package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeInventoryReport is the FHIR resource type name for InventoryReport.
const ResourceTypeInventoryReport = "InventoryReport"

// InventoryReportInventoryListingItem represents a FHIR BackboneElement for InventoryReport.inventoryListing.item.
type InventoryReportInventoryListingItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The inventory category or classification of the items being reported
	Category *CodeableConcept `json:"category,omitempty"`
	// The quantity of the item or items being reported
	Quantity Quantity `json:"quantity"`
	// The code or reference to the item type
	Item CodeableReference `json:"item"`
}

// InventoryReportInventoryListing represents a FHIR BackboneElement for InventoryReport.inventoryListing.
type InventoryReportInventoryListing struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Location of the inventory items
	Location *Reference `json:"location,omitempty"`
	// The status of the items that are being reported
	ItemStatus *CodeableConcept `json:"itemStatus,omitempty"`
	// The date and time when the items were counted
	CountingDateTime *primitives.DateTime `json:"countingDateTime,omitempty"`
	// The item or items in this listing
	Item []InventoryReportInventoryListingItem `json:"item,omitempty"`
}

// InventoryReport represents a FHIR InventoryReport.
type InventoryReport struct {
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
	// Business identifier for the report
	Identifier []Identifier `json:"identifier,omitempty"`
	// draft | requested | active | entered-in-error
	Status string `json:"status"`
	// snapshot | difference
	CountType string `json:"countType"`
	// addition | subtraction
	OperationType *CodeableConcept `json:"operationType,omitempty"`
	// The reason for this count - regular count, ad-hoc count, new arrivals, etc
	OperationTypeReason *CodeableConcept `json:"operationTypeReason,omitempty"`
	// When the report has been submitted
	ReportedDateTime primitives.DateTime `json:"reportedDateTime"`
	// Who submits the report
	Reporter *Reference `json:"reporter,omitempty"`
	// The period the report refers to
	ReportingPeriod *Period `json:"reportingPeriod,omitempty"`
	// An inventory listing section (grouped by any of the attributes)
	InventoryListing []InventoryReportInventoryListing `json:"inventoryListing,omitempty"`
	// A note associated with the InventoryReport
	Note []Annotation `json:"note,omitempty"`
}
