package resources

// ResourceTypeFormularyItem is the FHIR resource type name for FormularyItem.
const ResourceTypeFormularyItem = "FormularyItem"

// FormularyItem represents a FHIR FormularyItem.
type FormularyItem struct {
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
	// Business identifier for this formulary item
	Identifier []Identifier `json:"identifier,omitempty"`
	// Codes that identify this formulary item
	Code *CodeableConcept `json:"code,omitempty"`
	// active | entered-in-error | inactive
	Status *string `json:"status,omitempty"`
}
