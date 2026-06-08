package command

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
)

// TestServeFHIRFailsClosed confirms serve fhir is a typed not-implemented error (exit 1) writing
// nothing, since the FHIR server role is a separate increment (the serve-fhir deferral).
func TestServeFHIRFailsClosed(t *testing.T) {
	stdout, _, code := runRadx(t, "serve", "fhir")
	if code != exitcode.GeneralFailure {
		t.Fatalf("serve fhir exit = %d, want %d (fail-closed)", code, exitcode.GeneralFailure)
	}
	if stdout != "" {
		t.Errorf("serve fhir wrote to stdout: %q (must write nothing)", stdout)
	}
}

// TestServeDICOMwebInsecureBindIsUsageError confirms a non-loopback bind without authentication is
// refused as a usage error (ErrInsecureBind), never a silent unauthenticated exposure (RADX-017).
func TestServeDICOMwebInsecureBindIsUsageError(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "objects")
	cat := filepath.Join(dir, "catalogue.db")

	_, _, code := runRadx(t, "serve", "dicomweb", "--bind", "0.0.0.0",
		"--object-store", store, "--catalogue", cat)
	if code != exitcode.UsageError {
		t.Fatalf("serve dicomweb --bind 0.0.0.0 exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}

// TestServeDICOMwebLoopbackRoundTrip runs the daemon on loopback in a goroutine, confirms it binds
// and answers an HTTP request, that the PHI catalogue is created 0600, then stops it with SIGINT
// and asserts a clean exit. The signal is caught by the command's own signal.NotifyContext, so it
// does not disturb the test binary.
func TestServeDICOMwebLoopbackRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "objects")
	cat := filepath.Join(dir, "catalogue.db")

	// A free loopback port for a deterministic bind.
	port := freeLoopbackPort(t)

	done := make(chan int, 1)
	go func() {
		_, _, code := runRadx(t, "serve", "dicomweb",
			"--object-store", store, "--catalogue", cat,
			"--port", strconv.Itoa(port), "--base-path", "/dicom-web")
		done <- code
	}()

	// Wait for the QIDO-RS endpoint to answer, proving the daemon bound and wired its backends.
	base := "http://127.0.0.1:" + strconv.Itoa(port) + "/dicom-web"
	if !waitForHTTP(t, base+"/studies", 5*time.Second) {
		t.Fatal("daemon did not answer within the deadline")
	}

	// The PHI catalogue must be owner-only (RADX-008).
	if info, err := os.Stat(cat); err != nil {
		t.Errorf("catalogue not created: %v", err)
	} else if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("catalogue perm = %o, want 600", perm)
	}

	// Stop the daemon with SIGINT; the command catches it and drains cleanly.
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case code := <-done:
		if code != exitcode.Success {
			t.Errorf("serve dicomweb exit = %d, want %d after SIGINT", code, exitcode.Success)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not stop within the deadline after SIGINT")
	}
}

// TestAwaitDaemonStopSurfacesPostStartupFailure is the no-hang regression: when the daemon's Run
// returns an error after it reported ready (a listener or role that died post-startup), awaitDaemonStop
// must surface that error promptly instead of blocking on a signal that never comes. The prior code
// waited on sigCtx.Done() first and ignored runErr, so a post-startup failure hung the CLI.
func TestAwaitDaemonStopSurfacesPostStartupFailure(t *testing.T) {
	// A never-cancelled signal context, so the only way out is the runErr channel.
	runErr := make(chan error, 1)
	want := errors.New("listener died after startup")
	runErr <- want

	done := make(chan error, 1)
	go func() { done <- awaitDaemonStop(context.Background(), runErr, zap.NewNop()) }()

	select {
	case got := <-done:
		if !errors.Is(got, want) {
			t.Fatalf("awaitDaemonStop returned %v, want %v", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("awaitDaemonStop hung on a post-startup daemon failure instead of returning the error")
	}
}

// TestAwaitDaemonStopGracefulOnSignal confirms the normal path: when the signal context is cancelled,
// awaitDaemonStop waits for the daemon to finish draining and returns its terminal error (nil on a
// clean stop), not a hang.
func TestAwaitDaemonStopGracefulOnSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)

	done := make(chan error, 1)
	go func() { done <- awaitDaemonStop(ctx, runErr, zap.NewNop()) }()

	// Cancel the context (a SIGINT in production), then let the daemon's Run report a clean drain.
	cancel()
	runErr <- nil

	select {
	case got := <-done:
		if got != nil {
			t.Fatalf("awaitDaemonStop on a clean signal-driven stop returned %v, want nil", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("awaitDaemonStop did not return after a clean signal-driven shutdown")
	}
}

// freeLoopbackPort returns a currently-free loopback TCP port by binding and releasing it.
func freeLoopbackPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

// waitForHTTP polls url until it answers any HTTP status (a bound server) or the deadline elapses.
func waitForHTTP(t *testing.T, url string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:noctx // a short-lived test probe
		if err == nil {
			_ = resp.Body.Close()
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}
