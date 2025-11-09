package resources

// ResourceTypeEncounter is the FHIR resource type name for Encounter.
const ResourceTypeEncounter = "Encounter"

// EncounterStatusHistory represents a FHIR BackboneElement for Encounter.statusHistory.
type EncounterStatusHistory struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// planned | arrived | triaged | in-progress | onleave | finished | cancelled +
	Status string `json:"status"`
	// The time that the episode was in the specified status
	Period Period `json:"period"`
}

// EncounterClassHistory represents a FHIR BackboneElement for Encounter.classHistory.
type EncounterClassHistory struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// inpatient | outpatient | ambulatory | emergency +
	Class Coding `json:"class"`
	// The time that the episode was in the specified class
	Period Period `json:"period"`
}

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
	// Persons involved in the encounter other than the patient
	Individual *Reference `json:"individual,omitempty"`
}

// EncounterDiagnosis represents a FHIR BackboneElement for Encounter.diagnosis.
type EncounterDiagnosis struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The diagnosis or procedure relevant to the encounter
	Condition Reference `json:"condition"`
	// Role that this diagnosis has within the encounter (e.g. admission, billing, discharge …)
	Use *CodeableConcept `json:"use,omitempty"`
	// Ranking of the diagnosis (for each role type)
	Rank *int `json:"rank,omitempty"`
}

// EncounterHospitalization represents a FHIR BackboneElement for Encounter.hospitalization.
type EncounterHospitalization struct {
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
	// The type of hospital re-admission that has occurred (if any). If the value is absent, then this is not identified as a readmission
	ReAdmission *CodeableConcept `json:"reAdmission,omitempty"`
	// Diet preferences reported by the patient
	DietPreference []CodeableConcept `json:"dietPreference,omitempty"`
	// Special courtesies (VIP, board member)
	SpecialCourtesy []CodeableConcept `json:"specialCourtesy,omitempty"`
	// Wheelchair, translator, stretcher, etc.
	SpecialArrangement []CodeableConcept `json:"specialArrangement,omitempty"`
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
	// The physical type of the location (usually the level in the location hierachy - bed room ward etc.)
	PhysicalType *CodeableConcept `json:"physicalType,omitempty"`
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
	// planned | arrived | triaged | in-progress | onleave | finished | cancelled +
	Status string `json:"status"`
	// List of past encounter statuses
	StatusHistory []EncounterStatusHistory `json:"statusHistory,omitempty"`
	// Classification of patient encounter
	Class Coding `json:"class"`
	// List of past encounter classes
	ClassHistory []EncounterClassHistory `json:"classHistory,omitempty"`
	// Specific type of encounter
	Type []CodeableConcept `json:"type,omitempty"`
	// Specific type of service
	ServiceType *CodeableConcept `json:"serviceType,omitempty"`
	// Indicates the urgency of the encounter
	Priority *CodeableConcept `json:"priority,omitempty"`
	// The patient or group present at the encounter
	Subject *Reference `json:"subject,omitempty"`
	// Episode(s) of care that this encounter should be recorded against
	EpisodeOfCare []Reference `json:"episodeOfCare,omitempty"`
	// The ServiceRequest that initiated this encounter
	BasedOn []Reference `json:"basedOn,omitempty"`
	// List of participants involved in the encounter
	Participant []EncounterParticipant `json:"participant,omitempty"`
	// The appointment that scheduled this encounter
	Appointment []Reference `json:"appointment,omitempty"`
	// The start and end time of the encounter
	Period *Period `json:"period,omitempty"`
	// Quantity of time the encounter lasted (less time absent)
	Length *Duration `json:"length,omitempty"`
	// Coded reason the encounter takes place
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Reason the encounter takes place (reference)
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// The list of diagnosis relevant to this encounter
	Diagnosis []EncounterDiagnosis `json:"diagnosis,omitempty"`
	// The set of accounts that may be used for billing for this Encounter
	Account []Reference `json:"account,omitempty"`
	// Details about the admission to a healthcare service
	Hospitalization *EncounterHospitalization `json:"hospitalization,omitempty"`
	// List of locations where the patient has been
	Location []EncounterLocation `json:"location,omitempty"`
	// The organization (facility) responsible for this encounter
	ServiceProvider *Reference `json:"serviceProvider,omitempty"`
	// Another Encounter this encounter is part of
	PartOf *Reference `json:"partOf,omitempty"`
}
