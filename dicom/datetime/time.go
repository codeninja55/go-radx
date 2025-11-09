package datetime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Time represents a DICOM Time (TM) Value Representation.
//
// DICOM times follow the HHMMSS.FFFFFF format but support variable precision:
//   - HH (hours only)
//   - HHMM (hours and minutes)
//   - HHMMSS (hours, minutes, seconds)
//   - HHMMSS.F through HHMMSS.FFFFFF (fractional seconds, 1-6 decimal places)
//
// The Time field always contains a complete time.Time value (missing components
// default to 0). The Precision field tracks which components were actually
// specified in the original DICOM string.
type Time struct {
	// Time is the parsed time value. Missing components default to 0.
	Time time.Time

	// Precision indicates which components were specified in the DICOM string.
	Precision PrecisionLevel
}

var (
	// timeRegex matches DICOM TM format with optional fractional seconds
	// Groups: (HH)(MM)?(SS)?(.FFFFFF)?
	timeRegex = regexp.MustCompile(`^(\d{2})(?:(\d{2})(?:(\d{2})(?:\.(\d{1,6}))?)?)?$`)
)

// ParseTime parses a DICOM Time (TM) string into a Time struct.
//
// Supported formats:
//   - HH: Hours only (e.g., "14")
//   - HHMM: Hours and minutes (e.g., "1430")
//   - HHMMSS: Hours, minutes, seconds (e.g., "143025")
//   - HHMMSS.F to HHMMSS.FFFFFF: With fractional seconds (e.g., "143025.123456")
//
// Missing components default to 0:
//   - minutes: 0
//   - seconds: 0
//   - fractional seconds: 0
//
// Examples:
//
//	tim, err := ParseTime("143025.123456")  // 14:30:25.123456, microsecond precision
//	tim, err := ParseTime("143025")         // 14:30:25, second precision
//	tim, err := ParseTime("1430")           // 14:30:00, minute precision
//	tim, err := ParseTime("14")             // 14:00:00, hour precision
func ParseTime(s string) (Time, error) {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Check for empty input
	if s == "" {
		return Time{}, newParseError("TM", s, "empty input")
	}

	// Match against time regex
	matches := timeRegex.FindStringSubmatch(s)
	if matches == nil {
		return Time{}, newParseError("TM", s, "invalid format (expected HH, HHMM, HHMMSS, or HHMMSS.FFFFFF)")
	}

	return parseTimeComponents(s, matches)
}

// parseTimeComponents parses individual time components from regex matches.
func parseTimeComponents(input string, matches []string) (Time, error) {
	// matches[0] = full match
	// matches[1] = hours (always present)
	// matches[2] = minutes (optional)
	// matches[3] = seconds (optional)
	// matches[4] = fractional seconds (optional)

	// Parse hours (required)
	hour, err := strconv.Atoi(matches[1])
	if err != nil {
		return Time{}, newParseError("TM", input, "invalid hour")
	}
	if hour < 0 || hour > 23 {
		return Time{}, newRangeError("hour", hour, 0, 23)
	}

	// Default values for missing components
	minute := 0
	second := 0
	nanosecond := 0
	precision := PrecisionHours

	// Parse minutes if present
	if matches[2] != "" {
		minute, err = strconv.Atoi(matches[2])
		if err != nil {
			return Time{}, newParseError("TM", input, "invalid minute")
		}
		if minute < 0 || minute > 59 {
			return Time{}, newRangeError("minute", minute, 0, 59)
		}
		precision = PrecisionMinutes

		// Parse seconds if present
		if matches[3] != "" {
			second, err = strconv.Atoi(matches[3])
			if err != nil {
				return Time{}, newParseError("TM", input, "invalid second")
			}
			if second < 0 || second > 59 {
				return Time{}, newRangeError("second", second, 0, 59)
			}
			precision = PrecisionSeconds

			// Parse fractional seconds if present
			if matches[4] != "" {
				frac := matches[4]

				// Determine precision based on number of decimal places
				switch len(frac) {
				case 1:
					precision = PrecisionMS1
				case 2:
					precision = PrecisionMS2
				case 3:
					precision = PrecisionMS3
				case 4:
					precision = PrecisionMS4
				case 5:
					precision = PrecisionMS5
				case 6:
					precision = PrecisionMS6
				default:
					return Time{}, newParseError("TM", input, "fractional seconds must be 1-6 digits")
				}

				// Pad fractional part to 6 digits (microseconds)
				for len(frac) < 6 {
					frac += "0"
				}

				microseconds, err := strconv.Atoi(frac)
				if err != nil {
					return Time{}, newParseError("TM", input, "invalid fractional seconds")
				}

				// Convert microseconds to nanoseconds
				nanosecond = microseconds * 1000
			}
		}
	}

	// Create time.Time (use arbitrary date, we only care about time components)
	t := time.Date(1970, 1, 1, hour, minute, second, nanosecond, time.UTC)

	return Time{
		Time:      t,
		Precision: precision,
	}, nil
}

// DCM returns the time in DICOM TM format, respecting the original precision.
//
// Output format depends on precision:
//   - PrecisionHours: HH
//   - PrecisionMinutes: HHMM
//   - PrecisionSeconds: HHMMSS
//   - PrecisionMS1-MS6: HHMMSS.F through HHMMSS.FFFFFF
//
// Examples:
//
//	tim.DCM()  // "143025.123456" (microsecond precision)
//	tim.DCM()  // "143025" (second precision)
//	tim.DCM()  // "1430" (minute precision)
//	tim.DCM()  // "14" (hour precision)
func (t Time) DCM() string {
	switch t.Precision {
	case PrecisionHours:
		return fmt.Sprintf("%02d", t.Time.Hour())

	case PrecisionMinutes:
		return fmt.Sprintf("%02d%02d", t.Time.Hour(), t.Time.Minute())

	case PrecisionSeconds:
		return fmt.Sprintf("%02d%02d%02d", t.Time.Hour(), t.Time.Minute(), t.Time.Second())

	case PrecisionMS1:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d%02d%02d.%01d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/100000)

	case PrecisionMS2:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d%02d%02d.%02d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/10000)

	case PrecisionMS3:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d%02d%02d.%03d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/1000)

	case PrecisionMS4:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d%02d%02d.%04d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/100)

	case PrecisionMS5:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d%02d%02d.%05d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/10)

	case PrecisionMS6, PrecisionFull:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d%02d%02d.%06d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros)

	default:
		// Default to second precision
		return fmt.Sprintf("%02d%02d%02d", t.Time.Hour(), t.Time.Minute(), t.Time.Second())
	}
}

// String returns a human-readable representation of the time.
//
// Output format depends on precision:
//   - PrecisionHours: HH
//   - PrecisionMinutes: HH:MM
//   - PrecisionSeconds: HH:MM:SS
//   - PrecisionMS1-MS6: HH:MM:SS.F through HH:MM:SS.FFFFFF
//
// Examples:
//
//	tim.String()  // "14:30:25.123456" (microsecond precision)
//	tim.String()  // "14:30:25" (second precision)
//	tim.String()  // "14:30" (minute precision)
//	tim.String()  // "14" (hour precision)
func (t Time) String() string {
	switch t.Precision {
	case PrecisionHours:
		return fmt.Sprintf("%02d", t.Time.Hour())

	case PrecisionMinutes:
		return fmt.Sprintf("%02d:%02d", t.Time.Hour(), t.Time.Minute())

	case PrecisionSeconds:
		return fmt.Sprintf("%02d:%02d:%02d", t.Time.Hour(), t.Time.Minute(), t.Time.Second())

	case PrecisionMS1:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d:%02d:%02d.%01d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/100000)

	case PrecisionMS2:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d:%02d:%02d.%02d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/10000)

	case PrecisionMS3:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d:%02d:%02d.%03d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/1000)

	case PrecisionMS4:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d:%02d:%02d.%04d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/100)

	case PrecisionMS5:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d:%02d:%02d.%05d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros/10)

	case PrecisionMS6, PrecisionFull:
		micros := t.Time.Nanosecond() / 1000
		return fmt.Sprintf("%02d:%02d:%02d.%06d",
			t.Time.Hour(), t.Time.Minute(), t.Time.Second(), micros)

	default:
		return fmt.Sprintf("%02d:%02d:%02d", t.Time.Hour(), t.Time.Minute(), t.Time.Second())
	}
}
