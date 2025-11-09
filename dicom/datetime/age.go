package datetime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Age represents a DICOM Age String (AS) Value Representation.
//
// DICOM age strings follow the nnnU format where:
//   - nnn: A 3-digit number (000-999)
//   - U: A single-character unit: D (days), W (weeks), M (months), or Y (years)
//
// Examples: "007D" (7 days), "004W" (4 weeks), "006M" (6 months), "042Y" (42 years)
//
// The Age type provides conversion to time.Duration using standard medical factors:
//   - Days: 24 hours
//   - Weeks: 7 days
//   - Months: 30.4375 days (365.25/12 - accounts for leap years)
//   - Years: 365.25 days (accounts for leap years)
type Age struct {
	// Value is the numeric age component (0-999).
	Value int

	// Unit is the time unit (Days, Weeks, Months, or Years).
	Unit AgeUnit
}

var (
	// ageRegex matches DICOM AS format: nnnU where nnn is 000-999, U is D/W/M/Y
	ageRegex = regexp.MustCompile(`^(\d{3})([DWMY])$`)
)

// ParseAge parses a DICOM Age String (AS) into an Age struct.
//
// The format must be exactly 4 characters: nnnU where:
//   - nnn: 3-digit number (000-999)
//   - U: Unit character (D, W, M, or Y)
//
// Examples:
//
//	age, err := ParseAge("007D")  // 7 days
//	age, err := ParseAge("004W")  // 4 weeks
//	age, err := ParseAge("006M")  // 6 months
//	age, err := ParseAge("042Y")  // 42 years
func ParseAge(s string) (Age, error) {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Check for empty input
	if s == "" {
		return Age{}, newParseError("AS", s, "empty input")
	}

	// Match against age regex
	matches := ageRegex.FindStringSubmatch(s)
	if matches == nil {
		return Age{}, newParseError("AS", s, "invalid format (expected nnnU where nnn=000-999, U=D/W/M/Y)")
	}

	return parseAgeComponents(s, matches)
}

// parseAgeComponents parses individual age components from regex matches.
func parseAgeComponents(input string, matches []string) (Age, error) {
	// matches[0] = full match
	// matches[1] = value (3 digits)
	// matches[2] = unit (D/W/M/Y)

	// Parse value
	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return Age{}, newParseError("AS", input, "invalid numeric value")
	}

	// Value is already validated by regex to be 000-999
	// No range check needed

	// Parse unit
	var unit AgeUnit
	switch matches[2] {
	case "D":
		unit = Days
	case "W":
		unit = Weeks
	case "M":
		unit = Months
	case "Y":
		unit = Years
	default:
		return Age{}, newParseError("AS", input, fmt.Sprintf("invalid unit '%s' (expected D, W, M, or Y)", matches[2]))
	}

	return Age{
		Value: value,
		Unit:  unit,
	}, nil
}

// Duration converts the age to a time.Duration using standard medical factors.
//
// Conversion factors:
//   - Days: 24 hours
//   - Weeks: 7 days = 168 hours
//   - Months: 30.4375 days = 730.5 hours (365.25/12, accounts for leap years)
//   - Years: 365.25 days = 8766 hours (accounts for leap years)
//
// Note: The fractional days in months and years are handled by converting to
// float64, performing the calculation, then converting back to Duration.
//
// Examples:
//
//	age := Age{Value: 7, Unit: Days}
//	d := age.Duration()  // 168 hours (7 days)
//
//	age := Age{Value: 1, Unit: Years}
//	d := age.Duration()  // 8766 hours (365.25 days)
func (a Age) Duration() time.Duration {
	switch a.Unit {
	case Days:
		return time.Duration(a.Value) * 24 * time.Hour

	case Weeks:
		return time.Duration(a.Value) * 7 * 24 * time.Hour

	case Months:
		// 30.4375 days per month (365.25/12)
		days := float64(a.Value) * 30.4375
		return time.Duration(days * 24 * float64(time.Hour))

	case Years:
		// 365.25 days per year (accounts for leap years)
		days := float64(a.Value) * 365.25
		return time.Duration(days * 24 * float64(time.Hour))

	default:
		return 0
	}
}

// DCM returns the age in DICOM AS format.
//
// Output format: nnnU where nnn is zero-padded to 3 digits.
//
// Examples:
//
//	age.DCM()  // "007D" (7 days)
//	age.DCM()  // "042Y" (42 years)
//	age.DCM()  // "000D" (0 days)
func (a Age) DCM() string {
	return fmt.Sprintf("%03d%s", a.Value, a.Unit.String())
}

// String returns a human-readable representation of the age.
//
// Format: "n unit" or "n units" (singular/plural based on value).
//
// Examples:
//
//	age.String()  // "7 days"
//	age.String()  // "1 day" (singular)
//	age.String()  // "42 years"
//	age.String()  // "0 days"
func (a Age) String() string {
	unitStr := a.Unit.LongString()

	// Handle singular vs plural
	if a.Value == 1 {
		// Remove trailing 's' for singular
		if len(unitStr) > 0 && unitStr[len(unitStr)-1] == 's' {
			unitStr = unitStr[:len(unitStr)-1]
		}
	}

	return fmt.Sprintf("%d %s", a.Value, unitStr)
}
