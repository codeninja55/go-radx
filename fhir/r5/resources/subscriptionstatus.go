package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSubscriptionStatus is the FHIR resource type name for SubscriptionStatus.
const ResourceTypeSubscriptionStatus = "SubscriptionStatus"

// SubscriptionStatusNotificationEvent represents a FHIR BackboneElement for SubscriptionStatus.notificationEvent.
type SubscriptionStatusNotificationEvent struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Sequencing index of this event
	EventNumber int64 `json:"eventNumber"`
	// The instant this event occurred
	Timestamp *primitives.Instant `json:"timestamp,omitempty"`
	// Reference to the primary resource or information of this event
	Focus *Reference `json:"focus,omitempty"`
	// References related to the focus resource and/or context of this event
	AdditionalContext []Reference `json:"additionalContext,omitempty"`
}

// SubscriptionStatus represents a FHIR SubscriptionStatus.
type SubscriptionStatus struct {
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
	// requested | active | error | off | entered-in-error
	Status *string `json:"status,omitempty"`
	// handshake | heartbeat | event-notification | query-status | query-event
	Type string `json:"type"`
	// Events since the Subscription was created
	EventsSinceSubscriptionStart *int64 `json:"eventsSinceSubscriptionStart,omitempty"`
	// Detailed information about any events relevant to this notification
	NotificationEvent []SubscriptionStatusNotificationEvent `json:"notificationEvent,omitempty"`
	// Reference to the Subscription responsible for this notification
	Subscription Reference `json:"subscription"`
	// Reference to the SubscriptionTopic this notification relates to
	Topic *string `json:"topic,omitempty"`
	// List of errors on the subscription
	Error []CodeableConcept `json:"error,omitempty"`
}
