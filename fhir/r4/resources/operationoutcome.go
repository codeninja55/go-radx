package resources

// ResourceTypeOperationOutcome is the FHIR resource type name for OperationOutcome.
const ResourceTypeOperationOutcome = "OperationOutcome"

// OperationOutcomeIssue represents a FHIR BackboneElement for OperationOutcome.issue.
type OperationOutcomeIssue struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// fatal | error | warning | information
	Severity string `json:"severity"`
	// Error or warning code
	Code string `json:"code"`
	// Additional details about the error
	Details *CodeableConcept `json:"details,omitempty"`
	// Additional diagnostic information about the issue
	Diagnostics *string `json:"diagnostics,omitempty"`
	// Deprecated: Path of element(s) related to issue
	Location []string `json:"location,omitempty"`
	// FHIRPath of element(s) related to issue
	Expression []string `json:"expression,omitempty"`
}

// OperationOutcome represents a FHIR OperationOutcome.
type OperationOutcome struct {
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
	// A single issue associated with the action
	Issue []OperationOutcomeIssue `json:"issue,omitempty"`
}
