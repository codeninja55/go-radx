package primitives

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// Date represents a FHIR date primitive.
// FHIR dates allow partial precision: YYYY, YYYY-MM, or YYYY-MM-DD.
type Date struct {
	value string
}

// FHIR date format patterns
var (
	dateYearPattern      = regexp.MustCompile(`^\d{4}$`)
	dateYearMonthPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)
	dateFullPattern      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// NewDate creates a new Date from a string value.
// Returns error if the format is invalid.
func NewDate(value string) (Date, error) {
	d := Date{value: value}
	if err := d.Validate(); err != nil {
		return Date{}, err
	}
	return d, nil
}

// MustDate creates a new Date, panicking if invalid.
func MustDate(value string) Date {
	d, err := NewDate(value)
	if err != nil {
		panic(err)
	}
	return d
}

// String returns the string representation of the date.
func (d Date) String() string {
	return d.value
}

// Validate checks if the date string conforms to FHIR date format.
func (d Date) Validate() error {
	if d.value == "" {
		return fmt.Errorf("date cannot be empty")
	}

	// Check against valid patterns
	if dateYearPattern.MatchString(d.value) {
		return nil
	}
	if dateYearMonthPattern.MatchString(d.value) {
		return nil
	}
	if dateFullPattern.MatchString(d.value) {
		return nil
	}

	return fmt.Errorf("invalid FHIR date format: %s (expected YYYY, YYYY-MM, or YYYY-MM-DD)", d.value)
}

// Time converts the date to time.Time.
// For partial dates (year only or year-month), uses the first day.
func (d Date) Time() (time.Time, error) {
	// Try parsing as full date
	if dateFullPattern.MatchString(d.value) {
		return time.Parse("2006-01-02", d.value)
	}

	// Try parsing as year-month (use first day of month)
	if dateYearMonthPattern.MatchString(d.value) {
		return time.Parse("2006-01", d.value)
	}

	// Try parsing as year only (use January 1st)
	if dateYearPattern.MatchString(d.value) {
		return time.Parse("2006", d.value)
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", d.value)
}

// Precision returns the precision level of the date.
func (d Date) Precision() string {
	if dateYearPattern.MatchString(d.value) {
		return "year"
	}
	if dateYearMonthPattern.MatchString(d.value) {
		return "month"
	}
	if dateFullPattern.MatchString(d.value) {
		return "day"
	}
	return "unknown"
}

// MarshalJSON implements json.Marshaler.
func (d Date) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	date, err := NewDate(s)
	if err != nil {
		return err
	}

	*d = date
	return nil
}

// IsZero reports whether the date is the zero value.
func (d Date) IsZero() bool {
	return d.value == ""
}

// Equal reports whether d and other represent the same date.
func (d Date) Equal(other Date) bool {
	return d.value == other.value
}

// FromTime creates a Date from a time.Time with full precision (YYYY-MM-DD).
func FromTime(t time.Time) Date {
	return Date{value: t.Format("2006-01-02")}
}

// FromTimeYear creates a Date from a time.Time with year precision (YYYY).
func FromTimeYear(t time.Time) Date {
	return Date{value: t.Format("2006")}
}

// FromTimeMonth creates a Date from a time.Time with month precision (YYYY-MM).
func FromTimeMonth(t time.Time) Date {
	return Date{value: t.Format("2006-01")}
}
