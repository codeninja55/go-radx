package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedicationDispense is the FHIR resource type name for MedicationDispense.
const ResourceTypeMedicationDispense = "MedicationDispense"

// MedicationDispensePerformer represents a FHIR BackboneElement for MedicationDispense.performer.
type MedicationDispensePerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Who performed the dispense and what they did
	Function *CodeableConcept `json:"function,omitempty"`
	// Individual who was performing
	Actor Reference `json:"actor"`
}

// MedicationDispenseSubstitution represents a FHIR BackboneElement for MedicationDispense.substitution.
type MedicationDispenseSubstitution struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Whether a substitution was or was not performed on the dispense
	WasSubstituted bool `json:"wasSubstituted"`
	// Code signifying whether a different drug was dispensed from what was prescribed
	Type *CodeableConcept `json:"type,omitempty"`
	// Why was substitution made
	Reason []CodeableConcept `json:"reason,omitempty"`
	// Who is responsible for the substitution
	ResponsibleParty []Reference `json:"responsibleParty,omitempty"`
}

// MedicationDispense represents a FHIR MedicationDispense.
type MedicationDispense struct {
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
	// Event that dispense is part of
	PartOf []Reference `json:"partOf,omitempty"`
	// preparation | in-progress | cancelled | on-hold | completed | entered-in-error | stopped | declined | unknown
	Status string `json:"status"`
	// Why a dispense was not performed
	StatusReason *any `json:"statusReason,omitempty"`
	// Type of medication dispense
	Category *CodeableConcept `json:"category,omitempty"`
	// What medication was supplied
	Medication any `json:"medication"`
	// Who the dispense is for
	Subject *Reference `json:"subject,omitempty"`
	// Encounter / Episode associated with event
	Context *Reference `json:"context,omitempty"`
	// Information that supports the dispensing of the medication
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
	// Who performed event
	Performer []MedicationDispensePerformer `json:"performer,omitempty"`
	// Where the dispense occurred
	Location *Reference `json:"location,omitempty"`
	// Medication order that authorizes the dispense
	AuthorizingPrescription []Reference `json:"authorizingPrescription,omitempty"`
	// Trial fill, partial fill, emergency fill, etc.
	Type *CodeableConcept `json:"type,omitempty"`
	// Amount dispensed
	Quantity *Quantity `json:"quantity,omitempty"`
	// Amount of medication expressed as a timing amount
	DaysSupply *Quantity `json:"daysSupply,omitempty"`
	// When product was packaged and reviewed
	WhenPrepared *primitives.DateTime `json:"whenPrepared,omitempty"`
	// When product was given out
	WhenHandedOver *primitives.DateTime `json:"whenHandedOver,omitempty"`
	// Where the medication was sent
	Destination *Reference `json:"destination,omitempty"`
	// Who collected the medication
	Receiver []Reference `json:"receiver,omitempty"`
	// Information about the dispense
	Note []Annotation `json:"note,omitempty"`
	// How the medication is to be used by the patient or administered by the caregiver
	DosageInstruction []Dosage `json:"dosageInstruction,omitempty"`
	// Whether a substitution was performed on the dispense
	Substitution *MedicationDispenseSubstitution `json:"substitution,omitempty"`
	// Clinical issue with action
	DetectedIssue []Reference `json:"detectedIssue,omitempty"`
	// A list of relevant lifecycle events
	EventHistory []Reference `json:"eventHistory,omitempty"`
}
