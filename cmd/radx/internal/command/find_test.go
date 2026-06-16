package command

import (
	"context"
	"encoding/json"
	"iter"
	"net"
	"strconv"
	"sync"
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

// cannedWorklistHandler answers a Modality Worklist C-FIND by yielding one scheduled-step match per
// accession number then a terminal Success, recording the identifier the SCU sent so the test can
// assert the worklist query shape (SPS sequence skeleton, no Query/Retrieve Level).
type cannedWorklistHandler struct {
	accessions []string

	mu    sync.Mutex
	query *dicom.DataSet
}

func (h *cannedWorklistHandler) Find(_ context.Context, query *dicom.DataSet, _ dimse.QueryLevel, _ dimse.OpInfo) iter.Seq2[dimse.Status, *dicom.DataSet] {
	h.mu.Lock()
	h.query = query
	h.mu.Unlock()
	return func(yield func(dimse.Status, *dicom.DataSet) bool) {
		for _, acc := range h.accessions {
			ds := dicom.NewDataSet()
			ds.SetString(dicom.TagAccessionNumber, acc)
			if !yield(dimse.StatusWorklistPending, ds) {
				return
			}
		}
	}
}

func (h *cannedWorklistHandler) receivedQuery() *dicom.DataSet {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.query
}

// startWorklistServer runs a Modality Worklist C-FIND SCP on loopback returning one scheduled step
// per accession number. It mirrors startFindServer but negotiates the worklist abstract syntax.
func startWorklistServer(t *testing.T, h *cannedWorklistHandler) (host string, port int) {
	t.Helper()
	ae, err := dimse.NewAE(dimse.AETitle("MWLSCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := dimse.NewServer(ae, dimse.BasicWorklistContexts(), h)

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("worklist SCP did not bind within the deadline")
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

// TestFindWorklistFlagStreamsScheduledSteps is the -W golden (dcmtk findscu -W): the worklist flag
// negotiates the Modality Worklist context, sends the SPS-sequence query skeleton with the --match
// keys and NO Query/Retrieve Level (the worklist model is flat, PS3.4 K.6.1.2.1), and streams one
// JSON Line per scheduled step.
func TestFindWorklistFlagStreamsScheduledSteps(t *testing.T) {
	want := []string{"ACC-1001", "ACC-1002"}
	handler := &cannedWorklistHandler{accessions: want}
	host, port := startWorklistServer(t, handler)

	stdout, stderr, code := runRadx(t, "find", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "MWLSCP",
		"-W", "--match", "AccessionNumber=")
	if code != exitcode.Success {
		t.Fatalf("find -W exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
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
		got[m.Attributes["0008,0050"]] = true // AccessionNumber
	}
	for _, acc := range want {
		if !got[acc] {
			t.Errorf("missing scheduled-step match %q in:\n%s", acc, stdout)
		}
	}

	// The sent identifier carries the worklist skeleton: the Scheduled Procedure Step Sequence
	// (0040,0100) universal item plus the --match key, and no Query/Retrieve Level (flat model).
	query := handler.receivedQuery()
	if query == nil {
		t.Fatal("worklist SCP recorded no query identifier")
	}
	if _, ok := query.Get(dicom.TagScheduledProcedureStepSequence); !ok {
		t.Error("worklist query has no Scheduled Procedure Step Sequence (0040,0100) skeleton")
	}
	if _, ok := query.Get(dicom.TagQueryRetrieveLevel); ok {
		t.Error("worklist query carries a Query/Retrieve Level (0008,0052); the worklist model is flat")
	}
	if _, ok := query.Get(dicom.TagAccessionNumber); !ok {
		t.Error("worklist query dropped the --match AccessionNumber return key")
	}
}

// TestFindWorklistRoutesSPSMatchKeysIntoSequence is the PS3.4 Table K.6-1 routing golden for
// find -W: an SPS requirement key (--match Modality=MR) must land INSIDE the Scheduled Procedure
// Step Sequence item — where an MWL SCP matches it — while a non-SPS key (--match PatientName=X)
// stays at the identifier's top level. A top-level Modality would never constrain the query.
func TestFindWorklistRoutesSPSMatchKeysIntoSequence(t *testing.T) {
	handler := &cannedWorklistHandler{accessions: []string{"ACC-2001"}}
	host, port := startWorklistServer(t, handler)

	stdout, stderr, code := runRadx(t, "find", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "MWLSCP",
		"-W", "--match", "Modality=MR", "--match", "PatientName=X")
	if code != exitcode.Success {
		t.Fatalf("find -W exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	query := handler.receivedQuery()
	if query == nil {
		t.Fatal("worklist SCP recorded no query identifier")
	}
	if _, topLevel := query.Get(dicom.TagModality); topLevel {
		t.Error("--match Modality landed at the identifier's top level; Table K.6-1 scopes it to the SPS item")
	}
	if name, ok := query.GetString(dicom.TagPatientName); !ok || name != "X" {
		t.Errorf("top-level PatientName = %q (ok=%v), want X at the top level", name, ok)
	}
	seq, ok := query.GetSequence(dicom.TagScheduledProcedureStepSequence)
	if !ok || seq.Len() != 1 {
		t.Fatalf("worklist identifier SPS sequence missing or not single-item (ok=%v)", ok)
	}
	for item := range seq.Items() {
		if modality, mok := item.DataSet.GetString(dicom.TagModality); !mok || modality != "MR" {
			t.Errorf("SPS item Modality = %q (ok=%v), want MR inside the sequence item", modality, mok)
		}
		if _, pok := item.DataSet.Get(dicom.TagPatientName); pok {
			t.Error("--match PatientName leaked into the SPS item; it is not a Table K.6-1 SPS key")
		}
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
