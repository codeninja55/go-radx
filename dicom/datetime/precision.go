package datetime

// PrecisionLevel represents the level of temporal precision in a DICOM date, time, or datetime value.
//
// DICOM temporal values support variable precision - not all components need to be specified.
// For example, a date might specify only year and month ("202310"), or a time might specify
// only hours and minutes ("1430"). This enum tracks which components were present in the
// original DICOM string.
//
// Precision levels are ordered from most precise to least precise, allowing comparisons.
type PrecisionLevel int

const (
	// PrecisionFull indicates all components specified with full microsecond precision.
	// Used for times like "143025.123456" or datetimes with full fractional seconds.
	PrecisionFull PrecisionLevel = iota

	// PrecisionMS6 indicates microsecond precision (6 decimal places in fractional seconds).
	// Example: "143025.123456"
	PrecisionMS6

	// PrecisionMS5 indicates precision to 1/100000 second (5 decimal places).
	// Example: "143025.12345"
	PrecisionMS5

	// PrecisionMS4 indicates precision to 1/10000 second (4 decimal places).
	// Example: "143025.1234"
	PrecisionMS4

	// PrecisionMS3 indicates precision to 1/1000 second (3 decimal places, milliseconds).
	// Example: "143025.123"
	PrecisionMS3

	// PrecisionMS2 indicates precision to 1/100 second (2 decimal places, centiseconds).
	// Example: "143025.12"
	PrecisionMS2

	// PrecisionMS1 indicates precision to 1/10 second (1 decimal place, deciseconds).
	// Example: "143025.1"
	PrecisionMS1

	// PrecisionSeconds indicates precision to the second level.
	// Example: "143025" (time) or "20231015143025" (datetime)
	PrecisionSeconds

	// PrecisionMinutes indicates precision to the minute level.
	// Example: "1430" (time) or "202310151430" (datetime)
	PrecisionMinutes

	// PrecisionHours indicates precision to the hour level.
	// Example: "14" (time) or "2023101514" (datetime)
	PrecisionHours

	// PrecisionDay indicates precision to the day level.
	// Example: "20231015" (date) or datetime with date only
	PrecisionDay

	// PrecisionMonth indicates precision to the month level.
	// Example: "202310" (date)
	PrecisionMonth

	// PrecisionYear indicates precision to the year level only.
	// Example: "2023" (date)
	PrecisionYear
)

// String returns a human-readable representation of the precision level.
func (p PrecisionLevel) String() string {
	switch p {
	case PrecisionFull:
		return "Full"
	case PrecisionMS6:
		return "Microsecond"
	case PrecisionMS5:
		return "MS5"
	case PrecisionMS4:
		return "MS4"
	case PrecisionMS3:
		return "Millisecond"
	case PrecisionMS2:
		return "Centisecond"
	case PrecisionMS1:
		return "Decisecond"
	case PrecisionSeconds:
		return "Second"
	case PrecisionMinutes:
		return "Minute"
	case PrecisionHours:
		return "Hour"
	case PrecisionDay:
		return "Day"
	case PrecisionMonth:
		return "Month"
	case PrecisionYear:
		return "Year"
	default:
		return "Unknown"
	}
}

// AgeUnit represents the time unit used in DICOM Age String (AS) Value Representation.
//
// DICOM Age Strings follow the format "nnnU" where:
//   - nnn is a three-digit number (000-999)
//   - U is a single character indicating the unit (D, W, M, or Y)
//
// Example: "042Y" represents 42 years, "007D" represents 7 days.
type AgeUnit int

const (
	// Days represents age in days. DICOM code: 'D'
	// Example: "007D" = 7 days
	Days AgeUnit = iota

	// Weeks represents age in weeks. DICOM code: 'W'
	// Example: "004W" = 4 weeks
	Weeks

	// Months represents age in months. DICOM code: 'M'
	// Example: "006M" = 6 months
	Months

	// Years represents age in years. DICOM code: 'Y'
	// Example: "042Y" = 42 years
	Years
)

// String returns the DICOM single-character code for the age unit.
func (u AgeUnit) String() string {
	switch u {
	case Days:
		return "D"
	case Weeks:
		return "W"
	case Months:
		return "M"
	case Years:
		return "Y"
	default:
		return ""
	}
}

// LongString returns a human-readable description of the age unit.
func (u AgeUnit) LongString() string {
	switch u {
	case Days:
		return "days"
	case Weeks:
		return "weeks"
	case Months:
		return "months"
	case Years:
		return "years"
	default:
		return "unknown"
	}
}
