package resources

// ResourceTypeLinkage is the FHIR resource type name for Linkage.
const ResourceTypeLinkage = "Linkage"

// LinkageItem represents a FHIR BackboneElement for Linkage.item.
type LinkageItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// source | alternate | historical
	Type string `json:"type"`
	// Resource being linked
	Resource Reference `json:"resource"`
}

// Linkage represents a FHIR Linkage.
type Linkage struct {
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
	// Whether this linkage assertion is active or not
	Active *bool `json:"active,omitempty"`
	// Who is responsible for linkages
	Author *Reference `json:"author,omitempty"`
	// Item to be linked
	Item []LinkageItem `json:"item,omitempty"`
}
