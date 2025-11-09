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

// ImmunizationProgramEligibility represents a FHIR BackboneElement for Immunization.programEligibility.
type ImmunizationProgramEligibility struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The program that eligibility is declared for
	Program CodeableConcept `json:"program"`
	// The patient's eligibility status for the program
	ProgramStatus CodeableConcept `json:"programStatus"`
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
	Manifestation *CodeableReference `json:"manifestation,omitempty"`
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
	// Vaccine preventatable disease being targeted
	TargetDisease []CodeableConcept `json:"targetDisease,omitempty"`
	// Dose number within series
	DoseNumber string `json:"doseNumber"`
	// Recommended number of doses for immunity
	SeriesDoses *string `json:"seriesDoses,omitempty"`
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
	// Authority that the immunization event is based on
	BasedOn []Reference `json:"basedOn,omitempty"`
	// completed | entered-in-error | not-done
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// Vaccine administered
	VaccineCode CodeableConcept `json:"vaccineCode"`
	// Product that was administered
	AdministeredProduct *CodeableReference `json:"administeredProduct,omitempty"`
	// Vaccine manufacturer
	Manufacturer *CodeableReference `json:"manufacturer,omitempty"`
	// Vaccine lot number
	LotNumber *string `json:"lotNumber,omitempty"`
	// Vaccine expiration date
	ExpirationDate *primitives.Date `json:"expirationDate,omitempty"`
	// Who was immunized
	Patient Reference `json:"patient"`
	// Encounter immunization was part of
	Encounter *Reference `json:"encounter,omitempty"`
	// Additional information in support of the immunization
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
	// Vaccine administration date
	Occurrence any `json:"occurrence"`
	// Indicates context the data was captured in
	PrimarySource *bool `json:"primarySource,omitempty"`
	// Indicates the source of a  reported record
	InformationSource *CodeableReference `json:"informationSource,omitempty"`
	// Where immunization occurred
	Location *Reference `json:"location,omitempty"`
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
	Reason []CodeableReference `json:"reason,omitempty"`
	// Dose potency
	IsSubpotent *bool `json:"isSubpotent,omitempty"`
	// Reason for being subpotent
	SubpotentReason []CodeableConcept `json:"subpotentReason,omitempty"`
	// Patient eligibility for a specific vaccination program
	ProgramEligibility []ImmunizationProgramEligibility `json:"programEligibility,omitempty"`
	// Funding source for the vaccine
	FundingSource *CodeableConcept `json:"fundingSource,omitempty"`
	// Details of a reaction that follows immunization
	Reaction []ImmunizationReaction `json:"reaction,omitempty"`
	// Protocol followed by the provider
	ProtocolApplied []ImmunizationProtocolApplied `json:"protocolApplied,omitempty"`
}
