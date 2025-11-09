package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// Annotation represents a FHIR Annotation.
type Annotation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Individual responsible for the annotation
	Author *any `json:"author,omitempty"`
	// When the annotation was made
	Time *primitives.DateTime `json:"time,omitempty"`
	// The annotation  - text content (as markdown)
	Text string `json:"text"`
}
