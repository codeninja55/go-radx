package command

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// TestModifyBatchPreservesStudySeriesGrouping is the reference-graph regression: re-keying a batch of
// files that share a Study (and Series) UID must map each distinct OLD grouping UID to ONE new UID
// for the whole run, so the study is not silently split into N unrelated objects. Two files sharing a
// StudyInstanceUID and SeriesInstanceUID, run together with --regenerate-study-uid and
// --regenerate-series-uid, must come out with the SAME new Study UID and the SAME new Series UID
// (each changed from the original), while their SOP Instance UIDs, when also regenerated, stay unique
// per file. Every minted UID must be conformant.
func TestModifyBatchPreservesStudySeriesGrouping(t *testing.T) {
	dir := t.TempDir()
	// writeStorableDICOM writes both files under Study 1.2.3.4.5.1 and Series 1.2.3.4.5.2, so they
	// share a study and series — the batch case the run-level UID map must keep grouped.
	srcA := writeStorableDICOM(t, dir, "1.2.3.4.700.1")
	srcB := writeStorableDICOM(t, dir, "1.2.3.4.700.2")
	outDir := filepath.Join(t.TempDir(), "rekeyed")

	stdout, stderr, code := runRadx(t, "modify", "--format", "json", "--output-dir", outDir,
		"--regenerate-study-uid", "--regenerate-series-uid", "--regenerate-instance-uid", srcA, srcB)
	if code != exitcode.Success {
		t.Fatalf("modify batch re-key exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	outA, errA := dicom.ReadFile(filepath.Join(outDir, filepath.Base(srcA)))
	if errA != nil {
		t.Fatalf("read rekeyed A: %v", errA)
	}
	outB, errB := dicom.ReadFile(filepath.Join(outDir, filepath.Base(srcB)))
	if errB != nil {
		t.Fatalf("read rekeyed B: %v", errB)
	}

	studyA, _ := outA.DataSet.GetString(dicom.TagStudyInstanceUID)
	studyB, _ := outB.DataSet.GetString(dicom.TagStudyInstanceUID)
	seriesA, _ := outA.DataSet.GetString(dicom.TagSeriesInstanceUID)
	seriesB, _ := outB.DataSet.GetString(dicom.TagSeriesInstanceUID)
	sopA, _ := outA.DataSet.GetString(dicom.TagSOPInstanceUID)
	sopB, _ := outB.DataSet.GetString(dicom.TagSOPInstanceUID)

	// Both files must land under ONE new Study UID and ONE new Series UID (grouping preserved).
	if studyA != studyB {
		t.Errorf("batch re-key split the study: A study=%q, B study=%q, want one shared new UID", studyA, studyB)
	}
	if seriesA != seriesB {
		t.Errorf("batch re-key split the series: A series=%q, B series=%q, want one shared new UID", seriesA, seriesB)
	}
	// The new grouping UIDs must differ from the originals (the re-key actually happened).
	if studyA == "1.2.3.4.5.1" {
		t.Errorf("Study UID was not regenerated: %q", studyA)
	}
	if seriesA == "1.2.3.4.5.2" {
		t.Errorf("Series UID was not regenerated: %q", seriesA)
	}
	// SOP Instance UIDs stay unique per instance — never shared.
	if sopA == sopB {
		t.Errorf("SOP Instance UIDs were shared across files: A=%q B=%q, want unique per instance", sopA, sopB)
	}
	if sopA == "1.2.3.4.700.1" || sopB == "1.2.3.4.700.2" {
		t.Errorf("a SOP Instance UID was not regenerated: A=%q B=%q", sopA, sopB)
	}
	// Every minted UID validates as conformant.
	for name, uid := range map[string]string{
		"studyA": studyA, "seriesA": seriesA, "sopA": sopA, "sopB": sopB,
	} {
		if err := dicom.UID(uid).Validate(); err != nil {
			t.Errorf("regenerated %s UID %q is not conformant: %v", name, uid, err)
		}
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

// TestModifyInPlaceWriteFailureLeavesOriginalIntact is the clinical-data-safety regression: when an
// --in-place write cannot complete (here the source's directory is read-only, so no sibling temp can
// be created), the command exits non-zero AND leaves the original file byte-for-byte unchanged — it
// never truncates or partially overwrites the source. The prior path called dicom.WriteFile, which
// truncates the destination with os.Create before writing, so a mid-write failure would have left
// the original DICOM partially overwritten.
func TestModifyInPlaceWriteFailureLeavesOriginalIntact(t *testing.T) {
	dir := t.TempDir()
	src := writeStorableDICOM(t, dir, "1.2.3.4.600.50")

	before, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read original before: %v", err)
	}

	// Make the source's directory read-only so the atomic writer cannot create its sibling temp
	// file. The file itself stays readable (the dir keeps r-x), so the read succeeds and only the
	// write fails — exactly the disk-full / NFS-error shape the fix guards against.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod read-only dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	stdout, _, code := runRadx(t, "modify", "--format", "json",
		"--in-place", "--insert", "PatientID=ANON-050", src)
	if code == exitcode.Success {
		t.Fatalf("in-place modify into a read-only dir exited 0; want non-zero\nstdout=%q", stdout)
	}

	// Restore write so we can read the file back and compare bytes.
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("restore dir perms: %v", err)
	}
	after, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read original after: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("the original was modified despite the write failure: %d bytes before, %d after",
			len(before), len(after))
	}

	// No stray temp file may be left beside the original.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("a stray temp file was left behind: dir entries = %v, want only the original", names)
	}
}

// TestModifyOutputDirBasenameCollisionFailsClosed confirms that two inputs whose base names collide
// under --output-dir do not silently overwrite one another: the run errors and the later input is
// not falsely reported successful. The prior code flattened every source to filepath.Base and the
// write truncated, so b/IM0001.dcm silently clobbered a/IM0001.dcm while both were reported success.
func TestModifyOutputDirBasenameCollisionFailsClosed(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	if err := os.MkdirAll(dirA, 0o700); err != nil {
		t.Fatalf("mkdir a: %v", err)
	}
	if err := os.MkdirAll(dirB, 0o700); err != nil {
		t.Fatalf("mkdir b: %v", err)
	}

	// Two distinct instances with the SAME base name in different directories.
	srcA := writeStorableDICOMNamed(t, dirA, "1.2.3.4.600.60", "IM0001.dcm")
	srcB := writeStorableDICOMNamed(t, dirB, "1.2.3.4.600.61", "IM0001.dcm")

	outDir := filepath.Join(root, "out")
	stdout, _, code := runRadx(t, "modify", "--format", "json",
		"--output-dir", outDir, "--insert", "PatientID=ANON-060", srcA, srcB)
	if code == exitcode.Success {
		t.Fatalf("colliding output names exited 0; want non-zero\nstdout=%q", stdout)
	}

	// The first input wrote its output; the second must be reported a failure, never a silent
	// overwrite. Decode the JSON object stream and check the per-file outcomes.
	dec := json.NewDecoder(strings.NewReader(stdout))
	var results []modifyResult
	for {
		var r modifyResult
		if err := dec.Decode(&r); err != nil {
			break
		}
		results = append(results, r)
	}
	if len(results) != 2 {
		t.Fatalf("want a result per input, got %d:\n%s", len(results), stdout)
	}
	if results[0].Status != "success" {
		t.Errorf("first input status = %q, want success", results[0].Status)
	}
	if results[1].Status != "failure" {
		t.Errorf("second (colliding) input status = %q, want failure (no silent overwrite)", results[1].Status)
	}

	// Exactly one output file exists, and it is the one written by the first input.
	out := filepath.Join(outDir, "IM0001.dcm")
	f, err := dicom.ReadFile(out)
	if err != nil {
		t.Fatalf("read written output: %v", err)
	}
	if got, _ := f.DataSet.GetString(dicom.TagSOPInstanceUID); got != "1.2.3.4.600.60" {
		t.Errorf("written output SOP Instance UID = %q, want the first input's 1.2.3.4.600.60", got)
	}
}
