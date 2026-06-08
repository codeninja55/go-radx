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

	// A constrained query matches exactly one instance by PatientID.
	matched := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagPatientID: "MRN-A"},
	}))
	if len(matched) != 1 {
		t.Fatalf("PatientID=MRN-A query returned %d rows, want 1", len(matched))
	}
	if v, _ := matched[0].GetString(dicom.TagModality); v != "CT" {
		t.Errorf("matched modality = %q, want CT", v)
	}

	// A wildcard match on StudyInstanceUID returns both studies under the prefix.
	wild := collect(t, cat.Query(ctx, CatalogueQuery{
		Level: dimse.QueryLevelStudy,
		Match: map[dicom.Tag]string{dicom.TagStudyInstanceUID: "1.2.*"},
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

// TestSQLiteCatalogueRedaction asserts WithRedaction hashes the direct identifiers so the catalogue
// never stores cleartext PatientID/PatientName; equality matching still works against the hash.
func TestSQLiteCatalogueRedaction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cat, err := SQLiteCatalogue(ctx, ":memory:", WithRedaction(true))
	if err != nil {
		t.Fatalf("SQLiteCatalogue: %v", err)
	}
	const mrn = "MRN-SECRET-123"
	if err := cat.Index(ctx, indexed("1.3.1", "1.3.1.1", "1.3.1.1.1", mrn, "CT")); err != nil {
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
