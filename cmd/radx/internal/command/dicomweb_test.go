package command

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
)

// memBackend is an in-memory DICOMweb backend implementing the store, retrieve, and query
// interfaces, so the CLI's wado/stow/qido commands can be exercised against an httptest server
// without an external PACS. It holds only synthetic, non-PHI fixtures.
type memBackend struct {
	mu        sync.Mutex
	instances map[string]*dicom.DataSet // keyed by SOP Instance UID
}

func newMemBackend() *memBackend {
	return &memBackend{instances: make(map[string]*dicom.DataSet)}
}

// Store records an instance by its SOP Instance UID.
func (m *memBackend) Store(_ context.Context, ds *dicom.DataSet) error {
	instance, _ := ds.GetString(dicom.TagSOPInstanceUID)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instances[instance] = ds
	return nil
}

// RetrieveInstance returns the instance addressed by the path's SOP Instance UID.
func (m *memBackend) RetrieveInstance(_ context.Context, p dicomweb.ResourcePath) (*dicom.DataSet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ds, ok := m.instances[string(p.Instance)]; ok {
		return ds, nil
	}
	return nil, dicomweb.ErrInvalidResource
}

// Query returns the studies matching the request (here, every stored instance projected to its
// study identity — sufficient for the CLI's qido happy path).
func (m *memBackend) Query(_ context.Context, _ dicomweb.QueryRequest) ([]*dicom.DataSet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*dicom.DataSet, 0, len(m.instances))
	for _, ds := range m.instances {
		out = append(out, ds)
	}
	return out, nil
}

// startDICOMwebServer mounts the in-memory backend on an httptest server and returns its base URL
// and the backend (so a test can assert stored instances).
func startDICOMwebServer(t *testing.T) (baseURL string, backend *memBackend) {
	t.Helper()
	backend = newMemBackend()
	srv, err := dicomweb.NewServer(
		dicomweb.WithStoreBackend(backend),
		dicomweb.WithRetrieveBackend(backend),
		dicomweb.WithQueryBackend(backend),
	)
	if err != nil {
		t.Fatalf("dicomweb.NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts.URL, backend
}

// webInstance builds a synthetic instance for the DICOMweb fixtures.
func webInstance(study, series, instance string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7") // Secondary Capture
	ds.SetString(dicom.TagSOPInstanceUID, instance)
	ds.SetString(dicom.TagStudyInstanceUID, study)
	ds.SetString(dicom.TagSeriesInstanceUID, series)
	ds.SetString(dicom.TagModality, "OT")
	return ds
}

// TestDICOMwebStowThenWado is the store-then-retrieve round trip: a file is stored over STOW-RS,
// then the same instance is retrieved over WADO-RS and written to disk.
func TestDICOMwebStowThenWado(t *testing.T) {
	url, backend := startDICOMwebServer(t)
	dir := t.TempDir()
	src := writeStorableDICOM(t, dir, "1.2.900.10") // CT-class storable fixture

	// STOW-RS.
	stdout, stderr, code := runRadx(t, "dicomweb", "stow", "--format", "json", "--url", url, src)
	if code != exitcode.Success {
		t.Fatalf("stow exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	var stow stowResult
	if err := json.Unmarshal([]byte(stdout), &stow); err != nil {
		t.Fatalf("stow stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if stow.Accepted != 1 || stow.Failed != 0 {
		t.Errorf("stow result = %+v, want 1 accepted 0 failed", stow)
	}

	backend.mu.Lock()
	_, stored := backend.instances["1.2.900.10"]
	backend.mu.Unlock()
	if !stored {
		t.Fatalf("instance was not stored on the origin")
	}

	// WADO-RS instance retrieve to disk.
	out := filepath.Join(t.TempDir(), "retrieved")
	wadoStdout, _, wadoCode := runRadx(t, "dicomweb", "wado", "--format", "json", "--url", url,
		"--study", "1.2.3.4.5.1", "--series", "1.2.3.4.5.2", "--instance", "1.2.900.10",
		"--output-dir", out)
	if wadoCode != exitcode.Success {
		t.Fatalf("wado exit = %d, want %d\nstdout=%q", wadoCode, exitcode.Success, wadoStdout)
	}
	var wado wadoResult
	if err := json.Unmarshal([]byte(wadoStdout), &wado); err != nil {
		t.Fatalf("wado stdout is not valid JSON: %v", err)
	}
	if wado.Retrieved != 1 {
		t.Errorf("wado retrieved = %d, want 1", wado.Retrieved)
	}
	dest := filepath.Join(out, "1.2.3.4.5.1", "1.2.3.4.5.2", "1.2.900.10.dcm")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected retrieved instance at %s: %v", dest, err)
	}
}

// TestDICOMwebQidoStreamsMatches confirms a QIDO-RS study search streams one JSON Line per match.
func TestDICOMwebQidoStreamsMatches(t *testing.T) {
	url, backend := startDICOMwebServer(t)
	_ = backend.Store(context.Background(), webInstance("1.2.901.1", "1.2.901.2", "1.2.901.3"))

	stdout, stderr, code := runRadx(t, "dicomweb", "qido", "--format", "json", "--url", url,
		"--level", "studies")
	if code != exitcode.Success {
		t.Fatalf("qido exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	if len(nonEmptyLines(stdout)) == 0 {
		t.Errorf("qido returned no matches:\n%s", stdout)
	}
}

// TestDICOMwebStowScopedToStudyTargetsStudyPath confirms --study routes the STOW-RS request to the
// study-scoped /studies/{study} endpoint, not the root /studies target, so the origin can constrain
// the store to that StudyInstanceUID. The fixture's StudyInstanceUID is 1.2.3.4.5.1.
func TestDICOMwebStowScopedToStudyTargetsStudyPath(t *testing.T) {
	const study = "1.2.3.4.5.1"
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/dicom+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}")) // complete store response: nothing failed
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	src := writeStorableDICOM(t, dir, "1.2.900.20")

	stdout, stderr, code := runRadx(t, "dicomweb", "stow", "--format", "json",
		"--url", ts.URL, "--study", study, src)
	if code != exitcode.Success {
		t.Fatalf("stow exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	if want := "/studies/" + study; gotPath != want {
		t.Errorf("STOW-RS request path = %q, want %q (scoped --study must target the study path)", gotPath, want)
	}
}

// TestDICOMwebStowWithoutStudyTargetsRoot confirms that, absent --study, STOW-RS still posts to the
// unconstrained root /studies target.
func TestDICOMwebStowWithoutStudyTargetsRoot(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/dicom+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	src := writeStorableDICOM(t, dir, "1.2.900.21")

	stdout, stderr, code := runRadx(t, "dicomweb", "stow", "--format", "json", "--url", ts.URL, src)
	if code != exitcode.Success {
		t.Fatalf("stow exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	if gotPath != "/studies" {
		t.Errorf("STOW-RS request path = %q, want /studies (no --study posts to the root target)", gotPath)
	}
}

// TestDICOMwebQidoCSVRendersMatchedValues is the load-bearing regression for the QIDO CSV emitter: a
// --match key=value must render the matched column populated, not an empty cell from parsing the tag
// out of the unstripped "key=value" string. PatientID here is a synthetic sentinel, never real PHI.
func TestDICOMwebQidoCSVRendersMatchedValues(t *testing.T) {
	const sentinel = "QIDO-SENTINEL-001"
	url, backend := startDICOMwebServer(t)
	ds := webInstance("1.2.902.1", "1.2.902.2", "1.2.902.3")
	ds.SetString(dicom.TagPatientID, sentinel)
	_ = backend.Store(context.Background(), ds)

	stdout, stderr, code := runRadx(t, "dicomweb", "qido", "--format", "csv", "--url", url,
		"--level", "studies", "--match", "PatientID="+sentinel)
	if code != exitcode.Success {
		t.Fatalf("qido exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	records, err := csv.NewReader(strings.NewReader(stdout)).ReadAll()
	if err != nil {
		t.Fatalf("qido csv is not parseable: %v\nstdout=%q", err, stdout)
	}
	if len(records) < 2 {
		t.Fatalf("want a header and at least one match row, got %d records:\n%s", len(records), stdout)
	}
	header := records[0]
	col := -1
	for i, name := range header {
		if name == "PatientID" {
			col = i
		}
	}
	if col < 0 {
		t.Fatalf("PatientID column missing from header %v", header)
	}
	if got := records[1][col]; got != sentinel {
		t.Errorf("PatientID cell = %q, want %q (CSV emitter dropped the matched value)", got, sentinel)
	}
}

// TestDICOMwebQidoNormalizesParenthesizedTag is the load-bearing regression for finding 4: a
// validated --match key in the parenthesised "(GGGG,EEEE)" form must reach the origin as the
// eight-hex GGGGEEEE tag the QIDO-RS wire parser accepts — identical to what the keyword form
// produces — not the parenthesised text the parser rejects. It captures the request query against a
// raw httptest origin and asserts the normalised parameter name.
func TestDICOMwebQidoNormalizesParenthesizedTag(t *testing.T) {
	var gotQuery url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/dicom+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]")) // empty result set
	}))
	t.Cleanup(ts.Close)

	// PatientID is (0010,0020) -> 00100020; Modality is (0008,0060) -> 00080060. The keyword form
	// resolves to the same eight-hex tag, so the match key and includefield must equal these.
	stdout, stderr, code := runRadx(t, "dicomweb", "qido", "--format", "json", "--url", ts.URL,
		"--level", "studies", "--match", "(0010,0020)=SENTINEL", "--include", "(0008,0060)")
	if code != exitcode.Success {
		t.Fatalf("qido exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	if got := gotQuery.Get("00100020"); got != "SENTINEL" {
		t.Errorf("match param 00100020 = %q, want SENTINEL; full query = %v", got, gotQuery)
	}
	if _, ok := gotQuery["(0010,0020)"]; ok {
		t.Errorf("the un-normalized parenthesized match key reached the wire: %v", gotQuery)
	}
	inc := gotQuery["includefield"]
	if len(inc) != 1 || inc[0] != "00080060" {
		t.Errorf("includefield = %v, want [00080060] (normalized eight-hex tag)", inc)
	}
}

// TestDICOMwebQidoParenthesizedTagAcceptedByOrigin proves end-to-end that the normalised request is
// one the real QIDO-RS parser accepts: a parenthesised --match key against the in-process dicomweb
// server returns a clean success (exit 0), where the un-normalized form would be rejected as an
// unknown attribute (HTTP 400, exit 4).
func TestDICOMwebQidoParenthesizedTagAcceptedByOrigin(t *testing.T) {
	const sentinel = "QIDO-PAREN-001"
	url, backend := startDICOMwebServer(t)
	ds := webInstance("1.2.903.1", "1.2.903.2", "1.2.903.3")
	ds.SetString(dicom.TagPatientID, sentinel)
	_ = backend.Store(context.Background(), ds)

	stdout, stderr, code := runRadx(t, "dicomweb", "qido", "--format", "json", "--url", url,
		"--level", "studies", "--match", "(0010,0020)="+sentinel)
	if code != exitcode.Success {
		t.Fatalf("qido with a parenthesized --match exit = %d, want %d (origin accepted the request)\nstdout=%q\nstderr=%q",
			code, exitcode.Success, stdout, stderr)
	}
}

// TestDICOMwebMissingURLIsUsageError confirms an absent --url is a usage error.
func TestDICOMwebMissingURLIsUsageError(t *testing.T) {
	// Kong marks --url required, so the parse fails with a usage exit before Run.
	_, _, code := runRadx(t, "dicomweb", "qido", "--level", "studies")
	if code != exitcode.UsageError {
		t.Fatalf("qido with no --url exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}
