//go:build interop

// Package integration holds the DIMSE interop regression net: C-ECHO and C-STORE driven as an SCU
// against a real Orthanc container, plus a C-STORE received by the go-radx Server SCP. It is the
// gate that proves the prototype's last-fragment-bit defect (Codex DIMSE-001) is fixed end-to-end:
// the prototype aborted on a C-STORE to Orthanc, go-radx must succeed and the instance must be
// retrievable from Orthanc's REST API.
//
// Every test is behind the interop build tag so the default build and test run are unaffected and
// the testcontainers dependency stays out of the default build graph.
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

// TestInteropOrthancStoresToGoRadxServer stands up a go-radx Server SCP on loopback and drives a
// C-STORE into it from a second go-radx AE acting as the SCU. It proves the Server receive path:
// the handler receives the exact instance and the returned status is success. A go-radx-SCU →
// go-radx-Server store is the deterministic interop direction (no dependency on Orthanc's outbound
// modality send), so the SCP receive contract is exercised without external networking quirks. It
// then shuts the server down cleanly within a deadline.
func TestInteropOrthancStoresToGoRadxServer(t *testing.T) {
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
