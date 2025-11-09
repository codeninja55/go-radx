package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSubscription is the FHIR resource type name for Subscription.
const ResourceTypeSubscription = "Subscription"

// SubscriptionChannel represents a FHIR BackboneElement for Subscription.channel.
type SubscriptionChannel struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// rest-hook | websocket | email | sms | message
	Type string `json:"type"`
	// Where the channel points to
	Endpoint *string `json:"endpoint,omitempty"`
	// MIME type to send, or omit for no payload
	Payload *string `json:"payload,omitempty"`
	// Usage depends on the channel type
	Header []string `json:"header,omitempty"`
}

// Subscription represents a FHIR Subscription.
type Subscription struct {
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
	// requested | active | error | off
	Status string `json:"status"`
	// Contact details for source (e.g. troubleshooting)
	Contact []ContactPoint `json:"contact,omitempty"`
	// When to automatically delete the subscription
	End *primitives.Instant `json:"end,omitempty"`
	// Description of why this subscription was created
	Reason string `json:"reason"`
	// Rule for server push
	Criteria string `json:"criteria"`
	// Latest error note
	Error *string `json:"error,omitempty"`
	// The channel on which to report matches to the criteria
	Channel SubscriptionChannel `json:"channel"`
}
