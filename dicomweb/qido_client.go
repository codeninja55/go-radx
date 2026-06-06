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

// search issues the QIDO-RS GET against path with the query encoded from q, parses the
// application/dicom+json result array, and returns the matched datasets. A 204 No Content
// is the empty result set (nil slice, nil error); a non-success status is a typed
// *HTTPError with its query stripped (PRD §9.1).
func (c *Client) search(ctx context.Context, path string, q SearchQuery) ([]SearchResult, error) {
	full := path + "?" + encodeSearchQuery(q).Encode()
	req, err := c.newRequest(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", mediaTypeDICOMJSON)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(http.MethodGet, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.httpError(http.MethodGet, path, resp)
	}

	raw, err := io.ReadAll(c.boundedBody(resp))
	if err != nil {
		return nil, c.readError(http.MethodGet, path, err)
	}
	return parseSearchResults(raw)
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
