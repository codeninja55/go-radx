package resources

// ResourceTypeEpisodeOfCare is the FHIR resource type name for EpisodeOfCare.
const ResourceTypeEpisodeOfCare = "EpisodeOfCare"

// EpisodeOfCareStatusHistory represents a FHIR BackboneElement for EpisodeOfCare.statusHistory.
type EpisodeOfCareStatusHistory struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// planned | waitlist | active | onhold | finished | cancelled | entered-in-error
	Status string `json:"status"`
	// Duration the EpisodeOfCare was in the specified status
	Period Period `json:"period"`
}

// EpisodeOfCareDiagnosis represents a FHIR BackboneElement for EpisodeOfCare.diagnosis.
type EpisodeOfCareDiagnosis struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Conditions/problems/diagnoses this episode of care is for
	Condition Reference `json:"condition"`
	// Role that this diagnosis has within the episode of care (e.g. admission, billing, discharge …)
	Role *CodeableConcept `json:"role,omitempty"`
	// Ranking of the diagnosis (for each role type)
	Rank *int `json:"rank,omitempty"`
}

// EpisodeOfCare represents a FHIR EpisodeOfCare.
type EpisodeOfCare struct {
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
	// Business Identifier(s) relevant for this EpisodeOfCare
	Identifier []Identifier `json:"identifier,omitempty"`
	// planned | waitlist | active | onhold | finished | cancelled | entered-in-error
	Status string `json:"status"`
	// Past list of status codes (the current status may be included to cover the start date of the status)
	StatusHistory []EpisodeOfCareStatusHistory `json:"statusHistory,omitempty"`
	// Type/class  - e.g. specialist referral, disease management
	Type []CodeableConcept `json:"type,omitempty"`
	// The list of diagnosis relevant to this episode of care
	Diagnosis []EpisodeOfCareDiagnosis `json:"diagnosis,omitempty"`
	// The patient who is the focus of this episode of care
	Patient Reference `json:"patient"`
	// Organization that assumes care
	ManagingOrganization *Reference `json:"managingOrganization,omitempty"`
	// Interval during responsibility is assumed
	Period *Period `json:"period,omitempty"`
	// Originating Referral Request(s)
	ReferralRequest []Reference `json:"referralRequest,omitempty"`
	// Care manager/care coordinator for the patient
	CareManager *Reference `json:"careManager,omitempty"`
	// Other practitioners facilitating this episode of care
	Team []Reference `json:"team,omitempty"`
	// The set of accounts that may be used for billing for this EpisodeOfCare
	Account []Reference `json:"account,omitempty"`
}
