package resources

// ResourceTypeCareTeam is the FHIR resource type name for CareTeam.
const ResourceTypeCareTeam = "CareTeam"

// CareTeamParticipant represents a FHIR BackboneElement for CareTeam.participant.
type CareTeamParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Type of involvement
	Role []CodeableConcept `json:"role,omitempty"`
	// Who is involved
	Member *Reference `json:"member,omitempty"`
	// Organization of the practitioner
	OnBehalfOf *Reference `json:"onBehalfOf,omitempty"`
	// Time period of participant
	Period *Period `json:"period,omitempty"`
}

// CareTeam represents a FHIR CareTeam.
type CareTeam struct {
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
	// External Ids for this team
	Identifier []Identifier `json:"identifier,omitempty"`
	// proposed | active | suspended | inactive | entered-in-error
	Status *string `json:"status,omitempty"`
	// Type of team
	Category []CodeableConcept `json:"category,omitempty"`
	// Name of the team, such as crisis assessment team
	Name *string `json:"name,omitempty"`
	// Who care team is for
	Subject *Reference `json:"subject,omitempty"`
	// Encounter created as part of
	Encounter *Reference `json:"encounter,omitempty"`
	// Time period team covers
	Period *Period `json:"period,omitempty"`
	// Members of the team
	Participant []CareTeamParticipant `json:"participant,omitempty"`
	// Why the care team exists
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Why the care team exists
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Organization responsible for the care team
	ManagingOrganization []Reference `json:"managingOrganization,omitempty"`
	// A contact detail for the care team (that applies to all members)
	Telecom []ContactPoint `json:"telecom,omitempty"`
	// Comments made about the CareTeam
	Note []Annotation `json:"note,omitempty"`
}
