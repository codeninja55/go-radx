package dicom

// This file implements the three DICOM datetime value representations from
// PS3.5 §6.2: DA (Date, YYYYMMDD), TM (Time, HHMMSS.FFFFFF), and DT (Date Time,
// YYYYMMDDHHMMSS.FFFFFF&ZZXX). Each type preserves its source lexical form so a
// value read from a file serialises back byte-identically (like pydicom's
// original_string), and each resolves to a Go time.Time where representable
// without fabricating absent fields.

// DateMode selects how the de-identification profile treats retained dates and
// times under the PS3.15 "Retain Longitudinal Temporal Information" sub-option.
// Defined here because the datetime layer owns the date semantics; Increment 7's
// profile consumes it.
type DateMode uint8

const (
	// DateModeKeep retains dates and times verbatim. It is the zero value so a
	// profile that opts in to temporal retention without naming a mode keeps the
	// source values unchanged.
	DateModeKeep DateMode = iota
	// DateModeShift moves every retained date and time by one consistent per-study
	// offset, preserving intervals while obscuring absolute dates.
	DateModeShift
)

// DateOption configures ParseDA. The default is strict YYYYMMDD; an option may
// relax it to accept the legacy partial-date forms.
type DateOption func(*dateConfig)

// dateConfig holds resolved date-parse options. The zero value is strict. There is
// no global mutable config; every knob is an explicit option (PRD §9.4).
type dateConfig struct {
	lenient bool
}

// withLenient relaxes ParseDA to accept the legacy YYYY and YYYYMM partial forms.
// It is unexported: callers reach it through WithLenientDates at the read layer,
// keeping a single public name for the behaviour.
func withLenient() DateOption {
	return func(c *dateConfig) { c.lenient = true }
}

// newDateConfig resolves opts over the strict default.
func newDateConfig(opts ...DateOption) dateConfig {
	var cfg dateConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
