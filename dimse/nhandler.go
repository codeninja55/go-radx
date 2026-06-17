package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
)

// NRequest carries the parsed fields of an inbound DIMSE-N request the SCP dispatch hands to an
// N-service handler. It is the normalised-service counterpart of the per-operation OpInfo a
// C-service handler receives, extended with the DIMSE-N reference pair and the request data set.
//
// NRequest may carry PHI. Its DataSet holds the managed object's attributes (the N-CREATE/N-SET/
// N-ACTION/N-EVENT-REPORT data set), which can include patient, study, and procedure identifiers,
// and its SOP Instance UIDs are object identifiers tied to a study. A handler MUST NOT log the
// request's contents by default (no PHI in logs, errors, or telemetry, PRD §9.1); the no-PHI
// diagnostic context is NRequest.Info (an OpInfo: AE Titles, the presentation context, the Message
// ID, the SOP Class), which is safe to log.
//
// Which fields are populated depends on the operation. An N-GET-RQ carries
// RequestedSOPClassUID/RequestedSOPInstanceUID and an optional AttributeIdentifierList, and no data
// set. An N-DELETE-RQ carries the reference pair and no data set. An N-CREATE-RQ carries an
// AffectedSOPClassUID (and optionally an AffectedSOPInstanceUID the SCU proposed) and the new
// object's attributes as DataSet. An N-SET-RQ and an N-ACTION-RQ carry the reference pair and the
// updated/action attributes as DataSet (N-ACTION also carries ActionTypeID). An N-EVENT-REPORT-RQ
// carries the Affected pair, EventTypeID, and the event attributes as DataSet.
type NRequest struct {
	// Info is the no-PHI per-operation context (AE Titles, the presentation context and its negotiated
	// transfer syntax, the Message ID, the affected SOP Class).
	Info OpInfo
	// RequestedSOPClassUID (0000,0003) and RequestedSOPInstanceUID (0000,1001) reference the existing
	// managed object an N-GET/N-SET/N-ACTION/N-DELETE targets. They are empty on an N-CREATE-RQ (which
	// carries the Affected pair) and on an N-EVENT-REPORT-RQ.
	RequestedSOPClassUID    dicom.UID
	RequestedSOPInstanceUID dicom.UID
	// AffectedSOPClassUID (0000,0002) and AffectedSOPInstanceUID (0000,1000) identify the managed
	// object an N-CREATE creates or an N-EVENT-REPORT reports against. On other operations they are
	// empty.
	AffectedSOPClassUID    dicom.UID
	AffectedSOPInstanceUID dicom.UID
	// AttributeIdentifierList is the optional N-GET Attribute Identifier List (0000,1005): the tags of
	// the attributes the SCU wants returned, or empty/nil to request every attribute (PS3.7 §10.3.2).
	AttributeIdentifierList []dicom.Tag
	// HasActionTypeID/ActionTypeID carry the N-ACTION Action Type ID (0000,1008) naming which action is
	// invoked; HasEventTypeID/EventTypeID carry the N-EVENT-REPORT Event Type ID (0000,1002).
	HasActionTypeID bool
	ActionTypeID    uint16
	HasEventTypeID  bool
	EventTypeID     uint16
	// DataSet is the request's data set, decoded with the presentation context's negotiated transfer
	// syntax: the new attributes (N-CREATE), the updated attributes (N-SET), the action information
	// (N-ACTION), or the event information (N-EVENT-REPORT). It is nil on an N-GET-RQ and an
	// N-DELETE-RQ, which carry no data set.
	DataSet *dicom.DataSet
}

// NGetHandler answers an inbound N-GET (PS3.7 §10.1.2) as an SCP. It returns the typed status and,
// on success, the attribute list it read from the managed SOP Instance — the data set the dispatch
// sends in the N-GET-RSP. A handler returns a Failure-category status (with a nil attribute set)
// when it cannot read the object — for example StatusNoSuchSOPInstance — rather than an empty
// success, so a missing object is never laundered into a success (PRD §9.2 fail-closed).
//
// An SCP that supports only some N-services implements the narrower interface (interface
// segregation, PRD §8.2); the dispatch type-asserts the capability and refuses an unsupported
// N-service with StatusSOPClassNotSupported rather than panicking.
//
// The handler MUST observe its context: Server.Shutdown cancels it, so a handler that selects on
// ctx.Done() (or threads ctx through its I/O) returns promptly.
type NGetHandler interface {
	// NGet answers an N-GET. On a Success status it returns the attribute list to send in the RSP; on a
	// non-success status it returns a nil data set.
	NGet(ctx context.Context, req NRequest) (Status, *dicom.DataSet)
}

// NDeleteHandler answers an inbound N-DELETE (PS3.7 §10.1.6) as an SCP. It returns the typed status:
// StatusSuccess only after the managed SOP Instance has been removed, or a Failure-category status
// (for example StatusNoSuchSOPInstance) when it cannot be — returning success without deleting
// violates the honest-failure rule (PRD §9.2 fail-closed).
//
// The handler MUST observe its context, as for the other N-service handlers.
type NDeleteHandler interface {
	// NDelete answers an N-DELETE. Return StatusSuccess only after the object has been removed.
	NDelete(ctx context.Context, req NRequest) Status
}

// NCreateHandler answers an inbound N-CREATE (PS3.7 §10.1.5). On success it returns the SOP Instance
// UID of the created object — the one the SCU proposed in req.AffectedSOPInstanceUID, or the one the
// SCP assigned when the SCU did not — which the dispatch echoes in the N-CREATE-RSP. It is the
// foundation hook a later MPPS SCP plugs into.
type NCreateHandler interface {
	// NCreate answers an N-CREATE. On a Success status it returns the created object's SOP Instance UID
	// to echo in the RSP; on a non-success status the returned UID is ignored.
	NCreate(ctx context.Context, req NRequest) (Status, dicom.UID)
}

// NSetHandler answers an inbound N-SET (PS3.7 §10.1.3). It is the foundation hook a later MPPS SCP
// (advancing a procedure step) plugs into.
type NSetHandler interface {
	// NSet answers an N-SET, updating the referenced managed object's attributes.
	NSet(ctx context.Context, req NRequest) Status
}

// NActionHandler answers an inbound N-ACTION (PS3.7 §10.1.4). It is the hook the Storage Commitment
// SCP (acting on a commitment request) plugs into.
type NActionHandler interface {
	// NAction answers an N-ACTION, performing the action named by req.ActionTypeID.
	NAction(ctx context.Context, req NRequest) Status
}

// NActionReporter is the optional extension an NActionHandler implements when, after the N-ACTION-RSP
// has been sent, it must emit an N-EVENT-REPORT to the requestor on the SAME association — the
// synchronous-reporting model of the Storage Commitment Push Model SCP (PS3.4 J.3.3), where the SCP
// accepts the request (N-ACTION-RSP) and then reports the commitment result (N-EVENT-REPORT) back on
// the association the N-ACTION arrived on.
//
// The dispatch type-asserts this capability AFTER it has sent the N-ACTION-RSP, and only when the
// N-ACTION status was Success — a refused request carries no follow-up report. It is invoked with an
// NReportSender bound to the same presentation context the N-ACTION arrived on, so the reporter sends
// the N-EVENT-REPORT-RQ and reads its RSP without owning the connection. A reporter that returns an
// error has faulted the association (a wire/protocol failure on the report leg); returning nil after
// a non-success N-EVENT-REPORT-RSP is the reporter's choice — the RSP status is in-band data.
//
// An NActionHandler that does NOT implement NActionReporter sends no follow-up report: the N-ACTION
// is answered with its RSP alone, the SCU-side separate-association reporting model (PS3.4 J.3.3
// also permits the report on a later association the SCP opens back to the SCU).
type NActionReporter interface {
	NActionHandler
	// ReportAfterAction emits the follow-up N-EVENT-REPORT for an N-ACTION that returned Success. send
	// is bound to the N-ACTION's presentation context and reference object; req is the original
	// N-ACTION request. A returned error faults the association.
	ReportAfterAction(ctx context.Context, send NReportSender, req NRequest) error
}

// NReportSender sends one N-EVENT-REPORT-RQ to the requestor on the association the originating
// N-ACTION arrived on and returns the requestor's N-EVENT-REPORT-RSP status. It is handed to an
// NActionReporter so a provider can report a result (for example a Storage Commitment outcome) without
// owning the connection, the presentation context, or the Message ID — the dispatch binds those.
//
// eventTypeID names the event being reported (PS3.4 J.3.3: 1 = complete, 2 = failures exist); ds is
// the event-information data set (the Referenced/Failed SOP Sequences and the Transaction UID). The
// returned Status is the requestor's N-EVENT-REPORT-RSP status, meaningful only when err is nil; a
// returned error is a wire/protocol fault on the report leg. The event-information data set may carry
// SOP Instance UIDs, so the sender never logs it (PRD §9.1).
type NReportSender func(ctx context.Context, eventTypeID uint16, ds *dicom.DataSet) (Status, error)

// NEventReportHandler answers an inbound N-EVENT-REPORT (PS3.7 §10.1.1). It is the foundation hook a
// general N-EVENT-REPORT receiver plugs into; the special-purpose CommitmentReceiver remains the
// Storage-Commitment-specific receiver.
type NEventReportHandler interface {
	// NEventReport answers an N-EVENT-REPORT, acknowledging the event named by req.EventTypeID.
	NEventReport(ctx context.Context, req NRequest) Status
}
