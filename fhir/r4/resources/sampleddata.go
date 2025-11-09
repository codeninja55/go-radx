package resources

// SampledData represents a FHIR SampledData.
type SampledData struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Zero value and units
	Origin Quantity `json:"origin"`
	// Number of milliseconds between samples
	Period float64 `json:"period"`
	// Multiply data by this before adding to origin
	Factor *float64 `json:"factor,omitempty"`
	// Lower limit of detection
	LowerLimit *float64 `json:"lowerLimit,omitempty"`
	// Upper limit of detection
	UpperLimit *float64 `json:"upperLimit,omitempty"`
	// Number of sample points at each time point
	Dimensions int `json:"dimensions"`
	// Decimal values with spaces, or "E" | "U" | "L"
	Data *string `json:"data,omitempty"`
}
