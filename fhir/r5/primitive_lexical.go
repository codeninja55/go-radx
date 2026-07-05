// This file is HAND-WRITTEN, not generated. The generated validation descriptors
// (validation_descriptors.go) name which fields are date/dateTime/time/instant and call
// the lexical validators below; the lexical rules themselves are FHIR prose-and-regex
// constraints pinned to a release, so — like the Bundle bdl-* checks — they live in a
// hand-written per-release file the generator never rewrites. The extension-url walk at
// the bottom is here for the same reason: Extension.url's 1..1 cardinality is uniform,
// but the walk is release-typed.

package r5

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The patterns below are the official FHIR R5 primitive regexes, quoted verbatim from
// the R5 datatypes page (hl7.org/fhir/R5/datatypes.html, 5.0.0) and anchored:
//
//	date:     ([0-9]([0-9]([0-9][1-9]|[1-9]0)|[1-9]00)|[1-9]000)(-(0[1-9]|1[0-2])(-(0[1-9]|[1-2][0-9]|3[0-1]))?)?
//	dateTime: ([0-9]([0-9]([0-9][1-9]|[1-9]0)|[1-9]00)|[1-9]000)(-(0[1-9]|1[0-2])(-(0[1-9]|[1-2][0-9]|3[0-1])(T([01][0-9]|2[0-3]):[0-5][0-9]:([0-5][0-9]|60)(\.[0-9]{1,9})?)?)?(Z|(\+|-)((0[0-9]|1[0-3]):[0-5][0-9]|14:00)?)?)?
//	time:     ([01][0-9]|2[0-3]):[0-5][0-9]:([0-5][0-9]|60)(\.[0-9]{1,9})?
//	instant:  ([0-9]([0-9]([0-9][1-9]|[1-9]0)|[1-9]00)|[1-9]000)-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])T([01][0-9]|2[0-3]):[0-5][0-9]:([0-5][0-9]|60)(\.[0-9]{1,9})?(Z|(\+|-)((0[0-9]|1[0-3]):[0-5][0-9]|14:00))
//
// One deliberate correction: the published R5 dateTime pattern ends its offset group
// with "|14:00)?)?", whose inner "?" would accept a bare "+" or "-" sign with no
// offset digits. R4's published pattern and the R5 prose show the intended group is
// "(Z|(\+|-)((0[0-9]|1[0-3]):[0-5][0-9]|14:00))?", which is what dateTimeLexicalRE
// compiles — the erratum's garbage forms are rejected, nothing valid is.
//
// Two prose rules the regexes alone do not carry are enforced alongside them:
// "If hours and minutes are specified, a timezone offset SHALL be populated"
// (datatypes.html#dateTime; the R5 pattern structurally allows a zone-less time), and
// "Dates SHALL be valid dates" (a regex-shaped 2026-02-30 is still not a date).
var (
	dateLexicalRE = regexp.MustCompile(`^([0-9]([0-9]([0-9][1-9]|[1-9]0)|[1-9]00)|[1-9]000)(-(0[1-9]|1[0-2])(-(0[1-9]|[1-2][0-9]|3[0-1]))?)?$`)

	dateTimeLexicalRE = regexp.MustCompile(`^([0-9]([0-9]([0-9][1-9]|[1-9]0)|[1-9]00)|[1-9]000)(-(0[1-9]|1[0-2])(-(0[1-9]|[1-2][0-9]|3[0-1])(T([01][0-9]|2[0-3]):[0-5][0-9]:([0-5][0-9]|60)(\.[0-9]{1,9})?)?)?(Z|(\+|-)((0[0-9]|1[0-3]):[0-5][0-9]|14:00))?)?$`)

	timeLexicalRE = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]:([0-5][0-9]|60)(\.[0-9]{1,9})?$`)

	instantLexicalRE = regexp.MustCompile(`^([0-9]([0-9]([0-9][1-9]|[1-9]0)|[1-9]00)|[1-9]000)-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])T([01][0-9]|2[0-3]):[0-5][0-9]:([0-5][0-9]|60)(\.[0-9]{1,9})?(Z|(\+|-)((0[0-9]|1[0-3]):[0-5][0-9]|14:00))$`)

	// dateTimeOffsetSuffixRE recognises a populated timezone offset at the end of a
	// dateTime, the SHALL-companion to the time-of-day part.
	dateTimeOffsetSuffixRE = regexp.MustCompile(`(Z|[+-][0-9][0-9]:[0-9][0-9])$`)
)

// validDateLexical reports whether s is a lexically valid FHIR R5 date: the official
// regex plus the valid-calendar-date rule for a full YYYY-MM-DD value.
func validDateLexical(s string) bool {
	return dateLexicalRE.MatchString(s) && validCalendarDay(s)
}

// validDateTimeLexical reports whether s is a lexically valid FHIR R5 dateTime: the
// official regex, the offset rule in both directions, and the valid-calendar-date rule.
// The offset rule is a biconditional the published R5 pattern does not carry because its
// offset group sits outside the T-time group: a value with a time-of-day SHALL carry a
// populated offset (datatypes.html#dateTime), and — the converse — a date-only value
// (no 'T') SHALL NOT carry an offset. R4's pattern nests the offset inside the T-group
// and so rejects "2026-05Z" and "2026-01+02:00" outright; enforcing the converse here
// makes R5 agree with R4 rather than silently accepting an offset on a bare date.
func validDateTimeLexical(s string) bool {
	if !dateTimeLexicalRE.MatchString(s) {
		return false
	}
	hasTime := strings.ContainsRune(s, 'T')
	hasOffset := dateTimeOffsetSuffixRE.MatchString(s)
	if hasTime != hasOffset {
		return false
	}
	return validCalendarDay(s)
}

// validTimeLexical reports whether s is a lexically valid FHIR R5 time. The official
// regex allows a leap second (:60) and forbids both "24:00:00" and any timezone.
func validTimeLexical(s string) bool {
	return timeLexicalRE.MatchString(s)
}

// validInstantLexical reports whether s is a lexically valid FHIR R5 instant: at least
// second precision with a mandatory timezone offset (the official regex carries both),
// plus the valid-calendar-date rule.
func validInstantLexical(s string) bool {
	return instantLexicalRE.MatchString(s) && validCalendarDay(s)
}

// validCalendarDay enforces "Dates SHALL be valid dates" for a value that carries a
// full YYYY-MM-DD prefix: the day must exist in its month (the regex alone admits a
// February 30th). A year- or year-month-precision value has no day to check and
// passes. The caller has already regex-matched s, so the prefix shape is trusted here.
func validCalendarDay(s string) bool {
	if len(s) < 10 || s[4] != '-' || s[7] != '-' {
		return true
	}
	_, err := time.Parse("2006-01-02", s[:10])
	return err == nil
}

// missingExtensionURLs reports the element path of every extension under path —
// including extensions nested inside extensions — whose required url (Extension.url,
// 1..1) is absent. It is called by the generated Required closures over a resource's
// extension and modifierExtension arrays, so a wire extension that dropped its url is
// a required-issue exactly like any other missing 1..1 element. Paths carry indices
// only, never values (PRD §9.1).
func missingExtensionURLs(path string, exts []Extension) []string {
	var missing []string
	for i := range exts {
		p := path + "[" + strconv.Itoa(i) + "]"
		if exts[i].URL == nil {
			missing = append(missing, p+".url")
		}
		missing = append(missing, missingExtensionURLs(p+".extension", exts[i].Extension)...)
	}
	return missing
}
