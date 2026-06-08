package server

import (
	"context"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
	"github.com/codeninja55/go-radx/dimse"
)

// TestQueryCatalogueUnindexedKeyFetchesFromStore asserts the shared C-FIND/QIDO matching path honours a
// match key the catalogue does not index (BodyPartExamined): the stored dataset is fetched from the
// ObjectStore so the matcher sees the real value, and only the matching study is returned rather than
// every candidate being rejected as missing the attribute (Finding 3). It also asserts the level
// collapse still yields one row per matching study even though matching ran at instance granularity.
func TestQueryCatalogueUnindexedKeyFetchesFromStore(t *testing.T) {
	t.Parallel()
	store, cat := newTestBackends(t)
	ctx := context.Background()

	// Two studies. The first has two instances both CHEST; the second is ABDOMEN. BodyPartExamined is
	// not a catalogue-indexed column, so the match must be decided against the stored datasets.
	chest := []struct{ study, series, instance string }{
		{"10.1", "10.1.1", "10.1.1.1"},
		{"10.1", "10.1.1", "10.1.1.2"},
	}
	for _, o := range chest {
		ds := newTestObject(o.study, o.series, o.instance)
		ds.SetString(dicom.TagBodyPartExamined, "CHEST")
		if err := store.Put(ctx, ds); err != nil {
			t.Fatalf("Put %s: %v", o.instance, err)
		}
		if err := cat.Index(ctx, ds); err != nil {
			t.Fatalf("Index %s: %v", o.instance, err)
		}
	}
	abd := newTestObject("20.1", "20.1.1", "20.1.1.1")
	abd.SetString(dicom.TagBodyPartExamined, "ABDOMEN")
	if err := store.Put(ctx, abd); err != nil {
		t.Fatalf("Put abdomen: %v", err)
	}
	if err := cat.Index(ctx, abd); err != nil {
		t.Fatalf("Index abdomen: %v", err)
	}

	match := map[dicom.Tag]string{dicom.TagBodyPartExamined: "CHEST"}
	var rows []*dicom.DataSet
	for ds, err := range queryCatalogue(ctx, cat, store, match, dimse.QueryLevelStudy, false) {
		if err != nil {
			t.Fatalf("queryCatalogue: %v", err)
		}
		rows = append(rows, ds)
	}

	// The CHEST study matches (despite two instances, the study-level collapse yields one row); the
	// ABDOMEN study does not.
	if len(rows) != 1 {
		t.Fatalf("BodyPartExamined=CHEST study-level query returned %d rows, want 1", len(rows))
	}
	if v, _ := rows[0].GetString(dicom.TagStudyInstanceUID); v != "10.1" {
		t.Errorf("matched StudyInstanceUID = %q, want 10.1", v)
	}
}
