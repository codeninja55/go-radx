package resources

// SampledData represents a FHIR SampledData.
type SampledData struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Zero value and units
	Origin Quantity `json:"origin"`
	// Number of intervalUnits between samples
	Interval *float64 `json:"interval,omitempty"`
	// The measurement unit of the interval between samples
	IntervalUnit string `json:"intervalUnit"`
	// Multiply data by this before adding to origin
	Factor *float64 `json:"factor,omitempty"`
	// Lower limit of detection
	LowerLimit *float64 `json:"lowerLimit,omitempty"`
	// Upper limit of detection
	UpperLimit *float64 `json:"upperLimit,omitempty"`
	// Number of sample points at each time point
	Dimensions int `json:"dimensions"`
	// Defines the codes used in the data
	CodeMap *string `json:"codeMap,omitempty"`
	// Offsets, typically in time, at which data values were taken
	Offsets *string `json:"offsets,omitempty"`
	// Decimal values with spaces, or "E" | "U" | "L", or another code
	Data *string `json:"data,omitempty"`
}
