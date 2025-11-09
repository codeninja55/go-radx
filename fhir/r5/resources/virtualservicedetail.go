package resources

// VirtualServiceDetail represents a FHIR VirtualServiceDetail.
type VirtualServiceDetail struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Channel Type
	ChannelType *Coding `json:"channelType,omitempty"`
	// Contact address/number
	Address *any `json:"address,omitempty"`
	// Address to see alternative connection details
	AdditionalInfo []string `json:"additionalInfo,omitempty"`
	// Maximum number of participants supported by the virtual service
	MaxParticipants *int `json:"maxParticipants,omitempty"`
	// Session Key required by the virtual service
	SessionKey *string `json:"sessionKey,omitempty"`
}
