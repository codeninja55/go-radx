package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// AvailabilityAvailableTime represents a FHIR BackboneElement for Availability.availableTime.
type AvailabilityAvailableTime struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// mon | tue | wed | thu | fri | sat | sun
	DaysOfWeek []string `json:"daysOfWeek,omitempty"`
	// Always available? i.e. 24 hour service
	AllDay *bool `json:"allDay,omitempty"`
	// Opening time of day (ignored if allDay = true)
	AvailableStartTime *primitives.Time `json:"availableStartTime,omitempty"`
	// Closing time of day (ignored if allDay = true)
	AvailableEndTime *primitives.Time `json:"availableEndTime,omitempty"`
}

// AvailabilityNotAvailableTime represents a FHIR BackboneElement for Availability.notAvailableTime.
type AvailabilityNotAvailableTime struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Reason presented to the user explaining why time not available
	Description *string `json:"description,omitempty"`
	// Service not available during this period
	During *Period `json:"during,omitempty"`
}

// Availability represents a FHIR Availability.
type Availability struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Times the {item} is available
	AvailableTime []AvailabilityAvailableTime `json:"availableTime,omitempty"`
	// Not available during this time due to provided reason
	NotAvailableTime []AvailabilityNotAvailableTime `json:"notAvailableTime,omitempty"`
}
