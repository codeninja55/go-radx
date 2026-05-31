package dicom

import (
	"fmt"
	"strings"
	"time"
)

// TimePrecision records how many time-of-day components a parsed TM (or the time
// portion of a DT) carried, so a variable-precision value never has its absent
// components fabricated and its source fraction is never zero-filled.
type TimePrecision uint8

const (
	// TimePrecisionHour is HH.
	TimePrecisionHour TimePrecision = iota
	// TimePrecisionMinute is HHMM.
	TimePrecisionMinute
	// TimePrecisionSecond is HHMMSS.
	TimePrecisionSecond
	// TimePrecisionFraction is HHMMSS.F..FFFFFF (1-6 fractional digits).
	TimePrecisionFraction
)

// maxFractionDigits is the PS3.5 cap on TM/DT fractional-second digits.
const maxFractionDigits = 6

// timeOfDay is the parsed, normalised time-of-day shared by TM and DT. It records
// whether the source carried a leap second so callers can preserve the lexical 60
// while Time() reports the normalised 59.
type timeOfDay struct {
	hour, minute, second int
	nanos                int // 0..999_999_999, derived from the fractional digits
	leapSecond           bool
	precision            TimePrecision
}

// TM is VR TM (Time), the PS3.5 §6.2 form HHMMSS.FFFFFF with variable precision and
// optional fractional seconds. It preserves its source lexical form for a byte-stable
// round-trip and resolves to a Go time.Time on the zero date.
type TM struct {
	lexical string
	tod     timeOfDay
}

// ParseTM validates s as a DICOM TM value: 2, 4, or 6 leading digits (HH, HHMM,
// HHMMSS) optionally followed by a dot and 1-6 fractional-second digits. The
// leap-second value SS=60 is accepted (it is valid in DICOM); Time() normalises it to
// 59 while the preserved string keeps 60 (Codex DCM-010).
func ParseTM(s string) (TM, error) {
	tod, err := parseTimeOfDay(s, VRTM)
	if err != nil {
		return TM{}, err
	}
	return TM{lexical: s, tod: tod}, nil
}

// String returns the preserved lexical form.
func (t TM) String() string { return t.lexical }

// Precision reports how many components the source carried.
func (t TM) Precision() TimePrecision { return t.tod.precision }

// IsLeapSecond reports whether the source named the leap second SS=60.
func (t TM) IsLeapSecond() bool { return t.tod.leapSecond }

// Time resolves the time-of-day on the Go zero date in UTC. A source leap second is
// normalised to :59 (Go's time package has no representation for :60); the preserved
// String keeps 60. ok is false only for the zero value.
func (t TM) Time() (time.Time, bool) {
	if t.lexical == "" {
		return time.Time{}, false
	}
	return t.tod.toTime(time.UTC), true
}

// toTime builds a time.Time on the Go zero date (year 1) in loc from a time-of-day,
// normalising a leap second to :59.
func (tod timeOfDay) toTime(loc *time.Location) time.Time {
	sec := tod.second
	if tod.leapSecond {
		sec = 59
	}
	return time.Date(1, time.January, 1, tod.hour, tod.minute, sec, tod.nanos, loc)
}

// parseTimeOfDay parses the HHMMSS.FFFFFF time body shared by TM and DT. vr selects
// the VR named in any error.
func parseTimeOfDay(s string, vr VR) (timeOfDay, error) {
	if s == "" {
		return timeOfDay{}, &ValueError{VR: vr, Msg: "time is empty"}
	}

	whole, frac, hasFrac := strings.Cut(s, ".")
	if !allDigits(whole) {
		return timeOfDay{}, &ValueError{VR: vr, Msg: "time must be HHMMSS digits"}
	}

	var tod timeOfDay
	switch len(whole) {
	case 2:
		tod.precision = TimePrecisionHour
	case 4:
		tod.precision = TimePrecisionMinute
	case 6:
		tod.precision = TimePrecisionSecond
	default:
		return timeOfDay{}, &ValueError{VR: vr, Msg: fmt.Sprintf("time has %d integer digits, want 2, 4, or 6", len(whole))}
	}

	tod.hour = atoi(whole[0:2])
	if tod.hour > 23 {
		return timeOfDay{}, &ValueError{VR: vr, Msg: "time hour out of range (00-23)"}
	}
	if len(whole) >= 4 {
		tod.minute = atoi(whole[2:4])
		if tod.minute > 59 {
			return timeOfDay{}, &ValueError{VR: vr, Msg: "time minute out of range (00-59)"}
		}
	}
	if len(whole) >= 6 {
		tod.second = atoi(whole[4:6])
		switch {
		case tod.second == 60:
			tod.leapSecond = true // PS3.5 allows the leap second; Time() normalises to 59.
		case tod.second > 60:
			return timeOfDay{}, &ValueError{VR: vr, Msg: "time second out of range (00-60)"}
		}
	}

	if hasFrac {
		if tod.precision != TimePrecisionSecond {
			return timeOfDay{}, &ValueError{VR: vr, Msg: "fractional seconds require HHMMSS"}
		}
		if frac == "" || len(frac) > maxFractionDigits || !allDigits(frac) {
			return timeOfDay{}, &ValueError{VR: vr, Msg: "fractional seconds must be 1-6 digits"}
		}
		tod.precision = TimePrecisionFraction
		tod.nanos = fractionToNanos(frac)
	}

	return tod, nil
}

// fractionToNanos scales 1-6 fractional-second digits to nanoseconds without
// zero-filling the source string; the lexical form keeps the original digit count.
func fractionToNanos(frac string) int {
	// Right-pad conceptually to 9 digits (nanosecond resolution) by scaling.
	n := atoi(frac)
	for i := len(frac); i < 9; i++ {
		n *= 10
	}
	return n
}
