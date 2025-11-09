package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeRequestOrchestration is the FHIR resource type name for RequestOrchestration.
const ResourceTypeRequestOrchestration = "RequestOrchestration"

// RequestOrchestrationActionCondition represents a FHIR BackboneElement for RequestOrchestration.action.condition.
type RequestOrchestrationActionCondition struct {
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

// RequestOrchestrationActionInput represents a FHIR BackboneElement for RequestOrchestration.action.input.
type RequestOrchestrationActionInput struct {
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

// RequestOrchestrationActionOutput represents a FHIR BackboneElement for RequestOrchestration.action.output.
type RequestOrchestrationActionOutput struct {
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

// RequestOrchestrationActionRelatedAction represents a FHIR BackboneElement for RequestOrchestration.action.relatedAction.
type RequestOrchestrationActionRelatedAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What action this is related to
	TargetId string `json:"targetId"`
	// before | before-start | before-end | concurrent | concurrent-with-start | concurrent-with-end | after | after-start | after-end
	Relationship string `json:"relationship"`
	// before | before-start | before-end | concurrent | concurrent-with-start | concurrent-with-end | after | after-start | after-end
	EndRelationship *string `json:"endRelationship,omitempty"`
	// Time offset for the relationship
	Offset *any `json:"offset,omitempty"`
}

// RequestOrchestrationActionParticipant represents a FHIR BackboneElement for RequestOrchestration.action.participant.
type RequestOrchestrationActionParticipant struct {
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
	// E.g. Nurse, Surgeon, Parent, etc
	Role *CodeableConcept `json:"role,omitempty"`
	// E.g. Author, Reviewer, Witness, etc
	Function *CodeableConcept `json:"function,omitempty"`
	// Who/what is participating?
	Actor *any `json:"actor,omitempty"`
}

// RequestOrchestrationActionDynamicValue represents a FHIR BackboneElement for RequestOrchestration.action.dynamicValue.
type RequestOrchestrationActionDynamicValue struct {
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

// RequestOrchestrationActionAction represents a FHIR BackboneElement for RequestOrchestration.action.action.
type RequestOrchestrationActionAction struct {
}

// RequestOrchestrationAction represents a FHIR BackboneElement for RequestOrchestration.action.
type RequestOrchestrationAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Pointer to specific item from the PlanDefinition
	LinkId *string `json:"linkId,omitempty"`
	// User-visible prefix for the action (e.g. 1. or A.)
	Prefix *string `json:"prefix,omitempty"`
	// User-visible title
	Title *string `json:"title,omitempty"`
	// Short description of the action
	Description *string `json:"description,omitempty"`
	// Static text equivalent of the action, used if the dynamic aspects cannot be interpreted by the receiving system
	TextEquivalent *string `json:"textEquivalent,omitempty"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// Code representing the meaning of the action or sub-actions
	Code []CodeableConcept `json:"code,omitempty"`
	// Supporting documentation for the intended performer of the action
	Documentation []RelatedArtifact `json:"documentation,omitempty"`
	// What goals
	Goal []Reference `json:"goal,omitempty"`
	// Whether or not the action is applicable
	Condition []RequestOrchestrationActionCondition `json:"condition,omitempty"`
	// Input data requirements
	Input []RequestOrchestrationActionInput `json:"input,omitempty"`
	// Output data definition
	Output []RequestOrchestrationActionOutput `json:"output,omitempty"`
	// Relationship to another action
	RelatedAction []RequestOrchestrationActionRelatedAction `json:"relatedAction,omitempty"`
	// When the action should take place
	Timing *any `json:"timing,omitempty"`
	// Where it should happen
	Location *CodeableReference `json:"location,omitempty"`
	// Who should perform the action
	Participant []RequestOrchestrationActionParticipant `json:"participant,omitempty"`
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
	// The target of the action
	Resource *Reference `json:"resource,omitempty"`
	// Description of the activity to be performed
	Definition *any `json:"definition,omitempty"`
	// Transform to apply the template
	Transform *string `json:"transform,omitempty"`
	// Dynamic aspects of the definition
	DynamicValue []RequestOrchestrationActionDynamicValue `json:"dynamicValue,omitempty"`
	// Sub action
	Action []RequestOrchestrationActionAction `json:"action,omitempty"`
}

// RequestOrchestration represents a FHIR RequestOrchestration.
type RequestOrchestration struct {
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
	// Business identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Instantiates FHIR protocol or definition
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	// Instantiates external protocol or definition
	InstantiatesUri []string `json:"instantiatesUri,omitempty"`
	// Fulfills plan, proposal, or order
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Request(s) replaced by this request
	Replaces []Reference `json:"replaces,omitempty"`
	// Composite request this is part of
	GroupIdentifier *Identifier `json:"groupIdentifier,omitempty"`
	// draft | active | on-hold | revoked | completed | entered-in-error | unknown
	Status string `json:"status"`
	// proposal | plan | directive | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// What's being requested/ordered
	Code *CodeableConcept `json:"code,omitempty"`
	// Who the request orchestration is about
	Subject *Reference `json:"subject,omitempty"`
	// Created as part of
	Encounter *Reference `json:"encounter,omitempty"`
	// When the request orchestration was authored
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Device or practitioner that authored the request orchestration
	Author *Reference `json:"author,omitempty"`
	// Why the request orchestration is needed
	Reason []CodeableReference `json:"reason,omitempty"`
	// What goals
	Goal []Reference `json:"goal,omitempty"`
	// Additional notes about the response
	Note []Annotation `json:"note,omitempty"`
	// Proposed actions, if any
	Action []RequestOrchestrationAction `json:"action,omitempty"`
}
