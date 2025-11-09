package resources

// ResourceTypeBodyStructure is the FHIR resource type name for BodyStructure.
const ResourceTypeBodyStructure = "BodyStructure"

// BodyStructureIncludedStructureBodyLandmarkOrientationDistanceFromLandmark represents a FHIR BackboneElement for BodyStructure.includedStructure.bodyLandmarkOrientation.distanceFromLandmark.
type BodyStructureIncludedStructureBodyLandmarkOrientationDistanceFromLandmark struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Measurement device
	Device []CodeableReference `json:"device,omitempty"`
	// Measured distance from body landmark
	Value []Quantity `json:"value,omitempty"`
}

// BodyStructureIncludedStructureBodyLandmarkOrientation represents a FHIR BackboneElement for BodyStructure.includedStructure.bodyLandmarkOrientation.
type BodyStructureIncludedStructureBodyLandmarkOrientation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Body ]andmark description
	LandmarkDescription []CodeableConcept `json:"landmarkDescription,omitempty"`
	// Clockface orientation
	ClockFacePosition []CodeableConcept `json:"clockFacePosition,omitempty"`
	// Landmark relative location
	DistanceFromLandmark []BodyStructureIncludedStructureBodyLandmarkOrientationDistanceFromLandmark `json:"distanceFromLandmark,omitempty"`
	// Relative landmark surface orientation
	SurfaceOrientation []CodeableConcept `json:"surfaceOrientation,omitempty"`
}

// BodyStructureIncludedStructure represents a FHIR BackboneElement for BodyStructure.includedStructure.
type BodyStructureIncludedStructure struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Code that represents the included structure
	Structure CodeableConcept `json:"structure"`
	// Code that represents the included structure laterality
	Laterality *CodeableConcept `json:"laterality,omitempty"`
	// Landmark relative location
	BodyLandmarkOrientation []BodyStructureIncludedStructureBodyLandmarkOrientation `json:"bodyLandmarkOrientation,omitempty"`
	// Cartesian reference for structure
	SpatialReference []Reference `json:"spatialReference,omitempty"`
	// Code that represents the included structure qualifier
	Qualifier []CodeableConcept `json:"qualifier,omitempty"`
}

// BodyStructureExcludedStructure represents a FHIR BackboneElement for BodyStructure.excludedStructure.
type BodyStructureExcludedStructure struct {
}

// BodyStructure represents a FHIR BodyStructure.
type BodyStructure struct {
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
	// Bodystructure identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Whether this record is in active use
	Active *bool `json:"active,omitempty"`
	// Kind of Structure
	Morphology *CodeableConcept `json:"morphology,omitempty"`
	// Included anatomic location(s)
	IncludedStructure []BodyStructureIncludedStructure `json:"includedStructure,omitempty"`
	// Excluded anatomic locations(s)
	ExcludedStructure []BodyStructureExcludedStructure `json:"excludedStructure,omitempty"`
	// Text description
	Description *string `json:"description,omitempty"`
	// Attached images
	Image []Attachment `json:"image,omitempty"`
	// Who this is about
	Patient Reference `json:"patient"`
}
