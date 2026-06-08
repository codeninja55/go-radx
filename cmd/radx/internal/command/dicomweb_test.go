package command

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// TestDICOMwebMissingURLIsUsageError confirms an absent --url is a usage error.
func TestDICOMwebMissingURLIsUsageError(t *testing.T) {
	// Kong marks --url required, so the parse fails with a usage exit before Run.
	_, _, code := runRadx(t, "dicomweb", "qido", "--level", "studies")
	if code != exitcode.UsageError {
		t.Fatalf("qido with no --url exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}
