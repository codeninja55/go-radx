package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeEvidence is the FHIR resource type name for Evidence.
const ResourceTypeEvidence = "Evidence"

// EvidenceVariableDefinition represents a FHIR BackboneElement for Evidence.variableDefinition.
type EvidenceVariableDefinition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// A text description or summary of the variable
	Description *string `json:"description,omitempty"`
	// Footnotes and/or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// population | subpopulation | exposure | referenceExposure | measuredVariable | confounder
	VariableRole CodeableConcept `json:"variableRole"`
	// Definition of the actual variable related to the statistic(s)
	Observed *Reference `json:"observed,omitempty"`
	// Definition of the intended variable related to the Evidence
	Intended *Reference `json:"intended,omitempty"`
	// low | moderate | high | exact
	DirectnessMatch *CodeableConcept `json:"directnessMatch,omitempty"`
}

// EvidenceStatisticSampleSize represents a FHIR BackboneElement for Evidence.statistic.sampleSize.
type EvidenceStatisticSampleSize struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Textual description of sample size for statistic
	Description *string `json:"description,omitempty"`
	// Footnote or explanatory note about the sample size
	Note []Annotation `json:"note,omitempty"`
	// Number of contributing studies
	NumberOfStudies *uint `json:"numberOfStudies,omitempty"`
	// Cumulative number of participants
	NumberOfParticipants *uint `json:"numberOfParticipants,omitempty"`
	// Number of participants with known results for measured variables
	KnownDataCount *uint `json:"knownDataCount,omitempty"`
}

// EvidenceStatisticAttributeEstimateAttributeEstimate represents a FHIR BackboneElement for Evidence.statistic.attributeEstimate.attributeEstimate.
type EvidenceStatisticAttributeEstimateAttributeEstimate struct {
}

// EvidenceStatisticAttributeEstimate represents a FHIR BackboneElement for Evidence.statistic.attributeEstimate.
type EvidenceStatisticAttributeEstimate struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Textual description of the attribute estimate
	Description *string `json:"description,omitempty"`
	// Footnote or explanatory note about the estimate
	Note []Annotation `json:"note,omitempty"`
	// The type of attribute estimate, e.g., confidence interval or p value
	Type *CodeableConcept `json:"type,omitempty"`
	// The singular quantity of the attribute estimate, for attribute estimates represented as single values; also used to report unit of measure
	Quantity *Quantity `json:"quantity,omitempty"`
	// Level of confidence interval, e.g., 0.95 for 95% confidence interval
	Level *float64 `json:"level,omitempty"`
	// Lower and upper bound values of the attribute estimate
	Range *Range `json:"range,omitempty"`
	// A nested attribute estimate; which is the attribute estimate of an attribute estimate
	AttributeEstimate []EvidenceStatisticAttributeEstimateAttributeEstimate `json:"attributeEstimate,omitempty"`
}

// EvidenceStatisticModelCharacteristicVariable represents a FHIR BackboneElement for Evidence.statistic.modelCharacteristic.variable.
type EvidenceStatisticModelCharacteristicVariable struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of the variable
	VariableDefinition Reference `json:"variableDefinition"`
	// continuous | dichotomous | ordinal | polychotomous
	Handling *string `json:"handling,omitempty"`
	// Description for grouping of ordinal or polychotomous variables
	ValueCategory []CodeableConcept `json:"valueCategory,omitempty"`
	// Discrete value for grouping of ordinal or polychotomous variables
	ValueQuantity []Quantity `json:"valueQuantity,omitempty"`
	// Range of values for grouping of ordinal or polychotomous variables
	ValueRange []Range `json:"valueRange,omitempty"`
}

// EvidenceStatisticModelCharacteristicAttributeEstimate represents a FHIR BackboneElement for Evidence.statistic.modelCharacteristic.attributeEstimate.
type EvidenceStatisticModelCharacteristicAttributeEstimate struct {
}

// EvidenceStatisticModelCharacteristic represents a FHIR BackboneElement for Evidence.statistic.modelCharacteristic.
type EvidenceStatisticModelCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Model specification
	Code CodeableConcept `json:"code"`
	// Numerical value to complete model specification
	Value *Quantity `json:"value,omitempty"`
	// A variable adjusted for in the adjusted analysis
	Variable []EvidenceStatisticModelCharacteristicVariable `json:"variable,omitempty"`
	// An attribute of the statistic used as a model characteristic
	AttributeEstimate []EvidenceStatisticModelCharacteristicAttributeEstimate `json:"attributeEstimate,omitempty"`
}

// EvidenceStatistic represents a FHIR BackboneElement for Evidence.statistic.
type EvidenceStatistic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of content
	Description *string `json:"description,omitempty"`
	// Footnotes and/or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// Type of statistic, e.g., relative risk
	StatisticType *CodeableConcept `json:"statisticType,omitempty"`
	// Associated category for categorical variable
	Category *CodeableConcept `json:"category,omitempty"`
	// Statistic value
	Quantity *Quantity `json:"quantity,omitempty"`
	// The number of events associated with the statistic
	NumberOfEvents *uint `json:"numberOfEvents,omitempty"`
	// The number of participants affected
	NumberAffected *uint `json:"numberAffected,omitempty"`
	// Number of samples in the statistic
	SampleSize *EvidenceStatisticSampleSize `json:"sampleSize,omitempty"`
	// An attribute of the Statistic
	AttributeEstimate []EvidenceStatisticAttributeEstimate `json:"attributeEstimate,omitempty"`
	// An aspect of the statistical model
	ModelCharacteristic []EvidenceStatisticModelCharacteristic `json:"modelCharacteristic,omitempty"`
}

// EvidenceCertaintySubcomponent represents a FHIR BackboneElement for Evidence.certainty.subcomponent.
type EvidenceCertaintySubcomponent struct {
}

// EvidenceCertainty represents a FHIR BackboneElement for Evidence.certainty.
type EvidenceCertainty struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Textual description of certainty
	Description *string `json:"description,omitempty"`
	// Footnotes and/or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// Aspect of certainty being rated
	Type *CodeableConcept `json:"type,omitempty"`
	// Assessment or judgement of the aspect
	Rating *CodeableConcept `json:"rating,omitempty"`
	// Individual or group who did the rating
	Rater *string `json:"rater,omitempty"`
	// A domain or subdomain of certainty
	Subcomponent []EvidenceCertaintySubcomponent `json:"subcomponent,omitempty"`
}

// Evidence represents a FHIR Evidence.
type Evidence struct {
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
	// Canonical identifier for this evidence, represented as a globally unique URI
	URL *string `json:"url,omitempty"`
	// Additional identifier for the summary
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of this summary
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this summary (machine friendly)
	Name *string `json:"name,omitempty"`
	// Name for this summary (human friendly)
	Title *string `json:"title,omitempty"`
	// Citation for this evidence
	CiteAs *any `json:"citeAs,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// When the summary was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the summary was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Who authored the content
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the content
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the content
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the content
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Why this Evidence is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// Link or citation to artifact associated with the summary
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Description of the particular summary
	Description *string `json:"description,omitempty"`
	// Declarative description of the Evidence
	Assertion *string `json:"assertion,omitempty"`
	// Footnotes and/or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// Evidence variable such as population, exposure, or outcome
	VariableDefinition []EvidenceVariableDefinition `json:"variableDefinition,omitempty"`
	// The method to combine studies
	SynthesisType *CodeableConcept `json:"synthesisType,omitempty"`
	// The design of the study that produced this evidence
	StudyDesign []CodeableConcept `json:"studyDesign,omitempty"`
	// Values and parameters for a single statistic
	Statistic []EvidenceStatistic `json:"statistic,omitempty"`
	// Certainty or quality of the evidence
	Certainty []EvidenceCertainty `json:"certainty,omitempty"`
}
