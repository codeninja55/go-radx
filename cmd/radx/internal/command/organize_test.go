package command

import (
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
