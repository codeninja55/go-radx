package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// Period represents a FHIR Period.
type Period struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Starting time with inclusive boundary
	Start *primitives.DateTime `json:"start,omitempty"`
	// End time with inclusive boundary, if not ongoing
	End *primitives.DateTime `json:"end,omitempty"`
}
