package resources

// ResourceTypeObservationDefinition is the FHIR resource type name for ObservationDefinition.
const ResourceTypeObservationDefinition = "ObservationDefinition"

// ObservationDefinitionQuantitativeDetails represents a FHIR BackboneElement for ObservationDefinition.quantitativeDetails.
type ObservationDefinitionQuantitativeDetails struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Customary unit for quantitative results
	CustomaryUnit *CodeableConcept `json:"customaryUnit,omitempty"`
	// SI unit for quantitative results
	Unit *CodeableConcept `json:"unit,omitempty"`
	// SI to Customary unit conversion factor
	ConversionFactor *float64 `json:"conversionFactor,omitempty"`
	// Decimal precision of observation quantitative results
	DecimalPrecision *int `json:"decimalPrecision,omitempty"`
}

// ObservationDefinitionQualifiedInterval represents a FHIR BackboneElement for ObservationDefinition.qualifiedInterval.
type ObservationDefinitionQualifiedInterval struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// reference | critical | absolute
	Category *string `json:"category,omitempty"`
	// The interval itself, for continuous or ordinal observations
	Range *Range `json:"range,omitempty"`
	// Range context qualifier
	Context *CodeableConcept `json:"context,omitempty"`
	// Targetted population of the range
	AppliesTo []CodeableConcept `json:"appliesTo,omitempty"`
	// male | female | other | unknown
	Gender *string `json:"gender,omitempty"`
	// Applicable age range, if relevant
	Age *Range `json:"age,omitempty"`
	// Applicable gestational age range, if relevant
	GestationalAge *Range `json:"gestationalAge,omitempty"`
	// Condition associated with the reference range
	Condition *string `json:"condition,omitempty"`
}

// ObservationDefinition represents a FHIR ObservationDefinition.
type ObservationDefinition struct {
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
	// Category of observation
	Category []CodeableConcept `json:"category,omitempty"`
	// Type of observation (code / type)
	Code CodeableConcept `json:"code"`
	// Business identifier for this ObservationDefinition instance
	Identifier []Identifier `json:"identifier,omitempty"`
	// Quantity | CodeableConcept | string | boolean | integer | Range | Ratio | SampledData | time | dateTime | Period
	PermittedDataType []string `json:"permittedDataType,omitempty"`
	// Multiple results allowed
	MultipleResultsAllowed *bool `json:"multipleResultsAllowed,omitempty"`
	// Method used to produce the observation
	Method *CodeableConcept `json:"method,omitempty"`
	// Preferred report name
	PreferredReportName *string `json:"preferredReportName,omitempty"`
	// Characteristics of quantitative results
	QuantitativeDetails *ObservationDefinitionQuantitativeDetails `json:"quantitativeDetails,omitempty"`
	// Qualified range for continuous and ordinal observation results
	QualifiedInterval []ObservationDefinitionQualifiedInterval `json:"qualifiedInterval,omitempty"`
	// Value set of valid coded values for the observations conforming to this ObservationDefinition
	ValidCodedValueSet *Reference `json:"validCodedValueSet,omitempty"`
	// Value set of normal coded values for the observations conforming to this ObservationDefinition
	NormalCodedValueSet *Reference `json:"normalCodedValueSet,omitempty"`
	// Value set of abnormal coded values for the observations conforming to this ObservationDefinition
	AbnormalCodedValueSet *Reference `json:"abnormalCodedValueSet,omitempty"`
	// Value set of critical coded values for the observations conforming to this ObservationDefinition
	CriticalCodedValueSet *Reference `json:"criticalCodedValueSet,omitempty"`
}
