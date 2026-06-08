package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/cmd/radx/internal/exitcode"
	"github.com/codeninja55/go-radx/dicom"
)

// writeCataloguePatientDICOM writes a synthetic storable file carrying a PatientID sentinel, so a
// redaction test can exercise the hashed PHI column. The value is a synthetic sentinel, never PHI.
func writeCataloguePatientDICOM(t *testing.T, dir, sopInstanceUID, patientID string) string {
	t.Helper()
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	ds.SetString(dicom.TagSOPInstanceUID, sopInstanceUID)
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3.4.5.1")
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4.5.2")
	ds.SetString(dicom.TagModality, "CT")
	ds.SetString(dicom.TagPatientID, patientID)

	path := filepath.Join(dir, strings.ReplaceAll(sopInstanceUID, ".", "_")+".dcm")
	if err := ds.WriteFile(path, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write patient DICOM: %v", err)
	}
	return path
}

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

// TestCatalogueQueryMissingDatabaseExits5 confirms a --query against a non-existent catalogue is a
// file-I/O failure (exit 5), not a silent empty success: query mode reads an existing catalogue, so a
// mistyped --database path must refuse the run rather than create and migrate an empty file and stream
// zero matches at exit 0. The database file is never created.
func TestCatalogueQueryMissingDatabaseExits5(t *testing.T) {
	db := filepath.Join(t.TempDir(), "does-not-exist.db")
	_, _, code := runRadx(t, "catalogue", "--format", "json", "--database", db, "--query", "Modality=CT")
	if code != exitcode.FileIOError {
		t.Fatalf("query of a missing catalogue exit = %d, want %d (file-I/O failure, not a clean empty result)", code, exitcode.FileIOError)
	}
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Errorf("query of a missing catalogue created the database file; query mode must not conjure a catalogue")
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

// TestCatalogueRedactedQueryHonoursRedactFlag is the load-bearing regression for the redacted-query
// path: a catalogue indexed with --redact stores PatientID as a one-way hash, so an exact PatientID
// filter only matches when the query is opened with --redact too (the backend hashes the filter
// value the same way). Without --redact the cleartext filter compares against the stored hash and
// returns nothing. The PatientID is a synthetic sentinel, never real PHI.
func TestCatalogueRedactedQueryHonoursRedactFlag(t *testing.T) {
	const sentinel = "CAT-REDACT-SENTINEL-1"
	srcDir := t.TempDir()
	writeCataloguePatientDICOM(t, srcDir, "1.2.804.1", sentinel)
	db := filepath.Join(t.TempDir(), "catalogue.db")

	if _, _, code := runRadx(t, "catalogue", "--database", db, "--redact", srcDir); code != exitcode.Success {
		t.Fatalf("catalogue --redact index exit = %d, want %d", code, exitcode.Success)
	}

	// Querying the redacted catalogue WITH --redact hashes the filter value and matches the row.
	stdout, stderr, code := runRadx(t, "catalogue", "--format", "json", "--database", db,
		"--redact", "--query", "PatientID="+sentinel)
	if code != exitcode.Success {
		t.Fatalf("redacted query exit = %d, want %d\nstdout=%q\nstderr=%q", code, exitcode.Success, stdout, stderr)
	}
	if len(nonEmptyLines(stdout)) == 0 {
		t.Errorf("redacted query with --redact returned no matches; want the indexed instance")
	}

	// Without --redact the cleartext filter compares against the stored hash and returns nothing,
	// proving the flag is what wires the matching redaction setting through.
	plainOut, _, plainCode := runRadx(t, "catalogue", "--format", "json", "--database", db,
		"--query", "PatientID="+sentinel)
	if plainCode != exitcode.Success {
		t.Fatalf("cleartext query exit = %d, want %d\nstdout=%q", plainCode, exitcode.Success, plainOut)
	}
	if len(nonEmptyLines(plainOut)) != 0 {
		t.Errorf("cleartext query against a redacted catalogue returned matches; want none:\n%s", plainOut)
	}
}

// TestCatalogueRedactStoresNoCleartextIdentifier is the P1 privacy regression: `catalogue --redact`
// WITHOUT --confirm-phi is allowed precisely because a redacted catalogue persists no cleartext for
// any direct identifier — patient name/ID AND accession number. A read-only SQL scan of the redacted
// catalogue must return zero rows for any of the cleartext sentinels, proving the bypass of the PHI
// acknowledgement is honest. The sentinels are synthetic, never real PHI.
func TestCatalogueRedactStoresNoCleartextIdentifier(t *testing.T) {
	const (
		patientID = "CAT-REDACT-PID-1"
		accession = "CAT-REDACT-ACC-1"
	)
	srcDir := t.TempDir()
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.2") // CT Image Storage
	ds.SetString(dicom.TagSOPInstanceUID, "1.2.807.1")
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.807")
	ds.SetString(dicom.TagSeriesInstanceUID, "1.2.807.0")
	ds.SetString(dicom.TagModality, "CT")
	ds.SetString(dicom.TagPatientID, patientID)
	ds.SetString(dicom.TagAccessionNumber, accession)
	path := filepath.Join(srcDir, "redact_accession.dcm")
	if err := ds.WriteFile(path, dicom.ExplicitVRLittleEndian); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	db := filepath.Join(t.TempDir(), "catalogue.db")

	// --redact WITHOUT --confirm-phi indexes successfully (the acknowledgement is not required).
	if _, _, code := runRadx(t, "catalogue", "--format", "json", "--database", db, "--redact", srcDir); code != exitcode.Success {
		t.Fatalf("catalogue --redact (no --confirm-phi) exit = %d, want %d", code, exitcode.Success)
	}

	// A read-only SQL scan must find no cleartext for any direct identifier. A row would prove the
	// redacted catalogue still persists reversible PHI, contradicting the --confirm-phi bypass.
	for _, sentinel := range []string{patientID, accession} {
		stdout, _, code := runRadx(t, "catalogue", "--database", db, "--mode", "list",
			"--sql", "SELECT sop_instance_uid FROM instances WHERE patient_id = '"+sentinel+
				"' OR accession_number = '"+sentinel+"'")
		if code != exitcode.Success {
			t.Fatalf("scan for %q exit = %d, want %d\nstdout=%q", sentinel, code, exitcode.Success, stdout)
		}
		if len(nonEmptyLines(stdout)) != 0 {
			t.Errorf("redacted catalogue persists cleartext for %q:\n%s", sentinel, stdout)
		}
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

// TestCatalogueIndexTruncatedFileExits3 confirms a truncated/malformed object makes the index run
// exit with ParseError (exit 3): catalogue must preserve the underlying dicom parse error so
// exitcode.Classify routes it to its real class rather than collapsing it into a usage error. The
// exit-code taxonomy is the contract operators branch on.
func TestCatalogueIndexTruncatedFileExits3(t *testing.T) {
	srcDir := t.TempDir()
	good := writeStorableDICOM(t, srcDir, "1.2.804.1")
	full, err := os.ReadFile(good)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// Cut well inside the main dataset so the reader fails mid-element, a parse failure not a clean EOF.
	bad := filepath.Join(srcDir, "truncated.dcm")
	if err := os.WriteFile(bad, full[:len(full)-8], 0o600); err != nil {
		t.Fatalf("write truncated fixture: %v", err)
	}
	db := filepath.Join(t.TempDir(), "catalogue.db")

	_, _, code := runRadx(t, "catalogue", "--format", "json", "--database", db, "--confirm-phi", srcDir)
	if code != exitcode.ParseError {
		t.Fatalf("catalogue index of a truncated file exit = %d, want %d (parse failure, not usage)", code, exitcode.ParseError)
	}
}

// TestCatalogueIndexPermissionDeniedExits5 confirms a permission-denied input makes the index run
// exit with FileIOError (exit 5): catalogue must preserve the underlying fs.ErrPermission so Classify
// routes it to the file-I/O class rather than a usage error.
func TestCatalogueIndexPermissionDeniedExits5(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses file mode bits; cannot provoke a permission denial")
	}
	srcDir := t.TempDir()
	denied := writeStorableDICOM(t, srcDir, "1.2.805.1")
	if err := os.Chmod(denied, 0o000); err != nil {
		t.Fatalf("chmod 000: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0o600) }) // let t.TempDir cleanup remove it
	db := filepath.Join(t.TempDir(), "catalogue.db")

	_, _, code := runRadx(t, "catalogue", "--format", "json", "--database", db, "--confirm-phi", srcDir)
	if code != exitcode.FileIOError {
		t.Fatalf("catalogue index of a permission-denied file exit = %d, want %d (file-I/O failure, not usage)", code, exitcode.FileIOError)
	}
}

// TestCatalogueIndexIgnoreErrorsExits0RecordsFailure confirms --ignore-errors opts a failing index
// into a zero exit while still tallying the failure in the machine output, so an exploratory run does
// not hide the unindexed file.
func TestCatalogueIndexIgnoreErrorsExits0RecordsFailure(t *testing.T) {
	srcDir := t.TempDir()
	writeStorableDICOM(t, srcDir, "1.2.806.1") // one valid file
	good := writeStorableDICOM(t, srcDir, "1.2.806.2")
	full, err := os.ReadFile(good)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(good, full[:len(full)-8], 0o600); err != nil { // truncate one file
		t.Fatalf("truncate fixture: %v", err)
	}
	db := filepath.Join(t.TempDir(), "catalogue.db")

	stdout, _, code := runRadx(t, "catalogue", "--format", "json",
		"--database", db, "--confirm-phi", "--ignore-errors", srcDir)
	if code != exitcode.Success {
		t.Fatalf("catalogue index --ignore-errors exit = %d, want %d", code, exitcode.Success)
	}
	var got catalogueIndexResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if got.Failed != 1 {
		t.Errorf("--ignore-errors index failed tally = %d, want 1 (the failure must still be recorded)", got.Failed)
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
