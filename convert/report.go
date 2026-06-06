package convert

import "errors"

// Sentinel errors are comparable with errors.Is. They are reserved for genuine
// failures — a malformed source, a missing required identifier the converter
// cannot synthesise, or a source whose message type or SOP class is out of
// scope. A clean conversion that merely dropped optional, unmappable attributes
// returns a nil error and a populated *Report instead (docs/reference/convert.md
// "The conversion report and the error model").
var (
	// ErrMissingIdentifier is returned when the source lacks an identifier the
	// converter cannot synthesise (for example a DICOM dataset with no Study
	// Instance UID).
	ErrMissingIdentifier = errors.New("convert: source lacks a required identifier")
	// ErrUnsupportedSource is returned when the source message type or SOP class
	// is out of the v1 scope (for example a multi-order ORM, which the
	// single-order-per-call converters reject fail-closed).
	ErrUnsupportedSource = errors.New("convert: source message type or SOP class is out of scope")
	// ErrMalformedSource is returned when the source fails structural validation.
	ErrMalformedSource = errors.New("convert: source failed structural validation")
)

// Report accompanies every successful conversion and records what could not be
// mapped cleanly. A non-nil *Report is returned even on a clean conversion (it is
// simply empty); callers inspect it to decide whether the loss is acceptable —
// the library provides the facts, the consumer owns the policy.
//
// The diagnostics name concepts, never raw patient values: a dropped DICOM
// attribute is rendered as its keyword plus (gggg,eeee), an HL7 locus as its
// SEG-Fn accessor, and a FHIR locus by element path. No Report string embeds PHI
// by default (docs/reference/convert.md; PRD §9.1).
type Report struct {
	// Dropped lists source data that had no target home.
	Dropped []DroppedField
	// Defaulted lists target elements that required a value the source did not
	// supply, and the default the converter chose.
	Defaulted []DefaultedField
	// Substituted lists target elements the converter populated with an
	// approximation of a source value it could not map exactly, so a consumer can
	// distinguish a lossy-but-present mapping from a dropped one: an unknown HL7
	// gender code rendered as the value-set-safe "unknown", a trigger event with
	// no exact FHIR status, an unrecognised coding-scheme designator carried under
	// a synthetic system URI.
	Substituted []Substitution
}

// DroppedField is one item of source data with no target home, named by concept.
type DroppedField struct {
	// Source is a human-readable source locus, e.g.
	// "DICOM (0008,1030) StudyDescription" or "OBX-17 ObservationMethod".
	Source string
	// Reason explains, in plain language, why the field was dropped.
	Reason string
}

// DefaultedField is one target element the converter populated with a default
// because the source did not supply it, named by FHIR element path.
type DefaultedField struct {
	// Target is the FHIR element path the default was applied to, e.g.
	// "ServiceRequest.intent" or "ImagingStudy.status".
	Target string
	// Value is the default value the converter chose, e.g. "order" or "available".
	Value string
	// Reason explains why the default was applied.
	Reason string
}

// Substitution is one target element the converter populated with an approximation
// of a source value it could not map exactly, named by concept. It names the FHIR
// concept and the approximation chosen, never the raw patient value the source
// carried (PRD §9.1).
type Substitution struct {
	// Concept is the FHIR element path the approximation was applied to, e.g.
	// "Patient.gender" or "Encounter.status".
	Concept string
	// Approximation is the value-set-safe value the converter chose, e.g.
	// "unknown" or "in-progress".
	Approximation string
	// Reason explains, in plain language, why the source value could not be mapped
	// exactly. It names the source concept, never the source value.
	Reason string
}

// dropped records a dropped source field on the report.
func (r *Report) dropped(source, reason string) {
	r.Dropped = append(r.Dropped, DroppedField{Source: source, Reason: reason})
}

// defaulted records a defaulted target element on the report.
func (r *Report) defaulted(target, value, reason string) {
	r.Defaulted = append(r.Defaulted, DefaultedField{Target: target, Value: value, Reason: reason})
}

// substituted records an approximate mapping on the report.
func (r *Report) substituted(concept, approximation, reason string) {
	r.Substituted = append(r.Substituted, Substitution{Concept: concept, Approximation: approximation, Reason: reason})
}

// LossError is returned in place of a Report.Dropped entry only when
// WithStrictLoss is set and a drop occurs. It escalates a lossy conversion to a
// returned error so a consumer that cannot accept loss fails closed. It is
// comparable with errors.As.
type LossError struct {
	Dropped []DroppedField
}

// Error renders the dropped fields without embedding any patient value.
func (e *LossError) Error() string {
	if len(e.Dropped) == 0 {
		return "convert: lossy conversion"
	}
	msg := "convert: lossy conversion dropped " + e.Dropped[0].Source
	if len(e.Dropped) > 1 {
		msg += " and other fields"
	}
	return msg
}
