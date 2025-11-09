package resources

// TriggerDefinition represents a FHIR TriggerDefinition.
type TriggerDefinition struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// named-event | periodic | data-changed | data-added | data-modified | data-removed | data-accessed | data-access-ended
	Type string `json:"type"`
	// Name or URI that identifies the event
	Name *string `json:"name,omitempty"`
	// Coded definition of the event
	Code *CodeableConcept `json:"code,omitempty"`
	// What event
	SubscriptionTopic *string `json:"subscriptionTopic,omitempty"`
	// Timing of the event
	Timing *any `json:"timing,omitempty"`
	// Triggering data of the event (multiple = 'and')
	Data []DataRequirement `json:"data,omitempty"`
	// Whether the event triggers (boolean expression)
	Condition *Expression `json:"condition,omitempty"`
}
