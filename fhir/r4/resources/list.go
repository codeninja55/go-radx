package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeList is the FHIR resource type name for List.
const ResourceTypeList = "List"

// ListEntry represents a FHIR BackboneElement for List.entry.
type ListEntry struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Status/Workflow information about this item
	Flag *CodeableConcept `json:"flag,omitempty"`
	// If this item is actually marked as deleted
	Deleted *bool `json:"deleted,omitempty"`
	// When item added to list
	Date *primitives.DateTime `json:"date,omitempty"`
	// Actual entry
	Item Reference `json:"item"`
}

// List represents a FHIR List.
type List struct {
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
	// Business identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// current | retired | entered-in-error
	Status string `json:"status"`
	// working | snapshot | changes
	Mode string `json:"mode"`
	// Descriptive name for the list
	Title *string `json:"title,omitempty"`
	// What the purpose of this list is
	Code *CodeableConcept `json:"code,omitempty"`
	// If all resources have the same subject
	Subject *Reference `json:"subject,omitempty"`
	// Context in which list created
	Encounter *Reference `json:"encounter,omitempty"`
	// When the list was prepared
	Date *primitives.DateTime `json:"date,omitempty"`
	// Who and/or what defined the list contents (aka Author)
	Source *Reference `json:"source,omitempty"`
	// What order the list has
	OrderedBy *CodeableConcept `json:"orderedBy,omitempty"`
	// Comments about the list
	Note []Annotation `json:"note,omitempty"`
	// Entries in the list
	Entry []ListEntry `json:"entry,omitempty"`
	// Why list is empty
	EmptyReason *CodeableConcept `json:"emptyReason,omitempty"`
}
