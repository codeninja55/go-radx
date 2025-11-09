package resources

// RelatedArtifact represents a FHIR RelatedArtifact.
type RelatedArtifact struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// documentation | justification | citation | predecessor | successor | derived-from | depends-on | composed-of
	Type string `json:"type"`
	// Short label
	Label *string `json:"label,omitempty"`
	// Brief description of the related artifact
	Display *string `json:"display,omitempty"`
	// Bibliographic citation for the artifact
	Citation *string `json:"citation,omitempty"`
	// Where the artifact can be accessed
	URL *string `json:"url,omitempty"`
	// What document is being referenced
	Document *Attachment `json:"document,omitempty"`
	// What resource is being referenced
	Resource *string `json:"resource,omitempty"`
}
