package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeTask is the FHIR resource type name for Task.
const ResourceTypeTask = "Task"

// TaskRestriction represents a FHIR BackboneElement for Task.restriction.
type TaskRestriction struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// How many times to repeat
	Repetitions *int `json:"repetitions,omitempty"`
	// When fulfillment sought
	Period *Period `json:"period,omitempty"`
	// For whom is fulfillment sought?
	Recipient []Reference `json:"recipient,omitempty"`
}

// TaskInput represents a FHIR BackboneElement for Task.input.
type TaskInput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for the input
	Type CodeableConcept `json:"type"`
	// Content to use in performing the task
	Value any `json:"value"`
}

// TaskOutput represents a FHIR BackboneElement for Task.output.
type TaskOutput struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for output
	Type CodeableConcept `json:"type"`
	// Result of output
	Value any `json:"value"`
}

// Task represents a FHIR Task.
type Task struct {
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
	// Task Instance Identifier
	Identifier []Identifier `json:"identifier,omitempty"`
	// Formal definition of task
	InstantiatesCanonical *string `json:"instantiatesCanonical,omitempty"`
	// Formal definition of task
	InstantiatesUri *string `json:"instantiatesUri,omitempty"`
	// Request fulfilled by this task
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Requisition or grouper id
	GroupIdentifier *Identifier `json:"groupIdentifier,omitempty"`
	// Composite task
	PartOf []Reference `json:"partOf,omitempty"`
	// draft | requested | received | accepted | +
	Status string `json:"status"`
	// Reason for current status
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`
	// E.g. "Specimen collected", "IV prepped"
	BusinessStatus *CodeableConcept `json:"businessStatus,omitempty"`
	// unknown | proposal | plan | order | original-order | reflex-order | filler-order | instance-order | option
	Intent string `json:"intent"`
	// routine | urgent | asap | stat
	Priority *string `json:"priority,omitempty"`
	// Task Type
	Code *CodeableConcept `json:"code,omitempty"`
	// Human-readable explanation of task
	Description *string `json:"description,omitempty"`
	// What task is acting on
	Focus *Reference `json:"focus,omitempty"`
	// Beneficiary of the Task
	For *Reference `json:"for,omitempty"`
	// Healthcare event during which this task originated
	Encounter *Reference `json:"encounter,omitempty"`
	// Start and end time of execution
	ExecutionPeriod *Period `json:"executionPeriod,omitempty"`
	// Task Creation Date
	AuthoredOn *primitives.DateTime `json:"authoredOn,omitempty"`
	// Task Last Modified Date
	LastModified *primitives.DateTime `json:"lastModified,omitempty"`
	// Who is asking for task to be done
	Requester *Reference `json:"requester,omitempty"`
	// Requested performer
	PerformerType []CodeableConcept `json:"performerType,omitempty"`
	// Responsible individual
	Owner *Reference `json:"owner,omitempty"`
	// Where task occurs
	Location *Reference `json:"location,omitempty"`
	// Why task is needed
	ReasonCode *CodeableConcept `json:"reasonCode,omitempty"`
	// Why task is needed
	ReasonReference *Reference `json:"reasonReference,omitempty"`
	// Associated insurance coverage
	Insurance []Reference `json:"insurance,omitempty"`
	// Comments made about the task
	Note []Annotation `json:"note,omitempty"`
	// Key events in history of the Task
	RelevantHistory []Reference `json:"relevantHistory,omitempty"`
	// Constraints on fulfillment tasks
	Restriction *TaskRestriction `json:"restriction,omitempty"`
	// Information used to perform task
	Input []TaskInput `json:"input,omitempty"`
	// Information produced as part of task
	Output []TaskOutput `json:"output,omitempty"`
}
