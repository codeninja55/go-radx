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

	// A valid SOP UID under the wrong study is not-found (the dicomweb sentinel, so the server
	// answers 404 rather than treating the mismatch as a backend fault).
	_, err = b.RetrieveInstance(ctx, dicomweb.NewInstance("9.9.9", dicom.UID(series), dicom.UID(instance)))
	if !errors.Is(err, dicomweb.ErrNotFound) {
		t.Errorf("RetrieveInstance under wrong study = %v, want dicomweb.ErrNotFound", err)
	}

	// A valid SOP UID under the wrong series is not-found.
	_, err = b.RetrieveInstance(ctx, dicomweb.NewInstance(dicom.UID(study), "9.9.9.9", dicom.UID(instance)))
	if !errors.Is(err, dicomweb.ErrNotFound) {
		t.Errorf("RetrieveInstance under wrong series = %v, want dicomweb.ErrNotFound", err)
	}
}

// TestDICOMwebQueryIndexedMatchOutsideLevelProjection asserts that a QIDO match on an indexed attribute
// the search level does NOT project (a series search keyed on SOPClassUID) still returns the matching
// series. The level collapse projects a series row to its series-identifying columns, which omit
// SOPClassUID; without carrying the matched attribute through, the DICOMweb server's re-match would
// reject every candidate as missing the attribute and answer an empty result. The candidate must carry
// the matched SOPClassUID so the re-match passes.
func TestDICOMwebQueryIndexedMatchOutsideLevelProjection(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	ctx := context.Background()

	// newTestObject sets SOPClassUID to the Secondary Capture SOP Class; index two series, one of each
	// SOP Class, so the SOPClassUID match must select exactly the matching series.
	target := newTestObject("12.1", "12.1.1", "12.1.1.1")
	target.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7")
	other := newTestObject("12.2", "12.2.1", "12.2.1.1")
	other.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.2")
	for _, ds := range []*dicom.DataSet{target, other} {
		if err := store.Put(ctx, ds); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := cat.Index(ctx, ds); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	b := &dicomwebQuery{cat: cat, store: store}
	q := dicomweb.QueryRequest{
		Level: dicomweb.QuerySeries,
		Match: []dicomweb.MatchKey{
			dicomweb.NewMatchKey(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7"),
		},
	}
	got, err := b.Query(ctx, q)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("series search keyed on SOPClassUID returned %d rows, want 1", len(got))
	}
	if v, _ := got[0].GetString(dicom.TagSeriesInstanceUID); v != "12.1.1" {
		t.Errorf("matched SeriesInstanceUID = %q, want 12.1.1", v)
	}
	// The matched attribute is carried onto the collapsed series row so the server's re-match sees it.
	if v, _ := got[0].GetString(dicom.TagSOPClassUID); v != "1.2.840.10008.5.1.4.1.1.7" {
		t.Errorf("collapsed row SOPClassUID = %q, want the matched SOP Class", v)
	}
}

// TestDICOMwebQueryIncludeFieldProjectedFromStore asserts a QIDO includefield naming an attribute
// outside the search level's default projection (a series search requesting SOPClassUID) yields the
// requested attribute on the result. The adapter must carry the includefield through the level collapse
// so the DICOMweb server's projection has the value rather than dropping it as absent.
func TestDICOMwebQueryIncludeFieldProjectedFromStore(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	ctx := context.Background()

	ds := newTestObject("13.1", "13.1.1", "13.1.1.1")
	ds.SetString(dicom.TagSOPClassUID, "1.2.840.10008.5.1.4.1.1.7")
	if err := store.Put(ctx, ds); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := cat.Index(ctx, ds); err != nil {
		t.Fatalf("Index: %v", err)
	}

	b := &dicomwebQuery{cat: cat, store: store}
	q := dicomweb.QueryRequest{
		Level:         dicomweb.QuerySeries,
		IncludeFields: []dicom.Tag{dicom.TagSOPClassUID},
	}
	got, err := b.Query(ctx, q)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("series search returned %d rows, want 1", len(got))
	}
	if v, _ := got[0].GetString(dicom.TagSOPClassUID); v != "1.2.840.10008.5.1.4.1.1.7" {
		t.Errorf("included SOPClassUID = %q, want it projected onto the result", v)
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
