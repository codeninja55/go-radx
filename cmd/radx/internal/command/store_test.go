package command

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// captureStoreHandler is an in-process Storage SCP that records the SOP Instance UIDs it receives
// and answers each C-STORE with a status keyed by a fail predicate, so a test can force a single
// transfer to fail and assert the batch exits non-zero.
type captureStoreHandler struct {
	failInstance string
}

func (h *captureStoreHandler) Store(_ context.Context, ds *dicom.DataSet, _ dimse.OpInfo) dimse.Status {
	instance, _ := ds.GetString(dicom.TagSOPInstanceUID)
	if h.failInstance != "" && instance == h.failInstance {
		// 0xA700 Out of Resources: a Failure-category C-STORE status the SCU must not read as success.
		return dimse.NewStatus(0xA700, dimse.ServiceClassStorage)
	}
	return dimse.StatusStoreSuccess
}

// startStorageServer runs a Storage SCP on loopback and returns its host and port. The handler's
// failInstance forces that one instance to a failure status.
func startStorageServer(t *testing.T, failInstance string) (host string, port int) {
	t.Helper()
	ae, err := dimse.NewAE(dimse.AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := dimse.NewServer(ae, dimse.StorageContexts(), &captureStoreHandler{failInstance: failInstance})

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("storage SCP did not bind within the deadline")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})

	tcp := srv.Addr().(*net.TCPAddr)
	return "127.0.0.1", tcp.Port
}

// countingStoreHandler is a Storage SCP that counts every C-STORE it answers (always success), so a
// test can assert all assigned objects were sent over the held worker associations.
type countingStoreHandler struct {
	stored atomic.Int64
}

func (h *countingStoreHandler) Store(_ context.Context, _ *dicom.DataSet, _ dimse.OpInfo) dimse.Status {
	h.stored.Add(1)
	return dimse.StatusStoreSuccess
}

// startCountingStorageServer runs a Storage SCP on loopback that counts both the associations it
// negotiates (via the association authorizer, which fires once per inbound association) and the
// C-STORE operations it answers. It returns the host, port, and the two counters so a test can assert
// associations track the worker count rather than the file count.
func startCountingStorageServer(t *testing.T) (host string, port int, associations, stored *atomic.Int64) {
	t.Helper()
	ae, err := dimse.NewAE(dimse.AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	handler := &countingStoreHandler{}
	var assocCount atomic.Int64
	srv := dimse.NewServer(ae, dimse.StorageContexts(), handler,
		dimse.WithAssociationAuthorizer(func(string, net.Addr) error {
			assocCount.Add(1)
			return nil
		}),
	)

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("storage SCP did not bind within the deadline")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-served
	})

	tcp := srv.Addr().(*net.TCPAddr)
	return "127.0.0.1", tcp.Port, &assocCount, &handler.stored
}

// writeStorableDICOM writes a synthetic CT-class Part-10 file carrying the Study/Series/SOP UIDs a
// Storage SCP needs, with a caller-chosen SOP Instance UID so a test can target it for failure.
func writeStorableDICOM(t *testing.T, dir, sopInstanceUID string) string {
	t.Helper()
	return writeStorableDICOMNamed(t, dir, sopInstanceUID, strings.ReplaceAll(sopInstanceUID, ".", "_")+".dcm")
}

// writeStorableDICOMNamed is writeStorableDICOM with a caller-chosen file name, so a test can write
// two distinct instances that share a base name in different directories (the --output-dir collision
// case).
func writeStorableDICOMNamed(t *testing.T, dir, sopInstanceUID, name string) string {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	ds.SetString(dicom.TagSOPInstanceUID, sopInstanceUID)
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3.4.5.1")
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4.5.2")
	ds.SetString(dicom.TagModality, "CT")

	path := filepath.Join(dir, name)
	if err := ds.WriteFile(path, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write storable DICOM: %v", err)
	}
	return path
}

// TestStoreSuccessGolden is the success golden: two objects stored to an in-process SCP yield two
// success result lines plus a success summary, all clean JSON Lines, and exit 0.
func TestStoreSuccessGolden(t *testing.T) {
	host, port := startStorageServer(t, "")
	dir := t.TempDir()
	a := writeStorableDICOM(t, dir, "1.2.3.4.5.10")
	b := writeStorableDICOM(t, dir, "1.2.3.4.5.11")

	stdout, stderr, code := runRadx(t, "store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port),
		"--called-ae", "RADX-SCP", "--workers", "2", a, b)
	if code != exitcode.Success {
		t.Fatalf("store exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	lines := nonEmptyLines(stdout)
	if len(lines) != 3 {
		t.Fatalf("want 2 result lines + 1 summary, got %d:\n%s", len(lines), stdout)
	}
	var summary storeSummary
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &summary); err != nil {
		t.Fatalf("summary line is not valid JSON: %v\nline=%q", err, lines[len(lines)-1])
	}
	if summary.Status != "success" || summary.Succeeded != 2 || summary.Failed != 0 {
		t.Errorf("summary = %+v, want success 2/2", summary)
	}
}

// TestStoreReusesOneAssociationPerWorker is the association-amortisation regression: storing many
// objects with a small worker pool must negotiate roughly one association per worker, NOT one per
// object. Re-associating per file is heavy ACSE overhead and risks a PACS's association-rate limits,
// so each worker holds one association for its whole slice of work. With 12 objects over 3 workers the
// SCP must see exactly 3 associations (each worker opens one and keeps it) and all 12 C-STOREs.
func TestStoreReusesOneAssociationPerWorker(t *testing.T) {
	host, port, associations, stored := startCountingStorageServer(t)
	dir := t.TempDir()
	const fileCount = 12
	const workers = 3
	files := make([]string, 0, fileCount)
	for i := 0; i < fileCount; i++ {
		files = append(files, writeStorableDICOM(t, dir, "1.2.3.4.6."+strconv.Itoa(i)))
	}

	args := append([]string{"store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port),
		"--called-ae", "RADX-SCP", "--workers", strconv.Itoa(workers)}, files...)
	stdout, stderr, code := runRadx(t, args...)
	if code != exitcode.Success {
		t.Fatalf("store exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	if got := stored.Load(); got != fileCount {
		t.Errorf("C-STOREs answered = %d, want %d (every object must be sent)", got, fileCount)
	}
	// The load-bearing assertion: associations track the worker count, not the file count. A regression
	// to per-file association would make this equal fileCount (12).
	if got := associations.Load(); got != workers {
		t.Errorf("associations negotiated = %d, want %d (one per worker, not one per object)", got, workers)
	}
}

// TestStoreParseErrorKeepsWorkerAssociation confirms a per-file parse error is recorded without
// tearing down the worker's held association: a single malformed object must not cost a re-association
// nor abort the worker's remaining good files. With one worker and a truncated file among good ones,
// the SCP must see exactly one association (the parse error skipped a file, never the connection), the
// good files still succeed, and the run exits non-zero for the recorded parse failure.
func TestStoreParseErrorKeepsWorkerAssociation(t *testing.T) {
	host, port, associations, stored := startCountingStorageServer(t)
	dir := t.TempDir()
	good1 := writeStorableDICOM(t, dir, "1.2.3.4.7.1")
	good2 := writeStorableDICOM(t, dir, "1.2.3.4.7.2")

	full, err := os.ReadFile(good1)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	truncPath := filepath.Join(dir, "truncated.dcm")
	if err := os.WriteFile(truncPath, full[:len(full)-8], 0o600); err != nil {
		t.Fatalf("write truncated fixture: %v", err)
	}

	// One worker so all three files share a single association; the truncated file is in the middle so a
	// torn-down association would also lose the trailing good file. --continue-on-error keeps the batch
	// going past the recorded parse failure.
	stdout, _, code := runRadx(t, "store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP",
		"--workers", "1", "--continue-on-error", good1, truncPath, good2)
	if code == exitcode.Success {
		t.Fatalf("store with a truncated file exited 0; want non-zero\nstdout=%q", stdout)
	}

	if got := associations.Load(); got != 1 {
		t.Errorf("associations negotiated = %d, want 1 (a parse error must not tear down the worker's association)", got)
	}
	if got := stored.Load(); got != 2 {
		t.Errorf("C-STOREs answered = %d, want 2 (both good files sent over the surviving association)", got)
	}

	// Both good files succeed; the truncated file is recorded as a failure. The trailing summary line
	// has no "file" field, so it is skipped when tallying the per-file results.
	var succeeded, failed int
	for _, line := range nonEmptyLines(stdout) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("output line is not valid JSON: %v\nline=%q", err, line)
		}
		if _, isResult := fields["file"]; !isResult {
			continue // the trailing summary line
		}
		var r storeResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("result line is not a storeResult: %v\nline=%q", err, line)
		}
		switch r.Status {
		case "success":
			succeeded++
		case "failure":
			failed++
		}
	}
	if succeeded != 2 || failed != 1 {
		t.Errorf("results = %d succeeded, %d failed; want 2 succeeded 1 failed", succeeded, failed)
	}
}

// TestStorePartialFailureExitsNonZero is the load-bearing honest-failure regression: a single
// failed C-STORE makes the command exit non-zero even though the other object succeeded, so CI
// catches partial failure (docs/reference/cli.md store; closes RADX-003).
func TestStorePartialFailureExitsNonZero(t *testing.T) {
	const failUID = "1.2.3.4.5.21"
	host, port := startStorageServer(t, failUID)
	dir := t.TempDir()
	ok := writeStorableDICOM(t, dir, "1.2.3.4.5.20")
	bad := writeStorableDICOM(t, dir, failUID)

	stdout, _, code := runRadx(t, "store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port),
		"--called-ae", "RADX-SCP", "--continue-on-error", ok, bad)
	if code == exitcode.Success {
		t.Fatalf("store with a failed transfer exited 0; want non-zero\nstdout=%q", stdout)
	}

	var summary storeSummary
	lines := nonEmptyLines(stdout)
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &summary); err != nil {
		t.Fatalf("summary line is not valid JSON: %v", err)
	}
	if summary.Failed != 1 || summary.Succeeded != 1 {
		t.Errorf("summary = %+v, want 1 succeeded 1 failed", summary)
	}
	if summary.Status != "failure" {
		t.Errorf("summary status = %q, want failure", summary.Status)
	}
}

// TestStoreNoFilesIsUsageError confirms invoking store with no resolvable files is a usage error
// (exit 2), not a silent success.
func TestStoreNoFilesIsUsageError(t *testing.T) {
	dir := t.TempDir() // empty directory
	_, _, code := runRadx(t, "store", "--host", "127.0.0.1", "--port", "11112", "-R", dir)
	if code != exitcode.UsageError {
		t.Fatalf("store with no files exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}

// TestStoreTranscodeToEncapsulatedTargetFailsClosed confirms an encapsulated
// --transcode-to target fails closed (exit 2): the store transport encodes each
// dataset in the negotiated uncompressed transfer syntax, so a compressed target
// cannot be honoured on the wire and is never silently downgraded (RADX-011).
func TestStoreTranscodeToEncapsulatedTargetFailsClosed(t *testing.T) {
	dir := t.TempDir()
	f := writeStorableDICOM(t, dir, "1.2.3.4.5.30")
	_, _, code := runRadx(t, "store", "--host", "127.0.0.1", "--port", "11112",
		"--transcode-to", "1.2.840.10008.1.2.5", f)
	if code != exitcode.UsageError {
		t.Fatalf("store --transcode-to exit = %d, want %d (usage error, not a silent passthrough)", code, exitcode.UsageError)
	}
}

// TestStoreMissingFileExits5 confirms a missing input is a file-I/O failure (exit 5), not a usage
// error: store must preserve the underlying os/fs error so exitcode.Classify routes a missing file
// to FileIOError rather than collapsing it into UsageError. The exit-code taxonomy is the contract.
func TestStoreMissingFileExits5(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.dcm")
	_, _, code := runRadx(t, "store", "--format", "json",
		"--host", "127.0.0.1", "--port", "11112", "--called-ae", "RADX-SCP", missing)
	if code != exitcode.FileIOError {
		t.Fatalf("store of a missing file exit = %d, want %d (file-I/O failure, not usage)", code, exitcode.FileIOError)
	}
}

// TestStoreTruncatedFileExits3 confirms a malformed/truncated object is a parse failure (exit 3):
// store reads the file before opening any association, and a read/parse error must surface its real
// class rather than a flattened usage error.
func TestStoreTruncatedFileExits3(t *testing.T) {
	dir := t.TempDir()
	good := writeStorableDICOM(t, dir, "1.2.3.4.5.40")
	full, err := os.ReadFile(good)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// Cut well inside the main dataset (past the preamble/magic and file-meta group) so the reader
	// fails mid-element, not at a clean record boundary.
	truncPath := filepath.Join(dir, "truncated.dcm")
	if err := os.WriteFile(truncPath, full[:len(full)-8], 0o600); err != nil {
		t.Fatalf("write truncated fixture: %v", err)
	}

	_, _, code := runRadx(t, "store", "--format", "json",
		"--host", "127.0.0.1", "--port", "11112", "--called-ae", "RADX-SCP", truncPath)
	if code != exitcode.ParseError {
		t.Fatalf("store of a truncated file exit = %d, want %d (parse failure, not usage)", code, exitcode.ParseError)
	}
}

// TestStoreUnreachablePeerExits4 confirms a refused/unreachable peer is a network failure (exit 4):
// the file reads fine, the association cannot be opened, and the dimse error must classify to
// NetworkError rather than a usage error. The port is bound and immediately closed so the connection
// is refused deterministically.
func TestStoreUnreachablePeerExits4(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close() // close so the port refuses connections deterministically

	dir := t.TempDir()
	f := writeStorableDICOM(t, dir, "1.2.3.4.5.41")
	_, _, code := runRadx(t, "store", "--format", "json", "--timeout", "5s",
		"--host", "127.0.0.1", "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP", f)
	if code != exitcode.NetworkError {
		t.Fatalf("store against a refused peer exit = %d, want %d (network failure, not usage)", code, exitcode.NetworkError)
	}
}

// TestStoreNonSuccessStatusExits4 confirms a non-success C-STORE terminal status is a network
// failure (exit 4): the conversation reached the peer and the peer answered with a Failure-category
// status, which store promotes to a *exitcode.StatusError so an operator branches on a peer "no" the
// same way regardless of where it surfaced.
func TestStoreNonSuccessStatusExits4(t *testing.T) {
	const failUID = "1.2.3.4.5.42"
	host, port := startStorageServer(t, failUID)
	dir := t.TempDir()
	bad := writeStorableDICOM(t, dir, failUID)

	_, _, code := runRadx(t, "store", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP", bad)
	if code != exitcode.NetworkError {
		t.Fatalf("store with a non-success C-STORE status exit = %d, want %d (network failure)", code, exitcode.NetworkError)
	}
}

// nonEmptyLines splits s into its non-empty trimmed lines, the JSON Lines a streaming command
// emits.
func nonEmptyLines(s string) []string {
	var out []string
	for line := range strings.SplitSeq(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
