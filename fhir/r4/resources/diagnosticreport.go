package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDiagnosticReport is the FHIR resource type name for DiagnosticReport.
const ResourceTypeDiagnosticReport = "DiagnosticReport"

// DiagnosticReportMedia represents a FHIR BackboneElement for DiagnosticReport.media.
type DiagnosticReportMedia struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Comment about the image (e.g. explanation)
	Comment *string `json:"comment,omitempty"`
	// Reference to the image source
	Link Reference `json:"link"`
}

// DiagnosticReport represents a FHIR DiagnosticReport.
type DiagnosticReport struct {
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
	// Business identifier for report
	Identifier []Identifier `json:"identifier,omitempty"`
	// What was requested
	BasedOn []Reference `json:"basedOn,omitempty"`
	// registered | partial | preliminary | final +
	Status string `json:"status"`
	// Service category
	Category []CodeableConcept `json:"category,omitempty"`
	// Name/Code for this diagnostic report
	Code CodeableConcept `json:"code"`
	// The subject of the report - usually, but not always, the patient
	Subject *Reference `json:"subject,omitempty"`
	// Health care event when test ordered
	Encounter *Reference `json:"encounter,omitempty"`
	// Clinically relevant time/time-period for report
	Effective *any `json:"effective,omitempty"`
	// DateTime this version was made
	Issued *primitives.Instant `json:"issued,omitempty"`
	// Responsible Diagnostic Service
	Performer []Reference `json:"performer,omitempty"`
	// Primary result interpreter
	ResultsInterpreter []Reference `json:"resultsInterpreter,omitempty"`
	// Specimens this report is based on
	Specimen []Reference `json:"specimen,omitempty"`
	// Observations
	Result []Reference `json:"result,omitempty"`
	// Reference to full details of imaging associated with the diagnostic report
	ImagingStudy []Reference `json:"imagingStudy,omitempty"`
	// Key images associated with this report
	Media []DiagnosticReportMedia `json:"media,omitempty"`
	// Clinical conclusion (interpretation) of test results
	Conclusion *string `json:"conclusion,omitempty"`
	// Codes for the clinical conclusion of test results
	ConclusionCode []CodeableConcept `json:"conclusionCode,omitempty"`
	// Entire report as issued
	PresentedForm []Attachment `json:"presentedForm,omitempty"`
}
