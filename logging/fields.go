package logging

import (
	"fmt"
	"regexp"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// The field helpers below render DICOM, HL7 v2, and FHIR concepts by their
// structural name only. Every parameter is a structural locator — a tag
// coordinate, a dictionary keyword, a segment id, a field index, an element path
// — and none accepts a data element's value. There is deliberately no helper that
// logs a value, so the API refuses raw patient values by construction.
//
// The string locators are additionally shape-guarded as defense-in-depth: each is
// validated against the narrow grammar its concept uses (an identifier-shaped
// keyword, a three-character segment id, a dotted element path). A patient value
// such as "DOE^JANE", a date, an MRN, or a raw "PID|..." segment does not match
// these grammars, so a caller that misroutes one into a locator slot gets
// redactedToken in the log, never the raw string.
//
// Shape validation cannot distinguish a single identifier-shaped token (a bare
// surname like "Smith") from a real keyword, because both occupy the same lexical
// space. Binding a locator to the canonical vocabulary — the DICOM data
// dictionary, the HL7 segment set — is the caller's responsibility at the domain
// package boundary, which owns those dictionaries; the dicom package, for example,
// resolves the keyword from the tag rather than letting an arbitrary string reach
// this helper. This package stays a dependency-free leaf so any domain package can
// import it, which precludes embedding those dictionaries here.

// redactedToken replaces a locator that fails structural validation. It is a
// fixed, value-free marker so a misrouted patient value can never reach the sink.
const redactedToken = "[redacted-non-structural]"

var (
	// dicomKeywordPattern matches PS3.6 dictionary keywords: a letter followed by
	// letters or digits (e.g. PatientName, StudyInstanceUID).
	dicomKeywordPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	// hl7SegmentPattern matches an HL7 v2 segment id: three characters, the first a
	// letter and the rest uppercase letters or digits (e.g. MSH, PID, OBX, ZX1).
	hl7SegmentPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{2}$`)
	// fhirPathPattern matches a dotted FHIR element path: identifier segments, each
	// a letter then letters or digits, with an optional [x] choice suffix
	// (e.g. Patient.name.family, Observation.value[x]).
	fhirPathPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(\[x\])?(\.[A-Za-z][A-Za-z0-9]*(\[x\])?)*$`)
)

func structural(value string, pattern *regexp.Regexp) string {
	if pattern.MatchString(value) {
		return value
	}
	return redactedToken
}

// DICOMTag renders a DICOM data element by its (group,element) coordinate and
// dictionary keyword, e.g. (0010,0010)/PatientName. It logs the element's
// identity, never its value; a keyword that is not identifier-shaped is redacted.
func DICOMTag(group, element uint16, keyword string) zap.Field {
	return zap.Object("dicom_tag", dicomTag{group: group, element: element, keyword: keyword})
}

type dicomTag struct {
	group, element uint16
	keyword        string
}

func (t dicomTag) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("tag", fmt.Sprintf("(%04X,%04X)", t.group, t.element))
	if t.keyword != "" {
		enc.AddString("keyword", structural(t.keyword, dicomKeywordPattern))
	}
	return nil
}

// HL7Field renders an HL7 v2 field locator: the segment id and the 1-based field
// position within it, e.g. PID-5 for the patient-name field. It identifies which
// field, never the field's content; a segment id outside the three-character
// grammar, or a field index below 1, is redacted.
func HL7Field(segment string, field int) zap.Field {
	return zap.Object("hl7_field", hl7Field{segment: segment, field: field})
}

type hl7Field struct {
	segment string
	field   int
}

func (f hl7Field) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	segment := structural(f.segment, hl7SegmentPattern)
	enc.AddString("segment", segment)
	if f.field < 1 {
		enc.AddString("locator", redactedToken)
		return nil
	}
	enc.AddInt("field", f.field)
	enc.AddString("locator", fmt.Sprintf("%s-%d", segment, f.field))
	return nil
}

// FHIRPath renders a FHIR element path, e.g. Patient.name.family. It identifies
// which element of a resource is in play, never the element's value; a path that
// is not a dotted chain of identifiers is redacted.
func FHIRPath(path string) zap.Field {
	return zap.String("fhir_path", structural(path, fhirPathPattern))
}
