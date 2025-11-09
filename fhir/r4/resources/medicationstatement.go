package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedicationStatement is the FHIR resource type name for MedicationStatement.
const ResourceTypeMedicationStatement = "MedicationStatement"

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
	// Fulfils plan, proposal or order
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// active | completed | entered-in-error | intended | stopped | on-hold | unknown | not-taken
	Status string `json:"status"`
	// Reason for current status
	StatusReason []CodeableConcept `json:"statusReason,omitempty"`
	// Type of medication usage
	Category *CodeableConcept `json:"category,omitempty"`
	// What medication was taken
	Medication any `json:"medication"`
	// Who is/was taking  the medication
	Subject Reference `json:"subject"`
	// Encounter / Episode associated with MedicationStatement
	Context *Reference `json:"context,omitempty"`
	// The date/time or interval when the medication is/was/will be taken
	Effective *any `json:"effective,omitempty"`
	// When the statement was asserted?
	DateAsserted *primitives.DateTime `json:"dateAsserted,omitempty"`
	// Person or organization that provided the information about the taking of this medication
	InformationSource *Reference `json:"informationSource,omitempty"`
	// Additional supporting information
	DerivedFrom []Reference `json:"derivedFrom,omitempty"`
	// Reason for why the medication is being/was taken
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Condition or observation that supports why the medication is being/was taken
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Further information about the statement
	Note []Annotation `json:"note,omitempty"`
	// Details of how medication is/was taken or should be taken
	Dosage []Dosage `json:"dosage,omitempty"`
}
