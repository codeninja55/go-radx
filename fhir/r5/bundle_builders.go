// This file is HAND-WRITTEN, not generated. The generated set (bundle.go,
// operation_outcome.go, base.go, bindings.go, primitives.go, registry.go, and the
// per-resource files) models the Bundle structure from the StructureDefinition, but
// FHIR's per-type Bundle invariants (the bdl-* rules: where total is allowed, the
// document/message first-entry rules, request/response presence, fullUrl uniqueness)
// are prose constraints in the specification, not anything the StructureDefinition
// encodes. The typed builders below make those invariants unrepresentable-when-wrong:
// each builder produces a Bundle of exactly one type and validates the invariants up
// front, returning an error instead of an invalid Bundle. This is the one deliberate
// hand-written-per-release exception to "all generated"; the builders live outside the
// generated file set so gen:verify still reproduces the generated tree byte-for-byte.
//
// Concurrency / mutability (FHIR-015): a builder constructs a fresh Bundle, validates
// it, and returns it. There is no shared mutable builder state and no mutex; the
// prototype's mutex papered over a concurrency bug rather than fixing it. The rule is
// "build once, then treat as immutable": a returned Bundle is plain data that is safe
// to read concurrently, and a single goroutine owns it until it is published. Mutating
// a Bundle after a builder returns it bypasses the invariant checks, so a caller that
// must change a Bundle builds a new one rather than editing fields in place.

package r5

import (
	"errors"
	"fmt"

	"github.com/codeninja55/go-radx/fhir"
)

// ErrInvalidBundle is the sentinel wrapped by every builder rejection. A caller
// matches it with errors.Is to detect "this bundle violates a bdl-* invariant"
// without parsing the message; the wrapped text names the offending entry index and
// the rule, never patient data.
var ErrInvalidBundle = errors.New("fhir/r5: invalid bundle")

// SearchEntry is one entry of a searchset Bundle: the matched resource plus its
// optional search metadata (the match mode and relevance score). search metadata is
// permitted only in a searchset, so it is carried on this entry type alone and never
// reaches the other builders.
type SearchEntry struct {
	FullURL  string
	Resource fhir.Resource
	Mode     *SearchEntryMode
	Score    *fhir.Decimal
}

// TransactionEntry is one entry of a transaction or batch Bundle: the resource to
// act on plus the request line (HTTP verb and URL) the server processes. Every
// transaction/batch entry must carry a request, so the request fields are first-class
// here rather than optional.
type TransactionEntry struct {
	FullURL  string
	Resource fhir.Resource
	Method   HTTPVerb
	URL      string
	// IfNoneExist, IfMatch, IfNoneMatch, IfModifiedSince are the optional
	// conditional-operation headers; an unset header is omitted from the request.
	IfNoneExist     string
	IfMatch         string
	IfNoneMatch     string
	IfModifiedSince string
}

// DocumentEntry is one entry of a document Bundle after the leading Composition: a
// resource the Composition references (the subject, authors, sections' targets). A
// document carries no request/response/search metadata, so this entry is a plain
// fullUrl + resource pair.
type DocumentEntry struct {
	FullURL  string
	Resource fhir.Resource
}

// MessageEntry is one entry of a message Bundle after the leading MessageHeader: a
// resource the message conveys (the focus of the event). Like a document entry it
// carries no request/response/search metadata.
type MessageEntry struct {
	FullURL  string
	Resource fhir.Resource
}

// CollectionEntry is one entry of a collection Bundle: a fullUrl + resource pair
// with no request/response/search metadata. A collection is an unconstrained set of
// resources with no per-type semantics beyond fullUrl uniqueness.
type CollectionEntry struct {
	FullURL  string
	Resource fhir.Resource
}

// NewSearchSet builds a searchset Bundle. total is the reported total match count and
// is set because searchset is one of only two types for which total is meaningful
// (FHIR-010 — the prototype set total for every type). A negative total is rejected.
// Each entry's optional search metadata is carried through; search metadata is valid
// only in a searchset, which is exactly why it lives on SearchEntry. fullUrl values,
// when present, must be unique across the bundle.
func NewSearchSet(total int32, entries ...SearchEntry) (*Bundle, error) {
	if total < 0 {
		return nil, fmt.Errorf("%w: searchset total %d is negative", ErrInvalidBundle, total)
	}
	bundleEntries := make([]BundleEntry, 0, len(entries))
	for i := range entries {
		entry := entries[i]
		bundleEntry := BundleEntry{
			Resource: resourcePtr(entry.Resource),
			FullUrl:  fullURLPtr(entry.FullURL),
		}
		if entry.Mode != nil || entry.Score != nil {
			bundleEntry.Search = &BundleEntrySearch{Mode: entry.Mode, Score: entry.Score}
		}
		bundleEntries = append(bundleEntries, bundleEntry)
	}
	if err := checkUniqueFullURLs(bundleEntries); err != nil {
		return nil, err
	}
	bundleType := BundleTypeSearchset
	return &Bundle{Type: &bundleType, Total: &total, Entry: bundleEntries}, nil
}

// NewTransaction builds a transaction Bundle. Every entry must carry an HTTP verb and
// a non-empty URL (the bdl-* request-presence rule for transaction/batch); an entry
// with an unknown verb or empty URL is rejected naming its index. total is never set
// (FHIR-010), search metadata never appears, and fullUrl values must be unique.
func NewTransaction(entries ...TransactionEntry) (*Bundle, error) {
	return newRequestBundle(BundleTypeTransaction, entries...)
}

// NewBatch builds a batch Bundle. A batch has the same per-entry request-presence
// invariant as a transaction (every entry carries a verb and URL); the two differ
// only in server-side atomicity, which is not a structural invariant. total is never
// set, search metadata never appears, and fullUrl values must be unique.
func NewBatch(entries ...TransactionEntry) (*Bundle, error) {
	return newRequestBundle(BundleTypeBatch, entries...)
}

// newRequestBundle is the shared constructor for the two request-bearing bundle types
// (transaction and batch): both require a verb and a non-empty URL on every entry and
// forbid total and search metadata. The bundle type is the only difference, so it is a
// parameter rather than two near-identical builders.
func newRequestBundle(bundleType BundleType, entries ...TransactionEntry) (*Bundle, error) {
	bundleEntries := make([]BundleEntry, 0, len(entries))
	for i := range entries {
		entry := entries[i]
		if !validHTTPVerb(string(entry.Method)) {
			return nil, fmt.Errorf("%w: entry %d has invalid request method %q", ErrInvalidBundle, i, entry.Method)
		}
		if entry.URL == "" {
			return nil, fmt.Errorf("%w: entry %d is missing the required request.url", ErrInvalidBundle, i)
		}
		method := entry.Method
		request := &BundleEntryRequest{Method: &method, URL: &entry.URL}
		if entry.IfNoneExist != "" {
			request.IfNoneExist = strPtr(entry.IfNoneExist)
		}
		if entry.IfMatch != "" {
			request.IfMatch = strPtr(entry.IfMatch)
		}
		if entry.IfNoneMatch != "" {
			request.IfNoneMatch = strPtr(entry.IfNoneMatch)
		}
		if entry.IfModifiedSince != "" {
			request.IfModifiedSince = strPtr(entry.IfModifiedSince)
		}
		bundleEntries = append(bundleEntries, BundleEntry{
			Resource: resourcePtr(entry.Resource),
			FullUrl:  fullURLPtr(entry.FullURL),
			Request:  request,
		})
	}
	if err := checkUniqueFullURLs(bundleEntries); err != nil {
		return nil, err
	}
	bt := bundleType
	return &Bundle{Type: &bt, Entry: bundleEntries}, nil
}

// NewDocument builds a document Bundle. The first entry must be a Composition
// (bdl-3: a document's first resource is the Composition that organises it); a nil
// composition or a first resource of any other type is rejected. total is never set,
// search/request/response metadata never appears, and fullUrl values across the
// Composition and the remaining entries must be unique.
func NewDocument(composition fhir.Resource, entries ...DocumentEntry) (*Bundle, error) {
	if err := requireResourceType(composition, CompositionResourceType, "document"); err != nil {
		return nil, err
	}
	bundleEntries := make([]BundleEntry, 0, len(entries)+1)
	bundleEntries = append(bundleEntries, BundleEntry{Resource: resourcePtr(composition)})
	for i := range entries {
		bundleEntries = append(bundleEntries, BundleEntry{
			Resource: resourcePtr(entries[i].Resource),
			FullUrl:  fullURLPtr(entries[i].FullURL),
		})
	}
	if err := checkUniqueFullURLs(bundleEntries); err != nil {
		return nil, err
	}
	bundleType := BundleTypeDocument
	return &Bundle{Type: &bundleType, Entry: bundleEntries}, nil
}

// NewMessage builds a message Bundle. The first entry must be a MessageHeader (the
// message-bundle analogue of the document first-entry rule); a nil header or a first
// resource of any other type is rejected. total is never set, search/request/response
// metadata never appears, and fullUrl values must be unique.
func NewMessage(header fhir.Resource, entries ...MessageEntry) (*Bundle, error) {
	if err := requireResourceType(header, MessageHeaderResourceType, "message"); err != nil {
		return nil, err
	}
	bundleEntries := make([]BundleEntry, 0, len(entries)+1)
	bundleEntries = append(bundleEntries, BundleEntry{Resource: resourcePtr(header)})
	for i := range entries {
		bundleEntries = append(bundleEntries, BundleEntry{
			Resource: resourcePtr(entries[i].Resource),
			FullUrl:  fullURLPtr(entries[i].FullURL),
		})
	}
	if err := checkUniqueFullURLs(bundleEntries); err != nil {
		return nil, err
	}
	bundleType := BundleTypeMessage
	return &Bundle{Type: &bundleType, Entry: bundleEntries}, nil
}

// NewCollection builds a collection Bundle: an unconstrained set of resources with no
// per-type semantics beyond fullUrl uniqueness. total is never set (a collection is
// not a search result) and no request/response/search metadata appears.
func NewCollection(entries ...CollectionEntry) (*Bundle, error) {
	bundleEntries := make([]BundleEntry, 0, len(entries))
	for i := range entries {
		bundleEntries = append(bundleEntries, BundleEntry{
			Resource: resourcePtr(entries[i].Resource),
			FullUrl:  fullURLPtr(entries[i].FullURL),
		})
	}
	if err := checkUniqueFullURLs(bundleEntries); err != nil {
		return nil, err
	}
	bundleType := BundleTypeCollection
	return &Bundle{Type: &bundleType, Entry: bundleEntries}, nil
}

// requireResourceType checks that r is non-nil and reports the expected resourceType,
// the precondition the document/message builders share for their first entry. A nil
// resource (a nil interface or a typed-nil pointer) and a wrong type are both rejected
// naming the offending and expected types, never patient data.
func requireResourceType(r fhir.Resource, want, bundleKind string) error {
	got, ok := fhir.As[fhir.Resource](r)
	if !ok {
		return fmt.Errorf("%w: %s bundle requires a non-nil first %s", ErrInvalidBundle, bundleKind, want)
	}
	if got.ResourceType() != want {
		return fmt.Errorf("%w: %s bundle first entry must be a %s, got %s",
			ErrInvalidBundle, bundleKind, want, got.ResourceType())
	}
	return nil
}

// checkUniqueFullURLs reports the first duplicate fullUrl across the entries (bdl-7:
// a Bundle's fullUrl values must be unique). An entry with no fullUrl is skipped,
// because the uniqueness rule constrains only the values that are present.
func checkUniqueFullURLs(entries []BundleEntry) error {
	seen := make(map[string]struct{}, len(entries))
	for i := range entries {
		if entries[i].FullUrl == nil {
			continue
		}
		url := *entries[i].FullUrl
		if _, dup := seen[url]; dup {
			return fmt.Errorf("%w: entry %d repeats fullUrl %q", ErrInvalidBundle, i, url)
		}
		seen[url] = struct{}{}
	}
	return nil
}

// resourcePtr boxes a resource into the *fhir.Resource the generated BundleEntry
// carries, returning nil for a nil or typed-nil resource so an absent resource never
// becomes a non-nil pointer to a nil interface.
func resourcePtr(r fhir.Resource) *fhir.Resource {
	if _, ok := fhir.As[fhir.Resource](r); !ok {
		return nil
	}
	return &r
}

// fullURLPtr returns a pointer to url, or nil when url is empty, so an unset fullUrl
// is omitted on the wire rather than serialised as an empty string.
func fullURLPtr(url string) *string {
	if url == "" {
		return nil
	}
	return &url
}

// strPtr returns a pointer to s. It is the local pointer helper for the non-empty
// optional request headers the request builder has already checked.
func strPtr(s string) *string { return &s }
