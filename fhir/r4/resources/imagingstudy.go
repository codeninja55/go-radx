package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeImagingStudy is the FHIR resource type name for ImagingStudy.
const ResourceTypeImagingStudy = "ImagingStudy"

// ImagingStudySeriesPerformer represents a FHIR BackboneElement for ImagingStudy.series.performer.
type ImagingStudySeriesPerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of performance
	Function *CodeableConcept `json:"function,omitempty"`
	// Who performed the series
	Actor Reference `json:"actor"`
}

// ImagingStudySeriesInstance represents a FHIR BackboneElement for ImagingStudy.series.instance.
type ImagingStudySeriesInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// DICOM SOP Instance UID
	Uid string `json:"uid"`
	// DICOM class type
	SopClass Coding `json:"sopClass"`
	// The number of this instance in the series
	Number *uint `json:"number,omitempty"`
	// Description of instance
	Title *string `json:"title,omitempty"`
}

// ImagingStudySeries represents a FHIR BackboneElement for ImagingStudy.series.
type ImagingStudySeries struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// DICOM Series Instance UID for the series
	Uid string `json:"uid"`
	// Numeric identifier of this series
	Number *uint `json:"number,omitempty"`
	// The modality of the instances in the series
	Modality Coding `json:"modality"`
	// A short human readable summary of the series
	Description *string `json:"description,omitempty"`
	// Number of Series Related Instances
	NumberOfInstances *uint `json:"numberOfInstances,omitempty"`
	// Series access endpoint
	Endpoint []Reference `json:"endpoint,omitempty"`
	// Body part examined
	BodySite *Coding `json:"bodySite,omitempty"`
	// Body part laterality
	Laterality *Coding `json:"laterality,omitempty"`
	// Specimen imaged
	Specimen []Reference `json:"specimen,omitempty"`
	// When the series started
	Started *primitives.DateTime `json:"started,omitempty"`
	// Who performed the series
	Performer []ImagingStudySeriesPerformer `json:"performer,omitempty"`
	// A single SOP instance from the series
	Instance []ImagingStudySeriesInstance `json:"instance,omitempty"`
}

// ImagingStudy represents a FHIR ImagingStudy.
type ImagingStudy struct {
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
	// Identifiers for the whole study
	Identifier []Identifier `json:"identifier,omitempty"`
	// registered | available | cancelled | entered-in-error | unknown
	Status string `json:"status"`
	// All series modality if actual acquisition modalities
	Modality []Coding `json:"modality,omitempty"`
	// Who or what is the subject of the study
	Subject Reference `json:"subject"`
	// Encounter with which this imaging study is associated
	Encounter *Reference `json:"encounter,omitempty"`
	// When the study was started
	Started *primitives.DateTime `json:"started,omitempty"`
	// Request fulfilled
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Referring physician
	Referrer *Reference `json:"referrer,omitempty"`
	// Who interpreted images
	Interpreter []Reference `json:"interpreter,omitempty"`
	// Study access endpoint
	Endpoint []Reference `json:"endpoint,omitempty"`
	// Number of Study Related Series
	NumberOfSeries *uint `json:"numberOfSeries,omitempty"`
	// Number of Study Related Instances
	NumberOfInstances *uint `json:"numberOfInstances,omitempty"`
	// The performed Procedure reference
	ProcedureReference *Reference `json:"procedureReference,omitempty"`
	// The performed procedure code
	ProcedureCode []CodeableConcept `json:"procedureCode,omitempty"`
	// Where ImagingStudy occurred
	Location *Reference `json:"location,omitempty"`
	// Why the study was requested
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Why was study performed
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// User-defined comments
	Note []Annotation `json:"note,omitempty"`
	// Institution-generated description
	Description *string `json:"description,omitempty"`
	// Each study has one or more series of instances
	Series []ImagingStudySeries `json:"series,omitempty"`
}
