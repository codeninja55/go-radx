package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSpecimen is the FHIR resource type name for Specimen.
const ResourceTypeSpecimen = "Specimen"

// SpecimenFeature represents a FHIR BackboneElement for Specimen.feature.
type SpecimenFeature struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Highlighted feature
	Type CodeableConcept `json:"type"`
	// Information about the feature
	Description string `json:"description"`
}

// SpecimenCollection represents a FHIR BackboneElement for Specimen.collection.
type SpecimenCollection struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Who collected the specimen
	Collector *Reference `json:"collector,omitempty"`
	// Collection time
	Collected *any `json:"collected,omitempty"`
	// How long it took to collect specimen
	Duration *Duration `json:"duration,omitempty"`
	// The quantity of specimen collected
	Quantity *Quantity `json:"quantity,omitempty"`
	// Technique used to perform collection
	Method *CodeableConcept `json:"method,omitempty"`
	// Device used to perform collection
	Device *CodeableReference `json:"device,omitempty"`
	// The procedure that collects the specimen
	Procedure *Reference `json:"procedure,omitempty"`
	// Anatomical collection site
	BodySite *CodeableReference `json:"bodySite,omitempty"`
	// Whether or how long patient abstained from food and/or drink
	FastingStatus *any `json:"fastingStatus,omitempty"`
}

// SpecimenProcessing represents a FHIR BackboneElement for Specimen.processing.
type SpecimenProcessing struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Textual description of procedure
	Description *string `json:"description,omitempty"`
	// Indicates the treatment step  applied to the specimen
	Method *CodeableConcept `json:"method,omitempty"`
	// Material used in the processing step
	Additive []Reference `json:"additive,omitempty"`
	// Date and time of specimen processing
	Time *any `json:"time,omitempty"`
}

// SpecimenContainer represents a FHIR BackboneElement for Specimen.container.
type SpecimenContainer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Device resource for the container
	Device Reference `json:"device"`
	// Where the container is
	Location *Reference `json:"location,omitempty"`
	// Quantity of specimen within container
	SpecimenQuantity *Quantity `json:"specimenQuantity,omitempty"`
}

// Specimen represents a FHIR Specimen.
type Specimen struct {
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
	// External Identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Identifier assigned by the lab
	AccessionIdentifier *Identifier `json:"accessionIdentifier,omitempty"`
	// available | unavailable | unsatisfactory | entered-in-error
	Status *string `json:"status,omitempty"`
	// Kind of material that forms the specimen
	Type *CodeableConcept `json:"type,omitempty"`
	// Where the specimen came from. This may be from patient(s), from a location (e.g., the source of an environmental sample), or a sampling of a substance, a biologically-derived product, or a device
	Subject *Reference `json:"subject,omitempty"`
	// The time when specimen is received by the testing laboratory
	ReceivedTime *primitives.DateTime `json:"receivedTime,omitempty"`
	// Specimen from which this specimen originated
	Parent []Reference `json:"parent,omitempty"`
	// Why the specimen was collected
	Request []Reference `json:"request,omitempty"`
	// grouped | pooled
	Combined *string `json:"combined,omitempty"`
	// The role the specimen serves
	Role []CodeableConcept `json:"role,omitempty"`
	// The physical feature of a specimen
	Feature []SpecimenFeature `json:"feature,omitempty"`
	// Collection details
	Collection *SpecimenCollection `json:"collection,omitempty"`
	// Processing and processing step details
	Processing []SpecimenProcessing `json:"processing,omitempty"`
	// Direct container of specimen (tube/slide, etc.)
	Container []SpecimenContainer `json:"container,omitempty"`
	// State of the specimen
	Condition []CodeableConcept `json:"condition,omitempty"`
	// Comments
	Note []Annotation `json:"note,omitempty"`
}
