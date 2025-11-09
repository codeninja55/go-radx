package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeQuestionnaire is the FHIR resource type name for Questionnaire.
const ResourceTypeQuestionnaire = "Questionnaire"

// QuestionnaireItemEnableWhen represents a FHIR BackboneElement for Questionnaire.item.enableWhen.
type QuestionnaireItemEnableWhen struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Question that determines whether item is enabled
	Question string `json:"question"`
	// exists | = | != | > | < | >= | <=
	Operator string `json:"operator"`
	// Value for question comparison based on operator
	Answer any `json:"answer"`
}

// QuestionnaireItemAnswerOption represents a FHIR BackboneElement for Questionnaire.item.answerOption.
type QuestionnaireItemAnswerOption struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Answer value
	Value any `json:"value"`
	// Whether option is selected by default
	InitialSelected *bool `json:"initialSelected,omitempty"`
}

// QuestionnaireItemInitial represents a FHIR BackboneElement for Questionnaire.item.initial.
type QuestionnaireItemInitial struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Actual value for initializing the question
	Value any `json:"value"`
}

// QuestionnaireItemItem represents a FHIR BackboneElement for Questionnaire.item.item.
type QuestionnaireItemItem struct {
}

// QuestionnaireItem represents a FHIR BackboneElement for Questionnaire.item.
type QuestionnaireItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Unique id for item in questionnaire
	LinkId string `json:"linkId"`
	// ElementDefinition - details for the item
	Definition *string `json:"definition,omitempty"`
	// Corresponding concept for this item in a terminology
	Code []Coding `json:"code,omitempty"`
	// E.g. "1(a)", "2.5.3"
	Prefix *string `json:"prefix,omitempty"`
	// Primary text for the item
	Text *string `json:"text,omitempty"`
	// group | display | boolean | decimal | integer | date | dateTime +
	Type string `json:"type"`
	// Only allow data when
	EnableWhen []QuestionnaireItemEnableWhen `json:"enableWhen,omitempty"`
	// all | any
	EnableBehavior *string `json:"enableBehavior,omitempty"`
	// Whether the item must be included in data results
	Required *bool `json:"required,omitempty"`
	// Whether the item may repeat
	Repeats *bool `json:"repeats,omitempty"`
	// Don't allow human editing
	ReadOnly *bool `json:"readOnly,omitempty"`
	// No more than this many characters
	MaxLength *int `json:"maxLength,omitempty"`
	// Valueset containing permitted answers
	AnswerValueSet *string `json:"answerValueSet,omitempty"`
	// Permitted answer
	AnswerOption []QuestionnaireItemAnswerOption `json:"answerOption,omitempty"`
	// Initial value(s) when item is first rendered
	Initial []QuestionnaireItemInitial `json:"initial,omitempty"`
	// Nested questionnaire items
	Item []QuestionnaireItemItem `json:"item,omitempty"`
}

// Questionnaire represents a FHIR Questionnaire.
type Questionnaire struct {
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
	// Canonical identifier for this questionnaire, represented as a URI (globally unique)
	URL *string `json:"url,omitempty"`
	// Additional identifier for the questionnaire
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the questionnaire
	Version *string `json:"version,omitempty"`
	// Name for this questionnaire (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this questionnaire (human friendly)
	Title *string `json:"title,omitempty"`
	// Instantiates protocol or definition
	DerivedFrom []string `json:"derivedFrom,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// For testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Resource that can be subject of QuestionnaireResponse
	SubjectType []string `json:"subjectType,omitempty"`
	// Date last changed
	Date *primitives.DateTime `json:"date,omitempty"`
	// Name of the publisher (organization or individual)
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the questionnaire
	Description *string `json:"description,omitempty"`
	// The context that the content is intended to support
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction for questionnaire (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this questionnaire is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// When the questionnaire was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// When the questionnaire was last reviewed
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// When the questionnaire is expected to be used
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// Concept that represents the overall questionnaire
	Code []Coding `json:"code,omitempty"`
	// Questions and sections within the Questionnaire
	Item []QuestionnaireItem `json:"item,omitempty"`
}
