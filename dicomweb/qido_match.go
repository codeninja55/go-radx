package dicomweb

import (
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// matchKeyTypes are the VRs whose values support a date/time range match ("lo-hi", PS3.18
// §10.6.1, drawing on PS3.4 C.2.2.2.5): DA (date), TM (time), and DT (date-time).
//
// matchDataSet reports whether ds satisfies every matching key in q. An attribute the
// dataset does not carry fails the match unless the key value is universal (empty or a
// bare "*"), which matches everything including absent attributes (PS3.4 C.2.2.2.4). A
// query with no keys matches every candidate.
func matchDataSet(ds *dicom.DataSet, q QueryRequest) bool {
	for _, mk := range q.Match {
		if !matchKey(ds, mk, q.Fuzzy) {
			return false
		}
	}
	return true
}

// matchKey reports whether one matching key is satisfied by the dataset. The match is
// dispatched by VR: DA/TM/DT support range matching, UI matches against a backslash list
// of UIDs, PN supports fuzzy matching when requested, and the remaining string VRs use
// single-value or wildcard matching. A universal key value matches everything.
func matchKey(ds *dicom.DataSet, mk MatchKey, fuzzy bool) bool {
	if isUniversalMatch(mk.Value) {
		return true
	}
	vals, ok := ds.GetStrings(mk.Tag)
	if !ok || len(vals) == 0 {
		// A present, non-universal key against an absent attribute never matches: the
		// candidate lacks the value the caller constrained on (PS3.4 C.2.2.2.4).
		return false
	}
	switch mk.VR {
	case dicom.VRDA, dicom.VRTM, dicom.VRDT:
		return matchAnyValue(vals, func(v string) bool { return matchRange(v, mk.Value) })
	case dicom.VRUI:
		return matchUIDList(vals, mk.Value)
	case dicom.VRPN:
		return matchAnyValue(vals, func(v string) bool { return matchPersonName(v, mk.Value, fuzzy) })
	default:
		return matchAnyValue(vals, func(v string) bool { return matchString(v, mk.Value) })
	}
}

// isUniversalMatch reports whether a key value matches every candidate: an empty value or
// a bare "*" is universal matching (PS3.4 C.2.2.2.4), so it is never used to exclude a
// candidate.
func isUniversalMatch(value string) bool {
	return value == "" || value == "*"
}

// matchAnyValue reports whether any of a multi-valued attribute's values satisfies pred.
// A multi-valued attribute matches if any single value matches (PS3.4 C.2.2.2.1).
func matchAnyValue(vals []string, pred func(string) bool) bool {
	for _, v := range vals {
		if pred(v) {
			return true
		}
	}
	return false
}

// matchUIDList reports whether candidate is one of the UIDs in a backslash-separated list,
// the UID-list matching of PS3.4 C.2.2.2.2. The comparison is exact: UIDs are
// case-sensitive and have no wildcards.
func matchUIDList(candidates []string, list string) bool {
	wanted := strings.Split(list, "\\")
	for _, w := range wanted {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if matchAnyValue(candidates, func(v string) bool { return v == w }) {
			return true
		}
	}
	return false
}

// matchString reports whether candidate satisfies a single-value or wildcard match. A
// pattern containing "*" or "?" is wildcard matching (PS3.4 C.2.2.2.4): "*" matches zero
// or more characters, "?" exactly one. Without a wildcard the match is exact and
// case-sensitive (single-value matching, PS3.4 C.2.2.2.1).
func matchString(candidate, pattern string) bool {
	if !strings.ContainsAny(pattern, "*?") {
		return candidate == pattern
	}
	return wildcardMatch(candidate, pattern)
}

// matchPersonName reports whether a PN candidate satisfies the pattern. Without fuzzy
// matching it is the same wildcard/single-value rule as any string VR. With fuzzy matching
// requested the comparison is case-insensitive and ignores the component-group structure,
// so "smith" matches "Smith^John" (an approximation of PS3.4 C.2.2.2.4 fuzzy semantics,
// not a phonetic algorithm).
func matchPersonName(candidate, pattern string, fuzzy bool) bool {
	if !fuzzy {
		return matchString(candidate, pattern)
	}
	c := strings.ToLower(normalizePersonName(candidate))
	p := strings.ToLower(normalizePersonName(strings.Trim(pattern, "*")))
	if p == "" {
		return true
	}
	return strings.Contains(c, p)
}

// normalizePersonName flattens a PN value's component-group delimiters (^ and =) to spaces
// and collapses runs of whitespace, so fuzzy matching compares name tokens rather than the
// exact delimiter layout.
func normalizePersonName(s string) string {
	s = strings.NewReplacer("^", " ", "=", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// matchRange reports whether candidate falls within a range query against a DA/TM/DT
// value. The range form is "lo-hi", "lo-" (open upper bound), or "-hi" (open lower bound);
// a value with no hyphen is single-value matching. DICOM DA/TM/DT canonical strings sort
// lexicographically in chronological order, so the bounds are compared as strings after
// the candidate is normalised to the same width. A malformed range never matches rather
// than matching everything (fail closed, PRD §9.2).
func matchRange(candidate, pattern string) bool {
	lo, hi, isRange := splitRange(pattern)
	if !isRange {
		return matchString(candidate, pattern)
	}
	c := normalizeTemporal(candidate)
	if lo != "" && c < padLower(normalizeTemporal(lo), len(c)) {
		return false
	}
	if hi != "" && c > padUpper(normalizeTemporal(hi), len(c)) {
		return false
	}
	// A range with both bounds empty ("-") is malformed: reject it rather than match all.
	return lo != "" || hi != ""
}

// splitRange splits a DA/TM/DT range value "lo-hi" into its bounds. It reports isRange
// false for a value that carries no hyphen (a single-value match). A DT value may itself
// contain a "+"/"-" timezone suffix, but PS3.18 range queries on DT are out of this
// slice's scope for timezone-suffixed bounds; a single hyphen is treated as the range
// separator.
func splitRange(pattern string) (lo, hi string, isRange bool) {
	i := strings.IndexByte(pattern, '-')
	if i < 0 {
		return "", "", false
	}
	return pattern[:i], pattern[i+1:], true
}

// normalizeTemporal strips the separators a DA/TM/DT value may carry (DICOM canonical
// forms omit them, but a query value may include them) so two values compare on the same
// digit string.
func normalizeTemporal(s string) string {
	return strings.NewReplacer("-", "", ":", "", "/", "", " ", "").Replace(s)
}

// padLower right-pads a lower-bound string with '0' to width, so a coarse bound ("2020"
// against a "20200115" candidate) compares as the earliest matching instant.
func padLower(s string, width int) string {
	for len(s) < width {
		s += "0"
	}
	return s
}

// padUpper right-pads an upper-bound string with '9' to width, so a coarse bound ("2020"
// against a "20200115" candidate) compares as the latest matching instant.
func padUpper(s string, width int) string {
	for len(s) < width {
		s += "9"
	}
	return s
}

// wildcardMatch reports whether s matches a DICOM wildcard pattern where "*" matches zero
// or more characters and "?" matches exactly one (PS3.4 C.2.2.2.4). It is an iterative
// two-pointer matcher with backtracking on "*", bounded by the input lengths so a hostile
// pattern cannot drive super-linear blow-up.
func wildcardMatch(s, pattern string) bool {
	si, pi := 0, 0
	star, mark := -1, 0
	for si < len(s) {
		switch {
		case pi < len(pattern) && (pattern[pi] == '?' || pattern[pi] == s[si]):
			si++
			pi++
		case pi < len(pattern) && pattern[pi] == '*':
			star = pi
			mark = si
			pi++
		case star >= 0:
			pi = star + 1
			mark++
			si = mark
		default:
			return false
		}
	}
	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}
	return pi == len(pattern)
}
