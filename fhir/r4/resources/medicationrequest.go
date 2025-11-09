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
	// Intended dispenser
	Performer *Reference `json:"performer,omitempty"`
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
	// active | on-hold | cancelled | completed | entered-in-error | stopped | draft | unknown
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// proposal | plan | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// Type of medication usage
	Category []CodeableConcept `json:"category,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// True if request is prohibiting action
	DoNotPerform *bool `json:"doNotPerform,omitempty"`
	// Reported rather than primary record
	Reported *any `json:"reported,omitempty"`
	// Medication to be taken
	Medication any `json:"medication"`
	// Who or group medication request is for
	Subject Reference `json:"subject"`
	// Encounter created as part of encounter/admission/stay
	Encounter *Reference `json:"encounter,omitempty"`
	// Information to support ordering of the medication
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
	// When request was initially authored
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Who/What requested the Request
	Requester *Reference `json:"requester,omitempty"`
	// Intended performer of administration
	Performer *Reference `json:"performer,omitempty"`
	// Desired kind of performer of the medication administration
	PerformerType *CodeableConcept `json:"performerType,omitempty"`
	// Person who entered the request
	Recorder *Reference `json:"recorder,omitempty"`
	// Reason or indication for ordering or not ordering the medication
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Condition or observation that supports why the prescription is being written
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// What request fulfills
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Composite request this is part of
	GroupIdentifier *Identifier `json:"groupIdentifier,omitempty"`
	// Overall pattern of medication administration
	CourseOfTherapyType *CodeableConcept `json:"courseOfTherapyType,omitempty"`
	// Associated insurance coverage
	Insurance []Reference `json:"insurance,omitempty"`
	// Information about the prescription
	Note []Annotation `json:"note,omitempty"`
	// How the medication should be taken
	DosageInstruction []Dosage `json:"dosageInstruction,omitempty"`
	// Medication supply authorization
	DispenseRequest *MedicationRequestDispenseRequest `json:"dispenseRequest,omitempty"`
	// Any restrictions on medication substitution
	Substitution *MedicationRequestSubstitution `json:"substitution,omitempty"`
	// An order/prescription that is being replaced
	PriorPrescription *Reference `json:"priorPrescription,omitempty"`
	// Clinical Issue with action
	DetectedIssue []Reference `json:"detectedIssue,omitempty"`
	// A list of events of interest in the lifecycle
	EventHistory []Reference `json:"eventHistory,omitempty"`
}
