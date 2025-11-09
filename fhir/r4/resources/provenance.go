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
	// Who participated
	Who Reference `json:"who"`
	// Who the agent is representing
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
	// derivation | revision | quotation | source | removal
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
	Recorded primitives.Instant `json:"recorded"`
	// Policy or plan the activity was defined by
	Policy []string `json:"policy,omitempty"`
	// Where the activity occurred, if relevant
	Location *Reference `json:"location,omitempty"`
	// Reason the activity is occurring
	Reason []CodeableConcept `json:"reason,omitempty"`
	// Activity that occurred
	Activity *CodeableConcept `json:"activity,omitempty"`
	// Actor involved
	Agent []ProvenanceAgent `json:"agent,omitempty"`
	// An entity used in this activity
	Entity []ProvenanceEntity `json:"entity,omitempty"`
	// Signature on target
	Signature []Signature `json:"signature,omitempty"`
}
