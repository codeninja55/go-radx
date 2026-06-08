package server

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// newTestObject builds a minimal valid object carrying the study/series/instance hierarchy the
// store keys on, plus a SOP Class so the written file meta is well-formed.
func newTestObject(study, series, instance string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagStudyInstanceUID, study)
	ds.SetString(dicom.TagSeriesInstanceUID, series)
	ds.SetString(dicom.TagSOPInstanceUID, instance)
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7")
	ds.SetString(dicom.TagModality, "OT")
	return ds
}

func TestFileStoreRoundTrip(t *testing.T) {
	t.Parallel()
	store, err := FileStore(t.TempDir())
	if err != nil {
		t.Fatalf("FileStore: %v", err)
	}
	ctx := context.Background()

	const study = "1.2.840.113619.2.1.1"
	const series = "1.2.840.113619.2.1.2"
	const instance = "1.2.840.113619.2.1.3"
	ds := newTestObject(study, series, instance)

	if err := store.Put(ctx, ds); err != nil {
		t.Fatalf("Put: %v", err)
	}

	exists, err := store.Exists(ctx, dicom.SOPInstanceUID(instance))
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("Exists = false after Put, want true")
	}

	got, err := store.Get(ctx, dicom.SOPInstanceUID(instance))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if v, _ := got.GetString(dicom.TagSOPInstanceUID); v != instance {
		t.Errorf("round-tripped SOPInstanceUID = %q, want %q", v, instance)
	}
	if v, _ := got.GetString(dicom.TagStudyInstanceUID); v != study {
		t.Errorf("round-tripped StudyInstanceUID = %q, want %q", v, study)
	}

	// Put is idempotent on SOP Instance UID: storing the same instance twice is not an error.
	if err := store.Put(ctx, ds); err != nil {
		t.Fatalf("second Put (idempotent): %v", err)
	}

	if err := store.Delete(ctx, dicom.SOPInstanceUID(instance)); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, dicom.SOPInstanceUID(instance)); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete = %v, want ErrNotFound", err)
	}
	if err := store.Delete(ctx, dicom.SOPInstanceUID(instance)); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete of absent instance = %v, want ErrNotFound", err)
	}
}

// TestFileStoreRejectsTraversalUID asserts a traversal-style UID cannot escape the store root: the
// store validates each UID path component and refuses one that is not a conformant DICOM UID, so a
// "../" component is rejected before any path is built (PRD §9.1 input validation).
func TestFileStoreRejectsTraversalUID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := FileStore(root)
	if err != nil {
		t.Fatalf("FileStore: %v", err)
	}
	ctx := context.Background()

	// A Put whose instance UID is a path-traversal string must be rejected, never written.
	ds := newTestObject("1.2.3", "1.2.4", "../../etc/evil")
	if err := store.Put(ctx, ds); err == nil {
		t.Fatal("Put with a traversal-style SOPInstanceUID succeeded, want a rejection")
	}

	// A Put whose study UID escapes the tree must be rejected too.
	ds2 := newTestObject("../../../tmp/escape", "1.2.4", "1.2.5")
	if err := store.Put(ctx, ds2); err == nil {
		t.Fatal("Put with a traversal-style StudyInstanceUID succeeded, want a rejection")
	}

	// A Get/Exists/Delete keyed by a traversal UID must be rejected before any filesystem walk.
	if _, err := store.Get(ctx, dicom.SOPInstanceUID("../../etc/passwd")); err == nil {
		t.Fatal("Get with a traversal-style UID succeeded, want a rejection")
	}
	if ok, err := store.Exists(ctx, dicom.SOPInstanceUID("../escape")); err == nil && ok {
		t.Fatal("Exists with a traversal-style UID reported true, want a rejection or false")
	}

	// Belt-and-braces: nothing escaped the root.
	matches, _ := filepath.Glob(filepath.Join(root, "..", "*evil*"))
	if len(matches) > 0 {
		t.Fatalf("traversal wrote outside the root: %v", matches)
	}
}
