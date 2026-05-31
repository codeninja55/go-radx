package dicom

import (
	"fmt"
	"strings"
	"time"
)

// DTPrecision records how many components a parsed DT carried. DICOM DT is itself a
// variable-precision form (PS3.5 §6.2), so a partial value never has its absent
// components fabricated.
type DTPrecision uint8

const (
	// DTPrecisionYear is YYYY.
	DTPrecisionYear DTPrecision = iota
	// DTPrecisionMonth is YYYYMM.
	DTPrecisionMonth
	// DTPrecisionDay is YYYYMMDD.
	DTPrecisionDay
	// DTPrecisionHour is YYYYMMDDHH.
	DTPrecisionHour
	// DTPrecisionMinute is YYYYMMDDHHMM.
	DTPrecisionMinute
	// DTPrecisionSecond is YYYYMMDDHHMMSS.
	DTPrecisionSecond
	// DTPrecisionFraction is YYYYMMDDHHMMSS.F..FFFFFF.
	DTPrecisionFraction
)

// DT is VR DT (Date Time), the PS3.5 §6.2 form YYYYMMDDHHMMSS.FFFFFF&ZZXX where &ZZXX
// is an optional signed UTC offset (+/-HHMM). It preserves its source lexical form for
// a byte-stable round-trip and resolves to a zone-aware Go time.Time when it carries a
// full date.
type DT struct {
	lexical    string
	year       int
	month      int
	day        int
	tod        timeOfDay
	precision  DTPrecision
	hasOffset  bool
	offsetSecs int // signed UTC offset in seconds; only meaningful when hasOffset
}

// ParseDT validates s as a DICOM DT value: a variable-precision YYYY[MM[DD[HH[MM[SS[
// .FFFFFF]]]]]] datetime with an optional signed &ZZXX UTC offset. The leap-second
// value SS=60 is accepted and normalised to :59 by Time(); the preserved string keeps
// 60 (Codex DCM-010). Fractional precision is preserved exactly, never zero-filled.
func ParseDT(s string) (DT, error) {
	if s == "" {
		return DT{}, &ValueError{VR: VRDT, Msg: "datetime is empty"}
	}

	body, offsetSecs, hasOffset, err := splitDTOffset(s)
	if err != nil {
		return DT{}, err
	}

	dt := DT{lexical: s, hasOffset: hasOffset, offsetSecs: offsetSecs}

	whole, frac, hasFrac := strings.Cut(body, ".")
	if !allDigits(whole) {
		return DT{}, &ValueError{VR: VRDT, Msg: "datetime must be YYYYMMDDHHMMSS digits"}
	}

	// The integer body must be an even count of 4..14 digits stepping by two.
	switch len(whole) {
	case 4:
		dt.precision = DTPrecisionYear
	case 6:
		dt.precision = DTPrecisionMonth
	case 8:
		dt.precision = DTPrecisionDay
	case 10:
		dt.precision = DTPrecisionHour
	case 12:
		dt.precision = DTPrecisionMinute
	case 14:
		dt.precision = DTPrecisionSecond
	default:
		return DT{}, &ValueError{VR: VRDT, Msg: fmt.Sprintf("datetime has %d integer digits, want 4-14 in steps of 2", len(whole))}
	}

	dt.year = atoi(whole[0:4])
	if dt.year < 1 {
		return DT{}, &ValueError{VR: VRDT, Msg: "datetime year out of range"}
	}
	if len(whole) >= 6 {
		dt.month = atoi(whole[4:6])
		if dt.month < 1 || dt.month > 12 {
			return DT{}, &ValueError{VR: VRDT, Msg: "datetime month out of range (01-12)"}
		}
	}
	if len(whole) >= 8 {
		dt.day = atoi(whole[6:8])
		if !validYMD(dt.year, dt.month, dt.day) {
			return DT{}, &ValueError{VR: VRDT, Msg: "datetime is not a valid calendar date"}
		}
	}
	if len(whole) >= 10 {
		dt.tod.hour = atoi(whole[8:10])
		if dt.tod.hour > 23 {
			return DT{}, &ValueError{VR: VRDT, Msg: "datetime hour out of range (00-23)"}
		}
	}
	if len(whole) >= 12 {
		dt.tod.minute = atoi(whole[10:12])
		if dt.tod.minute > 59 {
			return DT{}, &ValueError{VR: VRDT, Msg: "datetime minute out of range (00-59)"}
		}
	}
	if len(whole) >= 14 {
		dt.tod.second = atoi(whole[12:14])
		switch {
		case dt.tod.second == 60:
			dt.tod.leapSecond = true // PS3.5 allows the leap second; Time() normalises to 59.
		case dt.tod.second > 60:
			return DT{}, &ValueError{VR: VRDT, Msg: "datetime second out of range (00-60)"}
		}
	}

	if hasFrac {
		if dt.precision != DTPrecisionSecond {
			return DT{}, &ValueError{VR: VRDT, Msg: "fractional seconds require a full YYYYMMDDHHMMSS datetime"}
		}
		if frac == "" || len(frac) > maxFractionDigits || !allDigits(frac) {
			return DT{}, &ValueError{VR: VRDT, Msg: "fractional seconds must be 1-6 digits"}
		}
		dt.precision = DTPrecisionFraction
		dt.tod.nanos = fractionToNanos(frac)
	}

	return dt, nil
}

// splitDTOffset peels an optional trailing signed &ZZXX (+/-HHMM) UTC offset off s,
// returning the remaining datetime body, the offset in seconds, and whether one was
// present.
func splitDTOffset(s string) (body string, offsetSecs int, hasOffset bool, err error) {
	idx := strings.IndexAny(s, "+-")
	if idx < 0 {
		return s, 0, false, nil
	}
	sign := 1
	if s[idx] == '-' {
		sign = -1
	}
	off := s[idx+1:]
	if len(off) != 4 || !allDigits(off) {
		return "", 0, false, &ValueError{VR: VRDT, Msg: "datetime offset must be (+|-)HHMM"}
	}
	oh, om := atoi(off[0:2]), atoi(off[2:4])
	if oh > 23 || om > 59 {
		return "", 0, false, &ValueError{VR: VRDT, Msg: "datetime offset out of range (HH 00-23, MM 00-59)"}
	}
	return s[:idx], sign * (oh*3600 + om*60), true, nil
}

// String returns the preserved lexical form.
func (t DT) String() string { return t.lexical }

// Precision reports how many components the source carried.
func (t DT) Precision() DTPrecision { return t.precision }

// HasOffset reports whether the source carried an explicit &ZZXX UTC offset.
func (t DT) HasOffset() bool { return t.hasOffset }

// OffsetSeconds returns the parsed UTC offset in seconds; it is 0 when HasOffset is
// false, which the implicit DICOM "local time" default treats as zone-unspecified.
func (t DT) OffsetSeconds() int { return t.offsetSecs }

// IsLeapSecond reports whether the source named the leap second SS=60.
func (t DT) IsLeapSecond() bool { return t.tod.leapSecond }

// Time resolves the datetime, carrying the parsed UTC offset (or UTC when the source
// gave none). A source leap second is normalised to :59. ok is false when the source
// lacked a full date (year/month/day), so an absent month, day, or time is never
// fabricated.
func (t DT) Time() (time.Time, bool) {
	if t.precision < DTPrecisionDay {
		return time.Time{}, false
	}
	loc := time.UTC
	if t.hasOffset {
		loc = time.FixedZone("", t.offsetSecs)
	}
	sec := t.tod.second
	if t.tod.leapSecond {
		sec = 59
	}
	return time.Date(t.year, time.Month(t.month), t.day,
		t.tod.hour, t.tod.minute, sec, t.tod.nanos, loc), true
}
