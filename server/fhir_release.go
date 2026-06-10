package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/codeninja55/go-radx/fhir"
	"github.com/codeninja55/go-radx/fhir/r4"
	"github.com/codeninja55/go-radx/fhir/r5"
)

// This file is the one place the FHIR server role binds to a concrete FHIR release. A Bundle, a
// CapabilityStatement, and an OperationOutcome are release-specific Go types (r4.Bundle vs
// r5.Bundle), so the release-fixed role needs a release-neutral way to build them. The
// releaseAdapter interface names exactly the release-specific operations the role and the in-memory
// repository need; the r4Adapter and r5Adapter implement them over their concrete types, so the rest
// of the role stays release-agnostic and the release split lives here.

// releaseAdapter performs the release-specific construction the FHIR role and the in-memory
// repository need: decoding a request body into the release's concrete type, validating it,
// assigning and reading a resource id, and building the release's searchset Bundle,
// transaction-response Bundle, OperationOutcome, and CapabilityStatement. One adapter is selected at
// role construction from the configured release.
type releaseAdapter interface {
	// unmarshalResource decodes a request body into the release's concrete resource type, dispatched
	// by its resourceType. An unknown or absent resourceType is an error.
	unmarshalResource(data []byte) (fhir.Resource, error)

	// validate runs the release validator over a resource, returning the release-agnostic outcome.
	validate(r fhir.Resource) *fhir.OperationOutcome

	// resourceID returns a resource's logical id, or "" when it has none.
	resourceID(r fhir.Resource) string

	// withResourceVersion returns the resource with its logical id and version metadata
	// (meta.versionId, meta.lastUpdated) set, so a create can assign the server id and stamp the
	// stored version in one pass. It re-decodes through the release registry, returning a fresh
	// concrete resource of the release rather than mutating r in place.
	withResourceVersion(r fhir.Resource, id, versionID, lastUpdated string) (fhir.Resource, error)

	// newSearchSet builds a searchset Bundle of the release carrying the matched resources, with
	// total set to the match count. It returns the Bundle behind the fhir.Resource interface.
	newSearchSet(total int32, matches []fhir.Resource) (fhir.Resource, error)

	// newHistoryBundle builds a history Bundle of the release from a resource's version list (newest
	// first), with total set and each entry carrying the request/response pair FHIR R5 http.html
	// requires of a history entry (bdl-3 makes entry.request mandatory in a history bundle).
	newHistoryBundle(entries []historyEntry) (fhir.Resource, error)

	// processTransaction applies a transaction Bundle through repo and builds the
	// transaction-response Bundle of the release. It is on the adapter because both decoding the
	// request bundle and building the response bundle are release-specific.
	processTransaction(ctx context.Context, bundle fhir.Resource, repo Repository) (fhir.Resource, error)

	// transactionPostEntries returns one transactionPostEntry per POST entry of a transaction Bundle:
	// the resource the entry would create and the target resource type its request.url names. The role
	// validates each through the shared create gate before the repository commits, so a transaction
	// create cannot bypass the validation a direct create enforces. Reading the entries is
	// release-specific (the entry shape differs between r4 and r5), so it lives on the adapter; an entry
	// of a verb other than POST is skipped (the repository handles a GET entry, which performs no
	// create). A POST entry missing its resource is an error so a malformed transaction is rejected
	// before commit rather than at apply time.
	transactionPostEntries(bundle fhir.Resource) ([]transactionPostEntry, error)

	// operationOutcome builds a release OperationOutcome resource from a set of issues, for the error
	// response body. The issues are PHI-free structural locators.
	operationOutcome(issues []outcomeIssue) fhir.Resource

	// capabilityStatement builds the served CapabilityStatement advertising the role's supported
	// interactions over the workflow resource set.
	capabilityStatement(basePath string) fhir.Resource
}

// transactionPostEntry is one create a transaction Bundle would perform, extracted by the adapter so
// the role can validate it through the shared create gate before the repository commits. resource is
// the entry's resource (a release concrete type behind fhir.Resource) and targetType is the resource
// type the entry's request.url names, the transaction analogue of the create endpoint's {type}.
// targetID is the id segment of that request.url when one is present; a create must target the type
// endpoint ("Patient"), so a POST entry whose url carries an id ("Patient/123") is malformed and is
// rejected before commit.
type transactionPostEntry struct {
	resource   fhir.Resource
	targetType string
	targetID   string
}

// historyEntry is one release-neutral entry of a history Bundle the role hands the adapter to
// render: the version's resource (nil for a deleted version), the interaction that produced it
// (request method/url), and the response facts (status line, weak ETag, lastModified instant) FHIR
// R5 http.html#history says a history entry reports. Every field is structural — ids, version ids,
// status lines — never a patient value (PRD §9.1).
type historyEntry struct {
	resource     fhir.Resource
	method       string
	requestURL   string
	status       string
	etag         string
	lastModified string
}

// outcomeIssue is the release-neutral issue the role hands an adapter to render into a release
// OperationOutcome. Every field is a severity, a code, a structural locator, or a rule name — never
// a patient value (PRD §9.1).
type outcomeIssue struct {
	Severity    fhir.IssueSeverity
	Code        fhir.IssueType
	Diagnostics string
	Expression  string
}

// adapterForRelease returns the adapter for a release and whether the release is supported, so a
// role or repository wired to an unsupported release fails closed at construction.
func adapterForRelease(release fhir.Release) (releaseAdapter, bool) {
	switch release {
	case fhir.R4:
		return r4Adapter{}, true
	case fhir.R5:
		return r5Adapter{}, true
	default:
		return nil, false
	}
}

// workflowResourceTypes is the conformance-subset resource set the FHIR role serves and advertises
// (servers.md "FHIR REST server"). The role serves read/create/search/transaction over these; a
// request for another type is answered with an OperationOutcome, never silently.
var workflowResourceTypes = []string{
	"Patient",
	"Encounter",
	"ServiceRequest",
	"ImagingStudy",
	"DiagnosticReport",
	"Observation",
}

// idEnvelope reads or writes only a resource's "id" through JSON, so resource id get/set works over
// any release's concrete type without reflection or a per-type switch. A resource always serialises
// its id under the top-level "id" key, so this round-trips faithfully.
type idEnvelope struct {
	ID string `json:"id"`
}

// resourceIDViaJSON reads a resource's id by marshalling it and peeking the "id" key. It is the
// shared, release-neutral id getter the adapters reuse.
func resourceIDViaJSON(r fhir.Resource) string {
	data, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	var env idEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return ""
	}
	return env.ID
}

// withResourceVersionViaJSON sets a resource's id and version metadata by marshalling it to a JSON
// object, splicing in the "id" key and the "meta" versionId/lastUpdated keys (merging with any meta
// the resource already carries), and re-decoding through decode (the release registry's
// UnmarshalResource), so version assignment works over any release's concrete type without
// reflection. It returns a fresh resource of the release with the version set; the original is left
// untouched, matching the build-once, immutable-after discipline.
func withResourceVersionViaJSON(r fhir.Resource, id, versionID, lastUpdated string, decode func([]byte) (fhir.Resource, error)) (fhir.Resource, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("server: encode resource to assign version: %w", err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("server: decode resource to assign version: %w", err)
	}
	idBytes, err := json.Marshal(id)
	if err != nil {
		return nil, err
	}
	obj["id"] = idBytes

	meta := map[string]json.RawMessage{}
	if raw, ok := obj["meta"]; ok {
		if err := json.Unmarshal(raw, &meta); err != nil {
			return nil, fmt.Errorf("server: decode resource meta to assign version: %w", err)
		}
	}
	vidBytes, err := json.Marshal(versionID)
	if err != nil {
		return nil, err
	}
	luBytes, err := json.Marshal(lastUpdated)
	if err != nil {
		return nil, err
	}
	meta["versionId"] = vidBytes
	meta["lastUpdated"] = luBytes
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	obj["meta"] = metaBytes

	merged, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return decode(merged)
}

// resourceVersionViaJSON reads a resource's meta.versionId and meta.lastUpdated by marshalling it
// and peeking the "meta" key, the version twin of resourceIDViaJSON. A resource with no meta (or an
// unversioned one from a custom Repository) yields empty strings, which the handlers treat as
// "emit no version headers" rather than an error.
func resourceVersionViaJSON(r fhir.Resource) (versionID, lastUpdated string) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", ""
	}
	var env struct {
		Meta struct {
			VersionID   string `json:"versionId"`
			LastUpdated string `json:"lastUpdated"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return "", ""
	}
	return env.Meta.VersionID, env.Meta.LastUpdated
}

// ---- R5 adapter ----

type r5Adapter struct{}

func (r5Adapter) unmarshalResource(data []byte) (fhir.Resource, error) {
	return r5.UnmarshalResource(data)
}

func (r5Adapter) validate(r fhir.Resource) *fhir.OperationOutcome { return r5.Validate(r) }

func (r5Adapter) resourceID(r fhir.Resource) string { return resourceIDViaJSON(r) }

func (r5Adapter) withResourceVersion(r fhir.Resource, id, versionID, lastUpdated string) (fhir.Resource, error) {
	return withResourceVersionViaJSON(r, id, versionID, lastUpdated, r5.UnmarshalResource)
}

func (r5Adapter) newSearchSet(total int32, matches []fhir.Resource) (fhir.Resource, error) {
	entries := make([]r5.SearchEntry, 0, len(matches))
	mode := r5.SearchEntryModeMatch
	for _, m := range matches {
		entries = append(entries, r5.SearchEntry{Resource: m, Mode: &mode})
	}
	return r5.NewSearchSet(total, entries...)
}

// newHistoryBundle builds the R5 history Bundle directly (no typed builder exists for history):
// type "history", total set (bdl-1 permits total on searchset and history), and every entry
// carrying the request that produced the version and the response facts (status, weak ETag,
// lastModified), per FHIR R5 http.html#history. A deleted version's entry has no resource.
func (r5Adapter) newHistoryBundle(entries []historyEntry) (fhir.Resource, error) {
	bt := r5.BundleTypeHistory
	total := int32(len(entries)) // #nosec G115 -- an in-memory version count is far below int32
	bundle := &r5.Bundle{Type: &bt, Total: &total}
	for _, e := range entries {
		verb := r5.HTTPVerb(e.method)
		entry := r5.BundleEntry{
			Request:  &r5.BundleEntryRequest{Method: &verb, URL: strptr(e.requestURL)},
			Response: &r5.BundleEntryResponse{Status: strptr(e.status), Etag: optptr(e.etag), LastModified: optptr(e.lastModified)},
		}
		if e.resource != nil {
			res := e.resource
			entry.Resource = &res
		}
		bundle.Entry = append(bundle.Entry, entry)
	}
	return bundle, nil
}

func (a r5Adapter) processTransaction(ctx context.Context, bundle fhir.Resource, repo Repository) (fhir.Resource, error) {
	b, ok := bundle.(*r5.Bundle)
	if !ok {
		return nil, fmt.Errorf("server: transaction bundle is a %s, not an R5 Bundle", bundle.ResourceType())
	}
	responses := make([]r5.BundleEntry, 0, len(b.Entry))
	for i := range b.Entry {
		entry := &b.Entry[i]
		status, location, err := applyTransactionEntryR5(ctx, entry, repo)
		if err != nil {
			return nil, err
		}
		responses = append(responses, r5.BundleEntry{
			Response: &r5.BundleEntryResponse{Status: strptr(status), Location: optptr(location)},
		})
	}
	bt := r5.BundleTypeTransactionResponse
	return &r5.Bundle{Type: &bt, Entry: responses}, nil
}

func (a r5Adapter) transactionPostEntries(bundle fhir.Resource) ([]transactionPostEntry, error) {
	b, ok := bundle.(*r5.Bundle)
	if !ok {
		return nil, fmt.Errorf("server: transaction bundle is a %s, not an R5 Bundle", bundle.ResourceType())
	}
	var writes []transactionPostEntry
	for i := range b.Entry {
		entry := &b.Entry[i]
		if entry.Request == nil || entry.Request.Method == nil || *entry.Request.Method != r5.HTTPVerbPOST {
			continue
		}
		if entry.Resource == nil {
			return nil, fmt.Errorf("%w: POST entry missing resource", errUnsupportedTxnVerb)
		}
		targetType, targetID := splitTypeID(deref(entry.Request.URL))
		writes = append(writes, transactionPostEntry{resource: *entry.Resource, targetType: targetType, targetID: targetID})
	}
	return writes, nil
}

func (r5Adapter) operationOutcome(issues []outcomeIssue) fhir.Resource {
	oo := &r5.OperationOutcome{}
	for _, iss := range issues {
		sev := r5.IssueSeverity(iss.Severity)
		code := r5.IssueType(iss.Code)
		issue := r5.OperationOutcomeIssue{Severity: &sev, Code: &code}
		if iss.Diagnostics != "" {
			issue.Diagnostics = strptr(iss.Diagnostics)
		}
		if iss.Expression != "" {
			issue.Expression = []string{iss.Expression}
		}
		oo.Issue = append(oo.Issue, issue)
	}
	return oo
}

func (a r5Adapter) capabilityStatement(basePath string) fhir.Resource {
	status := r5.PublicationStatusActive
	kind := r5.CapabilityStatementKindInstance
	version := r5.FHIRVersionN500
	mode := r5.RestfulCapabilityModeServer
	jsonFmt := mediaTypeFHIRJSON

	cs := &r5.CapabilityStatement{
		StatusElement: nil,
		Status:        &status,
		Date:          strptr(capabilityStatementDate),
		Kind:          &kind,
		FhirVersion:   &version,
		Format:        []string{jsonFmt},
	}
	cs.Software = &r5.CapabilityStatementSoftware{Name: strptr(capabilitySoftwareName)}

	rest := r5.CapabilityStatementRest{Mode: &mode}
	// The role advertises only the system interaction it actually implements: the base POST does
	// transaction processing. It does not return a batch-response (a batch Bundle is processed as a
	// transaction), and GET at the base is a 405, so neither batch nor search-system is advertised —
	// the served metadata matches the handler exactly rather than over-advertising.
	txn := r5.SystemRestfulInteractionTransaction
	rest.Interaction = []r5.CapabilityStatementRestInteraction{{Code: &txn}}
	for _, rt := range workflowResourceTypes {
		typ, err := r5.ParseResourceType(rt)
		if err != nil {
			continue
		}
		read := r5.TypeRestfulInteractionRead
		vread := r5.TypeRestfulInteractionVread
		historyInstance := r5.TypeRestfulInteractionHistoryInstance
		create := r5.TypeRestfulInteractionCreate
		searchType := r5.TypeRestfulInteractionSearchType
		rest.Resource = append(rest.Resource, r5.CapabilityStatementRestResource{
			Type: &typ,
			Interaction: []r5.CapabilityStatementRestResourceInteraction{
				{Code: &read}, {Code: &vread}, {Code: &historyInstance}, {Code: &create}, {Code: &searchType},
			},
			Operation: []r5.CapabilityStatementRestResourceOperation{
				{Name: strptr(validateOperationName), Definition: strptr(validateOperationDefinition)},
			},
		})
	}
	cs.Rest = []r5.CapabilityStatementRest{rest}
	return cs
}

// applyTransactionEntryR5 applies one transaction entry through the repository and returns the
// response status line and Location. A POST creates the entry's resource; a GET reads the resource
// the entry's request.url names. An unsupported verb fails the transaction (errUnsupportedTxnVerb),
// never a silent skip.
func applyTransactionEntryR5(ctx context.Context, entry *r5.BundleEntry, repo Repository) (status, location string, err error) {
	if entry.Request == nil || entry.Request.Method == nil {
		return "", "", fmt.Errorf("%w: entry missing request.method", errUnsupportedTxnVerb)
	}
	switch *entry.Request.Method {
	case r5.HTTPVerbPOST:
		if entry.Resource == nil {
			return "", "", fmt.Errorf("%w: POST entry missing resource", errUnsupportedTxnVerb)
		}
		created, cerr := repo.Create(ctx, *entry.Resource)
		if cerr != nil {
			return "", "", cerr
		}
		return "201 Created", created.ResourceType() + "/" + resourceIDViaJSON(created), nil
	case r5.HTTPVerbGET:
		rt, id := splitTypeID(deref(entry.Request.URL))
		if _, rerr := repo.Read(ctx, rt, id); rerr != nil {
			return "", "", rerr
		}
		return "200 OK", "", nil
	default:
		return "", "", fmt.Errorf("%w: %s", errUnsupportedTxnVerb, *entry.Request.Method)
	}
}

// ---- R4 adapter ----

type r4Adapter struct{}

func (r4Adapter) unmarshalResource(data []byte) (fhir.Resource, error) {
	return r4.UnmarshalResource(data)
}

func (r4Adapter) validate(r fhir.Resource) *fhir.OperationOutcome { return r4.Validate(r) }

func (r4Adapter) resourceID(r fhir.Resource) string { return resourceIDViaJSON(r) }

func (r4Adapter) withResourceVersion(r fhir.Resource, id, versionID, lastUpdated string) (fhir.Resource, error) {
	return withResourceVersionViaJSON(r, id, versionID, lastUpdated, r4.UnmarshalResource)
}

func (r4Adapter) newSearchSet(total int32, matches []fhir.Resource) (fhir.Resource, error) {
	entries := make([]r4.SearchEntry, 0, len(matches))
	mode := r4.SearchEntryModeMatch
	for _, m := range matches {
		entries = append(entries, r4.SearchEntry{Resource: m, Mode: &mode})
	}
	return r4.NewSearchSet(total, entries...)
}

// newHistoryBundle builds the R4 history Bundle, the R4 twin of the R5 adapter's (see that method
// for the bdl-1/http.html#history rationale).
func (r4Adapter) newHistoryBundle(entries []historyEntry) (fhir.Resource, error) {
	bt := r4.BundleTypeHistory
	total := int32(len(entries)) // #nosec G115 -- an in-memory version count is far below int32
	bundle := &r4.Bundle{Type: &bt, Total: &total}
	for _, e := range entries {
		verb := r4.HTTPVerb(e.method)
		entry := r4.BundleEntry{
			Request:  &r4.BundleEntryRequest{Method: &verb, URL: strptr(e.requestURL)},
			Response: &r4.BundleEntryResponse{Status: strptr(e.status), Etag: optptr(e.etag), LastModified: optptr(e.lastModified)},
		}
		if e.resource != nil {
			res := e.resource
			entry.Resource = &res
		}
		bundle.Entry = append(bundle.Entry, entry)
	}
	return bundle, nil
}

func (a r4Adapter) processTransaction(ctx context.Context, bundle fhir.Resource, repo Repository) (fhir.Resource, error) {
	b, ok := bundle.(*r4.Bundle)
	if !ok {
		return nil, fmt.Errorf("server: transaction bundle is a %s, not an R4 Bundle", bundle.ResourceType())
	}
	responses := make([]r4.BundleEntry, 0, len(b.Entry))
	for i := range b.Entry {
		entry := &b.Entry[i]
		status, location, err := applyTransactionEntryR4(ctx, entry, repo)
		if err != nil {
			return nil, err
		}
		responses = append(responses, r4.BundleEntry{
			Response: &r4.BundleEntryResponse{Status: strptr(status), Location: optptr(location)},
		})
	}
	bt := r4.BundleTypeTransactionResponse
	return &r4.Bundle{Type: &bt, Entry: responses}, nil
}

func (a r4Adapter) transactionPostEntries(bundle fhir.Resource) ([]transactionPostEntry, error) {
	b, ok := bundle.(*r4.Bundle)
	if !ok {
		return nil, fmt.Errorf("server: transaction bundle is a %s, not an R4 Bundle", bundle.ResourceType())
	}
	var writes []transactionPostEntry
	for i := range b.Entry {
		entry := &b.Entry[i]
		if entry.Request == nil || entry.Request.Method == nil || *entry.Request.Method != r4.HTTPVerbPOST {
			continue
		}
		if entry.Resource == nil {
			return nil, fmt.Errorf("%w: POST entry missing resource", errUnsupportedTxnVerb)
		}
		targetType, targetID := splitTypeID(deref(entry.Request.URL))
		writes = append(writes, transactionPostEntry{resource: *entry.Resource, targetType: targetType, targetID: targetID})
	}
	return writes, nil
}

func (r4Adapter) operationOutcome(issues []outcomeIssue) fhir.Resource {
	oo := &r4.OperationOutcome{}
	for _, iss := range issues {
		sev := r4.IssueSeverity(iss.Severity)
		code := r4.IssueType(iss.Code)
		issue := r4.OperationOutcomeIssue{Severity: &sev, Code: &code}
		if iss.Diagnostics != "" {
			issue.Diagnostics = strptr(iss.Diagnostics)
		}
		if iss.Expression != "" {
			issue.Expression = []string{iss.Expression}
		}
		oo.Issue = append(oo.Issue, issue)
	}
	return oo
}

func (a r4Adapter) capabilityStatement(basePath string) fhir.Resource {
	status := r4.PublicationStatusActive
	kind := r4.CapabilityStatementKindInstance
	version := r4.FHIRVersionN401
	mode := r4.RestfulCapabilityModeServer
	jsonFmt := mediaTypeFHIRJSON

	cs := &r4.CapabilityStatement{
		Status:      &status,
		Date:        strptr(capabilityStatementDate),
		Kind:        &kind,
		FhirVersion: &version,
		Format:      []string{jsonFmt},
	}
	cs.Software = &r4.CapabilityStatementSoftware{Name: strptr(capabilitySoftwareName)}

	rest := r4.CapabilityStatementRest{Mode: &mode}
	// The role advertises only the system interaction it actually implements: the base POST does
	// transaction processing. It does not return a batch-response (a batch Bundle is processed as a
	// transaction), and GET at the base is a 405, so neither batch nor search-system is advertised —
	// the served metadata matches the handler exactly rather than over-advertising.
	txn := r4.SystemRestfulInteractionTransaction
	rest.Interaction = []r4.CapabilityStatementRestInteraction{{Code: &txn}}
	for _, rt := range workflowResourceTypes {
		typ, err := r4.ParseResourceType(rt)
		if err != nil {
			continue
		}
		read := r4.TypeRestfulInteractionRead
		vread := r4.TypeRestfulInteractionVread
		historyInstance := r4.TypeRestfulInteractionHistoryInstance
		create := r4.TypeRestfulInteractionCreate
		searchType := r4.TypeRestfulInteractionSearchType
		rest.Resource = append(rest.Resource, r4.CapabilityStatementRestResource{
			Type: &typ,
			Interaction: []r4.CapabilityStatementRestResourceInteraction{
				{Code: &read}, {Code: &vread}, {Code: &historyInstance}, {Code: &create}, {Code: &searchType},
			},
			Operation: []r4.CapabilityStatementRestResourceOperation{
				{Name: strptr(validateOperationName), Definition: strptr(validateOperationDefinition)},
			},
		})
	}
	cs.Rest = []r4.CapabilityStatementRest{rest}
	return cs
}

// applyTransactionEntryR4 applies one R4 transaction entry, the R4 twin of applyTransactionEntryR5.
func applyTransactionEntryR4(ctx context.Context, entry *r4.BundleEntry, repo Repository) (status, location string, err error) {
	if entry.Request == nil || entry.Request.Method == nil {
		return "", "", fmt.Errorf("%w: entry missing request.method", errUnsupportedTxnVerb)
	}
	switch *entry.Request.Method {
	case r4.HTTPVerbPOST:
		if entry.Resource == nil {
			return "", "", fmt.Errorf("%w: POST entry missing resource", errUnsupportedTxnVerb)
		}
		created, cerr := repo.Create(ctx, *entry.Resource)
		if cerr != nil {
			return "", "", cerr
		}
		return "201 Created", created.ResourceType() + "/" + resourceIDViaJSON(created), nil
	case r4.HTTPVerbGET:
		rt, id := splitTypeID(deref(entry.Request.URL))
		if _, rerr := repo.Read(ctx, rt, id); rerr != nil {
			return "", "", rerr
		}
		return "200 OK", "", nil
	default:
		return "", "", fmt.Errorf("%w: %s", errUnsupportedTxnVerb, *entry.Request.Method)
	}
}

// capabilitySoftwareName is the software name the served CapabilityStatement advertises. It names the
// library, not a deployment, and carries no PHI.
const capabilitySoftwareName = "go-radx"

// capabilityStatementDate is the publication date the served CapabilityStatement carries.
// CapabilityStatement.date is a required (1..1) element, so it must be present for the statement to
// validate. It is a fixed value rather than the current time so the served metadata is deterministic
// and golden-testable (the role serves the same statement every run); it advances when the served
// capability surface changes.
const capabilityStatementDate = "2026-06-11"

// validateOperationName and validateOperationDefinition advertise the server-side $validate
// operation in the CapabilityStatement; the definition is the canonical HL7 OperationDefinition for
// Resource/$validate, the same canonical across R4 and R5.
const (
	validateOperationName       = "validate"
	validateOperationDefinition = "http://hl7.org/fhir/OperationDefinition/Resource-validate"
)

// strptr returns a pointer to s, the local helper for the non-empty optional string fields the
// release Bundle and OperationOutcome builders take.
func strptr(s string) *string { return &s }

// optptr returns a pointer to s, or nil when s is empty, so an absent optional field is omitted on
// the wire rather than serialised as an empty string.
func optptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// deref returns the string a pointer points at, or "" when it is nil, so a nil request.url reads as
// an empty string rather than panicking.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// splitTypeID splits a "ResourceType/id" reference (a transaction entry's GET request.url) into its
// type and id. A url with no slash yields the whole string as the type and an empty id, which the
// repository read then reports as not-found rather than panicking.
func splitTypeID(ref string) (resourceType, id string) {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '/' {
			return ref[:i], ref[i+1:]
		}
	}
	return ref, ""
}
