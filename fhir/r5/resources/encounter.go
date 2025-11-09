package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeEncounter is the FHIR resource type name for Encounter.
const ResourceTypeEncounter = "Encounter"

// EncounterParticipant represents a FHIR BackboneElement for Encounter.participant.
type EncounterParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Role of participant in encounter
	Type []CodeableConcept `json:"type,omitempty"`
	// Period of time during the encounter that the participant participated
	Period *Period `json:"period,omitempty"`
	// The individual, device, or service participating in the encounter
	Actor *Reference `json:"actor,omitempty"`
}

// EncounterReason represents a FHIR BackboneElement for Encounter.reason.
type EncounterReason struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What the reason value should be used for/as
	Use []CodeableConcept `json:"use,omitempty"`
	// Reason the encounter takes place (core or reference)
	Value []CodeableReference `json:"value,omitempty"`
}

// EncounterDiagnosis represents a FHIR BackboneElement for Encounter.diagnosis.
type EncounterDiagnosis struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The diagnosis relevant to the encounter
	Condition []CodeableReference `json:"condition,omitempty"`
	// Role that this diagnosis has within the encounter (e.g. admission, billing, discharge …)
	Use []CodeableConcept `json:"use,omitempty"`
}

// EncounterAdmission represents a FHIR BackboneElement for Encounter.admission.
type EncounterAdmission struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Pre-admission identifier
	PreAdmissionIdentifier *Identifier `json:"preAdmissionIdentifier,omitempty"`
	// The location/organization from which the patient came before admission
	Origin *Reference `json:"origin,omitempty"`
	// From where patient was admitted (physician referral, transfer)
	AdmitSource *CodeableConcept `json:"admitSource,omitempty"`
	// Indicates that the patient is being re-admitted
	ReAdmission *CodeableConcept `json:"reAdmission,omitempty"`
	// Location/organization to which the patient is discharged
	Destination *Reference `json:"destination,omitempty"`
	// Category or kind of location after discharge
	DischargeDisposition *CodeableConcept `json:"dischargeDisposition,omitempty"`
}

// EncounterLocation represents a FHIR BackboneElement for Encounter.location.
type EncounterLocation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Location the encounter takes place
	Location Reference `json:"location"`
	// planned | active | reserved | completed
	Status *string `json:"status,omitempty"`
	// The physical type of the location (usually the level in the location hierarchy - bed, room, ward, virtual etc.)
	Form *CodeableConcept `json:"form,omitempty"`
	// Time period during which the patient was present at the location
	Period *Period `json:"period,omitempty"`
}

// Encounter represents a FHIR Encounter.
type Encounter struct {
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
	// Identifier(s) by which this encounter is known
	Identifier []Identifier `json:"identifier,omitempty"`
	// planned | in-progress | on-hold | discharged | completed | cancelled | discontinued | entered-in-error | unknown
	Status string `json:"status"`
	// Classification of patient encounter context - e.g. Inpatient, outpatient
	Class []CodeableConcept `json:"class,omitempty"`
	// Indicates the urgency of the encounter
	Priority *CodeableConcept `json:"priority,omitempty"`
	// Specific type of encounter (e.g. e-mail consultation, surgical day-care, ...)
	Type []CodeableConcept `json:"type,omitempty"`
	// Specific type of service
	ServiceType []CodeableReference `json:"serviceType,omitempty"`
	// The patient or group related to this encounter
	Subject *Reference `json:"subject,omitempty"`
	// The current status of the subject in relation to the Encounter
	SubjectStatus *CodeableConcept `json:"subjectStatus,omitempty"`
	// Episode(s) of care that this encounter should be recorded against
	EpisodeOfCare []Reference `json:"episodeOfCare,omitempty"`
	// The request that initiated this encounter
	BasedOn []Reference `json:"basedOn,omitempty"`
	// The group(s) that are allocated to participate in this encounter
	CareTeam []Reference `json:"careTeam,omitempty"`
	// Another Encounter this encounter is part of
	PartOf *Reference `json:"partOf,omitempty"`
	// The organization (facility) responsible for this encounter
	ServiceProvider *Reference `json:"serviceProvider,omitempty"`
	// List of participants involved in the encounter
	Participant []EncounterParticipant `json:"participant,omitempty"`
	// The appointment that scheduled this encounter
	Appointment []Reference `json:"appointment,omitempty"`
	// Connection details of a virtual service (e.g. conference call)
	VirtualService []VirtualServiceDetail `json:"virtualService,omitempty"`
	// The actual start and end time of the encounter
	ActualPeriod *Period `json:"actualPeriod,omitempty"`
	// The planned start date/time (or admission date) of the encounter
	PlannedStartDate *primitives.DateTime `json:"plannedStartDate,omitempty"`
	// The planned end date/time (or discharge date) of the encounter
	PlannedEndDate *primitives.DateTime `json:"plannedEndDate,omitempty"`
	// Actual quantity of time the encounter lasted (less time absent)
	Length *Duration `json:"length,omitempty"`
	// The list of medical reasons that are expected to be addressed during the episode of care
	Reason []EncounterReason `json:"reason,omitempty"`
	// The list of diagnosis relevant to this encounter
	Diagnosis []EncounterDiagnosis `json:"diagnosis,omitempty"`
	// The set of accounts that may be used for billing for this Encounter
	Account []Reference `json:"account,omitempty"`
	// Diet preferences reported by the patient
	DietPreference []CodeableConcept `json:"dietPreference,omitempty"`
	// Wheelchair, translator, stretcher, etc
	SpecialArrangement []CodeableConcept `json:"specialArrangement,omitempty"`
	// Special courtesies (VIP, board member)
	SpecialCourtesy []CodeableConcept `json:"specialCourtesy,omitempty"`
	// Details about the admission to a healthcare service
	Admission *EncounterAdmission `json:"admission,omitempty"`
	// List of locations where the patient has been
	Location []EncounterLocation `json:"location,omitempty"`
}
