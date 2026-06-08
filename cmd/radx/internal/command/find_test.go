package command

import (
	"context"
	"encoding/json"
	"iter"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// cannedFindHandler answers a C-FIND by yielding a fixed set of matches then a terminal Success.
// Each match carries the synthetic StudyInstanceUID it represents, so a test can assert the
// streamed match output without any real patient data.
type cannedFindHandler struct {
	studyUIDs []string
}

func (h *cannedFindHandler) Find(_ context.Context, _ *dicom.DataSet, _ dimse.QueryLevel, _ dimse.OpInfo) iter.Seq2[dimse.Status, *dicom.DataSet] {
	return func(yield func(dimse.Status, *dicom.DataSet) bool) {
		for _, uid := range h.studyUIDs {
			ds := dicom.NewDataSet()
			ds.SetString(dicom.TagStudyInstanceUID, uid)
			ds.SetString(dicom.TagModality, "CT")
			if !yield(dimse.StatusFindPending, ds) {
				return
			}
		}
	}
}

// startFindServer runs a C-FIND SCP on loopback returning the given study UIDs as matches.
func startFindServer(t *testing.T, studyUIDs []string) (host string, port int) {
	t.Helper()
	ae, err := dimse.NewAE(dimse.AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := dimse.NewServer(ae, dimse.QueryRetrieveContexts(), &cannedFindHandler{studyUIDs: studyUIDs})

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("find SCP did not bind within the deadline")
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

// TestFindStreamsMatches is the streaming golden: a C-FIND against an in-process SCP streams one
// JSON Line per match, each carrying the study UID, and exits 0.
func TestFindStreamsMatches(t *testing.T) {
	want := []string{"1.2.3.4.100", "1.2.3.4.101"}
	host, port := startFindServer(t, want)

	stdout, stderr, code := runRadx(t, "find", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP",
		"--level", "STUDY", "--match", "StudyInstanceUID=", "--match", "ModalitiesInStudy=CT")
	if code != exitcode.Success {
		t.Fatalf("find exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	lines := nonEmptyLines(stdout)
	if len(lines) != len(want) {
		t.Fatalf("want %d match lines, got %d:\n%s", len(want), len(lines), stdout)
	}
	got := make(map[string]bool)
	for _, line := range lines {
		var m findMatch
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("match line is not valid JSON: %v\nline=%q", err, line)
		}
		if m.Status != "match" {
			t.Errorf("match status = %q, want match", m.Status)
		}
		got[m.Attributes["0020,000D"]] = true // StudyInstanceUID
	}
	for _, uid := range want {
		if !got[uid] {
			t.Errorf("missing study match %q in:\n%s", uid, stdout)
		}
	}
}

// TestFindCSVGolden confirms find emits RFC 4180 CSV with a header keyed by the requested match
// columns and one row per match.
func TestFindCSVGolden(t *testing.T) {
	host, port := startFindServer(t, []string{"1.2.3.4.200"})
	stdout, _, code := runRadx(t, "find", "--format", "csv",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP",
		"--level", "STUDY", "--match", "StudyInstanceUID=")
	if code != exitcode.Success {
		t.Fatalf("find --format csv exit = %d, want %d\nstdout=%q", code, exitcode.Success, stdout)
	}
	lines := nonEmptyLines(stdout)
	if len(lines) != 2 { // header + one match row
		t.Fatalf("want 1 header + 1 row, got %d:\n%s", len(lines), stdout)
	}
	if lines[0] != "status,StudyInstanceUID" {
		t.Errorf("header = %q, want status,StudyInstanceUID", lines[0])
	}
}

// TestFindBadMatchKeyIsUsageError confirms an unparseable --match key is a usage error (exit 2),
// rejected before any network call.
func TestFindBadMatchKeyIsUsageError(t *testing.T) {
	_, _, code := runRadx(t, "find", "--host", "127.0.0.1", "--port", "11112",
		"--match", "NotARealKeyword=1")
	if code != exitcode.UsageError {
		t.Fatalf("find with a bad match key exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}
