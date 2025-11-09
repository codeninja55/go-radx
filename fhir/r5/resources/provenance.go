package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeProvenance is the FHIR resource type name for Provenance.
const ResourceTypeProvenance = "Provenance"

// ProvenanceAgent represents a FHIR BackboneElement for Provenance.agent.
type ProvenanceAgent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// How the agent participated
	Type *CodeableConcept `json:"type,omitempty"`
	// What the agents role was
	Role []CodeableConcept `json:"role,omitempty"`
	// The agent that participated in the event
	Who Reference `json:"who"`
	// The agent that delegated
	OnBehalfOf *Reference `json:"onBehalfOf,omitempty"`
}

// ProvenanceEntityAgent represents a FHIR BackboneElement for Provenance.entity.agent.
type ProvenanceEntityAgent struct {
}

// ProvenanceEntity represents a FHIR BackboneElement for Provenance.entity.
type ProvenanceEntity struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// revision | quotation | source | instantiates | removal
	Role string `json:"role"`
	// Identity of entity
	What Reference `json:"what"`
	// Entity is attributed to this agent
	Agent []ProvenanceEntityAgent `json:"agent,omitempty"`
}

// Provenance represents a FHIR Provenance.
type Provenance struct {
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
	// Target Reference(s) (usually version specific)
	Target []Reference `json:"target,omitempty"`
	// When the activity occurred
	Occurred *any `json:"occurred,omitempty"`
	// When the activity was recorded / updated
	Recorded *primitives.Instant `json:"recorded,omitempty"`
	// Policy or plan the activity was defined by
	Policy []string `json:"policy,omitempty"`
	// Where the activity occurred, if relevant
	Location *Reference `json:"location,omitempty"`
	// Authorization (purposeOfUse) related to the event
	Authorization []CodeableReference `json:"authorization,omitempty"`
	// Activity that occurred
	Activity *CodeableConcept `json:"activity,omitempty"`
	// Workflow authorization within which this event occurred
	BasedOn []Reference `json:"basedOn,omitempty"`
	// The patient is the subject of the data created/updated (.target) by the activity
	Patient *Reference `json:"patient,omitempty"`
	// Encounter within which this event occurred or which the event is tightly associated
	Encounter *Reference `json:"encounter,omitempty"`
	// Actor involved
	Agent []ProvenanceAgent `json:"agent,omitempty"`
	// An entity used in this activity
	Entity []ProvenanceEntity `json:"entity,omitempty"`
	// Signature on target
	Signature []Signature `json:"signature,omitempty"`
}
