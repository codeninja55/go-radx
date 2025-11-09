package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeVisionPrescription is the FHIR resource type name for VisionPrescription.
const ResourceTypeVisionPrescription = "VisionPrescription"

// VisionPrescriptionLensSpecificationPrism represents a FHIR BackboneElement for VisionPrescription.lensSpecification.prism.
type VisionPrescriptionLensSpecificationPrism struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Amount of adjustment
	Amount float64 `json:"amount"`
	// up | down | in | out
	Base string `json:"base"`
}

// VisionPrescriptionLensSpecification represents a FHIR BackboneElement for VisionPrescription.lensSpecification.
type VisionPrescriptionLensSpecification struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Product to be supplied
	Product CodeableConcept `json:"product"`
	// right | left
	Eye string `json:"eye"`
	// Power of the lens
	Sphere *float64 `json:"sphere,omitempty"`
	// Lens power for astigmatism
	Cylinder *float64 `json:"cylinder,omitempty"`
	// Lens meridian which contain no power for astigmatism
	Axis *int `json:"axis,omitempty"`
	// Eye alignment compensation
	Prism []VisionPrescriptionLensSpecificationPrism `json:"prism,omitempty"`
	// Added power for multifocal levels
	Add *float64 `json:"add,omitempty"`
	// Contact lens power
	Power *float64 `json:"power,omitempty"`
	// Contact lens back curvature
	BackCurve *float64 `json:"backCurve,omitempty"`
	// Contact lens diameter
	Diameter *float64 `json:"diameter,omitempty"`
	// Lens wear duration
	Duration *Quantity `json:"duration,omitempty"`
	// Color required
	Color *string `json:"color,omitempty"`
	// Brand required
	Brand *string `json:"brand,omitempty"`
	// Notes for coatings
	Note []Annotation `json:"note,omitempty"`
}

// VisionPrescription represents a FHIR VisionPrescription.
type VisionPrescription struct {
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
	// Business Identifier for vision prescription
	Identifier []Identifier `json:"identifier,omitempty"`
	// active | cancelled | draft | entered-in-error
	Status string `json:"status"`
	// Response creation date
	Created primitives.DateTime `json:"created"`
	// Who prescription is for
	Patient Reference `json:"patient"`
	// Created during encounter / admission / stay
	Encounter *Reference `json:"encounter,omitempty"`
	// When prescription was authorized
	DateWritten primitives.DateTime `json:"dateWritten"`
	// Who authorized the vision prescription
	Prescriber Reference `json:"prescriber"`
	// Vision lens authorization
	LensSpecification []VisionPrescriptionLensSpecification `json:"lensSpecification,omitempty"`
}
