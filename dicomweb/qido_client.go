package dicomweb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// SearchQuery is the client-side QIDO-RS query a caller builds for SearchStudies,
// SearchSeries, or SearchInstances. It carries the attribute-matching keys, the projected
// return attributes, and the paging window. The match values may carry patient
// identifiers, so a SearchQuery is never logged in full (PRD §9.1).
type SearchQuery struct {
	// Match maps an attribute keyword (or GGGGEEEE tag string) to its matching value, e.g.
	// {"PatientID": "12345"} or {"StudyDate": "20200101-20201231"}.
	Match map[string]string
	// IncludeFields names additional return attributes (keywords or GGGGEEEE tag strings)
	// beyond the level's default set.
	IncludeFields []string
	// IncludeAll requests every available attribute (includefield=all).
	IncludeAll bool
	// Fuzzy requests fuzzy matching (fuzzymatching=true).
	Fuzzy bool
	// Limit caps the number of results returned (0 leaves it to the origin's default).
	Limit int
	// Offset skips the given number of leading results.
	Offset int
}

// SearchResult is one QIDO-RS match: the DICOM-JSON attribute object parsed into a
// dataset. The returned attributes are exactly those the origin projected for the search
// level plus any includefield attributes.
type SearchResult struct {
	DataSet *dicom.DataSet
}

// SearchStudies issues a study-level QIDO-RS search against /studies and parses the
// application/dicom+json result array. A search with no matches returns an empty slice
// and a nil error (the origin's 204 No Content), never an error a caller would confuse
// with a transport failure.
func (c *Client) SearchStudies(ctx context.Context, q SearchQuery) ([]SearchResult, error) {
	return c.search(ctx, "/studies", q)
}

// SearchSeries issues a series-level QIDO-RS search. With study set it searches
// /studies/{study}/series (the series of one study); with study empty it searches the
// all-studies /series relative resource.
func (c *Client) SearchSeries(ctx context.Context, study dicom.UID, q SearchQuery) ([]SearchResult, error) {
	path, err := seriesSearchPath(study)
	if err != nil {
		return nil, err
	}
	return c.search(ctx, path, q)
}

// SearchInstances issues an instance-level QIDO-RS search. The path is scoped by the
// non-empty parents: /studies/{study}/series/{series}/instances,
// /studies/{study}/instances, or the all-studies /instances.
func (c *Client) SearchInstances(ctx context.Context, study, series dicom.UID, q SearchQuery) ([]SearchResult, error) {
	path, err := instanceSearchPath(study, series)
	if err != nil {
		return nil, err
	}
	return c.search(ctx, path, q)
}

// seriesSearchPath builds the series-search path for the optional study scope, validating
// the study UID when present so a malformed identifier never reaches the wire.
func seriesSearchPath(study dicom.UID) (string, error) {
	if study == "" {
		return "/series", nil
	}
	if err := validateUID(study, "StudyInstanceUID"); err != nil {
		return "", err
	}
	return "/studies/" + string(study) + "/series", nil
}

// instanceSearchPath builds the instance-search path for the optional study/series scope.
// A series scope requires a study scope; a series without a study is rejected as an
// out-of-order path (ErrInvalidResource).
func instanceSearchPath(study, series dicom.UID) (string, error) {
	switch {
	case study == "" && series == "":
		return "/instances", nil
	case study == "" && series != "":
		return "", fmt.Errorf("%w: series scope present without a study scope", ErrInvalidResource)
	}
	if err := validateUID(study, "StudyInstanceUID"); err != nil {
		return "", err
	}
	if series == "" {
		return "/studies/" + string(study) + "/instances", nil
	}
	if err := validateUID(series, "SeriesInstanceUID"); err != nil {
		return "", err
	}
	return "/studies/" + string(study) + "/series/" + string(series) + "/instances", nil
}

// SearchStudiesAll is the auto-paginating form of SearchStudies (dicomweb-client's
// get_remaining): it walks the offset/limit window until the origin reports the result
// set exhausted and returns every match in one slice. q.Limit is the page size (0 leaves
// each page to the origin's default) and q.Offset the starting point. The walk continues
// while the origin signals additional results with the Warning: 299 header (PS3.18
// §10.6.1.4) or fills the requested page. With a page size set (q.Limit > 0) it stops on
// a short page served without that warning — a conformant origin that truncated would
// have sent it; with no page size it walks until an empty page, matching get_remaining,
// so a silently capped origin is never misread as exhausted. A page that fails to
// advance (an origin ignoring the offset parameter) aborts the walk with the accumulated
// results and an error. maxPages bounds the walk so an origin endlessly reporting more
// results cannot loop forever (0 applies a conservative default); exceeding it returns
// the accumulated results together with a truncation error, never a silent partial set
// (PRD §9.2). The context cancels the whole walk. A caller streaming a very large result
// set pages manually with Limit/Offset instead.
func (c *Client) SearchStudiesAll(ctx context.Context, q SearchQuery, maxPages int) ([]SearchResult, error) {
	return c.searchAll(ctx, "/studies", q, maxPages)
}

// SearchSeriesAll is the auto-paginating form of SearchSeries, with SearchStudiesAll's
// paging semantics.
func (c *Client) SearchSeriesAll(ctx context.Context, study dicom.UID, q SearchQuery, maxPages int) ([]SearchResult, error) {
	path, err := seriesSearchPath(study)
	if err != nil {
		return nil, err
	}
	return c.searchAll(ctx, path, q, maxPages)
}

// SearchInstancesAll is the auto-paginating form of SearchInstances, with
// SearchStudiesAll's paging semantics.
func (c *Client) SearchInstancesAll(ctx context.Context, study, series dicom.UID, q SearchQuery, maxPages int) ([]SearchResult, error) {
	path, err := instanceSearchPath(study, series)
	if err != nil {
		return nil, err
	}
	return c.searchAll(ctx, path, q, maxPages)
}

// defaultMaxQIDOSearchPages bounds an auto-paginating search when the caller passes 0.
// It is generous for a realistic paged result yet finite, so an origin that always
// signals more results cannot drive an unbounded walk (PRD §9.3).
const defaultMaxQIDOSearchPages = 1000

// searchAll walks the offset window behind the *All search methods, accumulating pages
// until exhaustion, the page bound, a non-advancing page, or a page error. Each page is
// one searchPage request under the caller's context.
func (c *Client) searchAll(ctx context.Context, path string, q SearchQuery, maxPages int) ([]SearchResult, error) {
	if maxPages <= 0 {
		maxPages = defaultMaxQIDOSearchPages
	}
	var out []SearchResult
	var prev pageFingerprint
	havePrev := false
	for page := 0; page < maxPages; page++ {
		results, more, err := c.searchPage(ctx, path, q)
		if err != nil {
			return nil, err
		}
		if len(results) == 0 {
			return out, nil
		}
		fp := fingerprintPage(results)
		if havePrev && fp == prev {
			// The origin served the same page again despite the advanced offset: walking on
			// would only accumulate duplicates, so abort with what is genuinely collected.
			return out, fmt.Errorf(
				"dicomweb: QIDO-RS search page did not advance (the origin ignored the paging offset); results are truncated")
		}
		prev, havePrev = fp, true
		out = append(out, results...)
		if !more && q.Limit > 0 && len(results) < q.Limit {
			// The origin filled less than the requested page and did not warn of truncation:
			// the result set is exhausted (PS3.18 §10.6.1.4). With no page size the walk runs
			// to an empty page instead, since a short first page may just be the origin's own
			// cap.
			return out, nil
		}
		q.Offset += len(results)
	}
	return out, fmt.Errorf("dicomweb: QIDO-RS search exceeded the %d-page limit; results are truncated", maxPages)
}

// pageFingerprint is the cheap identity of one result page the non-advance check
// compares: the page length plus the identifying UID of its first and last result. Two
// consecutive pages with the same fingerprint mean the origin ignored the offset.
type pageFingerprint struct {
	count int
	first string
	last  string
}

func fingerprintPage(results []SearchResult) pageFingerprint {
	return pageFingerprint{
		count: len(results),
		first: resultIdentity(results[0]),
		last:  resultIdentity(results[len(results)-1]),
	}
}

// resultIdentity returns a result's most specific identifying UID (SOP, then series,
// then study), the attribute set every QIDO level's default return attributes carry.
func resultIdentity(r SearchResult) string {
	for _, tag := range []dicom.Tag{dicom.TagSOPInstanceUID, dicom.TagSeriesInstanceUID, dicom.TagStudyInstanceUID} {
		if v, ok := r.DataSet.GetString(tag); ok && v != "" {
			return v
		}
	}
	return ""
}

// SearchStudiesPage is the single-shot form of SearchStudies that also surfaces the
// origin's truncation signal: moreAvailable is true when the response carried the
// Warning: 299 additional-results header (PS3.18 §10.6.1.4), so a caller paging manually
// can tell an incomplete page from an exhausted result set without auto-pagination. An
// origin may omit the warning, so a false moreAvailable on a full page is not proof of
// exhaustion.
func (c *Client) SearchStudiesPage(ctx context.Context, q SearchQuery) ([]SearchResult, bool, error) {
	return c.searchPage(ctx, "/studies", q)
}

// SearchSeriesPage is the single-shot form of SearchSeries with SearchStudiesPage's
// truncation signal.
func (c *Client) SearchSeriesPage(ctx context.Context, study dicom.UID, q SearchQuery) ([]SearchResult, bool, error) {
	path, err := seriesSearchPath(study)
	if err != nil {
		return nil, false, err
	}
	return c.searchPage(ctx, path, q)
}

// SearchInstancesPage is the single-shot form of SearchInstances with
// SearchStudiesPage's truncation signal.
func (c *Client) SearchInstancesPage(ctx context.Context, study, series dicom.UID, q SearchQuery) ([]SearchResult, bool, error) {
	path, err := instanceSearchPath(study, series)
	if err != nil {
		return nil, false, err
	}
	return c.searchPage(ctx, path, q)
}

// search issues one QIDO-RS GET and returns the matched datasets, discarding the paging
// signal the auto-paginating walk consumes.
func (c *Client) search(ctx context.Context, path string, q SearchQuery) ([]SearchResult, error) {
	results, _, err := c.searchPage(ctx, path, q)
	return results, err
}

// searchPage issues the QIDO-RS GET against path with the query encoded from q, parses
// the application/dicom+json result array, and returns the matched datasets together with
// whether the origin signalled that more results exist beyond this page (the Warning: 299
// header, PS3.18 §10.6.1.4). A 204 No Content is the empty result set (nil slice, nil
// error); a non-success status is a typed *HTTPError with its query stripped (PRD §9.1).
func (c *Client) searchPage(ctx context.Context, path string, q SearchQuery) ([]SearchResult, bool, error) {
	full := path + "?" + encodeSearchQuery(q).Encode()
	req, err := c.newRequest(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", mediaTypeDICOMJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, c.transportError(http.MethodGet, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, c.httpError(http.MethodGet, path, resp)
	}

	raw, err := io.ReadAll(c.boundedBody(resp))
	if err != nil {
		return nil, false, c.readError(http.MethodGet, path, err)
	}
	results, err := parseSearchResults(raw)
	if err != nil {
		return nil, false, err
	}
	return results, hasMoreResultsWarning(resp.Header), nil
}

// hasMoreResultsWarning reports whether a response carries the PS3.18 §10.6.1.4 signal
// that the returned page was truncated and more results exist: a Warning element with
// warn-code 299 whose warn-text carries the additional-results semantic. Each Warning
// field is split into its warning-values on the commas between them (not the commas
// inside a quoted warn-text), and the text is required because 299 is HTTP's generic
// miscellaneous-persistent-warning code — origins use it for unrelated notices (a
// fuzzymatching downgrade, say) that must not read as "more results". The paging walk
// therefore leans primarily on the page-count heuristic; this warning only extends it.
func hasMoreResultsWarning(h http.Header) bool {
	for _, field := range h.Values("Warning") {
		for _, w := range splitWarningElements(field) {
			if warningSignalsMoreResults(w) {
				return true
			}
		}
	}
	return false
}

// splitWarningElements splits a Warning field value on the commas separating its
// warning-values, honouring quoted warn-texts (a comma inside the quoted string, or an
// escaped quote, does not split).
func splitWarningElements(field string) []string {
	var out []string
	inQuote, escaped := false, false
	start := 0
	for i := 0; i < len(field); i++ {
		switch {
		case escaped:
			escaped = false
		case field[i] == '\\' && inQuote:
			escaped = true
		case field[i] == '"':
			inQuote = !inQuote
		case field[i] == ',' && !inQuote:
			out = append(out, field[start:i])
			start = i + 1
		}
	}
	return append(out, field[start:])
}

// warningSignalsMoreResults reports whether one warning-value is the additional-results
// signal: warn-code exactly 299 (bounded, so 2990 does not match) with a warn-text
// naming additional or more results. The two phrasings cover the PS3.18 example text and
// this package's own server.
func warningSignalsMoreResults(w string) bool {
	w = strings.TrimSpace(w)
	if !strings.HasPrefix(w, "299") {
		return false
	}
	if len(w) > 3 && w[3] != ' ' && w[3] != '\t' {
		return false
	}
	text := strings.ToLower(w)
	return strings.Contains(text, "additional results") || strings.Contains(text, "more results")
}

// encodeSearchQuery renders a SearchQuery into url.Values. The match keys are emitted in
// sorted order so the encoded query is deterministic for a given input, which keeps tests
// stable and caches keyed on the URL predictable. The control parameters (includefield,
// fuzzymatching, limit, offset) are emitted only when they carry a non-default value.
func encodeSearchQuery(q SearchQuery) url.Values {
	v := url.Values{}
	keys := make([]string, 0, len(q.Match))
	for k := range q.Match {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v.Set(k, q.Match[k])
	}
	if q.IncludeAll {
		v.Set("includefield", "all")
	} else {
		for _, f := range q.IncludeFields {
			v.Add("includefield", f)
		}
	}
	if q.Fuzzy {
		v.Set("fuzzymatching", "true")
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	return v
}

// parseSearchResults decodes the QIDO-RS application/dicom+json result array into a slice
// of SearchResult. The body is a JSON array of DICOM-JSON attribute objects (PS3.18
// §F.2.2); each element is decoded through UnmarshalJSON. A body that is not a JSON array
// is a typed decode error rather than a silent empty result (PRD §9.2).
func parseSearchResults(raw []byte) ([]SearchResult, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, nil
	}
	var objs []json.RawMessage
	if err := json.Unmarshal(raw, &objs); err != nil {
		return nil, &DecodeError{Msg: "QIDO-RS response is not a JSON array"}
	}
	results := make([]SearchResult, 0, len(objs))
	for _, obj := range objs {
		ds, err := UnmarshalJSON(obj)
		if err != nil {
			return nil, fmt.Errorf("dicomweb: parse QIDO-RS result: %w", err)
		}
		results = append(results, SearchResult{DataSet: ds})
	}
	return results, nil
}
