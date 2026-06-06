package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// schemeDesignatorFor is the inverse of schemeDesignatorSystem: it maps a FHIR
// system URI back to its DICOM coding-scheme designator so a CodeableConcept can be
// re-encoded as a DICOM coded entry without losing the scheme. A system carried under
// the synthetic urn:dicom:scheme: prefix is unwrapped to the verbatim designator it
// preserved on the forward path; an unrecognised system yields "" so the caller leaves
// the designator empty rather than fabricating one.
func schemeDesignatorFor(system string) string {
	switch system {
	case "http://dicom.nema.org/resources/ontology/DCM":
		return "DCM"
	case "http://snomed.info/sct":
		return "SCT"
	case "http://loinc.org":
		return "LN"
	case ucumSystem:
		return "UCUM"
	}
	const dicomSchemePrefix = "urn:dicom:scheme:"
	if len(system) > len(dicomSchemePrefix) && system[:len(dicomSchemePrefix)] == dicomSchemePrefix {
		return system[len(dicomSchemePrefix):]
	}
	return ""
}

// conceptNameFor is the inverse of conceptNameCode: it maps the first Coding of a FHIR
// CodeableConcept back to a DICOM ConceptNameCode triplet (code, scheme designator,
// meaning). The Coding.system maps back to its scheme designator via schemeDesignatorFor;
// the Coding.display, or the CodeableConcept.text when the Coding has none, becomes the
// code meaning. A concept with no Coding yields the zero ConceptNameCode.
func conceptNameFor(cc *r5.CodeableConcept) dicom.ConceptNameCode {
	if cc == nil || len(cc.Coding) == 0 {
		return dicom.ConceptNameCode{}
	}
	coding := cc.Coding[0]
	out := dicom.ConceptNameCode{}
	if coding.Code != nil {
		out.CodeValue = *coding.Code
	}
	if coding.System != nil {
		out.CodingSchemeDesignator = schemeDesignatorFor(*coding.System)
	}
	switch {
	case coding.Display != nil:
		out.CodeMeaning = *coding.Display
	case cc.Text != nil:
		out.CodeMeaning = *cc.Text
	}
	return out
}

// fhirDateTimeToDICOM converts a FHIR dateTime lexical form to a DICOM DT lexical form:
// it strips the "-" date separators and ":" time separators, joins the date and time on
// no separator, and rewrites a "+hh:mm"/"-hh:mm"/"Z" timezone suffix as the DICOM
// "&ZZXX" form ("+0000" for "Z"). A date-only value yields the YYYYMMDD form. The
// fractional second is carried verbatim. ok is false for a value that is not at least a
// full calendar date, so a partial-precision FHIR value is never widened into a
// fabricated DICOM date.
func fhirDateTimeToDICOM(value string) (string, bool) {
	date, rest, ok := splitFHIRDate(value)
	if !ok {
		return "", false
	}
	if rest == "" {
		return date, true
	}
	timePart, offset := splitFHIRTimeOffset(rest)
	return date + stripColons(timePart) + dicomOffset(offset), true
}

// splitFHIRDate splits a FHIR dateTime into its YYYYMMDD date head and the remainder
// after the "T". ok is false unless the value carries a full year-month-day date, so a
// year- or month-only FHIR value is rejected rather than padded.
func splitFHIRDate(value string) (date, rest string, ok bool) {
	head := value
	if t := indexByte(value, 'T'); t >= 0 {
		head, rest = value[:t], value[t+1:]
	}
	if len(head) != len("YYYY-MM-DD") || head[4] != '-' || head[7] != '-' {
		return "", "", false
	}
	return head[0:4] + head[5:7] + head[8:10], rest, true
}

// splitFHIRTimeOffset splits the post-"T" remainder into its time-of-day and its
// trailing timezone suffix ("Z", "+hh:mm", or "-hh:mm"). A remainder with no suffix
// yields an empty offset.
func splitFHIRTimeOffset(rest string) (timePart, offset string) {
	if rest == "" {
		return "", ""
	}
	if rest[len(rest)-1] == 'Z' {
		return rest[:len(rest)-1], "Z"
	}
	// The sign of an offset can only follow the time body, never lead it, and the
	// fractional dot precedes it; scan from the right for the first sign.
	for i := len(rest) - 1; i > 0; i-- {
		if rest[i] == '+' || rest[i] == '-' {
			return rest[:i], rest[i:]
		}
	}
	return rest, ""
}

// dicomOffset rewrites a FHIR timezone suffix as the DICOM "&ZZXX" offset: "Z" becomes
// "+0000", "+hh:mm"/"-hh:mm" lose their colon. An empty suffix yields "".
func dicomOffset(offset string) string {
	switch offset {
	case "":
		return ""
	case "Z":
		return "+0000"
	default:
		return stripColons(offset)
	}
}

// stripColons removes every ":" from s, used to turn FHIR "hh:mm:ss" / "+hh:mm" into
// the DICOM colon-free form.
func stripColons(s string) string {
	if indexByte(s, ':') < 0 {
		return s
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != ':' {
			out = append(out, s[i])
		}
	}
	return string(out)
}

// indexByte returns the first index of b in s, or -1.
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
