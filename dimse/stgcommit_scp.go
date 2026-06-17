package dimse

import (
	"context"

	"github.com/codeninja55/go-radx/dicom"
)

// StorageCommitmentNoSuchAction is the N-ACTION Failure "No Such Action" (0x0123, PS3.7 Annex C, in
// the Storage Commitment N-ACTION status table): the Action Type ID is not the single action the Push
// Model defines (Request Storage Commitment, type 1). It is constructed against
// ServiceClassStorageCommitment so its category resolves through the general status table.
var StorageCommitmentNoSuchAction = NewStatus(0x0123, ServiceClassStorageCommitment)

// CommitmentDecision is the per-instance outcome a CommitmentDecider returns for one referenced SOP
// Instance: whether the SCP took custody of it (committed), and, when it did not, the Failure Reason
// to report in the Failed SOP Sequence (PS3.4 J.3.3, Table J.3-3). A Committed decision ignores the
// FailureReason; a non-committed decision must carry a reason (default StorageCommitmentProcessingFailure).
type CommitmentDecision struct {
	// Committed reports whether the SCP took storage commitment of the instance.
	Committed bool
	// FailureReason is the per-instance reason a non-committed instance failed (a StorageCommitment*
	// reason code); it is ignored when Committed is true.
	FailureReason uint16
}

// CommitmentDecider decides, for one referenced SOP Instance, whether the Storage Commitment SCP
// commits to retaining it (PS3.4 J.3.2.2). The provider calls it once per instance in the N-ACTION
// request's Referenced SOP Sequence and reports the aggregate result in the N-EVENT-REPORT: the
// committed instances in the Referenced SOP Sequence, the failed ones in the Failed SOP Sequence with
// their reasons.
//
// ref names a SOP Instance (SOP Class UID and SOP Instance UID), never a patient value; an
// implementation MUST NOT log a wider data set (PRD §9.1).
type CommitmentDecider func(ctx context.Context, ref dicom.ReferencedSOPInstance) CommitmentDecision

// CommitAllPresent is the minimal default CommitmentDecider for tests and small deployments: it
// commits every referenced instance whose SOP Instance UID is present in the provided set, and fails
// any other with StorageCommitmentNoSuchObjectInstance (0x0112). A production SCP supplies its own
// decider that checks its actual store. The returned decider reads present only; it never mutates it,
// so it is safe to share across associations.
func CommitAllPresent(present map[dicom.SOPInstanceUID]bool) CommitmentDecider {
	return func(_ context.Context, ref dicom.ReferencedSOPInstance) CommitmentDecision {
		if present[ref.SOPInstanceUID] {
			return CommitmentDecision{Committed: true}
		}
		return CommitmentDecision{Committed: false, FailureReason: StorageCommitmentNoSuchObjectInstance}
	}
}

// StorageCommitmentProvider is the Storage Commitment Push Model SCP (PS3.4 Annex J): it answers the
// N-ACTION "Request Storage Commitment" (action type 1) that asks it to take custody of a set of SOP
// Instances, then reports the outcome back to the requestor with an N-EVENT-REPORT on the SAME
// association — the synchronous reporting model. It plugs into the DIMSE-N SCP dispatch substrate as
// an NActionHandler and (for the follow-up report) an NActionReporter; register it as the Server's
// handler over a Storage Commitment Push Model presentation context (StorageCommitmentContexts).
//
// The provider holds no per-request state; it re-reads the action-information data set in
// ReportAfterAction and asks the CommitmentDecider per instance, so it is safe to share across the
// per-association goroutines the Server runs. The SCU drives a CommitmentReceiver to read the report
// on its side of the association.
//
// This is the SAME-association reporting leg: the SCP accepts the request and reports on the
// association the N-ACTION arrived on, the simplest of the two models PS3.4 J.3.3 permits. The
// separate-association model (the SCP opening a new association back to the SCU to report later) is
// not implemented on the provider side.
type StorageCommitmentProvider struct {
	decider CommitmentDecider
}

// NewStorageCommitmentProvider builds a Storage Commitment SCP whose commitment decision is delegated
// to decider. Pass CommitAllPresent for the minimal default, or a custom decider that checks the SCP's
// actual store. A nil decider is a programming error; the caller supplies one.
func NewStorageCommitmentProvider(decider CommitmentDecider) *StorageCommitmentProvider {
	return &StorageCommitmentProvider{decider: decider}
}

// NAction answers the Storage Commitment N-ACTION (PS3.4 J.3.2). It validates the request and returns
// the accept status; the commitment outcome itself is reported afterwards by ReportAfterAction via
// N-EVENT-REPORT (the request and the result are distinct DIMSE operations in the Push Model). It
// enforces:
//
//   - the Storage Commitment SOP class — the dispatch validated the context against the Requested SOP
//     Class UID; this rejects an N-ACTION whose Requested SOP Class UID is not Storage Commitment Push
//     Model with StatusSOPClassNotSupported;
//   - the well-known target instance — the N-ACTION must target the single Storage Commitment Push
//     Model SOP Instance (PS3.4 J.3.2); any other Requested SOP Instance UID is StatusNoSuchSOPInstance;
//   - the action type — only Request Storage Commitment (type 1) is defined (PS3.4 J.3.2.1); any other
//     Action Type ID is StorageCommitmentNoSuchAction (0x0123);
//   - a present action-information data set carrying the Transaction UID and Referenced SOP Sequence —
//     an absent or uncorrelatable request is StatusNoSuchArgument-shaped (0x0114, Invalid Argument).
//
// A Success status means the request was accepted; the report follows. No patient or per-instance SOP
// Instance value is logged (PRD §9.1).
func (p *StorageCommitmentProvider) NAction(_ context.Context, req NRequest) Status {
	if dicom.SOPClassUID(req.RequestedSOPClassUID) != storageCommitmentPushModelSOPClass {
		return StatusSOPClassNotSupported
	}
	if req.RequestedSOPInstanceUID != storageCommitmentPushModelInstance {
		return StatusNoSuchSOPInstance
	}
	if !req.HasActionTypeID || req.ActionTypeID != storageCommitmentActionType {
		return StorageCommitmentNoSuchAction
	}
	if req.DataSet == nil {
		return NewStatus(0x0114, ServiceClassStorageCommitment) // No Such Argument: no action information
	}
	if tuid, ok := req.DataSet.GetString(dicom.TagTransactionUID); !ok || tuid == "" {
		return NewStatus(0x0114, ServiceClassStorageCommitment) // No Such Argument: uncorrelatable
	}
	return StatusStorageCommitmentSuccess
}

// ReportAfterAction emits the follow-up N-EVENT-REPORT for an accepted Storage Commitment request
// (PS3.4 J.3.3). It re-reads the N-ACTION action-information data set, asks the CommitmentDecider per
// referenced instance, partitions the instances into committed and failed, and sends one
// N-EVENT-REPORT: event type 1 (complete) when every instance committed, event type 2 (failures exist)
// when one or more failed. The Referenced SOP Sequence lists the committed instances; the Failed SOP
// Sequence lists the failed ones with their Failure Reason. The Transaction UID is echoed so the SCU
// correlates the result to its request.
//
// A returned error is a wire/protocol fault on the report leg (it faults the association); a
// non-success N-EVENT-REPORT-RSP status from the SCU is in-band data the provider does not treat as a
// Go error. No patient or per-instance SOP Instance value is logged (PRD §9.1).
func (p *StorageCommitmentProvider) ReportAfterAction(ctx context.Context, send NReportSender, req NRequest) error {
	transactionUID, _ := req.DataSet.GetString(dicom.TagTransactionUID)

	var committed []dicom.ReferencedSOPInstance
	var failed []dicom.FailedSOPInstance
	if seq, ok := req.DataSet.GetSequence(dicom.TagReferencedSOPSequence); ok {
		for it := range seq.Items() {
			class, _ := it.DataSet.GetString(dicom.TagReferencedSOPClassUID)
			instance, _ := it.DataSet.GetString(dicom.TagReferencedSOPInstanceUID)
			ref := dicom.ReferencedSOPInstance{
				SOPClassUID:    dicom.SOPClassUID(class),
				SOPInstanceUID: dicom.SOPInstanceUID(instance),
			}
			decision := p.decider(ctx, ref)
			if decision.Committed {
				committed = append(committed, ref)
				continue
			}
			reason := decision.FailureReason
			if reason == 0 {
				reason = StorageCommitmentProcessingFailure
			}
			failed = append(failed, dicom.FailedSOPInstance{ReferencedSOPInstance: ref, FailureReason: reason})
		}
	}

	eventType := StorageCommitmentEventComplete
	if len(failed) > 0 {
		eventType = StorageCommitmentEventFailures
	}
	eventInfo := buildCommitmentReport(transactionUID, committed, failed)

	_, err := send(ctx, uint16(eventType), eventInfo)
	return err
}

// buildCommitmentReport encodes the N-EVENT-REPORT event-information data set (PS3.4 J.3.3): the
// Transaction UID (0008,1195), the Referenced SOP Sequence (0008,1199) of committed instances, and —
// only when there are failures — the Failed SOP Sequence (0008,1198) of failed instances, each item
// carrying its SOP Class UID, SOP Instance UID, and Failure Reason (0008,1197). It is the SCP-side
// counterpart of the SCU-side parseCommitmentResult, so the two round-trip.
func buildCommitmentReport(transactionUID string, committed []dicom.ReferencedSOPInstance, failed []dicom.FailedSOPInstance) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagTransactionUID, transactionUID)

	committedItems := make([]*dicom.DataSet, 0, len(committed))
	for _, ref := range committed {
		item := dicom.NewDataSet()
		item.SetString(dicom.TagReferencedSOPClassUID, string(ref.SOPClassUID))
		item.SetString(dicom.TagReferencedSOPInstanceUID, string(ref.SOPInstanceUID))
		committedItems = append(committedItems, item)
	}
	ds.Set(dicom.Element{Tag: dicom.TagReferencedSOPSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence(committedItems...))})

	if len(failed) > 0 {
		failedItems := make([]*dicom.DataSet, 0, len(failed))
		for _, f := range failed {
			item := dicom.NewDataSet()
			item.SetString(dicom.TagReferencedSOPClassUID, string(f.SOPClassUID))
			item.SetString(dicom.TagReferencedSOPInstanceUID, string(f.SOPInstanceUID))
			item.Set(dicom.Element{Tag: dicom.TagFailureReason, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, int64(f.FailureReason))})
			failedItems = append(failedItems, item)
		}
		ds.Set(dicom.Element{Tag: dicom.TagFailedSOPSequence, VR: dicom.VRSQ, Value: dicom.NewSequenceValue(dicom.NewSequence(failedItems...))})
	}

	return ds
}
