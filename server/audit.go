package server

import (
	"time"

	"github.com/codeninja55/go-radx/dicom"
)

// AuditOp identifies the server-side write an AuditEvent reports.
type AuditOp string

const (
	// AuditOpDIMSEStore is one DIMSE C-STORE accepted, durably stored, and indexed.
	AuditOpDIMSEStore AuditOp = "dimse.c-store"
	// AuditOpSTOWStore is one STOW-RS instance durably stored and indexed.
	AuditOpSTOWStore AuditOp = "dicomweb.stow-rs"
	// AuditOpFHIRCreate is one FHIR create interaction committed by the repository.
	AuditOpFHIRCreate AuditOp = "fhir.create"
)

// AuditEvent reports one committed server-side write.
//
// No-PHI contract: every field is a structural fact — the operation kind, a
// timestamp, and identifiers. The DICOM fields are SOP Class and Study/Series/SOP
// Instance UIDs, the same identifiers the role logs at default verbosity (the §9.1
// posture in servers.md: identifiers and structure, never patient values). The FHIR
// fields are the resource type and the id/version the server minted itself — the
// repository always assigns the id and ignores any client-supplied one, so ResourceID
// is a server artifact, never a patient identifier supplied in the resource body. No
// field carries an attribute or element value, and no field that does may ever be
// added; the sentinel test (audit_test.go) enforces this.
type AuditEvent struct {
	// Op identifies the write that produced the event.
	Op AuditOp
	// Time is when the write committed, in UTC.
	Time time.Time

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
// nil comparison — no allocation, no event. go-radx provides the seam, not the sink:
// durable audit storage, retention, and schema beyond AuditEvent are the consumer's
// policy (PRD §9.1, §9.5).
func WithAudit(f AuditFunc) Option {
	return func(c *config) { c.audit = f }
}

// dicomAuditEvent builds the structural event for one stored DICOM object: the
// operation, a UTC timestamp, and the object's hierarchy identifiers. It reads
// identifier attributes only, never patient values.
func dicomAuditEvent(op AuditOp, ds *dicom.DataSet) AuditEvent {
	ev := AuditEvent{Op: op, Time: time.Now().UTC()}
	ev.SOPClassUID, _ = ds.GetString(dicom.TagSOPClassUID)
	ev.StudyInstanceUID, _ = ds.GetString(dicom.TagStudyInstanceUID)
	ev.SeriesInstanceUID, _ = ds.GetString(dicom.TagSeriesInstanceUID)
	ev.SOPInstanceUID, _ = ds.GetString(dicom.TagSOPInstanceUID)
	return ev
}
