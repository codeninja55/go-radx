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

// TestDICOMwebQueryDoesNotPreLimitUnindexedMatch asserts the QIDO-limit fix: when a request carries
// an unindexed match key alongside a limit, the catalogue is NOT asked to pre-limit, so the matching
// row that sorts after the first N candidates is still fetched for the server's matchDataSet to find
// (Finding 5). The catalogue indexes neither StudyDescription nor any free-text attribute, so a match
// on it must be applied AFTER the candidate set is fetched, not pushed into SQLite's LIMIT.
func TestDICOMwebQueryDoesNotPreLimitUnindexedMatch(t *testing.T) {
	t.Parallel()
	_, cat := newTestBackends(t)
	ctx := context.Background()

	// Index three studies; only the THIRD (which sorts last) carries the StudyDescription the request
	// will match on. With a limit of 1 pushed into SQLite, the third row would be discarded.
	for i, study := range []string{"6.1.1", "6.1.2", "6.1.3"} {
		ds := newTestObject(study, study+".1", study+".1.1")
		if i == 2 {
			ds.SetString(dicom.TagStudyDescription, "TARGET")
		}
		if err := cat.Index(ctx, ds); err != nil {
			t.Fatalf("Index %s: %v", study, err)
		}
	}

	b := &dicomwebQuery{cat: cat}
	q := dicomweb.QueryRequest{
		Level: dicomweb.QueryStudies,
		Limit: 1,
		Match: []dicomweb.MatchKey{
			{Tag: dicom.TagStudyDescription, VR: dicom.VRLO, Value: "TARGET"},
		},
	}
	candidates, err := b.Query(ctx, q)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	// The backend must return the full candidate set (all three studies), not the first one, so the
	// server's matchDataSet can find the TARGET study that sorts last.
	if len(candidates) != 3 {
		t.Fatalf("Query with an unindexed match key + limit returned %d candidates, want 3 (no pre-limit)", len(candidates))
	}

	// Sanity: with only indexed match keys present, the limit IS pushed down (the catalogue filtering
	// is complete), so the candidate set is bounded.
	indexedQ := dicomweb.QueryRequest{
		Level: dicomweb.QueryStudies,
		Limit: 1,
		Match: []dicomweb.MatchKey{
			{Tag: dicom.TagStudyInstanceUID, VR: dicom.VRUI, Value: "6.*"},
		},
	}
	bounded, err := b.Query(ctx, indexedQ)
	if err != nil {
		t.Fatalf("Query (indexed): %v", err)
	}
	if len(bounded) != 1 {
		t.Errorf("Query with only indexed match keys + limit 1 returned %d candidates, want 1 (limit pushed down)", len(bounded))
	}
}
