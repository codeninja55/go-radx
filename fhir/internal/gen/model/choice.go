package model

import "strings"

// choiceSuffix is the FHIR marker on a polymorphic element's path ("value[x]").
const choiceSuffix = "[x]"

// isChoicePath reports whether a path's final segment is a polymorphic "[x]"
// element.
func isChoicePath(path string) bool {
	return strings.HasSuffix(path, choiceSuffix)
}

// stripChoiceSuffix removes the trailing "[x]" from a choice element's name or
// path, returning the base ("value" from "value[x]"). It is a no-op on a
// non-choice name.
func stripChoiceSuffix(s string) string {
	return strings.TrimSuffix(s, choiceSuffix)
}

// systemPrimitivePrefix is the FHIRPath System type namespace FHIR uses for the
// hidden primitive value of Element.id and similar fields
// ("http://hl7.org/fhirpath/System.String").
const systemPrimitivePrefix = "http://hl7.org/fhirpath/System."

// SystemPrimitive returns the FHIR primitive a FHIRPath System type code resolves
// to (System.String -> "string") and whether the code is such a system type. FHIR
// encodes the value of certain fields (Element.id) as a FHIRPath System type
// rather than a FHIR primitive; the model normalises the code so the planner sees
// a uniform primitive name. The returned name is lower-cased to match FHIR
// primitive type names.
func SystemPrimitive(code string) (string, bool) {
	if !strings.HasPrefix(code, systemPrimitivePrefix) {
		return "", false
	}
	name := strings.TrimPrefix(code, systemPrimitivePrefix)
	if name == "" {
		return "", false
	}
	return strings.ToLower(name[:1]) + name[1:], true
}
