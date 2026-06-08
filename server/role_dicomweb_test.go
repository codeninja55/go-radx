package server

import (
	"context"
	"errors"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dicomweb"
)

// TestDICOMwebRetrieveValidatesParentUIDs asserts the WADO hierarchy fix: an instance fetched by SOP
// Instance UID under the WRONG study/series path is reported not-found, while the correct path
// returns it (Finding 6). The object store keys only on SOP UID, so the parent UIDs must be checked
// against the stored dataset.
func TestDICOMwebRetrieveValidatesParentUIDs(t *testing.T) {
	t.Parallel()
	store, _ := newTestBackends(t)
	ctx := context.Background()

	const study = "5.1.1"
	const series = "5.1.1.1"
	const instance = "5.1.1.1.1"
	if err := store.Put(ctx, newTestObject(study, series, instance)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	b := &dicomwebRetrieve{store: store}

	// The correct path returns the instance.
	got, err := b.RetrieveInstance(ctx, dicomweb.NewInstance(dicom.UID(study), dicom.UID(series), dicom.UID(instance)))
	if err != nil {
		t.Fatalf("RetrieveInstance under the correct path: %v", err)
	}
	if v, _ := got.GetString(dicom.TagSOPInstanceUID); v != instance {
		t.Errorf("retrieved SOPInstanceUID = %q, want %q", v, instance)
	}

	// A valid SOP UID under the wrong study is not-found.
	_, err = b.RetrieveInstance(ctx, dicomweb.NewInstance("9.9.9", dicom.UID(series), dicom.UID(instance)))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("RetrieveInstance under wrong study = %v, want ErrNotFound", err)
	}

	// A valid SOP UID under the wrong series is not-found.
	_, err = b.RetrieveInstance(ctx, dicomweb.NewInstance(dicom.UID(study), "9.9.9.9", dicom.UID(instance)))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("RetrieveInstance under wrong series = %v, want ErrNotFound", err)
	}
}

// TestDICOMwebQueryMatchesUnindexedKeyFromStore asserts the catalogue-narrows-vs-matcher-decides fix:
// a QIDO match on an attribute the catalogue does NOT index (StudyDescription) is honoured by fetching
// the stored dataset from the ObjectStore and applying the full DICOM match against the real value, so
// the matching study is returned rather than every candidate being rejected as missing the attribute
// (Finding 3). The candidate is collapsed to one row per study and carries the matched attribute so the
// DICOMweb server's re-match passes.
func TestDICOMwebQueryMatchesUnindexedKeyFromStore(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	ctx := context.Background()

	// Store and index three studies; only the THIRD carries the StudyDescription the request matches on.
	for i, study := range []string{"6.1.1", "6.1.2", "6.1.3"} {
		ds := newTestObject(study, study+".1", study+".1.1")
		if i == 2 {
			ds.SetString(dicom.TagStudyDescription, "TARGET")
		}
		if err := store.Put(ctx, ds); err != nil {
			t.Fatalf("Put %s: %v", study, err)
		}
		if err := cat.Index(ctx, ds); err != nil {
			t.Fatalf("Index %s: %v", study, err)
		}
	}

	b := &dicomwebQuery{cat: cat, store: store}
	q := dicomweb.QueryRequest{
		Level: dicomweb.QueryStudies,
		Match: []dicomweb.MatchKey{
			{Tag: dicom.TagStudyDescription, VR: dicom.VRLO, Value: "TARGET"},
		},
	}
	got, err := b.Query(ctx, q)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	// The unindexed match must select exactly the TARGET study, not every candidate and not none.
	if len(got) != 1 {
		t.Fatalf("Query on an unindexed key returned %d rows, want 1 (the TARGET study)", len(got))
	}
	if v, _ := got[0].GetString(dicom.TagStudyInstanceUID); v != "6.1.3" {
		t.Errorf("matched StudyInstanceUID = %q, want 6.1.3", v)
	}
	// The matched attribute is carried onto the row so the DICOMweb server's re-match and projection see
	// it rather than treating the projected study row as missing the attribute.
	if v, _ := got[0].GetString(dicom.TagStudyDescription); v != "TARGET" {
		t.Errorf("matched row StudyDescription = %q, want TARGET", v)
	}
}
