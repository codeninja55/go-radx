package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedicationStatement is the FHIR resource type name for MedicationStatement.
const ResourceTypeMedicationStatement = "MedicationStatement"

// MedicationStatementAdherence represents a FHIR BackboneElement for MedicationStatement.adherence.
type MedicationStatementAdherence struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of adherence
	Code CodeableConcept `json:"code"`
	// Details of the reason for the current use of the medication
	Reason *CodeableConcept `json:"reason,omitempty"`
}

// MedicationStatement represents a FHIR MedicationStatement.
type MedicationStatement struct {
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
	// External identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// recorded | entered-in-error | draft
	Status string `json:"status"`
	// Type of medication statement
	Category []CodeableConcept `json:"category,omitempty"`
	// What medication was taken
	Medication CodeableReference `json:"medication"`
	// Who is/was taking  the medication
	Subject Reference `json:"subject"`
	// Encounter associated with MedicationStatement
	Encounter *Reference `json:"encounter,omitempty"`
	// The date/time or interval when the medication is/was/will be taken
	Effective *any `json:"effective,omitempty"`
	// When the usage was asserted?
	DateAsserted *primitives.DateTime `json:"dateAsserted,omitempty"`
	// Person or organization that provided the information about the taking of this medication
	InformationSource []Reference `json:"informationSource,omitempty"`
	// Link to information used to derive the MedicationStatement
	DerivedFrom []Reference `json:"derivedFrom,omitempty"`
	// Reason for why the medication is being/was taken
	Reason []CodeableReference `json:"reason,omitempty"`
	// Further information about the usage
	Note []Annotation `json:"note,omitempty"`
	// Link to information relevant to the usage of a medication
	RelatedClinicalInformation []Reference `json:"relatedClinicalInformation,omitempty"`
	// Full representation of the dosage instructions
	RenderedDosageInstruction *string `json:"renderedDosageInstruction,omitempty"`
	// Details of how medication is/was taken or should be taken
	Dosage []Dosage `json:"dosage,omitempty"`
	// Indicates whether the medication is or is not being consumed or administered
	Adherence *MedicationStatementAdherence `json:"adherence,omitempty"`
}
