package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
)

// TestCatalogueIndexAndQuery is the happy path: a directory of synthetic files is indexed under
// --confirm-phi, the database is created 0600, and a tag-filter query returns the indexed instance.
func TestCatalogueIndexAndQuery(t *testing.T) {
	srcDir := t.TempDir()
	writeStorableDICOM(t, srcDir, "1.2.800.1")
	writeStorableDICOM(t, srcDir, "1.2.800.2")
	db := filepath.Join(t.TempDir(), "catalogue.db")

	stdout, stderr, code := runRadx(t, "catalogue", "--format", "json",
		"--database", db, "--confirm-phi", srcDir)
	if code != exitcode.Success {
		t.Fatalf("catalogue index exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	var got catalogueIndexResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Indexed != 2 || got.Failed != 0 {
		t.Errorf("index result = %+v, want 2 indexed 0 failed", got)
	}

	// The PHI store must be owner-only (RADX-008).
	info, err := os.Stat(db)
	if err != nil {
		t.Fatalf("stat catalogue: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("catalogue file perm = %o, want 600", perm)
	}

	// A tag-filter query returns the indexed instances.
	qOut, _, qCode := runRadx(t, "catalogue", "--format", "json", "--database", db,
		"--query", "Modality=CT")
	if qCode != exitcode.Success {
		t.Fatalf("catalogue query exit = %d, want %d\nstdout=%q", qCode, exitcode.Success, qOut)
	}
	if len(nonEmptyLines(qOut)) == 0 {
		t.Errorf("query returned no matches; want the CT instances")
	}
}

// TestCataloguePHIGate is the privacy regression: indexing PHI columns without --confirm-phi is a
// usage error (the catalogue is an opt-in PHI store, RADX-007), and --redact indexes without the
// acknowledgement.
func TestCataloguePHIGate(t *testing.T) {
	srcDir := t.TempDir()
	writeStorableDICOM(t, srcDir, "1.2.801.1")
	db := filepath.Join(t.TempDir(), "catalogue.db")

	_, _, code := runRadx(t, "catalogue", "--database", db, srcDir)
	if code != exitcode.UsageError {
		t.Fatalf("catalogue index without --confirm-phi exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}

	// --redact indexes structural-only fields without the PHI acknowledgement.
	_, _, redactCode := runRadx(t, "catalogue", "--database", db, "--redact", srcDir)
	if redactCode != exitcode.Success {
		t.Fatalf("catalogue --redact index exit = %d, want %d", redactCode, exitcode.Success)
	}
}

// TestCatalogueSQLReadOnlyGate confirms --sql rejects empty SQL and a non-SELECT statement as a
// clean usage error rather than a panic (RADX-014), and that a valid SELECT runs.
func TestCatalogueSQLReadOnlyGate(t *testing.T) {
	srcDir := t.TempDir()
	writeStorableDICOM(t, srcDir, "1.2.802.1")
	db := filepath.Join(t.TempDir(), "catalogue.db")
	if _, _, code := runRadx(t, "catalogue", "--database", db, "--confirm-phi", srcDir); code != exitcode.Success {
		t.Fatalf("index exit = %d, want 0", code)
	}

	// Empty SQL is a usage error.
	if _, _, code := runRadx(t, "catalogue", "--database", db, "--sql", "   "); code != exitcode.UsageError {
		t.Errorf("empty --sql exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
	// A mutating statement is rejected.
	if _, _, code := runRadx(t, "catalogue", "--database", db, "--sql", "DELETE FROM instances"); code != exitcode.UsageError {
		t.Errorf("mutating --sql exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}
	// A stacked statement is rejected.
	if _, _, code := runRadx(t, "catalogue", "--database", db,
		"--sql", "SELECT 1; DROP TABLE instances"); code != exitcode.UsageError {
		t.Errorf("stacked --sql exit = %d, want %d (usage error)", code, exitcode.UsageError)
	}

	// A valid read-only SELECT runs and renders rows.
	stdout, _, code := runRadx(t, "catalogue", "--database", db,
		"--sql", "SELECT sop_instance_uid FROM instances", "--mode", "csv")
	if code != exitcode.Success {
		t.Fatalf("valid --sql exit = %d, want %d\nstdout=%q", code, exitcode.Success, stdout)
	}
	if len(nonEmptyLines(stdout)) < 2 { // header + at least one row
		t.Errorf("--sql returned no rows:\n%s", stdout)
	}
}

// TestCatalogueIndexTruncatedFileFailsClosed is the truncation regression: an unparseable input
// makes the index run exit non-zero (RADX-012/013), never logging and returning a clean success.
func TestCatalogueIndexTruncatedFileFailsClosed(t *testing.T) {
	srcDir := t.TempDir()
	writeStorableDICOM(t, srcDir, "1.2.803.1") // one valid file
	// A truncated .dcm: a DICM preamble with a short, mid-element body. The reader must reject it.
	bad := filepath.Join(srcDir, "truncated.dcm")
	preamble := make([]byte, 132)
	copy(preamble[128:], []byte("DICM"))
	if err := os.WriteFile(bad, preamble, 0o600); err != nil {
		t.Fatalf("write truncated fixture: %v", err)
	}
	db := filepath.Join(t.TempDir(), "catalogue.db")

	_, _, code := runRadx(t, "catalogue", "--format", "json", "--database", db, "--confirm-phi", srcDir)
	if code == exitcode.Success {
		t.Fatalf("catalogue index over a truncated file exited 0; want non-zero (RADX-012/013)")
	}

	// --ignore-errors opts into a zero exit while still recording the failure.
	stdout, _, igCode := runRadx(t, "catalogue", "--format", "json",
		"--database", db, "--confirm-phi", "--ignore-errors", "--rebuild", srcDir)
	if igCode != exitcode.Success {
		t.Fatalf("catalogue index --ignore-errors exit = %d, want %d", igCode, exitcode.Success)
	}
	var got catalogueIndexResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if got.Failed == 0 {
		t.Errorf("expected the truncated file to be tallied as failed even with --ignore-errors")
	}
}

// TestCatalogueSchema confirms --schema prints the column set without opening a database.
func TestCatalogueSchema(t *testing.T) {
	stdout, _, code := runRadx(t, "catalogue", "--format", "json", "--schema")
	if code != exitcode.Success {
		t.Fatalf("catalogue --schema exit = %d, want %d", code, exitcode.Success)
	}
	var cols []map[string]any
	if err := json.Unmarshal([]byte(stdout), &cols); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if len(cols) == 0 {
		t.Errorf("schema is empty")
	}
}
