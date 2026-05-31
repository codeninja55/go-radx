package dicom

import (
	"fmt"
	"time"
)

// DatePrecision records which date components a parsed DA carried, so a partial
// (lenient) date never has its missing fields fabricated. A strict DA is always
// DatePrecisionDay.
type DatePrecision uint8

const (
	// DatePrecisionDay is a full YYYYMMDD date.
	DatePrecisionDay DatePrecision = iota
	// DatePrecisionMonth is a YYYYMM partial date (lenient mode only).
	DatePrecisionMonth
	// DatePrecisionYear is a YYYY partial date (lenient mode only).
	DatePrecisionYear
)

// DA is VR DA (Date), the PS3.5 §6.2 form YYYYMMDD. It preserves its source lexical
// form so a round-trip is byte-stable and resolves to a Go time.Time when it carries
// a full date.
type DA struct {
	lexical   string // preserved source form
	year      int
	month     int // 0 when absent (year-only lenient form)
	day       int // 0 when absent (year- or month-only lenient form)
	precision DatePrecision
}

// ParseDA validates s as a DICOM DA value. In strict mode (the default) it requires
// exactly 8 digits forming a valid YYYYMMDD calendar date. WithLenientDates (passed
// as withLenient) also accepts the legacy YYYY and YYYYMM partial forms. The
// prototype accepted the partial forms unconditionally, silently treating them as
// valid clinical metadata (Codex DCM-010).
func ParseDA(s string, opts ...DateOption) (DA, error) {
	cfg := newDateConfig(opts...)
	if !allDigits(s) {
		return DA{}, &ValueError{VR: VRDA, Msg: fmt.Sprintf("DA must be all digits, got %d-byte non-numeric form", len(s))}
	}

	switch len(s) {
	case 8:
		y, mo, d := atoi(s[0:4]), atoi(s[4:6]), atoi(s[6:8])
		if !validYMD(y, mo, d) {
			return DA{}, &ValueError{VR: VRDA, Msg: "DA is not a valid YYYYMMDD calendar date"}
		}
		return DA{lexical: s, year: y, month: mo, day: d, precision: DatePrecisionDay}, nil

	case 6:
		if !cfg.lenient {
			return DA{}, &ValueError{VR: VRDA, Msg: "DA requires 8 digits (YYYYMMDD); enable lenient dates for YYYYMM"}
		}
		y, mo := atoi(s[0:4]), atoi(s[4:6])
		if !validYearMonth(y, mo) {
			return DA{}, &ValueError{VR: VRDA, Msg: "DA YYYYMM is not a valid year-month"}
		}
		return DA{lexical: s, year: y, month: mo, precision: DatePrecisionMonth}, nil

	case 4:
		if !cfg.lenient {
			return DA{}, &ValueError{VR: VRDA, Msg: "DA requires 8 digits (YYYYMMDD); enable lenient dates for YYYY"}
		}
		y := atoi(s)
		if y < 1 {
			return DA{}, &ValueError{VR: VRDA, Msg: "DA YYYY year out of range"}
		}
		return DA{lexical: s, year: y, precision: DatePrecisionYear}, nil

	default:
		return DA{}, &ValueError{VR: VRDA, Msg: fmt.Sprintf("DA has %d digits, want 8 (or 4/6 in lenient mode)", len(s))}
	}
}

// String returns the preserved lexical form.
func (d DA) String() string { return d.lexical }

// Precision reports which components the source carried.
func (d DA) Precision() DatePrecision { return d.precision }

// Year returns the parsed year (0 only for the zero value).
func (d DA) Year() int { return d.year }

// Month returns the parsed month, or 0 for a year-only lenient date.
func (d DA) Month() int { return d.month }

// Day returns the parsed day, or 0 for a partial lenient date.
func (d DA) Day() int { return d.day }

// Time resolves a full date to midnight UTC. ok is false for a partial (lenient)
// date: the missing month or day is never fabricated.
func (d DA) Time() (time.Time, bool) {
	if d.precision != DatePrecisionDay {
		return time.Time{}, false
	}
	return time.Date(d.year, time.Month(d.month), d.day, 0, 0, 0, 0, time.UTC), true
}

// allDigits reports whether s is non-empty and contains only ASCII digits.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// atoi parses an all-digit string to an int. The caller guarantees s is non-empty
// digits and short enough not to overflow.
func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// validYearMonth reports whether y is a positive year and mo is 1..12.
func validYearMonth(y, mo int) bool {
	return y >= 1 && mo >= 1 && mo <= 12
}

// validYMD reports whether (y, mo, d) is a real calendar date, rejecting the
// non-existent days time.Date would otherwise normalise away (e.g. Feb 30).
func validYMD(y, mo, d int) bool {
	if !validYearMonth(y, mo) || d < 1 || d > 31 {
		return false
	}
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	return t.Year() == y && int(t.Month()) == mo && t.Day() == d
}
