package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeClinicalImpression is the FHIR resource type name for ClinicalImpression.
const ResourceTypeClinicalImpression = "ClinicalImpression"

// ClinicalImpressionInvestigation represents a FHIR BackboneElement for ClinicalImpression.investigation.
type ClinicalImpressionInvestigation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A name/code for the set
	Code CodeableConcept `json:"code"`
	// Record of a specific investigation
	Item []Reference `json:"item,omitempty"`
}

// ClinicalImpressionFinding represents a FHIR BackboneElement for ClinicalImpression.finding.
type ClinicalImpressionFinding struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What was found
	ItemCodeableConcept *CodeableConcept `json:"itemCodeableConcept,omitempty"`
	// What was found
	ItemReference *Reference `json:"itemReference,omitempty"`
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
	// in-progress | completed | entered-in-error
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// Kind of assessment performed
	Code *CodeableConcept `json:"code,omitempty"`
	// Why/how the assessment was performed
	Description *string `json:"description,omitempty"`
	// Patient or group assessed
	Subject Reference `json:"subject"`
	// Encounter created as part of
	Encounter *Reference `json:"encounter,omitempty"`
	// Time of assessment
	Effective *any `json:"effective,omitempty"`
	// When the assessment was documented
	Date *primitives.DateTime `json:"date,omitempty"`
	// The clinician performing the assessment
	Assessor *Reference `json:"assessor,omitempty"`
	// Reference to last assessment
	Previous *Reference `json:"previous,omitempty"`
	// Relevant impressions of patient state
	Problem []Reference `json:"problem,omitempty"`
	// One or more sets of investigations (signs, symptoms, etc.)
	Investigation []ClinicalImpressionInvestigation `json:"investigation,omitempty"`
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
