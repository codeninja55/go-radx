package dicomweb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/codeninja55/go-radx/dicom"
)

// memQuery is an in-memory QueryBackend for the QIDO-RS round-trip tests. It returns its
// fixed candidate set for every level; the server applies matching, projection, and
// paging on top, so the backend stays trivial.
type memQuery struct {
	candidates []*dicom.DataSet
	err        error
}

func (m *memQuery) Query(_ context.Context, _ QueryRequest) ([]*dicom.DataSet, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.candidates, nil
}

// studyRecord builds a study-level candidate dataset carrying the attributes the matching
// and projection tests exercise.
func studyRecord(study, patientID, patientName, studyDate, accession, modalities string) *dicom.DataSet {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagStudyInstanceUID, study)
	ds.SetString(dicom.TagPatientID, patientID)
	ds.Set(dicom.Element{Tag: dicom.TagPatientName, VR: dicom.VRPN, Value: dicom.NewStrings(dicom.VRPN, patientName)})
	ds.SetString(dicom.TagStudyDate, studyDate)
	ds.SetString(dicom.TagAccessionNumber, accession)
	ds.SetString(dicom.TagModalitiesInStudy, modalities)
	// A non-default attribute the projection should drop unless includefield names it.
	ds.SetString(dicom.TagBodyPartExamined, "BRAIN")
	return ds
}

func newQueryTestServer(t *testing.T, q QueryBackend, opts ...ServerOption) *httptest.Server {
	t.Helper()
	opts = append([]ServerOption{WithQueryBackend(q)}, opts...)
	srv, err := NewServer(opts...)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	return hs
}

func TestMatchString(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		pattern   string
		want      bool
	}{
		{"exact match", "CT", "CT", true},
		{"exact mismatch", "CT", "MR", false},
		{"case sensitive", "ct", "CT", false},
		{"star suffix", "BRAIN", "BRA*", true},
		{"star prefix", "BRAIN", "*AIN", true},
		{"star middle", "BRAIN", "B*N", true},
		{"star no match", "BRAIN", "X*", false},
		{"question single", "CT", "C?", true},
		{"question too many", "CT", "C??", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchString(tc.candidate, tc.pattern); got != tc.want {
				t.Errorf("matchString(%q, %q) = %v, want %v", tc.candidate, tc.pattern, got, tc.want)
			}
		})
	}
}

func TestUniversalMatch(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"", true},
		{"*", true},
		{"CT", false},
		{"BRA*", false},
	}
	for _, tc := range cases {
		if got := isUniversalMatch(tc.value); got != tc.want {
			t.Errorf("isUniversalMatch(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

// TestMatchKeyAbsentAttribute asserts a present, non-universal key against an absent
// attribute never matches, while a universal key matches even an absent attribute (PS3.4
// C.2.2.2.4).
func TestMatchKeyAbsentAttribute(t *testing.T) {
	ds := dicom.NewDataSet()
	ds.SetString(dicom.TagStudyInstanceUID, "1.2.3")

	present := MatchKey{Tag: dicom.TagPatientID, VR: dicom.VRLO, Value: "PID1"}
	if matchKey(ds, present, false) {
		t.Error("a non-universal key against an absent attribute matched, want no match")
	}
	universal := MatchKey{Tag: dicom.TagPatientID, VR: dicom.VRLO, Value: "*"}
	if !matchKey(ds, universal, false) {
		t.Error("a universal key against an absent attribute did not match, want match")
	}
}

func TestMatchUIDList(t *testing.T) {
	cands := []string{"1.2.3"}
	cases := []struct {
		name string
		list string
		want bool
	}{
		{"single match", "1.2.3", true},
		{"single mismatch", "9.9.9", false},
		{"list contains", "9.9.9\\1.2.3\\4.5.6", true},
		{"list excludes", "9.9.9\\4.5.6", false},
		{"trailing separator", "1.2.3\\", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchUIDList(cands, tc.list); got != tc.want {
				t.Errorf("matchUIDList(%v, %q) = %v, want %v", cands, tc.list, got, tc.want)
			}
		})
	}
}

func TestMatchRange(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		pattern   string
		want      bool
	}{
		{"in closed range", "20200615", "20200101-20201231", true},
		{"below closed range", "20191231", "20200101-20201231", false},
		{"above closed range", "20210101", "20200101-20201231", false},
		{"open upper in", "20250101", "20200101-", true},
		{"open upper below", "20190101", "20200101-", false},
		{"open lower in", "20190101", "-20200101", true},
		{"open lower above", "20200102", "-20200101", false},
		{"single value match", "20200101", "20200101", true},
		{"single value mismatch", "20200102", "20200101", false},
		{"coarse upper bound year", "20201231", "20200101-2020", true},
		{"empty range rejected", "20200101", "-", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchRange(tc.candidate, tc.pattern); got != tc.want {
				t.Errorf("matchRange(%q, %q) = %v, want %v", tc.candidate, tc.pattern, got, tc.want)
			}
		})
	}
}

func TestMatchPersonNameFuzzy(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		pattern   string
		fuzzy     bool
		want      bool
	}{
		{"exact non-fuzzy", "Doe^Jane", "Doe^Jane", false, true},
		{"case mismatch non-fuzzy", "Doe^Jane", "doe^jane", false, false},
		{"fuzzy case insensitive token", "Doe^Jane", "jane", true, true},
		{"fuzzy family token", "Doe^Jane", "doe", true, true},
		{"fuzzy no token", "Doe^Jane", "smith", true, false},
		{"fuzzy wildcard wrap", "Doe^Jane", "*Doe*", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchPersonName(tc.candidate, tc.pattern, tc.fuzzy); got != tc.want {
				t.Errorf("matchPersonName(%q, %q, fuzzy=%v) = %v, want %v",
					tc.candidate, tc.pattern, tc.fuzzy, got, tc.want)
			}
		})
	}
}

func TestParseQueryRequestLevels(t *testing.T) {
	cases := []struct {
		name      string
		segs      []string
		wantLevel QueryLevel
		wantStudy dicom.UID
		wantErr   bool
	}{
		{"studies", []string{"studies"}, QueryStudies, "", false},
		{"series under study", []string{"studies", "1.2.3", "series"}, QuerySeries, "1.2.3", false},
		{"relative series", []string{"series"}, QuerySeries, "", false},
		{"instances under series", []string{"studies", "1.2.3", "series", "1.2.3.4", "instances"}, QueryInstances, "1.2.3", false},
		{"relative instances", []string{"instances"}, QueryInstances, "", false},
		{"bad study uid", []string{"studies", "not-a-uid", "series"}, QueryStudies, "", true},
		{"unknown resource", []string{"frames"}, QueryStudies, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := parseQueryRequest(tc.segs, url.Values{})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseQueryRequest(%v) = nil error, want error", tc.segs)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseQueryRequest(%v): %v", tc.segs, err)
			}
			if q.Level != tc.wantLevel {
				t.Errorf("level = %v, want %v", q.Level, tc.wantLevel)
			}
			if q.StudyUID != tc.wantStudy {
				t.Errorf("study = %q, want %q", q.StudyUID, tc.wantStudy)
			}
		})
	}
}

func TestParseQueryRequestParams(t *testing.T) {
	v := url.Values{}
	v.Set("PatientID", "12345")
	v.Set("includefield", "BodyPartExamined")
	v.Add("includefield", "00080060")
	v.Set("fuzzymatching", "true")
	v.Set("limit", "50")
	v.Set("offset", "10")

	q, err := parseQueryRequest([]string{"studies"}, v)
	if err != nil {
		t.Fatalf("parseQueryRequest: %v", err)
	}
	if len(q.Match) != 1 || q.Match[0].Tag != dicom.TagPatientID || q.Match[0].Value != "12345" {
		t.Fatalf("Match = %+v, want one PatientID=12345 key", q.Match)
	}
	if !q.Fuzzy {
		t.Error("Fuzzy = false, want true")
	}
	if q.Limit != 50 || q.Offset != 10 {
		t.Errorf("Limit/Offset = %d/%d, want 50/10", q.Limit, q.Offset)
	}
	wantInclude := map[dicom.Tag]bool{dicom.TagBodyPartExamined: true, dicom.TagModality: true}
	if len(q.IncludeFields) != 2 {
		t.Fatalf("IncludeFields = %+v, want two fields", q.IncludeFields)
	}
	for _, tag := range q.IncludeFields {
		if !wantInclude[tag] {
			t.Errorf("unexpected includefield tag %v", tag)
		}
	}
}

func TestParseQueryRequestIncludeAll(t *testing.T) {
	v := url.Values{}
	v.Set("includefield", "all")
	q, err := parseQueryRequest([]string{"studies"}, v)
	if err != nil {
		t.Fatalf("parseQueryRequest: %v", err)
	}
	if !q.IncludeAll {
		t.Fatal("IncludeAll = false, want true for includefield=all")
	}
}

func TestParseQueryRequestRejectsBadParams(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{"unknown attribute", "NotARealAttribute", "x"},
		{"negative limit", "limit", "-5"},
		{"non-numeric offset", "offset", "abc"},
		{"unknown includefield", "includefield", "NotAField"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := url.Values{}
			v.Set(tc.key, tc.value)
			if _, err := parseQueryRequest([]string{"studies"}, v); err == nil {
				t.Fatalf("parseQueryRequest with %s=%s returned nil error", tc.key, tc.value)
			}
		})
	}
}

// TestParseQueryRequestErrorOmitsValue asserts a rejected attribute name carrying
// non-structural characters never echoes the raw input back, since a query parameter could
// be crafted to smuggle data (PRD §9.1).
func TestParseQueryRequestErrorOmitsValue(t *testing.T) {
	v := url.Values{}
	v.Set("Doe^Jane^SENSITIVE", "x")
	_, err := parseQueryRequest([]string{"studies"}, v)
	if err == nil {
		t.Fatal("expected an error for a non-structural attribute name")
	}
	if strings.Contains(err.Error(), "SENSITIVE") || strings.Contains(err.Error(), "Doe") {
		t.Fatalf("error leaked the attribute name: %q", err.Error())
	}
}

func TestPageResults(t *testing.T) {
	mk := func(n int) []*dicom.DataSet {
		out := make([]*dicom.DataSet, n)
		for i := range out {
			out[i] = dicom.NewDataSet()
		}
		return out
	}
	cases := []struct {
		name          string
		total         int
		offset        int
		limit         int
		maxCap        int
		wantLen       int
		wantTruncated bool
	}{
		{"all fit", 5, 0, 0, 100, 5, false},
		{"limit truncates", 5, 0, 2, 100, 2, true},
		{"offset skips", 5, 3, 0, 100, 2, false},
		{"offset past end", 5, 10, 0, 100, 0, false},
		{"cap truncates", 10, 0, 0, 4, 4, true},
		{"limit above cap clamps", 10, 0, 100, 4, 4, true},
		{"exact fit no truncation", 4, 0, 4, 100, 4, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, truncated := pageResults(mk(tc.total), tc.offset, tc.limit, tc.maxCap)
			if len(page) != tc.wantLen {
				t.Errorf("len(page) = %d, want %d", len(page), tc.wantLen)
			}
			if truncated != tc.wantTruncated {
				t.Errorf("truncated = %v, want %v", truncated, tc.wantTruncated)
			}
		})
	}
}

func TestProjectResultDefaultDropsNonDefault(t *testing.T) {
	ds := studyRecord("1.2.3", "PID1", "Doe^Jane", "20200101", "ACC1", "CT")
	q := QueryRequest{Level: QueryStudies}
	out := projectResult(ds, q)
	if _, ok := out.GetString(dicom.TagStudyInstanceUID); !ok {
		t.Error("default projection dropped StudyInstanceUID")
	}
	if _, ok := out.GetString(dicom.TagBodyPartExamined); ok {
		t.Error("default projection kept BodyPartExamined, a non-default attribute")
	}
}

func TestProjectResultIncludeFieldKeeps(t *testing.T) {
	ds := studyRecord("1.2.3", "PID1", "Doe^Jane", "20200101", "ACC1", "CT")
	q := QueryRequest{Level: QueryStudies, IncludeFields: []dicom.Tag{dicom.TagBodyPartExamined}}
	out := projectResult(ds, q)
	if _, ok := out.GetString(dicom.TagBodyPartExamined); !ok {
		t.Error("includefield projection dropped the requested BodyPartExamined")
	}
}

func TestProjectResultIncludeAllKeepsEverything(t *testing.T) {
	ds := studyRecord("1.2.3", "PID1", "Doe^Jane", "20200101", "ACC1", "CT")
	q := QueryRequest{Level: QueryStudies, IncludeAll: true}
	out := projectResult(ds, q)
	if _, ok := out.GetString(dicom.TagBodyPartExamined); !ok {
		t.Error("includefield=all projection dropped BodyPartExamined")
	}
}

func TestProjectResultKeepsMatchKey(t *testing.T) {
	ds := studyRecord("1.2.3", "PID1", "Doe^Jane", "20200101", "ACC1", "CT")
	// PatientID is a study default, so match instead on a non-default to prove the matched
	// key is always returned.
	q := QueryRequest{
		Level: QueryStudies,
		Match: []MatchKey{{Tag: dicom.TagBodyPartExamined, VR: dicom.VRCS, Value: "BRAIN"}},
	}
	out := projectResult(ds, q)
	if _, ok := out.GetString(dicom.TagBodyPartExamined); !ok {
		t.Error("projection dropped the matched-on BodyPartExamined")
	}
}

// TestQIDOStudySearchRoundTrip is the end-to-end study search: the client posts a
// PatientID match and the server returns the matching study as application/dicom+json,
// which the client parses back into a dataset.
func TestQIDOStudySearchRoundTrip(t *testing.T) {
	backend := &memQuery{candidates: []*dicom.DataSet{
		studyRecord("1.2.3", "PID1", "Doe^Jane", "20200101", "ACC1", "CT"),
		studyRecord("4.5.6", "PID2", "Roe^John", "20210101", "ACC2", "MR"),
	}}
	hs := newQueryTestServer(t, backend)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudies(context.Background(), SearchQuery{Match: map[string]string{"PatientID": "PID1"}})
	if err != nil {
		t.Fatalf("SearchStudies: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	gotPID, _ := results[0].DataSet.GetString(dicom.TagPatientID)
	if gotPID != "PID1" {
		t.Fatalf("result PatientID = %q, want PID1", gotPID)
	}
}

// TestQIDOContentTypeAndStatus asserts a successful QIDO-RS search emits
// application/dicom+json with a 200 status.
func TestQIDOContentTypeAndStatus(t *testing.T) {
	backend := &memQuery{candidates: []*dicom.DataSet{
		studyRecord("1.2.3", "PID1", "Doe^Jane", "20200101", "ACC1", "CT"),
	}}
	hs := newQueryTestServer(t, backend)

	resp, err := hs.Client().Get(hs.URL + "/studies?PatientID=PID1")
	if err != nil {
		t.Fatalf("GET /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != mediaTypeDICOMJSON {
		t.Fatalf("Content-Type = %q, want %q", ct, mediaTypeDICOMJSON)
	}
}

// TestQIDONoMatchReturns204 asserts a search with no matches answers 204 No Content rather
// than a 200 empty array, so an empty result is unambiguous (PS3.18 §10.6.1.4).
func TestQIDONoMatchReturns204(t *testing.T) {
	backend := &memQuery{candidates: []*dicom.DataSet{
		studyRecord("1.2.3", "PID1", "Doe^Jane", "20200101", "ACC1", "CT"),
	}}
	hs := newQueryTestServer(t, backend)

	resp, err := hs.Client().Get(hs.URL + "/studies?PatientID=NOBODY")
	if err != nil {
		t.Fatalf("GET /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("no-match status = %d, want 204", resp.StatusCode)
	}
}

// TestQIDOTruncationEmitsWarning asserts a result set larger than the cap is truncated and
// the response carries the Warning: 299 header (PS3.18 §10.6.1.4).
func TestQIDOTruncationEmitsWarning(t *testing.T) {
	cands := make([]*dicom.DataSet, 5)
	for i := range cands {
		cands[i] = studyRecord("1.2."+string(rune('a'+i)), "PID", "Doe^Jane", "20200101", "ACC", "CT")
	}
	backend := &memQuery{candidates: cands}
	hs := newQueryTestServer(t, backend, WithMaxQueryResults(2))

	resp, err := hs.Client().Get(hs.URL + "/studies")
	if err != nil {
		t.Fatalf("GET /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if w := resp.Header.Get("Warning"); !strings.HasPrefix(w, "299 ") {
		t.Fatalf("Warning header = %q, want a 299 truncation warning", w)
	}
}

// TestQIDOBackendErrorFailsClosed asserts a backend error answers 500, never a 200 empty
// result a caller would read as "no matches" (PRD §9.2).
func TestQIDOBackendErrorFailsClosed(t *testing.T) {
	backend := &memQuery{err: errors.New("backend down")}
	hs := newQueryTestServer(t, backend)

	resp, err := hs.Client().Get(hs.URL + "/studies?PatientID=PID1")
	if err != nil {
		t.Fatalf("GET /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("backend-error status = %d, want 500", resp.StatusCode)
	}
}

// TestQIDOUnacceptableReturns406 asserts a QIDO-RS search whose Accept admits only XML
// answers 406, since the server serves application/dicom+json only.
func TestQIDOUnacceptableReturns406(t *testing.T) {
	backend := &memQuery{}
	hs := newQueryTestServer(t, backend)

	req, _ := http.NewRequest(http.MethodGet, hs.URL+"/studies?PatientID=PID1", http.NoBody)
	req.Header.Set("Accept", "application/dicom+xml")
	resp, err := hs.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("XML-only Accept status = %d, want 406", resp.StatusCode)
	}
}

// TestQIDONotImplementedWhenNoBackend asserts a QIDO-RS search with no query backend
// answers 501, never a 200 empty result.
func TestQIDONotImplementedWhenNoBackend(t *testing.T) {
	srv, err := NewServer(WithStoreBackend(newMemStore()))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	resp, err := hs.Client().Get(hs.URL + "/studies?PatientID=PID1")
	if err != nil {
		t.Fatalf("GET /studies: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("no-backend QIDO status = %d, want 501", resp.StatusCode)
	}
}

// TestQIDOSeriesSearchScopedByStudy asserts the series search under a study reaches the
// backend with the study scope set and returns the series-level default attributes.
func TestQIDOSeriesSearchScopedByStudy(t *testing.T) {
	series := dicom.NewDataSet()
	series.SetString(dicom.TagStudyInstanceUID, "1.2.3")
	series.SetString(dicom.TagSeriesInstanceUID, "1.2.3.4")
	series.SetString(dicom.TagModality, "CT")
	series.SetString(dicom.TagSeriesNumber, "1")
	backend := &memQuery{candidates: []*dicom.DataSet{series}}
	hs := newQueryTestServer(t, backend)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchSeries(context.Background(), "1.2.3", SearchQuery{})
	if err != nil {
		t.Fatalf("SearchSeries: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	gotSeries, _ := results[0].DataSet.GetString(dicom.TagSeriesInstanceUID)
	if gotSeries != "1.2.3.4" {
		t.Fatalf("result SeriesInstanceUID = %q, want 1.2.3.4", gotSeries)
	}
}

// TestSearchInstancesRejectsSeriesWithoutStudy asserts the client rejects an
// instance search that names a series scope but no study scope, before any request.
func TestSearchInstancesRejectsSeriesWithoutStudy(t *testing.T) {
	c, err := NewClient("https://pacs.example.org")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.SearchInstances(context.Background(), "", "1.2.3.4", SearchQuery{}); err == nil {
		t.Fatal("SearchInstances with a series but no study returned nil error")
	}
}

// TestQIDORangeMatchOverHTTP asserts a StudyDate range query filters at the server and
// returns only the in-range study.
func TestQIDORangeMatchOverHTTP(t *testing.T) {
	backend := &memQuery{candidates: []*dicom.DataSet{
		studyRecord("1.2.3", "PID1", "Doe^Jane", "20200615", "ACC1", "CT"),
		studyRecord("4.5.6", "PID2", "Roe^John", "20251231", "ACC2", "MR"),
	}}
	hs := newQueryTestServer(t, backend)
	c, err := NewClient(hs.URL, WithHTTPClient(hs.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	results, err := c.SearchStudies(context.Background(),
		SearchQuery{Match: map[string]string{"StudyDate": "20200101-20201231"}})
	if err != nil {
		t.Fatalf("SearchStudies: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("range query results = %d, want 1", len(results))
	}
	gotStudy, _ := results[0].DataSet.GetString(dicom.TagStudyInstanceUID)
	if gotStudy != "1.2.3" {
		t.Fatalf("range result study = %q, want 1.2.3", gotStudy)
	}
}

func TestEncodeSearchQueryDeterministic(t *testing.T) {
	q := SearchQuery{
		Match:         map[string]string{"PatientID": "PID1", "AccessionNumber": "ACC1"},
		IncludeFields: []string{"BodyPartExamined"},
		Limit:         10,
	}
	got := encodeSearchQuery(q).Encode()
	// url.Values.Encode sorts keys, so the output is stable; assert the keys are present.
	for _, want := range []string{"PatientID=PID1", "AccessionNumber=ACC1", "includefield=BodyPartExamined", "limit=10"} {
		if !strings.Contains(got, want) {
			t.Errorf("encoded query %q missing %q", got, want)
		}
	}
}
