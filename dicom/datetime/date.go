package datetime

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Date represents a DICOM Date (DA) Value Representation.
//
// DICOM dates follow the YYYYMMDD format but support variable precision:
//   - YYYY (year only)
//   - YYYYMM (year and month)
//   - YYYYMMDD (full date)
//
// Legacy NEMA-300 format YYYY.MM.DD is also supported.
//
// The Time field always contains a complete time.Time value (missing components
// default to 1 for month/day to avoid date rollback). The Precision field tracks
// which components were actually specified in the original DICOM string.
type Date struct {
	// Time is the parsed date value. Missing components default to:
	// month=January (1), day=1.
	Time time.Time

	// Precision indicates which components were specified in the DICOM string.
	Precision PrecisionLevel

	// IsNEMA indicates if the date was in legacy NEMA-300 format (YYYY.MM.DD).
	IsNEMA bool
}

var (
	// dateRegex matches DICOM DA format: YYYYMMDD, YYYYMM, or YYYY
	dateRegex = regexp.MustCompile(`^(\d{4})(?:(\d{2})(?:(\d{2}))?)?$`)

	// nemaDateRegex matches legacy NEMA-300 format: YYYY.MM.DD
	nemaDateRegex = regexp.MustCompile(`^(\d{4})\.(\d{2})\.(\d{2})$`)
)

// ParseDate parses a DICOM Date (DA) string into a Date struct.
//
// Supported formats:
//   - YYYYMMDD: Full date (e.g., "20231015")
//   - YYYYMM: Year and month (e.g., "202310")
//   - YYYY: Year only (e.g., "2023")
//   - YYYY.MM.DD: Legacy NEMA-300 format (e.g., "2023.10.15")
//
// Missing components default to:
//   - month: January (1)
//   - day: 1
//
// Examples:
//
//	date, err := ParseDate("20231015")  // 2023-10-15, day precision
//	date, err := ParseDate("202310")    // 2023-10-01, month precision
//	date, err := ParseDate("2023")      // 2023-01-01, year precision
//	date, err := ParseDate("2023.10.15") // 2023-10-15, NEMA format
func ParseDate(s string) (Date, error) {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Check for empty input
	if s == "" {
		return Date{}, newParseError("DA", s, "empty input")
	}

	// Try NEMA format first (YYYY.MM.DD)
	if matches := nemaDateRegex.FindStringSubmatch(s); matches != nil {
		return parseNEMADate(s, matches)
	}

	// Try standard DICOM format (YYYYMMDD, YYYYMM, YYYY)
	matches := dateRegex.FindStringSubmatch(s)
	if matches == nil {
		return Date{}, newParseError("DA", s, "invalid format (expected YYYYMMDD, YYYYMM, YYYY, or YYYY.MM.DD)")
	}

	return parseStandardDate(s, matches)
}

// parseStandardDate parses standard DICOM date format (YYYYMMDD, YYYYMM, YYYY).
func parseStandardDate(input string, matches []string) (Date, error) {
	// matches[0] = full match
	// matches[1] = year (always present)
	// matches[2] = month (optional)
	// matches[3] = day (optional)

	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return Date{}, newParseError("DA", input, "invalid year")
	}

	// Default values for missing components
	month := 1
	day := 1
	precision := PrecisionYear

	// Parse month if present
	if matches[2] != "" {
		month, err = strconv.Atoi(matches[2])
		if err != nil {
			return Date{}, newParseError("DA", input, "invalid month")
		}
		if month < 1 || month > 12 {
			return Date{}, newRangeError("month", month, 1, 12)
		}
		precision = PrecisionMonth

		// Parse day if present
		if matches[3] != "" {
			day, err = strconv.Atoi(matches[3])
			if err != nil {
				return Date{}, newParseError("DA", input, "invalid day")
			}
			if day < 1 || day > 31 {
				return Date{}, newRangeError("day", day, 1, 31)
			}
			precision = PrecisionDay
		}
	}

	// Create time.Time and validate date is real (catches Feb 31, etc.)
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// Check if date rolled over (e.g., Feb 31 became Mar 3)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return Date{}, newParseError("DA", input, "invalid day for month")
	}

	return Date{
		Time:      t,
		Precision: precision,
		IsNEMA:    false,
	}, nil
}

// parseNEMADate parses legacy NEMA-300 format (YYYY.MM.DD).
func parseNEMADate(input string, matches []string) (Date, error) {
	// matches[0] = full match
	// matches[1] = year
	// matches[2] = month
	// matches[3] = day

	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return Date{}, newParseError("DA", input, "invalid year")
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil {
		return Date{}, newParseError("DA", input, "invalid month")
	}
	if month < 1 || month > 12 {
		return Date{}, newRangeError("month", month, 1, 12)
	}

	day, err := strconv.Atoi(matches[3])
	if err != nil {
		return Date{}, newParseError("DA", input, "invalid day")
	}
	if day < 1 || day > 31 {
		return Date{}, newRangeError("day", day, 1, 31)
	}

	// Create time.Time and validate date is real
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// Check if date rolled over
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return Date{}, newParseError("DA", input, "invalid day for month")
	}

	return Date{
		Time:      t,
		Precision: PrecisionDay, // NEMA format always full precision
		IsNEMA:    true,
	}, nil
}

// DCM returns the date in DICOM DA format, respecting the original precision.
//
// Output format depends on precision:
//   - PrecisionDay: YYYYMMDD or YYYY.MM.DD (if NEMA)
//   - PrecisionMonth: YYYYMM
//   - PrecisionYear: YYYY
//
// Examples:
//
//	date.DCM()  // "20231015" (day precision)
//	date.DCM()  // "202310" (month precision)
//	date.DCM()  // "2023" (year precision)
//	date.DCM()  // "2023.10.15" (NEMA format)
func (d Date) DCM() string {
	if d.IsNEMA {
		return d.Time.Format("2006.01.02")
	}

	switch d.Precision {
	case PrecisionDay:
		return d.Time.Format("20060102")
	case PrecisionMonth:
		return d.Time.Format("200601")
	case PrecisionYear:
		return d.Time.Format("2006")
	default:
		// Default to full precision
		return d.Time.Format("20060102")
	}
}

// String returns a human-readable representation of the date.
//
// Output format depends on precision:
//   - PrecisionDay: YYYY-MM-DD
//   - PrecisionMonth: YYYY-MM
//   - PrecisionYear: YYYY
//
// Examples:
//
//	date.String()  // "2023-10-15" (day precision)
//	date.String()  // "2023-10" (month precision)
//	date.String()  // "2023" (year precision)
func (d Date) String() string {
	switch d.Precision {
	case PrecisionDay:
		return d.Time.Format("2006-01-02")
	case PrecisionMonth:
		return d.Time.Format("2006-01")
	case PrecisionYear:
		return d.Time.Format("2006")
	default:
		return d.Time.Format("2006-01-02")
	}
}
