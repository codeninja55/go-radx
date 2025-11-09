package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeBundle is the FHIR resource type name for Bundle.
const ResourceTypeBundle = "Bundle"

// BundleLink represents a FHIR BackboneElement for Bundle.link.
type BundleLink struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// See http://www.iana.org/assignments/link-relations/link-relations.xhtml#link-relations-1
	Relation string `json:"relation"`
	// Reference details for the link
	URL string `json:"url"`
}

// BundleEntryLink represents a FHIR BackboneElement for Bundle.entry.link.
type BundleEntryLink struct {
}

// BundleEntrySearch represents a FHIR BackboneElement for Bundle.entry.search.
type BundleEntrySearch struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// match | include | outcome - why this is in the result set
	Mode *string `json:"mode,omitempty"`
	// Search ranking (between 0 and 1)
	Score *float64 `json:"score,omitempty"`
}

// BundleEntryRequest represents a FHIR BackboneElement for Bundle.entry.request.
type BundleEntryRequest struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// GET | HEAD | POST | PUT | DELETE | PATCH
	Method string `json:"method"`
	// URL for HTTP equivalent of this entry
	URL string `json:"url"`
	// For managing cache currency
	IfNoneMatch *string `json:"ifNoneMatch,omitempty"`
	// For managing cache currency
	IfModifiedSince *primitives.Instant `json:"ifModifiedSince,omitempty"`
	// For managing update contention
	IfMatch *string `json:"ifMatch,omitempty"`
	// For conditional creates
	IfNoneExist *string `json:"ifNoneExist,omitempty"`
}

// BundleEntryResponse represents a FHIR BackboneElement for Bundle.entry.response.
type BundleEntryResponse struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Status response code (text optional)
	Status string `json:"status"`
	// The location (if the operation returns a location)
	Location *string `json:"location,omitempty"`
	// The Etag for the resource (if relevant)
	Etag *string `json:"etag,omitempty"`
	// Server's date time modified
	LastModified *primitives.Instant `json:"lastModified,omitempty"`
	// OperationOutcome with hints and warnings (for batch/transaction)
	Outcome *any `json:"outcome,omitempty"`
}

// BundleEntry represents a FHIR BackboneElement for Bundle.entry.
type BundleEntry struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Links related to this entry
	Link []BundleEntryLink `json:"link,omitempty"`
	// URI for resource (Absolute URL server address or URI for UUID/OID)
	FullUrl *string `json:"fullUrl,omitempty"`
	// A resource in the bundle
	Resource *any `json:"resource,omitempty"`
	// Search related information
	Search *BundleEntrySearch `json:"search,omitempty"`
	// Additional execution information (transaction/batch/history)
	Request *BundleEntryRequest `json:"request,omitempty"`
	// Results of execution (transaction/batch/history)
	Response *BundleEntryResponse `json:"response,omitempty"`
}

// Bundle represents a FHIR Bundle.
type Bundle struct {
	// Logical id of this artifact
	ID *string `json:"id,omitempty"`
	// Metadata about the resource
	Meta *Meta `json:"meta,omitempty"`
	// A set of rules under which this content was created
	ImplicitRules *string `json:"implicitRules,omitempty"`
	// Language of the resource content
	Language *string `json:"language,omitempty"`
	// Persistent identifier for the bundle
	Identifier *Identifier `json:"identifier,omitempty"`
	// document | message | transaction | transaction-response | batch | batch-response | history | searchset | collection
	Type string `json:"type"`
	// When the bundle was assembled
	Timestamp *primitives.Instant `json:"timestamp,omitempty"`
	// If search, the total number of matches
	Total *uint `json:"total,omitempty"`
	// Links related to this Bundle
	Link []BundleLink `json:"link,omitempty"`
	// Entry in the bundle - will have a resource or information
	Entry []BundleEntry `json:"entry,omitempty"`
	// Digital Signature
	Signature *Signature `json:"signature,omitempty"`
}
