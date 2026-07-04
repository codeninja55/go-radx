package dicomweb

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// QueryBackend is the server-side pluggable query source for QIDO-RS (PS3.18 §10.6). It
// is segregated from the store and retrieve backends so a query-only deployment
// implements only this interface (ISP, PRD §8.2). A backend returns the candidate
// datasets at the requested level; the server applies attribute matching, includefield
// projection, and paging on top, so a backend may return a superset and rely on the
// server to filter. Returning a typed error fails the query closed (HTTP 500) rather than
// reporting an empty result the caller would read as "no matches" (PRD §9.2).
type QueryBackend interface {
	Query(ctx context.Context, q QueryRequest) ([]*dicom.DataSet, error)
}

// QueryLevel is the QIDO-RS search level: study, series, or instance. It mirrors the
// resource Level but is the query-facing name so a backend reasons about the search
// target rather than a URL path.
type QueryLevel int

const (
	// QueryStudies searches /studies.
	QueryStudies QueryLevel = iota
	// QuerySeries searches /studies/{study}/series or /series.
	QuerySeries
	// QueryInstances searches the instance level.
	QueryInstances
)

// String renders the query level for diagnostics.
func (l QueryLevel) String() string {
	switch l {
	case QueryStudies:
		return "studies"
	case QuerySeries:
		return "series"
	case QueryInstances:
		return "instances"
	default:
		return "unknown"
	}
}

// MatchKey is one attribute-matching constraint parsed from a QIDO-RS query parameter
// (PS3.18 §10.6.1). The Tag names the attribute; Value is the raw query value, which the
// matcher interprets per the attribute's VR (single-value, UID list, range, or wildcard).
type MatchKey struct {
	Tag   dicom.Tag
	VR    dicom.VR
	Value string
}

// QueryRequest is the parsed, validated QIDO-RS request a QueryBackend receives. It
// carries the search level, the parent UIDs that scope a series/instance search, the
// attribute-matching keys, the projected return attributes, and the paging window. A
// backend that logs it must redact the Match values, which can carry patient identifiers
// (PRD §9.1).
type QueryRequest struct {
	// Level is the search level.
	Level QueryLevel
	// StudyUID scopes a series or instance search to one study (from the URL path); empty
	// for an all-studies search.
	StudyUID dicom.UID
	// SeriesUID scopes an instance search to one series (from the URL path); empty
	// otherwise.
	SeriesUID dicom.UID
	// Match lists the attribute-matching constraints, deduplicated by tag.
	Match []MatchKey
	// IncludeFields lists the additional return attributes requested by includefield.
	IncludeFields []dicom.Tag
	// IncludeAll is true when includefield=all was requested: every available attribute is
	// returned, not only the level's default set.
	IncludeAll bool
	// Fuzzy is true when fuzzymatching=true was requested.
	Fuzzy bool
	// Limit is the maximum number of results to return (0 means the server default cap).
	Limit int
	// Offset is the number of leading results to skip.
	Offset int
}

// defaultMaxQIDOResults caps the number of results a QIDO-RS search returns when the
// caller sets no limit, and is the hard ceiling a caller's limit is clamped to. It bounds
// a query that would otherwise stream an unbounded result set into memory (PRD §9.3). A
// response truncated by this cap carries the Warning: 299 header so the caller never reads
// a truncated page as the complete result (PS3.18 §10.6.1.4).
const defaultMaxQIDOResults = 5000

// parseQueryRequest parses a QIDO-RS request: the level and parent UIDs come from the URL
// path segments, the matching keys and control parameters from the query string. A
// malformed parameter (an unknown attribute, a non-numeric limit/offset, a bad parent
// UID) is rejected with a typed error rather than silently ignored, so a query never runs
// against a different constraint than the caller asked for (PRD §9.2). It never returns a
// PHI-bearing error: a rejected attribute is named by keyword, never by its value.
func parseQueryRequest(segs []string, query url.Values) (QueryRequest, error) {
	q := QueryRequest{}
	if err := q.applyPathScope(segs); err != nil {
		return QueryRequest{}, err
	}
	for key, vals := range query {
		if len(vals) == 0 {
			continue
		}
		v := vals[0]
		switch strings.ToLower(key) {
		case "includefield":
			if err := q.applyIncludeField(vals); err != nil {
				return QueryRequest{}, err
			}
		case "limit":
			n, err := parseNonNegative(v, "limit")
			if err != nil {
				return QueryRequest{}, err
			}
			q.Limit = n
		case "offset":
			n, err := parseNonNegative(v, "offset")
			if err != nil {
				return QueryRequest{}, err
			}
			q.Offset = n
		case "fuzzymatching":
			q.Fuzzy = parseBool(v)
		default:
			mk, err := parseMatchKey(key, v)
			if err != nil {
				return QueryRequest{}, err
			}
			q.addMatch(mk)
		}
	}
	return q, nil
}

// applyPathScope sets the level and parent UIDs from the URL path segments, validating any
// parent UID the path names. The recognised search targets are /studies,
// /studies/{study}/series, /series, /studies/{study}/series/{series}/instances,
// /studies/{study}/instances, and /instances (PS3.18 §10.6.1.1).
func (q *QueryRequest) applyPathScope(segs []string) error {
	switch {
	case len(segs) == 1 && segs[0] == "studies":
		q.Level = QueryStudies
	case len(segs) == 1 && segs[0] == "series":
		q.Level = QuerySeries
	case len(segs) == 1 && segs[0] == "instances":
		q.Level = QueryInstances
	case len(segs) == 3 && segs[0] == "studies" && segs[2] == "series":
		q.Level = QuerySeries
		if err := q.setStudyScope(segs[1]); err != nil {
			return err
		}
	case len(segs) == 3 && segs[0] == "studies" && segs[2] == "instances":
		q.Level = QueryInstances
		if err := q.setStudyScope(segs[1]); err != nil {
			return err
		}
	case len(segs) == 5 && segs[0] == "studies" && segs[2] == "series" && segs[4] == "instances":
		q.Level = QueryInstances
		if err := q.setStudyScope(segs[1]); err != nil {
			return err
		}
		if err := q.setSeriesScope(segs[3]); err != nil {
			return err
		}
	default:
		return &QueryError{Status: http.StatusNotImplemented, cause: ErrUnsupported,
			detail: "not a QIDO-RS search resource"}
	}
	return nil
}

func (q *QueryRequest) setStudyScope(uid string) error {
	if err := validateUID(dicom.UID(uid), "StudyInstanceUID"); err != nil {
		return &QueryError{Status: http.StatusBadRequest, cause: err, detail: "invalid study UID in the request URL"}
	}
	q.StudyUID = dicom.UID(uid)
	return nil
}

func (q *QueryRequest) setSeriesScope(uid string) error {
	if err := validateUID(dicom.UID(uid), "SeriesInstanceUID"); err != nil {
		return &QueryError{Status: http.StatusBadRequest, cause: err, detail: "invalid series UID in the request URL"}
	}
	q.SeriesUID = dicom.UID(uid)
	return nil
}

// applyIncludeField records the requested return attributes. includefield=all sets the
// IncludeAll flag; any other value is one or more attribute keywords or GGGGEEEE tag
// strings, comma-separated or repeated. An unresolvable attribute is rejected rather than
// silently dropped, so a caller never receives a projection narrower than it asked for.
func (q *QueryRequest) applyIncludeField(vals []string) error {
	for _, raw := range vals {
		for field := range strings.SplitSeq(raw, ",") {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			if strings.EqualFold(field, "all") {
				q.IncludeAll = true
				continue
			}
			tag, err := resolveAttribute(field)
			if err != nil {
				return err
			}
			q.IncludeFields = append(q.IncludeFields, tag)
		}
	}
	return nil
}

// addMatch records a matching key, replacing any earlier key for the same tag so a
// repeated parameter does not stack contradictory constraints.
func (q *QueryRequest) addMatch(mk MatchKey) {
	for i := range q.Match {
		if q.Match[i].Tag == mk.Tag {
			q.Match[i] = mk
			return
		}
	}
	q.Match = append(q.Match, mk)
}

// parseMatchKey resolves a query-parameter name to a matching key. The name is an
// attribute keyword or a GGGGEEEE tag string. A name that resolves to no known attribute
// is rejected; its value is never echoed in the error (PRD §9.1).
func parseMatchKey(name, value string) (MatchKey, error) {
	tag, err := resolveAttribute(name)
	if err != nil {
		return MatchKey{}, err
	}
	return MatchKey{Tag: tag, VR: dictVRForMatch(tag), Value: value}, nil
}

// resolveAttribute resolves an attribute reference (a dictionary keyword or a GGGGEEEE
// hex tag) to its tag. It rejects an unknown reference with a typed, value-free error.
func resolveAttribute(ref string) (dicom.Tag, error) {
	if tag, ok := dicom.LookupKeyword(ref); ok {
		return tag, nil
	}
	if tag, ok := parseHexTag(ref); ok {
		return tag, nil
	}
	return 0, &QueryError{Status: http.StatusBadRequest, cause: ErrInvalidResource,
		detail: "unknown query attribute " + safeAttributeName(ref)}
}

// parseHexTag parses an eight-hex-digit GGGGEEEE tag string (the DICOM-JSON key form). It
// rejects the zero tag (0000,0000) and any group-length element (element 0000): neither
// is a queryable attribute, so accepting one would let a query name a non-attribute.
func parseHexTag(s string) (dicom.Tag, bool) {
	if len(s) != 8 {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, false
	}
	tag := dicom.Tag(n)
	if tag == 0 || tag.IsGroupLength() {
		return 0, false
	}
	return tag, true
}

// dictVRForMatch resolves the VR used to interpret a matching value, defaulting to UI for
// an unknown attribute so it is matched as a UID list rather than a wildcard string.
func dictVRForMatch(t dicom.Tag) dicom.VR {
	if info, ok := dicom.Lookup(t); ok {
		return info.VR
	}
	return dicom.VRUI
}

// safeAttributeName returns an attribute reference for an error message only when it is a
// structural token (a keyword or a hex tag); any other input is replaced with a fixed
// placeholder, so a query-parameter name an attacker crafted to carry data is never
// echoed back (PRD §9.1).
func safeAttributeName(ref string) string {
	if isStructuralAttributeName(ref) {
		return ref
	}
	return "(redacted)"
}

// isStructuralAttributeName reports whether ref is a plain attribute keyword or hex tag:
// ASCII letters and digits only, bounded length.
func isStructuralAttributeName(ref string) bool {
	if ref == "" || len(ref) > 64 {
		return false
	}
	for _, r := range ref {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

// parseNonNegative parses a non-negative integer query parameter, rejecting a negative or
// non-numeric value with a typed error naming the parameter (never its value).
func parseNonNegative(s, name string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, &QueryError{Status: http.StatusBadRequest, cause: ErrInvalidResource,
			detail: "invalid " + name + " parameter"}
	}
	return n, nil
}

// parseBool reports whether a query value is the literal "true" (case-insensitive). Any
// other value is false: an absent or malformed flag never silently enables a behaviour.
func parseBool(s string) bool {
	return strings.EqualFold(strings.TrimSpace(s), "true")
}

// QueryError carries a QIDO-RS request fault with the HTTP status it maps to and a
// PHI-free structural detail. It unwraps to the package sentinel that classifies the
// fault so callers can check with errors.Is.
type QueryError struct {
	Status int
	cause  error
	detail string
}

func (e *QueryError) Error() string {
	if e.detail != "" {
		return "dicomweb: QIDO-RS request rejected: " + e.detail
	}
	return "dicomweb: QIDO-RS request rejected"
}

// Unwrap exposes the classifying sentinel (ErrInvalidResource, ErrUnsupported, ...).
func (e *QueryError) Unwrap() error { return e.cause }
