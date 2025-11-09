package resources

// Distance represents a FHIR Distance.
type Distance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Numerical value (with implicit precision)
	Value *float64 `json:"value,omitempty"`
	// < | <= | >= | > - how to understand the value
	Comparator *string `json:"comparator,omitempty"`
	// Unit representation
	Unit *string `json:"unit,omitempty"`
	// System that defines coded unit form
	System *string `json:"system,omitempty"`
	// Coded form of the unit
	Code *string `json:"code,omitempty"`
}
