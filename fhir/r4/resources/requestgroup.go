package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeRequestGroup is the FHIR resource type name for RequestGroup.
const ResourceTypeRequestGroup = "RequestGroup"

// RequestGroupActionCondition represents a FHIR BackboneElement for RequestGroup.action.condition.
type RequestGroupActionCondition struct {
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

// RequestGroupActionRelatedAction represents a FHIR BackboneElement for RequestGroup.action.relatedAction.
type RequestGroupActionRelatedAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// What action this is related to
	ActionId string `json:"actionId"`
	// before-start | before | before-end | concurrent-with-start | concurrent | concurrent-with-end | after-start | after | after-end
	Relationship string `json:"relationship"`
	// Time offset for the relationship
	Offset *any `json:"offset,omitempty"`
}

// RequestGroupActionAction represents a FHIR BackboneElement for RequestGroup.action.action.
type RequestGroupActionAction struct {
}

// RequestGroupAction represents a FHIR BackboneElement for RequestGroup.action.
type RequestGroupAction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
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
	// Whether or not the action is applicable
	Condition []RequestGroupActionCondition `json:"condition,omitempty"`
	// Relationship to another action
	RelatedAction []RequestGroupActionRelatedAction `json:"relatedAction,omitempty"`
	// When the action should take place
	Timing *any `json:"timing,omitempty"`
	// Who should perform the action
	Participant []Reference `json:"participant,omitempty"`
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
	// Sub action
	Action []RequestGroupActionAction `json:"action,omitempty"`
}

// RequestGroup represents a FHIR RequestGroup.
type RequestGroup struct {
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
	// Who the request group is about
	Subject *Reference `json:"subject,omitempty"`
	// Created as part of
	Encounter *Reference `json:"encounter,omitempty"`
	// When the request group was authored
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Device or practitioner that authored the request group
	Author *Reference `json:"author,omitempty"`
	// Why the request group is needed
	ReasonCode []CodeableConcept `json:"reasonCode,omitempty"`
	// Why the request group is needed
	ReasonReference []Reference `json:"reasonReference,omitempty"`
	// Additional notes about the response
	Note []Annotation `json:"note,omitempty"`
	// Proposed actions, if any
	Action []RequestGroupAction `json:"action,omitempty"`
}
