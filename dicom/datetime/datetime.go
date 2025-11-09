package datetime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateTime represents a DICOM DateTime (DT) Value Representation.
//
// DICOM datetimes follow the YYYYMMDDHHMMSS.FFFFFF&ZZXX format where:
//   - Date: YYYYMMDD (with variable precision like Date type)
//   - Time: HHMMSS.FFFFFF (with variable precision like Time type)
//   - Timezone: &ZZXX where & is +/-, ZZ is hours (00-14), XX is minutes (00-59)
//
// All components except the year are optional. The NoOffset field indicates
// whether a timezone offset was specified in the original DICOM string.
type DateTime struct {
	// Time is the parsed datetime value with timezone if specified.
	Time time.Time

	// Precision indicates which components were specified in the DICOM string.
	Precision PrecisionLevel

	// NoOffset is true if no timezone offset was specified in the DICOM string.
	NoOffset bool
}

var (
	// datetimeRegex matches DICOM DT format with optional time and timezone
	// Format: YYYYMMDD[HHMMSS[.FFFFFF]][+/-HHMM]
	datetimeRegex = regexp.MustCompile(`^(\d{4})(?:(\d{2})(?:(\d{2})(?:(\d{2})(?:(\d{2})(?:(\d{2})(?:\.(\d{1,6}))?)?)?)?)?)?([+-]\d{4})?$`)
)

// ParseDateTime parses a DICOM DateTime (DT) string into a DateTime struct.
//
// Supported formats (all components except year are optional):
//   - YYYY: Year only
//   - YYYYMM: Year and month
//   - YYYYMMDD: Date only
//   - YYYYMMDDHH: Date and hours
//   - YYYYMMDDHHMM: Date, hours, minutes
//   - YYYYMMDDHHMMSS: Full date and time
//   - YYYYMMDDHHMMSS.FFFFFF: With fractional seconds (1-6 digits)
//   - Any of above + timezone: +HHMM or -HHMM
//
// Examples:
//
//	dt, err := ParseDateTime("20231015143025+1000")        // Full with timezone
//	dt, err := ParseDateTime("20231015143025")             // Full without timezone
//	dt, err := ParseDateTime("20231015143025.123456-0500") // With fractional seconds
//	dt, err := ParseDateTime("202310151430")               // Minute precision
//	dt, err := ParseDateTime("20231015")                   // Date only
func ParseDateTime(s string) (DateTime, error) {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Check for empty input
	if s == "" {
		return DateTime{}, newParseError("DT", s, "empty input")
	}

	// Match against datetime regex
	matches := datetimeRegex.FindStringSubmatch(s)
	if matches == nil {
		return DateTime{}, newParseError("DT", s, "invalid format")
	}

	return parseDateTimeComponents(s, matches)
}

// parseDateTimeComponents parses individual datetime components from regex matches.
func parseDateTimeComponents(input string, matches []string) (DateTime, error) {
	// matches[0] = full match
	// matches[1] = year (required)
	// matches[2] = month (optional)
	// matches[3] = day (optional)
	// matches[4] = hour (optional)
	// matches[5] = minute (optional)
	// matches[6] = second (optional)
	// matches[7] = fractional seconds (optional)
	// matches[8] = timezone offset (optional)

	// Parse year (required)
	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return DateTime{}, newParseError("DT", input, "invalid year")
	}

	// Default values for missing components
	month := 1
	day := 1
	hour := 0
	minute := 0
	second := 0
	nanosecond := 0
	precision := PrecisionYear

	// Parse month if present
	if matches[2] != "" {
		month, err = strconv.Atoi(matches[2])
		if err != nil {
			return DateTime{}, newParseError("DT", input, "invalid month")
		}
		if month < 1 || month > 12 {
			return DateTime{}, newRangeError("month", month, 1, 12)
		}
		precision = PrecisionMonth

		// Parse day if present
		if matches[3] != "" {
			day, err = strconv.Atoi(matches[3])
			if err != nil {
				return DateTime{}, newParseError("DT", input, "invalid day")
			}
			if day < 1 || day > 31 {
				return DateTime{}, newRangeError("day", day, 1, 31)
			}
			precision = PrecisionDay

			// Parse hour if present
			if matches[4] != "" {
				hour, err = strconv.Atoi(matches[4])
				if err != nil {
					return DateTime{}, newParseError("DT", input, "invalid hour")
				}
				if hour < 0 || hour > 23 {
					return DateTime{}, newRangeError("hour", hour, 0, 23)
				}
				precision = PrecisionHours

				// Parse minute if present
				if matches[5] != "" {
					minute, err = strconv.Atoi(matches[5])
					if err != nil {
						return DateTime{}, newParseError("DT", input, "invalid minute")
					}
					if minute < 0 || minute > 59 {
						return DateTime{}, newRangeError("minute", minute, 0, 59)
					}
					precision = PrecisionMinutes

					// Parse second if present
					if matches[6] != "" {
						second, err = strconv.Atoi(matches[6])
						if err != nil {
							return DateTime{}, newParseError("DT", input, "invalid second")
						}
						if second < 0 || second > 59 {
							return DateTime{}, newRangeError("second", second, 0, 59)
						}
						precision = PrecisionSeconds

						// Parse fractional seconds if present
						if matches[7] != "" {
							frac := matches[7]

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
							}

							// Pad fractional part to 6 digits (microseconds)
							for len(frac) < 6 {
								frac += "0"
							}

							microseconds, err := strconv.Atoi(frac)
							if err != nil {
								return DateTime{}, newParseError("DT", input, "invalid fractional seconds")
							}

							// Convert microseconds to nanoseconds
							nanosecond = microseconds * 1000
						}
					}
				}
			}
		}
	}

	// Parse timezone offset if present
	var loc *time.Location
	noOffset := true

	if matches[8] != "" {
		noOffset = false
		offset := matches[8]

		// Parse timezone: +/-HHMM
		if len(offset) != 5 {
			return DateTime{}, newParseError("DT", input, "invalid timezone format")
		}

		sign := offset[0]
		tzHour, err := strconv.Atoi(offset[1:3])
		if err != nil || tzHour < 0 || tzHour > 14 {
			return DateTime{}, newParseError("DT", input, "invalid timezone hour")
		}

		tzMin, err := strconv.Atoi(offset[3:5])
		if err != nil || tzMin < 0 || tzMin > 59 {
			return DateTime{}, newParseError("DT", input, "invalid timezone minute")
		}

		// Calculate offset in seconds
		offsetSeconds := tzHour*3600 + tzMin*60
		if sign == '-' {
			offsetSeconds = -offsetSeconds
		}

		// Create fixed timezone location
		loc = time.FixedZone(offset, offsetSeconds)
	} else {
		// No timezone specified, use UTC
		loc = time.UTC
	}

	// Create time.Time and validate date is real
	t := time.Date(year, time.Month(month), day, hour, minute, second, nanosecond, loc)

	// Check if date rolled over (e.g., Feb 31 became Mar 3)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return DateTime{}, newParseError("DT", input, "invalid day for month")
	}

	return DateTime{
		Time:      t,
		Precision: precision,
		NoOffset:  noOffset,
	}, nil
}

// DCM returns the datetime in DICOM DT format, respecting the original precision.
//
// Output format depends on precision and whether timezone was specified:
//   - PrecisionYear: YYYY
//   - PrecisionMonth: YYYYMM
//   - PrecisionDay: YYYYMMDD
//   - PrecisionHours: YYYYMMDDHH
//   - PrecisionMinutes: YYYYMMDDHHMM
//   - PrecisionSeconds: YYYYMMDDHHMMSS
//   - PrecisionMS1-MS6: YYYYMMDDHHMMSS.F through YYYYMMDDHHMMSS.FFFFFF
//   - Timezone appended if NoOffset is false: +HHMM or -HHMM
//
// Examples:
//
//	dt.DCM()  // "20231015143025+1000" (with timezone)
//	dt.DCM()  // "20231015143025" (without timezone)
//	dt.DCM()  // "202310151430" (minute precision)
func (dt DateTime) DCM() string {
	var base string

	switch dt.Precision {
	case PrecisionYear:
		base = dt.Time.Format("2006")
	case PrecisionMonth:
		base = dt.Time.Format("200601")
	case PrecisionDay:
		base = dt.Time.Format("20060102")
	case PrecisionHours:
		base = dt.Time.Format("2006010215")
	case PrecisionMinutes:
		base = dt.Time.Format("200601021504")
	case PrecisionSeconds:
		base = dt.Time.Format("20060102150405")
	case PrecisionMS1:
		micros := dt.Time.Nanosecond() / 1000
		base = fmt.Sprintf("%s.%01d", dt.Time.Format("20060102150405"), micros/100000)
	case PrecisionMS2:
		micros := dt.Time.Nanosecond() / 1000
		base = fmt.Sprintf("%s.%02d", dt.Time.Format("20060102150405"), micros/10000)
	case PrecisionMS3:
		micros := dt.Time.Nanosecond() / 1000
		base = fmt.Sprintf("%s.%03d", dt.Time.Format("20060102150405"), micros/1000)
	case PrecisionMS4:
		micros := dt.Time.Nanosecond() / 1000
		base = fmt.Sprintf("%s.%04d", dt.Time.Format("20060102150405"), micros/100)
	case PrecisionMS5:
		micros := dt.Time.Nanosecond() / 1000
		base = fmt.Sprintf("%s.%05d", dt.Time.Format("20060102150405"), micros/10)
	case PrecisionMS6, PrecisionFull:
		micros := dt.Time.Nanosecond() / 1000
		base = fmt.Sprintf("%s.%06d", dt.Time.Format("20060102150405"), micros)
	default:
		base = dt.Time.Format("20060102150405")
	}

	// Append timezone if present
	if !dt.NoOffset {
		_, offset := dt.Time.Zone()
		sign := "+"
		if offset < 0 {
			sign = "-"
			offset = -offset
		}
		hours := offset / 3600
		minutes := (offset % 3600) / 60
		base += fmt.Sprintf("%s%02d%02d", sign, hours, minutes)
	}

	return base
}

// String returns a human-readable representation of the datetime.
//
// Format: YYYY-MM-DD HH:MM:SS[.FFFFFF] ZONE
// Components depend on precision. Timezone shown as +HHMM, -HHMM, or UTC.
//
// Examples:
//
//	dt.String()  // "2023-10-15 14:30:25 +1000"
//	dt.String()  // "2023-10-15 14:30:25.123456 UTC"
//	dt.String()  // "2023-10-15 UTC" (date only)
func (dt DateTime) String() string {
	var base string

	switch dt.Precision {
	case PrecisionYear:
		base = dt.Time.Format("2006")
	case PrecisionMonth:
		base = dt.Time.Format("2006-01")
	case PrecisionDay:
		base = dt.Time.Format("2006-01-02")
	case PrecisionHours:
		base = dt.Time.Format("2006-01-02 15")
	case PrecisionMinutes:
		base = dt.Time.Format("2006-01-02 15:04")
	case PrecisionSeconds:
		base = dt.Time.Format("2006-01-02 15:04:05")
	case PrecisionMS1, PrecisionMS2, PrecisionMS3, PrecisionMS4, PrecisionMS5, PrecisionMS6, PrecisionFull:
		micros := dt.Time.Nanosecond() / 1000
		// Determine number of decimal places based on precision
		var fracFormat string
		switch dt.Precision {
		case PrecisionMS1:
			fracFormat = fmt.Sprintf(".%01d", micros/100000)
		case PrecisionMS2:
			fracFormat = fmt.Sprintf(".%02d", micros/10000)
		case PrecisionMS3:
			fracFormat = fmt.Sprintf(".%03d", micros/1000)
		case PrecisionMS4:
			fracFormat = fmt.Sprintf(".%04d", micros/100)
		case PrecisionMS5:
			fracFormat = fmt.Sprintf(".%05d", micros/10)
		default: // MS6, Full
			fracFormat = fmt.Sprintf(".%06d", micros)
		}
		base = dt.Time.Format("2006-01-02 15:04:05") + fracFormat
	default:
		base = dt.Time.Format("2006-01-02 15:04:05")
	}

	// Append timezone
	if dt.NoOffset {
		base += " UTC"
	} else {
		_, offset := dt.Time.Zone()
		sign := "+"
		if offset < 0 {
			sign = "-"
			offset = -offset
		}
		hours := offset / 3600
		minutes := (offset % 3600) / 60
		base += fmt.Sprintf(" %s%02d%02d", sign, hours, minutes)
	}

	return base
}
