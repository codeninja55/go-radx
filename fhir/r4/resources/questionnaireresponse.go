package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeQuestionnaireResponse is the FHIR resource type name for QuestionnaireResponse.
const ResourceTypeQuestionnaireResponse = "QuestionnaireResponse"

// QuestionnaireResponseItemAnswerItem represents a FHIR BackboneElement for QuestionnaireResponse.item.answer.item.
type QuestionnaireResponseItemAnswerItem struct {
}

// QuestionnaireResponseItemAnswer represents a FHIR BackboneElement for QuestionnaireResponse.item.answer.
type QuestionnaireResponseItemAnswer struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Single-valued answer to the question
	Value *any `json:"value,omitempty"`
	// Nested groups and questions
	Item []QuestionnaireResponseItemAnswerItem `json:"item,omitempty"`
}

// QuestionnaireResponseItemItem represents a FHIR BackboneElement for QuestionnaireResponse.item.item.
type QuestionnaireResponseItemItem struct {
}

// QuestionnaireResponseItem represents a FHIR BackboneElement for QuestionnaireResponse.item.
type QuestionnaireResponseItem struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Pointer to specific item from Questionnaire
	LinkId string `json:"linkId"`
	// ElementDefinition - details for the item
	Definition *string `json:"definition,omitempty"`
	// Name for group or question text
	Text *string `json:"text,omitempty"`
	// The response(s) to the question
	Answer []QuestionnaireResponseItemAnswer `json:"answer,omitempty"`
	// Nested questionnaire response items
	Item []QuestionnaireResponseItemItem `json:"item,omitempty"`
}

// QuestionnaireResponse represents a FHIR QuestionnaireResponse.
type QuestionnaireResponse struct {
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
	// Unique id for this set of answers
	Identifier *Identifier `json:"identifier,omitempty"`
	// Request fulfilled by this QuestionnaireResponse
	BasedOn []Reference `json:"basedOn,omitempty"`
	// Part of this action
	PartOf []Reference `json:"partOf,omitempty"`
	// Form being answered
	Questionnaire *string `json:"questionnaire,omitempty"`
	// in-progress | completed | amended | entered-in-error | stopped
	Status string `json:"status"`
	// The subject of the questions
	Subject *Reference `json:"subject,omitempty"`
	// Encounter created as part of
	Encounter *Reference `json:"encounter,omitempty"`
	// Date the answers were gathered
	Authored *primitives.DateTime `json:"authored,omitempty"`
	// Person who received and recorded the answers
	Author *Reference `json:"author,omitempty"`
	// The person who answered the questions
	Source *Reference `json:"source,omitempty"`
	// Groups and questions
	Item []QuestionnaireResponseItem `json:"item,omitempty"`
}
