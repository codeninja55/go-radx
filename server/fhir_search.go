package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/fhir"
)

// This file is the server role's search-depth layer over the Repository: it adds the search-result
// surface the FHIR REST spec defines beyond a bare match list — paging via Bundle.link (next/self/
// prev) with a _count page size, _include/_revinclude reference expansion, and one-hop chained
// parameters — on top of whatever matching the configured Repository performs. The handler owns this
// orchestration (not the Repository) because paging links are absolute URLs derived from the request,
// and include/chain resolution is a release-neutral graph walk over resources the Repository already
// stores; keeping it here lets the same logic serve an R4 or an R5 role and lets a production
// Repository supply richer base matching without re-implementing the result-shaping rules.
//
// Scope (matched to FHIR R5 search.html and HAPI's plain-server semantics):
//   - Paging: _count caps the page; the response carries self and, when more matches remain, next
//     (and prev when not on the first page). The cursor is an _offset the next link round-trips.
//   - _include / _revinclude: one level deep. :iterate (recursive include) is OUT of scope and a
//     params carrying it is ignored for the iterate hop (documented), not an error.
//   - Chained parameters: one hop (Observation?subject:Patient.name=... or the typeless
//     Observation?subject.name=...). Reverse chaining (_has) is OUT of scope and ignored.
//
// No method logs the query string or a parameter value, which can carry PHI (PRD §9.1): the search
// log line names the interaction and the resource type only.

const (
	// defaultSearchCount is the page size used when a request sends no _count, matching HAPI's
	// default page size of 50 (FHIR leaves the default to the server).
	defaultSearchCount = 50
	// maxSearchCount caps _count so a hostile or careless client cannot ask for an unbounded page
	// that materialises the whole store into one response. A _count above the cap is clamped down to
	// it, the same bounding HAPI applies to its page size.
	maxSearchCount = 200
	// searchOffsetParam is the cursor parameter the next/prev links round-trip. FHIR does not mandate
	// a cursor spelling (the next link is opaque to the client), so an explicit, self-describing
	// _offset is used: the client never constructs it, only follows the link the server emits.
	searchOffsetParam = "_offset"
	searchCountParam  = "_count"
	searchSortParam   = "_sort"
	includeParam      = "_include"
	revIncludeParam   = "_revinclude"
)

// searchLink is one Bundle.link the searchset carries: a relation (self, next, prev) and the absolute
// URL that realises it. It is release-neutral; the adapter renders it into the release's BundleLink.
type searchLink struct {
	relation string
	url      string
}

const (
	linkRelationSelf = "self"
	linkRelationNext = "next"
	linkRelationPrev = "prev"
)

// handleSearch serves a type-level search with the full result surface: it resolves any one-hop
// chained parameters against the repository, asks the repository for the base matches, pages the
// result with _count and an _offset cursor (emitting self/next/prev Bundle.link), expands
// _include/_revinclude into include-mode entries, and writes the searchset Bundle. A repository
// failure is mapped to an OperationOutcome; the result shaping itself never fails a well-formed
// search.
func (h *fhirHandler) handleSearch(w http.ResponseWriter, r *http.Request, resourceType string) {
	query := r.URL.Query()

	matches, err := h.resolveBaseMatches(r.Context(), resourceType, query)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}

	page, links := h.pageMatches(r, resourceType, query, matches)

	includes, err := h.expandIncludes(r.Context(), resourceType, query, page)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}

	bundle, err := h.adapter.newSearchSetWithLinks(int32(len(matches)), page, includes, links) // #nosec G115 -- an in-memory match count is far below int32
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, issueTypeException, "the server could not build the search bundle")
		return
	}
	h.logger.Info("fhir search", zap.String("type", resourceType), zap.String("interaction", "search-type"))
	h.writeResource(w, r, http.StatusOK, bundle, "")
}

// resolveBaseMatches returns the resources that match the request's non-result parameters (everything
// that is not _count/_offset/_include/_revinclude), with any one-hop chained parameter resolved first
// into a constraint the repository can answer. The base matching is the Repository's: the chained
// hops are resolved here (a chained reference is dereferenced against the store, then the base type is
// filtered to those whose reference points at a resolved id), and the remaining plain parameters
// (including _sort) are forwarded to Repository.Search.
//
// Ordering follows the FHIR REST search contract (search.html#sort): when the request carries _sort
// the Repository owns the result order, so the order it returns is preserved exactly. Only when no
// _sort is given does this layer impose a deterministic id sort, so paging is stable over a map-backed
// store whose iteration order is otherwise unspecified — an id sort applied unconditionally would
// silently discard the Repository's _sort order.
func (h *fhirHandler) resolveBaseMatches(ctx context.Context, resourceType string, query url.Values) ([]fhir.Resource, error) {
	plain, chains := splitChainedParams(query)

	matchedByChain, err := h.resolveChains(ctx, resourceType, chains)
	if err != nil {
		return nil, err
	}

	bundle, err := h.repo.Search(ctx, resourceType, plain)
	if err != nil {
		return nil, err
	}
	matches := resourcesFromBundleVia(bundle, h.adapter.unmarshalResource)

	if matchedByChain != nil {
		matches = filterByIDSet(h.adapter, matches, matchedByChain)
	}
	if query.Get(searchSortParam) == "" {
		sortResourcesByID(h.adapter, matches)
	}
	return matches, nil
}

// pageMatches applies the _count page size and the _offset cursor to the full match list and returns
// the page plus its Bundle.link set (self always; next when matches remain after the page; prev when
// the page does not start at the first match). The links are absolute URLs built from the request so
// they resolve when the searchset travels beyond the connection (R5 bundle.html), and the next link
// round-trips: following it returns the next page. _count is clamped to [0, maxSearchCount].
func (h *fhirHandler) pageMatches(r *http.Request, resourceType string, query url.Values, matches []fhir.Resource) ([]fhir.Resource, []searchLink) {
	count := clampCount(query.Get(searchCountParam))
	offset := parseOffset(query.Get(searchOffsetParam))

	if offset > len(matches) {
		offset = len(matches)
	}
	end := offset + count
	if end > len(matches) {
		end = len(matches)
	}
	page := matches[offset:end]

	links := []searchLink{{relation: linkRelationSelf, url: h.searchLinkURL(r, resourceType, query, offset, count)}}
	// A next link is emitted only when this page advanced the cursor (end > offset) AND matches remain
	// past it. _count=0 is the FHIR count-only request (search.html#count): it returns total with no
	// entries, and a next link whose _offset equals this page's would not advance, looping the client
	// on the same empty page forever — so no next link is emitted.
	if end > offset && end < len(matches) {
		links = append(links, searchLink{relation: linkRelationNext, url: h.searchLinkURL(r, resourceType, query, end, count)})
	}
	if offset > 0 {
		prev := offset - count
		if prev < 0 {
			prev = 0
		}
		links = append(links, searchLink{relation: linkRelationPrev, url: h.searchLinkURL(r, resourceType, query, prev, count)})
	}
	return page, links
}

// searchLinkURL builds an absolute self/next/prev link for a search page: the request's scheme and
// host, the type's search path, the request's original query parameters, and the page's _offset and
// _count overwritten so the link names exactly this page. The query is rebuilt from the parsed values
// (not the raw string) so the offset/count are normalised and a follow of the link reproduces the
// page deterministically.
func (h *fhirHandler) searchLinkURL(r *http.Request, resourceType string, query url.Values, offset, count int) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	q := cloneValues(query)
	q.Set(searchCountParam, strconv.Itoa(count))
	if offset > 0 {
		q.Set(searchOffsetParam, strconv.Itoa(offset))
	} else {
		q.Del(searchOffsetParam)
	}
	return scheme + "://" + r.Host + h.resourceLocation(resourceType, "") + "?" + encodeValuesSorted(q)
}

// expandIncludes resolves the request's _include and _revinclude parameters against the page of
// matches and returns the referenced (or referencing) resources as the searchset's include entries,
// deduplicated and with the matches themselves excluded (a match is never also an include). One level
// deep only: :iterate is not followed. An _include reads the matches' references and fetches their
// targets; an _revinclude scans the named source type for resources whose reference points back at a
// match.
func (h *fhirHandler) expandIncludes(ctx context.Context, resourceType string, query url.Values, page []fhir.Resource) ([]fhir.Resource, error) {
	matchKeys := map[string]struct{}{}
	for _, m := range page {
		matchKeys[resourceKey(m.ResourceType(), h.adapter.resourceID(m))] = struct{}{}
	}
	seen := map[string]struct{}{}
	var includes []fhir.Resource

	add := func(res fhir.Resource) {
		key := resourceKey(res.ResourceType(), h.adapter.resourceID(res))
		if _, isMatch := matchKeys[key]; isMatch {
			return
		}
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		includes = append(includes, res)
	}

	for _, spec := range query[includeParam] {
		incl, ok := parseIncludeSpec(spec)
		if !ok || incl.sourceType != resourceType {
			continue
		}
		param, ok := lookupSearchParam(resourceType, incl.param)
		if !ok || !param.isReference {
			continue
		}
		for _, m := range page {
			for _, ref := range referenceValues(m, param.jsonPath) {
				refType, refID := splitReference(ref)
				if refType == "" || refID == "" {
					continue
				}
				// An _include target-type modifier (Observation:derived-from:ImagingStudy,
				// search.html#include) constrains a multi-target reference to one type: include only the
				// referenced resources of that type, not every type the reference can point at.
				if incl.targetType != "" && refType != incl.targetType {
					continue
				}
				target, err := h.repo.Read(ctx, refType, refID)
				if err != nil {
					continue // a dangling or deleted reference is skipped, not a search failure
				}
				add(target)
			}
		}
	}

	for _, spec := range query[revIncludeParam] {
		incl, ok := parseIncludeSpec(spec)
		if !ok || incl.targetType != "" && incl.targetType != resourceType {
			continue
		}
		param, ok := lookupSearchParam(incl.sourceType, incl.param)
		if !ok || !param.isReference {
			continue
		}
		sources, err := h.allOfType(ctx, incl.sourceType)
		if err != nil {
			return nil, err
		}
		for _, src := range sources {
			for _, ref := range referenceValues(src, param.jsonPath) {
				refType, refID := splitReference(ref)
				if refType != resourceType {
					continue
				}
				if _, isMatch := matchKeys[resourceKey(refType, refID)]; isMatch {
					add(src)
					break
				}
			}
		}
	}
	return includes, nil
}

// resolveChains resolves one-hop chained parameters to the set of base-resource ids that satisfy
// them. A chained parameter names a reference search parameter on the base type, an optional target
// type modifier, and a parameter on the target (Observation?subject:Patient.name=X or the typeless
// subject.name=X). It is resolved by finding the target resources that match the target parameter,
// then keeping the base resources whose reference points at one of them.
//
// The return distinguishes "no constraint" from "constrained to zero", which the caller must not
// conflate (FHIR REST search.html#chaining and the documented out-of-scope stance): a nil result
// means no SUPPORTED chain produced a constraint — there were no chains, or every dotted parameter was
// out of scope (an unknown/non-reference head, or a multi-hop / unrecognised target parameter that
// resolves on no candidate target type) — so the base matches must be returned unfiltered (the
// parameter is ignored, not applied). A non-nil result (possibly empty) means at least one supported
// chain ran: an empty set then legitimately filters every match out. Returning an empty non-nil set
// for an ignored chain would turn "ignored" into "no results", excluding matches a lenient server must
// keep. Reverse chaining (_has) is out of scope and never reaches here.
func (h *fhirHandler) resolveChains(ctx context.Context, resourceType string, chains []chainedParam) (map[string]struct{}, error) {
	if len(chains) == 0 {
		return nil, nil
	}
	var result map[string]struct{} // nil until a supported chain contributes a constraint
	for _, chain := range chains {
		param, ok := lookupSearchParam(resourceType, chain.refParam)
		if !ok || !param.isReference {
			continue // an unknown or non-reference chain head is out of scope: ignore, do not constrain
		}
		// Determine the candidate target types: the explicit modifier when present, else the
		// reference parameter's declared targets.
		targetTypes := param.targets
		if chain.targetType != "" {
			targetTypes = []string{chain.targetType}
		}
		matchedTargets := map[string]struct{}{}
		supported := false // the target parameter resolved on at least one candidate target type
		for _, tt := range targetTypes {
			if !isWorkflowResourceType(tt) {
				continue
			}
			tparam, ok := lookupSearchParam(tt, chain.targetParam)
			if !ok {
				continue // a multi-hop or unrecognised target parameter is not a supported single hop
			}
			supported = true
			targets, err := h.allOfType(ctx, tt)
			if err != nil {
				return nil, err
			}
			for _, t := range targets {
				if resourceMatchesValue(t, tparam, chain.value) {
					matchedTargets[resourceKey(tt, h.adapter.resourceID(t))] = struct{}{}
				}
			}
		}
		if !supported {
			continue // the chain resolves to no single hop: out of scope, ignore rather than exclude all
		}
		// Keep the base resources whose reference points at a matched target.
		bases, err := h.allOfType(ctx, resourceType)
		if err != nil {
			return nil, err
		}
		thisChain := map[string]struct{}{}
		for _, b := range bases {
			for _, ref := range referenceValues(b, param.jsonPath) {
				refType, refID := splitReference(ref)
				if _, ok := matchedTargets[resourceKey(refType, refID)]; ok {
					thisChain[h.adapter.resourceID(b)] = struct{}{}
					break
				}
			}
		}
		if result == nil {
			result = thisChain
			continue
		}
		result = intersectIDSets(result, thisChain) // multiple supported chains AND together
	}
	return result, nil
}

// allOfType returns every resource of resourceType the repository holds, by issuing a parameterless
// type-level search. It is how the include/revinclude/chain resolution reads the resource graph: the
// Repository is the single source of truth, so a production Repository's contents are walked exactly
// as the dev MemoryRepository's are.
func (h *fhirHandler) allOfType(ctx context.Context, resourceType string) ([]fhir.Resource, error) {
	if !isWorkflowResourceType(resourceType) {
		return nil, nil
	}
	bundle, err := h.repo.Search(ctx, resourceType, url.Values{})
	if err != nil {
		return nil, err
	}
	return resourcesFromBundleVia(bundle, h.adapter.unmarshalResource), nil
}

// ---- parameter parsing ----

// chainedParam is one parsed one-hop chained parameter: the reference parameter on the base type, an
// optional target-type modifier (the ":Patient" in subject:Patient), the parameter name on the
// target, and the value to match. Only single-hop chains are represented; a deeper chain is left in
// the plain params for the Repository (or ignored).
type chainedParam struct {
	refParam    string
	targetType  string
	targetParam string
	value       string
}

// splitChainedParams separates a query's parameters into the plain parameters the Repository searches
// on and the one-hop chained parameters this layer resolves. A parameter name containing a "." is a
// chain: the segment before the dot is the reference parameter (with an optional :Type modifier) and
// the segment after is the target parameter. The result/control parameters (_count, _offset,
// _include, _revinclude) are dropped from the plain set so they never reach the Repository as search
// criteria. A multi-hop chain (more than one dot) is not represented here; only its first split is
// taken, and resolution ignores it if the head is not a known reference — documented out of scope.
func splitChainedParams(query url.Values) (plain url.Values, chains []chainedParam) {
	plain = url.Values{}
	for name, values := range query {
		switch name {
		case searchCountParam, searchOffsetParam, includeParam, revIncludeParam:
			continue
		}
		if dot := strings.IndexByte(name, '.'); dot >= 0 {
			head := name[:dot]
			targetParam := name[dot+1:]
			refParam, targetType := head, ""
			if colon := strings.IndexByte(head, ':'); colon >= 0 {
				refParam, targetType = head[:colon], head[colon+1:]
			}
			for _, v := range values {
				chains = append(chains, chainedParam{
					refParam:    refParam,
					targetType:  targetType,
					targetParam: targetParam,
					value:       v,
				})
			}
			continue
		}
		for _, v := range values {
			plain.Add(name, v)
		}
	}
	return plain, chains
}

// includeSpec is a parsed _include or _revinclude value: SourceType:param[:targetType]. For an
// _include the source is the base type and the param's targets are read from the registry; for an
// _revinclude the source is the type to scan and targetType (when present) constrains which reference
// target counts.
type includeSpec struct {
	sourceType string
	param      string
	targetType string
}

// parseIncludeSpec parses an _include/_revinclude value of the form "Type:param" or
// "Type:param:targetType". A value that is not at least Type:param (or the wildcard "*", which is out
// of scope) is rejected with ok=false so it is ignored rather than mis-resolved. An :iterate modifier
// is not supported; a spec carrying it is rejected so the iterate hop is not silently half-applied.
func parseIncludeSpec(spec string) (includeSpec, bool) {
	if strings.Contains(spec, ":iterate") || strings.HasSuffix(spec, ":*") || spec == "*" {
		return includeSpec{}, false
	}
	parts := strings.Split(spec, ":")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return includeSpec{}, false
	}
	out := includeSpec{sourceType: parts[0], param: parts[1]}
	if len(parts) >= 3 {
		out.targetType = parts[2]
	}
	return out, true
}

// clampCount parses the _count page size and clamps it to [0, maxSearchCount], defaulting to
// defaultSearchCount when absent, non-numeric, or negative. A _count of 0 is honoured (an empty page
// with the honest total and a next link), the FHIR "count me but return no resources" request.
func clampCount(raw string) int {
	if raw == "" {
		return defaultSearchCount
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return defaultSearchCount
	}
	if n > maxSearchCount {
		return maxSearchCount
	}
	return n
}

// parseOffset parses the _offset cursor, defaulting to 0 when absent, non-numeric, or negative. The
// client never constructs the offset; it follows the next/prev link the server emits, so a malformed
// offset simply starts at the first match rather than erroring.
func parseOffset(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// ---- release-neutral resource reads (JSON-based, like the rest of the role) ----

// resourcesFromBundleVia decodes a searchset Bundle's entry resources via the given release decoder.
// It is how the search layer recovers the individual resources from the Bundle the Repository returns
// so it can page, include, and chain over them. An entry with no resource is skipped.
// The Bundle is marshalled and each entry.resource is re-decoded, so the result is the release's
// concrete resource type behind fhir.Resource — exactly what the include/chain reads and the final
// bundle builder expect.
func resourcesFromBundleVia(bundle fhir.Resource, decode func([]byte) (fhir.Resource, error)) []fhir.Resource {
	data, err := json.Marshal(bundle)
	if err != nil {
		return nil
	}
	var env struct {
		Entry []struct {
			Resource json.RawMessage `json:"resource"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil
	}
	var out []fhir.Resource
	for _, e := range env.Entry {
		if len(e.Resource) == 0 || string(e.Resource) == "null" {
			continue
		}
		res, err := decode(e.Resource)
		if err != nil {
			continue
		}
		out = append(out, res)
	}
	return out
}

// referenceValues reads the reference string(s) at a JSON path within a resource, the release-neutral
// way the include/chain resolution dereferences a reference search parameter. jsonPath is a
// dot-separated path to a Reference element (for example "subject" or "basedOn"); the element may be a
// single Reference or an array of References, so both shapes are read. Each Reference's "reference"
// string (for example "Patient/123") is returned; a Reference with only an identifier (no literal
// reference) yields nothing, because it cannot be dereferenced by id.
func referenceValues(resource fhir.Resource, jsonPath string) []string {
	data, err := json.Marshal(resource)
	if err != nil {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil
	}
	raw, ok := obj[jsonPath]
	if !ok {
		return nil
	}
	return referenceStrings(raw)
}

// referenceStrings extracts the "reference" literal(s) from a raw JSON value that is either a single
// Reference object or an array of them. Anything else yields nothing.
func referenceStrings(raw json.RawMessage) []string {
	type ref struct {
		Reference string `json:"reference"`
	}
	var one ref
	if err := json.Unmarshal(raw, &one); err == nil && one.Reference != "" {
		return []string{one.Reference}
	}
	var many []ref
	if err := json.Unmarshal(raw, &many); err == nil {
		var out []string
		for _, r := range many {
			if r.Reference != "" {
				out = append(out, r.Reference)
			}
		}
		return out
	}
	return nil
}

// resourceMatchesValue reports whether a resource matches a search parameter's value, the predicate
// the chained-parameter target filter uses. A token/string parameter matches when the value at the
// parameter's JSON path equals (string/token) or contains (HumanName) the wanted value; the match is
// deliberately simple — exact for token, substring-anywhere for a name — because the dev resolution
// proves the chain plumbing, and a production Repository performs full parameter matching. An
// _id-style parameter matches the resource's logical id.
func resourceMatchesValue(resource fhir.Resource, param searchParam, want string) bool {
	if param.isID {
		return resourceIDViaJSON(resource) == want
	}
	data, err := json.Marshal(resource)
	if err != nil {
		return false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return false
	}
	raw, ok := obj[param.jsonPath]
	if !ok {
		return false
	}
	if param.isHumanName {
		return humanNameContains(raw, want)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s == want
	}
	return false
}

// humanNameContains reports whether any HumanName at the raw value (a single HumanName or an array)
// carries the wanted text in its family, given, or text — the substring match a name search performs.
// The comparison is case-insensitive, matching FHIR string-parameter default semantics (starts-with
// in the spec; substring here is a superset that keeps the dev match forgiving).
func humanNameContains(raw json.RawMessage, want string) bool {
	type humanName struct {
		Text   string   `json:"text"`
		Family string   `json:"family"`
		Given  []string `json:"given"`
	}
	want = strings.ToLower(want)
	check := func(n humanName) bool {
		if strings.Contains(strings.ToLower(n.Text), want) || strings.Contains(strings.ToLower(n.Family), want) {
			return true
		}
		for _, g := range n.Given {
			if strings.Contains(strings.ToLower(g), want) {
				return true
			}
		}
		return false
	}
	var one humanName
	if err := json.Unmarshal(raw, &one); err == nil && check(one) {
		return true
	}
	var many []humanName
	if err := json.Unmarshal(raw, &many); err == nil {
		for _, n := range many {
			if check(n) {
				return true
			}
		}
	}
	return false
}

// ---- id-set helpers ----

// resourceKey composes the "Type/id" key used to dedupe and look up resources across the include and
// chain resolution. It is the same shape the repository's store key uses.
func resourceKey(resourceType, id string) string { return resourceType + "/" + id }

// splitReference splits a literal reference ("Patient/123", possibly with a leading base URL or a
// version suffix) into its type and id. The last "Type/id" pair is taken so an absolute reference
// (http://host/fhir/Patient/123) and a relative one (Patient/123) both resolve; a version suffix
// (Patient/123/_history/2) is trimmed so a versioned reference still resolves to the resource.
func splitReference(ref string) (resourceType, id string) {
	ref = strings.TrimSpace(ref)
	if idx := strings.Index(ref, "/_history/"); idx >= 0 {
		ref = ref[:idx]
	}
	segs := strings.Split(strings.Trim(ref, "/"), "/")
	if len(segs) < 2 {
		return "", ""
	}
	return segs[len(segs)-2], segs[len(segs)-1]
}

// filterByIDSet keeps the resources whose logical id is in the set, the chained-parameter filter
// applied to the base matches.
func filterByIDSet(adapter releaseAdapter, resources []fhir.Resource, ids map[string]struct{}) []fhir.Resource {
	out := resources[:0:0]
	for _, r := range resources {
		if _, ok := ids[adapter.resourceID(r)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// intersectIDSets returns the ids present in both sets, the AND of multiple chained parameters.
func intersectIDSets(a, b map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for id := range a {
		if _, ok := b[id]; ok {
			out[id] = struct{}{}
		}
	}
	return out
}

// sortResourcesByID sorts resources by logical id so paging is stable: a map-backed Repository
// iterates in an unspecified order, so without a deterministic sort a next link could re-show or skip
// a resource. The sort is numeric-aware so server-minted ids ("2" before "10") page in their natural
// order.
func sortResourcesByID(adapter releaseAdapter, resources []fhir.Resource) {
	sort.SliceStable(resources, func(i, j int) bool {
		return lessID(adapter.resourceID(resources[i]), adapter.resourceID(resources[j]))
	})
}

// lessID compares two logical ids, numerically when both are numeric (so "2" sorts before "10") and
// lexically otherwise, the stable order paging relies on.
func lessID(a, b string) bool {
	an, aerr := strconv.Atoi(a)
	bn, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil {
		return an < bn
	}
	return a < b
}

// ---- query encoding ----

// cloneValues makes a deep copy of a url.Values so the link builder can overwrite _offset/_count
// without mutating the request's own parsed query.
func cloneValues(in url.Values) url.Values {
	out := url.Values{}
	for k, vs := range in {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

// encodeValuesSorted encodes url.Values with the keys (and each key's values) in a stable sorted
// order, so a self/next link is byte-identical run to run for the same query — a stable link the
// client and tests can compare. url.Values.Encode already sorts by key; this wrapper exists as the
// single encode point so the link spelling stays consistent if the encoding is ever customised.
func encodeValuesSorted(v url.Values) string {
	return v.Encode()
}
