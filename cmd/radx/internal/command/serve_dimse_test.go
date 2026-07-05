package command

import (
	"context"
	"net"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// TestServeDIMSEInsecureBindIsUsageError confirms a non-loopback bind without authentication is
// refused as a usage error (ErrInsecureBind), matching serve dicomweb and serve fhir (RADX-017).
func TestServeDIMSEInsecureBindIsUsageError(t *testing.T) {
	dir := t.TempDir()
	_, _, code := runRadx(t, "serve", "dimse", "--bind", "0.0.0.0",
		"--object-store", filepath.Join(dir, "objects"),
		"--catalogue", filepath.Join(dir, "catalogue.db"))
	if code != exitcode.UsageError {
		t.Fatalf("serve dimse --bind 0.0.0.0 exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}

// TestServeDIMSEInvalidMoveDestinationIsUsageError pins the --move-destination grammar: the value
// must be AET=host:port with a conformant AE Title, validated before any bind.
func TestServeDIMSEInvalidMoveDestinationIsUsageError(t *testing.T) {
	dir := t.TempDir()
	for name, value := range map[string]string{
		"no-equals":   "DEST-SCP",
		"bad-aet":     "THIS-AE-TITLE-IS-WAY-TOO-LONG=127.0.0.1:104",
		"bad-address": "DEST-SCP=no-port-here",
	} {
		_, _, code := runRadx(t, "serve", "dimse",
			"--object-store", filepath.Join(dir, "objects"),
			"--catalogue", filepath.Join(dir, "catalogue.db"),
			"--move-destination", value)
		if code != exitcode.UsageError {
			t.Errorf("%s: serve dimse --move-destination %q exit = %d, want %d", name, value, code, exitcode.UsageError)
		}
	}
}

// TestServeDIMSERoundTrip is the qrscp end-to-end: radx serve dimse on loopback accepts a
// C-STORE, the stored instance is immediately retrievable over C-GET on a fresh association
// (proving the ObjectStore+Catalogue Q/R plane, not just a bound port), and a SIGINT stops the
// daemon cleanly.
func TestServeDIMSERoundTrip(t *testing.T) {
	dir := t.TempDir()
	port := freeLoopbackPort(t)

	done := make(chan int, 1)
	go func() {
		_, _, code := runRadx(t, "serve", "dimse", "--format", "json",
			"--port", strconv.Itoa(port),
			"--object-store", filepath.Join(dir, "objects"),
			"--catalogue", filepath.Join(dir, "catalogue.db"))
		done <- code
	}()

	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	// Poll until the daemon's SCP answers an association (it may still be binding).
	var storeAssoc *dimse.Association
	deadline := time.Now().Add(10 * time.Second)
	for {
		storeAssoc, err = scu.Associate(ctx, addr, dimse.AETitle("RADX-SCP"), dimse.StorageContexts())
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Associate to serve dimse: %v", err)
	}
	storeStatus, err := storeAssoc.Store(ctx, getInstance("1.2.3.4.500.1"))
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if !storeStatus.IsSuccess() {
		t.Fatalf("Store status = %s, want success", storeStatus)
	}
	_ = storeAssoc.Release(ctx)

	// Retrieve the stored instance back over C-GET on a fresh association.
	getAssoc, err := scu.Associate(ctx, addr, dimse.AETitle("RADX-SCP"),
		dimse.QueryRetrieveWithStorageContexts(), storageSCPRoles()...)
	if err != nil {
		t.Fatalf("Associate (C-GET): %v", err)
	}
	identifier := dicom.NewDataSet()
	identifier.SetString(dicom.TagStudyInstanceUID, "1.2.3.4.300.1")
	sink := &fileStoreSink{root: t.TempDir()}
	var terminal dimse.Status
	for status := range getAssoc.Get(ctx, identifier, dimse.QueryLevelStudy, sink) {
		terminal = status
	}
	if err := getAssoc.LastError(); err != nil {
		t.Fatalf("Get transport fault: %v", err)
	}
	if !terminal.IsSuccess() {
		t.Errorf("C-GET terminal status = %s, want success", terminal)
	}
	if sink.count() != 1 {
		t.Errorf("C-GET delivered %d instances, want 1", sink.count())
	}
	_ = getAssoc.Release(ctx)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case code := <-done:
		if code != exitcode.Success {
			t.Errorf("serve dimse exit = %d, want %d after SIGINT", code, exitcode.Success)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve dimse did not stop within the deadline after SIGINT")
	}
}

// TestServeDIMSEMemoryCatalogueStartsClean pins the catalogue-hardening fix: an in-memory
// (":memory:") catalogue has no filesystem file, so startup must not abort on an os.Chmod of a
// non-path. The daemon binds, answers, and stops cleanly.
func TestServeDIMSEMemoryCatalogueStartsClean(t *testing.T) {
	dir := t.TempDir()
	port := freeLoopbackPort(t)

	done := make(chan int, 1)
	go func() {
		_, _, code := runRadx(t, "serve", "dimse", "--format", "json",
			"--port", strconv.Itoa(port),
			"--object-store", filepath.Join(dir, "objects"),
			"--catalogue", ":memory:")
		done <- code
	}()

	scu, err := dimse.NewAE(dimse.AETitle("RADX-SCU"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	var assoc *dimse.Association
	deadline := time.Now().Add(10 * time.Second)
	for {
		assoc, err = scu.Associate(ctx, addr, dimse.AETitle("RADX-SCP"), dimse.VerificationContexts())
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Associate to serve dimse with :memory: catalogue: %v", err)
	}
	if status, err := assoc.Echo(ctx); err != nil || !status.IsSuccess() {
		t.Fatalf("Echo over :memory: daemon: status=%s err=%v", status, err)
	}
	_ = assoc.Release(ctx)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case code := <-done:
		if code != exitcode.Success {
			t.Errorf("serve dimse exit = %d, want %d after SIGINT", code, exitcode.Success)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve dimse did not stop within the deadline after SIGINT")
	}
}
