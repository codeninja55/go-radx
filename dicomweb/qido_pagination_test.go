package dicomweb

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// fiveStudyBackend returns a QueryBackend holding five study candidates.
func fiveStudyBackend() *memQuery {
	var candidates []*dicom.DataSet
	for i := 1; i <= 5; i++ {
		uid := fmt.Sprintf("1.2.3.%d", i)
		candidates = append(candidates, studyRecord(uid, "P1", "Doe^Jane", "20240101", "ACC1", "CT"))
	}
	return &memQuery{candidates: candidates}
}

// studyUIDs extracts the StudyInstanceUID of every result, for order/completeness checks.
func studyUIDs(results []SearchResult) []string {
	out := make([]string, 0, len(results))
	for _, r := range results {
		uid, _ := r.DataSet.GetString(dicom.TagStudyInstanceUID)
		out = append(out, uid)
	}
	return out
}

// TestSearchStudiesAllFollowsWarning299 asserts the auto-paginating search keeps following
// the offset window while the origin signals truncation (Warning: 299, PS3.18 §10.6.1.4)
// and stops on the first page served without it, collecting the complete result set.
func TestSearchStudiesAllFollowsWarning299(t *testing.T) {
	hs := newQueryTestServer(t, fiveStudyBackend(), WithMaxQueryResults(2))
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// No caller limit: each page is truncated at the server's 2-result cap and carries
	// Warning 299 until the final, uncapped page.
	results, err := c.SearchStudiesAll(context.Background(), SearchQuery{}, 0)
	if err != nil {
		t.Fatalf("SearchStudiesAll: %v", err)
	}
	if got := studyUIDs(results); len(got) != 5 {
		t.Fatalf("SearchStudiesAll returned %d results (%v), want all 5", len(got), got)
	}
}

// TestSearchStudiesAllHonorsLimitAndOffset asserts q.Limit is the page size and q.Offset
// the starting point of the walk.
func TestSearchStudiesAllHonorsLimitAndOffset(t *testing.T) {
	hs := newQueryTestServer(t, fiveStudyBackend())
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudiesAll(context.Background(), SearchQuery{Limit: 2, Offset: 1}, 0)
	if err != nil {
		t.Fatalf("SearchStudiesAll: %v", err)
	}
	got := studyUIDs(results)
	want := []string{"1.2.3.2", "1.2.3.3", "1.2.3.4", "1.2.3.5"}
	if len(got) != len(want) {
		t.Fatalf("results = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("results = %v, want %v", got, want)
		}
	}
}

// TestSearchAllStopsOnShortPageWithoutWarning asserts a page shorter than the requested
// limit, served without Warning 299, ends the walk after a single request: a conformant
// origin that truncated would have sent the warning.
func TestSearchAllStopsOnShortPageWithoutWarning(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", mediaTypeDICOMJSON)
		_, _ = w.Write([]byte(`[{"0020000D":{"vr":"UI","Value":["1.2.3.1"]}}]`))
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudiesAll(context.Background(), SearchQuery{Limit: 10}, 0)
	if err != nil {
		t.Fatalf("SearchStudiesAll: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("origin saw %d requests, want 1 (short page without warning ends the walk)", got)
	}
}

// TestSearchAllPageBoundTruncates asserts the walk is bounded: an origin that always
// signals more results (each page advancing, so the non-advance guard stays quiet)
// cannot loop forever, and the bounded walk reports truncation as an error alongside the
// accumulated results rather than passing them off as complete.
func TestSearchAllPageBoundTruncates(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		w.Header().Set("Content-Type", mediaTypeDICOMJSON)
		w.Header().Set("Warning", warningResultsTruncated)
		body := fmt.Sprintf(`[{"0020000D":{"vr":"UI","Value":["1.2.3.%d"]}}]`, n)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudiesAll(context.Background(), SearchQuery{}, 3)
	if err == nil {
		t.Fatal("SearchStudiesAll on an endless origin returned nil error, want a truncation error")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("error = %v, want it to name truncation", err)
	}
	if len(results) != 3 {
		t.Fatalf("accumulated results = %d, want the 3 bounded pages' worth", len(results))
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("origin saw %d requests, want exactly the 3-page bound", got)
	}
}

// TestSearchSeriesAllAndInstancesAllScopePaths asserts the level variants walk the scoped
// search resources and validate their parent UIDs before any request.
func TestSearchSeriesAllAndInstancesAllScopePaths(t *testing.T) {
	var paths []string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.SearchSeriesAll(context.Background(), "1.2.3", SearchQuery{}, 0); err != nil {
		t.Fatalf("SearchSeriesAll: %v", err)
	}
	if _, err := c.SearchInstancesAll(context.Background(), "1.2.3", "1.2.3.4", SearchQuery{}, 0); err != nil {
		t.Fatalf("SearchInstancesAll: %v", err)
	}
	want := []string{"/studies/1.2.3/series", "/studies/1.2.3/series/1.2.3.4/instances"}
	if len(paths) != len(want) || paths[0] != want[0] || paths[1] != want[1] {
		t.Fatalf("origin saw paths %v, want %v", paths, want)
	}

	// A series scope without a study scope is rejected before any request is made.
	if _, err := c.SearchInstancesAll(context.Background(), "", "1.2.3.4", SearchQuery{}, 0); err == nil {
		t.Fatal("SearchInstancesAll with a series scope but no study scope returned nil error")
	}
}

// TestSearchAllStopsOnContextCancel asserts the walk is context-aware: a context
// cancelled between pages ends the walk with the context's error.
func TestSearchAllStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		cancel() // the caller gives up while the origin still reports more results
		w.Header().Set("Content-Type", mediaTypeDICOMJSON)
		w.Header().Set("Warning", warningResultsTruncated)
		_, _ = w.Write([]byte(`[{"0020000D":{"vr":"UI","Value":["1.2.3.1"]}}]`))
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := c.SearchStudiesAll(ctx, SearchQuery{}, 0); err == nil {
		t.Fatal("SearchStudiesAll with a cancelled context returned nil error")
	}
	if got := hits.Load(); got > 2 {
		t.Fatalf("origin saw %d requests after cancellation, want the walk to stop promptly", got)
	}
}

// TestHasMoreResultsWarning covers the Warning header parse: elements are split on the
// commas between warning-values (not commas inside the quoted warn-text), only warn-code
// 299 qualifies, and only when its text carries the additional-results semantic - a 299
// about an unrelated condition (fuzzymatching downgrade, say) must not read as "more
// results".
func TestHasMoreResultsWarning(t *testing.T) {
	cases := []struct {
		name string
		vals []string
		want bool
	}{
		{"combined field with unrelated 199 first",
			[]string{`199 - "went ahead anyway", 299 - "There are 40 additional results that can be requested"`}, true},
		{"comma inside the quoted warn-text",
			[]string{`299 - "more results exist, page with offset"`}, true},
		{"unrelated 299 (fuzzymatching downgrade)",
			[]string{`299 orthanc "The fuzzymatching parameter is not supported. Only literal matching performed."`}, false},
		{"code prefix must not over-match", []string{`2990 - "additional results"`}, false},
		{"our own server truncation text", []string{warningResultsTruncated}, true},
		{"no warning", nil, false},
	}
	for _, tc := range cases {
		h := http.Header{}
		for _, v := range tc.vals {
			h.Add("Warning", v)
		}
		if got := hasMoreResultsWarning(h); got != tc.want {
			t.Fatalf("%s: hasMoreResultsWarning = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestSearchAllIgnoresUnrelatedWarning299 asserts a 299 warning about something other
// than additional results does not force an extra page: a short page with only that
// warning ends the walk after one request.
func TestSearchAllIgnoresUnrelatedWarning299(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", mediaTypeDICOMJSON)
		w.Header().Set("Warning", `299 orthanc "The fuzzymatching parameter is not supported."`)
		_, _ = w.Write([]byte(`[{"0020000D":{"vr":"UI","Value":["1.2.3.1"]}}]`))
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudiesAll(context.Background(), SearchQuery{Limit: 10}, 0)
	if err != nil {
		t.Fatalf("SearchStudiesAll: %v", err)
	}
	if len(results) != 1 || hits.Load() != 1 {
		t.Fatalf("results = %d, requests = %d; want 1 and 1", len(results), hits.Load())
	}
}

// TestSearchAllWalksToEmptyPageWithoutLimit asserts the no-page-size walk matches
// dicomweb-client's get_remaining: it stops only on an empty page, so a short unwarned
// page from an origin with its own cap is not misread as exhaustion.
func TestSearchAllWalksToEmptyPageWithoutLimit(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.Header().Set("Content-Type", mediaTypeDICOMJSON)
			_, _ = w.Write([]byte(`[{"0020000D":{"vr":"UI","Value":["1.2.3.1"]}}]`))
			return
		}
		if r.URL.Query().Get("offset") != "1" {
			t.Errorf("second page offset = %q, want 1", r.URL.Query().Get("offset"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudiesAll(context.Background(), SearchQuery{}, 0)
	if err != nil {
		t.Fatalf("SearchStudiesAll: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("origin saw %d requests, want 2 (page then the empty page that ends the walk)", got)
	}
}

// TestSearchAllAbortsWhenOriginIgnoresOffset asserts a page that does not advance (an
// origin ignoring the offset parameter) aborts the walk with the accumulated results and
// an error, rather than accumulating duplicates until the page bound.
func TestSearchAllAbortsWhenOriginIgnoresOffset(t *testing.T) {
	var hits atomic.Int32
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", mediaTypeDICOMJSON)
		_, _ = w.Write([]byte(`[{"0020000D":{"vr":"UI","Value":["1.2.3.1"]}}]`))
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudiesAll(context.Background(), SearchQuery{}, 0)
	if err == nil {
		t.Fatal("SearchStudiesAll against an offset-ignoring origin returned nil error")
	}
	if !strings.Contains(err.Error(), "advance") {
		t.Fatalf("error = %v, want it to name the non-advancing page", err)
	}
	if len(results) != 1 {
		t.Fatalf("accumulated results = %d, want the 1 unique page (no duplicates)", len(results))
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("origin saw %d requests, want 2 (the repeat detected immediately)", got)
	}
}

// TestSearchPageVariantsSurfaceTruncation asserts the single-shot Page variants expose
// the origin's truncation signal so a non-paginating caller can see an incomplete page.
func TestSearchPageVariantsSurfaceTruncation(t *testing.T) {
	hs := newQueryTestServer(t, fiveStudyBackend(), WithMaxQueryResults(2))
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, more, err := c.SearchStudiesPage(context.Background(), SearchQuery{})
	if err != nil {
		t.Fatalf("SearchStudiesPage: %v", err)
	}
	if len(results) != 2 || !more {
		t.Fatalf("first page = %d results, more=%v; want 2 and true", len(results), more)
	}

	results, more, err = c.SearchStudiesPage(context.Background(), SearchQuery{Offset: 4})
	if err != nil {
		t.Fatalf("SearchStudiesPage(offset 4): %v", err)
	}
	if len(results) != 1 || more {
		t.Fatalf("final page = %d results, more=%v; want 1 and false", len(results), more)
	}
}

// TestSearchPageVariantsScopePaths asserts the series and instance Page variants walk
// the scoped search resources and validate parent UIDs before any request.
func TestSearchPageVariantsScopePaths(t *testing.T) {
	var paths []string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(hs.Close)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, _, err := c.SearchSeriesPage(context.Background(), "1.2.3", SearchQuery{}); err != nil {
		t.Fatalf("SearchSeriesPage: %v", err)
	}
	if _, _, err := c.SearchInstancesPage(context.Background(), "1.2.3", "1.2.3.4", SearchQuery{}); err != nil {
		t.Fatalf("SearchInstancesPage: %v", err)
	}
	want := []string{"/studies/1.2.3/series", "/studies/1.2.3/series/1.2.3.4/instances"}
	if len(paths) != len(want) || paths[0] != want[0] || paths[1] != want[1] {
		t.Fatalf("origin saw paths %v, want %v", paths, want)
	}
	if _, _, err := c.SearchInstancesPage(context.Background(), "", "1.2.3.4", SearchQuery{}); err == nil {
		t.Fatal("SearchInstancesPage with a series scope but no study scope returned nil error")
	}
}
