package dicom

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// PHI sentinels for the audit-event no-leak assertion: synthetic, never real, yet
// shaped like the values they stand in for (the internal/phisweep convention), so a
// value leaking into any event field is caught exactly as a real value would be.
const (
	auditSentinelName = "SENTINEL^PHI^DONOTLOG"
	auditSentinelID   = "ZZZTEST-MRN-PHI-SENTINEL"
	auditSentinelAcc  = "ZZZTEST-ACC-PHI-SENTINEL"
	auditSentinelPriv = "ZZZTEST-PRIVATE-PHI-SENTINEL"
	// auditSentinelUID stands in for an identifying UID. The dicom event carries no
	// UIDs at all (unlike the server event), so neither this original nor its remap
	// may appear in any event field.
	auditSentinelUID = "1.2.3.4.5"
)

// newAuditTestDataSet builds a dataset whose patient attributes all carry PHI
// sentinels, covering each action class: D (PatientName), Z (PatientID,
// AccessionNumber), X (PatientBirthTime), U (StudyInstanceUID), private-tag removal,
// and a nested sequence item repeating PatientName so the recursive walk's changes
// are audited per occurrence.
func newAuditTestDataSet() *DataSet {
	ds := NewDataSet()
	ds.SetString(TagPatientName, auditSentinelName)         // D: replace-dummy
	ds.SetString(TagPatientID, auditSentinelID)             // D: replace-dummy
	ds.SetString(TagAccessionNumber, auditSentinelAcc)      // Z: zero
	ds.SetString(TagPatientBirthTime, "235959")             // X: remove
	ds.SetString(TagStudyInstanceUID, auditSentinelUID)     // U: remap-uid
	ds.SetString(NewTag(0x0009, 0x0010), auditSentinelPriv) // private: remove

	item := NewDataSet()
	item.SetString(TagPatientName, auditSentinelName)
	seq := NewSequence()
	seq.Append(item)
	ds.Set(Element{Tag: TagAnatomicRegionSequence, VR: VRSQ, Value: NewSequenceValue(seq)})
	return ds
}

// countChanges returns how many changes in ev act on tag with action.
func countChanges(ev AuditEvent, tag Tag, action AuditAction) int {
	n := 0
	for _, c := range ev.Changes {
		if c.Tag == tag && c.Action == action {
			n++
		}
	}
	return n
}

// TestDeidentifyAuditEvent asserts the PRD §9.5 hook fires exactly once per
// successful Deidentify and reports the structural modification metadata: the
// operation kind, a timestamp, and one (tag, action) change per attribute occurrence
// acted on, including inside sequence items.
func TestDeidentifyAuditEvent(t *testing.T) {
	var events []AuditEvent
	prof := NewProfile(testGenerator(t), WithAudit(func(ev AuditEvent) {
		events = append(events, ev)
	}))

	if _, err := prof.Deidentify(newAuditTestDataSet()); err != nil {
		t.Fatalf("Deidentify: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("audit events = %d, want exactly 1 per Deidentify call", len(events))
	}
	ev := events[0]
	if ev.Op != AuditOpDeidentify {
		t.Errorf("Op = %q, want %q", ev.Op, AuditOpDeidentify)
	}
	if ev.Time.IsZero() {
		t.Error("Time is zero, want the operation completion time")
	}

	want := []struct {
		tag    Tag
		action AuditAction
		count  int
	}{
		{TagPatientName, AuditActionReplaceDummy, 2}, // top level + sequence item
		{TagPatientID, AuditActionReplaceDummy, 1},
		{TagAccessionNumber, AuditActionZero, 1},
		{TagPatientBirthTime, AuditActionRemove, 1},
		{TagStudyInstanceUID, AuditActionRemapUID, 1},
		{NewTag(0x0009, 0x0010), AuditActionRemove, 1},
	}
	for _, w := range want {
		if got := countChanges(ev, w.tag, w.action); got != w.count {
			t.Errorf("changes for %s/%s = %d, want %d", w.tag, w.action, got, w.count)
		}
	}
}

// TestDeidentifyAuditEventCarriesNoValues is the value-absence contract test:
// sentinel values planted in every patient attribute (and a private tag) must not
// surface through any field of the emitted event, scanned over the event's full
// rendered form so a value smuggled into any current or future field is caught.
// Unlike the server-side events, the dicom event carries no UIDs either — tag
// coordinates and action names are its only identifiers.
func TestDeidentifyAuditEventCarriesNoValues(t *testing.T) {
	var events []AuditEvent
	prof := NewProfile(testGenerator(t), WithAudit(func(ev AuditEvent) {
		events = append(events, ev)
	}))

	if _, err := prof.Deidentify(newAuditTestDataSet()); err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(events))
	}

	rendered := fmt.Sprintf("%+v", events[0])
	for _, sentinel := range []string{auditSentinelName, auditSentinelID, auditSentinelAcc, auditSentinelPriv, auditSentinelUID} {
		if strings.Contains(rendered, sentinel) {
			t.Errorf("PHI sentinel %q leaked through the audit event: %s", sentinel, rendered)
		}
	}
}

// TestDeidentifyAuditFailedRunEmitsNothing asserts a fail-closed run (burned-in
// pixel PHI) emits no event: nothing was modified, so there is nothing to audit.
func TestDeidentifyAuditFailedRunEmitsNothing(t *testing.T) {
	var events []AuditEvent
	prof := NewProfile(testGenerator(t), WithAudit(func(ev AuditEvent) {
		events = append(events, ev)
	}))

	ds := newAuditTestDataSet()
	ds.SetString(TagBurnedInAnnotation, "YES")
	if _, err := prof.Deidentify(ds); err == nil {
		t.Fatal("Deidentify with burned-in pixel data should fail closed")
	}
	if len(events) != 0 {
		t.Errorf("audit events after a failed run = %d, want 0", len(events))
	}
}

// TestDeidentifyAuditDisabledIsNoBehaviorChange asserts the default (no hook): no
// events fire, and the de-identified output is byte-for-byte the same as a hooked
// run's. WithRetainUIDs pins the only nondeterministic step (UID minting) so the two
// outputs are directly comparable.
func TestDeidentifyAuditDisabledIsNoBehaviorChange(t *testing.T) {
	hooked := 0
	withHook := NewProfile(nil, WithRetainUIDs(), WithAudit(func(AuditEvent) { hooked++ }))
	without := NewProfile(nil, WithRetainUIDs())

	outHooked, err := withHook.Deidentify(newAuditTestDataSet())
	if err != nil {
		t.Fatalf("Deidentify (hooked): %v", err)
	}
	outPlain, err := without.Deidentify(newAuditTestDataSet())
	if err != nil {
		t.Fatalf("Deidentify (no hook): %v", err)
	}

	if hooked != 1 {
		t.Errorf("hooked profile emitted %d events, want 1", hooked)
	}

	tags := map[Tag]bool{}
	for e := range outHooked.All() {
		tags[e.Tag] = true
	}
	for e := range outPlain.All() {
		tags[e.Tag] = true
	}
	for tag := range tags {
		hv, hok := outHooked.GetString(tag)
		pv, pok := outPlain.GetString(tag)
		if hok != pok || hv != pv {
			t.Errorf("tag %s differs between hooked and unhooked runs: %q,%v vs %q,%v", tag, hv, hok, pv, pok)
		}
	}
}

// TestDeidentifyAuditShiftRecordsOnlyChangedValues asserts a shift-date change is
// recorded only when a value actually changes. Under DateModeShift a DA date is
// shifted and audited, but a TM time-only value - which a day-granular shift leaves
// verbatim - records no change: the audit's modification count must not be inflated
// by a no-op shift.
func TestDeidentifyAuditShiftRecordsOnlyChangedValues(t *testing.T) {
	var events []AuditEvent
	prof := NewProfile(testGenerator(t),
		WithRetainLongitudinalTemporalInformation(DateModeShift),
		WithAudit(func(ev AuditEvent) { events = append(events, ev) }))

	ds := NewDataSet()
	ds.SetString(TagStudyInstanceUID, "1.2.3.4.5")
	ds.SetString(TagStudyDate, "20240115") // DA: shifted by the per-run offset -> recorded
	ds.SetString(TagStudyTime, "143000")   // TM: a day-granular shift is a no-op -> not recorded

	out, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(events))
	}
	ev := events[0]

	if got := countChanges(ev, TagStudyDate, AuditActionShiftDate); got != 1 {
		t.Errorf("StudyDate shift changes = %d, want 1 (the date was shifted)", got)
	}
	if got := countChanges(ev, TagStudyTime, AuditActionShiftDate); got != 0 {
		t.Errorf("StudyTime shift changes = %d, want 0 (a day-granular shift is a no-op for a time-only value)", got)
	}
	if v, _ := out.GetString(TagStudyTime); v != "143000" {
		t.Errorf("StudyTime = %q, want it retained verbatim", v)
	}
}

// TestDeidentifyAuditFailClosedDeferredDeleteIsAudited asserts that when a deferred
// value cannot be loaded (its source vanished), the fail-closed removal the walk
// applies is itself audited. A removed element is a modification, never an unaudited
// one - the deferred-load error path is no exception.
func TestDeidentifyAuditFailClosedDeferredDeleteIsAudited(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone.dcm") // never created, so every Load fails
	ds := NewDataSet()
	ds.Set(Element{Tag: TagStudyInstanceUID, VR: VRUI, Value: &DeferredValue{
		tag: TagStudyInstanceUID, vr: VRUI, path: gone, offset: 0, length: 32,
	}})

	var events []AuditEvent
	prof := NewProfile(testGenerator(t), WithAudit(func(ev AuditEvent) { events = append(events, ev) }))
	out, err := prof.Deidentify(ds)
	if err != nil {
		t.Fatalf("Deidentify: %v", err)
	}
	if _, ok := out.GetString(TagStudyInstanceUID); ok {
		t.Error("a deferred UID whose source vanished must be removed fail-closed")
	}
	if len(events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(events))
	}
	if got := countChanges(events[0], TagStudyInstanceUID, AuditActionRemove); got != 1 {
		t.Errorf("fail-closed deferred delete recorded %d remove changes, want 1", got)
	}
}
