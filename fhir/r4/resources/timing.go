package resources

import "github.com/codeninja55/go-radx/fhir/primitives"

// TimingRepeat represents a FHIR BackboneElement for Timing.repeat.
type TimingRepeat struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Length/Range of lengths, or (Start and/or end) limits
	Bounds *any `json:"bounds,omitempty"`
	// Number of times to repeat
	Count *int `json:"count,omitempty"`
	// Maximum number of times to repeat
	CountMax *int `json:"countMax,omitempty"`
	// How long when it happens
	Duration *float64 `json:"duration,omitempty"`
	// How long when it happens (Max)
	DurationMax *float64 `json:"durationMax,omitempty"`
	// s | min | h | d | wk | mo | a - unit of time (UCUM)
	DurationUnit *string `json:"durationUnit,omitempty"`
	// Event occurs frequency times per period
	Frequency *int `json:"frequency,omitempty"`
	// Event occurs up to frequencyMax times per period
	FrequencyMax *int `json:"frequencyMax,omitempty"`
	// Event occurs frequency times per period
	Period *float64 `json:"period,omitempty"`
	// Upper limit of period (3-4 hours)
	PeriodMax *float64 `json:"periodMax,omitempty"`
	// s | min | h | d | wk | mo | a - unit of time (UCUM)
	PeriodUnit *string `json:"periodUnit,omitempty"`
	// mon | tue | wed | thu | fri | sat | sun
	DayOfWeek []string `json:"dayOfWeek,omitempty"`
	// Time of day for action
	TimeOfDay []primitives.Time `json:"timeOfDay,omitempty"`
	// Code for time period of occurrence
	When []string `json:"when,omitempty"`
	// Minutes from event (before or after)
	Offset *uint `json:"offset,omitempty"`
}

// Timing represents a FHIR Timing.
type Timing struct {
	// Unique id for inter-element referencing
	ID *string `json:"id,omitempty"`
	// Additional content defined by implementations
	Extension []Extension `json:"extension,omitempty"`
	// Extensions that cannot be ignored even if unrecognized
	ModifierExtension []Extension `json:"modifierExtension,omitempty"`
	// When the event occurs
	Event []primitives.DateTime `json:"event,omitempty"`
	// When the event is to occur
	Repeat *TimingRepeat `json:"repeat,omitempty"`
	// BID | TID | QID | AM | PM | QD | QOD | +
	Code *CodeableConcept `json:"code,omitempty"`
}
