package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeImmunizationRecommendation is the FHIR resource type name for ImmunizationRecommendation.
const ResourceTypeImmunizationRecommendation = "ImmunizationRecommendation"

// ImmunizationRecommendationRecommendationDateCriterion represents a FHIR BackboneElement for ImmunizationRecommendation.recommendation.dateCriterion.
type ImmunizationRecommendationRecommendationDateCriterion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of date
	Code CodeableConcept `json:"code"`
	// Recommended date
	Value primitives.DateTime `json:"value"`
}

// ImmunizationRecommendationRecommendation represents a FHIR BackboneElement for ImmunizationRecommendation.recommendation.
type ImmunizationRecommendationRecommendation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Vaccine  or vaccine group recommendation applies to
	VaccineCode []CodeableConcept `json:"vaccineCode,omitempty"`
	// Disease to be immunized against
	TargetDisease *CodeableConcept `json:"targetDisease,omitempty"`
	// Vaccine which is contraindicated to fulfill the recommendation
	ContraindicatedVaccineCode []CodeableConcept `json:"contraindicatedVaccineCode,omitempty"`
	// Vaccine recommendation status
	ForecastStatus CodeableConcept `json:"forecastStatus"`
	// Vaccine administration status reason
	ForecastReason []CodeableConcept `json:"forecastReason,omitempty"`
	// Dates governing proposed immunization
	DateCriterion []ImmunizationRecommendationRecommendationDateCriterion `json:"dateCriterion,omitempty"`
	// Protocol details
	Description *string `json:"description,omitempty"`
	// Name of vaccination series
	Series *string `json:"series,omitempty"`
	// Recommended dose number within series
	DoseNumber *any `json:"doseNumber,omitempty"`
	// Recommended number of doses for immunity
	SeriesDoses *any `json:"seriesDoses,omitempty"`
	// Past immunizations supporting recommendation
	SupportingImmunization []Reference `json:"supportingImmunization,omitempty"`
	// Patient observations supporting recommendation
	SupportingPatientInformation []Reference `json:"supportingPatientInformation,omitempty"`
}

// ImmunizationRecommendation represents a FHIR ImmunizationRecommendation.
type ImmunizationRecommendation struct {
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
	// Who this profile is for
	Patient Reference `json:"patient"`
	// Date recommendation(s) created
	Date primitives.DateTime `json:"date"`
	// Who is responsible for protocol
	Authority *Reference `json:"authority,omitempty"`
	// Vaccine administration recommendations
	Recommendation []ImmunizationRecommendationRecommendation `json:"recommendation,omitempty"`
}
