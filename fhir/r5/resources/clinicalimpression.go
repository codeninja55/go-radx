package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeClinicalImpression is the FHIR resource type name for ClinicalImpression.
const ResourceTypeClinicalImpression = "ClinicalImpression"

// ClinicalImpressionFinding represents a FHIR BackboneElement for ClinicalImpression.finding.
type ClinicalImpressionFinding struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What was found
	Item *CodeableReference `json:"item,omitempty"`
	// Which investigations support finding
	Basis *string `json:"basis,omitempty"`
}

// ClinicalImpression represents a FHIR ClinicalImpression.
type ClinicalImpression struct {
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
	// preparation | in-progress | not-done | on-hold | stopped | completed | entered-in-error | unknown
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// Why/how the assessment was performed
	Description *string `json:"description,omitempty"`
	// Patient or group assessed
	Subject Reference `json:"subject"`
	// The Encounter during which this ClinicalImpression was created
	Encounter *Reference `json:"encounter,omitempty"`
	// Time of assessment
	Effective *any `json:"effective,omitempty"`
	// When the assessment was documented
	Date *primitives.DateTime `json:"date,omitempty"`
	// The clinician performing the assessment
	Performer *Reference `json:"performer,omitempty"`
	// Reference to last assessment
	Previous *Reference `json:"previous,omitempty"`
	// Relevant impressions of patient state
	Problem []Reference `json:"problem,omitempty"`
	// Change in the status/pattern of a subject's condition since previously assessed, such as worsening, improving, or no change
	ChangePattern *CodeableConcept `json:"changePattern,omitempty"`
	// Clinical Protocol followed
	Protocol []string `json:"protocol,omitempty"`
	// Summary of the assessment
	Summary *string `json:"summary,omitempty"`
	// Possible or likely findings and diagnoses
	Finding []ClinicalImpressionFinding `json:"finding,omitempty"`
	// Estimate of likely outcome
	PrognosisCodeableConcept []CodeableConcept `json:"prognosisCodeableConcept,omitempty"`
	// RiskAssessment expressing likely outcome
	PrognosisReference []Reference `json:"prognosisReference,omitempty"`
	// Information supporting the clinical impression
	SupportingInfo []Reference `json:"supportingInfo,omitempty"`
	// Comments made about the ClinicalImpression
	Note []Annotation `json:"note,omitempty"`
}
