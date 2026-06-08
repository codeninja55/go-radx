package command

import (
	"context"
	"encoding/json"
	"iter"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// gettingHandler yields a fixed list of matched instances (each as a Pending with the instance
// dataset) then a terminal Success, so a C-GET test asserts the runtime C-STOREs each matched
// instance back to the requestor's sink.
type gettingHandler struct {
	instances []*dicom.DataSet
}

func (h *gettingHandler) Get(_ context.Context, _ *dicom.DataSet, _ dimse.QueryLevel, _ dimse.OpInfo) iter.Seq2[dimse.Status, *dicom.DataSet] {
	return func(yield func(dimse.Status, *dicom.DataSet) bool) {
		for _, ds := range h.instances {
			if !yield(dimse.StatusGetPending, ds) {
				return
			}
		}
		yield(dimse.StatusGetSuccess, nil)
	}
}

// storageRolesFromContexts grants the Storage SCP role for every Storage class in the validated
// preset, derived from StorageContexts so the test never duplicates the class list.
func storageRolesFromContexts() []dicom.SOPClassUID {
	contexts := dimse.StorageContexts()
	classes := make([]dicom.SOPClassUID, 0, len(contexts))
	for _, pc := range contexts {
		classes = append(classes, pc.AbstractSyntax)
	}
	return classes
}

// startGetServer runs a C-GET SCP on loopback yielding the given instances.
func startGetServer(t *testing.T, instances []*dicom.DataSet) (host string, port int) {
	t.Helper()
	ae, err := dimse.NewAE(dimse.AETitle("RADX-SCP"))
	if err != nil {
		t.Fatalf("NewAE: %v", err)
	}
	srv := dimse.NewServer(ae, dimse.QueryRetrieveWithStorageContexts(),
		&gettingHandler{instances: instances},
		dimse.WithGetStorageRoles(storageRolesFromContexts()...))

	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && srv.Addr() == nil {
		time.Sleep(time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("get SCP did not bind within the deadline")
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

// getInstance builds a synthetic MR instance the C-GET SCP yields and the SCU writes to disk.
func getInstance(sopInstanceUID string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.4") // MR Image Storage
	ds.SetString(dicom.TagSOPInstanceUID, sopInstanceUID)
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3.4.300.1")
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4.300.2")
	return ds
}

// TestGetRetrievesToDir is the C-GET golden: two instances are retrieved over the same association
// and written into --output-dir, the result reports a success terminal status and stored count,
// and the command exits 0.
func TestGetRetrievesToDir(t *testing.T) {
	instances := []*dicom.DataSet{getInstance("1.2.3.4.300.10"), getInstance("1.2.3.4.300.11")}
	host, port := startGetServer(t, instances)
	outDir := filepath.Join(t.TempDir(), "received")

	stdout, stderr, code := runRadx(t, "get", "--format", "json",
		"--host", host, "--port", strconv.Itoa(port), "--called-ae", "RADX-SCP",
		"--level", "SERIES", "--output-dir", outDir,
		"--match", "StudyInstanceUID=1.2.3.4.300.1", "--match", "SeriesInstanceUID=1.2.3.4.300.2")
	if code != exitcode.Success {
		t.Fatalf("get exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	var got retrieveResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Status != "success" {
		t.Errorf("status = %q, want success", got.Status)
	}
	if got.Stored != 2 {
		t.Errorf("stored = %d, want 2", got.Stored)
	}

	// The two instances must be on disk in the Study/Series/SOP layout.
	for _, uid := range []string{"1.2.3.4.300.10", "1.2.3.4.300.11"} {
		p := filepath.Join(outDir, "1.2.3.4.300.1", "1.2.3.4.300.2", uid+".dcm")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected retrieved file %s on disk: %v", p, err)
		}
	}
}
