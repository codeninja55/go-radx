package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDetectedIssue is the FHIR resource type name for DetectedIssue.
const ResourceTypeDetectedIssue = "DetectedIssue"

// DetectedIssueEvidence represents a FHIR BackboneElement for DetectedIssue.evidence.
type DetectedIssueEvidence struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Manifestation
	Code []CodeableConcept `json:"code,omitempty"`
	// Supporting information
	Detail []Reference `json:"detail,omitempty"`
}

// DetectedIssueMitigation represents a FHIR BackboneElement for DetectedIssue.mitigation.
type DetectedIssueMitigation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What mitigation?
	Action CodeableConcept `json:"action"`
	// Date committed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Who is committing?
	Author *Reference `json:"author,omitempty"`
	// Additional notes about the mitigation
	Note []Annotation `json:"note,omitempty"`
}

// DetectedIssue represents a FHIR DetectedIssue.
type DetectedIssue struct {
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
	// Unique id for the detected issue
	Identifier []Identifier `json:"identifier,omitempty"`
	// preliminary | final | entered-in-error | mitigated
	Status string `json:"status"`
	// Type of detected issue, e.g. drug-drug, duplicate therapy, etc
	Category []CodeableConcept `json:"category,omitempty"`
	// Specific type of detected issue, e.g. drug-drug, duplicate therapy, etc
	Code *CodeableConcept `json:"code,omitempty"`
	// high | moderate | low
	Severity *string `json:"severity,omitempty"`
	// Associated subject
	Subject *Reference `json:"subject,omitempty"`
	// Encounter detected issue is part of
	Encounter *Reference `json:"encounter,omitempty"`
	// When identified
	Identified *any `json:"identified,omitempty"`
	// The provider or device that identified the issue
	Author *Reference `json:"author,omitempty"`
	// Problem resource
	Implicated []Reference `json:"implicated,omitempty"`
	// Supporting evidence
	Evidence []DetectedIssueEvidence `json:"evidence,omitempty"`
	// Description and context
	Detail *string `json:"detail,omitempty"`
	// Authority for issue
	Reference *string `json:"reference,omitempty"`
	// Step taken to address
	Mitigation []DetectedIssueMitigation `json:"mitigation,omitempty"`
}
