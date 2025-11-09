package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeMedicationRequest is the FHIR resource type name for MedicationRequest.
const ResourceTypeMedicationRequest = "MedicationRequest"

// MedicationRequestDispenseRequestInitialFill represents a FHIR BackboneElement for MedicationRequest.dispenseRequest.initialFill.
type MedicationRequestDispenseRequestInitialFill struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// First fill quantity
	Quantity *Quantity `json:"quantity,omitempty"`
	// First fill duration
	Duration *Duration `json:"duration,omitempty"`
}

// MedicationRequestDispenseRequest represents a FHIR BackboneElement for MedicationRequest.dispenseRequest.
type MedicationRequestDispenseRequest struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// First fill details
	InitialFill *MedicationRequestDispenseRequestInitialFill `json:"initialFill,omitempty"`
	// Minimum period of time between dispenses
	DispenseInterval *Duration `json:"dispenseInterval,omitempty"`
	// Time period supply is authorized for
	ValidityPeriod *Period `json:"validityPeriod,omitempty"`
	// Number of refills authorized
	NumberOfRepeatsAllowed *uint `json:"numberOfRepeatsAllowed,omitempty"`
	// Amount of medication to supply per dispense
	Quantity *Quantity `json:"quantity,omitempty"`
	// Number of days supply per dispense
	ExpectedSupplyDuration *Duration `json:"expectedSupplyDuration,omitempty"`
	// Intended performer of dispense
	Dispenser *Reference `json:"dispenser,omitempty"`
	// Additional information for the dispenser
	DispenserInstruction []Annotation `json:"dispenserInstruction,omitempty"`
	// Type of adherence packaging to use for the dispense
	DoseAdministrationAid *CodeableConcept `json:"doseAdministrationAid,omitempty"`
}

// MedicationRequestSubstitution represents a FHIR BackboneElement for MedicationRequest.substitution.
type MedicationRequestSubstitution struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Whether substitution is allowed or not
	Allowed any `json:"allowed"`
	// Why should (not) substitution be made
	Reason *CodeableConcept `json:"reason,omitempty"`
}

// MedicationRequest represents a FHIR MedicationRequest.
type MedicationRequest struct {
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
	// External ids for this request
	Identifier []Identifier `json:"identifier,omitempty"`
	// A plan or request that is fulfilled in whole or in part by this medication request
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Reference to an order/prescription that is being replaced by this MedicationRequest
	PriorPrescription *Reference `json:"priorPrescription,omitempty"`
	// Composite request this is part of
	GroupIdentifier *Identifier `json:"groupIdentifier,omitempty"`
	// active | on-hold | ended | stopped | completed | cancelled | entered-in-error | draft | unknown
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// When the status was changed
	StatusChanged *primitives.DateTime `json:"statusChanged,omitempty"`
	// proposal | plan | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// Grouping or category of medication request
	Category []CodeableConcept `json:"category,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// True if patient is to stop taking or not to start taking the medication
	DoNotPerform *bool `json:"doNotPerform,omitempty"`
	// Medication to be taken
	Medication CodeableReference `json:"medication"`
	// Individual or group for whom the medication has been requested
	Subject Reference `json:"subject"`
	// The person or organization who provided the information about this request, if the source is someone other than the requestor
	InformationSource []Reference `json:"informationSource,omitempty"`
	// Encounter created as part of encounter/admission/stay
	Encounter *Reference `json:"encounter,omitempty"`
	// Information to support fulfilling of the medication
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
	// When request was initially authored
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Who/What requested the Request
	Requester *Reference `json:"requester,omitempty"`
	// Reported rather than primary record
	Reported *bool `json:"reported,omitempty"`
	// Desired kind of performer of the medication administration
	PerformerType *CodeableConcept `json:"performerType,omitempty"`
	// Intended performer of administration
	Performer []Reference `json:"performer,omitempty"`
	// Intended type of device for the administration
	Device []CodeableReference `json:"device,omitempty"`
	// Person who entered the request
	Recorder *Reference `json:"recorder,omitempty"`
	// Reason or indication for ordering or not ordering the medication
	Reason []CodeableReference `json:"reason,omitempty"`
	// Overall pattern of medication administration
	CourseOfTherapyType *CodeableConcept `json:"courseOfTherapyType,omitempty"`
	// Associated insurance coverage
	Insurance []Reference `json:"insurance,omitempty"`
	// Information about the prescription
	Note []Annotation `json:"note,omitempty"`
	// Full representation of the dosage instructions
	RenderedDosageInstruction *string `json:"renderedDosageInstruction,omitempty"`
	// Period over which the medication is to be taken
	EffectiveDosePeriod *Period `json:"effectiveDosePeriod,omitempty"`
	// Specific instructions for how the medication should be taken
	DosageInstruction []Dosage `json:"dosageInstruction,omitempty"`
	// Medication supply authorization
	DispenseRequest *MedicationRequestDispenseRequest `json:"dispenseRequest,omitempty"`
	// Any restrictions on medication substitution
	Substitution *MedicationRequestSubstitution `json:"substitution,omitempty"`
	// A list of events of interest in the lifecycle
	EventHistory []Reference `json:"eventHistory,omitempty"`
}
