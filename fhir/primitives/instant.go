package primitives

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// Instant represents a FHIR instant primitive.
// FHIR instant format: YYYY-MM-DDThh:mm:ss.sss+zz:zz (always includes timezone).
// Fractional seconds are optional.
type Instant struct {
	value string
}

// FHIR instant format pattern - requires timezone
var (
	instantPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)
)

// NewInstant creates a new Instant from a string value.
// Returns error if the format is invalid.
func NewInstant(value string) (Instant, error) {
	i := Instant{value: value}
	if err := i.Validate(); err != nil {
		return Instant{}, err
	}
	return i, nil
}

// MustInstant creates a new Instant, panicking if invalid.
func MustInstant(value string) Instant {
	i, err := NewInstant(value)
	if err != nil {
		panic(err)
	}
	return i
}

// String returns the string representation of the instant.
func (i Instant) String() string {
	return i.value
}

// Validate checks if the instant string conforms to FHIR instant format.
func (i Instant) Validate() error {
	if i.value == "" {
		return fmt.Errorf("instant cannot be empty")
	}

	if !instantPattern.MatchString(i.value) {
		return fmt.Errorf("invalid FHIR instant format: %s (expected YYYY-MM-DDThh:mm:ss[.sss](Z|+/-hh:mm))", i.value)
	}

	return nil
}

// Time converts the instant to time.Time.
func (i Instant) Time() (time.Time, error) {
	// Try RFC3339 format first
	t, err := time.Parse(time.RFC3339, i.value)
	if err == nil {
		return t, nil
	}

	// Try RFC3339Nano for fractional seconds
	t, err = time.Parse(time.RFC3339Nano, i.value)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid instant format: %s", i.value)
}

// MarshalJSON implements json.Marshaler.
func (i Instant) MarshalJSON() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(i.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (i *Instant) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	instant, err := NewInstant(s)
	if err != nil {
		return err
	}

	*i = instant
	return nil
}

// IsZero reports whether the instant is the zero value.
func (i Instant) IsZero() bool {
	return i.value == ""
}

// Equal reports whether i and other represent the same instant.
func (i Instant) Equal(other Instant) bool {
	return i.value == other.value
}

// FromTimeInstant creates an Instant from a time.Time.
func FromTimeInstant(t time.Time) Instant {
	return Instant{value: t.Format(time.RFC3339)}
}

// FromTimeInstantNano creates an Instant from a time.Time with nanosecond precision.
func FromTimeInstantNano(t time.Time) Instant {
	value := t.Format(time.RFC3339Nano)
	// Remove trailing zeros from fractional seconds
	for i := len(value) - 1; i > 0; i-- {
		if value[i] == '0' && i > 0 && value[i-1] != 'T' {
			continue
		}
		if value[i] != '0' || value[i-1] == '.' {
			value = value[:i+1]
			break
		}
	}
	return Instant{value: value}
}
