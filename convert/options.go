package convert

import (
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// Option configures a conversion. The zero options mean: strict-loss off (drops
// are recorded on the Report, not escalated to an error) and minted UIDs use no
// configured org root. The options carry per-call configuration independent of
// the FHIR release, except WithSubjectR4 and WithSubjectR5, which are release-typed
// because a FHIR Reference is a release sub-package datatype.
type Option func(*config)

// config is the resolved per-call configuration. There is no global mutable
// state; every knob is a functional option (PRD §9.4).
type config struct {
	uidRoot    dicom.UID
	strictLoss bool
	subjectR4  *r4.Reference
	subjectR5  *r5.Reference
}

// newConfig resolves the options into a config.
func newConfig(opts ...Option) config {
	var c config
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// WithUIDRoot sets the org root for any UIDs a converter must mint (used by the
// FHIR→SR direction, deferred past M2). It is accepted by every converter so the
// option set is uniform.
func WithUIDRoot(root dicom.UID) Option {
	return func(c *config) { c.uidRoot = root }
}

// WithStrictLoss turns a lossy drop into a returned *LossError instead of a
// Report.Dropped entry. A consumer that cannot accept any silent loss passes this
// so the conversion fails closed (docs/reference/convert.md error model).
func WithStrictLoss() Option {
	return func(c *config) { c.strictLoss = true }
}

// WithSubjectR5 injects the Patient (or other subject) Reference the source
// cannot supply, for the R5 converters. Absent it, a converter leaves subject
// either unset (recording a Defaulted entry) or carrying the source's logical
// identity as a Reference.identifier — never a fabricated Reference.reference URL
// (the identity rule).
func WithSubjectR5(ref r5.Reference) Option {
	return func(c *config) {
		clone := ref
		c.subjectR5 = &clone
	}
}

// WithSubjectR4 injects the Patient (or other subject) Reference the source
// cannot supply, for the R4 converters. It is the R4 twin of WithSubjectR5,
// release-typed because a FHIR Reference is a release sub-package datatype.
// Absent it, a converter leaves subject either unset (recording a Defaulted
// entry) or carrying the source's logical identity as a Reference.identifier —
// never a fabricated Reference.reference URL (the identity rule).
func WithSubjectR4(ref r4.Reference) Option {
	return func(c *config) {
		clone := ref
		c.subjectR4 = &clone
	}
}

// finalize escalates a strict-loss report to a *LossError when any field was
// dropped, otherwise returns the report and a nil error. Converters call it on
// their way out so the strict-loss policy lives in one place.
func (c config) finalize(r *Report) (*Report, error) {
	if c.strictLoss && len(r.Dropped) > 0 {
		return r, &LossError{Dropped: r.Dropped}
	}
	return r, nil
}
