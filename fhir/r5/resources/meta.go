package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// Meta represents a FHIR Meta.
type Meta struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Version specific identifier
	VersionId *string `json:"versionId,omitempty"`
	// When the resource version last changed
	LastUpdated *primitives.Instant `json:"lastUpdated,omitempty"`
	// Identifies where the resource comes from
	Source *string `json:"source,omitempty"`
	// Profiles this resource claims to conform to
	Profile []string `json:"profile,omitempty"`
	// Security Labels applied to this resource
	Security []Coding `json:"security,omitempty"`
	// Tags applied to this resource
	Tag []Coding `json:"tag,omitempty"`
}
