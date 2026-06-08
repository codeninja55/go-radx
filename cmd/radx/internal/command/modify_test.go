package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
)

// TestModifyActuallyWritesChange is the load-bearing honest-failure regression: an inserted tag is
// genuinely written to the output file and a deleted tag is genuinely gone, proven by reading the
// modified file back from disk. The prototype's modify logged "Would insert" and wrote unchanged
// files while reporting success (RADX-001); this asserts the bytes on disk changed.
func TestModifyActuallyWritesChange(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.600.10")
	outDir := filepath.Join(t.TempDir(), "modified")

	stdout, stderr, code := runRadx(t, "modify", "--format", "json",
		"--output-dir", outDir,
		"--insert", "PatientID=ANON-001",
		"--delete", "Modality",
		src)
	if code != exitcode.Success {
		t.Fatalf("modify exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	var got modifyResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Status != "success" {
		t.Fatalf("status = %q, want success", got.Status)
	}

	// Read the WRITTEN file back from disk and confirm the dataset actually changed.
	out := filepath.Join(outDir, filepath.Base(src))
	f, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read modified file: %v", err)
	}
	pid, ok := f.DataSet.GetString(dicom.TagPatientID)
	if !ok || pid != "ANON-001" {
		t.Errorf("PatientID in written file = %q ok=%v, want ANON-001 (the insert did not land)", pid, ok)
	}
	if _, present := f.DataSet.Get(dicom.TagModality); present {
		t.Errorf("Modality is still present in the written file (the delete did not land)")
	}

	// The original must be untouched (output-dir mode, not in-place).
	orig, err := dicom.ReadFile(src)
	if err != nil {
		t.Fatalf("read original file: %v", err)
	}
	if _, present := orig.DataSet.Get(dicom.TagModality); !present {
		t.Errorf("the original file was mutated; --output-dir must not touch the source")
	}
}

// TestModifyRegeneratesUIDs confirms --regenerate-all-uids writes fresh, conformant Study/Series/
// SOP UIDs that differ from the originals, and that the SOP Instance UID is mirrored into the file
// meta so the Part 10 header and the dataset agree (RADX-002).
func TestModifyRegeneratesUIDs(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.600.20")
	outDir := filepath.Join(t.TempDir(), "rekeyed")

	_, stderr, code := runRadx(t, "modify", "--format", "json",
		"--output-dir", outDir, "--regenerate-all-uids", src)
	if code != exitcode.Success {
		t.Fatalf("modify --regenerate-all-uids exit = %d, want %d\nstderr=%q", code, exitcode.Success, stderr)
	}

	out := filepath.Join(outDir, filepath.Base(src))
	f, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read rekeyed file: %v", err)
	}
	study, _ := f.DataSet.GetString(dicom.TagStudyInstanceUID)
	series, _ := f.DataSet.GetString(dicom.TagSeriesInstanceUID)
	instance, _ := f.DataSet.GetString(dicom.TagSOPInstanceUID)
	if study == "1.2.3.4.5.1" || series == "1.2.3.4.5.2" || instance == "1.2.3.4.600.20" {
		t.Errorf("a UID was not regenerated: study=%q series=%q instance=%q", study, series, instance)
	}
	for name, uid := range map[string]string{"study": study, "series": series, "instance": instance} {
		if err := dicom.UID(uid).Validate(); err != nil {
			t.Errorf("regenerated %s UID %q is not conformant: %v", name, uid, err)
		}
	}
	if string(f.Meta.MediaStorageSOPInstanceUID) != instance {
		t.Errorf("file meta MediaStorageSOPInstanceUID = %q, want %q (must mirror the dataset)",
			f.Meta.MediaStorageSOPInstanceUID, instance)
	}
}

// TestModifyUnwritableTargetErrorsWritingNothing is the second honest-failure regression: when the
// output destination cannot be created, modify exits non-zero and leaves no output file — it never
// reports success on an edit that did not land (the fail-closed rule).
func TestModifyUnwritableTargetErrorsWritingNothing(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.600.30")

	// Point --output-dir at a path under a read-only parent so the file write fails. Create a
	// directory, make it read-only, and target a child path inside it.
	parent := t.TempDir()
	roDir := filepath.Join(parent, "readonly")
	if err := os.Mkdir(roDir, 0o500); err != nil {
		t.Fatalf("mkdir read-only dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(roDir, 0o700) })

	// The output-dir exists (it is roDir) but is not writable, so the per-file write fails.
	stdout, _, code := runRadx(t, "modify", "--format", "json",
		"--output-dir", roDir, "--insert", "PatientID=ANON-002", src)
	if code == exitcode.Success {
		t.Fatalf("modify into a read-only dir exited 0; want non-zero\nstdout=%q", stdout)
	}

	// No output file may have been created.
	out := filepath.Join(roDir, filepath.Base(src))
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a partial output file was written despite the failure: %v", err)
	}
}

// TestModifyRequiresOutputMode confirms modify rejects an invocation that names neither
// --output-dir nor --in-place (and both) as a usage error.
func TestModifyRequiresOutputMode(t *testing.T) {
	src := writeStorableDICOM(t, t.TempDir(), "1.2.3.4.600.40")
	_, _, code := runRadx(t, "modify", "--insert", "PatientID=X", src)
	if code != exitcode.UsageError {
		t.Fatalf("modify with no output mode exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
}
