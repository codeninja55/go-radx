package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMeasureReport is the FHIR resource type name for MeasureReport.
const ResourceTypeMeasureReport = "MeasureReport"

// MeasureReportGroupPopulation represents a FHIR BackboneElement for MeasureReport.group.population.
type MeasureReportGroupPopulation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// initial-population | numerator | numerator-exclusion | denominator | denominator-exclusion | denominator-exception | measure-population | measure-population-exclusion | measure-observation
	Code *CodeableConcept `json:"code,omitempty"`
	// Size of the population
	Count *int `json:"count,omitempty"`
	// For subject-list reports, the subject results in this population
	SubjectResults *Reference `json:"subjectResults,omitempty"`
}

// MeasureReportGroupStratifierStratumComponent represents a FHIR BackboneElement for MeasureReport.group.stratifier.stratum.component.
type MeasureReportGroupStratifierStratumComponent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What stratifier component of the group
	Code CodeableConcept `json:"code"`
	// The stratum component value, e.g. male
	Value CodeableConcept `json:"value"`
}

// MeasureReportGroupStratifierStratumPopulation represents a FHIR BackboneElement for MeasureReport.group.stratifier.stratum.population.
type MeasureReportGroupStratifierStratumPopulation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// initial-population | numerator | numerator-exclusion | denominator | denominator-exclusion | denominator-exception | measure-population | measure-population-exclusion | measure-observation
	Code *CodeableConcept `json:"code,omitempty"`
	// Size of the population
	Count *int `json:"count,omitempty"`
	// For subject-list reports, the subject results in this population
	SubjectResults *Reference `json:"subjectResults,omitempty"`
}

// MeasureReportGroupStratifierStratum represents a FHIR BackboneElement for MeasureReport.group.stratifier.stratum.
type MeasureReportGroupStratifierStratum struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The stratum value, e.g. male
	Value *CodeableConcept `json:"value,omitempty"`
	// Stratifier component values
	Component []MeasureReportGroupStratifierStratumComponent `json:"component,omitempty"`
	// Population results in this stratum
	Population []MeasureReportGroupStratifierStratumPopulation `json:"population,omitempty"`
	// What score this stratum achieved
	MeasureScore *Quantity `json:"measureScore,omitempty"`
}

// MeasureReportGroupStratifier represents a FHIR BackboneElement for MeasureReport.group.stratifier.
type MeasureReportGroupStratifier struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What stratifier of the group
	Code []CodeableConcept `json:"code,omitempty"`
	// Stratum results, one for each unique value, or set of values, in the stratifier, or stratifier components
	Stratum []MeasureReportGroupStratifierStratum `json:"stratum,omitempty"`
}

// MeasureReportGroup represents a FHIR BackboneElement for MeasureReport.group.
type MeasureReportGroup struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Meaning of the group
	Code *CodeableConcept `json:"code,omitempty"`
	// The populations in the group
	Population []MeasureReportGroupPopulation `json:"population,omitempty"`
	// What score this group achieved
	MeasureScore *Quantity `json:"measureScore,omitempty"`
	// Stratification results
	Stratifier []MeasureReportGroupStratifier `json:"stratifier,omitempty"`
}

// MeasureReport represents a FHIR MeasureReport.
type MeasureReport struct {
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
	// Additional identifier for the MeasureReport
	Identifier []Identifier `json:"identifier,omitempty"`
	// complete | pending | error
	Status string `json:"status"`
	// individual | subject-list | summary | data-collection
	Type string `json:"type"`
	// What measure was calculated
	Measure string `json:"measure"`
	// What individual(s) the report is for
	Subject *Reference `json:"subject,omitempty"`
	// When the report was generated
	Date *primitives.DateTime `json:"date,omitempty"`
	// Who is reporting the data
	Reporter *Reference `json:"reporter,omitempty"`
	// What period the report covers
	Period Period `json:"period"`
	// increase | decrease
	ImprovementNotation *CodeableConcept `json:"improvementNotation,omitempty"`
	// Measure results for each group
	Group []MeasureReportGroup `json:"group,omitempty"`
	// What data was used to calculate the measure score
	EvaluatedResource []Reference `json:"evaluatedResource,omitempty"`
}
