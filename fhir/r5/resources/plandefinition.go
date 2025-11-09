package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypePlanDefinition is the FHIR resource type name for PlanDefinition.
const ResourceTypePlanDefinition = "PlanDefinition"

// PlanDefinitionGoalTarget represents a FHIR BackboneElement for PlanDefinition.goal.target.
type PlanDefinitionGoalTarget struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The parameter whose value is to be tracked
	Measure *CodeableConcept `json:"measure,omitempty"`
	// The target value to be achieved
	Detail *any `json:"detail,omitempty"`
	// Reach goal within
	Due *Duration `json:"due,omitempty"`
}

// PlanDefinitionGoal represents a FHIR BackboneElement for PlanDefinition.goal.
type PlanDefinitionGoal struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// E.g. Treatment, dietary, behavioral
	Category *CodeableConcept `json:"category,omitempty"`
	// Code or text describing the goal
	Description CodeableConcept `json:"description"`
	// high-priority | medium-priority | low-priority
	Priority *CodeableConcept `json:"priority,omitempty"`
	// When goal pursuit begins
	Start *CodeableConcept `json:"start,omitempty"`
	// What does the goal address
	Addresses []CodeableConcept `json:"addresses,omitempty"`
	// Supporting documentation for the goal
	Documentation []RelatedArtifact `json:"documentation,omitempty"`
	// Target outcome for the goal
	Target []PlanDefinitionGoalTarget `json:"target,omitempty"`
}

// PlanDefinitionActorOption represents a FHIR BackboneElement for PlanDefinition.actor.option.
type PlanDefinitionActorOption struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// careteam | device | group | healthcareservice | location | organization | patient | practitioner | practitionerrole | relatedperson
	Type *string `json:"type,omitempty"`
	// Who or what can participate
	TypeCanonical *string `json:"typeCanonical,omitempty"`
	// Who or what can participate
	TypeReference *Reference `json:"typeReference,omitempty"`
	// E.g. Nurse, Surgeon, Parent
	Role *CodeableConcept `json:"role,omitempty"`
}

// PlanDefinitionActor represents a FHIR BackboneElement for PlanDefinition.actor.
type PlanDefinitionActor struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// User-visible title
	Title *string `json:"title,omitempty"`
	// Describes the actor
	Description *string `json:"description,omitempty"`
	// Who or what can be this actor
	Option []PlanDefinitionActorOption `json:"option,omitempty"`
}

// PlanDefinitionActionCondition represents a FHIR BackboneElement for PlanDefinition.action.condition.
type PlanDefinitionActionCondition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// applicability | start | stop
	Kind string `json:"kind"`
	// Boolean-valued expression
	Expression *Expression `json:"expression,omitempty"`
}

// PlanDefinitionActionInput represents a FHIR BackboneElement for PlanDefinition.action.input.
type PlanDefinitionActionInput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// User-visible title
	Title *string `json:"title,omitempty"`
	// What data is provided
	Requirement *DataRequirement `json:"requirement,omitempty"`
	// What data is provided
	RelatedData *string `json:"relatedData,omitempty"`
}

// PlanDefinitionActionOutput represents a FHIR BackboneElement for PlanDefinition.action.output.
type PlanDefinitionActionOutput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// User-visible title
	Title *string `json:"title,omitempty"`
	// What data is provided
	Requirement *DataRequirement `json:"requirement,omitempty"`
	// What data is provided
	RelatedData *string `json:"relatedData,omitempty"`
}

// PlanDefinitionActionRelatedAction represents a FHIR BackboneElement for PlanDefinition.action.relatedAction.
type PlanDefinitionActionRelatedAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What action is this related to
	TargetId string `json:"targetId"`
	// before | before-start | before-end | concurrent | concurrent-with-start | concurrent-with-end | after | after-start | after-end
	Relationship string `json:"relationship"`
	// before | before-start | before-end | concurrent | concurrent-with-start | concurrent-with-end | after | after-start | after-end
	EndRelationship *string `json:"endRelationship,omitempty"`
	// Time offset for the relationship
	Offset *any `json:"offset,omitempty"`
}

// PlanDefinitionActionParticipant represents a FHIR BackboneElement for PlanDefinition.action.participant.
type PlanDefinitionActionParticipant struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What actor
	ActorId *string `json:"actorId,omitempty"`
	// careteam | device | group | healthcareservice | location | organization | patient | practitioner | practitionerrole | relatedperson
	Type *string `json:"type,omitempty"`
	// Who or what can participate
	TypeCanonical *string `json:"typeCanonical,omitempty"`
	// Who or what can participate
	TypeReference *Reference `json:"typeReference,omitempty"`
	// E.g. Nurse, Surgeon, Parent
	Role *CodeableConcept `json:"role,omitempty"`
	// E.g. Author, Reviewer, Witness, etc
	Function *CodeableConcept `json:"function,omitempty"`
}

// PlanDefinitionActionDynamicValue represents a FHIR BackboneElement for PlanDefinition.action.dynamicValue.
type PlanDefinitionActionDynamicValue struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The path to the element to be set dynamically
	Path *string `json:"path,omitempty"`
	// An expression that provides the dynamic value for the customization
	Expression *Expression `json:"expression,omitempty"`
}

// PlanDefinitionActionAction represents a FHIR BackboneElement for PlanDefinition.action.action.
type PlanDefinitionActionAction struct {
}

// PlanDefinitionAction represents a FHIR BackboneElement for PlanDefinition.action.
type PlanDefinitionAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Unique id for the action in the PlanDefinition
	LinkId *string `json:"linkId,omitempty"`
	// User-visible prefix for the action (e.g. 1. or A.)
	Prefix *string `json:"prefix,omitempty"`
	// User-visible title
	Title *string `json:"title,omitempty"`
	// Brief description of the action
	Description *string `json:"description,omitempty"`
	// Static text equivalent of the action, used if the dynamic aspects cannot be interpreted by the receiving system
	TextEquivalent *string `json:"textEquivalent,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// Code representing the meaning of the action or sub-actions
	Code *CodeableConcept `json:"code,omitempty"`
	// Why the action should be performed
	Reason []CodeableConcept `json:"reason,omitempty"`
	// Supporting documentation for the intended performer of the action
	Documentation []RelatedArtifact `json:"documentation,omitempty"`
	// What goals this action supports
	GoalId []string `json:"goalId,omitempty"`
	// Type of individual the action is focused on
	Subject *any `json:"subject,omitempty"`
	// When the action should be triggered
	Trigger []TriggerDefinition `json:"trigger,omitempty"`
	// Whether or not the action is applicable
	Condition []PlanDefinitionActionCondition `json:"condition,omitempty"`
	// Input data requirements
	Input []PlanDefinitionActionInput `json:"input,omitempty"`
	// Output data definition
	Output []PlanDefinitionActionOutput `json:"output,omitempty"`
	// Relationship to another action
	RelatedAction []PlanDefinitionActionRelatedAction `json:"relatedAction,omitempty"`
	// When the action should take place
	Timing *any `json:"timing,omitempty"`
	// Where it should happen
	Location *CodeableReference `json:"location,omitempty"`
	// Who should participate in the action
	Participant []PlanDefinitionActionParticipant `json:"participant,omitempty"`
	// create | update | remove | fire-event
	Type *CodeableConcept `json:"type,omitempty"`
	// visual-group | logical-group | sentence-group
	GroupingBehavior *string `json:"groupingBehavior,omitempty"`
	// any | all | all-or-none | exactly-one | at-most-one | one-or-more
	SelectionBehavior *string `json:"selectionBehavior,omitempty"`
	// must | could | must-unless-documented
	RequiredBehavior *string `json:"requiredBehavior,omitempty"`
	// yes | no
	PrecheckBehavior *string `json:"precheckBehavior,omitempty"`
	// single | multiple
	CardinalityBehavior *string `json:"cardinalityBehavior,omitempty"`
	// Description of the activity to be performed
	Definition *any `json:"definition,omitempty"`
	// Transform to apply the template
	Transform *string `json:"transform,omitempty"`
	// Dynamic aspects of the definition
	DynamicValue []PlanDefinitionActionDynamicValue `json:"dynamicValue,omitempty"`
	// A sub-action
	Action []PlanDefinitionActionAction `json:"action,omitempty"`
}

// PlanDefinition represents a FHIR PlanDefinition.
type PlanDefinition struct {
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
	// Canonical identifier for this plan definition, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the plan definition
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the plan definition
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this plan definition (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this plan definition (human friendly)
	Title *string `json:"title,omitempty"`
	// Subordinate title of the plan definition
	Subtitle *string `json:"subtitle,omitempty"`
	// order-set | clinical-protocol | eca-rule | workflow-definition
	Type *CodeableConcept `json:"type,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Type of individual the plan definition is focused on
	Subject *any `json:"subject,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the plan definition
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for plan definition (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this plan definition is defined
	Purpose *string `json:"purpose,omitempty"`
	// Describes the clinical usage of the plan
	Usage *string `json:"usage,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the plan definition was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the plan definition was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the plan definition is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// E.g. Education, Treatment, Assessment
	Topic []CodeableConcept `json:"topic,omitempty"`
	// Who authored the content
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the content
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the content
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the content
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Logic used by the plan definition
	Library []string `json:"library,omitempty"`
	// What the plan is trying to accomplish
	Goal []PlanDefinitionGoal `json:"goal,omitempty"`
	// Actors within the plan
	Actor []PlanDefinitionActor `json:"actor,omitempty"`
	// Action defined by the plan
	Action []PlanDefinitionAction `json:"action,omitempty"`
	// Preconditions for service
	AsNeeded *any `json:"asNeeded,omitempty"`
}
