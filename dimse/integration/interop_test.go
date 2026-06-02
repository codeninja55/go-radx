//go:build interop

// Package integration holds the DIMSE interop regression net. It drives both directions against a
// real Orthanc container: go-radx as the SCU (C-ECHO and C-STORE to Orthanc) and go-radx as the SCP
// (a C-STORE received by the go-radx Server, both from a go-radx SCU on loopback and from Orthanc
// acting as a real third-party SCU). The SCU C-STORE leg is the gate that proves the prototype's
// last-fragment-bit defect (Codex DIMSE-001) is fixed end-to-end: the prototype aborted on a C-STORE
// to Orthanc, go-radx must succeed and the instance must be retrievable from Orthanc's REST API.
//
// Every test is behind the interop build tag so the default build and test run are unaffected and
// the testcontainers dependency stays out of the default build graph.
package integration

import (
	"context"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
	"github.com/codeninja55/go-radx/dimse/integration/orthanc"
)

// storeFixture is a vendored, uncompressed (Explicit VR Little Endian) MR Image Storage instance.
// MR Image Storage is in the validated Storage SOP Class set and is universally accepted by
// Orthanc, so the C-STORE leg exercises a real SOP Class without transcoding.
const storeFixture = "../../testdata/dicom/MR2_UNCI.dcm"

// readFixture loads the vendored .dcm and returns its main dataset plus its SOP Instance UID. The
// dataset already carries (0008,0016) SOP Class UID and (0008,0018) SOP Instance UID, which
// Association.Store reads to select the presentation context and build the C-STORE-RQ.
func readFixture(t *testing.T) (*dicom.DataSet, string) {
	t.Helper()
	f, err := dicom.ReadFile(storeFixture)
	if err != nil {
		t.Fatalf("read fixture %s: %v", storeFixture, err)
	}
	sopInstanceUID, ok := f.DataSet.GetString(dicom.NewTag(0x0008, 0x0018))
	if !ok || sopInstanceUID == "" {
		t.Fatalf("fixture %s has no SOP Instance UID (0008,0018)", storeFixture)
	}
	return f.DataSet, sopInstanceUID
}

// startOrthanc starts an Orthanc container and registers its teardown on the test. Container start
// pulls the image on first run, so the caller's context should allow several minutes.
func startOrthanc(ctx context.Context, t *testing.T) *orthanc.Container {
	t.Helper()
	c, err := orthanc.Start(ctx)
	if err != nil {
		t.Fatalf("start Orthanc: %v", err)
	}
	t.Cleanup(func() {
		if err := c.Stop(context.Background()); err != nil {
			t.Logf("stop Orthanc: %v", err)
		}
	})
	return c
}

// TestInteropOrthancCEcho drives a go-radx SCU C-ECHO against the Orthanc container and asserts the
// returned status is the success-class verification status.
func TestInteropOrthancCEcho(t *testing.T) {
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

	assoc, err := ae.Associate(ctx, orth.DICOMAddr(), called, dimse.VerificationContexts())
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

// TestInteropOrthancCStore stores a vendored .dcm to the Orthanc container, asserts the peer
// returned the success status, and then verifies via the Orthanc REST API that the exact SOP
// Instance UID was persisted. This is the named regression for the prototype's Orthanc abort: the
// prototype aborted here on the final-command-PDV last-fragment bit (Codex DIMSE-001); go-radx must
// succeed and the instance must be retrievable.
func TestInteropOrthancCStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	ds, sopInstanceUID := readFixture(t)

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

	assoc, err := ae.Associate(ctx, orth.DICOMAddr(), called, dimse.StorageContexts())
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

	// Release before the REST check so Orthanc has finished ingesting the association's data.
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("release association: %v", err)
	}

	// Orthanc indexes asynchronously; poll the REST API briefly for the exact instance we sent.
	if !waitForInstance(ctx, t, orth, sopInstanceUID) {
		t.Fatalf("Orthanc did not persist the stored instance %s", sopInstanceUID)
	}
}

// TestInteropOrthancCFind is the first M3 query/retrieve interop gate against Orthanc: a go-radx SCU
// stores a vendored study, then drives a study-level C-FIND (Association.Find, the Inc 2 iterator)
// against Orthanc and asserts the stored study's Study Instance UID is returned as a Pending match
// with a terminal Success. It proves the streaming C-FIND SCU interoperates with a real third-party
// C-FIND SCP end to end.
func TestInteropOrthancCFind(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	ds, sopInstanceUID := readFixture(t)
	studyInstanceUID, ok := ds.GetString(dicom.NewTag(0x0020, 0x000D))
	if !ok || studyInstanceUID == "" {
		t.Fatalf("fixture has no Study Instance UID (0020,000D)")
	}

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

	// Store the fixture so Orthanc holds a study to find.
	storeAssoc, err := ae.Associate(ctx, orth.DICOMAddr(), called, dimse.StorageContexts())
	if err != nil {
		t.Fatalf("associate for C-STORE: %v", err)
	}
	status, err := storeAssoc.Store(ctx, ds)
	if err != nil {
		t.Fatalf("C-STORE transport error: %v", err)
	}
	if !status.IsSuccess() {
		t.Fatalf("C-STORE status = %s, want success", status)
	}
	if err := storeAssoc.Release(ctx); err != nil {
		t.Fatalf("release C-STORE association: %v", err)
	}

	// Orthanc indexes asynchronously; wait for the instance before querying so the C-FIND has a match.
	if !waitForInstance(ctx, t, orth, sopInstanceUID) {
		t.Fatalf("Orthanc did not persist the stored instance %s before the C-FIND", sopInstanceUID)
	}

	// Study-level C-FIND filtered on the stored Study Instance UID.
	findAssoc, err := ae.Associate(ctx, orth.DICOMAddr(), called, dimse.QueryRetrieveContexts())
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

// waitForInstance polls Orthanc's REST API until the given SOP Instance UID appears or the deadline
// elapses, accommodating Orthanc's asynchronous indexing.
func waitForInstance(ctx context.Context, t *testing.T, orth *orthanc.Container, sopInstanceUID string) bool {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		found, err := orth.HasInstanceWithSOPUID(ctx, sopInstanceUID)
		if err != nil {
			t.Fatalf("query Orthanc instances: %v", err)
		}
		if found {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

// recordingStore is a StoreHandler that records the SOP Instance UID it received and answers with
// the storage success status, so the SCP-receive leg can assert the dataset reached the handler.
type recordingStore struct {
	mu       sync.Mutex
	received []string
}

func (r *recordingStore) Echo(_ context.Context, _ dimse.OpInfo) dimse.Status {
	return dimse.StatusEchoSuccess
}

func (r *recordingStore) Store(_ context.Context, ds *dicom.DataSet, _ dimse.OpInfo) dimse.Status {
	sopInstanceUID, _ := ds.GetString(dicom.NewTag(0x0008, 0x0018))
	r.mu.Lock()
	r.received = append(r.received, sopInstanceUID)
	r.mu.Unlock()
	return dimse.StatusStoreSuccess
}

func (r *recordingStore) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.received))
	copy(out, r.received)
	return out
}

// TestServerReceivesCStoreFromGoRadxSCU stands up a go-radx Server SCP on loopback and drives a
// C-STORE into it from a second go-radx AE acting as the SCU. It is the fast, deterministic SCP
// receive check (no container, no external networking): the handler receives the exact instance, the
// returned status is success, and the server shuts down cleanly within a deadline. The companion
// TestInteropOrthancStoresToGoRadxServer proves the same receive path against a real third-party SCU.
func TestServerReceivesCStoreFromGoRadxSCU(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ds, sopInstanceUID := readFixture(t)

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

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe(ctx, "127.0.0.1:0") }()

	// Wait for the listener to bind so Addr() reports the OS-assigned port.
	addr := waitForAddr(t, srv)

	scuTitle, err := dimse.ParseAETitle("RADX-SCU")
	if err != nil {
		t.Fatalf("parse SCU AE title: %v", err)
	}
	scuAE, err := dimse.NewAE(scuTitle)
	if err != nil {
		t.Fatalf("new SCU AE: %v", err)
	}

	assoc, err := scuAE.Associate(ctx, addr, scpTitle, dimse.StorageContexts())
	if err != nil {
		t.Fatalf("associate to go-radx Server: %v", err)
	}

	status, err := assoc.Store(ctx, ds)
	if err != nil {
		t.Fatalf("C-STORE to go-radx Server transport error: %v", err)
	}
	if !status.IsSuccess() || status.Code != dimse.StatusStoreSuccess.Code {
		t.Fatalf("C-STORE status = %s, want StatusStoreSuccess", status)
	}
	if err := assoc.Release(ctx); err != nil {
		t.Fatalf("release association: %v", err)
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

// TestInteropOrthancStoresToGoRadxServer is the foreign-SCU SCP interop gate: a real Orthanc PACS
// acts as the C-STORE SCU against a go-radx Server SCP. It uploads the fixture into Orthanc, binds the
// Server on a container-reachable interface, registers the Server as a remote modality in Orthanc, and
// drives Orthanc to C-STORE the instance to it, then asserts the Server's handler received the exact
// SOP Instance UID. This proves the go-radx receive path interoperates with a third-party SCU, not
// only with a go-radx SCU (the loopback companion above), closing the direction the gate previously
// only simulated.
func TestInteropOrthancStoresToGoRadxServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	orth := startOrthanc(ctx, t)
	_, sopInstanceUID := readFixture(t)

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

	// Bind a container-reachable interface (NOT loopback) so Orthanc, running in the container, can
	// dial back to the Server via host.docker.internal.
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

	// Upload the fixture into Orthanc so it holds an instance to forward.
	instance, err := os.ReadFile(storeFixture)
	if err != nil {
		t.Fatalf("read fixture bytes %s: %v", storeFixture, err)
	}
	orthancID, err := orth.UploadInstance(ctx, instance)
	if err != nil {
		t.Fatalf("upload fixture to Orthanc: %v", err)
	}

	// Register the go-radx Server as a remote modality, then drive Orthanc to C-STORE to it.
	if err := orth.ConfigureModality(ctx, string(scpTitle), orthanc.HostAccessHost, port); err != nil {
		t.Fatalf("configure go-radx modality in Orthanc: %v", err)
	}
	if err := orth.StoreToModality(ctx, string(scpTitle), orthancID); err != nil {
		t.Fatalf("drive Orthanc to C-STORE to go-radx Server: %v", err)
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

// waitForAddr blocks until the server has bound a listener and reports its address, failing the
// test if it does not bind promptly.
func waitForAddr(t *testing.T, srv *dimse.Server) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if addr := srv.Addr(); addr != nil {
			return addr.String()
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server did not bind a listener within the deadline")
	return ""
}
