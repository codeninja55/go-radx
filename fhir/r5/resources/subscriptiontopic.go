package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSubscriptionTopic is the FHIR resource type name for SubscriptionTopic.
const ResourceTypeSubscriptionTopic = "SubscriptionTopic"

// SubscriptionTopicResourceTriggerQueryCriteria represents a FHIR BackboneElement for SubscriptionTopic.resourceTrigger.queryCriteria.
type SubscriptionTopicResourceTriggerQueryCriteria struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Rule applied to previous resource state
	Previous *string `json:"previous,omitempty"`
	// test-passes | test-fails
	ResultForCreate *string `json:"resultForCreate,omitempty"`
	// Rule applied to current resource state
	Current *string `json:"current,omitempty"`
	// test-passes | test-fails
	ResultForDelete *string `json:"resultForDelete,omitempty"`
	// Both must be true flag
	RequireBoth *bool `json:"requireBoth,omitempty"`
}

// SubscriptionTopicResourceTrigger represents a FHIR BackboneElement for SubscriptionTopic.resourceTrigger.
type SubscriptionTopicResourceTrigger struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Text representation of the resource trigger
	Description *string `json:"description,omitempty"`
	// Data Type or Resource (reference to definition) for this trigger definition
	Resource string `json:"resource"`
	// create | update | delete
	SupportedInteraction []string `json:"supportedInteraction,omitempty"`
	// Query based trigger rule
	QueryCriteria *SubscriptionTopicResourceTriggerQueryCriteria `json:"queryCriteria,omitempty"`
	// FHIRPath based trigger rule
	FhirPathCriteria *string `json:"fhirPathCriteria,omitempty"`
}

// SubscriptionTopicEventTrigger represents a FHIR BackboneElement for SubscriptionTopic.eventTrigger.
type SubscriptionTopicEventTrigger struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Text representation of the event trigger
	Description *string `json:"description,omitempty"`
	// Event which can trigger a notification from the SubscriptionTopic
	Event CodeableConcept `json:"event"`
	// Data Type or Resource (reference to definition) for this trigger definition
	Resource string `json:"resource"`
}

// SubscriptionTopicCanFilterBy represents a FHIR BackboneElement for SubscriptionTopic.canFilterBy.
type SubscriptionTopicCanFilterBy struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Description of this filter parameter
	Description *string `json:"description,omitempty"`
	// URL of the triggering Resource that this filter applies to
	Resource *string `json:"resource,omitempty"`
	// Human-readable and computation-friendly name for a filter parameter usable by subscriptions on this topic, via Subscription.filterBy.filterParameter
	FilterParameter string `json:"filterParameter"`
	// Canonical URL for a filterParameter definition
	FilterDefinition *string `json:"filterDefinition,omitempty"`
	// eq | ne | gt | lt | ge | le | sa | eb | ap
	Comparator []string `json:"comparator,omitempty"`
	// missing | exact | contains | not | text | in | not-in | below | above | type | identifier | of-type | code-text | text-advanced | iterate
	Modifier []string `json:"modifier,omitempty"`
}

// SubscriptionTopicNotificationShape represents a FHIR BackboneElement for SubscriptionTopic.notificationShape.
type SubscriptionTopicNotificationShape struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// URL of the Resource that is the focus (main) resource in a notification shape
	Resource string `json:"resource"`
	// Include directives, rooted in the resource for this shape
	Include []string `json:"include,omitempty"`
	// Reverse include directives, rooted in the resource for this shape
	RevInclude []string `json:"revInclude,omitempty"`
}

// SubscriptionTopic represents a FHIR SubscriptionTopic.
type SubscriptionTopic struct {
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
	// Canonical identifier for this subscription topic, represented as an absolute URI (globally unique)
	URL string `json:"url"`
	// Business identifier for subscription topic
	Identifier []Identifier `json:"identifier,omitempty"`
	// Business version of the subscription topic
	Version *string `json:"version,omitempty"`
	// How to compare versions
	VersionAlgorithm *any `json:"versionAlgorithm,omitempty"`
	// Name for this subscription topic (computer friendly)
	Name *string `json:"name,omitempty"`
	// Name for this subscription topic (human friendly)
	Title *string `json:"title,omitempty"`
	// Based on FHIR protocol or definition
	DerivedFrom []string `json:"derivedFrom,omitempty"`
	// draft | active | retired | unknown
	Status string `json:"status"`
	// If for testing purposes, not real usage
	Experimental *bool `json:"experimental,omitempty"`
	// Date status first applied
	Date *primitives.DateTime `json:"date,omitempty"`
	// The name of the individual or organization that published the SubscriptionTopic
	Publisher *string `json:"publisher,omitempty"`
	// Contact details for the publisher
	Contact []ContactDetail `json:"contact,omitempty"`
	// Natural language description of the SubscriptionTopic
	Description *string `json:"description,omitempty"`
	// Content intends to support these contexts
	UseContext []UsageContext `json:"useContext,omitempty"`
	// Intended jurisdiction of the SubscriptionTopic (if applicable)
	Jurisdiction []CodeableConcept `json:"jurisdiction,omitempty"`
	// Why this SubscriptionTopic is defined
	Purpose *string `json:"purpose,omitempty"`
	// Use and/or publishing restrictions
	Copyright *string `json:"copyright,omitempty"`
	// Copyright holder and year(s)
	CopyrightLabel *string `json:"copyrightLabel,omitempty"`
	// When SubscriptionTopic is/was approved by publisher
	ApprovalDate *primitives.Date `json:"approvalDate,omitempty"`
	// Date the Subscription Topic was last reviewed by the publisher
	LastReviewDate *primitives.Date `json:"lastReviewDate,omitempty"`
	// The effective date range for the SubscriptionTopic
	EffectivePeriod *Period `json:"effectivePeriod,omitempty"`
	// Definition of a resource-based trigger for the subscription topic
	ResourceTrigger []SubscriptionTopicResourceTrigger `json:"resourceTrigger,omitempty"`
	// Event definitions the SubscriptionTopic
	EventTrigger []SubscriptionTopicEventTrigger `json:"eventTrigger,omitempty"`
	// Properties by which a Subscription can filter notifications from the SubscriptionTopic
	CanFilterBy []SubscriptionTopicCanFilterBy `json:"canFilterBy,omitempty"`
	// Properties for describing the shape of notifications generated by this topic
	NotificationShape []SubscriptionTopicNotificationShape `json:"notificationShape,omitempty"`
}
