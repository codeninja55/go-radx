package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeImagingSelection is the FHIR resource type name for ImagingSelection.
const ResourceTypeImagingSelection = "ImagingSelection"

// ImagingSelectionPerformer represents a FHIR BackboneElement for ImagingSelection.performer.
type ImagingSelectionPerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of performer
	Function *CodeableConcept `json:"function,omitempty"`
	// Author (human or machine)
	Actor *Reference `json:"actor,omitempty"`
}

// ImagingSelectionInstanceImageRegion2D represents a FHIR BackboneElement for ImagingSelection.instance.imageRegion2D.
type ImagingSelectionInstanceImageRegion2D struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// point | polyline | interpolated | circle | ellipse
	RegionType string `json:"regionType"`
	// Specifies the coordinates that define the image region
	Coordinate []float64 `json:"coordinate,omitempty"`
}

// ImagingSelectionInstanceImageRegion3D represents a FHIR BackboneElement for ImagingSelection.instance.imageRegion3D.
type ImagingSelectionInstanceImageRegion3D struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// point | multipoint | polyline | polygon | ellipse | ellipsoid
	RegionType string `json:"regionType"`
	// Specifies the coordinates that define the image region
	Coordinate []float64 `json:"coordinate,omitempty"`
}

// ImagingSelectionInstance represents a FHIR BackboneElement for ImagingSelection.instance.
type ImagingSelectionInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// DICOM SOP Instance UID
	Uid string `json:"uid"`
	// DICOM Instance Number
	Number *uint `json:"number,omitempty"`
	// DICOM SOP Class UID
	SopClass *Coding `json:"sopClass,omitempty"`
	// The selected subset of the SOP Instance
	Subset []string `json:"subset,omitempty"`
	// A specific 2D region in a DICOM image / frame
	ImageRegion2D []ImagingSelectionInstanceImageRegion2D `json:"imageRegion2D,omitempty"`
	// A specific 3D region in a DICOM frame of reference
	ImageRegion3D []ImagingSelectionInstanceImageRegion3D `json:"imageRegion3D,omitempty"`
}

// ImagingSelection represents a FHIR ImagingSelection.
type ImagingSelection struct {
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
	// Business Identifier for Imaging Selection
	Identifier []Identifier `json:"identifier,omitempty"`
	// available | entered-in-error | unknown
	Status string `json:"status"`
	// Subject of the selected instances
	Subject *Reference `json:"subject,omitempty"`
	// Date / Time when this imaging selection was created
	Issued *primitives.Instant `json:"issued,omitempty"`
	// Selector of the instances (human or machine)
	Performer []ImagingSelectionPerformer `json:"performer,omitempty"`
	// Associated request
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Classifies the imaging selection
	Category []CodeableConcept `json:"category,omitempty"`
	// Imaging Selection purpose text or code
	Code CodeableConcept `json:"code"`
	// DICOM Study Instance UID
	StudyUid *string `json:"studyUid,omitempty"`
	// The imaging study from which the imaging selection is derived
	DerivedFrom []Reference `json:"derivedFrom,omitempty"`
	// The network service providing retrieval for the images referenced in the imaging selection
	Endpoint []Reference `json:"endpoint,omitempty"`
	// DICOM Series Instance UID
	SeriesUid *string `json:"seriesUid,omitempty"`
	// DICOM Series Number
	SeriesNumber *uint `json:"seriesNumber,omitempty"`
	// The Frame of Reference UID for the selected images
	FrameOfReferenceUid *string `json:"frameOfReferenceUid,omitempty"`
	// Body part examined
	BodySite *CodeableReference `json:"bodySite,omitempty"`
	// Related resource that is the focus for the imaging selection
	Focus []Reference `json:"focus,omitempty"`
	// The selected instances
	Instance []ImagingSelectionInstance `json:"instance,omitempty"`
}
