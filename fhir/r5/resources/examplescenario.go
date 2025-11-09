package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeExampleScenario is the FHIR resource type name for ExampleScenario.
const ResourceTypeExampleScenario = "ExampleScenario"

// ExampleScenarioActor represents a FHIR BackboneElement for ExampleScenario.actor.
type ExampleScenarioActor struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// ID or acronym of the actor
	Key string `json:"key"`
	// person | system
	Type string `json:"type"`
	// Label for actor when rendering
	Title string `json:"title"`
	// Details about actor
	Description *string `json:"description,omitempty"`
}

// ExampleScenarioInstanceVersion represents a FHIR BackboneElement for ExampleScenario.instance.version.
type ExampleScenarioInstanceVersion struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// ID or acronym of the version
	Key string `json:"key"`
	// Label for instance version
	Title string `json:"title"`
	// Details about version
	Description *string `json:"description,omitempty"`
	// Example instance version data
	Content *Reference `json:"content,omitempty"`
}

// ExampleScenarioInstanceContainedInstance represents a FHIR BackboneElement for ExampleScenario.instance.containedInstance.
type ExampleScenarioInstanceContainedInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Key of contained instance
	InstanceReference string `json:"instanceReference"`
	// Key of contained instance version
	VersionReference *string `json:"versionReference,omitempty"`
}

// ExampleScenarioInstance represents a FHIR BackboneElement for ExampleScenario.instance.
type ExampleScenarioInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// ID or acronym of the instance
	Key string `json:"key"`
	// Data structure for example
	StructureType Coding `json:"structureType"`
	// E.g. 4.0.1
	StructureVersion *string `json:"structureVersion,omitempty"`
	// Rules instance adheres to
	StructureProfile *any `json:"structureProfile,omitempty"`
	// Label for instance
	Title string `json:"title"`
	// Human-friendly description of the instance
	Description *string `json:"description,omitempty"`
	// Example instance data
	Content *Reference `json:"content,omitempty"`
	// Snapshot of instance that changes
	Version []ExampleScenarioInstanceVersion `json:"version,omitempty"`
	// Resources contained in the instance
	ContainedInstance []ExampleScenarioInstanceContainedInstance `json:"containedInstance,omitempty"`
}

// ExampleScenarioProcessStepProcess represents a FHIR BackboneElement for ExampleScenario.process.step.process.
type ExampleScenarioProcessStepProcess struct {
}

// ExampleScenarioProcessStepOperationRequest represents a FHIR BackboneElement for ExampleScenario.process.step.operation.request.
type ExampleScenarioProcessStepOperationRequest struct {
}

// ExampleScenarioProcessStepOperationResponse represents a FHIR BackboneElement for ExampleScenario.process.step.operation.response.
type ExampleScenarioProcessStepOperationResponse struct {
}

// ExampleScenarioProcessStepOperation represents a FHIR BackboneElement for ExampleScenario.process.step.operation.
type ExampleScenarioProcessStepOperation struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Kind of action
	Type *Coding `json:"type,omitempty"`
	// Label for step
	Title string `json:"title"`
	// Who starts the operation
	Initiator *string `json:"initiator,omitempty"`
	// Who receives the operation
	Receiver *string `json:"receiver,omitempty"`
	// Human-friendly description of the operation
	Description *string `json:"description,omitempty"`
	// Initiator stays active?
	InitiatorActive *bool `json:"initiatorActive,omitempty"`
	// Receiver stays active?
	ReceiverActive *bool `json:"receiverActive,omitempty"`
	// Instance transmitted on invocation
	Request *ExampleScenarioProcessStepOperationRequest `json:"request,omitempty"`
	// Instance transmitted on invocation response
	Response *ExampleScenarioProcessStepOperationResponse `json:"response,omitempty"`
}

// ExampleScenarioProcessStepAlternativeStep represents a FHIR BackboneElement for ExampleScenario.process.step.alternative.step.
type ExampleScenarioProcessStepAlternativeStep struct {
}

// ExampleScenarioProcessStepAlternative represents a FHIR BackboneElement for ExampleScenario.process.step.alternative.
type ExampleScenarioProcessStepAlternative struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for alternative
	Title string `json:"title"`
	// Human-readable description of option
	Description *string `json:"description,omitempty"`
	// Alternative action(s)
	Step []ExampleScenarioProcessStepAlternativeStep `json:"step,omitempty"`
}

// ExampleScenarioProcessStep represents a FHIR BackboneElement for ExampleScenario.process.step.
type ExampleScenarioProcessStep struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Sequential number of the step
	Number *string `json:"number,omitempty"`
	// Step is nested process
	Process *ExampleScenarioProcessStepProcess `json:"process,omitempty"`
	// Step is nested workflow
	Workflow *string `json:"workflow,omitempty"`
	// Step is simple action
	Operation *ExampleScenarioProcessStepOperation `json:"operation,omitempty"`
	// Alternate non-typical step action
	Alternative []ExampleScenarioProcessStepAlternative `json:"alternative,omitempty"`
	// Pause in the flow?
	Pause *bool `json:"pause,omitempty"`
}

// ExampleScenarioProcess represents a FHIR BackboneElement for ExampleScenario.process.
type ExampleScenarioProcess struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for procss
	Title string `json:"title"`
	// Human-friendly description of the process
	Description *string `json:"description,omitempty"`
	// Status before process starts
	PreConditions *string `json:"preConditions,omitempty"`
	// Status after successful completion
	PostConditions *string `json:"postConditions,omitempty"`
	// Event within of the process
	Step []ExampleScenarioProcessStep `json:"step,omitempty"`
}

// ExampleScenario represents a FHIR ExampleScenario.
type ExampleScenario struct {
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
	// Canonical identifier for this example scenario, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the example scenario
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the example scenario
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// To be removed?
	Name *string `json:"name,omitempty"`
	// Name for this example scenario (human friendly)
	Title *string `json:"title,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher/steward (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the ExampleScenario
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for example scenario (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// The purpose of the example, e.g. to illustrate a scenario
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// Individual involved in exchange
	Actor []ExampleScenarioActor `json:"actor,omitempty"`
	// Data used in the scenario
	Instance []ExampleScenarioInstance `json:"instance,omitempty"`
	// Major process within scenario
	Process []ExampleScenarioProcess `json:"process,omitempty"`
}
