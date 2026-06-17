package server

import (
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// AuditOp identifies the server-side write an AuditEvent reports.
type AuditOp string

const (
	// AuditOpDIMSEStore is one DIMSE C-STORE durably stored (see Outcome for whether
	// the catalogue index also succeeded).
	AuditOpDIMSEStore AuditOp = "dimse.c-store"
	// AuditOpSTOWStore is one STOW-RS instance durably stored (see Outcome).
	AuditOpSTOWStore AuditOp = "dicomweb.stow-rs"
	// AuditOpFHIRCreate is one FHIR create interaction committed by the repository.
	AuditOpFHIRCreate AuditOp = "fhir.create"
	// AuditOpFHIRUpdate is one FHIR update (PUT) interaction committed by the
	// repository — a full-resource replace that minted a new version (the
	// create-on-update case is audited as a create, not an update).
	AuditOpFHIRUpdate AuditOp = "fhir.update"
	// AuditOpFHIRPatch is one FHIR patch (PATCH) interaction committed by the
	// repository — a partial modification that minted a new version.
	AuditOpFHIRPatch AuditOp = "fhir.patch"
	// AuditOpFHIRDelete is one FHIR delete (DELETE) interaction committed by the
	// repository — a deletion version appended to the resource's history.
	AuditOpFHIRDelete AuditOp = "fhir.delete"
)

// AuditOutcome records how far a committed write got. The durable write is the
// audited modification, so an event fires as soon as the object store has committed —
// including when the catalogue index then fails (the partial state the C-STORE
// warning status reports), which would otherwise be a durable, unaudited
// modification. A failed Put writes nothing durable and emits nothing.
type AuditOutcome string

const (
	// AuditOutcomeStoredIndexed is the clean write: durably stored and indexed. It is
	// also the FHIR create's single value — the repository create is atomic, so a
	// committed create is always fully visible.
	AuditOutcomeStoredIndexed AuditOutcome = "stored-indexed"
	// AuditOutcomeStoredUnindexed is the partial write: the object is durably stored
	// but un-indexed because the catalogue index failed — the state the DIMSE handler
	// reports as a Storage warning status and STOW-RS as the per-instance failure.
	AuditOutcomeStoredUnindexed AuditOutcome = "stored-unindexed"
	// AuditOutcomeDeleted is the FHIR delete's single value: the resource's current
	// version was retired by appending a deletion version to its history. A
	// subsequent read answers 410 Gone while prior versions remain vread-able.
	AuditOutcomeDeleted AuditOutcome = "deleted"
)

// AuditEvent reports one committed server-side write.
//
// Contract — values never, object identity always. No field carries an attribute or
// element VALUE (a patient name, an identifier value from the resource or dataset
// body, a birth date, free text), and no field that does may ever be added; the
// sentinel test (audit_test.go) enforces value absence. The event DOES carry
// object-identity UIDs (SOP Class and Study/Series/SOP Instance) — an audit trail
// that cannot name the object it audits is useless. Those UIDs are PHI-adjacent
// under PS3.15: the de-identification profile remaps them precisely because they
// identify a study's reference graph. The hook is therefore an explicit,
// operator-wired surface, not ambient diagnostics — route the audit sink with the
// same access control as the archive itself. The FHIR fields are the resource type
// and the id/version the server minted itself — the repository always assigns the
// id and ignores any client-supplied one, so ResourceID is a server artifact, never
// a patient identifier supplied in the resource body.
type AuditEvent struct {
	// Op identifies the write that produced the event.
	Op AuditOp
	// Time is when the write committed, in UTC.
	Time time.Time
	// Outcome records how far the write got: stored-indexed (clean) or
	// stored-unindexed (durably stored, catalogue index failed).
	Outcome AuditOutcome

	// SOPClassUID, StudyInstanceUID, SeriesInstanceUID, and SOPInstanceUID identify a
	// stored DICOM object (AuditOpDIMSEStore, AuditOpSTOWStore); empty for FHIR writes.
	SOPClassUID       string
	StudyInstanceUID  string
	SeriesInstanceUID string
	SOPInstanceUID    string

	// ResourceType, ResourceID, and VersionID identify a created FHIR resource
	// (AuditOpFHIRCreate); empty for DICOM writes. ResourceID and VersionID are
	// server-assigned.
	ResourceType string
	ResourceID   string
	VersionID    string
}

// AuditFunc receives one AuditEvent per committed write. It is invoked synchronously
// after the write succeeds (a failed or rejected write emits nothing), so a slow sink
// slows the request — buffer in the sink if that matters. The roles call it from
// concurrent request handlers, so it must be safe for concurrent use.
type AuditFunc func(AuditEvent)

// WithAudit registers f as the daemon's data-modification audit hook (PRD §9.5): every
// mounted role emits one AuditEvent per committed server-side write — DIMSE C-STORE,
// STOW-RS per stored instance, and the FHIR create interaction (a transaction's
// creates commit inside Repository.Transaction and are not individually audited in
// v1). The default is no hook; with none configured the cost on the write path is a
// nil comparison — no allocation, no event. The events never carry attribute values
// but do carry object-identity UIDs (see AuditEvent), which are PHI-adjacent under
// PS3.15: wiring the hook is an explicit opt-in, and the sink warrants the same
// access control as the archive itself. go-radx provides the seam, not the sink:
// durable audit storage, retention, and schema beyond AuditEvent are the consumer's
// policy (PRD §9.1, §9.5).
func WithAudit(f AuditFunc) Option {
	return func(c *config) { c.audit = f }
}

// dicomAuditEvent builds the structural event for one stored DICOM object: the
// operation, the outcome, a UTC timestamp, and the object's hierarchy identifiers.
// It reads identifier attributes only, never patient values. The SOP Class is taken
// from the dataset's (0008,0016), which is correct for STOW-RS where the stored
// instance is the only authority; the DIMSE path overrides it (see dimseStoreAuditEvent).
func dicomAuditEvent(op AuditOp, outcome AuditOutcome, ds *dicom.DataSet) AuditEvent {
	ev := AuditEvent{Op: op, Time: time.Now().UTC(), Outcome: outcome}
	ev.SOPClassUID, _ = ds.GetString(dicom.TagSOPClassUID)
	ev.StudyInstanceUID, _ = ds.GetString(dicom.TagStudyInstanceUID)
	ev.SeriesInstanceUID, _ = ds.GetString(dicom.TagSeriesInstanceUID)
	ev.SOPInstanceUID, _ = ds.GetString(dicom.TagSOPInstanceUID)
	return ev
}

// dimseStoreAuditEvent builds the event for one object stored over DIMSE C-STORE. It
// takes the SOP Class from sopClassUID - the Affected SOP Class UID of the validated
// C-STORE command, which the association negotiated and the dispatch layer checked
// against the presentation context - rather than the dataset's (0008,0016): only the
// SOP Instance UID is validated to agree between command and dataset, so a dataset
// missing (0008,0016) would otherwise emit an event with an empty SOP Class.
func dimseStoreAuditEvent(outcome AuditOutcome, ds *dicom.DataSet, sopClassUID string) AuditEvent {
	ev := dicomAuditEvent(AuditOpDIMSEStore, outcome, ds)
	ev.SOPClassUID = sopClassUID
	return ev
}
