package dicomweb

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/codeninja55/go-radx/dicom"
)

// studyReturnAttributes are the default Study-level return attributes a QIDO-RS study
// search emits when includefield does not widen the projection (PS3.18 Table 10.6.1-5).
// The set is the union of the required matching keys and the required return keys for the
// study level.
var studyReturnAttributes = []dicom.Tag{
	dicom.TagStudyDate,
	dicom.TagStudyTime,
	dicom.TagAccessionNumber,
	dicom.TagModalitiesInStudy,
	dicom.TagReferringPhysicianName,
	dicom.TagPatientName,
	dicom.TagPatientID,
	dicom.TagPatientBirthDate,
	dicom.TagPatientSex,
	dicom.TagStudyInstanceUID,
	dicom.TagStudyID,
	dicom.TagNumberOfStudyRelatedSeries,
	dicom.TagNumberOfStudyRelatedInstances,
}

// seriesReturnAttributes are the default Series-level return attributes (PS3.18 Table
// 10.6.1-5a): the series identity and count plus the study identity that scopes it.
var seriesReturnAttributes = []dicom.Tag{
	dicom.TagStudyInstanceUID,
	dicom.TagModality,
	dicom.TagSeriesInstanceUID,
	dicom.TagSeriesNumber,
	dicom.TagSeriesDescription,
	dicom.TagNumberOfSeriesRelatedInstances,
}

// instanceReturnAttributes are the default Instance-level return attributes (PS3.18 Table
// 10.6.1-5b): the instance identity plus the study and series identity that scope it.
var instanceReturnAttributes = []dicom.Tag{
	dicom.TagStudyInstanceUID,
	dicom.TagSeriesInstanceUID,
	dicom.TagSOPClassUID,
	dicom.TagSOPInstanceUID,
	dicom.TagInstanceNumber,
}

// defaultReturnAttributes returns the level's default return-attribute set (PS3.18 Tables
// 10.6.1-5/-5a/-5b). It is the projection applied when includefield neither names extra
// fields nor requests all attributes.
func defaultReturnAttributes(level QueryLevel) []dicom.Tag {
	switch level {
	case QuerySeries:
		return seriesReturnAttributes
	case QueryInstances:
		return instanceReturnAttributes
	default:
		return studyReturnAttributes
	}
}

// handleQuery answers a QIDO-RS search at the study, series, or instance level. It
// negotiates application/dicom+json, parses and validates the query, asks the backend for
// candidates, applies attribute matching and includefield projection, pages the result,
// and emits the Warning: 299 header when the page is truncated by the result cap. A
// backend error fails the query closed (500) rather than returning an empty result a
// caller would read as "no matches" (PRD §9.2).
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request, segs []string) {
	if s.query == nil {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "QIDO-RS search is not implemented")
		return
	}
	if !negotiateDICOMJSON(r.Header.Get("Accept")) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"QIDO-RS search serves application/dicom+json only")
		return
	}

	q, err := parseQueryRequest(segs, r.URL.Query())
	if err != nil {
		status, cause := queryErrorStatus(err)
		s.writeProblem(w, r, status, cause, "invalid QIDO-RS query")
		return
	}

	candidates, err := s.query.Query(r.Context(), q)
	if err != nil {
		// A backend failure must not read as an empty result set; fail closed (PRD §9.2).
		s.writeProblem(w, r, http.StatusInternalServerError, err, "the query backend failed")
		return
	}

	matched := make([]*dicom.DataSet, 0, len(candidates))
	for _, ds := range candidates {
		if matchDataSet(ds, q) {
			matched = append(matched, projectResult(ds, q))
		}
	}

	page, truncated := pageResults(matched, q.Offset, q.Limit, s.maxQueryResults)
	body, err := marshalResults(page)
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot encode the query results")
		return
	}

	w.Header().Set("Content-Type", mediaTypeDICOMJSON)
	if truncated {
		// PS3.18 §10.6.1.4: a result set truncated to the maximum cap carries a Warning: 299
		// so the caller never reads the page as the complete result. The text is fixed and
		// PHI-free.
		w.Header().Set("Warning", warningResultsTruncated)
	}
	if len(page) == 0 {
		// PS3.18 §10.6.1.4: a search with no matches returns 204 No Content, not a 200 with an
		// empty array, so an empty result is unambiguous.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body) // #nosec G705 -- Content-Type is application/dicom+json (set above), not an HTML sink
}

// warningResultsTruncated is the HTTP Warning header value for a QIDO-RS response capped
// at the maximum result count (PS3.18 §10.6.1.4). It names the cause structurally and
// carries no PHI.
const warningResultsTruncated = `299 dicomweb "There are more results than the maximum number returned; refine the query or page with offset/limit"`

// queryErrorStatus maps a parse error to its HTTP status and the classifying sentinel for
// the problem document. A *QueryError names both; any other error is a 400 bad request.
func queryErrorStatus(err error) (int, error) {
	if qe, ok := errors.AsType[*QueryError](err); ok {
		return qe.Status, qe.cause
	}
	return http.StatusBadRequest, err
}

// projectResult returns the dataset reduced to the attributes the query asks to return:
// the level's default set plus any includefield attributes, or every attribute when
// includefield=all. The matching keys are always returned, so a caller sees the values it
// constrained on (PS3.18 §10.6.1.5). The projection always copies, never aliases the
// backend's dataset.
func projectResult(ds *dicom.DataSet, q QueryRequest) *dicom.DataSet {
	if q.IncludeAll {
		return ds.Clone()
	}
	want := make(map[dicom.Tag]struct{})
	for _, t := range defaultReturnAttributes(q.Level) {
		want[t] = struct{}{}
	}
	for _, t := range q.IncludeFields {
		want[t] = struct{}{}
	}
	for _, mk := range q.Match {
		want[mk.Tag] = struct{}{}
	}
	out := dicom.NewDataSet()
	for e := range ds.All() {
		if _, ok := want[e.Tag]; ok {
			out.Set(cloneElement(e))
		}
	}
	return out
}

// cloneElement deep-copies an element through a single-element dataset clone, so a
// projected result never aliases the backend's stored value (Codex DCM-016).
func cloneElement(e dicom.Element) dicom.Element {
	tmp := dicom.NewDataSet()
	tmp.Set(e)
	cloned := tmp.Clone()
	out, _ := cloned.Get(e.Tag)
	return out
}

// pageResults applies the offset/limit paging window to the matched results and reports
// whether the result set was truncated by the maximum cap. The effective limit is the
// caller's limit clamped to maxCap (a zero limit means the cap), so a query can never
// stream an unbounded result set (PRD §9.3). truncated is true when results beyond the
// returned page exist because the cap was reached, signalling the Warning: 299 header.
func pageResults(matched []*dicom.DataSet, offset, limit, maxCap int) (page []*dicom.DataSet, truncated bool) {
	if maxCap <= 0 {
		maxCap = defaultMaxQIDOResults
	}
	if offset >= len(matched) {
		return nil, false
	}
	rest := matched[offset:]

	effective := limit
	if effective <= 0 || effective > maxCap {
		effective = maxCap
	}
	if len(rest) > effective {
		return rest[:effective], true
	}
	return rest, false
}

// marshalResults encodes the result page as the application/dicom+json array QIDO-RS
// returns: a JSON array of DICOM-JSON attribute objects, one per matched resource (PS3.18
// §F.2.2). An empty page marshals to an empty array, though the caller answers 204 rather
// than emit it.
func marshalResults(page []*dicom.DataSet) ([]byte, error) {
	parts := make([]json.RawMessage, 0, len(page))
	for _, ds := range page {
		raw, err := MarshalJSON(ds)
		if err != nil {
			return nil, err
		}
		parts = append(parts, raw)
	}
	return json.Marshal(parts)
}
