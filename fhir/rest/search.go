package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/fhir"
)

// linkRelationNext and linkRelationPrevious are the FHIR Bundle navigation relations the paging
// loop follows. The relation codes are stable across R4 and R5, so they are matched as literal
// strings rather than a release enum.
const (
	linkRelationNext     = "next"
	linkRelationPrevious = "previous"
	linkRelationSelf     = "self"
)

// SearchParams builds the query string for a type-level search. It models the common, correctly
// supported FHIR search features: ordinary name=value parameters, modifiers (name:modifier),
// chained parameters (reference.target), and _include / _revinclude. A parameter name may appear
// more than once (FHIR ANDs repeated parameters), so Add appends rather than replaces; Set replaces
// every value for a name. The zero value is an empty, ready-to-use search.
//
// What is supported, and the documented limit: single-level chaining (Patient?general-practitioner.name=)
// and reverse chaining via _has are expressible by passing the chained name verbatim to Add; the
// client does not parse or validate the chain semantically, it forwards the parameter the FHIR
// server interprets. Deep multi-hop chaining beyond what the server's SearchParameter definitions
// allow is the server's concern, not the client's — the client transmits whatever chained name the
// caller supplies. _include and _revinclude (including the :iterate and wildcard forms) are added
// through Include / RevInclude, which append the parameter the server interprets.
type SearchParams struct {
	values url.Values
}

// NewSearchParams returns an empty SearchParams ready to accumulate parameters.
func NewSearchParams() *SearchParams {
	return &SearchParams{values: url.Values{}}
}

// Add appends a value for a search parameter, so repeated names AND together per FHIR search
// semantics (Patient?language=en&language=fr is two AND constraints). The name may carry a modifier
// (status:not) or be a chained name (subject.name); the client forwards it verbatim.
func (p *SearchParams) Add(name, value string) *SearchParams {
	p.ensure()
	p.values.Add(name, value)
	return p
}

// Set replaces every value for a search parameter with the single value given, for a parameter that
// should appear once (such as _count).
func (p *SearchParams) Set(name, value string) *SearchParams {
	p.ensure()
	p.values.Set(name, value)
	return p
}

// Modifier adds a parameter with a type modifier, the name:modifier=value form (for example
// name:exact=Jones, status:not=entered-in-error). It is sugar over Add that composes the
// "name:modifier" token, so a caller need not assemble the colon form by hand.
func (p *SearchParams) Modifier(name, modifier, value string) *SearchParams {
	return p.Add(name+":"+modifier, value)
}

// Chain adds a chained search parameter, the reference.target=value form (for example
// general-practitioner.name=Smith on a Patient search, which constrains the referenced
// Practitioner). The chain is the dotted token; the client forwards it for the server to resolve
// against its SearchParameter definitions. Multi-segment chains are passed as the full dotted name.
func (p *SearchParams) Chain(reference, target, value string) *SearchParams {
	return p.Add(reference+"."+target, value)
}

// Include adds an _include parameter, pulling referenced resources into the result Bundle (for
// example _include=Observation:subject). The value is the SourceType:searchParam[:targetType] token
// FHIR defines; an :iterate form is passed verbatim.
func (p *SearchParams) Include(value string) *SearchParams {
	return p.Add("_include", value)
}

// RevInclude adds a _revinclude parameter, pulling resources that reference the matches into the
// result Bundle (for example _revinclude=Observation:subject on a Patient search).
func (p *SearchParams) RevInclude(value string) *SearchParams {
	return p.Add("_revinclude", value)
}

// Count sets the _count page-size hint, the number of matches the server returns per page. It is a
// hint the server may cap; paging follows the Bundle links regardless of the value.
func (p *SearchParams) Count(n int) *SearchParams {
	return p.Set("_count", strconv.Itoa(n))
}

// Sort sets the _sort parameter (a comma-separated list of search parameter names, each optionally
// prefixed with "-" for descending).
func (p *SearchParams) Sort(value string) *SearchParams {
	return p.Set("_sort", value)
}

// Values returns the accumulated parameters as url.Values, so a caller can inspect or forward them.
// The returned map is the live backing map; mutate it through the SearchParams methods rather than
// directly to keep the builder's intent.
func (p *SearchParams) Values() url.Values {
	p.ensure()
	return p.values
}

// encode renders the parameters as a sorted query string, so the request URL is deterministic
// (stable across runs for a given set of parameters), which keeps tests and logs reproducible.
func (p *SearchParams) encode() string {
	if p == nil || len(p.values) == 0 {
		return ""
	}
	return p.values.Encode()
}

func (p *SearchParams) ensure() {
	if p.values == nil {
		p.values = url.Values{}
	}
}

// SearchPage is one page of a type-level search result: the resources of this page (the matches and
// any _include/_revinclude resources the server added) and the navigation links to the next and
// previous pages. A caller iterates pages by calling FollowNext while HasNext is true, or calls
// SearchAll to collect every page.
type SearchPage struct {
	// Bundle is the concrete release searchset Bundle this page decoded from, behind the
	// fhir.Resource interface; a caller narrows it with fhir.As for the full Bundle (entry.search
	// mode, total, and the rest).
	Bundle fhir.Resource

	// Resources are the resources the page's entries carry, in entry order. For a searchset this is
	// the matches plus any _include/_revinclude resources; the caller distinguishes match from
	// include via the Bundle's entry.search.mode when it needs to.
	Resources []fhir.Resource

	// SelfURL, NextURL, and PrevURL are the Bundle's self/next/previous link URLs, or "" when the
	// Bundle carries no such link. NextURL drives FollowNext.
	SelfURL string
	NextURL string
	PrevURL string

	// RequestURL is the absolute URL of the request that produced this page. FollowNext/FollowPrev
	// resolve a relative NextURL/PrevURL against it (RFC 3986 reference resolution), so a query-only
	// continuation link ("?page=2") resolves against the search endpoint's path, not the service root.
	RequestURL string
}

// HasNext reports whether the page carries a "next" link, so a caller can write
// `for page.HasNext() { page, err = c.FollowNext(ctx, page) }`.
func (p *SearchPage) HasNext() bool { return p.NextURL != "" }

// HasPrev reports whether the page carries a "previous" link.
func (p *SearchPage) HasPrev() bool { return p.PrevURL != "" }

// Search executes a type-level search (GET [type]?params) and returns the first page of results. A
// nil params searches with no constraints. The result is a searchset Bundle decoded into the
// client's release; follow page.NextURL with FollowNext to page through a multi-page result, or use
// SearchAll to collect every page. A non-2xx status maps to a typed error.
func (c *Client) Search(ctx context.Context, resourceType string, params *SearchParams) (*SearchPage, error) {
	if strings.TrimSpace(resourceType) == "" {
		return nil, fmt.Errorf("fhir/rest: Search requires a resourceType")
	}
	path := resourceType
	if q := params.encode(); q != "" {
		path = resourceType + "?" + q
	}
	return c.searchPath(ctx, path)
}

// FollowNext fetches the next page of a paged search by following the current page's "next" link.
// It returns ErrNoNextPage (wrapped) when the page has no next link, so a paging loop terminates
// cleanly. The link is the server's own next URL — absolute, origin-relative, relative-path, or
// query-only — resolved against the URL of the request that produced this page (RFC 3986 reference
// resolution), which is what makes paging robust to a server that encodes an opaque continuation
// token in the link in any of those forms.
func (c *Client) FollowNext(ctx context.Context, page *SearchPage) (*SearchPage, error) {
	if page == nil || page.NextURL == "" {
		return nil, ErrNoNextPage
	}
	next, err := resolveLink(page.RequestURL, page.NextURL)
	if err != nil {
		return nil, err
	}
	return c.searchPath(ctx, next)
}

// FollowPrev fetches the previous page by following the "previous" link, the symmetric counterpart
// of FollowNext. It returns ErrNoNextPage when the page has no previous link.
func (c *Client) FollowPrev(ctx context.Context, page *SearchPage) (*SearchPage, error) {
	if page == nil || page.PrevURL == "" {
		return nil, ErrNoNextPage
	}
	prev, err := resolveLink(page.RequestURL, page.PrevURL)
	if err != nil {
		return nil, err
	}
	return c.searchPath(ctx, prev)
}

// resolveLink resolves a Bundle navigation link against the absolute URL of the request that
// produced the page, per RFC 3986. requestURL is the absolute URL searchPath fetched; link is the
// server-supplied next/previous URL. ResolveReference handles every form uniformly: an absolute link
// is returned as-is, an origin-relative link ("/fhir/Patient?page=2") resolves to the origin plus
// path, a relative-path link ("Patient?page=2") resolves relative to the request path, and a
// query-only link ("?page=2") keeps the request path and replaces only the query — so a continuation
// link pages from the search endpoint, never the service root. A requestURL or link that does not
// parse is an error rather than a silently misrouted request.
func resolveLink(requestURL, link string) (string, error) {
	base, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("fhir/rest: paging request URL does not parse: %w", err)
	}
	ref, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("fhir/rest: paging link does not parse: %w", err)
	}
	return base.ResolveReference(ref).String(), nil
}

// SearchAll executes a search and follows every "next" link to collect the resources of all pages
// into one slice, bounded by maxPages so a server returning an unbounded chain of pages cannot loop
// forever (a maxPages of 0 applies a conservative default). It is the convenience over the manual
// FollowNext loop for a caller that wants the whole result set in memory; a caller streaming a large
// result pages manually instead. The context cancels the whole walk.
func (c *Client) SearchAll(ctx context.Context, resourceType string, params *SearchParams, maxPages int) ([]fhir.Resource, error) {
	if maxPages <= 0 {
		maxPages = defaultMaxSearchPages
	}
	page, err := c.Search(ctx, resourceType, params)
	if err != nil {
		return nil, err
	}
	out := append([]fhir.Resource(nil), page.Resources...)
	for pages := 1; page.HasNext() && pages < maxPages; pages++ {
		page, err = c.FollowNext(ctx, page)
		if err != nil {
			return nil, err
		}
		out = append(out, page.Resources...)
	}
	if page.HasNext() {
		return out, fmt.Errorf("fhir/rest: search exceeded the %d-page limit; results are truncated", maxPages)
	}
	return out, nil
}

// defaultMaxSearchPages bounds an unbounded SearchAll walk when the caller passes 0. It is generous
// for a realistic paged result yet finite so a server returning a self-referential next link cannot
// loop forever.
const defaultMaxSearchPages = 1000

// searchPath issues a GET against path (a relative type?query or an absolute Bundle.link URL),
// decodes the searchset Bundle, and lifts its navigation links into a SearchPage. It is shared by
// Search and the FollowNext/FollowPrev paging methods, so the decode and link extraction are
// written once.
func (c *Client) searchPath(ctx context.Context, path string) (*SearchPage, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != http.StatusOK {
		return nil, c.errorForResponse(http.MethodGet, path, resp)
	}
	r, err := c.decodeResource(resp)
	if err != nil {
		return nil, err
	}
	view, ok := asBundleView(r)
	if !ok {
		return nil, fmt.Errorf("fhir/rest: search response is a %s, not a Bundle", r.ResourceType())
	}
	return &SearchPage{
		Bundle:     r,
		Resources:  view.resources,
		SelfURL:    view.linkURL(linkRelationSelf),
		NextURL:    view.linkURL(linkRelationNext),
		PrevURL:    view.linkURL(linkRelationPrevious),
		RequestURL: c.resolveURL(path),
	}, nil
}
