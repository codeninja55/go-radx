package server

import (
	"context"
	"errors"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// indexed builds a dataset carrying the queryable attributes the catalogue indexes.
func indexed(study, series, instance, patientID, modality string) *dicom.DataSet {
	ds := newTestObject(study, series, instance)
	ds.SetString(dicom.TagPatientID, patientID)
	ds.SetString(dicom.TagModality, modality)
	return ds
}

func TestSQLiteCatalogueIndexAndQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cat, err := SQLiteCatalogue(ctx, ":memory:")
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}

	if err := cat.Index(ctx, indexed("1.2.1", "1.2.1.1", "1.2.1.1.1", "MRN-A", "CT")); err != nil {
		t.Fatalf("Index: %v", err)
	}
	if err := cat.Index(ctx, indexed("1.2.2", "1.2.2.1", "1.2.2.1.1", "MRN-B", "MR")); err != nil {
		t.Fatalf("Index: %v", err)
	}

	// A universal study-level query returns every indexed instance.
	all := collect(t, cat.Query(ctx, CatalogueQuery{Level: dimse.QueryLevelStudy}))
	if len(all) != 2 {
		t.Fatalf("universal query returned %d rows, want 2", len(all))
	}

	// A constrained study-level query matches exactly one study by PatientID and returns the
	// study-level identity (Modality is a series-level attribute, so a study-level row omits it).
	matched := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagPatientID: "MRN-A"},
	}))
	if len(matched) != 1 {
		t.Fatalf("PatientID=MRN-A query returned %d rows, want 1", len(matched))
	}
	if v, _ := matched[0].GetString(dicom.TagStudyInstanceUID); v != "1.2.1" {
		t.Errorf("matched StudyInstanceUID = %q, want 1.2.1", v)
	}

	// A wildcard match on a string-VR attribute (PatientID, LO) returns both studies under the prefix.
	// The authoritative matcher applies DICOM wildcard semantics to string VRs; a UID VR uses UID-list
	// matching, not wildcards (PS3.4 C.2.2.2), so the wildcard is exercised on PatientID here.
	wild := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagPatientID: "MRN-*"},
	}))
	if len(wild) != 2 {
		t.Fatalf("wildcard query returned %d rows, want 2", len(wild))
	}

	// Re-indexing the same instance updates rather than duplicates (idempotent on SOPInstanceUID).
	if err := cat.Index(ctx, indexed("1.2.1", "1.2.1.1", "1.2.1.1.1", "MRN-A", "US")); err != nil {
		t.Fatalf("re-Index: %v", err)
	}
	again := collect(t, cat.Query(ctx, CatalogueQuery{Level: dimse.QueryLevelStudy}))
	if len(again) != 2 {
		t.Fatalf("after re-index query returned %d rows, want 2 (no duplicate)", len(again))
	}

	// Remove drops the row; a second remove reports ErrNotFound.
	if err := cat.Remove(ctx, dicom.SOPInstanceUID("1.2.1.1.1")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := cat.Remove(ctx, dicom.SOPInstanceUID("1.2.1.1.1")); !errors.Is(err, ErrNotFound) {
		t.Errorf("Remove of absent instance = %v, want ErrNotFound", err)
	}
}

// TestSQLiteCatalogueRequiresExplicitPath asserts the catalogue refuses an empty path, so no
// PHI-bearing catalogue is ever created at an implicit default (PRD §9.1).
func TestSQLiteCatalogueRequiresExplicitPath(t *testing.T) {
	t.Parallel()
	if _, err := SQLiteCatalogue(context.Background(), ""); err == nil {
		t.Fatal("SQLiteCatalogue with an empty path succeeded, want a rejection")
	}
}

// TestSQLiteCatalogueRedaction asserts WithRedaction hashes every direct identifier so the catalogue
// never stores cleartext PatientID/PatientName/AccessionNumber; equality matching still works against
// the hash. AccessionNumber is a direct identifier (an order/visit record locator), so a redacted
// catalogue must carry no cleartext for it any more than for the patient name or ID.
func TestSQLiteCatalogueRedaction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cat, err := SQLiteCatalogue(ctx, ":memory:", WithRedaction(true))
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}
	const mrn = "MRN-SECRET-123"
	const accession = "ACC-SECRET-456"
	ds := indexed("1.3.1", "1.3.1.1", "1.3.1.1.1", mrn, "CT")
	ds.SetString(dicom.TagAccessionNumber, accession)
	if err := cat.Index(ctx, ds); err != nil {
		t.Fatalf("Index: %v", err)
	}

	rows := collect(t, cat.Query(ctx, CatalogueQuery{Level: dimse.QueryLevelStudy}))
	if len(rows) != 1 {
		t.Fatalf("query returned %d rows, want 1", len(rows))
	}
	stored, _ := rows[0].GetString(dicom.TagPatientID)
	if stored == mrn {
		t.Fatal("redacted catalogue stored the cleartext PatientID, want a hash")
	}
	if stored != hashIdentifier(mrn) {
		t.Errorf("stored PatientID = %q, want the hash of the cleartext", stored)
	}
	acc, _ := rows[0].GetString(dicom.TagAccessionNumber)
	if acc == accession {
		t.Fatal("redacted catalogue stored the cleartext AccessionNumber, want a hash")
	}
	if acc != hashIdentifier(accession) {
		t.Errorf("stored AccessionNumber = %q, want the hash of the cleartext", acc)
	}
}

// TestSQLiteCatalogueRedactionPersistsNoCleartextIdentifier asserts a redacted catalogue persists no
// cleartext for ANY direct identifier it stores: a raw scan of the backing table must not contain the
// PatientID, PatientName, or AccessionNumber sentinel. This is the load-bearing privacy guarantee that
// lets the CLI's --redact bypass the PHI acknowledgement honestly: a redacted catalogue carries no
// reversible PHI. The sentinels are synthetic, never real PHI.
func TestSQLiteCatalogueRedactionPersistsNoCleartextIdentifier(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const dbPath = ":memory:"
	cat, err := SQLiteCatalogue(ctx, dbPath, WithRedaction(true))
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}
	const (
		patientID   = "MRN-NO-CLEARTEXT"
		patientName = "DOE^NO-CLEARTEXT"
		accession   = "ACC-NO-CLEARTEXT"
	)
	ds := indexed("1.4.1", "1.4.1.1", "1.4.1.1.1", patientID, "CT")
	ds.SetString(dicom.TagPatientName, patientName)
	ds.SetString(dicom.TagAccessionNumber, accession)
	if err := cat.Index(ctx, ds); err != nil {
		t.Fatalf("Index: %v", err)
	}

	sc, ok := cat.(*sqliteCatalogue)
	if !ok {
		t.Fatalf("catalogue is %T, want *sqliteCatalogue", cat)
	}
	for _, sentinel := range []string{patientID, patientName, accession} {
		var n int
		if err := sc.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM instances WHERE patient_id = ? OR patient_name = ? OR accession_number = ?",
			sentinel, sentinel, sentinel).Scan(&n); err != nil {
			t.Fatalf("scan for cleartext %q: %v", sentinel, err)
		}
		if n != 0 {
			t.Errorf("redacted catalogue persists cleartext for %q (found %d rows); want none", sentinel, n)
		}
	}
}

// TestSQLiteCatalogueRedactedSearchMatchesCleartext asserts the redacted-search fix: with redaction
// on, a query by the cleartext PatientID matches the stored hash. Without hashing the incoming match
// value, the WHERE would compare cleartext against the stored hash and never match (Finding 3).
func TestSQLiteCatalogueRedactedSearchMatchesCleartext(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cat, err := SQLiteCatalogue(ctx, ":memory:", WithRedaction(true))
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}
	const mrn = "MRN-REDACT-SEARCH"
	if err := cat.Index(ctx, indexed("9.1.1", "9.1.1.1", "9.1.1.1.1", mrn, "CT")); err != nil {
		t.Fatalf("Index: %v", err)
	}

	rows := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagPatientID: mrn},
	}))
	if len(rows) != 1 {
		t.Fatalf("query by cleartext PatientID against a redacted catalogue returned %d rows, want 1", len(rows))
	}
	if v, _ := rows[0].GetString(dicom.TagStudyInstanceUID); v != "9.1.1" {
		t.Errorf("matched StudyInstanceUID = %q, want 9.1.1", v)
	}
}

// TestSQLiteCatalogueRedactedWildcardMatchesNothing asserts a redacted catalogue never over-matches on
// a wildcard/range PHI filter it cannot honour. PatientID is stored hashed, so a wildcard "MRN-*" or a
// range cannot be tested against the one-way hash (the cleartext needed to apply the pattern is gone).
// The constraint must match NOTHING for that clause, never be silently dropped (which would return
// every row). An exact PatientID still matches its row, so exact-identity matching keeps working.
func TestSQLiteCatalogueRedactedWildcardMatchesNothing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cat, err := SQLiteCatalogue(ctx, ":memory:", WithRedaction(true))
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}
	for _, mrn := range []string{"MRN-001", "MRN-002", "MRN-003"} {
		study := "11." + mrn
		if err := cat.Index(ctx, indexed(study, study+".1", study+".1.1", mrn, "CT")); err != nil {
			t.Fatalf("Index %s: %v", mrn, err)
		}
	}

	// A wildcard on the redacted PatientID matches nothing rather than over-matching every row.
	wild := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagPatientID: "MRN-*"},
	}))
	if len(wild) != 0 {
		t.Fatalf("redacted wildcard PatientID=MRN-* returned %d rows, want 0 (a hash cannot honour a wildcard)", len(wild))
	}

	// An exact PatientID still matches its row (exact-identity matching against the hash works).
	exact := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagPatientID: "MRN-001"},
	}))
	if len(exact) != 1 {
		t.Fatalf("redacted exact PatientID=MRN-001 returned %d rows, want 1", len(exact))
	}
	if v, _ := exact[0].GetString(dicom.TagStudyInstanceUID); v != "11.MRN-001" {
		t.Errorf("exact-matched StudyInstanceUID = %q, want 11.MRN-001", v)
	}
}

// TestSQLiteCatalogueStudyLevelDistinct asserts the query-level fix: a study with multiple instances
// returns ONE row for a study-level query (distinct study), not one row per instance (Finding 4).
func TestSQLiteCatalogueStudyLevelDistinct(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cat, err := SQLiteCatalogue(ctx, ":memory:")
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}
	const study = "7.1.1"
	const series = "7.1.1.1"
	// Three instances in one study/series.
	for _, inst := range []string{"7.1.1.1.1", "7.1.1.1.2", "7.1.1.1.3"} {
		if err := cat.Index(ctx, indexed(study, series, inst, "MRN-MULTI", "CT")); err != nil {
			t.Fatalf("Index %s: %v", inst, err)
		}
	}

	studyRows := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagStudyInstanceUID: study},
	}))
	if len(studyRows) != 1 {
		t.Fatalf("study-level query over a 3-instance study returned %d rows, want 1", len(studyRows))
	}

	// A series-level query likewise collapses the three instances to one series row.
	seriesRows := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelSeries,
		Match: map[dicom.Tag]string{dicom.TagStudyInstanceUID: study},
	}))
	if len(seriesRows) != 1 {
		t.Fatalf("series-level query returned %d rows, want 1", len(seriesRows))
	}

	// An instance-level query still returns every instance.
	instanceRows := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelImage,
		Match: map[dicom.Tag]string{dicom.TagStudyInstanceUID: study},
	}))
	if len(instanceRows) != 3 {
		t.Fatalf("instance-level query returned %d rows, want 3", len(instanceRows))
	}
}

// TestSQLiteCatalogueDateRangeMatch asserts the catalogue-narrows-vs-matcher-decides fix for a DA
// range value: a StudyDate range "20240110-20240120" must match the studies whose date falls in the
// window, not compare the whole literal as SQL equality and return nothing (Finding 2). The catalogue
// leaves a range value to the authoritative matcher rather than pushing it down as exact equality.
func TestSQLiteCatalogueDateRangeMatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cat, err := SQLiteCatalogue(ctx, ":memory:")
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}

	dates := map[string]string{"8.1.1": "20240105", "8.1.2": "20240115", "8.1.3": "20240125"}
	for study, date := range dates {
		ds := indexed(study, study+".1", study+".1.1", "MRN-DATE", "CT")
		ds.SetString(dicom.TagStudyDate, date)
		if err := cat.Index(ctx, ds); err != nil {
			t.Fatalf("Index %s: %v", study, err)
		}
	}

	rows := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagStudyDate: "20240110-20240120"},
	}))
	if len(rows) != 1 {
		t.Fatalf("StudyDate range query returned %d rows, want 1 (the 20240115 study)", len(rows))
	}
	if v, _ := rows[0].GetString(dicom.TagStudyInstanceUID); v != "8.1.2" {
		t.Errorf("range-matched StudyInstanceUID = %q, want 8.1.2", v)
	}
}

// TestSQLiteCatalogueUIDListMatch asserts the catalogue-narrows-vs-matcher-decides fix for a UID-list
// value: a StudyInstanceUID list "8.2.1\8.2.3" must match the listed studies, not compare the whole
// backslash literal as SQL equality and return nothing (Finding 2).
func TestSQLiteCatalogueUIDListMatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cat, err := SQLiteCatalogue(ctx, ":memory:")
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}
	for _, study := range []string{"8.2.1", "8.2.2", "8.2.3"} {
		if err := cat.Index(ctx, indexed(study, study+".1", study+".1.1", "MRN-LIST", "CT")); err != nil {
			t.Fatalf("Index %s: %v", study, err)
		}
	}

	rows := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagStudyInstanceUID: "8.2.1\\8.2.3"},
	}))
	if len(rows) != 2 {
		t.Fatalf("UID-list query returned %d rows, want 2 (the listed studies)", len(rows))
	}
	seen := map[string]bool{}
	for _, r := range rows {
		v, _ := r.GetString(dicom.TagStudyInstanceUID)
		seen[v] = true
	}
	if !seen["8.2.1"] || !seen["8.2.3"] || seen["8.2.2"] {
		t.Errorf("UID-list matched studies = %v, want exactly 8.2.1 and 8.2.3", seen)
	}
}

// collect drains a Catalogue.Query iterator into a slice, failing on the first error.
func collect(t *testing.T, seq func(yield func(*dicom.DataSet, error) bool)) []*dicom.DataSet {
	t.Helper()
	var out []*dicom.DataSet
	for ds, err := range seq {
		if err != nil {
			t.Fatalf("query iterator error: %v", err)
		}
		out = append(out, ds)
	}
	return out
}
