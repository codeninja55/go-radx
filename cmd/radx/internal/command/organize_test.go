package command

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
)

// writeOrganizeFixture writes a synthetic file with the given Study/Series/SOP UIDs so a test can
// assert the organised layout.
func writeOrganizeFixture(t *testing.T, dir, study, series, instance string) string {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.2")
	ds.SetString(dicom.TagSOPInstanceUID, instance)
	ds.SetString(dicom.TagStudyInstanceUID, study)
	ds.SetString(dicom.TagSeriesInstanceUID, series)
	path := filepath.Join(dir, instance+".dcm")
	if err := ds.WriteFile(path, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write organize fixture: %v", err)
	}
	return path
}

// writeOrganizeFixtureModality writes a synthetic file with the given UIDs and a Modality element, so
// an overwrite test can produce a destination payload that differs byte-for-byte from a fixture
// written without one. The value is a synthetic sentinel, never PHI.
func writeOrganizeFixtureModality(t *testing.T, dir, study, series, instance, modality string) string {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.2")
	ds.SetString(dicom.TagSOPInstanceUID, instance)
	ds.SetString(dicom.TagStudyInstanceUID, study)
	ds.SetString(dicom.TagSeriesInstanceUID, series)
	ds.SetString(dicom.TagModality, modality)
	path := filepath.Join(dir, instance+".dcm")
	if err := ds.WriteFile(path, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write organize fixture: %v", err)
	}
	return path
}

// TestOrganizeCopiesIntoLayout confirms organize copies files into a Study/Series/SOP tree and
// leaves the source intact (copy mode).
func TestOrganizeCopiesIntoLayout(t *testing.T) {
	srcDir := t.TempDir()
	src := writeOrganizeFixture(t, srcDir, "1.2.700.1", "1.2.700.2", "1.2.700.3")
	outDir := filepath.Join(t.TempDir(), "organized")

	stdout, stderr, code := runRadx(t, "organize", "--format", "json", "--output-dir", outDir, srcDir)
	if code != exitcode.Success {
		t.Fatalf("organize exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}

	dest := filepath.Join(outDir, "1.2.700.1", "1.2.700.2", "1.2.700.3.dcm")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected organised file at %s: %v", dest, err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source was removed in copy mode: %v", err)
	}

	var got organizeAction
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Status != "copied" {
		t.Errorf("status = %q, want copied", got.Status)
	}
}

// TestOrganizeMovesAndRemovesSource confirms --move relocates the file and removes the source.
func TestOrganizeMovesAndRemovesSource(t *testing.T) {
	srcDir := t.TempDir()
	src := writeOrganizeFixture(t, srcDir, "1.2.701.1", "1.2.701.2", "1.2.701.3")
	outDir := filepath.Join(t.TempDir(), "organized")

	_, _, code := runRadx(t, "organize", "--format", "json", "--move", "--output-dir", outDir, srcDir)
	if code != exitcode.Success {
		t.Fatalf("organize --move exit = %d, want %d", code, exitcode.Success)
	}
	dest := filepath.Join(outDir, "1.2.701.1", "1.2.701.2", "1.2.701.3.dcm")
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected moved file at %s: %v", dest, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source still present after --move: %v", err)
	}
}

// TestOrganizeDryRunTouchesNothing confirms --dry-run performs no I/O: it reports the planned
// layout and writes no files.
func TestOrganizeDryRunTouchesNothing(t *testing.T) {
	srcDir := t.TempDir()
	writeOrganizeFixture(t, srcDir, "1.2.702.1", "1.2.702.2", "1.2.702.3")
	outDir := filepath.Join(t.TempDir(), "organized")

	stdout, _, code := runRadx(t, "organize", "--format", "json", "--dry-run", "--output-dir", outDir, srcDir)
	if code != exitcode.Success {
		t.Fatalf("organize --dry-run exit = %d, want %d", code, exitcode.Success)
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("--dry-run created the output directory; it must touch nothing")
	}
	var got organizeAction
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if got.Status != "planned" {
		t.Errorf("status = %q, want planned", got.Status)
	}
}

// TestOrganizeNoOverwriteByDefault confirms organize refuses to overwrite an existing destination
// unless --overwrite is set, so a duplicate UID cannot silently truncate a stored file (RADX-018).
func TestOrganizeNoOverwriteByDefault(t *testing.T) {
	srcDir := t.TempDir()
	writeOrganizeFixture(t, srcDir, "1.2.703.1", "1.2.703.2", "1.2.703.3")
	outDir := filepath.Join(t.TempDir(), "organized")

	// First run lands the file.
	if _, _, code := runRadx(t, "organize", "--output-dir", outDir, srcDir); code != exitcode.Success {
		t.Fatalf("first organize exit = %d, want 0", code)
	}
	// Second run hits the existing destination and must fail (non-zero) without --overwrite.
	_, _, code := runRadx(t, "organize", "--format", "json", "--output-dir", outDir, srcDir)
	if code == exitcode.Success {
		t.Fatalf("organize over an existing destination exited 0; want non-zero without --overwrite")
	}
}

// TestOrganizeOverwriteSuccessReplaces confirms a successful --overwrite replaces the destination
// with the new contents, so the atomic temp-and-rename path commits when the copy succeeds.
func TestOrganizeOverwriteSuccessReplaces(t *testing.T) {
	const study, series, instance = "1.2.704.1", "1.2.704.2", "1.2.704.3"
	outDir := filepath.Join(t.TempDir(), "organized")
	dest := filepath.Join(outDir, study, series, instance+".dcm")

	// First run lands an initial file with the modality set to CT.
	first := t.TempDir()
	writeOrganizeFixture(t, first, study, series, instance)
	if _, _, code := runRadx(t, "organize", "--output-dir", outDir, first); code != exitcode.Success {
		t.Fatalf("first organize exit = %d, want 0", code)
	}

	// Second run overwrites with a source carrying a distinct byte payload (an extra Modality element),
	// so a successful overwrite is observable as a content change.
	second := t.TempDir()
	src := writeOrganizeFixtureModality(t, second, study, series, instance, "MR")
	wantBytes, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read replacement source: %v", err)
	}
	if _, _, code := runRadx(t, "organize", "--overwrite", "--output-dir", outDir, second); code != exitcode.Success {
		t.Fatalf("organize --overwrite exit = %d, want 0", code)
	}
	gotBytes, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read destination after overwrite: %v", err)
	}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Errorf("destination after a successful --overwrite was not replaced with the new contents")
	}
}

// TestOrganizeOverwriteFailurePreservesDestination is the data-safety regression for RADX-018: an
// --overwrite whose copy fails must leave the EXISTING destination byte-for-byte intact, never
// truncated or removed. The prior O_TRUNC path destroyed the destination before the copy, so a failed
// copy left no file at all. The fix copies into a sibling temp file and renames over the destination
// only on success; a failed copy removes the temp and leaves the original untouched. The failure is
// provoked by making the destination directory read-only so the sibling temp cannot be created, while
// the existing destination file inside it stays readable.
func TestOrganizeOverwriteFailurePreservesDestination(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses directory write permissions; cannot provoke the temp-create failure")
	}
	const study, series, instance = "1.2.705.1", "1.2.705.2", "1.2.705.3"
	outDir := filepath.Join(t.TempDir(), "organized")
	destDir := filepath.Join(outDir, study, series)
	dest := filepath.Join(destDir, instance+".dcm")

	// First run lands the original file we must not lose.
	first := t.TempDir()
	writeOrganizeFixture(t, first, study, series, instance)
	if _, _, code := runRadx(t, "organize", "--output-dir", outDir, first); code != exitcode.Success {
		t.Fatalf("first organize exit = %d, want 0", code)
	}
	original, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read original destination: %v", err)
	}

	// Make the destination directory read-only so the atomic copy's sibling temp file cannot be
	// created and the overwrite fails after the destination already exists.
	if err := os.Chmod(destDir, 0o500); err != nil {
		t.Fatalf("chmod destination dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(destDir, 0o750) }) // restore so t.TempDir cleanup can remove it

	second := t.TempDir()
	writeOrganizeFixtureModality(t, second, study, series, instance, "MR")
	_, _, code := runRadx(t, "organize", "--overwrite", "--output-dir", outDir, second)
	if code == exitcode.Success {
		t.Fatalf("organize --overwrite with a failing copy exited 0; want non-zero")
	}

	// Restore read access and assert the EXISTING destination is byte-for-byte intact: a failed
	// overwrite must never destroy the file it was replacing.
	if err := os.Chmod(destDir, 0o750); err != nil {
		t.Fatalf("restore destination dir perms: %v", err)
	}
	after, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("destination missing after a failed overwrite (data loss): %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Errorf("destination changed after a failed overwrite; want byte-identical to the original")
	}
}
