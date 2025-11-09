package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeEffectEvidenceSynthesis is the FHIR resource type name for EffectEvidenceSynthesis.
const ResourceTypeEffectEvidenceSynthesis = "EffectEvidenceSynthesis"

// EffectEvidenceSynthesisSampleSize represents a FHIR BackboneElement for EffectEvidenceSynthesis.sampleSize.
type EffectEvidenceSynthesisSampleSize struct {
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

// EffectEvidenceSynthesisResultsByExposure represents a FHIR BackboneElement for EffectEvidenceSynthesis.resultsByExposure.
type EffectEvidenceSynthesisResultsByExposure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of results by exposure
	Description *string `json:"description,omitempty"`
	// exposure | exposure-alternative
	ExposureState *string `json:"exposureState,omitempty"`
	// Variant exposure states
	VariantState *CodeableConcept `json:"variantState,omitempty"`
	// Risk evidence synthesis
	RiskEvidenceSynthesis Reference `json:"riskEvidenceSynthesis"`
}

// EffectEvidenceSynthesisEffectEstimatePrecisionEstimate represents a FHIR BackboneElement for EffectEvidenceSynthesis.effectEstimate.precisionEstimate.
type EffectEvidenceSynthesisEffectEstimatePrecisionEstimate struct {
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

// EffectEvidenceSynthesisEffectEstimate represents a FHIR BackboneElement for EffectEvidenceSynthesis.effectEstimate.
type EffectEvidenceSynthesisEffectEstimate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of effect estimate
	Description *string `json:"description,omitempty"`
	// Type of efffect estimate
	Type *CodeableConcept `json:"type,omitempty"`
	// Variant exposure states
	VariantState *CodeableConcept `json:"variantState,omitempty"`
	// Point estimate
	Value *float64 `json:"value,omitempty"`
	// What unit is the outcome described in?
	UnitOfMeasure *CodeableConcept `json:"unitOfMeasure,omitempty"`
	// How precise the estimate is
	PrecisionEstimate []EffectEvidenceSynthesisEffectEstimatePrecisionEstimate `json:"precisionEstimate,omitempty"`
}

// EffectEvidenceSynthesisCertaintyCertaintySubcomponent represents a FHIR BackboneElement for EffectEvidenceSynthesis.certainty.certaintySubcomponent.
type EffectEvidenceSynthesisCertaintyCertaintySubcomponent struct {
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

// EffectEvidenceSynthesisCertainty represents a FHIR BackboneElement for EffectEvidenceSynthesis.certainty.
type EffectEvidenceSynthesisCertainty struct {
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
	CertaintySubcomponent []EffectEvidenceSynthesisCertaintyCertaintySubcomponent `json:"certaintySubcomponent,omitempty"`
}

// EffectEvidenceSynthesis represents a FHIR EffectEvidenceSynthesis.
type EffectEvidenceSynthesis struct {
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
	// Canonical identifier for this effect evidence synthesis, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the effect evidence synthesis
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the effect evidence synthesis
	Version *string `json:"version,omitempty"`
	// Name for this effect evidence synthesis (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this effect evidence synthesis (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the effect evidence synthesis
	Description *string `json:"description,omitempty"`
	// Used for footnotes or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for effect evidence synthesis (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the effect evidence synthesis was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the effect evidence synthesis was last reviewed
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the effect evidence synthesis is expected to be used
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
	Exposure Reference `json:"exposure"`
	// What comparison exposure?
	ExposureAlternative Reference `json:"exposureAlternative"`
	// What outcome?
	Outcome Reference `json:"outcome"`
	// What sample size was involved?
	SampleSize *EffectEvidenceSynthesisSampleSize `json:"sampleSize,omitempty"`
	// What was the result per exposure?
	ResultsByExposure []EffectEvidenceSynthesisResultsByExposure `json:"resultsByExposure,omitempty"`
	// What was the estimated effect
	EffectEstimate []EffectEvidenceSynthesisEffectEstimate `json:"effectEstimate,omitempty"`
	// How certain is the effect
	Certainty []EffectEvidenceSynthesisCertainty `json:"certainty,omitempty"`
}
