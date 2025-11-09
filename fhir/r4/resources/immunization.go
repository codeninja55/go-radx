package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeImmunization is the FHIR resource type name for Immunization.
const ResourceTypeImmunization = "Immunization"

// ImmunizationPerformer represents a FHIR BackboneElement for Immunization.performer.
type ImmunizationPerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What type of performance was done
	Function *CodeableConcept `json:"function,omitempty"`
	// Individual or organization who was performing
	Actor Reference `json:"actor"`
}

// ImmunizationEducation represents a FHIR BackboneElement for Immunization.education.
type ImmunizationEducation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Educational material document identifier
	DocumentType *string `json:"documentType,omitempty"`
	// Educational material reference pointer
	Reference *string `json:"reference,omitempty"`
	// Educational material publication date
	PublicationDate *primitives.DateTime `json:"publicationDate,omitempty"`
	// Educational material presentation date
	PresentationDate *primitives.DateTime `json:"presentationDate,omitempty"`
}

// ImmunizationReaction represents a FHIR BackboneElement for Immunization.reaction.
type ImmunizationReaction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// When reaction started
	Date *primitives.DateTime `json:"date,omitempty"`
	// Additional information on reaction
	Detail *Reference `json:"detail,omitempty"`
	// Indicates self-reported reaction
	Reported *bool `json:"reported,omitempty"`
}

// ImmunizationProtocolApplied represents a FHIR BackboneElement for Immunization.protocolApplied.
type ImmunizationProtocolApplied struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of vaccine series
	Series *string `json:"series,omitempty"`
	// Who is responsible for publishing the recommendations
	Authority *Reference `json:"authority,omitempty"`
	// Vaccine preventatable disease being targetted
	TargetDisease []CodeableConcept `json:"targetDisease,omitempty"`
	// Dose number within series
	DoseNumber any `json:"doseNumber"`
	// Recommended number of doses for immunity
	SeriesDoses *any `json:"seriesDoses,omitempty"`
}

// Immunization represents a FHIR Immunization.
type Immunization struct {
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
	// Business identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// completed | entered-in-error | not-done
	Status string `json:"status"`
	// Reason not done
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// Vaccine product administered
	VaccineCode CodeableConcept `json:"vaccineCode"`
	// Who was immunized
	Patient Reference `json:"patient"`
	// Encounter immunization was part of
	Encounter *Reference `json:"encounter,omitempty"`
	// Vaccine administration date
	Occurrence any `json:"occurrence"`
	// When the immunization was first captured in the subject's record
	Recorded *primitives.DateTime `json:"recorded,omitempty"`
	// Indicates context the data was recorded in
	PrimarySource *bool `json:"primarySource,omitempty"`
	// Indicates the source of a secondarily reported record
	ReportOrigin *CodeableConcept `json:"reportOrigin,omitempty"`
	// Where immunization occurred
	Location *Reference `json:"location,omitempty"`
	// Vaccine manufacturer
	Manufacturer *Reference `json:"manufacturer,omitempty"`
	// Vaccine lot number
	LotNumber *string `json:"lotNumber,omitempty"`
	// Vaccine expiration date
	ExpirationDate *primitives.Date `json:"expirationDate,omitempty"`
	// Body site vaccine  was administered
	Site *CodeableConcept `json:"site,omitempty"`
	// How vaccine entered body
	Route *CodeableConcept `json:"route,omitempty"`
	// Amount of vaccine administered
	DoseQuantity *Quantity `json:"doseQuantity,omitempty"`
	// Who performed event
	Performer []ImmunizationPerformer `json:"performer,omitempty"`
	// Additional immunization notes
	Note []Annotation `json:"note,omitempty"`
	// Why immunization occurred
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Why immunization occurred
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Dose potency
	IsSubpotent *bool `json:"isSubpotent,omitempty"`
	// Reason for being subpotent
	SubpotentReason []CodeableConcept `json:"subpotentReason,omitempty"`
	// Educational material presented to patient
	Education []ImmunizationEducation `json:"education,omitempty"`
	// Patient eligibility for a vaccination program
	ProgramEligibility []CodeableConcept `json:"programEligibility,omitempty"`
	// Funding source for the vaccine
	FundingSource *CodeableConcept `json:"fundingSource,omitempty"`
	// Details of a reaction that follows immunization
	Reaction []ImmunizationReaction `json:"reaction,omitempty"`
	// Protocol followed by the provider
	ProtocolApplied []ImmunizationProtocolApplied `json:"protocolApplied,omitempty"`
}
