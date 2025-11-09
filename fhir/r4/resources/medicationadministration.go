package resources

// ResourceTypeMedicationAdministration is the FHIR resource type name for MedicationAdministration.
const ResourceTypeMedicationAdministration = "MedicationAdministration"

// MedicationAdministrationPerformer represents a FHIR BackboneElement for MedicationAdministration.performer.
type MedicationAdministrationPerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of performance
	Function *CodeableConcept `json:"function,omitempty"`
	// Who performed the medication administration
	Actor Reference `json:"actor"`
}

// MedicationAdministrationDosage represents a FHIR BackboneElement for MedicationAdministration.dosage.
type MedicationAdministrationDosage struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Free text dosage instructions e.g. SIG
	Text *string `json:"text,omitempty"`
	// Body site administered to
	Site *CodeableConcept `json:"site,omitempty"`
	// Path of substance into body
	Route *CodeableConcept `json:"route,omitempty"`
	// How drug was administered
	Method *CodeableConcept `json:"method,omitempty"`
	// Amount of medication per dose
	Dose *Quantity `json:"dose,omitempty"`
	// Dose quantity per unit of time
	Rate *any `json:"rate,omitempty"`
}

// MedicationAdministration represents a FHIR MedicationAdministration.
type MedicationAdministration struct {
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
	// Instantiates protocol or definition
	Instantiates []string `json:"instantiates,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// in-progress | not-done | on-hold | completed | entered-in-error | stopped | unknown
	Status string `json:"status"`
	// Reason administration not performed
	StatusReason []CodeableConcept `json:"statusReason,omitempty"`
	// Type of medication usage
	Category *CodeableConcept `json:"category,omitempty"`
	// What was administered
	Medication any `json:"medication"`
	// Who received medication
	Subject Reference `json:"subject"`
	// Encounter or Episode of Care administered as part of
	Context *Reference `json:"context,omitempty"`
	// Additional information to support administration
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`
	// Start and end time of administration
	Effective any `json:"effective"`
	// Who performed the medication administration and what they did
	Performer []MedicationAdministrationPerformer `json:"performer,omitempty"`
	// Reason administration performed
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Condition or observation that supports why the medication was administered
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Request administration performed against
	Request *Reference `json:"request,omitempty"`
	// Device used to administer
	Device []Reference `json:"device,omitempty"`
	// Information about the administration
	Note []Annotation `json:"note,omitempty"`
	// Details of how medication was taken
	Dosage *MedicationAdministrationDosage `json:"dosage,omitempty"`
	// A list of events of interest in the lifecycle
	EventHistory []Reference `json:"eventHistory,omitempty"`
}
