package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeAuditEvent is the FHIR resource type name for AuditEvent.
const ResourceTypeAuditEvent = "AuditEvent"

// AuditEventOutcome represents a FHIR BackboneElement for AuditEvent.outcome.
type AuditEventOutcome struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Whether the event succeeded or failed
	Code Coding `json:"code"`
	// Additional outcome detail
	Detail []CodeableConcept `json:"detail,omitempty"`
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
	Who Reference `json:"who"`
	// Whether user is initiator
	Requestor *bool `json:"requestor,omitempty"`
	// The agent location when the event occurred
	Location *Reference `json:"location,omitempty"`
	// Policy that authorized the agent participation in the event
	Policy []string `json:"policy,omitempty"`
	// This agent network location for the activity
	Network *any `json:"network,omitempty"`
	// Allowable authorization for this agent
	Authorization []CodeableConcept `json:"authorization,omitempty"`
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
	Site *Reference `json:"site,omitempty"`
	// The identity of source detecting the event
	Observer Reference `json:"observer"`
	// The type of source where event originated
	Type []CodeableConcept `json:"type,omitempty"`
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
	Type CodeableConcept `json:"type"`
	// Property value
	Value any `json:"value"`
}

// AuditEventEntityAgent represents a FHIR BackboneElement for AuditEvent.entity.agent.
type AuditEventEntityAgent struct {
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
	// What role the entity played
	Role *CodeableConcept `json:"role,omitempty"`
	// Security labels on the entity
	SecurityLabel []CodeableConcept `json:"securityLabel,omitempty"`
	// Query parameters
	Query *string `json:"query,omitempty"`
	// Additional Information about the entity
	Detail []AuditEventEntityDetail `json:"detail,omitempty"`
	// Entity is attributed to this agent
	Agent []AuditEventEntityAgent `json:"agent,omitempty"`
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
	Category []CodeableConcept `json:"category,omitempty"`
	// Specific type of event
	Code CodeableConcept `json:"code"`
	// Type of action performed during the event
	Action *string `json:"action,omitempty"`
	// emergency | alert | critical | error | warning | notice | informational | debug
	Severity *string `json:"severity,omitempty"`
	// When the activity occurred
	Occurred *any `json:"occurred,omitempty"`
	// Time when the event was recorded
	Recorded primitives.Instant `json:"recorded"`
	// Whether the event succeeded or failed
	Outcome *AuditEventOutcome `json:"outcome,omitempty"`
	// Authorization related to the event
	Authorization []CodeableConcept `json:"authorization,omitempty"`
	// Workflow authorization within which this event occurred
	BasedOn []Reference `json:"basedOn,omitempty"`
	// The patient is the subject of the data used/created/updated/deleted during the activity
	Patient *Reference `json:"patient,omitempty"`
	// Encounter within which this event occurred or which the event is tightly associated
	Encounter *Reference `json:"encounter,omitempty"`
	// Actor involved in the event
	Agent []AuditEventAgent `json:"agent,omitempty"`
	// Audit Event Reporter
	Source AuditEventSource `json:"source"`
	// Data or objects used
	Entity []AuditEventEntity `json:"entity,omitempty"`
}
