package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// Signature represents a FHIR Signature.
type Signature struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Indication of the reason the entity signed the object(s)
	Type []Coding `json:"type,omitempty"`
	// When the signature was created
	When primitives.Instant `json:"when"`
	// Who signed
	Who Reference `json:"who"`
	// The party represented
	OnBehalfOf *Reference `json:"onBehalfOf,omitempty"`
	// The technical format of the signed resources
	TargetFormat *string `json:"targetFormat,omitempty"`
	// The technical format of the signature
	SigFormat *string `json:"sigFormat,omitempty"`
	// The actual signature content (XML DigSig. JWS, picture, etc.)
	Data *string `json:"data,omitempty"`
}
