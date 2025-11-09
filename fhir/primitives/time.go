package primitives

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// Time represents a FHIR time primitive.
// FHIR time format: hh:mm:ss or hh:mm:ss.ffffff (with optional fractional seconds).
// Uses 24-hour format. No timezone information.
type Time struct {
	value string
}

// FHIR time format pattern
var (
	timePattern = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(\.\d+)?$`)
)

// NewTime creates a new Time from a string value.
// Returns error if the format is invalid.
func NewTime(value string) (Time, error) {
	t := Time{value: value}
	if err := t.Validate(); err != nil {
		return Time{}, err
	}
	return t, nil
}

// MustTime creates a new Time, panicking if invalid.
func MustTime(value string) Time {
	t, err := NewTime(value)
	if err != nil {
		panic(err)
	}
	return t
}

// String returns the string representation of the time.
func (t Time) String() string {
	return t.value
}

// Validate checks if the time string conforms to FHIR time format.
func (t Time) Validate() error {
	if t.value == "" {
		return fmt.Errorf("time cannot be empty")
	}

	if !timePattern.MatchString(t.value) {
		return fmt.Errorf("invalid FHIR time format: %s (expected hh:mm:ss or hh:mm:ss.ffffff)", t.value)
	}

	return nil
}

// Duration converts the time to time.Duration (duration from midnight).
func (t Time) Duration() (time.Duration, error) {
	// Parse the time using Go's time package
	tm, err := time.Parse("15:04:05", t.value)
	if err != nil {
		// Try with fractional seconds
		tm, err = time.Parse("15:04:05.999999999", t.value)
		if err != nil {
			return 0, fmt.Errorf("invalid time format: %s", t.value)
		}
	}

	// Calculate duration from midnight
	return time.Duration(tm.Hour())*time.Hour +
		time.Duration(tm.Minute())*time.Minute +
		time.Duration(tm.Second())*time.Second +
		time.Duration(tm.Nanosecond())*time.Nanosecond, nil
}

// TimeOfDay converts the time to a time.Time on the current date.
func (t Time) TimeOfDay(date time.Time) (time.Time, error) {
	duration, err := t.Duration()
	if err != nil {
		return time.Time{}, err
	}

	// Get the date at midnight
	midnight := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	// Add the duration
	return midnight.Add(duration), nil
}

// MarshalJSON implements json.Marshaler.
func (t Time) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(t.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	timeVal, err := NewTime(s)
	if err != nil {
		return err
	}

	*t = timeVal
	return nil
}

// IsZero reports whether the time is the zero value.
func (t Time) IsZero() bool {
	return t.value == ""
}

// Equal reports whether t and other represent the same time.
func (t Time) Equal(other Time) bool {
	return t.value == other.value
}

// FromDuration creates a Time from a time.Duration (duration from midnight).
func FromDuration(d time.Duration) Time {
	hours := int(d.Hours())
	d -= time.Duration(hours) * time.Hour
	minutes := int(d.Minutes())
	d -= time.Duration(minutes) * time.Minute
	seconds := int(d.Seconds())
	d -= time.Duration(seconds) * time.Second
	nanos := d.Nanoseconds()

	if nanos == 0 {
		return Time{value: fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)}
	}

	// Format with fractional seconds
	return Time{value: fmt.Sprintf("%02d:%02d:%02d.%09d", hours, minutes, seconds, nanos)}
}

// FromTimeTime creates a Time from a time.Time (extracts time component).
func FromTimeTime(t time.Time) Time {
	nanos := t.Nanosecond()
	if nanos == 0 {
		return Time{value: t.Format("15:04:05")}
	}

	// Format with fractional seconds, removing trailing zeros
	value := t.Format("15:04:05.999999999")
	// Remove trailing zeros from fractional seconds
	for i := len(value) - 1; i > 0; i-- {
		if value[i] != '0' {
			value = value[:i+1]
			break
		}
	}
	return Time{value: value}
}
