package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeDocumentManifest is the FHIR resource type name for DocumentManifest.
const ResourceTypeDocumentManifest = "DocumentManifest"

// DocumentManifestRelated represents a FHIR BackboneElement for DocumentManifest.related.
type DocumentManifestRelated struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifiers of things that are related
	Identifier *Identifier `json:"identifier,omitempty"`
	// Related Resource
	Ref *Reference `json:"ref,omitempty"`
}

// DocumentManifest represents a FHIR DocumentManifest.
type DocumentManifest struct {
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
	// Unique Identifier for the set of documents
	MasterIdentifier *Identifier `json:"masterIdentifier,omitempty"`
	// Other identifiers for the manifest
	Identifier []Identifier `json:"identifier,omitempty"`
	// current | superseded | entered-in-error
	Status string `json:"status"`
	// Kind of document set
	Type *CodeableConcept `json:"type,omitempty"`
	// The subject of the set of documents
	Subject *Reference `json:"subject,omitempty"`
	// When this document manifest created
	Created *primitives.DateTime `json:"created,omitempty"`
	// Who and/or what authored the DocumentManifest
	Author []Reference `json:"author,omitempty"`
	// Intended to get notified about this set of documents
	Recipient []Reference `json:"recipient,omitempty"`
	// The source system/application/software
	Source *string `json:"source,omitempty"`
	// Human-readable description (title)
	Description *string `json:"description,omitempty"`
	// Items in manifest
	Content []Reference `json:"content,omitempty"`
	// Related things
	Related []DocumentManifestRelated `json:"related,omitempty"`
}
