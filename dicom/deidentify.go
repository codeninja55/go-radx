package dicom

import (
	"errors"
	"fmt"
)

// ErrBurnedInPixelData reports that a dataset declares burned-in identifying pixel
// data (BurnedInAnnotation (0028,0301) == "YES") that this profile does not clean.
// The profile is fail-closed: it returns this error rather than reporting a complete
// de-identification while leaving identifying text rendered into the pixels.
// A caller that has independently handled the pixels can opt out with
// WithAllowBurnedInPixelData; treating burned-in PHI as cleaned would be the most
// dangerous possible silent failure (Codex DCM-013).
var ErrBurnedInPixelData = errors.New("dicom: burned-in pixel PHI not cleaned")

// errNoUIDGenerator reports that UID remapping is enabled but no UIDGenerator was
// supplied, so the profile cannot mint replacement UIDs. It fails closed rather than
// leaving source UIDs in place (which would leak the identifying reference graph).
var errNoUIDGenerator = errors.New("dicom: de-identification requires a UIDGenerator for UID remapping (or WithRetainUIDs)")

// basicProfileCode is the PS3.15 Context Group 7050 code for the Basic Application
// Level Confidentiality Profile.
const basicProfileCode = "113100"

// codingSchemeDCM is the DICOM coding scheme designator used for de-identification
// method codes (PS3.16).
const codingSchemeDCM = "DCM"

// Profile applies the PS3.15 Annex E Basic Application Level Confidentiality Profile
// to a dataset. It is configured once through NewProfile and then reused; Deidentify
// is the single operation. A Profile holds no per-call mutable state: the UID remap
// is created fresh inside each Deidentify call so concurrent callers never share a
// map (PRD §9, no global mutable state).
type Profile struct {
	gen     *UIDGenerator
	options profileOptions
	// dummies overrides the default replacement value for a D-action attribute.
	dummies map[Tag]string
	// safePrivateCreators is the allow-list of private creators preserved when
	// WithRetainSafePrivate is set; nil means remove all private tags.
	safePrivateCreators map[string]bool
}

// profileOptions records the resolved PS3.15 sub-options. The zero value is the
// strictest de-identification: nothing retained, UIDs remapped, dates removed,
// private tags removed, burned-in pixel data fail-closed.
type profileOptions struct {
	retainPatientCharacteristics bool
	retainDeviceIdentity         bool
	retainUIDs                   bool
	retainTemporal               bool
	temporalMode                 DateMode
	retainSafePrivate            bool
	allowBurnedInPixelData       bool
}

// ProfileOption configures a Profile. Options are applied in order by NewProfile.
type ProfileOption func(*Profile)

// WithRetainPatientCharacteristics keeps the patient's general characteristics
// (age, sex, size, weight) under the PS3.15 "Retain Patient Characteristics"
// sub-option instead of removing them.
func WithRetainPatientCharacteristics() ProfileOption {
	return func(p *Profile) { p.options.retainPatientCharacteristics = true }
}

// WithRetainLongitudinalTemporalInformation opts in to the PS3.15 "Retain
// Longitudinal Temporal Information" sub-option: instead of the default removal or
// zeroing, dates and times are kept either verbatim (DateModeKeep) or shifted by one
// consistent per-run offset (DateModeShift), which preserves intervals while
// obscuring absolute dates. It is opt-in precisely because retaining temporal data
// weakens de-identification.
func WithRetainLongitudinalTemporalInformation(mode DateMode) ProfileOption {
	return func(p *Profile) {
		p.options.retainTemporal = true
		p.options.temporalMode = mode
	}
}

// WithRetainDeviceIdentity keeps device and institution identity (station name,
// device serial number, institution name) under the PS3.15 "Retain Device Identity"
// sub-option instead of removing them.
func WithRetainDeviceIdentity() ProfileOption {
	return func(p *Profile) { p.options.retainDeviceIdentity = true }
}

// WithRetainUIDs skips UID remapping, leaving Study/Series/SOP and referenced UIDs in
// place. This is off by default: retaining UIDs preserves the identifying reference
// graph and weakens de-identification, so it is an explicit opt-in (PS3.15 "Retain
// UIDs" sub-option).
func WithRetainUIDs() ProfileOption {
	return func(p *Profile) { p.options.retainUIDs = true }
}

// WithRetainSafePrivate preserves private attributes whose private creator is on the
// supplied allow-list, under the PS3.15 "Retain Safe Private" sub-option. Without it,
// the Basic Profile removes all private tags because go-radx implements no private
// SOP-class logic to judge which are safe (PRD §3.2). Creators are matched case-
// sensitively against the private-creator (gggg,00xx) string values.
func WithRetainSafePrivate(creators ...string) ProfileOption {
	return func(p *Profile) {
		p.options.retainSafePrivate = true
		if p.safePrivateCreators == nil {
			p.safePrivateCreators = make(map[string]bool, len(creators))
		}
		for _, c := range creators {
			p.safePrivateCreators[c] = true
		}
	}
}

// WithDummyValues overrides the replacement value used for the named D-action
// attributes (for example a study-specific pseudonym for PatientID). Tags not named
// here keep the profile's generic VR-appropriate dummy.
func WithDummyValues(replacements map[Tag]string) ProfileOption {
	return func(p *Profile) {
		if p.dummies == nil {
			p.dummies = make(map[Tag]string, len(replacements))
		}
		for t, v := range replacements {
			p.dummies[t] = v
		}
	}
}

// WithAllowBurnedInPixelData accepts the residual risk of burned-in identifying pixel
// data: Deidentify will not return ErrBurnedInPixelData even when BurnedInAnnotation
// is "YES". The caller asserts it has handled the pixels by other means. This profile
// never itself removes burned-in pixel text (v1 documented limit).
func WithAllowBurnedInPixelData() ProfileOption {
	return func(p *Profile) { p.options.allowBurnedInPixelData = true }
}

// NewProfile builds the Basic Profile with the given options. The supplied
// UIDGenerator mints replacement UIDs so the caller controls the organisation root;
// a nil generator is permitted only when WithRetainUIDs is set (no UIDs are minted).
func NewProfile(g *UIDGenerator, opts ...ProfileOption) *Profile {
	p := &Profile{gen: g}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Deidentify returns a de-identified deep copy of ds. The source is never mutated
// (Codex DCM-016). It walks every level, including sequence items, applies the Table
// E.1-1 action for each confidentiality attribute, remaps UIDs through a stable
// per-call map so the reference graph stays consistent, and sets the de-identification
// metadata: PatientIdentityRemoved (0012,0062) = "YES", DeidentificationMethod
// (0012,0063), and DeidentificationMethodCodeSequence (0012,0064).
//
// If the dataset declares burned-in identifying pixel data (BurnedInAnnotation ==
// "YES") and the caller has not opted out, Deidentify returns ErrBurnedInPixelData
// before doing any work: it never reports a complete PS3.15 de-identification while
// leaving identifying pixel text in place (Codex DCM-013).
func (p *Profile) Deidentify(ds *DataSet) (*DataSet, error) {
	if !p.options.retainUIDs && p.gen == nil {
		return nil, errNoUIDGenerator
	}
	if err := p.checkBurnedIn(ds); err != nil {
		return nil, err
	}

	out := ds.Clone()
	w := &deidWalk{
		profile:  p,
		uidRemap: make(map[UID]UID),
	}
	if err := w.resolveDateShift(out); err != nil {
		return nil, err
	}
	w.walk(out)
	p.setMetadata(out)
	return out, nil
}

// checkBurnedIn enforces the fail-closed burned-in pixel rule.
func (p *Profile) checkBurnedIn(ds *DataSet) error {
	if p.options.allowBurnedInPixelData {
		return nil
	}
	if v, ok := ds.GetString(TagBurnedInAnnotation); ok && v == "YES" {
		return fmt.Errorf("%w: BurnedInAnnotation %s == YES", ErrBurnedInPixelData, TagBurnedInAnnotation)
	}
	return nil
}

// setMetadata writes the PS3.15 de-identification metadata. PatientIdentityRemoved is
// always YES; the method text and code sequence record the Basic Profile plus any
// retained-option codes the caller selected.
func (p *Profile) setMetadata(ds *DataSet) {
	ds.SetString(TagPatientIdentityRemoved, "YES")
	ds.SetString(TagDeidentificationMethod, p.methodDescription())

	seq := NewSequence()
	for _, code := range p.methodCodes() {
		item := NewDataSet()
		item.SetString(TagCodeValue, code.value)
		item.SetString(TagCodingSchemeDesignator, codingSchemeDCM)
		item.SetString(TagCodeMeaning, code.meaning)
		seq.Append(item)
	}
	ds.Set(Element{Tag: TagDeidentificationMethodCodeSequence, VR: VRSQ, Value: NewSequenceValue(seq)})
}

// deidMethodCode is one (code, meaning) pair for the method code sequence.
type deidMethodCode struct {
	value   string
	meaning string
}

// methodCodes returns the Basic Profile code plus a code for each retained sub-option
// the caller selected (PS3.15 Context Group 7050).
func (p *Profile) methodCodes() []deidMethodCode {
	codes := []deidMethodCode{
		{basicProfileCode, "Basic Application Confidentiality Profile"},
	}
	if p.options.retainUIDs {
		codes = append(codes, deidMethodCode{"113110", "Retain UIDs Option"})
	}
	if p.options.retainTemporal {
		if p.options.temporalMode == DateModeShift {
			codes = append(codes, deidMethodCode{"113107", "Retain Longitudinal Temporal Information Modified Dates Option"})
		} else {
			codes = append(codes, deidMethodCode{"113106", "Retain Longitudinal Temporal Information Full Dates Option"})
		}
	}
	if p.options.retainPatientCharacteristics {
		codes = append(codes, deidMethodCode{"113108", "Retain Patient Characteristics Option"})
	}
	if p.options.retainDeviceIdentity {
		codes = append(codes, deidMethodCode{"113109", "Retain Device Identity Option"})
	}
	if p.options.retainSafePrivate {
		codes = append(codes, deidMethodCode{"113111", "Retain Safe Private Option"})
	}
	return codes
}

// methodDescription is the human-readable DeidentificationMethod (0012,0063) text. It
// names the profile and any retained options but never the removed values (PRD §8.2).
func (p *Profile) methodDescription() string {
	desc := "go-radx PS3.15 Basic Application Level Confidentiality Profile"
	for _, c := range p.methodCodes()[1:] {
		desc += "; " + c.meaning
	}
	return desc
}
