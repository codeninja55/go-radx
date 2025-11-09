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
	ActorId string `json:"actorId"`
	// person | entity
	Type string `json:"type"`
	// The name of the actor as shown in the page
	Name *string `json:"name,omitempty"`
	// The description of the actor
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
	// The identifier of a specific version of a resource
	VersionId string `json:"versionId"`
	// The description of the resource version
	Description string `json:"description"`
}

// ExampleScenarioInstanceContainedInstance represents a FHIR BackboneElement for ExampleScenario.instance.containedInstance.
type ExampleScenarioInstanceContainedInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Each resource contained in the instance
	ResourceId string `json:"resourceId"`
	// A specific version of a resource contained in the instance
	VersionId *string `json:"versionId,omitempty"`
}

// ExampleScenarioInstance represents a FHIR BackboneElement for ExampleScenario.instance.
type ExampleScenarioInstance struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The id of the resource for referencing
	ResourceId string `json:"resourceId"`
	// The type of the resource
	ResourceType string `json:"resourceType"`
	// A short name for the resource instance
	Name *string `json:"name,omitempty"`
	// Human-friendly description of the resource instance
	Description *string `json:"description,omitempty"`
	// A specific version of the resource
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
	// The sequential number of the interaction
	Number string `json:"number"`
	// The type of operation - CRUD
	Type *string `json:"type,omitempty"`
	// The human-friendly name of the interaction
	Name *string `json:"name,omitempty"`
	// Who starts the transaction
	Initiator *string `json:"initiator,omitempty"`
	// Who receives the transaction
	Receiver *string `json:"receiver,omitempty"`
	// A comment to be inserted in the diagram
	Description *string `json:"description,omitempty"`
	// Whether the initiator is deactivated right after the transaction
	InitiatorActive *bool `json:"initiatorActive,omitempty"`
	// Whether the receiver is deactivated right after the transaction
	ReceiverActive *bool `json:"receiverActive,omitempty"`
	// Each resource instance used by the initiator
	Request *ExampleScenarioProcessStepOperationRequest `json:"request,omitempty"`
	// Each resource instance used by the responder
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
	// A human-readable description of each option
	Description *string `json:"description,omitempty"`
	// What happens in each alternative option
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
	// Nested process
	Process []ExampleScenarioProcessStepProcess `json:"process,omitempty"`
	// If there is a pause in the flow
	Pause *bool `json:"pause,omitempty"`
	// Each interaction or action
	Operation *ExampleScenarioProcessStepOperation `json:"operation,omitempty"`
	// Alternate non-typical step action
	Alternative []ExampleScenarioProcessStepAlternative `json:"alternative,omitempty"`
}

// ExampleScenarioProcess represents a FHIR BackboneElement for ExampleScenario.process.
type ExampleScenarioProcess struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// The diagram title of the group of operations
	Title string `json:"title"`
	// A longer description of the group of operations
	Description *string `json:"description,omitempty"`
	// Description of initial status before the process starts
	PreConditions *string `json:"preConditions,omitempty"`
	// Description of final status after the process ends
	PostConditions *string `json:"postConditions,omitempty"`
	// Each step of the process
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
	// Name for this example scenario (computer friendly)
	Name *string `json:"name,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for example scenario (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// The purpose of the example, e.g. to illustrate a scenario
	Purpose *string `json:"purpose,omitempty"`
	// Actor participating in the resource
	Actor []ExampleScenarioActor `json:"actor,omitempty"`
	// Each resource and each version that is present in the workflow
	Instance []ExampleScenarioInstance `json:"instance,omitempty"`
	// Each major process - a group of operations
	Process []ExampleScenarioProcess `json:"process,omitempty"`
	// Another nested workflow
	Workflow []string `json:"workflow,omitempty"`
}
