package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeGoal is the FHIR resource type name for Goal.
const ResourceTypeGoal = "Goal"

// GoalTarget represents a FHIR BackboneElement for Goal.target.
type GoalTarget struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The parameter whose value is being tracked
	Measure *CodeableConcept `json:"measure,omitempty"`
	// The target value to be achieved
	Detail *any `json:"detail,omitempty"`
	// Reach goal on or before
	Due *any `json:"due,omitempty"`
}

// Goal represents a FHIR Goal.
type Goal struct {
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
	// External Ids for this goal
	Identifier []Identifier `json:"identifier,omitempty"`
	// proposed | planned | accepted | active | on-hold | completed | cancelled | entered-in-error | rejected
	LifecycleStatus string `json:"lifecycleStatus"`
	// in-progress | improving | worsening | no-change | achieved | sustaining | not-achieved | no-progress | not-attainable
	AchievementStatus *CodeableConcept `json:"achievementStatus,omitempty"`
	// E.g. Treatment, dietary, behavioral, etc.
	Category []CodeableConcept `json:"category,omitempty"`
	// high-priority | medium-priority | low-priority
	Priority *CodeableConcept `json:"priority,omitempty"`
	// Code or text describing goal
	Description CodeableConcept `json:"description"`
	// Who this goal is intended for
	Subject Reference `json:"subject"`
	// When goal pursuit begins
	Start *any `json:"start,omitempty"`
	// Target outcome for the goal
	Target []GoalTarget `json:"target,omitempty"`
	// When goal status took effect
	StatusDate *primitives.Date `json:"statusDate,omitempty"`
	// Reason for current status
	StatusReason *string `json:"statusReason,omitempty"`
	// Who's responsible for creating Goal?
	ExpressedBy *Reference `json:"expressedBy,omitempty"`
	// Issues addressed by this goal
	Addresses []Reference `json:"addresses,omitempty"`
	// Comments about the goal
	Note []Annotation `json:"note,omitempty"`
	// What result was achieved regarding the goal?
	OutcomeCode []CodeableConcept `json:"outcomeCode,omitempty"`
	// Observation that resulted from goal
	OutcomeReference []Reference `json:"outcomeReference,omitempty"`
}
