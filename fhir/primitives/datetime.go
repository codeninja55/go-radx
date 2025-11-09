package primitives

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// DateTime represents a FHIR dateTime primitive.
// FHIR dateTime allows partial precision: YYYY, YYYY-MM, YYYY-MM-DD,
// YYYY-MM-DDThh:mm:ss+zz:zz (with optional fractional seconds).
type DateTime struct {
	value string
}

// FHIR dateTime format patterns
var (
	dateTimeYearPattern      = regexp.MustCompile(`^\d{4}$`)
	dateTimeYearMonthPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)
	dateTimeFullDatePattern  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	dateTimeWithTimePattern  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$`)
)

// NewDateTime creates a new DateTime from a string value.
// Returns error if the format is invalid.
func NewDateTime(value string) (DateTime, error) {
	dt := DateTime{value: value}
	if err := dt.Validate(); err != nil {
		return DateTime{}, err
	}
	return dt, nil
}

// MustDateTime creates a new DateTime, panicking if invalid.
func MustDateTime(value string) DateTime {
	dt, err := NewDateTime(value)
	if err != nil {
		panic(err)
	}
	return dt
}

// String returns the string representation of the datetime.
func (dt DateTime) String() string {
	return dt.value
}

// Validate checks if the datetime string conforms to FHIR dateTime format.
func (dt DateTime) Validate() error {
	if dt.value == "" {
		return fmt.Errorf("datetime cannot be empty")
	}

	// Check against valid patterns
	if dateTimeYearPattern.MatchString(dt.value) {
		return nil
	}
	if dateTimeYearMonthPattern.MatchString(dt.value) {
		return nil
	}
	if dateTimeFullDatePattern.MatchString(dt.value) {
		return nil
	}
	if dateTimeWithTimePattern.MatchString(dt.value) {
		return nil
	}

	return fmt.Errorf("invalid FHIR dateTime format: %s (expected YYYY, YYYY-MM, YYYY-MM-DD, or YYYY-MM-DDThh:mm:ss[.sss][Z|+/-hh:mm])", dt.value)
}

// Time converts the datetime to time.Time.
// For partial dates (year only or year-month), uses the first day.
// For dates without time, uses midnight UTC.
func (dt DateTime) Time() (time.Time, error) {
	// Try parsing as full datetime with timezone
	if dateTimeWithTimePattern.MatchString(dt.value) {
		// Try RFC3339 format first (with timezone)
		if t, err := time.Parse(time.RFC3339, dt.value); err == nil {
			return t, nil
		}
		// Try without timezone (assume UTC)
		if t, err := time.Parse("2006-01-02T15:04:05", dt.value); err == nil {
			return t.UTC(), nil
		}
		// Try with fractional seconds
		if t, err := time.Parse("2006-01-02T15:04:05.999999999", dt.value); err == nil {
			return t.UTC(), nil
		}
		return time.Time{}, fmt.Errorf("invalid datetime format: %s", dt.value)
	}

	// Try parsing as full date (midnight UTC)
	if dateTimeFullDatePattern.MatchString(dt.value) {
		return time.Parse("2006-01-02", dt.value)
	}

	// Try parsing as year-month (use first day of month, midnight UTC)
	if dateTimeYearMonthPattern.MatchString(dt.value) {
		return time.Parse("2006-01", dt.value)
	}

	// Try parsing as year only (use January 1st, midnight UTC)
	if dateTimeYearPattern.MatchString(dt.value) {
		return time.Parse("2006", dt.value)
	}

	return time.Time{}, fmt.Errorf("invalid datetime format: %s", dt.value)
}

// Precision returns the precision level of the datetime.
func (dt DateTime) Precision() string {
	if dateTimeYearPattern.MatchString(dt.value) {
		return "year"
	}
	if dateTimeYearMonthPattern.MatchString(dt.value) {
		return "month"
	}
	if dateTimeFullDatePattern.MatchString(dt.value) {
		return "day"
	}
	if dateTimeWithTimePattern.MatchString(dt.value) {
		return "second"
	}
	return "unknown"
}

// MarshalJSON implements json.Marshaler.
func (dt DateTime) MarshalJSON() ([]byte, error) {
	if err := dt.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(dt.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (dt *DateTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	datetime, err := NewDateTime(s)
	if err != nil {
		return err
	}

	*dt = datetime
	return nil
}

// IsZero reports whether the datetime is the zero value.
func (dt DateTime) IsZero() bool {
	return dt.value == ""
}

// Equal reports whether dt and other represent the same datetime.
func (dt DateTime) Equal(other DateTime) bool {
	return dt.value == other.value
}

// FromTimeDateTime creates a DateTime from a time.Time with full precision (RFC3339).
func FromTimeDateTime(t time.Time) DateTime {
	return DateTime{value: t.Format(time.RFC3339)}
}

// FromTimeDateTimeDate creates a DateTime from a time.Time with date precision only (YYYY-MM-DD).
func FromTimeDateTimeDate(t time.Time) DateTime {
	return DateTime{value: t.Format("2006-01-02")}
}

// FromTimeDateTimeYear creates a DateTime from a time.Time with year precision (YYYY).
func FromTimeDateTimeYear(t time.Time) DateTime {
	return DateTime{value: t.Format("2006")}
}

// FromTimeDateTimeMonth creates a DateTime from a time.Time with month precision (YYYY-MM).
func FromTimeDateTimeMonth(t time.Time) DateTime {
	return DateTime{value: t.Format("2006-01")}
}
