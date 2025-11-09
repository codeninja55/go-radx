package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// ResourceTypeSubscription is the FHIR resource type name for Subscription.
const ResourceTypeSubscription = "Subscription"

// SubscriptionFilterBy represents a FHIR BackboneElement for Subscription.filterBy.
type SubscriptionFilterBy struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Allowed Resource (reference to definition) for this Subscription filter
	ResourceType *string `json:"resourceType,omitempty"`
	// Filter label defined in SubscriptionTopic
	FilterParameter string `json:"filterParameter"`
	// eq | ne | gt | lt | ge | le | sa | eb | ap
	Comparator *string `json:"comparator,omitempty"`
	// missing | exact | contains | not | text | in | not-in | below | above | type | identifier | of-type | code-text | text-advanced | iterate
	Modifier *string `json:"modifier,omitempty"`
	// Literal value or resource path
	Value string `json:"value"`
}

// SubscriptionParameter represents a FHIR BackboneElement for Subscription.parameter.
type SubscriptionParameter struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// Name (key) of the parameter
	Name string `json:"name"`
	// Value of the parameter to use or pass through
	Value string `json:"value"`
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
	// Additional identifiers (business identifier)
	Identifier []Identifier `json:"identifier,omitempty"`
	// Human readable name for this subscription
	Name *string `json:"name,omitempty"`
	// requested | active | error | off | entered-in-error
	Status string `json:"status"`
	// Reference to the subscription topic being subscribed to
	Topic string `json:"topic"`
	// Contact details for source (e.g. troubleshooting)
	Contact []ContactPoint `json:"contact,omitempty"`
	// When to automatically delete the subscription
	End *primitives.Instant `json:"end,omitempty"`
	// Entity responsible for Subscription changes
	ManagingEntity *Reference `json:"managingEntity,omitempty"`
	// Description of why this subscription was created
	Reason *string `json:"reason,omitempty"`
	// Criteria for narrowing the subscription topic stream
	FilterBy []SubscriptionFilterBy `json:"filterBy,omitempty"`
	// Channel type for notifications
	ChannelType Coding `json:"channelType"`
	// Where the channel points to
	Endpoint *string `json:"endpoint,omitempty"`
	// Channel type
	Parameter []SubscriptionParameter `json:"parameter,omitempty"`
	// Interval in seconds to send 'heartbeat' notification
	HeartbeatPeriod *uint `json:"heartbeatPeriod,omitempty"`
	// Timeout in seconds to attempt notification delivery
	Timeout *uint `json:"timeout,omitempty"`
	// MIME type to send, or omit for no payload
	ContentType *string `json:"contentType,omitempty"`
	// empty | id-only | full-resource
	Content *string `json:"content,omitempty"`
	// Maximum number of events that can be combined in a single notification
	MaxCount *int `json:"maxCount,omitempty"`
}
