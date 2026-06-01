package dicom

import (
	"fmt"
	"time"
)

// deidWalk carries the per-Deidentify-call state: the configured profile, the stable
// UID remap (same source UID -> same replacement within one call), and the resolved
// per-run date shift. It is created fresh for each call so nothing is shared across
// invocations (PRD §9, no global mutable state).
type deidWalk struct {
	profile  *Profile
	uidRemap map[UID]UID
	// dateShift is the per-run offset applied to retained dates/times under
	// DateModeShift. It is zero unless temporal retention with shifting is enabled.
	dateShift time.Duration
}

// resolveDateShift derives the single per-run date offset used when the caller opts
// in to DateModeShift. The shift is keyed off the StudyInstanceUID so the same study
// always receives the same offset within a call (and intervals between its dates are
// preserved), while two studies de-identified separately need not share an offset.
// The key never leaves the function; only the derived duration is retained.
func (w *deidWalk) resolveDateShift(ds *DataSet) error {
	o := w.profile.options
	if !o.retainTemporal || o.temporalMode != DateModeShift {
		return nil
	}
	key, _ := ds.GetString(TagStudyInstanceUID)
	w.dateShift = deriveDateShift(key)
	return nil
}

// deriveDateShift maps a study key to a deterministic offset in the range
// (-365, 365) days. A stable hash of the key seeds the offset so every date in the
// study shifts by the same amount, preserving intervals. The hash is not a security
// primitive; it only needs to be stable and well-spread.
func deriveDateShift(key string) time.Duration {
	var h uint64 = 1469598103934665603 // FNV-1a 64-bit offset basis
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= 1099511628211
	}
	days := int64(h%729) - 364 // -364 .. 364
	return time.Duration(days) * 24 * time.Hour
}

// walk applies the profile to every element of ds, recursing into sequence items so a
// confidentiality attribute is acted on at every nesting level (the prototype acted
// only at the top level, the DCM-013 defect).
func (w *deidWalk) walk(ds *DataSet) {
	// Collect tags first: the action may delete the element, and mutating the map
	// while iterating ds.All would be unsafe.
	tags := make([]Tag, 0, ds.Len())
	for e := range ds.All() {
		tags = append(tags, e.Tag)
	}
	for _, t := range tags {
		e, ok := ds.Get(t)
		if !ok {
			continue
		}
		w.apply(ds, e)
	}
}

// apply acts on a single element: it recurses into a sequence first (so nested
// confidentiality attributes are always handled), then applies the element's own
// Table E.1-1 action or the private-tag policy.
func (w *deidWalk) apply(ds *DataSet, e Element) {
	if sv, ok := e.Value.(*sequenceValue); ok {
		for it := range sv.seq.Items() {
			w.walk(it.DataSet)
		}
	}

	if e.Tag.IsPrivate() {
		w.applyPrivate(ds, e)
		return
	}

	action, listed := basicProfileAction(e.Tag)
	if !listed {
		return // K: keep attributes not named in the table.
	}
	action = w.resolveAction(e.Tag, action)
	w.applyAction(ds, e, action)
}

// resolveAction adjusts a table action for the caller's retained-option choices and
// the temporal policy.
func (w *deidWalk) resolveAction(t Tag, action deidAction) deidAction {
	o := w.profile.options

	if o.retainUIDs && action == deidReplaceUID {
		return deidKeep
	}
	if o.retainPatientCharacteristics && isPatientCharacteristic(t) {
		return deidKeep
	}
	if o.retainDeviceIdentity && isDeviceIdentity(t) {
		return deidKeep
	}
	if o.retainTemporal && isDateOrTimeVR(dictVR(t)) {
		// Temporal retention overrides the default removal/zeroing: keep verbatim,
		// or shift, handled in applyAction via a dedicated action.
		return deidShiftDate
	}
	return action
}

// applyAction performs one resolved action against the element.
func (w *deidWalk) applyAction(ds *DataSet, e Element, action deidAction) {
	switch action {
	case deidKeep:
		// nothing
	case deidRemove:
		ds.Delete(e.Tag)
	case deidReplaceZero, deidClean:
		// C collapses to Z in v1: identity removed, content not preserved.
		ds.Set(Element{Tag: e.Tag, VR: e.VR, Value: NewStrings(e.VR)})
	case deidReplaceDummy:
		ds.Set(Element{Tag: e.Tag, VR: e.VR, Value: w.dummyValue(e)})
	case deidReplaceUID:
		w.applyUID(ds, e)
	case deidShiftDate:
		w.applyDate(ds, e)
	}
}

// applyUID rewrites every UID in the element's value through the stable remap so a
// repeated source UID always maps to the same replacement, preserving the reference
// graph (Codex DCM-013: referential integrity after remap).
func (w *deidWalk) applyUID(ds *DataSet, e Element) {
	sv, ok := e.Value.(*Strings)
	if !ok {
		// A UID-action attribute that is not a string value is left as-is; the table
		// only lists UI-VR attributes for U, so this is unreachable for valid input.
		return
	}
	src := sv.Strings()
	out := make([]string, len(src))
	for i, s := range src {
		out[i] = string(w.remapUID(UID(s)))
	}
	ds.Set(Element{Tag: e.Tag, VR: e.VR, Value: NewStrings(e.VR, out...)})
}

// remapUID returns the stable replacement for src, minting one on first use. An empty
// source maps to empty (a zero-length UID stays zero-length).
func (w *deidWalk) remapUID(src UID) UID {
	if src == "" {
		return ""
	}
	if dst, ok := w.uidRemap[src]; ok {
		return dst
	}
	dst := w.profile.gen.Generate()
	w.uidRemap[src] = dst
	return dst
}

// applyDate handles a retained date/time attribute under the temporal sub-option:
// DateModeKeep leaves it verbatim, DateModeShift rewrites it by the per-run offset.
func (w *deidWalk) applyDate(ds *DataSet, e Element) {
	if w.profile.options.temporalMode == DateModeKeep {
		return
	}
	sv, ok := e.Value.(*Strings)
	if !ok {
		return
	}
	src := sv.Strings()
	out := make([]string, len(src))
	for i, s := range src {
		out[i] = shiftDateValue(e.VR, s, w.dateShift)
	}
	ds.Set(Element{Tag: e.Tag, VR: e.VR, Value: NewStrings(e.VR, out...)})
}

// dummyValue produces the D-action replacement. A caller override (WithDummyValues)
// wins; otherwise a VR-appropriate generic dummy is used. Dummies are non-empty and
// VR-valid, never the original value (PRD §8.2 forbids echoing PHI).
func (w *deidWalk) dummyValue(e Element) Value {
	if override, ok := w.profile.dummies[e.Tag]; ok {
		return NewStrings(e.VR, override)
	}
	return NewStrings(e.VR, genericDummy(e.VR))
}

// applyPrivate removes private attributes unless the caller retained safe private
// creators and this attribute belongs to an allow-listed creator.
func (w *deidWalk) applyPrivate(ds *DataSet, e Element) {
	if !w.profile.options.retainSafePrivate {
		ds.Delete(e.Tag)
		return
	}
	if e.Tag.IsPrivateCreator() {
		// Keep the creator declaration itself only when it is allow-listed.
		if creator, ok := ds.GetString(e.Tag); ok && w.profile.safePrivateCreators[creator] {
			return
		}
		ds.Delete(e.Tag)
		return
	}
	// A private data element is kept only when its declaring creator is allow-listed.
	if w.privateCreatorAllowed(ds, e.Tag) {
		return
	}
	ds.Delete(e.Tag)
}

// privateCreatorAllowed reports whether the private data element t belongs to an
// allow-listed creator. The creator for a private element (gggg,xxyy) is declared at
// (gggg,00xx) where xx is the high byte of the element number.
func (w *deidWalk) privateCreatorAllowed(ds *DataSet, t Tag) bool {
	creatorElem := uint16(0x0010) | (t.Element() >> 8)
	creatorTag := NewTag(t.Group(), creatorElem)
	creator, ok := ds.GetString(creatorTag)
	if !ok {
		return false
	}
	return w.profile.safePrivateCreators[creator]
}

// genericDummy returns a non-empty, VR-valid placeholder for a D action.
func genericDummy(vr VR) string {
	switch vr {
	case VRDA:
		return "19000101"
	case VRTM:
		return "000000"
	case VRDT:
		return "19000101000000"
	case VRPN:
		return "Anonymous"
	case VRUI:
		// A UI dummy should not reach here (UIDs use the U remap), but keep it valid.
		return "0"
	default:
		return "Anonymous"
	}
}

// isDateOrTimeVR reports whether vr is a date/time VR subject to the temporal policy.
func isDateOrTimeVR(vr VR) bool {
	switch vr {
	case VRDA, VRTM, VRDT:
		return true
	default:
		return false
	}
}

// shiftDateValue moves a DA/TM/DT lexical value by d, preserving the source precision.
// A value that fails to parse, or a TM with no date component to anchor a day shift,
// is returned unchanged so a malformed value never becomes fabricated data.
func shiftDateValue(vr VR, s string, d time.Duration) string {
	if s == "" {
		return s
	}
	switch vr {
	case VRDA:
		da, err := ParseDA(s)
		if err != nil {
			return s
		}
		t, ok := da.Time()
		if !ok {
			return s
		}
		return t.Add(d).Format("20060102")
	case VRDT:
		dt, err := ParseDT(s)
		if err != nil {
			return s
		}
		t, ok := dt.Time()
		if !ok {
			return s
		}
		return formatShiftedDT(dt, t.Add(d))
	case VRTM:
		// A time-only value has no date to anchor a day-granular shift; the interval
		// within a day is preserved by leaving it unchanged.
		return s
	default:
		return s
	}
}

// formatShiftedDT renders a shifted datetime at the source precision, re-appending the
// source UTC offset when one was present.
func formatShiftedDT(src DT, t time.Time) string {
	var body string
	switch src.Precision() {
	case DTPrecisionYear:
		body = t.Format("2006")
	case DTPrecisionMonth:
		body = t.Format("200601")
	case DTPrecisionDay:
		body = t.Format("20060102")
	case DTPrecisionHour:
		body = t.Format("2006010215")
	case DTPrecisionMinute:
		body = t.Format("200601021504")
	default: // second and fraction collapse to second granularity after shift
		body = t.Format("20060102150405")
	}
	if src.HasOffset() {
		secs := src.OffsetSeconds()
		sign := "+"
		if secs < 0 {
			sign = "-"
			secs = -secs
		}
		body += fmt.Sprintf("%s%02d%02d", sign, secs/3600, (secs%3600)/60)
	}
	return body
}
