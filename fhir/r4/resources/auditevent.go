package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeAuditEvent is the FHIR resource type name for AuditEvent.
const ResourceTypeAuditEvent = "AuditEvent"

// AuditEventAgentNetwork represents a FHIR BackboneElement for AuditEvent.agent.network.
type AuditEventAgentNetwork struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Identifier for the network access point of the user device
	Address *string `json:"address,omitempty"`
	// The type of network access point
	Type *string `json:"type,omitempty"`
}

// AuditEventAgent represents a FHIR BackboneElement for AuditEvent.agent.
type AuditEventAgent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// How agent participated
	Type *CodeableConcept `json:"type,omitempty"`
	// Agent role in the event
	Role []CodeableConcept `json:"role,omitempty"`
	// Identifier of who
	Who *Reference `json:"who,omitempty"`
	// Alternative User identity
	AltId *string `json:"altId,omitempty"`
	// Human friendly name for the agent
	Name *string `json:"name,omitempty"`
	// Whether user is initiator
	Requestor bool `json:"requestor"`
	// Where
	Location *Reference `json:"location,omitempty"`
	// Policy that authorized event
	Policy []string `json:"policy,omitempty"`
	// Type of media
	Media *Coding `json:"media,omitempty"`
	// Logical network location for application activity
	Network *AuditEventAgentNetwork `json:"network,omitempty"`
	// Reason given for this user
	PurposeOfUse []CodeableConcept `json:"purposeOfUse,omitempty"`
}

// AuditEventSource represents a FHIR BackboneElement for AuditEvent.source.
type AuditEventSource struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Logical source location within the enterprise
	Site *string `json:"site,omitempty"`
	// The identity of source detecting the event
	Observer Reference `json:"observer"`
	// The type of source where event originated
	Type []Coding `json:"type,omitempty"`
}

// AuditEventEntityDetail represents a FHIR BackboneElement for AuditEvent.entity.detail.
type AuditEventEntityDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name of the property
	Type string `json:"type"`
	// Property value
	Value any `json:"value"`
}

// AuditEventEntity represents a FHIR BackboneElement for AuditEvent.entity.
type AuditEventEntity struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Specific instance of resource
	What *Reference `json:"what,omitempty"`
	// Type of entity involved
	Type *Coding `json:"type,omitempty"`
	// What role the entity played
	Role *Coding `json:"role,omitempty"`
	// Life-cycle stage for the entity
	Lifecycle *Coding `json:"lifecycle,omitempty"`
	// Security labels on the entity
	SecurityLabel []Coding `json:"securityLabel,omitempty"`
	// Descriptor for entity
	Name *string `json:"name,omitempty"`
	// Descriptive text
	Description *string `json:"description,omitempty"`
	// Query parameters
	Query *string `json:"query,omitempty"`
	// Additional Information about the entity
	Detail []AuditEventEntityDetail `json:"detail,omitempty"`
}

// AuditEvent represents a FHIR AuditEvent.
type AuditEvent struct {
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
	// Type/identifier of event
	Type Coding `json:"type"`
	// More specific type/id for the event
	Subtype []Coding `json:"subtype,omitempty"`
	// Type of action performed during the event
	Action *string `json:"action,omitempty"`
	// When the activity occurred
	Period *Period `json:"period,omitempty"`
	// Time when the event was recorded
	Recorded primitives.Instant `json:"recorded"`
	// Whether the event succeeded or failed
	Outcome *string `json:"outcome,omitempty"`
	// Description of the event outcome
	OutcomeDesc *string `json:"outcomeDesc,omitempty"`
	// The purposeOfUse of the event
	PurposeOfEvent []CodeableConcept `json:"purposeOfEvent,omitempty"`
	// Actor involved in the event
	Agent []AuditEventAgent `json:"agent,omitempty"`
	// Audit Event Reporter
	Source AuditEventSource `json:"source"`
	// Data or objects used
	Entity []AuditEventEntity `json:"entity,omitempty"`
}
