package resources

// DataRequirementCodeFilter represents a FHIR BackboneElement for DataRequirement.codeFilter.
type DataRequirementCodeFilter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// A code-valued attribute to filter on
	Path *string `json:"path,omitempty"`
	// A coded (token) parameter to search on
	SearchParam *string `json:"searchParam,omitempty"`
	// ValueSet for the filter
	ValueSet *string `json:"valueSet,omitempty"`
	// What code is expected
	Code []Coding `json:"code,omitempty"`
}

// DataRequirementDateFilter represents a FHIR BackboneElement for DataRequirement.dateFilter.
type DataRequirementDateFilter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// A date-valued attribute to filter on
	Path *string `json:"path,omitempty"`
	// A date valued parameter to search on
	SearchParam *string `json:"searchParam,omitempty"`
	// The value of the filter, as a Period, DateTime, or Duration value
	Value *any `json:"value,omitempty"`
}

// DataRequirementValueFilter represents a FHIR BackboneElement for DataRequirement.valueFilter.
type DataRequirementValueFilter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// An attribute to filter on
	Path *string `json:"path,omitempty"`
	// A parameter to search on
	SearchParam *string `json:"searchParam,omitempty"`
	// eq | gt | lt | ge | le | sa | eb
	Comparator *string `json:"comparator,omitempty"`
	// The value of the filter, as a Period, DateTime, or Duration value
	Value *any `json:"value,omitempty"`
}

// DataRequirementSort represents a FHIR BackboneElement for DataRequirement.sort.
type DataRequirementSort struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// The name of the attribute to perform the sort
	Path string `json:"path"`
	// ascending | descending
	Direction string `json:"direction"`
}

// DataRequirement represents a FHIR DataRequirement.
type DataRequirement struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// The type of the required data
	Type string `json:"type"`
	// The profile of the required data
	Profile []string `json:"profile,omitempty"`
	// E.g. Patient, Practitioner, RelatedPerson, Organization, Location, Device
	Subject *any `json:"subject,omitempty"`
	// Indicates specific structure elements that are referenced by the knowledge module
	MustSupport []string `json:"mustSupport,omitempty"`
	// What codes are expected
	CodeFilter []DataRequirementCodeFilter `json:"codeFilter,omitempty"`
	// What dates/date ranges are expected
	DateFilter []DataRequirementDateFilter `json:"dateFilter,omitempty"`
	// What values are expected
	ValueFilter []DataRequirementValueFilter `json:"valueFilter,omitempty"`
	// Number of results
	Limit *int `json:"limit,omitempty"`
	// Order of the results
	Sort []DataRequirementSort `json:"sort,omitempty"`
}
