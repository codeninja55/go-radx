package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// RelatedArtifact represents a FHIR RelatedArtifact.
type RelatedArtifact struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// documentation | justification | citation | predecessor | successor | derived-from | depends-on | composed-of | part-of | amends | amended-with | appends | appended-with | cites | cited-by | comments-on | comment-in | contains | contained-in | corrects | correction-in | replaces | replaced-with | retracts | retracted-by | signs | similar-to | supports | supported-with | transforms | transformed-into | transformed-with | documents | specification-of | created-with | cite-as
	Type string `json:"type"`
	// Additional classifiers
	Classifier []CodeableConcept `json:"classifier,omitempty"`
	// Short label
	Label *string `json:"label,omitempty"`
	// Brief description of the related artifact
	Display *string `json:"display,omitempty"`
	// Bibliographic citation for the artifact
	Citation *string `json:"citation,omitempty"`
	// What document is being referenced
	Document *Attachment `json:"document,omitempty"`
	// What artifact is being referenced
	Resource *string `json:"resource,omitempty"`
	// What artifact, if not a conformance resource
	ResourceReference *Reference `json:"resourceReference,omitempty"`
	// draft | active | retired | unknown
	PublicationStatus *string `json:"publicationStatus,omitempty"`
	// Date of publication of the artifact being referred to
	PublicationDate *primitives.Date `json:"publicationDate,omitempty"`
}
