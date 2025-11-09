package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeInventoryItem is the FHIR resource type name for InventoryItem.
const ResourceTypeInventoryItem = "InventoryItem"

// InventoryItemName represents a FHIR BackboneElement for InventoryItem.name.
type InventoryItemName struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of name e.g. 'brand-name', 'functional-name', 'common-name'
	NameType Coding `json:"nameType"`
	// The language used to express the item name
	Language string `json:"language"`
	// The name or designation of the item
	Name string `json:"name"`
}

// InventoryItemResponsibleOrganization represents a FHIR BackboneElement for InventoryItem.responsibleOrganization.
type InventoryItemResponsibleOrganization struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The role of the organization e.g. manufacturer, distributor, or other
	Role CodeableConcept `json:"role"`
	// An organization that is associated with the item
	Organization Reference `json:"organization"`
}

// InventoryItemDescription represents a FHIR BackboneElement for InventoryItem.description.
type InventoryItemDescription struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The language that is used in the item description
	Language *string `json:"language,omitempty"`
	// Textual description of the item
	Description *string `json:"description,omitempty"`
}

// InventoryItemAssociation represents a FHIR BackboneElement for InventoryItem.association.
type InventoryItemAssociation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The type of association between the device and the other item
	AssociationType CodeableConcept `json:"associationType"`
	// The related item or product
	RelatedItem Reference `json:"relatedItem"`
	// The quantity of the product in this product
	Quantity Ratio `json:"quantity"`
}

// InventoryItemCharacteristic represents a FHIR BackboneElement for InventoryItem.characteristic.
type InventoryItemCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The characteristic that is being defined
	CharacteristicType CodeableConcept `json:"characteristicType"`
	// The value of the attribute
	Value any `json:"value"`
}

// InventoryItemInstance represents a FHIR BackboneElement for InventoryItem.instance.
type InventoryItemInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The identifier for the physical instance, typically a serial number
	Identifier []Identifier `json:"identifier,omitempty"`
	// The lot or batch number of the item
	LotNumber *string `json:"lotNumber,omitempty"`
	// The expiry date or date and time for the product
	Expiry *primitives.DateTime `json:"expiry,omitempty"`
	// The subject that the item is associated with
	Subject *Reference `json:"subject,omitempty"`
	// The location that the item is associated with
	Location *Reference `json:"location,omitempty"`
}

// InventoryItem represents a FHIR InventoryItem.
type InventoryItem struct {
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
	// Business identifier for the inventory item
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | inactive | entered-in-error | unknown
	Status string `json:"status"`
	// Category or class of the item
	Category []CodeableConcept `json:"category,omitempty"`
	// Code designating the specific type of item
	Code []CodeableConcept `json:"code,omitempty"`
	// The item name(s) - the brand name, or common name, functional name, generic name or others
	Name []InventoryItemName `json:"name,omitempty"`
	// Organization(s) responsible for the product
	ResponsibleOrganization []InventoryItemResponsibleOrganization `json:"responsibleOrganization,omitempty"`
	// Descriptive characteristics of the item
	Description *InventoryItemDescription `json:"description,omitempty"`
	// The usage status like recalled, in use, discarded
	InventoryStatus []CodeableConcept `json:"inventoryStatus,omitempty"`
	// The base unit of measure - the unit in which the product is used or counted
	BaseUnit *CodeableConcept `json:"baseUnit,omitempty"`
	// Net content or amount present in the item
	NetContent *Quantity `json:"netContent,omitempty"`
	// Association with other items or products
	Association []InventoryItemAssociation `json:"association,omitempty"`
	// Characteristic of the item
	Characteristic []InventoryItemCharacteristic `json:"characteristic,omitempty"`
	// Instances or occurrences of the product
	Instance *InventoryItemInstance `json:"instance,omitempty"`
	// Link to a product resource used in clinical workflows
	ProductReference *Reference `json:"productReference,omitempty"`
}
