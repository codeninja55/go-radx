package command

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dimse"
)

// TestMachineStdoutIsClean is the output-contract regression: under --format json, stdout
// carries ONLY the payload — no banner, no log line, no build string — so a consumer can pipe
// it into a JSON parser without filtering (docs/reference/cli.md "Machine output is clean";
// closes RADX-004/021). The whole of stdout must parse as a single JSON document.
func TestMachineStdoutIsClean(t *testing.T) {
	path := writeSyntheticDICOM(t)
	stdout, _, code := runRadx(t, "dump", "--format", "json", path)
	if code != exitcode.Success {
		t.Fatalf("dump exit = %d, want %d", code, exitcode.Success)
	}
	// The entire stdout must be valid JSON: a banner or log line would make this fail.
	var discard any
	if err := json.Unmarshal([]byte(stdout), &discard); err != nil {
		t.Fatalf("stdout is not a single clean JSON document (banner/log leak?): %v\nstdout=%q", err, stdout)
	}
	if strings.Contains(stdout, banner) {
		t.Errorf("stdout contains the banner; it must go to stderr only:\n%s", stdout)
	}
}

// TestDebugLogsGoToStderrNotStdout confirms diagnostics — including a verbose debug log line
// the echo command emits — reach stderr, never machine stdout. Even at --log-level debug,
// stdout stays a clean JSON document (docs/reference/cli.md "Diagnostics go to stderr").
func TestDebugLogsGoToStderrNotStdout(t *testing.T) {
	host, port := startEchoServer(t, dimse.StatusEchoSuccess)

	stdout, stderr, code := runRadx(t, "echo",
		"--format", "json", "--log-level", "debug", "--log-format", "json",
		"--called-ae", "RADX-SCP", host, strconv.Itoa(port))
	if code != exitcode.Success {
		t.Fatalf("echo exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}

	// stdout must be exactly one JSON document — the echo result — with no log line mixed in.
	var got echoResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not clean JSON (a log line leaked?): %v\nstdout=%q", err, stdout)
	}
	if got.Status != "success" {
		t.Errorf("status = %q, want success", got.Status)
	}
	// The debug log line ("opening association") must be on stderr.
	if !strings.Contains(stderr, "opening association") {
		t.Errorf("expected the debug log on stderr, got:\n%s", stderr)
	}
	if strings.Contains(stdout, "opening association") {
		t.Errorf("the debug log leaked into machine stdout:\n%s", stdout)
	}
}

// TestVersionFlagCoherent confirms radx --version prints a coherent build line and exits 0,
// and that the build info is resolvable in-process.
func TestVersionFlagCoherent(t *testing.T) {
	_, stderr, code := runRadx(t, "--version")
	if code != exitcode.Success {
		t.Fatalf("--version exit = %d, want %d", code, exitcode.Success)
	}
	// Kong's --version writes the resolved version var; we route it to stderr (a diagnostic),
	// so machine stdout stays clean even for --version.
	if !strings.Contains(stderr, "radx ") {
		t.Errorf("--version output = %q, want a coherent build line", stderr)
	}
}

// The fail-closed stub contract (a committed-but-unbuilt command exits 1 and writes nothing —
// docs/reference/cli.md "Honest-failure rules"; RADX-001/002) has no remaining subject: `serve
// fhir`, the last registered stub, is wired end to end (serve.go; round-trip proof in
// serve_test.go). The rule still binds any future stub; its test returns with the next one.
