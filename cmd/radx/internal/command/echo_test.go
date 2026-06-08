package command

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dimse"
)

// echoSCPHandler answers a C-ECHO with a configurable status, so a test can drive both the
// success path and a non-success status that must exit 4.
type echoSCPHandler struct{ status dimse.Status }

func (h *echoSCPHandler) Echo(context.Context, dimse.OpInfo) dimse.Status { return h.status }

// startEchoServer runs an in-process Verification SCP on loopback and returns its host and
// port, ready for `radx echo`. The server shuts down on test cleanup.
func startEchoServer(t *testing.T, status dimse.Status) (host string, port int) {
	t.Helper()
	ae, err := dimse.NewAE(dimse.AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := dimse.NewServer(ae, dimse.VerificationContexts(), &echoSCPHandler{status: status})

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("echo SCP did not bind within the deadline")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})

	tcp, ok := srv.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("Addr() = %T, want *net.TCPAddr", srv.Addr())
	}
	return "127.0.0.1", tcp.Port
}

// TestEchoSuccessGolden is the success golden: a C-ECHO against an in-process SCP returns a
// clean JSON object with status "success" on stdout, and exits 0.
func TestEchoSuccessGolden(t *testing.T) {
	host, port := startEchoServer(t, dimse.StatusEchoSuccess)

	stdout, stderr, code := runRadx(t, "echo", "--format", "json",
		"--called-ae", "RADX-SCP", "--calling-ae", "RADX",
		host, strconv.Itoa(port))
	if code != exitcode.Success {
		t.Fatalf("echo exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	var got echoResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Status != "success" {
		t.Errorf("status = %q, want success", got.Status)
	}
	if got.CalledAE != "RADX-SCP" {
		t.Errorf("called_ae = %q, want RADX-SCP", got.CalledAE)
	}
	if got.Port != port {
		t.Errorf("port = %d, want %d", got.Port, port)
	}
}

// TestEchoUnreachablePeerExits4 is the network regression: a connection to a port with no
// listener is refused, and echo exits 4 (network error), never 0.
func TestEchoUnreachablePeerExits4(t *testing.T) {
	// Bind then immediately close a listener to obtain a port that is guaranteed free, so the
	// dial is refused rather than racing a real service.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	_, _, code := runRadx(t, "echo", "--format", "json", "--timeout", "2s",
		"127.0.0.1", strconv.Itoa(port))
	if code != exitcode.NetworkError {
		t.Fatalf("echo to a closed port exit = %d, want %d (network error)", code, exitcode.NetworkError)
	}
}

// TestEchoNonSuccessStatusExits4 confirms a peer that answers with a non-success C-ECHO status
// makes echo exit 4 (the conversation succeeded; the peer said no), never 0.
func TestEchoNonSuccessStatusExits4(t *testing.T) {
	host, port := startEchoServer(t, dimse.NewStatus(0x0122, dimse.ServiceClassVerification)) // SOP Class Not Supported

	stdout, _, code := runRadx(t, "echo", "--format", "json",
		"--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.NetworkError {
		t.Fatalf("echo with a non-success status exit = %d, want %d\nstdout=%q", code, exitcode.NetworkError, stdout)
	}
	// The machine output still carries the failure outcome so a json consumer sees it.
	var got echoResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Status != "failure" {
		t.Errorf("status = %q, want failure", got.Status)
	}
}

// TestEchoRejectsCSVFormat confirms echo treats --format csv as a usage error (exit 2): echo
// is not a tabular command.
func TestEchoRejectsCSVFormat(t *testing.T) {
	host, port := startEchoServer(t, dimse.StatusEchoSuccess)
	_, _, code := runRadx(t, "echo", "--format", "csv", "--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.UsageError {
		t.Fatalf("echo --format csv exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}

// TestEchoOverLongCalledAEIsUsageError confirms a --called-ae over the 16-character DICOM limit
// is a usage-class fault (exit 2), not a parse error (3): the fault is in the invocation, not in
// any DICOM input. The over-long title is rejected before any dial, so no server is needed.
func TestEchoOverLongCalledAEIsUsageError(t *testing.T) {
	_, _, code := runRadx(t, "echo", "--format", "json",
		"--called-ae", "THIS-AE-TITLE-IS-WAY-TOO-LONG",
		"127.0.0.1", "104")
	if code != exitcode.UsageError {
		t.Fatalf("echo with an over-long --called-ae exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}
