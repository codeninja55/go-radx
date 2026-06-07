// Package plan is the generator's planning stage. It consumes the release-agnostic
// model/IR (an occurrence-path element tree) and decides every Go-shape question the
// emitter then renders mechanically: the Go identifier for each FHIR name, the Go
// type for each element, whether an element is a pointer, a value, or a slice, and
// which nested backbone structures are distinct. The planner makes no I/O and reads
// no templates; its single responsibility is turning "what FHIR says" into "what Go
// to write", so the emitter never has to make a decision.
package plan

import (
	"strings"
	"unicode"
)

// commonInitialisms are the identifier fragments Go style renders in all caps. The
// planner upper-cases a whole segment that matches one of these so a FHIR "id"
// becomes "ID" and "url" becomes "URL", matching the surrounding hand-written code
// and golint's expectations. The list is the load-bearing subset for FHIR names; it
// is not the full golint table because most FHIR element names are ordinary words.
var commonInitialisms = map[string]string{
	"id":   "ID",
	"url":  "URL",
	"uri":  "URI",
	"uid":  "UID",
	"json": "JSON",
	"xml":  "XML",
	"http": "HTTP",
	"sop":  "SOP",
	"fhir": "FHIR",
}

// GoFieldName maps a FHIR element name to its exported Go struct-field identifier.
// The "[x]" choice suffix is stripped first (a choice group's base name is the field
// stem), then the remaining hyphen/underscore-separated words are title-cased and the
// initialism words upper-cased, so "value[x]" becomes "Value", "implicitRules"
// becomes "ImplicitRules", and "id" becomes "ID". The result is always exported and
// is never a bare Go keyword because title-casing lifts it out of the reserved set.
func GoFieldName(fhirName string) string {
	base := strings.TrimSuffix(fhirName, "[x]")
	return exportIdentifier(base)
}

// GoTypeName maps a FHIR type name to its Go type identifier. FHIR type names are
// already PascalCase ("CodeableConcept", "Reference", "Period"), so the mapping is
// largely identity; the function still normalises the casing and initialisms so a
// name that arrives lower-cased (a primitive type code reaching this path) is lifted
// to an exported identifier deterministically.
func GoTypeName(fhirType string) string {
	return exportIdentifier(fhirType)
}

// GoBackboneTypeName builds the Go type name for a nested backbone element by
// concatenating the owning type name with each path segment title-cased, so
// Observation.component.referenceRange becomes ObservationComponentReferenceRange.
// Concatenating the full occurrence path (rather than just the leaf) keeps backbone
// names unique across a resource and stable across regenerations.
func GoBackboneTypeName(ownerType string, pathSegments []string) string {
	var b strings.Builder
	b.WriteString(exportIdentifier(ownerType))
	for _, seg := range pathSegments {
		b.WriteString(exportIdentifier(strings.TrimSuffix(seg, "[x]")))
	}
	return b.String()
}

// exportIdentifier title-cases a possibly multi-word FHIR name into a single
// exported Go identifier, upper-casing any word that is a known initialism. A name
// is split on hyphens, underscores, and whitespace (FHIR element names use none of
// these today, but value-set code tokens carry hyphens and underscores, and some R4
// value-set names carry spaces such as "Medication Status Codes"); a name already in
// camelCase is treated as one word and only its first rune is upper-cased so the
// internal casing ("implicitRules" -> "ImplicitRules") is preserved.
func exportIdentifier(name string) string {
	if name == "" {
		return ""
	}
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || unicode.IsSpace(r)
	})
	var b strings.Builder
	for _, w := range words {
		if up, ok := commonInitialisms[strings.ToLower(w)]; ok {
			b.WriteString(up)
			continue
		}
		b.WriteString(upperFirst(w))
	}
	return b.String()
}

// upperFirst upper-cases the first rune of s and leaves the rest untouched, so an
// already-camelCased FHIR element name keeps its internal capitals.
func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// resolveCollision makes a candidate identifier unique within a set of names already
// used in the same scope. It appends an ascending numeric suffix (Name, Name2,
// Name3, ...) the first time a duplicate would occur, so collisions resolve the same
// way on every run regardless of map iteration order. The chosen name is recorded in
// used so a third clash continues the sequence. Determinism here is what keeps the
// generated output byte-stable: two FHIR names that map to the same Go identifier
// always produce the same disambiguated pair.
func resolveCollision(candidate string, used map[string]bool) string {
	if !used[candidate] {
		used[candidate] = true
		return candidate
	}
	for n := 2; ; n++ {
		alt := candidate + itoa(n)
		if !used[alt] {
			used[alt] = true
			return alt
		}
	}
}

// itoa renders a small positive int without importing strconv, keeping the planner's
// dependency surface to the standard string/unicode packages.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
