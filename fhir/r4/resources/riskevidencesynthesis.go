package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeRiskEvidenceSynthesis is the FHIR resource type name for RiskEvidenceSynthesis.
const ResourceTypeRiskEvidenceSynthesis = "RiskEvidenceSynthesis"

// RiskEvidenceSynthesisSampleSize represents a FHIR BackboneElement for RiskEvidenceSynthesis.sampleSize.
type RiskEvidenceSynthesisSampleSize struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of sample size
	Description *string `json:"description,omitempty"`
	// How many studies?
	NumberOfStudies *int `json:"numberOfStudies,omitempty"`
	// How many participants?
	NumberOfParticipants *int `json:"numberOfParticipants,omitempty"`
}

// RiskEvidenceSynthesisRiskEstimatePrecisionEstimate represents a FHIR BackboneElement for RiskEvidenceSynthesis.riskEstimate.precisionEstimate.
type RiskEvidenceSynthesisRiskEstimatePrecisionEstimate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of precision estimate
	Type *CodeableConcept `json:"type,omitempty"`
	// Level of confidence interval
	Level *float64 `json:"level,omitempty"`
	// Lower bound
	From *float64 `json:"from,omitempty"`
	// Upper bound
	To *float64 `json:"to,omitempty"`
}

// RiskEvidenceSynthesisRiskEstimate represents a FHIR BackboneElement for RiskEvidenceSynthesis.riskEstimate.
type RiskEvidenceSynthesisRiskEstimate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of risk estimate
	Description *string `json:"description,omitempty"`
	// Type of risk estimate
	Type *CodeableConcept `json:"type,omitempty"`
	// Point estimate
	Value *float64 `json:"value,omitempty"`
	// What unit is the outcome described in?
	UnitOfMeasure *CodeableConcept `json:"unitOfMeasure,omitempty"`
	// Sample size for group measured
	DenominatorCount *int `json:"denominatorCount,omitempty"`
	// Number with the outcome
	NumeratorCount *int `json:"numeratorCount,omitempty"`
	// How precise the estimate is
	PrecisionEstimate []RiskEvidenceSynthesisRiskEstimatePrecisionEstimate `json:"precisionEstimate,omitempty"`
}

// RiskEvidenceSynthesisCertaintyCertaintySubcomponent represents a FHIR BackboneElement for RiskEvidenceSynthesis.certainty.certaintySubcomponent.
type RiskEvidenceSynthesisCertaintyCertaintySubcomponent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of subcomponent of certainty rating
	Type *CodeableConcept `json:"type,omitempty"`
	// Subcomponent certainty rating
	Rating []CodeableConcept `json:"rating,omitempty"`
	// Used for footnotes or explanatory notes
	Note []Annotation `json:"note,omitempty"`
}

// RiskEvidenceSynthesisCertainty represents a FHIR BackboneElement for RiskEvidenceSynthesis.certainty.
type RiskEvidenceSynthesisCertainty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Certainty rating
	Rating []CodeableConcept `json:"rating,omitempty"`
	// Used for footnotes or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// A component that contributes to the overall certainty
	CertaintySubcomponent []RiskEvidenceSynthesisCertaintyCertaintySubcomponent `json:"certaintySubcomponent,omitempty"`
}

// RiskEvidenceSynthesis represents a FHIR RiskEvidenceSynthesis.
type RiskEvidenceSynthesis struct {
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
	// Canonical identifier for this risk evidence synthesis, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the risk evidence synthesis
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the risk evidence synthesis
	Version *string `json:"version,omitempty"`
	// Name for this risk evidence synthesis (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this risk evidence synthesis (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the risk evidence synthesis
	Description *string `json:"description,omitempty"`
	// Used for footnotes or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for risk evidence synthesis (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the risk evidence synthesis was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the risk evidence synthesis was last reviewed
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the risk evidence synthesis is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// The category of the EffectEvidenceSynthesis, such as Education, Treatment, Assessment, etc.
	Topic []CodeableConcept `json:"topic,omitempty"`
	// Who authored the content
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the content
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the content
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the content
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations, etc.
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Type of synthesis
	SynthesisType *CodeableConcept `json:"synthesisType,omitempty"`
	// Type of study
	StudyType *CodeableConcept `json:"studyType,omitempty"`
	// What population?
	Population Reference `json:"population"`
	// What exposure?
	Exposure *Reference `json:"exposure,omitempty"`
	// What outcome?
	Outcome Reference `json:"outcome"`
	// What sample size was involved?
	SampleSize *RiskEvidenceSynthesisSampleSize `json:"sampleSize,omitempty"`
	// What was the estimated risk
	RiskEstimate *RiskEvidenceSynthesisRiskEstimate `json:"riskEstimate,omitempty"`
	// How certain is the risk
	Certainty []RiskEvidenceSynthesisCertainty `json:"certainty,omitempty"`
}
