//go:build interop

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/dimse/integration/orthanc"
)

// cgetSink is the StoreHandler the C-GET SCU passes to receive the same-association sub-operation
// C-STOREs Orthanc sends back. It records each SOP Instance UID it received and answers success unless
// the instance is the configured rejectUID, for which it returns a Failure C-STORE-RSP — the lever the
// partial-failure fixture uses to make a sub-operation the destination rejects surface as a non-success
// terminal status. It is concurrency-safe; sub-operations arrive on the SCU goroutine that drives Get.
type cgetSink struct {
	rejectUID string

	mu        sync.Mutex
	received  []string
	rejected  []string
	storeUIDs map[string]bool
}

func newCGetSink(rejectUID string) *cgetSink {
	return &cgetSink{rejectUID: rejectUID, storeUIDs: map[string]bool{}}
}

func (s *cgetSink) Store(_ context.Context, ds *dicom.DataSet, _ dimse.OpInfo) dimse.Status {
	sopInstanceUID, _ := ds.GetString(dicom.NewTag(0x0008, 0x0018))
	s.mu.Lock()
	s.received = append(s.received, sopInstanceUID)
	s.storeUIDs[sopInstanceUID] = true
	reject := s.rejectUID != "" && sopInstanceUID == s.rejectUID
	if reject {
		s.rejected = append(s.rejected, sopInstanceUID)
	}
	s.mu.Unlock()
	if reject {
		// Refuse this sub-operation with a Storage failure. Orthanc, as the C-GET SCP, counts it as a
		// failed sub-operation and reports a non-success terminal C-GET-RSP — the property under test.
		return dimse.StatusStoreCannotUnderstand
	}
	return dimse.StatusStoreSuccess
}

func (s *cgetSink) snapshot() (received, rejected []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.received...), append([]string(nil), s.rejected...)
}

func (s *cgetSink) sawInstance(sopInstanceUID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storeUIDs[sopInstanceUID]
}

// seedStudyForGet stores the vendored fixture into Orthanc and waits for it to index, returning the
// study and SOP Instance UIDs the C-GET retrieves. It is the shared setup for the C-GET interop gates.
func seedStudyForGet(ctx context.Context, t *testing.T, orth *orthanc.Container, ae *dimse.AE, called dimse.AETitle) (studyInstanceUID, sopInstanceUID string) {
	t.Helper()
	ds, sopUID := readFixture(t)
	studyUID, ok := ds.GetString(dicom.NewTag(0x0020, 0x000D))
	if !ok || studyUID == "" {
		t.Fatalf("fixture has no Study Instance UID (0020,000D)")
	}

	storeAssoc, err := ae.Associate(ctx, orth.DICOMAddr(), called, dimse.StorageContexts())
	if err != nil {
		t.Fatalf("associate for seed C-STORE: %v", err)
	}
	if status, serr := storeAssoc.Store(ctx, ds); serr != nil || !status.IsSuccess() {
		t.Fatalf("seed C-STORE status=%s err=%v, want success", status, serr)
	}
	if err := storeAssoc.Release(ctx); err != nil {
		t.Fatalf("release seed C-STORE association: %v", err)
	}
	if !waitForInstance(ctx, t, orth, sopUID) {
		t.Fatalf("Orthanc did not persist the seed instance %s before the C-GET", sopUID)
	}
	return studyUID, sopUID
}

// requireStorageSCPRoleGranted fails the test unless the acceptor (Orthanc) granted the requestor the
// Storage SCP role for the C-GET sub-operation SOP Class; without it the same-association C-GET cannot
// deliver instances. The fixture is MR Image Storage.
func requireStorageSCPRoleGranted(t *testing.T, assoc *dimse.Association) {
	t.Helper()
	const mrImageStorage = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.4")
	for _, r := range assoc.NegotiatedRoles() {
		if r.SOPClassUID == mrImageStorage && r.SCPRole {
			return
		}
	}
	t.Fatalf("acceptor did not grant the Storage SCP role for %s; same-association C-GET cannot proceed", mrImageStorage)
}

// TestInteropOrthancCGet is the M3 C-GET interop gate against Orthanc, exercising go-radx as the C-GET
// SCU AND as the same-association sub-operation C-STORE sink. A vendored study is stored into Orthanc;
// the go-radx SCU then drives Association.Get against Orthanc, proposing the Query/Retrieve and Storage
// contexts plus the Storage SCP role selection so Orthanc C-STOREs the matched instance back on the
// SAME association. The test asserts the sink received the exact SOP Instance UID and the retrieve
// ended with a success-class terminal status with at least one completed sub-operation. This proves
// the same-association C-GET interoperates with a real third-party C-GET SCP end to end.
func TestInteropOrthancCGet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)

	calling, err := dimse.ParseAETitle("RADX-SCU")
	if err != nil {
		t.Fatalf("parse calling AE title: %v", err)
	}
	called, err := dimse.ParseAETitle(orthanc.AETitle)
	if err != nil {
		t.Fatalf("parse called AE title: %v", err)
	}
	ae, err := dimse.NewAE(calling)
	if err != nil {
		t.Fatalf("new AE: %v", err)
	}

	studyInstanceUID, sopInstanceUID := seedStudyForGet(ctx, t, orth, ae, called)

	const mrImageStorage = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.4")
	getAssoc, err := ae.Associate(ctx, orth.DICOMAddr(), called, dimse.QueryRetrieveWithStorageContexts(),
		dimse.WithRoleSelection(dimse.RoleSelection{SOPClassUID: mrImageStorage, SCURole: true, SCPRole: true}))
	if err != nil {
		t.Fatalf("associate for C-GET: %v", err)
	}
	defer func() { _ = getAssoc.Release(ctx) }()
	requireStorageSCPRoleGranted(t, getAssoc)

	sink := newCGetSink("")
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, studyInstanceUID)

	var terminal dimse.Status
	var terminalSeen bool
	for status := range getAssoc.Get(ctx, query, dimse.QueryLevelStudy, sink) {
		if !status.IsPending() {
			terminal = status
			terminalSeen = true
		}
	}
	if err := getAssoc.LastError(); err != nil {
		t.Fatalf("C-GET transport error: %v", err)
	}
	if !terminalSeen || (!terminal.IsSuccess() && !terminal.IsWarning()) {
		t.Fatalf("C-GET terminal status = %s (seen=%v), want success-class", terminal, terminalSeen)
	}
	if counts := getAssoc.SubOperationCounts(); counts.Completed < 1 {
		t.Errorf("C-GET final Completed sub-operations = %d, want >= 1", counts.Completed)
	}
	if !sink.sawInstance(sopInstanceUID) {
		received, _ := sink.snapshot()
		t.Fatalf("C-GET sink received %v, want the retrieved instance %s", received, sopInstanceUID)
	}
}

// TestInteropOrthancCGetPartialFailure is the highest-consequence C-GET correctness gate against a
// real PACS: a deliberately-rejected sub-operation must surface as a FAILURE (or WARNING) terminal
// status, never laundered into Success. A vendored study is stored into Orthanc, the go-radx SCU
// drives Association.Get, and its sink REJECTS the retrieved instance with a Storage failure
// C-STORE-RSP. Orthanc, as the C-GET SCP, counts that as a failed sub-operation and reports a
// non-success terminal C-GET-RSP. The test asserts the SCU never saw a Success terminal and the final
// Failed sub-operation count is non-zero — the partial failure is reported faithfully (PRD §9.2).
func TestInteropOrthancCGetPartialFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)

	calling, err := dimse.ParseAETitle("RADX-SCU")
	if err != nil {
		t.Fatalf("parse calling AE title: %v", err)
	}
	called, err := dimse.ParseAETitle(orthanc.AETitle)
	if err != nil {
		t.Fatalf("parse called AE title: %v", err)
	}
	ae, err := dimse.NewAE(calling)
	if err != nil {
		t.Fatalf("new AE: %v", err)
	}

	studyInstanceUID, sopInstanceUID := seedStudyForGet(ctx, t, orth, ae, called)

	const mrImageStorage = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.4")
	getAssoc, err := ae.Associate(ctx, orth.DICOMAddr(), called, dimse.QueryRetrieveWithStorageContexts(),
		dimse.WithRoleSelection(dimse.RoleSelection{SOPClassUID: mrImageStorage, SCURole: true, SCPRole: true}))
	if err != nil {
		t.Fatalf("associate for C-GET: %v", err)
	}
	defer func() { _ = getAssoc.Release(ctx) }()
	requireStorageSCPRoleGranted(t, getAssoc)

	// The sink rejects the only instance the study contains, forcing a failed sub-operation.
	sink := newCGetSink(sopInstanceUID)
	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, studyInstanceUID)

	var terminal dimse.Status
	var sawSuccess, terminalSeen bool
	for status := range getAssoc.Get(ctx, query, dimse.QueryLevelStudy, sink) {
		if status.IsSuccess() {
			sawSuccess = true
		}
		if !status.IsPending() {
			terminal = status
			terminalSeen = true
		}
	}
	if err := getAssoc.LastError(); err != nil {
		t.Fatalf("C-GET transport error: %v", err)
	}

	if sawSuccess {
		t.Error("a partial-failure C-GET was laundered into a Success terminal status (PRD §9.2)")
	}
	if !terminalSeen || (!terminal.IsFailure() && !terminal.IsWarning()) {
		t.Fatalf("partial-failure C-GET terminal status = %s (seen=%v), want a Failure or Warning, never Success",
			terminal, terminalSeen)
	}
	if counts := getAssoc.SubOperationCounts(); counts.Failed < 1 {
		t.Errorf("partial-failure C-GET final Failed sub-operations = %d, want >= 1", counts.Failed)
	}
	if _, rejected := sink.snapshot(); len(rejected) < 1 {
		t.Errorf("the sink never rejected the seeded instance %s; the partial-failure lever did not fire", sopInstanceUID)
	}
}
