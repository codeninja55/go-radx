package r5

// Reference is a FHIR reference datatype: a pointer to another resource by
// relative/absolute URL, by logical Identifier, or both, with an optional
// human-readable Display and a Type hint naming the referenced resource. A DICOM
// UID is never carried here as a Reference.reference URL; it becomes an
// Identifier (the glossary rule), which may sit in this datatype's Identifier
// field as a logical reference.
type Reference struct {
	Reference  *string     `json:"reference,omitempty"` // e.g. "Patient/pat-1" or "#contained-id"
	Type       *string     `json:"type,omitempty"`      // referenced resourceType, e.g. "Patient"
	Identifier *Identifier `json:"identifier,omitempty"`
	Display    *string     `json:"display,omitempty"`
}

// Identifier is a FHIR identifier datatype: a value qualified by the system that
// issued it. DICOM UIDs map to an Identifier with system "urn:dicom:uid" (the
// glossary rule), never to a Reference.
type Identifier struct {
	Use      *string          `json:"use,omitempty"`
	Type     *CodeableConcept `json:"type,omitempty"`
	System   *string          `json:"system,omitempty"`
	Value    *string          `json:"value,omitempty"`
	Assigner *Reference       `json:"assigner,omitempty"`
}

// Coding is a single coded value drawn from a code system: a System URI, a Code
// within it, and an optional human-readable Display.
type Coding struct {
	System  *string `json:"system,omitempty"`
	Version *string `json:"version,omitempty"`
	Code    *string `json:"code,omitempty"`
	Display *string `json:"display,omitempty"`
}

// CodeableConcept is a concept that may be coded by one or more Codings drawn
// from different systems, with an optional free-text Text rendering.
type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   *string  `json:"text,omitempty"`
}

// CodeableReference is the R5 datatype that carries either a coded Concept, a
// Reference to a resource, or both. R5 introduced it for elements that may be
// expressed by a code or by a pointer to a resource (for example
// ServiceRequest.code and ImagingStudy.series.bodySite), where R4 used a plain
// CodeableConcept or Coding.
type CodeableReference struct {
	Concept   *CodeableConcept `json:"concept,omitempty"`
	Reference *Reference       `json:"reference,omitempty"`
}
