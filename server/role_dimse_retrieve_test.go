package server

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// collectStoreHandler counts the datasets a retrieve delivers (the C-GET sub-operation sink on
// the requestor, or the C-MOVE destination SCP), reporting store success for each.
type collectStoreHandler struct {
	received atomic.Int64
}

func (h *collectStoreHandler) Store(_ context.Context, _ *dicom.DataSet, _ dimse.OpInfo) dimse.Status {
	h.received.Add(1)
	return dimse.StatusStoreSuccess
}

// putIndexed stores and indexes one dataset in the shared backends, failing the test on error.
func putIndexed(t *testing.T, store ObjectStore, cat Catalogue, ds *dicom.DataSet) {
	t.Helper()
	ctx := context.Background()
	if err := store.Put(ctx, ds); err != nil {
		t.Fatalf("seed Put: %v", err)
	}
	if err := cat.Index(ctx, ds); err != nil {
		t.Fatalf("seed Index: %v", err)
	}
}

// startDIMSEForRetrieve runs the DIMSE role in a daemon on loopback after seeding the shared
// backends via seed, returning the SCP address and role AE title. retrieve controls whether the
// C-GET/C-MOVE capability is mounted (the opt-in role option), so a test can drive both the
// enabled archive and the default refuse-retrieve posture.
func startDIMSEForRetrieve(t *testing.T, retrieve bool, seed func(store ObjectStore, cat Catalogue), opts ...DIMSERoleOption) (addr string, aet dimse.AETitle) {
	t.Helper()
	store, cat := newTestBackends(t)
	seed(store, cat)

	aet, err := dimse.ParseAETitle("RADX-SCP")
	if err != nil {
		t.Fatalf("ParseAETitle: %v", err)
	}
	roleOpts := append([]DIMSERoleOption{WithDIMSEPort(0)}, opts...)
	if retrieve {
		roleOpts = append(roleOpts, WithDIMSERetrieve())
	}
	role, err := NewDIMSERole(aet, store, cat, roleOpts...)
	if err != nil {
		t.Fatalf("NewDIMSERole: %v", err)
	}
	d, err := New(WithDIMSE(role))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runCtx, cancelRun := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(runCtx) }()
	waitForAddrs(t, d, "dimse")
	t.Cleanup(func() {
		cancelRun()
		select {
		case <-runErr:
		case <-time.After(10 * time.Second):
			t.Error("daemon did not stop after cancel")
		}
	})
	return d.Addrs()["dimse"].String(), aet
}

// startRetrieveDaemon seeds the shared backends with two instances of one study, mounts the
// retrieve capability, and runs the DIMSE role on loopback.
func startRetrieveDaemon(t *testing.T, opts ...DIMSERoleOption) (addr string, aet dimse.AETitle) {
	t.Helper()
	return startDIMSEForRetrieve(t, true, func(store ObjectStore, cat Catalogue) {
		const study = "1.2.840.113619.2.77.1"
		for _, instance := range []string{"1.2.840.113619.2.77.3", "1.2.840.113619.2.77.4"} {
			putIndexed(t, store, cat, newTestObject(study, "1.2.840.113619.2.77.2", instance))
		}
	}, opts...)
}

// retrieveStudyIdentifier is the STUDY-level identifier matching the seeded study.
func retrieveStudyIdentifier() *dicom.DataSet {
	identifier := dicom.NewDataSet()
	identifier.SetString(dicom.TagStudyInstanceUID, "1.2.840.113619.2.77.1")
	return identifier
}

// storageSCPRoleSelections proposes the Storage SCP role for every preset Storage class, so the
// C-GET SCP may C-STORE matched instances back to this requestor (PS3.7 D.3.3.4).
func storageSCPRoleSelections() []dimse.AssociateOption {
	contexts := dimse.StorageContexts()
	opts := make([]dimse.AssociateOption, 0, len(contexts))
	for _, pc := range contexts {
		opts = append(opts, dimse.WithRoleSelection(dimse.RoleSelection{
			SOPClassUID: pc.AbstractSyntax,
			SCURole:     true,
			SCPRole:     true,
		}))
	}
	return opts
}

// TestDIMSERoleCGetRetrievesStoredInstances proves the C-GET SCP mount: instances stored in the
// shared backends stream back to the requestor as same-association sub-operations, with a clean
// terminal status and both instances delivered.
func TestDIMSERoleCGetRetrievesStoredInstances(t *testing.T) {
	t.Parallel()
	addr, aet := startRetrieveDaemon(t)

	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	assoc, err := scu.Associate(ctx, addr, aet,
		dimse.QueryRetrieveWithStorageContexts(), storageSCPRoleSelections()...)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	sink := &collectStoreHandler{}
	var terminal dimse.Status
	for status := range assoc.Get(ctx, retrieveStudyIdentifier(), dimse.QueryLevelStudy, sink) {
		terminal = status
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get transport fault: %v", err)
	}
	if !terminal.IsSuccess() {
		t.Errorf("C-GET terminal status = %s, want success", terminal)
	}
	if got := sink.received.Load(); got != 2 {
		t.Errorf("received %d instances over C-GET, want 2", got)
	}
	if counts := assoc.SubOperationCounts(); counts.Completed != 2 {
		t.Errorf("completed sub-operations = %d, want 2", counts.Completed)
	}
}

// TestDIMSERoleCMoveRetrievesToKnownDestination proves the C-MOVE SCP mount with the static
// destination table: the role resolves the Move Destination AE Title, opens its own association
// to it, and C-STOREs both matched instances there.
func TestDIMSERoleCMoveRetrievesToKnownDestination(t *testing.T) {
	t.Parallel()
	// The destination Storage SCP the role C-STOREs matched instances to.
	destHandler := &collectStoreHandler{}
	destAE, err := dimse.NewAE(dimse.AETitle("DEST-SCP"))
	if err != nil {
		t.Fatalf("NewAE (dest): %v", err)
	}
	destSrv := dimse.NewServer(destAE, dimse.StorageContexts(), destHandler)
	destServed := make(chan error, 1)
	go func() { destServed <- destSrv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && destSrv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if destSrv.Addr() == nil {
		t.Fatal("destination SCP did not bind")
	}
	t.Cleanup(func() {
		sctx, scancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer scancel()
		_ = destSrv.Shutdown(sctx)
		<-destServed
	})
	destAddr := destSrv.Addr().(*net.TCPAddr).String()

	addr, aet := startRetrieveDaemon(t, WithDIMSEMoveDestinations(map[dimse.AETitle]string{
		"DEST-SCP": destAddr,
	}))

	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	assoc, err := scu.Associate(ctx, addr, aet, dimse.QueryRetrieveContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	var terminal dimse.Status
	for status := range assoc.Move(ctx, retrieveStudyIdentifier(), dimse.QueryLevelStudy, dimse.AETitle("DEST-SCP")) {
		terminal = status
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move transport fault: %v", err)
	}
	if !terminal.IsSuccess() {
		t.Errorf("C-MOVE terminal status = %s, want success", terminal)
	}
	if got := destHandler.received.Load(); got != 2 {
		t.Errorf("destination received %d instances, want 2", got)
	}
	if counts := assoc.SubOperationCounts(); counts.Completed != 2 {
		t.Errorf("completed sub-operations = %d, want 2", counts.Completed)
	}
}

// TestDIMSERoleCMoveUnknownDestinationFails pins the dcmqrscp known-AE model: a Move Destination
// AE Title outside the configured table is answered with the terminal 0xA801 "Move Destination
// Unknown" status before any sub-operation runs.
func TestDIMSERoleCMoveUnknownDestinationFails(t *testing.T) {
	t.Parallel()
	addr, aet := startRetrieveDaemon(t) // no destinations configured

	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	assoc, err := scu.Associate(ctx, addr, aet, dimse.QueryRetrieveContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	var terminal dimse.Status
	for status := range assoc.Move(ctx, retrieveStudyIdentifier(), dimse.QueryLevelStudy, dimse.AETitle("NOWHERE")) {
		terminal = status
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move transport fault: %v", err)
	}
	if terminal.Code != 0xA801 {
		t.Errorf("C-MOVE terminal status = %s (0x%04X), want 0xA801 Move Destination Unknown", terminal, terminal.Code)
	}
}

// cgetOneStudy runs a STUDY-level C-GET with the given identifier against addr and returns the
// terminal status and the count of instances delivered to a fresh sink.
func cgetOneStudy(t *testing.T, addr string, aet dimse.AETitle, identifier *dicom.DataSet) (dimse.Status, int64) {
	t.Helper()
	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	assoc, err := scu.Associate(ctx, addr, aet,
		dimse.QueryRetrieveWithStorageContexts(), storageSCPRoleSelections()...)
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	sink := &collectStoreHandler{}
	var terminal dimse.Status
	for status := range assoc.Get(ctx, identifier, dimse.QueryLevelStudy, sink) {
		terminal = status
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Get transport fault: %v", err)
	}
	return terminal, sink.received.Load()
}

// TestDIMSERoleRetrieveRequiresUniqueKey is the PHI merge-blocker guard: a STUDY-level retrieve
// whose StudyInstanceUID is absent, or present but universal/empty, must NOT run an unconstrained
// archive-wide query. It is refused with a 0xA900 identifier fault and delivers zero instances.
func TestDIMSERoleRetrieveRequiresUniqueKey(t *testing.T) {
	t.Parallel()
	addr, aet := startRetrieveDaemon(t)

	t.Run("absent unique key", func(t *testing.T) {
		terminal, delivered := cgetOneStudy(t, addr, aet, dicom.NewDataSet())
		if terminal.Code != 0xA900 {
			t.Errorf("terminal = 0x%04X, want 0xA900 for an absent unique key", terminal.Code)
		}
		if delivered != 0 {
			t.Errorf("delivered %d instances, want 0 (must not stream the archive)", delivered)
		}
	})
	t.Run("empty unique key", func(t *testing.T) {
		id := dicom.NewDataSet()
		id.SetEmpty(dicom.TagStudyInstanceUID)
		terminal, delivered := cgetOneStudy(t, addr, aet, id)
		if terminal.Code != 0xA900 {
			t.Errorf("terminal = 0x%04X, want 0xA900 for a universal unique key", terminal.Code)
		}
		if delivered != 0 {
			t.Errorf("delivered %d instances, want 0 (must not stream the archive)", delivered)
		}
	})
}

// TestDIMSERoleRetrieveUIDList is the PHI merge-blocker for UID-list matching: a retrieve carrying
// StudyInstanceUID=A\B (legal per PS3.4 C.2.2.2.4) must return instances of BOTH studies, not just
// the first backslash value.
func TestDIMSERoleRetrieveUIDList(t *testing.T) {
	t.Parallel()
	const studyA = "1.2.840.113619.2.90.1"
	const studyB = "1.2.840.113619.2.90.2"
	addr, aet := startDIMSEForRetrieve(t, true, func(store ObjectStore, cat Catalogue) {
		putIndexed(t, store, cat, newTestObject(studyA, studyA+".1", studyA+".1.1"))
		putIndexed(t, store, cat, newTestObject(studyB, studyB+".1", studyB+".1.1"))
	})

	id := dicom.NewDataSet()
	id.Set(dicom.Element{Tag: dicom.TagStudyInstanceUID, VR: dicom.VRUI, Value: dicom.NewStrings(dicom.VRUI, studyA, studyB)})
	terminal, delivered := cgetOneStudy(t, addr, aet, id)
	if !terminal.IsSuccess() {
		t.Errorf("terminal = %s, want success", terminal)
	}
	if delivered != 2 {
		t.Errorf("UID-list retrieve delivered %d instances, want 2 (both studies)", delivered)
	}
}

// TestDIMSERoleRetrieveUnindexedKeyFilters confirms an identifier key the catalogue does not index
// (BodyPartExamined) still constrains the retrieve: a study with a chest and a non-chest series
// yields only the chest instance, never over-disclosing like a dropped key would.
func TestDIMSERoleRetrieveUnindexedKeyFilters(t *testing.T) {
	t.Parallel()
	const study = "1.2.840.113619.2.91.1"
	addr, aet := startDIMSEForRetrieve(t, true, func(store ObjectStore, cat Catalogue) {
		chest := newTestObject(study, study+".1", study+".1.1")
		chest.SetString(dicom.TagBodyPartExamined, "CHEST")
		putIndexed(t, store, cat, chest)
		abdomen := newTestObject(study, study+".2", study+".2.1")
		abdomen.SetString(dicom.TagBodyPartExamined, "ABDOMEN")
		putIndexed(t, store, cat, abdomen)
	})

	id := dicom.NewDataSet()
	id.SetString(dicom.TagStudyInstanceUID, study)
	id.SetString(dicom.TagBodyPartExamined, "CHEST")
	terminal, delivered := cgetOneStudy(t, addr, aet, id)
	if !terminal.IsSuccess() {
		t.Errorf("terminal = %s, want success", terminal)
	}
	if delivered != 1 {
		t.Errorf("delivered %d instances, want 1 (only the chest series matches BodyPartExamined=CHEST)", delivered)
	}
}

// TestDIMSERoleRetrieveDisabledByDefault pins the opt-in mount: a role built without
// WithDIMSERetrieve does not implement the retrieve capability, so a C-MOVE is refused with
// SOPClassNotSupported (0x0122) rather than silently granting archive-wide retrieve on upgrade.
// (C-MOVE is used because a C-GET's own SCU preflight aborts on the absent Storage SCP role before
// any request reaches the server, so C-MOVE is the clean capability-refusal probe.)
func TestDIMSERoleRetrieveDisabledByDefault(t *testing.T) {
	t.Parallel()
	const study = "1.2.840.113619.2.92.1"
	addr, aet := startDIMSEForRetrieve(t, false, func(store ObjectStore, cat Catalogue) {
		putIndexed(t, store, cat, newTestObject(study, study+".1", study+".1.1"))
	})

	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	assoc, err := scu.Associate(ctx, addr, aet, dimse.QueryRetrieveContexts())
	if err != nil {
		t.Fatalf("Associate: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	id := dicom.NewDataSet()
	id.SetString(dicom.TagStudyInstanceUID, study)
	var terminal dimse.Status
	for status := range assoc.Move(ctx, id, dimse.QueryLevelStudy, dimse.AETitle("DEST-SCP")) {
		terminal = status
	}
	if err := assoc.LastError(); err != nil {
		t.Fatalf("Move transport fault: %v", err)
	}
	if terminal.Code != 0x0122 {
		t.Errorf("terminal = 0x%04X, want 0x0122 SOP Class Not Supported (retrieve unmounted)", terminal.Code)
	}
}

// TestDIMSERoleStorageClassesFromConfiguredContexts is the fix for the hard-coded C-GET role
// grant: the Storage SCP role classes must derive from the role's configured contexts (so a
// custom Storage class added via WithDIMSEContexts is deliverable), excluding the Verification and
// Query/Retrieve model abstract syntaxes.
func TestDIMSERoleStorageClassesFromConfiguredContexts(t *testing.T) {
	t.Parallel()
	const customClass = dicom.SOPClassUID("1.2.840.10008.5.1.4.1.1.88.11")
	contexts := dimse.QueryRetrieveWithStorageContexts()
	nextID := uint8(2*len(contexts) + 1)
	contexts = append(contexts, dimse.NewPresentationContext(nextID, customClass))

	aet, _ := dimse.ParseAETitle("RADX-SCP")
	store, cat := newTestBackends(t)
	role, err := NewDIMSERole(aet, store, cat, WithDIMSEContexts(contexts), WithDIMSERetrieve())
	if err != nil {
		t.Fatalf("NewDIMSERole: %v", err)
	}

	classes := role.storageContextClasses()
	found, sawVerification, sawQRModel := false, false, false
	for _, c := range classes {
		switch c {
		case customClass:
			found = true
		case "1.2.840.10008.1.1": // Verification
			sawVerification = true
		case "1.2.840.10008.5.1.4.1.2.2.1": // Study Root FIND
			sawQRModel = true
		}
	}
	if !found {
		t.Errorf("derived storage classes %v missing the custom class %s", classes, customClass)
	}
	if sawVerification || sawQRModel {
		t.Errorf("derived storage classes include a non-storage abstract syntax (verification=%v qr=%v)", sawVerification, sawQRModel)
	}
}
