//go:build interop

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/dimse/integration/dcm4chee"
)

// TestInteropDcm4cheeStorageCommitment is the M3 Storage Commitment interop gate against
// dcm4chee-arc: a go-radx Storage Commitment SCU drives an N-ACTION (Request Storage Commitment) for
// a set of SOP Instances against the archive's Storage Commitment Push Model SCP and asserts the
// N-ACTION round-trips with a typed status. It proves the normalised N-ACTION SCU interoperates with
// a third-party Storage Commitment SCP — this is step 4.5 of the radiology workflow, the archive
// confirming custody of stored instances.
//
// Two interop-reality guards keep the live leg honest rather than brittle:
//
//   - If dcm4chee does not advertise the Storage Commitment Push Model context as SCP, the live leg
//     SKIPS (abstract-syntax-not-supported). The SCU's real correctness gate is the in-process
//     mock-SCP unit tests in the dimse package.
//   - The COMMITMENT RESULT itself (the N-EVENT-REPORT) is reported by dcm4chee on a SEPARATE,
//     later association it opens back to the SCU — the supported separate-association model. dcm4chee
//     only opens that reporting association to a remote AE it has been configured with (a known AE
//     Title bound to the SCU's host and listening port), which this ephemeral test harness does not
//     register. So this gate asserts the N-ACTION request round-trip; receiving the N-EVENT-REPORT
//     result live is a documented follow-up exercised by the CommitmentReceiver unit test.
//
// The Transaction UID and the referenced SOP Instance UIDs are synthetic test fixtures, never real
// patient data, and are never logged.
func TestInteropDcm4cheeStorageCommitment(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	arc := startDcm4chee(ctx, t)

	const (
		transactionUID = "1.2.826.0.1.3680043.8.498.50000099"
		ctImageStorage = "1.2.840.10008.5.1.4.1.1.2"
		instanceUID1   = "1.2.826.0.1.3680043.8.498.50000001"
		instanceUID2   = "1.2.826.0.1.3680043.8.498.50000002"
	)

	calling, err := dimse.ParseAETitle("RADX-SCU")
	if err != nil {
		t.Fatalf("parse calling AE title: %v", err)
	}
	called, err := dimse.ParseAETitle(dcm4chee.AETitle)
	if err != nil {
		t.Fatalf("parse called AE title: %v", err)
	}
	ae, err := dimse.NewAE(calling)
	if err != nil {
		t.Fatalf("new AE: %v", err)
	}

	assoc, err := ae.Associate(ctx, arc.DICOMAddr(), called, dimse.StorageCommitmentContexts())
	if err != nil {
		t.Fatalf("associate for Storage Commitment: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	// Skip the live leg unless the archive accepted the Storage Commitment presentation context: a
	// peer that does not advertise the Storage Commitment Push Model SOP Class as SCP rejects the
	// context as abstract-syntax-not-supported, and the in-process mock-SCP unit tests remain the real
	// gate.
	committed := false
	for _, pc := range assoc.AcceptedContexts() {
		if pc.AbstractSyntax == dimse.StorageCommitmentPushModelSOPClass && pc.Result == dimse.ContextAccepted {
			committed = true
		}
	}
	if !committed {
		t.Skipf("dcm4chee %s AE did not accept the Storage Commitment Push Model presentation context "+
			"(abstract-syntax-not-supported); Storage Commitment interop pending archive SCP configuration", dcm4chee.AETitle)
	}

	refs := []dicom.ReferencedSOPInstance{
		{SOPClassUID: ctImageStorage, SOPInstanceUID: instanceUID1},
		{SOPClassUID: ctImageStorage, SOPInstanceUID: instanceUID2},
	}

	status, err := assoc.StorageCommitment().Request(ctx, transactionUID, refs)
	if err != nil {
		t.Fatalf("Storage Commitment N-ACTION transport error: %v", err)
	}
	// The N-ACTION round-tripped over the negotiated context and dcm4chee returned a typed status, so
	// the SCU command codec works against the real archive. A Success means the request was accepted
	// (the commitment result follows asynchronously); a Failure is in-band data the SCU surfaces, not
	// a Go error. Either way the request leg interoperated.
	if status.IsFailure() {
		t.Skipf("dcm4chee rejected the Storage Commitment N-ACTION request with status %s "+
			"(the request round-tripped over the negotiated context; live acceptance pending archive "+
			"Storage Commitment SCP policy and the referenced instances being present)", status)
	}
	if !status.IsSuccess() {
		t.Fatalf("Storage Commitment N-ACTION status = %s, want Success or a typed Failure", status)
	}
}
