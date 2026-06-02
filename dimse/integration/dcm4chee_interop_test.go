//go:build interop

package integration

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/dimse/integration/dcm4chee"
)

// startDcm4chee starts a dcm4chee-arc stack and registers its teardown on the test. Container start
// pulls three images on first run and WildFly is slow to boot, so the caller's context should allow
// several minutes.
func startDcm4chee(ctx context.Context, t *testing.T) *dcm4chee.Container {
	t.Helper()
	c, err := dcm4chee.Start(ctx)
	if err != nil {
		t.Fatalf("start dcm4chee: %v", err)
	}
	t.Cleanup(func() {
		if err := c.Stop(context.Background()); err != nil {
			t.Logf("stop dcm4chee: %v", err)
		}
	})
	return c
}

// TestInteropDcm4cheeCEcho drives a go-radx SCU C-ECHO against the dcm4chee-arc archive and asserts
// the returned status is the success-class verification status.
func TestInteropDcm4cheeCEcho(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	arc := startDcm4chee(ctx, t)

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

	assoc, err := ae.Associate(ctx, arc.DICOMAddr(), called, dimse.VerificationContexts())
	if err != nil {
		t.Fatalf("associate for C-ECHO: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	status, err := assoc.Echo(ctx)
	if err != nil {
		t.Fatalf("C-ECHO transport error: %v", err)
	}
	if !status.IsSuccess() || status.Code != dimse.StatusEchoSuccess.Code {
		t.Fatalf("C-ECHO status = %s, want StatusEchoSuccess", status)
	}
}

// TestInteropDcm4cheeCStore stores a vendored .dcm to the dcm4chee-arc archive, asserts the peer
// returned the success status, and then verifies via QIDO-RS that the exact SOP Instance UID was
// indexed. This is the SCU-direction storage gate against a second, independent third-party PACS:
// go-radx must produce a C-STORE that dcm4chee accepts and persists.
func TestInteropDcm4cheeCStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	arc := startDcm4chee(ctx, t)
	ds, sopInstanceUID := readFixture(t)

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

	assoc, err := ae.Associate(ctx, arc.DICOMAddr(), called, dimse.StorageContexts())
	if err != nil {
		t.Fatalf("associate for C-STORE: %v", err)
	}
	defer func() { _ = assoc.Release(ctx) }()

	status, err := assoc.Store(ctx, ds)
	if err != nil {
		t.Fatalf("C-STORE transport error: %v", err)
	}
	if !status.IsSuccess() || status.Code != dimse.StatusStoreSuccess.Code {
		t.Fatalf("C-STORE status = %s, want StatusStoreSuccess", status)
	}

	// Release before the QIDO check so the archive has finished ingesting the association's data.
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("release association: %v", err)
	}

	// dcm4chee indexes asynchronously; poll QIDO-RS briefly for the exact instance we sent.
	if !waitForDcm4cheeInstance(ctx, t, arc, sopInstanceUID) {
		t.Fatalf("dcm4chee did not index the stored instance %s", sopInstanceUID)
	}
}

// TestInteropDcm4cheeCFind is the first M3 query/retrieve interop gate against dcm4chee-arc: a
// vendored study is STOW-RS-stored into the archive, then a go-radx SCU drives a study-level C-FIND
// (Association.Find, the Inc 2 iterator) against the archive and asserts the stored study's Study
// Instance UID is returned as a Pending match with a terminal Success. It proves the streaming C-FIND
// SCU interoperates with a second independent third-party C-FIND SCP.
func TestInteropDcm4cheeCFind(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	arc := startDcm4chee(ctx, t)
	studyInstanceUID, sopInstanceUID := fixtureStudyAndInstanceUID(t)

	// Store the fixture into the archive so it holds a study to find.
	instance, err := os.ReadFile(storeFixture)
	if err != nil {
		t.Fatalf("read fixture bytes %s: %v", storeFixture, err)
	}
	if err := arc.StoreInstance(ctx, instance); err != nil {
		t.Fatalf("STOW-RS fixture into dcm4chee: %v", err)
	}
	// The archive indexes asynchronously; wait for the instance before querying so the C-FIND matches.
	if !waitForDcm4cheeInstance(ctx, t, arc, sopInstanceUID) {
		t.Fatalf("dcm4chee did not index the stored instance %s before the C-FIND", sopInstanceUID)
	}

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

	findAssoc, err := ae.Associate(ctx, arc.DICOMAddr(), called, dimse.QueryRetrieveContexts())
	if err != nil {
		t.Fatalf("associate for C-FIND: %v", err)
	}
	defer func() { _ = findAssoc.Release(ctx) }()

	query := dicom.NewDataSet()
	query.SetString(dicom.TagStudyInstanceUID, studyInstanceUID)

	matched := false
	terminalSuccess := false
	for st, match := range findAssoc.Find(ctx, query, dimse.QueryLevelStudy) {
		switch {
		case st.IsPending():
			if match == nil {
				t.Error("Pending C-FIND match had a nil identifier")
				continue
			}
			if uid, _ := match.GetString(dicom.TagStudyInstanceUID); uid == studyInstanceUID {
				matched = true
			}
		case st.IsSuccess():
			terminalSuccess = true
		case st.IsFailure():
			t.Errorf("C-FIND terminal status = %s, want Success", st)
		}
	}
	if err := findAssoc.LastError(); err != nil {
		t.Fatalf("C-FIND transport error: %v", err)
	}
	if !matched {
		t.Errorf("C-FIND did not return the stored study %s", studyInstanceUID)
	}
	if !terminalSuccess {
		t.Error("C-FIND did not end with a terminal Success status")
	}
}

// waitForDcm4cheeInstance polls the archive's QIDO-RS API until the given SOP Instance UID appears or
// the deadline elapses, accommodating dcm4chee's asynchronous indexing.
func waitForDcm4cheeInstance(ctx context.Context, t *testing.T, arc *dcm4chee.Container, sopInstanceUID string) bool {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		found, err := arc.HasInstanceWithSOPUID(ctx, sopInstanceUID)
		if err != nil {
			t.Fatalf("query dcm4chee instances: %v", err)
		}
		if found {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// TestInteropDcm4cheeStoresToGoRadxServer is the foreign-SCU SCP interop gate: a real dcm4chee-arc
// archive acts as the C-STORE SCU against a go-radx Server SCP. It STOW-RS-stores the fixture into
// the archive, binds the Server on a container-reachable interface, registers the Server as a
// destination AE device in the archive, and drives the archive to synchronously C-STORE the study to
// it, then asserts the Server's handler received the exact SOP Instance UID. This proves the go-radx
// receive path interoperates with a second independent third-party SCU.
func TestInteropDcm4cheeStoresToGoRadxServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	arc := startDcm4chee(ctx, t)
	studyInstanceUID, sopInstanceUID := fixtureStudyAndInstanceUID(t)

	scpTitle, err := dimse.ParseAETitle("RADX-SCP")
	if err != nil {
		t.Fatalf("parse SCP AE title: %v", err)
	}
	scpAE, err := dimse.NewAE(scpTitle)
	if err != nil {
		t.Fatalf("new SCP AE: %v", err)
	}

	handler := &recordingStore{}
	srv := dimse.NewServer(scpAE, dimse.StorageContexts(), handler)

	// Bind a container-reachable interface (NOT loopback) so the archive, running in the container,
	// can dial back to the Server via host.docker.internal.
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe(ctx, "0.0.0.0:0") }()
	addr := waitForAddr(t, srv)

	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split server addr %q: %v", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse server port %q: %v", portStr, err)
	}

	// Store the fixture into the archive so it holds a study to forward.
	instance, err := os.ReadFile(storeFixture)
	if err != nil {
		t.Fatalf("read fixture bytes %s: %v", storeFixture, err)
	}
	if err := arc.StoreInstance(ctx, instance); err != nil {
		t.Fatalf("STOW-RS fixture into dcm4chee: %v", err)
	}

	// Register the go-radx Server as a destination AE device, then drive the archive to C-STORE the
	// study to it.
	if err := arc.ConfigureDestinationAE(ctx, string(scpTitle), dcm4chee.HostAccessHost, port); err != nil {
		t.Fatalf("configure go-radx destination AE in dcm4chee: %v", err)
	}
	completed, err := arc.ExportStudy(ctx, studyInstanceUID, string(scpTitle))
	if err != nil {
		t.Fatalf("drive dcm4chee to C-STORE to go-radx Server: %v", err)
	}
	if completed != 1 {
		t.Fatalf("dcm4chee export completed = %d, want 1", completed)
	}

	received := handler.snapshot()
	if len(received) != 1 || received[0] != sopInstanceUID {
		t.Fatalf("Server handler received %v, want exactly [%s]", received, sopInstanceUID)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("server shutdown: %v", err)
	}
	if err := <-serveErr; err != nil {
		t.Fatalf("ListenAndServe returned an error: %v", err)
	}
}

// fixtureStudyAndInstanceUID loads the vendored .dcm and returns its (0020,000D) Study Instance UID
// and (0008,0018) SOP Instance UID. The export endpoint addresses a study by its UID, and the SCP
// handler asserts on the instance UID.
func fixtureStudyAndInstanceUID(t *testing.T) (studyInstanceUID, sopInstanceUID string) {
	t.Helper()
	f, err := dicom.ReadFile(storeFixture)
	if err != nil {
		t.Fatalf("read fixture %s: %v", storeFixture, err)
	}
	studyInstanceUID, ok := f.DataSet.GetString(dicom.NewTag(0x0020, 0x000D))
	if !ok || studyInstanceUID == "" {
		t.Fatalf("fixture %s has no Study Instance UID (0020,000D)", storeFixture)
	}
	sopInstanceUID, ok = f.DataSet.GetString(dicom.NewTag(0x0008, 0x0018))
	if !ok || sopInstanceUID == "" {
		t.Fatalf("fixture %s has no SOP Instance UID (0008,0018)", storeFixture)
	}
	return studyInstanceUID, sopInstanceUID
}
