package command

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
)

// TestServeFHIRInsecureBindIsUsageError confirms a non-loopback bind without authentication is
// refused as a usage error (ErrInsecureBind), never a silent unauthenticated exposure (RADX-017),
// matching serve dicomweb.
func TestServeFHIRInsecureBindIsUsageError(t *testing.T) {
	_, _, code := runRadx(t, "serve", "fhir", "--bind", "0.0.0.0")
	if code != exitcode.UsageError {
		t.Fatalf("serve fhir --bind 0.0.0.0 exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}

// TestServeFHIRLoopbackRoundTrip runs the FHIR daemon on loopback in a goroutine, creates a
// Patient over HTTP, reads back its version 1 through the vread interaction (proving the role is
// wired over the versioned in-memory repository), then stops the daemon with SIGINT and asserts a
// clean exit. The signal is caught by the command's own signal.NotifyContext, so it does not
// disturb the test binary.
func TestServeFHIRLoopbackRoundTrip(t *testing.T) {
	port := freeLoopbackPort(t)

	done := make(chan int, 1)
	go func() {
		_, _, code := runRadx(t, "serve", "fhir",
			"--port", strconv.Itoa(port), "--base-path", "/fhir", "--release", "r5")
		done <- code
	}()

	base := "http://127.0.0.1:" + strconv.Itoa(port) + "/fhir"
	if !waitForHTTP(t, base+"/metadata", 5*time.Second) {
		t.Fatal("daemon did not answer within the deadline")
	}

	// Create a Patient, then vread its version 1: the end-to-end proof the daemon serves the
	// versioned FHIR role, not just a bound port.
	resp, err := http.Post(base+"/Patient", "application/fhir+json", //nolint:noctx // a short-lived test probe
		strings.NewReader(`{"resourceType":"Patient","gender":"female"}`))
	if err != nil {
		t.Fatalf("create Patient: %v", err)
	}
	createBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", resp.StatusCode, createBody)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createBody, &created); err != nil || created.ID == "" {
		t.Fatalf("create body has no id (%v): %s", err, createBody)
	}

	resp, err = http.Get(base + "/Patient/" + created.ID + "/_history/1") //nolint:noctx // a short-lived test probe
	if err != nil {
		t.Fatalf("vread: %v", err)
	}
	vreadBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vread status = %d, want 200; body=%s", resp.StatusCode, vreadBody)
	}
	if etag := resp.Header.Get("ETag"); etag != `W/"1"` {
		t.Errorf("vread ETag = %q, want %q", etag, `W/"1"`)
	}

	// Stop the daemon with SIGINT; the command catches it and drains cleanly.
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case code := <-done:
		if code != exitcode.Success {
			t.Errorf("serve fhir exit = %d, want %d after SIGINT", code, exitcode.Success)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not stop within the deadline after SIGINT")
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
	go func() { done <- awaitDaemonStop(context.Background(), runErr, zap.NewNop(), "serve test") }()

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
	go func() { done <- awaitDaemonStop(ctx, runErr, zap.NewNop(), "serve test") }()

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

// TestAwaitListenerStopSurfacesPostStartupFailure is the no-hang regression shared by scp and hl7
// listen: when the serve goroutine returns an error after the listener reported ready (a dead accept
// loop), awaitListenerStop must surface that error promptly without calling Shutdown and without
// blocking on a signal that never comes. The prior commands waited on sigCtx.Done() first and ignored
// the served channel, so a post-startup failure hung the CLI.
func TestAwaitListenerStopSurfacesPostStartupFailure(t *testing.T) {
	served := make(chan error, 1)
	want := errors.New("accept loop died after startup")
	served <- want

	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	done := make(chan error, 1)
	go func() { done <- awaitListenerStop(context.Background(), served, shutdown) }()

	select {
	case got := <-done:
		if !errors.Is(got, want) {
			t.Fatalf("awaitListenerStop returned %v, want %v", got, want)
		}
		if shutdownCalled {
			t.Error("awaitListenerStop called Shutdown on a self-terminated server; want no shutdown")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("awaitListenerStop hung on a post-startup listener failure instead of returning the error")
	}
}

// TestAwaitListenerStopGracefulOnSignal confirms the normal path: when the signal context is
// cancelled, awaitListenerStop calls Shutdown to drain in-flight work and then returns the serve
// goroutine's terminal error (nil on a clean stop), not a hang.
func TestAwaitListenerStopGracefulOnSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)

	shutdownCalled := false
	shutdown := func(context.Context) error {
		shutdownCalled = true
		// A real Shutdown closes the listener, which makes the serve goroutine return; model that here.
		served <- nil
		return nil
	}

	done := make(chan error, 1)
	go func() { done <- awaitListenerStop(ctx, served, shutdown) }()

	cancel() // a SIGINT in production

	select {
	case got := <-done:
		if got != nil {
			t.Fatalf("awaitListenerStop on a clean signal-driven stop returned %v, want nil", got)
		}
		if !shutdownCalled {
			t.Error("awaitListenerStop did not call Shutdown on a signal; want a graceful drain")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("awaitListenerStop did not return after a clean signal-driven shutdown")
	}
}

// TestAwaitListenerStopSurfacesShutdownError confirms a Shutdown that fails to drain within its
// context surfaces that error rather than being swallowed.
func TestAwaitListenerStopSurfacesShutdownError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)

	want := errors.New("shutdown drain timed out")
	shutdown := func(context.Context) error { return want }

	done := make(chan error, 1)
	go func() { done <- awaitListenerStop(ctx, served, shutdown) }()

	cancel()

	select {
	case got := <-done:
		if !errors.Is(got, want) {
			t.Fatalf("awaitListenerStop returned %v, want the shutdown error %v", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("awaitListenerStop hung on a failing shutdown instead of returning the error")
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
