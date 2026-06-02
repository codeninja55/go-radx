package hl7v2

import (
	"fmt"
	"time"
)

// HD — hierarchic designator (namespace + universal ID). Used as the assigning
// authority of a CX and the application/facility fields of MSH.
type HD struct {
	NamespaceID     string // HD-1
	UniversalID     string // HD-2
	UniversalIDType string // HD-3, e.g. "ISO"
}

// parseHD reads an HD from a repetition's components.
func parseHD(r Repetition) HD {
	return HD{
		NamespaceID:     r.component(1),
		UniversalID:     r.component(2),
		UniversalIDType: r.component(3),
	}
}

// CX — extended composite ID with check digit (PID-3, identifier + authority).
// Only the fields the ORM/Patient converters read are modelled.
type CX struct {
	ID                 string // CX-1
	CheckDigit         string // CX-2
	AssigningAuthority HD     // CX-4
	IdentifierTypeCode string // CX-5, e.g. "MR" medical record
}

// parseCX reads a CX from one repetition of a field. CX-4 (assigning authority)
// is itself an HD whose subcomponents are the namespace/universal-ID parts.
func parseCX(r Repetition) CX {
	return CX{
		ID:                 r.component(1),
		CheckDigit:         r.component(2),
		AssigningAuthority: parseHDFromComponent(r, 4),
		IdentifierTypeCode: r.component(5),
	}
}

// parseHDFromComponent reads an HD from the subcomponents of the n-th 1-based
// component of r, which is how a nested HD (e.g. CX-4) is encoded.
func parseHDFromComponent(r Repetition, n int) HD {
	if n < 1 || n > len(r.Components) {
		return HD{}
	}
	subs := r.Components[n-1].Subcomponents
	return HD{
		NamespaceID:     subcomponent(subs, 1),
		UniversalID:     subcomponent(subs, 2),
		UniversalIDType: subcomponent(subs, 3),
	}
}

// CWE — coded with exceptions (supersedes the retired CE). Used for OBR-4 and
// other coded values the converters map to CodeableConcept. The alternate
// triplet (CWE-4/5/6) mirrors the primary one for a second coding system.
type CWE struct {
	Code            string // CWE-1
	Text            string // CWE-2
	CodingSystem    string // CWE-3
	AltCode         string // CWE-4
	AltText         string // CWE-5
	AltCodingSystem string // CWE-6
}

func parseCWE(r Repetition) CWE {
	return CWE{
		Code:            r.component(1),
		Text:            r.component(2),
		CodingSystem:    r.component(3),
		AltCode:         r.component(4),
		AltText:         r.component(5),
		AltCodingSystem: r.component(6),
	}
}

// XPN — extended person name (PID-5). Only the components the Patient converter
// reads are modelled.
type XPN struct {
	Family       string // XPN-1
	Given        string // XPN-2
	Middle       string // XPN-3 (second/further given names)
	Suffix       string // XPN-4
	Prefix       string // XPN-5
	Degree       string // XPN-6
	NameTypeCode string // XPN-7, e.g. "L" legal
}

func parseXPN(r Repetition) XPN {
	return XPN{
		Family:       r.component(1),
		Given:        r.component(2),
		Middle:       r.component(3),
		Suffix:       r.component(4),
		Prefix:       r.component(5),
		Degree:       r.component(6),
		NameTypeCode: r.component(7),
	}
}

// XAD — extended address (PID-11, ...). Only the modelled postal components are
// read; HL7's later XAD components (address type, geo coordinates, ...) are not.
type XAD struct {
	Street           string // XAD-1
	OtherDesignation string // XAD-2
	City             string // XAD-3
	State            string // XAD-4
	Zip              string // XAD-5
	Country          string // XAD-6
}

func parseXAD(r Repetition) XAD {
	return XAD{
		Street:           r.component(1),
		OtherDesignation: r.component(2),
		City:             r.component(3),
		State:            r.component(4),
		Zip:              r.component(5),
		Country:          r.component(6),
	}
}

// Precision records how much of an HL7 timestamp was supplied, so a DTM never
// fabricates the components the sender omitted (the same lexical-preserving
// philosophy as dicom.DA).
type Precision uint8

const (
	PrecisionYear Precision = iota
	PrecisionMonth
	PrecisionDay
	PrecisionHour
	PrecisionMinute
	PrecisionSecond
	PrecisionFraction
)

// DTM — variable-precision HL7 timestamp. Precision is preserved exactly: a
// value of "2026" is year precision, "202605" is month precision. Parsing does
// not zero-fill, and String re-emits at the original precision.
type DTM struct {
	lexical   string // preserved source form (sans timezone handling in M2)
	t         time.Time
	precision Precision
	valid     bool // false for an empty/absent value
}

// ParseDTM parses an HL7 v2 timestamp of the form
// YYYY[MM[DD[HH[MM[SS[.S+]]]]]][+/-ZZZZ]. An empty string is a valid absent
// value (zero DTM). A timezone offset suffix is permitted and preserved in the
// lexical form; the M2 ORM slice resolves the time to UTC at the supplied
// precision and does not apply the offset.
func ParseDTM(s string) (DTM, error) {
	if s == "" {
		return DTM{}, nil
	}

	// Strip an optional timezone offset ('+HHMM' / '-HHMM'); HL7 permits it on
	// any DTM. It is preserved in the lexical form and not applied in M2, but it
	// must be well-formed — a malformed zone such as "+ABCD" or a bare "+" is
	// rejected rather than silently dropped (the lexical-fidelity guard).
	body := s
	if tz := indexTZSign(s); tz >= 0 {
		zone := s[tz+1:]
		if len(zone) != 4 || !allDigitsDTM(zone) {
			return DTM{}, fmt.Errorf("hl7v2: malformed DTM timezone offset of length %d", len(zone))
		}
		body = s[:tz]
	}

	// Split off a fractional tail at '.'; M2 resolves down to seconds and
	// preserves the lexical form for the fraction.
	digits := body
	fraction := ""
	hasDot := false
	if dot := indexByteStr(body, '.'); dot >= 0 {
		hasDot = true
		digits = body[:dot]
		fraction = body[dot+1:]
	}

	prec, ok := dtmPrecision(len(digits))
	if !ok || !allDigitsDTM(digits) {
		return DTM{}, fmt.Errorf("hl7v2: malformed DTM of length %d", len(s))
	}

	t, err := dtmTime(digits, prec)
	if err != nil {
		return DTM{}, err
	}

	// A fractional tail is only meaningful at seconds precision and must carry at
	// least one digit; "20260531123045." and "...45.abc" are malformed, not
	// silently accepted (the lexical-fidelity guard, PRD §9.2).
	if hasDot {
		if prec != PrecisionSecond || !allDigitsDTM(fraction) {
			return DTM{}, fmt.Errorf("hl7v2: malformed DTM fractional second of length %d", len(s))
		}
		prec = PrecisionFraction
	}

	return DTM{lexical: s, t: t, precision: prec, valid: true}, nil
}

// indexTZSign returns the index of a timezone-offset sign ('+' or '-') in s, or
// -1. The sign can only follow the digit/fraction body, never lead it, so a
// scan from offset 1 onward identifies it unambiguously.
func indexTZSign(s string) int {
	for i := 1; i < len(s); i++ {
		if s[i] == '+' || s[i] == '-' {
			return i
		}
	}
	return -1
}

// String re-emits the DTM at its original lexical form.
func (d DTM) String() string { return d.lexical }

// Time returns the resolved time, the precision to which it is real, and whether
// the DTM carried a value at all.
func (d DTM) Time() (time.Time, Precision, bool) { return d.t, d.precision, d.valid }

// Precision reports how precise the parsed timestamp is.
func (d DTM) Precision() Precision { return d.precision }

// IsZero reports whether the DTM is an absent value.
func (d DTM) IsZero() bool { return !d.valid }

// dtmPrecision maps a digit count to its HL7 timestamp precision.
func dtmPrecision(n int) (Precision, bool) {
	switch n {
	case 4:
		return PrecisionYear, true
	case 6:
		return PrecisionMonth, true
	case 8:
		return PrecisionDay, true
	case 10:
		return PrecisionHour, true
	case 12:
		return PrecisionMinute, true
	case 14:
		return PrecisionSecond, true
	default:
		return 0, false
	}
}

// dtmTime resolves the digit string to a time.Time at the given precision,
// filling lower-order components with their minimum (the floor of the interval),
// without claiming that precision in the returned Precision.
func dtmTime(digits string, prec Precision) (time.Time, error) {
	year := atoiDTM(digits[0:4])
	month, day := 1, 1
	hour, minute, second := 0, 0, 0
	if prec >= PrecisionMonth {
		month = atoiDTM(digits[4:6])
	}
	if prec >= PrecisionDay {
		day = atoiDTM(digits[6:8])
	}
	if prec >= PrecisionHour {
		hour = atoiDTM(digits[8:10])
	}
	if prec >= PrecisionMinute {
		minute = atoiDTM(digits[10:12])
	}
	if prec >= PrecisionSecond {
		second = atoiDTM(digits[12:14])
	}
	// A second value of 60 is a leap second in HL7, but time.Date cannot
	// represent it and normalises it to the next minute, silently shifting the
	// instant; reject it rather than corrupt a clinical timestamp.
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, fmt.Errorf("hl7v2: DTM has an out-of-range calendar/clock component")
	}
	t := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	// time.Date normalises an impossible day (e.g. Feb 31 -> Mar 3); reject any
	// component that did not survive the round-trip rather than silently shift a
	// clinical timestamp to a different instant.
	if t.Year() != year || int(t.Month()) != month || t.Day() != day ||
		t.Hour() != hour || t.Minute() != minute || t.Second() != second {
		return time.Time{}, fmt.Errorf("hl7v2: DTM is not a valid calendar date or time")
	}
	return t, nil
}

func allDigitsDTM(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

func atoiDTM(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// indexByteStr returns the index of the first occurrence of c in s, or -1.
func indexByteStr(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// subcomponent returns the n-th 1-based subcomponent of subs, or "".
func subcomponent(subs []string, n int) string {
	if n < 1 || n > len(subs) {
		return ""
	}
	return subs[n-1]
}
