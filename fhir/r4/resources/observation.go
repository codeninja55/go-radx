package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeObservation is the FHIR resource type name for Observation.
const ResourceTypeObservation = "Observation"

// ObservationReferenceRange represents a FHIR BackboneElement for Observation.referenceRange.
type ObservationReferenceRange struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Low Range, if relevant
	Low *Quantity `json:"low,omitempty"`
	// High Range, if relevant
	High *Quantity `json:"high,omitempty"`
	// Reference range qualifier
	Type *CodeableConcept `json:"type,omitempty"`
	// Reference range population
	AppliesTo []CodeableConcept `json:"appliesTo,omitempty"`
	// Applicable age range, if relevant
	Age *Range `json:"age,omitempty"`
	// Text based reference range in an observation
	Text *string `json:"text,omitempty"`
}

// ObservationComponentReferenceRange represents a FHIR BackboneElement for Observation.component.referenceRange.
type ObservationComponentReferenceRange struct {
}

// ObservationComponent represents a FHIR BackboneElement for Observation.component.
type ObservationComponent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of component observation (code / type)
	Code CodeableConcept `json:"code"`
	// Actual component result
	Value *any `json:"value,omitempty"`
	// Why the component result is missing
	DataAbsentReason *CodeableConcept `json:"dataAbsentReason,omitempty"`
	// High, low, normal, etc.
	Interpretation []CodeableConcept `json:"interpretation,omitempty"`
	// Provides guide for interpretation of component result
	ReferenceRange []ObservationComponentReferenceRange `json:"referenceRange,omitempty"`
}

// Observation represents a FHIR Observation.
type Observation struct {
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
	// Business Identifier for observation
	Identifier []Identifier `json:"identifier,omitempty"`
	// Fulfills plan, proposal or order
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// registered | preliminary | final | amended +
	Status string `json:"status"`
	// Classification of  type of observation
	Category []CodeableConcept `json:"category,omitempty"`
	// Type of observation (code / type)
	Code CodeableConcept `json:"code"`
	// Who and/or what the observation is about
	Subject *Reference `json:"subject,omitempty"`
	// What the observation is about, when it is not about the subject of record
	Focus []Reference `json:"focus,omitempty"`
	// Healthcare event during which this observation is made
	Encounter *Reference `json:"encounter,omitempty"`
	// Clinically relevant time/time-period for observation
	Effective *any `json:"effective,omitempty"`
	// Date/Time this version was made available
	Issued *primitives.Instant `json:"issued,omitempty"`
	// Who is responsible for the observation
	Performer []Reference `json:"performer,omitempty"`
	// Actual result
	Value *any `json:"value,omitempty"`
	// Why the result is missing
	DataAbsentReason *CodeableConcept `json:"dataAbsentReason,omitempty"`
	// High, low, normal, etc.
	Interpretation []CodeableConcept `json:"interpretation,omitempty"`
	// Comments about the observation
	Note []Annotation `json:"note,omitempty"`
	// Observed body part
	BodySite *CodeableConcept `json:"bodySite,omitempty"`
	// How it was done
	Method *CodeableConcept `json:"method,omitempty"`
	// Specimen used for this observation
	Specimen *Reference `json:"specimen,omitempty"`
	// (Measurement) Device
	Device *Reference `json:"device,omitempty"`
	// Provides guide for interpretation
	ReferenceRange []ObservationReferenceRange `json:"referenceRange,omitempty"`
	// Related resource that belongs to the Observation group
	HasMember []Reference `json:"hasMember,omitempty"`
	// Related measurements the observation is made from
	DerivedFrom []Reference `json:"derivedFrom,omitempty"`
	// Component results
	Component []ObservationComponent `json:"component,omitempty"`
}
