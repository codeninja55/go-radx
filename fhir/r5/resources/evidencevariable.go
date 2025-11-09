package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeEvidenceVariable is the FHIR resource type name for EvidenceVariable.
const ResourceTypeEvidenceVariable = "EvidenceVariable"

// EvidenceVariableCharacteristicDefinitionByTypeAndValue represents a FHIR BackboneElement for EvidenceVariable.characteristic.definitionByTypeAndValue.
type EvidenceVariableCharacteristicDefinitionByTypeAndValue struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Expresses the type of characteristic
	Type CodeableConcept `json:"type"`
	// Method for how the characteristic value was determined
	Method []CodeableConcept `json:"method,omitempty"`
	// Device used for determining characteristic
	Device *Reference `json:"device,omitempty"`
	// Defines the characteristic when coupled with characteristic.type
	Value any `json:"value"`
	// Reference point for valueQuantity or valueRange
	Offset *CodeableConcept `json:"offset,omitempty"`
}

// EvidenceVariableCharacteristicDefinitionByCombinationCharacteristic represents a FHIR BackboneElement for EvidenceVariable.characteristic.definitionByCombination.characteristic.
type EvidenceVariableCharacteristicDefinitionByCombinationCharacteristic struct {
}

// EvidenceVariableCharacteristicDefinitionByCombination represents a FHIR BackboneElement for EvidenceVariable.characteristic.definitionByCombination.
type EvidenceVariableCharacteristicDefinitionByCombination struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// all-of | any-of | at-least | at-most | statistical | net-effect | dataset
	Code string `json:"code"`
	// Provides the value of "n" when "at-least" or "at-most" codes are used
	Threshold *int `json:"threshold,omitempty"`
	// A defining factor of the characteristic
	Characteristic []EvidenceVariableCharacteristicDefinitionByCombinationCharacteristic `json:"characteristic,omitempty"`
}

// EvidenceVariableCharacteristicTimeFromEvent represents a FHIR BackboneElement for EvidenceVariable.characteristic.timeFromEvent.
type EvidenceVariableCharacteristicTimeFromEvent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Human readable description
	Description *string `json:"description,omitempty"`
	// Used for footnotes or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// The event used as a base point (reference point) in time
	Event *any `json:"event,omitempty"`
	// Used to express the observation at a defined amount of time before or after the event
	Quantity *Quantity `json:"quantity,omitempty"`
	// Used to express the observation within a period before and/or after the event
	Range *Range `json:"range,omitempty"`
}

// EvidenceVariableCharacteristic represents a FHIR BackboneElement for EvidenceVariable.characteristic.
type EvidenceVariableCharacteristic struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Label for internal linking
	LinkId *string `json:"linkId,omitempty"`
	// Natural language description of the characteristic
	Description *string `json:"description,omitempty"`
	// Used for footnotes or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// Whether the characteristic is an inclusion criterion or exclusion criterion
	Exclude *bool `json:"exclude,omitempty"`
	// Defines the characteristic (without using type and value) by a Reference
	DefinitionReference *Reference `json:"definitionReference,omitempty"`
	// Defines the characteristic (without using type and value) by a Canonical
	DefinitionCanonical *string `json:"definitionCanonical,omitempty"`
	// Defines the characteristic (without using type and value) by a CodeableConcept
	DefinitionCodeableConcept *CodeableConcept `json:"definitionCodeableConcept,omitempty"`
	// Defines the characteristic (without using type and value) by an expression
	DefinitionExpression *Expression `json:"definitionExpression,omitempty"`
	// Defines the characteristic (without using type and value) by an id
	DefinitionId *string `json:"definitionId,omitempty"`
	// Defines the characteristic using type and value
	DefinitionByTypeAndValue *EvidenceVariableCharacteristicDefinitionByTypeAndValue `json:"definitionByTypeAndValue,omitempty"`
	// Used to specify how two or more characteristics are combined
	DefinitionByCombination *EvidenceVariableCharacteristicDefinitionByCombination `json:"definitionByCombination,omitempty"`
	// Number of occurrences meeting the characteristic
	Instances *any `json:"instances,omitempty"`
	// Length of time in which the characteristic is met
	Duration *any `json:"duration,omitempty"`
	// Timing in which the characteristic is determined
	TimeFromEvent []EvidenceVariableCharacteristicTimeFromEvent `json:"timeFromEvent,omitempty"`
}

// EvidenceVariableCategory represents a FHIR BackboneElement for EvidenceVariable.category.
type EvidenceVariableCategory struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of the grouping
	Name *string `json:"name,omitempty"`
	// Definition of the grouping
	Value *any `json:"value,omitempty"`
}

// EvidenceVariable represents a FHIR EvidenceVariable.
type EvidenceVariable struct {
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
	// Canonical identifier for this evidence variable, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the evidence variable
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the evidence variable
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this evidence variable (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this evidence variable (human friendly)
	Title *string `json:"title,omitempty"`
	// Title for use in informal contexts
	ShortTitle *string `json:"shortTitle,omitempty"`
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
	// Natural language description of the evidence variable
	Description *string `json:"description,omitempty"`
	// Used for footnotes or explanatory notes
	Note []Annotation `json:"note,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Why this EvidenceVariable is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When the resource was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the resource was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the resource is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// Who authored the content
	Author []ContactDetail `json:"author,omitempty"`
	// Who edited the content
	Editor []ContactDetail `json:"editor,omitempty"`
	// Who reviewed the content
	Reviewer []ContactDetail `json:"reviewer,omitempty"`
	// Who endorsed the content
	Endorser []ContactDetail `json:"endorser,omitempty"`
	// Additional documentation, citations, etc
	RelatedArtifact []RelatedArtifact `json:"relatedArtifact,omitempty"`
	// Actual or conceptual
	Actual *bool `json:"actual,omitempty"`
	// A defining factor of the EvidenceVariable
	Characteristic []EvidenceVariableCharacteristic `json:"characteristic,omitempty"`
	// continuous | dichotomous | ordinal | polychotomous
	Handling *string `json:"handling,omitempty"`
	// A grouping for ordinal or polychotomous variables
	Category []EvidenceVariableCategory `json:"category,omitempty"`
}
