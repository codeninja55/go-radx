package resources

// ResourceTypeProcedure is the FHIR resource type name for Procedure.
const ResourceTypeProcedure = "Procedure"

// ProcedurePerformer represents a FHIR BackboneElement for Procedure.performer.
type ProcedurePerformer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of performance
	Function *CodeableConcept `json:"function,omitempty"`
	// The reference to the practitioner
	Actor Reference `json:"actor"`
	// Organization the device or practitioner was acting for
	OnBehalfOf *Reference `json:"onBehalfOf,omitempty"`
}

// ProcedureFocalDevice represents a FHIR BackboneElement for Procedure.focalDevice.
type ProcedureFocalDevice struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Kind of change to device
	Action *CodeableConcept `json:"action,omitempty"`
	// Device that was changed
	Manipulated Reference `json:"manipulated"`
}

// Procedure represents a FHIR Procedure.
type Procedure struct {
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
	// External Identifiers for this procedure
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// A request for this procedure
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Part of referenced event
	PartOf []Reference `json:"partOf,omitempty"`
	// preparation | in-progress | not-done | on-hold | stopped | completed | entered-in-error | unknown
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// Classification of the procedure
	Category *CodeableConcept `json:"category,omitempty"`
	// Identification of the procedure
	Code *CodeableConcept `json:"code,omitempty"`
	// Who the procedure was performed on
	Subject Reference `json:"subject"`
	// Encounter created as part of
	Encounter *Reference `json:"encounter,omitempty"`
	// When the procedure was performed
	Performed *any `json:"performed,omitempty"`
	// Who recorded the procedure
	Recorder *Reference `json:"recorder,omitempty"`
	// Person who asserts this procedure
	Asserter *Reference `json:"asserter,omitempty"`
	// The people who performed the procedure
	Performer []ProcedurePerformer `json:"performer,omitempty"`
	// Where the procedure happened
	Location *Reference `json:"location,omitempty"`
	// Coded reason procedure performed
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// The justification that the procedure was performed
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Target body sites
	BodySite []CodeableConcept `json:"bodySite,omitempty"`
	// The result of procedure
	Outcome *CodeableConcept `json:"outcome,omitempty"`
	// Any report resulting from the procedure
	Report []Reference `json:"report,omitempty"`
	// Complication following the procedure
	Complication []CodeableConcept `json:"complication,omitempty"`
	// A condition that is a result of the procedure
	ComplicationDetail []Reference `json:"complicationDetail,omitempty"`
	// Instructions for follow up
	FollowUp []CodeableConcept `json:"followUp,omitempty"`
	// Additional information about the procedure
	Note []Annotation `json:"note,omitempty"`
	// Manipulated, implanted, or removed device
	FocalDevice []ProcedureFocalDevice `json:"focalDevice,omitempty"`
	// Items used during procedure
	UsedReference []Reference `json:"usedReference,omitempty"`
	// Coded items used during the procedure
	UsedCode []CodeableConcept `json:"usedCode,omitempty"`
}
