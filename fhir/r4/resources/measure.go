package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMeasure is the FHIR resource type name for Measure.
const ResourceTypeMeasure = "Measure"

// MeasureGroupPopulation represents a FHIR BackboneElement for Measure.group.population.
type MeasureGroupPopulation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// initial-population | numerator | numerator-exclusion | denominator | denominator-exclusion | denominator-exception | measure-population | measure-population-exclusion | measure-observation
	Code *CodeableConcept `json:"code,omitempty"`
	// The human readable description of this population criteria
	Description *string `json:"description,omitempty"`
	// The criteria that defines this population
	Criteria Expression `json:"criteria"`
}

// MeasureGroupStratifierComponent represents a FHIR BackboneElement for Measure.group.stratifier.component.
type MeasureGroupStratifierComponent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Meaning of the stratifier component
	Code *CodeableConcept `json:"code,omitempty"`
	// The human readable description of this stratifier component
	Description *string `json:"description,omitempty"`
	// Component of how the measure should be stratified
	Criteria Expression `json:"criteria"`
}

// MeasureGroupStratifier represents a FHIR BackboneElement for Measure.group.stratifier.
type MeasureGroupStratifier struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Meaning of the stratifier
	Code *CodeableConcept `json:"code,omitempty"`
	// The human readable description of this stratifier
	Description *string `json:"description,omitempty"`
	// How the measure should be stratified
	Criteria *Expression `json:"criteria,omitempty"`
	// Stratifier criteria component for the measure
	Component []MeasureGroupStratifierComponent `json:"component,omitempty"`
}

// MeasureGroup represents a FHIR BackboneElement for Measure.group.
type MeasureGroup struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Meaning of the group
	Code *CodeableConcept `json:"code,omitempty"`
	// Summary description
	Description *string `json:"description,omitempty"`
	// Population criteria
	Population []MeasureGroupPopulation `json:"population,omitempty"`
	// Stratifier criteria for the measure
	Stratifier []MeasureGroupStratifier `json:"stratifier,omitempty"`
}

// MeasureSupplementalData represents a FHIR BackboneElement for Measure.supplementalData.
type MeasureSupplementalData struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Meaning of the supplemental data
	Code *CodeableConcept `json:"code,omitempty"`
	// supplemental-data | risk-adjustment-factor
	Usage []CodeableConcept `json:"usage,omitempty"`
	// The human readable description of this supplemental data
	Description *string `json:"description,omitempty"`
	// Expression describing additional data to be reported
	Criteria Expression `json:"criteria"`
}

// Measure represents a FHIR Measure.
type Measure struct {
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
	// Canonical identifier for this measure, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the measure
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the measure
	Version *string `json:"version,omitempty"`
	// Name for this measure (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this measure (human friendly)
	Title *string `json:"title,omitempty"`
	// Subordinate title of the measure
	Subtitle *string `json:"subtitle,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// E.g. Patient, Practitioner, RelatedPerson, Organization, Location, Device
	Subject *any `json:"subject,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the measure
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for measure (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this measure is defined
	Purpose *string `json:"purpose,omitempty"`
	// Describes the clinical usage of the measure
	Usage *string `json:"usage,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the measure was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the measure was last reviewed
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the measure is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// The category of the measure, such as Education, Treatment, Assessment, etc.
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
	// Logic used by the measure
	Library []string `json:"library,omitempty"`
	// Disclaimer for use of the measure or its referenced content
	Disclaimer *string `json:"disclaimer,omitempty"`
	// proportion | ratio | continuous-variable | cohort
	Scoring *CodeableConcept `json:"scoring,omitempty"`
	// opportunity | all-or-nothing | linear | weighted
	CompositeScoring *CodeableConcept `json:"compositeScoring,omitempty"`
	// process | outcome | structure | patient-reported-outcome | composite
	Type []CodeableConcept `json:"type,omitempty"`
	// How risk adjustment is applied for this measure
	RiskAdjustment *string `json:"riskAdjustment,omitempty"`
	// How is rate aggregation performed for this measure
	RateAggregation *string `json:"rateAggregation,omitempty"`
	// Detailed description of why the measure exists
	Rationale *string `json:"rationale,omitempty"`
	// Summary of clinical guidelines
	ClinicalRecommendationStatement *string `json:"clinicalRecommendationStatement,omitempty"`
	// increase | decrease
	ImprovementNotation *CodeableConcept `json:"improvementNotation,omitempty"`
	// Defined terms used in the measure documentation
	Definition []string `json:"definition,omitempty"`
	// Additional guidance for implementers
	Guidance *string `json:"guidance,omitempty"`
	// Population criteria group
	Group []MeasureGroup `json:"group,omitempty"`
	// What other data should be reported with the measure
	SupplementalData []MeasureSupplementalData `json:"supplementalData,omitempty"`
}
