package convert

import "github.com/codeninja55/go-radx/hl7v2"

// hl7DateTimeToFHIR renders an HL7 v2 DTM as a FHIR dateTime string, preserving
// the source precision rather than fabricating components the sender omitted. An
// HL7 DTM is a packed lexical form (for example "202605311230" or
// "202605311230-0500"); FHIR dateTime is ISO 8601 ("2026-05-31T12:30:00-05:00").
// A year/month/day-precision DTM yields the matching FHIR date form; an
// hour-or-finer DTM yields a full "YYYY-MM-DDThh:mm:ss" with a timezone, because
// FHIR forbids a timezone-less time. The offset is taken from the DTM's lexical
// "+/-ZZZZ" suffix when present; absent any offset, the converter does NOT
// fabricate one (the sender never stated its zone), falling back to the date-only
// form and recording the dropped time precision on the report under timePath. An
// absent DTM yields "".
func hl7DateTimeToFHIR(dtm hl7v2.DTM, report *Report, timePath string) string {
	t, prec, ok := dtm.Time()
	if !ok {
		return ""
	}
	date := pad4(t.Year()) + "-" + pad2(int(t.Month())) + "-" + pad2(t.Day())
	switch prec {
	case hl7v2.PrecisionYear:
		return pad4(t.Year())
	case hl7v2.PrecisionMonth:
		return pad4(t.Year()) + "-" + pad2(int(t.Month()))
	case hl7v2.PrecisionDay:
		return date
	default:
		// Hour, minute, second, or fraction precision: FHIR dateTime requires both
		// seconds and a timezone once a time component is present.
		lexical := dtm.String()
		offset, hasOffset := fhirOffsetFromDTM(lexical)
		if !hasOffset {
			report.dropped(timePath+" source DTM",
				"time present but the HL7 DTM carries no +/-ZZZZ offset; FHIR dateTime requires a timezone, so the time was dropped")
			return date
		}
		frac := dtmFraction(lexical)
		return date + "T" + pad2(t.Hour()) + ":" + pad2(t.Minute()) + ":" + pad2(t.Second()) + frac + offset
	}
}

// dtmFraction returns the fractional-second suffix (including the leading dot) of
// an HL7 DTM lexical form, or "" when none is present. It stops before any
// trailing +/-ZZZZ offset, so the offset is not mistaken for fraction digits. The
// fraction is preserved verbatim so sub-second precision is not lost.
func dtmFraction(lexical string) string {
	dot := -1
	for i := 0; i < len(lexical); i++ {
		switch {
		case lexical[i] == '.':
			dot = i
		case (lexical[i] == '+' || lexical[i] == '-') && i > 0:
			if dot < 0 || i <= dot+1 {
				return ""
			}
			return lexical[dot:i]
		}
	}
	if dot < 0 || dot+1 >= len(lexical) {
		return ""
	}
	return lexical[dot:]
}

// fhirOffsetFromDTM extracts a FHIR timezone suffix from an HL7 DTM lexical form's
// trailing "+/-ZZZZ" offset ("Z" for +0000, otherwise "+hh:mm"/"-hh:mm"). ok is
// false when the lexical form carries no well-formed offset.
func fhirOffsetFromDTM(lexical string) (string, bool) {
	// The offset sign can only follow the digit/fraction body, never lead it.
	for i := 1; i < len(lexical); i++ {
		c := lexical[i]
		if c != '+' && c != '-' {
			continue
		}
		zone := lexical[i+1:]
		if len(zone) != 4 {
			return "", false
		}
		hh, mm := zone[0:2], zone[2:4]
		if !validOffset(hh, mm) {
			return "", false
		}
		if c == '+' && hh == "00" && mm == "00" {
			return "Z", true
		}
		return string(c) + hh + ":" + mm, true
	}
	return "", false
}
