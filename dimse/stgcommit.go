package dimse

import (
	"context"
	"fmt"
	"math"
	"net"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse/acse"
	"github.com/codeninja55/go-radx/dimse/dul"
)

// StorageCommitmentPushModelSOPClass is the Storage Commitment Push Model SOP Class UID (PS3.4 J.3),
// the abstract syntax of a Storage Commitment presentation context. It is exported so a caller may
// name it (for example to inspect the accepted contexts on an established association) or so a
// receiver recognises it.
const StorageCommitmentPushModelSOPClass = storageCommitmentPushModelSOPClass

// StorageCommitmentPushModelInstance is the well-known SOP Instance UID of the Storage Commitment
// Push Model (PS3.4 J.3.2): every Storage Commitment N-ACTION targets this single managed instance
// rather than allocating one, and an N-EVENT-REPORT reports against it. It is exported so a caller
// may recognise it on an inbound report.
const StorageCommitmentPushModelInstance = storageCommitmentPushModelInstance

// storageCommitmentPushModelInstance is the well-known Storage Commitment Push Model SOP Instance UID
// (PS3.4 J.3.2, verified against pynetdicom's STORAGE_COMMITMENT well-known instance). The N-ACTION
// request and the N-EVENT-REPORT result both reference this single instance.
const storageCommitmentPushModelInstance dicom.UID = "1.2.840.10008.1.20.1.1"

// storageCommitmentActionType is the Action Type ID (0000,1008) of the Storage Commitment
// "Request Storage Commitment" action (PS3.4 J.3.2.1): the only action type the Push Model defines.
const storageCommitmentActionType uint16 = 1

// StorageCommitmentEventType is the Event Type ID (0000,1002) an N-EVENT-REPORT carries to report a
// Storage Commitment result (PS3.4 J.3.3): event type 1 reports every requested instance committed,
// event type 2 reports that the commitment completed but one or more instances failed. It is the
// typed event the receiver reads to decide whether the report is a clean success or a partial
// failure.
type StorageCommitmentEventType uint16

const (
	// StorageCommitmentEventComplete is Event Type ID 1: the storage commitment request is complete
	// and every referenced SOP Instance was committed (PS3.4 J.3.3, "Storage Commitment Request
	// Complete").
	StorageCommitmentEventComplete StorageCommitmentEventType = 1
	// StorageCommitmentEventFailures is Event Type ID 2: the storage commitment request is complete
	// but one or more referenced SOP Instances failed commitment (PS3.4 J.3.3, "Storage Commitment
	// Request Complete — Failures Exist"). The failed instances appear in the Failed SOP Sequence with
	// their failure reasons.
	StorageCommitmentEventFailures StorageCommitmentEventType = 2
)

// storageCommitmentEventNames maps each event type to its PS3.4 J.3.3 English label.
var storageCommitmentEventNames = map[StorageCommitmentEventType]string{
	StorageCommitmentEventComplete: "Storage Commitment Request Complete",
	StorageCommitmentEventFailures: "Storage Commitment Request Complete — Failures Exist",
}

// String renders the event type's English label so a log or error never shows a bare integer.
func (e StorageCommitmentEventType) String() string {
	if name, ok := storageCommitmentEventNames[e]; ok {
		return name
	}
	return fmt.Sprintf("Unknown Storage Commitment Event (%d)", uint16(e))
}

// Storage Commitment Failure Reason codes (0008,1198 value), the per-instance reasons an
// N-EVENT-REPORT carries in the Failed SOP Sequence (PS3.4 J.3.3, Table J.3-3, verified against
// pynetdicom's storage-commitment failure reasons). They are uint16 so they assign directly to the
// dicom.FailedSOPInstance.FailureReason field.
const (
	// StorageCommitmentProcessingFailure (0x0110): a general failure in processing the commitment of
	// the SOP Instance.
	StorageCommitmentProcessingFailure uint16 = 0x0110
	// StorageCommitmentNoSuchObjectInstance (0x0112): one or more of the referenced SOP Instances is
	// not known to the SCP, so it cannot commit it.
	StorageCommitmentNoSuchObjectInstance uint16 = 0x0112
	// StorageCommitmentResourceLimitation (0x0213): the SCP cannot commit the SOP Instance because of
	// a resource limitation.
	StorageCommitmentResourceLimitation uint16 = 0x0213
	// StorageCommitmentReferencedSOPClassNotSupported (0x0122): the SCP does not support the
	// referenced SOP Class for commitment.
	StorageCommitmentReferencedSOPClassNotSupported uint16 = 0x0122
	// StorageCommitmentClassInstanceConflict (0x0119): the referenced SOP Instance does not match the
	// referenced SOP Class.
	StorageCommitmentClassInstanceConflict uint16 = 0x0119
	// StorageCommitmentDuplicateTransactionUID (0x0131): the Transaction UID is already in use for a
	// pending or completed commitment request.
	StorageCommitmentDuplicateTransactionUID uint16 = 0x0131
)

// StorageCommitmentResult is the parsed result of a Storage Commitment request, reported by the SCP
// in an N-EVENT-REPORT (PS3.4 J.3.3). It correlates to the original request by TransactionUID, lists
// the SOP Instances the SCP committed (took custody of), and lists those that failed with their
// per-instance failure reasons. A non-empty Failed list is a FAILURE that the caller must treat as
// such: HasFailures and Err exist so a failed instance is never laundered into success (the same
// typed-honesty rule as C-GET/STOW partial failures).
type StorageCommitmentResult struct {
	// TransactionUID (0008,1195) correlates the result to the originating N-ACTION request.
	TransactionUID string
	// EventType is the reported Event Type ID: Complete (all committed) or Failures (some failed).
	EventType StorageCommitmentEventType
	// Committed lists the SOP Instances the SCP committed, from the Referenced SOP Sequence (0008,1199).
	Committed []dicom.ReferencedSOPInstance
	// Failed lists the SOP Instances that failed commitment, from the Failed SOP Sequence (0008,1198),
	// each with its Failure Reason (0008,1198 item value, tag 0008,1197).
	Failed []dicom.FailedSOPInstance
}

// HasFailures reports whether any referenced SOP Instance failed commitment. A caller treating
// commitment as a gate checks this (or Err) rather than assuming success.
func (r StorageCommitmentResult) HasFailures() bool { return len(r.Failed) > 0 }

// Err returns a typed *CommitmentFailureError when one or more instances failed commitment, and nil
// otherwise. It lets a caller fail closed with errors.As on a partial-failure result rather than
// inspecting the Failed slice by hand, and the error string names only the failed-instance count and
// transaction UID — never a patient value (PRD §9.1).
func (r StorageCommitmentResult) Err() error {
	if !r.HasFailures() {
		return nil
	}
	return &CommitmentFailureError{TransactionUID: r.TransactionUID, FailedCount: len(r.Failed)}
}

// CommitmentFailureError reports that a Storage Commitment result carried one or more failed SOP
// Instances. It names the failed-instance count and the transaction UID for correlation; it never
// embeds a patient value or a per-instance SOP Instance UID in its message (PRD §9.1).
type CommitmentFailureError struct {
	TransactionUID string
	FailedCount    int
}

func (e *CommitmentFailureError) Error() string {
	return fmt.Sprintf("dimse: storage commitment reported %d failed instance(s) for transaction %s",
		e.FailedCount, e.TransactionUID)
}

// StorageCommitment is the Storage Commitment Push Model SCU bound to an association (PS3.4 J.3).
// Request issues the N-ACTION that asks the SCP to commit a set of SOP Instances; the result is
// reported asynchronously by the SCP via N-EVENT-REPORT, which a CommitmentReceiver handles. The SCP
// side (committing instances) is out of v1 scope. Obtain one with Association.StorageCommitment.
//
// An Association is not safe for concurrent operations; run one Storage Commitment exchange at a time
// on it.
type StorageCommitment struct {
	assoc *Association
}

// StorageCommitment returns the Storage Commitment Push Model SCU bound to this association. The
// caller negotiates the Storage Commitment presentation context with StorageCommitmentContexts
// before associating, then drives Request. It is nil-safe: a Request over a nil/unestablished
// association returns a typed error rather than panicking (Codex DIMSE-017).
func (a *Association) StorageCommitment() *StorageCommitment { return &StorageCommitment{assoc: a} }

// Request issues an N-ACTION that asks the SCP to take storage commitment of the referenced SOP
// Instances (PS3.4 J.3.2). transactionUID is the SCU-allocated Transaction UID (0008,1195) that
// correlates this request to the SCP's later N-EVENT-REPORT result; refs lists the SOP Instances
// whose commitment is sought, encoded as the Referenced SOP Sequence (0008,1199) of the
// action-information data set. The N-ACTION targets the well-known Storage Commitment Push Model SOP
// Instance with Action Type ID 1.
//
// An empty transaction UID, an empty reference set, or no negotiated Storage Commitment context is a
// fail-closed *ValidationError/*AssociationError returned before any wire I/O (PRD §9.2). The
// returned Status is the N-ACTION-RSP status, meaningful only when the returned error is nil; a
// Success status means the SCP accepted the request, NOT that commitment succeeded — the commitment
// result arrives later via N-EVENT-REPORT and is the authoritative success/failure signal.
//
// No patient value or per-instance SOP Instance UID is ever logged or embedded in an error string
// (PRD §9.1).
func (sc *StorageCommitment) Request(ctx context.Context, transactionUID string, refs []dicom.ReferencedSOPInstance) (Status, error) {
	a := sc.assoc
	if a == nil || a.requestor == nil {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "Storage Commitment Request on an unestablished association"}
	}
	a.mu.Lock()
	released := a.released
	a.mu.Unlock()
	if released {
		return Status{}, &AssociationError{Kind: AssociationNotEstablished, Detail: "Storage Commitment Request on a released association"}
	}
	if transactionUID == "" {
		return Status{}, &ValidationError{Detail: "Storage Commitment Request requires a Transaction UID to correlate the result"}
	}
	if len(refs) == 0 {
		return Status{}, &ValidationError{Detail: "Storage Commitment Request requires at least one referenced SOP Instance to commit"}
	}

	pcID, ts, ok := a.contextForQuery(storageCommitmentPushModelSOPClass)
	if !ok {
		return Status{}, &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "no accepted presentation context for the Storage Commitment Push Model SOP Class",
		}
	}

	actionInfo := buildCommitmentRequest(transactionUID, refs)

	conn := a.requestor.Conn()
	machine := a.requestor.Machine()
	opCtx, cancel := a.dimseContext(ctx)
	defer cancel()

	rq := CommandSet{
		CommandField:            CommandNActionRQ,
		MessageID:               a.nextMessageID(),
		RequestedSOPClassUID:    dicom.UID(storageCommitmentPushModelSOPClass),
		RequestedSOPInstanceUID: storageCommitmentPushModelInstance,
		HasActionTypeID:         true,
		ActionTypeID:            storageCommitmentActionType,
		CommandDataSetType:      CommandDataSetPresent,
	}
	if err := sendMessage(opCtx, conn, machine, pcID, rq, actionInfo, ts, a.sendCap()); err != nil {
		return Status{}, err
	}

	rsp, _, _, err := receiveMessage(opCtx, conn, machine, newMessageReassembler(ts))
	if err != nil {
		return Status{}, err
	}
	if rsp.CommandField != CommandNActionRSP {
		return Status{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "expected an N-ACTION-RSP command field in the response",
		}
	}
	if !rsp.HasStatus {
		return Status{}, &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "N-ACTION-RSP missing the mandatory Status element",
		}
	}
	return NewStatus(rsp.Status, ServiceClassStorageCommitment), nil
}

// buildCommitmentRequest encodes the N-ACTION action-information data set (PS3.4 J.3.2.1): the
// Transaction UID (0008,1195) and the Referenced SOP Sequence (0008,1199), one item per referenced
// instance carrying its SOP Class UID (0008,1150) and SOP Instance UID (0008,1155).
func buildCommitmentRequest(transactionUID string, refs []dicom.ReferencedSOPInstance) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagTransactionUID, transactionUID)
	items := make([]*dicom.DataSet, 0, len(refs))
	for _, ref := range refs {
		item := dicom.NewDataSet()
		item.SetString(dicom.TagReferencedSOPClassUID, string(ref.SOPClassUID))
		item.SetString(dicom.TagReferencedSOPInstanceUID, string(ref.SOPInstanceUID))
		items = append(items, item)
	}
	ds.Set(dicom.Element{Tag: dicom.TagReferencedSOPSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence(items...))})
	return ds
}

// parseCommitmentResult decodes an N-EVENT-REPORT event-information data set into a
// StorageCommitmentResult (PS3.4 J.3.3). eventType is the Event Type ID the N-EVENT-REPORT command
// carried; ds is the event-information data set. A nil data set or one with no Transaction UID is a
// *ValidationError: a result that cannot be correlated to a request is unusable. The committed
// instances are read from the Referenced SOP Sequence (0008,1199); the failed instances from the
// Failed SOP Sequence (0008,1198), each with its Failure Reason (0008,1197). A failed instance is
// surfaced as-is, never dropped, so HasFailures/Err can fail closed.
func parseCommitmentResult(eventType StorageCommitmentEventType, ds *dicom.DataSet) (StorageCommitmentResult, error) {
	if ds == nil {
		return StorageCommitmentResult{}, &ValidationError{Detail: "storage commitment result has no event-information data set"}
	}
	transactionUID, ok := ds.GetString(dicom.TagTransactionUID)
	if !ok || transactionUID == "" {
		return StorageCommitmentResult{}, &ValidationError{Detail: "storage commitment result has no Transaction UID to correlate it to a request"}
	}

	result := StorageCommitmentResult{TransactionUID: transactionUID, EventType: eventType}

	if seq, ok := ds.GetSequence(dicom.TagReferencedSOPSequence); ok {
		for it := range seq.Items() {
			class, _ := it.DataSet.GetString(dicom.TagReferencedSOPClassUID)
			instance, _ := it.DataSet.GetString(dicom.TagReferencedSOPInstanceUID)
			result.Committed = append(result.Committed, dicom.ReferencedSOPInstance{
				SOPClassUID:    dicom.SOPClassUID(class),
				SOPInstanceUID: dicom.SOPInstanceUID(instance),
			})
		}
	}

	if seq, ok := ds.GetSequence(dicom.TagFailedSOPSequence); ok {
		for it := range seq.Items() {
			class, _ := it.DataSet.GetString(dicom.TagReferencedSOPClassUID)
			instance, _ := it.DataSet.GetString(dicom.TagReferencedSOPInstanceUID)
			var reason uint16
			// FailureReason is a US (0..65535); a non-conformant wider value from the
			// peer is left as 0 (unknown) rather than truncated to a misleading code.
			if r, ok := it.DataSet.GetInt(dicom.TagFailureReason); ok && r >= 0 && r <= math.MaxUint16 {
				reason = uint16(r)
			}
			result.Failed = append(result.Failed, dicom.FailedSOPInstance{
				ReferencedSOPInstance: dicom.ReferencedSOPInstance{
					SOPClassUID:    dicom.SOPClassUID(class),
					SOPInstanceUID: dicom.SOPInstanceUID(instance),
				},
				FailureReason: reason,
			})
		}
	}

	return result, nil
}

// CommitmentResultHandler is invoked with each parsed Storage Commitment result a CommitmentReceiver
// reads from an inbound N-EVENT-REPORT. info is the no-PHI per-operation context (AE Titles, the
// presentation context, the affected SOP Class); result carries the committed and failed instances.
// Returning an error fails the receipt (the receiver still acknowledges the report with a Success
// N-EVENT-REPORT-RSP, since the report was validly received), so a handler should not return an error
// merely because result.HasFailures — a partial-failure report is a valid report; inspect
// result.Err for the commitment outcome.
type CommitmentResultHandler func(ctx context.Context, info OpInfo, result StorageCommitmentResult) error

// receiverConfig holds the resolved CommitmentReceiver options.
type receiverConfig struct {
	handler CommitmentResultHandler
}

// CommitmentReceiverOption configures a CommitmentReceiver at construction.
type CommitmentReceiverOption func(*receiverConfig)

// WithCommitmentHandler sets the handler a CommitmentReceiver invokes with each parsed Storage
// Commitment result from an inbound N-EVENT-REPORT. A receiver with no handler still reads and
// acknowledges the report but discards the parsed result; configure a handler to act on it (record
// the commitment, fail closed on result.Err, etc.).
func WithCommitmentHandler(h CommitmentResultHandler) CommitmentReceiverOption {
	return func(c *receiverConfig) { c.handler = h }
}

// CommitmentReceiver accepts an inbound Storage Commitment association on which the SCP reports a
// commitment result via N-EVENT-REPORT (PS3.4 J.3.3), parses the result, invokes the configured
// handler, and acknowledges the report with an N-EVENT-REPORT-RSP.
//
// This implements the SUPPORTED separate-association reporting model: a real Storage Commitment SCP
// (for example dcm4chee-arc) sends the result on a LATER association it opens back to the SCU, on
// which the roles invert — the original SCU is now the acceptor and the N-EVENT-REPORT receiver. The
// SCU drives ServeConn on each inbound connection it accepts on its listening port. Receiving the
// result SYNCHRONOUSLY on the original N-ACTION request association is a stated limitation, not
// supported: the request and the report are distinct associations in the Push Model.
//
// A CommitmentReceiver carries no global mutable state; its configuration is immutable after
// construction. It is safe to drive ServeConn from multiple goroutines, one per accepted connection.
type CommitmentReceiver struct {
	ae  *AE
	cfg receiverConfig
}

// NewCommitmentReceiver builds a Storage Commitment N-EVENT-REPORT receiver bound to the local AE.
// Options configure the result handler (WithCommitmentHandler).
func NewCommitmentReceiver(ae *AE, opts ...CommitmentReceiverOption) *CommitmentReceiver {
	var cfg receiverConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return &CommitmentReceiver{ae: ae, cfg: cfg}
}

// ServeConn negotiates an inbound Storage Commitment association on nc, reads the single inbound
// N-EVENT-REPORT-RQ carrying the commitment result, parses it, invokes the configured handler, and
// acknowledges it with an N-EVENT-REPORT-RSP before releasing the association. nc is an already
// accepted TCP connection (the caller owns the listener); ServeConn takes ownership of nc and closes
// it on return.
//
// It returns a typed error on a negotiation/protocol fault or when the parsed result cannot be
// correlated (no Transaction UID); a handler error is returned after the report is acknowledged, so
// the peer still sees a well-formed response. No patient value is logged or embedded in an error
// (PRD §9.1).
func (r *CommitmentReceiver) ServeConn(ctx context.Context, nc net.Conn) error {
	conn := dul.NewConn(nc, r.ae.config().acseTimeout)
	defer func() { _ = conn.Close() }()

	acc, err := acse.Accept(ctx, conn, acse.AcceptParams{
		CalledAETitle:          string(r.ae.Title()),
		MaxPDULength:           uint32(r.ae.config().maxPDULength),
		ImplementationClassUID: string(r.ae.config().implementationClassUID),
		ImplementationVersion:  r.ae.config().implementationVersion,
		Supported: []acse.SupportedContext{{
			AbstractSyntax:   string(storageCommitmentPushModelSOPClass),
			TransferSyntaxes: []string{string(dicom.ExplicitVRLittleEndian), string(dicom.ImplicitVRLittleEndian)},
		}},
		// In the separate-association reporting model the commitment provider opens this report
		// association and proposes the Storage Commitment Push Model SCP role (it is the
		// N-EVENT-REPORT SCP); grant the requestor that SCP role so the inverted-role negotiation
		// resolves and a strict peer is not refused. The acceptor takes the complementary SCU role.
		SupportedRoles: []acse.SupportedRole{{
			SOPClassUID: string(storageCommitmentPushModelSOPClass),
			SCPRole:     true,
		}},
	})
	if err != nil {
		return err
	}

	handlerErr := r.serveReport(ctx, acc)
	if relErr := acc.ServeRelease(ctx); relErr != nil && handlerErr == nil {
		return relErr
	}
	return handlerErr
}

// serveReport reads the inbound N-EVENT-REPORT-RQ, parses the result, invokes the handler, and writes
// the N-EVENT-REPORT-RSP. The RSP is sent for any validly-received report (including a partial-failure
// report) so the peer always sees a response; a parse or handler error is returned to the caller
// after the acknowledgement.
func (r *CommitmentReceiver) serveReport(ctx context.Context, acc *acse.Acceptor) error {
	machine := acc.Machine()
	conn := acc.Conn()

	have := false
	for _, pc := range acc.AcceptedContexts() {
		if pc.Result == 0 {
			have = true
			break
		}
	}
	if !have {
		return &AssociationError{
			Kind:   AssociationNotEstablished,
			Detail: "inbound Storage Commitment association accepted no presentation context for the report",
		}
	}

	// Decode the event data set with the transfer syntax of whichever accepted context the report
	// actually arrived on, and answer on that same context: when the reporter negotiated more than
	// one Storage Commitment context and sent on a later one, fixing on the first context would
	// mis-decode the data set or reply on the wrong context (mirrors the C-service dispatch path).
	cmd, ds, pcID, err := receiveMessage(ctx, conn, machine, newMessageReassemblerFunc(acceptedTransferSyntaxResolver(acc)))
	if err != nil {
		return err
	}
	if cmd.CommandField != CommandNEventReportRQ {
		return &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "expected an N-EVENT-REPORT-RQ on the inbound Storage Commitment association",
		}
	}

	rsp := CommandSet{
		CommandField:              CommandNEventReportRSP,
		MessageIDBeingRespondedTo: cmd.MessageID,
		AffectedSOPClassUID:       cmd.AffectedSOPClassUID,
		AffectedSOPInstanceUID:    cmd.AffectedSOPInstanceUID,
		HasEventTypeID:            cmd.HasEventTypeID,
		EventTypeID:               cmd.EventTypeID,
		CommandDataSetType:        CommandDataSetNotPresent,
		HasStatus:                 true,
		Status:                    StatusSuccess.Code,
	}

	// The Event Type ID is mandatory and bounds the report's meaning (PS3.4 J.3.3): only 1 (complete)
	// and 2 (failures exist) are defined. An absent or unrecognised type must not be parsed as a clean
	// result — a missing type leaves cmd.EventTypeID zero, which would otherwise look like a success
	// with no failed items. Answer "No Such Event Type" (PS3.7 §10.3.5) and surface the fault.
	eventType := StorageCommitmentEventType(cmd.EventTypeID)
	if !cmd.HasEventTypeID || (eventType != StorageCommitmentEventComplete && eventType != StorageCommitmentEventFailures) {
		rsp.Status = NewStatus(0x0113, ServiceClassStorageCommitment).Code
		if serr := sendCommand(ctx, conn, machine, pcID, rsp); serr != nil {
			return serr
		}
		return &ProtocolError{
			State:  machine.CurrentState(),
			Detail: "N-EVENT-REPORT-RQ carried no valid Storage Commitment Event Type ID",
		}
	}

	result, parseErr := parseCommitmentResult(eventType, ds)
	if parseErr != nil {
		// The report could not be parsed; still answer so the peer is not left hanging, then surface
		// the fault. PS3.7 §10.3.5 No Such Argument fits a malformed event report.
		rsp.Status = NewStatus(0x0114, ServiceClassStorageCommitment).Code
		if serr := sendCommand(ctx, conn, machine, pcID, rsp); serr != nil {
			return serr
		}
		return parseErr
	}

	if serr := sendCommand(ctx, conn, machine, pcID, rsp); serr != nil {
		return serr
	}

	if r.cfg.handler == nil {
		return nil
	}
	ts, _ := acceptedTransferSyntaxResolver(acc)(pcID)
	info := OpInfo{
		CallingAETitle: AETitle(acc.CallingAETitle()),
		CalledAETitle:  r.ae.Title(),
		PresentationID: pcID,
		TransferSyntax: ts,
		MessageID:      cmd.MessageID,
		SOPClassUID:    dicom.SOPClassUID(cmd.AffectedSOPClassUID),
	}
	return r.cfg.handler(ctx, info, result)
}
