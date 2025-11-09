package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedia is the FHIR resource type name for Media.
const ResourceTypeMedia = "Media"

// Media represents a FHIR Media.
type Media struct {
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
	// Identifier(s) for the image
	Identifier []Identifier `json:"identifier,omitempty"`
	// Procedure that caused this media to be created
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// preparation | in-progress | not-done | on-hold | stopped | completed | entered-in-error | unknown
	Status string `json:"status"`
	// Classification of media as image, video, or audio
	Type *CodeableConcept `json:"type,omitempty"`
	// The type of acquisition equipment/process
	Modality *CodeableConcept `json:"modality,omitempty"`
	// Imaging view, e.g. Lateral or Antero-posterior
	View *CodeableConcept `json:"view,omitempty"`
	// Who/What this Media is a record of
	Subject *Reference `json:"subject,omitempty"`
	// Encounter associated with media
	Encounter *Reference `json:"encounter,omitempty"`
	// When Media was collected
	Created *any `json:"created,omitempty"`
	// Date/Time this version was made available
	Issued *primitives.Instant `json:"issued,omitempty"`
	// The person who generated the image
	Operator *Reference `json:"operator,omitempty"`
	// Why was event performed?
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Observed body part
	BodySite *CodeableConcept `json:"bodySite,omitempty"`
	// Name of the device/manufacturer
	DeviceName *string `json:"deviceName,omitempty"`
	// Observing Device
	Device *Reference `json:"device,omitempty"`
	// Height of the image in pixels (photo/video)
	Height *int `json:"height,omitempty"`
	// Width of the image in pixels (photo/video)
	Width *int `json:"width,omitempty"`
	// Number of frames if > 1 (photo)
	Frames *int `json:"frames,omitempty"`
	// Length in seconds (audio / video)
	Duration *float64 `json:"duration,omitempty"`
	// Actual Media - reference or data
	Content Attachment `json:"content"`
	// Comments made about the media
	Note []Annotation `json:"note,omitempty"`
}
