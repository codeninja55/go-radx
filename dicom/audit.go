package dicom

import "time"

// AuditOp identifies the dataset-mutation operation an AuditEvent reports.
type AuditOp string

// AuditOpDeidentify is the Profile.Deidentify operation: one event per completed
// de-identification, carrying every Table E.1-1 and private-tag action applied.
const AuditOpDeidentify AuditOp = "dicom.deidentify"

// AuditAction names the structural change applied to one attribute occurrence. The
// values mirror the PS3.15 Table E.1-1 action codes the profile resolves to, by name
// rather than letter so an audit sink reads without the standard open.
type AuditAction string

const (
	// AuditActionRemove is the X action (attribute deleted), and the removal applied
	// to private tags outside a Retain Safe Private allow-list.
	AuditActionRemove AuditAction = "remove"
	// AuditActionZero is the Z action (replaced with a zero-length value), which v1
	// also applies for C (clean collapses to zero in the safe direction).
	AuditActionZero AuditAction = "zero"
	// AuditActionReplaceDummy is the D action (replaced with a VR-valid dummy).
	AuditActionReplaceDummy AuditAction = "replace-dummy"
	// AuditActionRemapUID is the U action (UID remapped through the per-call map).
	AuditActionRemapUID AuditAction = "remap-uid"
	// AuditActionShiftDate is the temporal-retention rewrite: a retained date/time
	// shifted by the per-run offset (DateModeShift). A date kept verbatim
	// (DateModeKeep) is not a modification and is not recorded.
	AuditActionShiftDate AuditAction = "shift-date"
)

// AuditChange records one attribute-level modification as structure only: the tag
// coordinate and the action applied at one occurrence (nested sequence items
// included). It never carries the attribute's value — not the original, not the
// replacement, not the minted UID.
type AuditChange struct {
	Tag    Tag
	Action AuditAction
}

// AuditEvent reports one completed dataset-mutation operation.
//
// Contract: every field is structure — the operation kind, a timestamp, tag
// coordinates, action names. Unlike the server-side audit event (package server),
// which carries object-identity UIDs because a server must name the object it
// stored, this event carries tag coordinates only: no attribute value and no UID —
// original or replacement — ever appears, and no field carrying one may be added;
// the §11.2-style sentinel test (deidentify_audit_test.go) enforces this. The fixed
// de-identification metadata the profile itself writes (PatientIdentityRemoved,
// DeidentificationMethod and its code sequence) is a documented output of every run
// and is not listed in Changes.
type AuditEvent struct {
	// Op identifies the operation that produced the event.
	Op AuditOp
	// Time is when the operation completed, in UTC.
	Time time.Time
	// Changes lists each attribute-level modification in application order. A tag
	// acted on at several nesting levels appears once per occurrence; len(Changes)
	// is the modification count.
	Changes []AuditChange
}

// AuditFunc receives one AuditEvent per completed mutation operation. It is invoked
// synchronously after the mutation succeeds (a failed operation emits nothing), so a
// slow sink slows the caller — buffer in the sink if that matters. It must be safe
// for concurrent use when the configured Profile is shared across goroutines. The
// event and its Changes slice are the callback's to keep; the profile never reuses
// them.
type AuditFunc func(AuditEvent)

// WithAudit registers f as the profile's data-modification audit hook (PRD §9.5):
// each successful Deidentify call emits exactly one AuditEvent describing the
// structural changes it applied. The default is no hook; with none configured the
// cost on the de-identification path is a nil comparison per applied action — no
// allocation, no event. go-radx provides the seam, not the sink or the schema
// beyond AuditEvent: wiring events to durable audit storage is the consumer's
// policy (PRD §9.1, §9.5).
func WithAudit(f AuditFunc) ProfileOption {
	return func(p *Profile) { p.audit = f }
}
