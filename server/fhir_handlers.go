package server

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/fhir"
)

// FHIR OperationOutcome issue-type codes the role emits that the root fhir package does not declare
// as constants (it declares only the subset its in-process Validate engine produces). The codes are
// the stable FHIR issue-type value-set codes, identical across R4 and R5, so they are named here as
// typed values rather than bare string literals at each call site.
const (
	issueTypeNotSupported fhir.IssueType = "not-supported"
	issueTypeProcessing   fhir.IssueType = "processing"
	issueTypeException    fhir.IssueType = "exception"
	// issueTypeSecurity is the FHIR issue-type code for an authentication/authorization failure, the
	// code a 401 OperationOutcome carries (the same code across R4 and R5).
	issueTypeSecurity fhir.IssueType = "security"
	// issueTypeConflict is the FHIR issue-type code for an edit-version conflict, the code a 412
	// Precondition Failed OperationOutcome carries when an If-Match version check fails (FHIR R5
	// http.html#concurrency).
	issueTypeConflict fhir.IssueType = "conflict"
	// issueTypeDeleted is the FHIR issue-type code for content that has been deleted, the code a 410
	// Gone OperationOutcome carries on a vread of a deleted version (FHIR R5 http.html#vread).
	issueTypeDeleted fhir.IssueType = "deleted"
	// issueTypeInformational is the FHIR issue-type code for a purely informational message, the
	// code a successful $validate's all-clear issue carries (OperationOutcome.issue is 1..*, so a
	// validation with no findings still reports one issue).
	issueTypeInformational fhir.IssueType = "informational"
)

// ServeHTTP routes a FHIR REST request to its interaction handler. The path arrives already stripped
// of the base prefix, so it is one of: "" / "/" (the system root, for a transaction POST),
// "/metadata" (the CapabilityStatement), "/{type}" (a create POST or a search-type GET),
// "/{type}/$validate" (the validate operation POST), "/{type}/{id}" (a read GET),
// "/{type}/{id}/_history" (history-instance GET), or "/{type}/{id}/_history/{vid}" (vread GET). An
// unrecognised method or path is answered with an OperationOutcome, never a bare body, so the
// FHIR-native error channel is uniform (servers.md). The handler logs the method and the
// interaction class only — never the query string, which can carry PHI (PRD §9.1).
func (h *fhirHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")

	if path == "metadata" {
		h.handleMetadata(w, r)
		return
	}
	if path == "" {
		h.handleSystem(w, r)
		return
	}

	segs := strings.Split(path, "/")
	switch len(segs) {
	case 1:
		h.handleType(w, r, segs[0])
	case 2:
		if segs[1] == "$validate" {
			h.handleValidate(w, r, segs[0])
			return
		}
		h.handleInstance(w, r, segs[0], segs[1])
	case 3, 4:
		if isWorkflowResourceType(segs[0]) && segs[2] == "_history" {
			if len(segs) == 4 {
				h.handleVRead(w, r, segs[0], segs[1], segs[3])
				return
			}
			h.handleHistoryInstance(w, r, segs[0], segs[1])
			return
		}
		h.writeUnsupported(w, r, "the requested interaction is not supported")
	default:
		h.writeUnsupported(w, r, "the requested interaction is not supported")
	}
}

// handleMetadata serves the CapabilityStatement (GET [base]/metadata), the conformance-negotiation
// endpoint. A non-GET is method-not-allowed.
func (h *fhirHandler) handleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeUnsupported(w, r, "metadata supports GET only")
		return
	}
	cs := h.adapter.capabilityStatement(h.basePath)
	h.writeResource(w, r, http.StatusOK, cs, "")
}

// handleSystem serves the system-level interactions at the base root: a transaction/batch POST. A
// non-POST at the root is unsupported.
func (h *fhirHandler) handleSystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeUnsupported(w, r, "the base endpoint supports a transaction POST only")
		return
	}
	h.handleTransaction(w, r)
}

// handleType serves the type-level interactions: a create POST or a search-type GET on a resource
// type. A type outside the served workflow set is rejected with a not-supported OperationOutcome,
// never silently.
func (h *fhirHandler) handleType(w http.ResponseWriter, r *http.Request, resourceType string) {
	if !isWorkflowResourceType(resourceType) {
		h.writeError(w, r, http.StatusNotFound, issueTypeNotSupported,
			"resource type "+resourceType+" is not served by this endpoint")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleSearch(w, r, resourceType)
	case http.MethodPost:
		h.handleCreate(w, r, resourceType)
	case http.MethodPut:
		h.handleConditionalUpdate(w, r, resourceType)
	case http.MethodPatch:
		h.handleConditionalPatch(w, r, resourceType)
	case http.MethodDelete:
		h.handleConditionalDelete(w, r, resourceType)
	default:
		h.writeUnsupported(w, r,
			"the type endpoint supports GET (search), POST (create), and conditional PUT/PATCH/DELETE only")
	}
}

// handleInstance serves the instance-level interactions on a resource by type and id: read (GET),
// update (PUT), patch (PATCH), and delete (DELETE). The If-Match version precondition, when present,
// is evaluated before any mutation so a version-aware client gets the honest 412 the spec defines
// for a stale version (FHIR R5 http.html#concurrency).
func (h *fhirHandler) handleInstance(w http.ResponseWriter, r *http.Request, resourceType, id string) {
	if !isWorkflowResourceType(resourceType) {
		h.writeError(w, r, http.StatusNotFound, issueTypeNotSupported,
			"resource type "+resourceType+" is not served by this endpoint")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleRead(w, r, resourceType, id)
	case http.MethodPut:
		h.handleUpdate(w, r, resourceType, id)
	case http.MethodPatch:
		h.handlePatch(w, r, resourceType, id)
	case http.MethodDelete:
		h.handleDelete(w, r, resourceType, id)
	default:
		h.writeUnsupported(w, r, "the instance endpoint supports GET, PUT, PATCH, and DELETE")
	}
}

// handleRead serves a read: it fetches the resource from the repository and writes it with the
// version headers (ETag W/"versionId" and Last-Modified, FHIR R5 http.html#read), mapping a
// repository ErrNotFound to a 404 OperationOutcome so an absent resource is a structured FHIR error,
// not a bare 404 body.
func (h *fhirHandler) handleRead(w http.ResponseWriter, r *http.Request, resourceType, id string) {
	resource, err := h.repo.Read(r.Context(), resourceType, id)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	h.setVersionHeaders(w, resource)
	h.logger.Info("fhir read", zap.String("type", resourceType), zap.String("interaction", "read"))
	h.writeResource(w, r, http.StatusOK, resource, "")
}

// handleVRead serves a vread (GET {type}/{id}/_history/{vid}, FHIR R5 http.html#vread): the named
// version is returned with its version headers; an unknown resource or version is a 404; a version
// that records a deletion is a 410 Gone with a deleted-coded OperationOutcome, the spec's
// distinct answer for "this version existed and was removed".
func (h *fhirHandler) handleVRead(w http.ResponseWriter, r *http.Request, resourceType, id, versionID string) {
	if r.Method != http.MethodGet {
		h.writeUnsupported(w, r, "the vread endpoint supports GET only")
		return
	}
	resource, err := h.repo.VRead(r.Context(), resourceType, id, versionID)
	if err != nil {
		if errors.Is(err, ErrGone) {
			h.writeError(w, r, http.StatusGone, issueTypeDeleted, "the requested version of the resource is deleted")
			return
		}
		h.writeRepoError(w, r, err)
		return
	}
	h.setVersionHeaders(w, resource)
	h.logger.Info("fhir vread", zap.String("type", resourceType), zap.String("interaction", "vread"))
	h.writeResource(w, r, http.StatusOK, resource, "")
}

// handleHistoryInstance serves an instance history (GET {type}/{id}/_history, FHIR R5
// http.html#history): the repository's version list, newest first, rendered as the release's
// history Bundle in which every entry carries the request that produced the version and the
// response facts (status, ETag, lastModified). A _count parameter caps the response at the newest
// _count entries while total still reports the full version count; paging links (Bundle.link
// next/prev) are deferred, so _count is a cap, not a page size. A resource that has never existed
// is a 404.
func (h *fhirHandler) handleHistoryInstance(w http.ResponseWriter, r *http.Request, resourceType, id string) {
	if r.Method != http.MethodGet {
		h.writeUnsupported(w, r, "the history endpoint supports GET only")
		return
	}
	versions, err := h.repo.History(r.Context(), resourceType, id)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	fullURL := h.absoluteResourceURL(r, resourceType, id)
	entries := h.historyEntries(fullURL, resourceType, id, versions)
	// Truncate AFTER deriving the entries: the create/update derivation keys off the oldest version
	// in the full list, so truncating the version list first would mis-render the oldest surviving
	// entry as the create. total stays the full version count.
	total := len(entries)
	entries = truncateHistoryEntries(entries, r.URL.Query().Get("_count"))
	bundle, err := h.adapter.newHistoryBundle(total, entries)
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, issueTypeException, "the server could not build the history bundle")
		return
	}
	h.logger.Info("fhir history", zap.String("type", resourceType), zap.String("interaction", "history-instance"))
	h.writeResource(w, r, http.StatusOK, bundle, "")
}

// historyEntries renders a resource's version list (newest first) into the release-neutral history
// Bundle entries. fullURL is the resource's absolute [base]/[type]/[id] URL: R5 bundle.html gives
// every version of a resource the same fullUrl (the version is distinguished by meta.versionId),
// so each entry carries it — including a deleted version's entry, which has no resource body to
// name itself. The interaction each version reports is derived from the version record: a
// deletion is a DELETE against the instance, the resource's first version is the create (a POST
// against the type, answered 201), and any later non-deleted version is an update (a PUT against
// the instance, answered 200) — the derivation the deferred update interaction (wave 3) slots into
// without reshaping the record.
func (h *fhirHandler) historyEntries(fullURL, resourceType, id string, versions []ResourceVersion) []historyEntry {
	entries := make([]historyEntry, 0, len(versions))
	for i, v := range versions {
		e := historyEntry{
			fullURL:      fullURL,
			etag:         weakETag(v.VersionID),
			lastModified: fhirInstant(v.LastUpdated),
		}
		switch {
		case v.Deleted:
			e.method, e.requestURL, e.status = http.MethodDelete, resourceType+"/"+id, "204 No Content"
		case i == len(versions)-1:
			// The oldest version (the list is newest first) is the create.
			e.method, e.requestURL, e.status = http.MethodPost, resourceType, "201 Created"
			e.resource = v.Resource
		default:
			e.method, e.requestURL, e.status = http.MethodPut, resourceType+"/"+id, "200 OK"
			e.resource = v.Resource
		}
		entries = append(entries, e)
	}
	return entries
}

// truncateHistoryEntries applies the history _count parameter: at most n of the newest entries are
// kept (the list is already newest first), per FHIR R5 http.html#history. A _count of 0 is an
// empty page with the honest total; an absent, non-numeric, or negative _count is ignored rather
// than rejected — _count is a hint, the same lenient stance MemoryRepository.Search documents for
// unrecognised parameters. Paging links over the truncated history are deferred.
func truncateHistoryEntries(entries []historyEntry, countParam string) []historyEntry {
	if countParam == "" {
		return entries
	}
	n, err := strconv.Atoi(countParam)
	if err != nil || n < 0 || n >= len(entries) {
		return entries
	}
	return entries[:n]
}

// handleCreate serves a create: it reads and validates the body, then stores it through the
// repository. A body that does not decode, whose resourceType does not match the endpoint, or that
// fails structural validation is rejected with the appropriate OperationOutcome (400 for malformed,
// 422 for an unprocessable but well-formed resource). On success it answers 201 with a Location
// header naming the created resource.
//
// A create carrying If-None-Exist (a conditional create, FHIR R5 http.html#cond-create) fails
// closed with a 400 OperationOutcome before anything is read or stored: the role has no search
// matching yet (the matching semantics are deferred to the search work), and silently ignoring the
// precondition would create the very duplicate the client asked the server to prevent — the
// fhir/rest client sends this header, so the role must answer it honestly, never drop it.
func (h *fhirHandler) handleCreate(w http.ResponseWriter, r *http.Request, resourceType string) {
	if r.Header.Get("If-None-Exist") != "" {
		h.writeError(w, r, http.StatusBadRequest, issueTypeNotSupported, conditionalCreateDiagnostics)
		return
	}
	if !h.requireFHIRWriteMedia(w, r) {
		return
	}
	body, err := h.readBody(r)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body could not be read or exceeds the size limit")
		return
	}
	resource, decErr := h.adapter.unmarshalResource(body)
	if decErr != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body is not a valid FHIR resource")
		return
	}
	if status, oo, ok := h.validateCreate(resource, resourceType); !ok {
		h.writeOutcome(w, r, status, oo)
		return
	}
	created, cerr := h.repo.Create(r.Context(), resource)
	if cerr != nil {
		h.writeRepoError(w, r, cerr)
		return
	}
	if h.audit != nil {
		// Structural identifiers only: the resource type plus the id and version the server minted
		// itself (Create always assigns the id), never a value from the resource body (PRD §9.5).
		versionID, _ := resourceVersionViaJSON(created)
		h.audit(AuditEvent{
			Op:           AuditOpFHIRCreate,
			Time:         time.Now().UTC(),
			Outcome:      AuditOutcomeStoredIndexed,
			ResourceType: created.ResourceType(),
			ResourceID:   h.adapter.resourceID(created),
			VersionID:    versionID,
		})
	}
	h.logger.Info("fhir create", zap.String("type", resourceType), zap.String("interaction", "create"))
	h.setVersionHeaders(w, created)
	h.writeResource(w, r, http.StatusCreated, created, h.createdLocation(created))
}

// createdLocation builds the Location header for a created resource: the versioned
// [base]/[type]/[id]/_history/[vid] form FHIR R5 http.html#create specifies for a server that
// maintains versions, which this role does. A resource with no version metadata (a custom
// unversioned Repository) falls back to the unversioned [base]/[type]/[id] rather than fabricating
// a version segment.
func (h *fhirHandler) createdLocation(created fhir.Resource) string {
	location := h.resourceLocation(created.ResourceType(), h.adapter.resourceID(created))
	if versionID, _ := resourceVersionViaJSON(created); versionID != "" {
		location += "/_history/" + versionID
	}
	return location
}

// setVersionHeaders emits the version headers a read, vread, or create response carries per FHIR R5
// http.html#read / #create: ETag as the weak form W/"versionId" and Last-Modified as the HTTP date
// of meta.lastUpdated. A resource with no version metadata (a custom unversioned Repository) emits
// neither header rather than fabricating a version.
func (h *fhirHandler) setVersionHeaders(w http.ResponseWriter, resource fhir.Resource) {
	versionID, lastUpdated := resourceVersionViaJSON(resource)
	if versionID != "" {
		w.Header().Set("ETag", weakETag(versionID))
	}
	if lastUpdated != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastUpdated); err == nil {
			w.Header().Set("Last-Modified", t.UTC().Format(http.TimeFormat))
		}
	}
}

// weakETag renders a versionId as the weak entity tag FHIR mandates (http.html#concurrency: ETags
// are weak because a resource may have semantically equal but byte-different renderings).
func weakETag(versionID string) string { return `W/"` + versionID + `"` }

// etagVersionID extracts the versionId from an If-Match entity tag, accepting the weak form
// (W/"1"), the strong form ("1") a non-FHIR-aware client may send, and a bare version id. An
// unparseable header yields the raw string, which then simply fails the version comparison — a
// malformed precondition is a failed precondition, never a bypassed one.
func etagVersionID(etag string) string {
	v := strings.TrimSpace(etag)
	v = strings.TrimPrefix(v, "W/")
	return strings.Trim(v, `"`)
}

// resourceLocation builds the Location header for a created resource by joining the role's base path
// with the resource's type and id under exactly one slash each. path.Join is used rather than string
// concatenation so a root-mounted role (basePath "/") yields "/Patient/1" rather than "//Patient/1":
// the latter parses as a network-path reference (host "Patient") and is not a valid relative
// Location. A "/fhir"-mounted role yields "/fhir/Patient/1" unchanged. path.Join drops a leading
// slash, so it is re-prefixed to keep the Location an absolute-path reference.
func (h *fhirHandler) resourceLocation(resourceType, id string) string {
	return "/" + path.Join(strings.Trim(h.basePath, "/"), resourceType, id)
}

// conditionalCreateDiagnostics is the PHI-free diagnostic both write paths answer a conditional
// create with: it names the unsupported precondition, never the search query the header carries
// (which a client could populate with an identifier or name, PRD §9.1).
const conditionalCreateDiagnostics = "conditional create (If-None-Exist) is not supported; " +
	"retry as an unconditional create"

// absoluteResourceURL builds a resource's absolute [base]/[type]/[id] URL from the request the role
// is answering: the scheme follows the connection (https when the daemon terminated TLS), the host
// is the one the client addressed (r.Host), and the path is the role's resourceLocation. It is the
// fullUrl a history Bundle entry carries — a Bundle travels beyond the requesting connection, so a
// relative reference would dangle; bundle.html wants the absolute form.
func (h *fhirHandler) absoluteResourceURL(r *http.Request, resourceType, id string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + h.resourceLocation(resourceType, id)
}

// validateCreate is the one create-validation gate both write paths share: the type-level create POST
// and every POST entry of a transaction. It enforces, in order, that the target type is a served
// workflow type, that the resource's own resourceType matches that target, and that the resource
// passes the release structural validator. targetType is the create endpoint's {type} (handleCreate)
// or a transaction POST entry's request.url (the transaction pre-commit pass), so a transaction entry
// cannot bypass a check the direct create enforces. On rejection it returns the HTTP status (400 for a
// malformed or out-of-scope request, 422 for a well-formed but unprocessable resource), the release
// OperationOutcome to write, and false; on acceptance it returns 0, nil, true. The returned outcome is
// PHI-free: it names a resource type, a code, or a structural rule, never a patient value (PRD §9.1).
func (h *fhirHandler) validateCreate(resource fhir.Resource, targetType string) (int, fhir.Resource, bool) {
	if !isWorkflowResourceType(targetType) {
		return http.StatusBadRequest, h.singleIssueOutcome(issueTypeNotSupported,
			"resource type "+targetType+" is not served by this endpoint"), false
	}
	if resource.ResourceType() != targetType {
		return http.StatusBadRequest, h.singleIssueOutcome(fhir.IssueTypeInvalid,
			"resource type "+resource.ResourceType()+" does not match the "+targetType+" endpoint"), false
	}
	if oo := h.adapter.validate(resource); oo.HasErrors() {
		return http.StatusUnprocessableEntity, h.adapter.operationOutcome(toOutcomeIssues(oo)), false
	}
	return 0, nil, true
}

// singleIssueOutcome builds a single-error-issue release OperationOutcome, the shared body for an
// error the role detects itself (an out-of-scope type, a type mismatch). It is the resource form of
// writeError's body, used where a caller needs the OperationOutcome value rather than writing it
// directly. The diagnostic is PHI-free (PRD §9.1).
func (h *fhirHandler) singleIssueOutcome(code fhir.IssueType, diagnostics string) fhir.Resource {
	return h.adapter.operationOutcome([]outcomeIssue{{
		Severity:    fhir.SeverityError,
		Code:        code,
		Diagnostics: diagnostics,
	}})
}

// handleValidate serves the type-level $validate operation (POST {type}/$validate, FHIR R5
// operation-resource-validate): the body is decoded and run through the same release validator that
// gates create, and the findings are returned as an OperationOutcome — nothing is persisted. A
// validation that executed answers 200 whatever it found (the operation succeeded; the findings are
// its result); a validation with no findings still carries one informational issue because
// OperationOutcome.issue is 1..*. A body that does not decode, or whose resourceType does not match
// the endpoint, is a 400 — the operation could not be performed at all.
func (h *fhirHandler) handleValidate(w http.ResponseWriter, r *http.Request, resourceType string) {
	if !isWorkflowResourceType(resourceType) {
		h.writeError(w, r, http.StatusNotFound, issueTypeNotSupported,
			"resource type "+resourceType+" is not served by this endpoint")
		return
	}
	if r.Method != http.MethodPost {
		h.writeUnsupported(w, r, "the $validate operation supports POST only")
		return
	}
	if !h.requireFHIRWriteMedia(w, r) {
		return
	}
	body, err := h.readBody(r)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body could not be read or exceeds the size limit")
		return
	}
	resource, decErr := h.adapter.unmarshalResource(body)
	if decErr != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body is not a valid FHIR resource")
		return
	}
	if resource.ResourceType() != resourceType {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid,
			"resource type "+resource.ResourceType()+" does not match the "+resourceType+" endpoint")
		return
	}
	issues := toOutcomeIssues(h.adapter.validate(resource))
	if len(issues) == 0 {
		issues = []outcomeIssue{{
			Severity:    fhir.SeverityInformation,
			Code:        issueTypeInformational,
			Diagnostics: "validation succeeded: no issues found",
		}}
	}
	h.logger.Info("fhir validate", zap.String("type", resourceType), zap.String("interaction", "$validate"))
	h.writeOutcome(w, r, http.StatusOK, h.adapter.operationOutcome(issues))
}

// handleSearch serves a type-level search: it forwards the raw query parameters to the repository
// and writes the searchset Bundle the repository builds.
func (h *fhirHandler) handleSearch(w http.ResponseWriter, r *http.Request, resourceType string) {
	bundle, err := h.repo.Search(r.Context(), resourceType, r.URL.Query())
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	h.logger.Info("fhir search", zap.String("type", resourceType), zap.String("interaction", "search-type"))
	h.writeResource(w, r, http.StatusOK, bundle, "")
}

// handleTransaction serves a transaction: it reads and decodes the request Bundle, then applies it
// through the repository and writes the transaction-response Bundle. A body that does not decode or
// is not a Bundle is a 400 OperationOutcome; a repository failure (an unsupported entry verb, a
// missing referenced resource) is mapped to an OperationOutcome rather than a silent partial success.
func (h *fhirHandler) handleTransaction(w http.ResponseWriter, r *http.Request) {
	if !h.requireFHIRWriteMedia(w, r) {
		return
	}
	body, err := h.readBody(r)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body could not be read or exceeds the size limit")
		return
	}
	bundle, decErr := h.adapter.unmarshalResource(body)
	if decErr != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body is not a valid FHIR resource")
		return
	}
	if bundle.ResourceType() != "Bundle" {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid,
			"the base endpoint requires a Bundle, got "+bundle.ResourceType())
		return
	}
	// Only a transaction Bundle is processed at the system endpoint: the repository applies it
	// atomically and the role advertises only the transaction interaction. A Bundle of any other type
	// (collection, searchset, document, batch, ...) is rejected before the repository is touched, so an
	// empty collection or searchset Bundle is never silently run as a transaction.
	if bt := bundleType(bundle); bt != bundleTypeTransaction {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid,
			"the base endpoint processes a transaction Bundle only, got Bundle.type "+bt)
		return
	}
	// Validate every create the transaction would perform through the same gate handleCreate uses,
	// before the repository commits anything. A transaction is atomic, so the whole transaction must be
	// rejected if any entry fails validation; a single invalid entry must not commit alongside its valid
	// siblings. Without this pass a transaction POST would reach the repository directly and bypass the
	// whitelist, url/type, and structural-validation checks a direct create enforces.
	if status, oo, ok := h.validateTransactionWrites(bundle); !ok {
		h.writeOutcome(w, r, status, oo)
		return
	}
	response, terr := h.repo.Transaction(r.Context(), bundle)
	if terr != nil {
		h.writeError(w, r, http.StatusBadRequest, issueTypeProcessing, sanitizeRepoMessage(terr))
		return
	}
	h.logger.Info("fhir transaction", zap.String("interaction", "transaction"))
	h.writeResource(w, r, http.StatusOK, response, "")
}

// validateTransactionWrites validates every create a transaction Bundle would perform through the
// shared validateCreate gate, before the repository commits anything. It is the transaction half of
// the single create-validation path: each POST entry's resource is checked against its request.url
// target type exactly as a direct create POST is, so the two write paths cannot diverge. Because the
// transaction is atomic, the first failing entry rejects the whole transaction (nothing commits) — the
// pass stops and returns that entry's status and OperationOutcome. On full success it returns 0, nil,
// true and the repository then applies the now-validated entries. The returned outcome is PHI-free.
func (h *fhirHandler) validateTransactionWrites(bundle fhir.Resource) (int, fhir.Resource, bool) {
	writes, err := h.adapter.transactionPostEntries(bundle)
	if err != nil {
		return http.StatusBadRequest, h.singleIssueOutcome(fhir.IssueTypeInvalid, sanitizeRepoMessage(err)), false
	}
	for _, wr := range writes {
		// A POST entry carrying request.ifNoneExist is a conditional create; it fails closed exactly
		// like the direct create's If-None-Exist header (see handleCreate) so the transaction path
		// cannot silently create the duplicate the precondition was meant to prevent.
		if wr.ifNoneExist != "" {
			return http.StatusBadRequest, h.singleIssueOutcome(issueTypeNotSupported, conditionalCreateDiagnostics), false
		}
		// A transaction CREATE must target the type endpoint ("Patient"), never an instance URL
		// ("Patient/123"). splitTypeID surfaces the id segment when the request.url carries one; a
		// non-empty id is a malformed create that would otherwise be silently created against the type,
		// ignoring the id, so it is rejected (400) before the repository commits anything.
		if wr.targetID != "" {
			return http.StatusBadRequest, h.singleIssueOutcome(fhir.IssueTypeInvalid,
				"transaction POST entry url "+wr.targetType+"/"+wr.targetID+" targets an instance; a create must target the "+wr.targetType+" type endpoint"), false
		}
		if status, oo, ok := h.validateCreate(wr.resource, wr.targetType); !ok {
			return status, oo, false
		}
	}
	return 0, nil, true
}

// writeResource marshals a resource and writes it with the FHIR JSON content type, optionally with a
// Location header (set on a create). A marshal failure is itself reported through an
// OperationOutcome rather than a half-written body.
func (h *fhirHandler) writeResource(w http.ResponseWriter, r *http.Request, status int, resource fhir.Resource, location string) {
	body, err := json.Marshal(resource)
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, issueTypeException, "the server could not encode the response")
		return
	}
	w.Header().Set("Content-Type", mediaTypeFHIRJSON)
	if location != "" {
		w.Header().Set("Location", location)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeError builds a single-issue release OperationOutcome and writes it with the given status, the
// FHIR-native error channel. The diagnostic names a resource type, a code, or a structural rule —
// never a patient value (PRD §9.1).
func (h *fhirHandler) writeError(w http.ResponseWriter, r *http.Request, status int, code fhir.IssueType, diagnostics string) {
	oo := h.adapter.operationOutcome([]outcomeIssue{{
		Severity:    fhir.SeverityError,
		Code:        code,
		Diagnostics: diagnostics,
	}})
	h.writeOutcome(w, r, status, oo)
}

// writeOutcome writes an already-built release OperationOutcome with the given status. A marshal
// failure falls back to a bare status code rather than panicking.
func (h *fhirHandler) writeOutcome(w http.ResponseWriter, _ *http.Request, status int, oo fhir.Resource) {
	body, err := json.Marshal(oo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", mediaTypeFHIRJSON)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeUnsupported writes a 405 OperationOutcome for an unsupported method, the deferred-interaction
// channel (servers.md: never a silent no-op).
func (h *fhirHandler) writeUnsupported(w http.ResponseWriter, r *http.Request, diagnostics string) {
	h.writeError(w, r, http.StatusMethodNotAllowed, issueTypeNotSupported, diagnostics)
}

// writeUnauthorized writes a 401 release OperationOutcome with the FHIR JSON content type, the
// rejection body the auth middleware uses for the FHIR role so a 401 stays on the role's FHIR-native
// error channel rather than net/http's plain-text body. It is the unauthorizedResponder the role hands
// authMiddleware; the diagnostic names the failure class only, never a credential or a request value
// (PRD §9.1).
func (h *fhirHandler) writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, r, http.StatusUnauthorized, issueTypeSecurity, "the request is not authenticated")
}

// requireFHIRWriteMedia enforces the FHIR write content type before a write body is read or decoded.
// A write (create, transaction) carries an application/fhir+json body; FHIR also permits the generic
// application/json. An unsupported or absent Content-Type is a 415 OperationOutcome written before the
// body is touched, so a body sent as text/plain or with no Content-Type never mutates the repository.
// It reports true to proceed, false when it has already written the 415 response.
func (h *fhirHandler) requireFHIRWriteMedia(w http.ResponseWriter, r *http.Request) bool {
	if isFHIRWriteMediaType(r.Header.Get("Content-Type")) {
		return true
	}
	h.writeError(w, r, http.StatusUnsupportedMediaType, issueTypeNotSupported,
		"a FHIR write requires Content-Type "+mediaTypeFHIRJSON+" (application/json is also accepted)")
	return false
}

// isFHIRWriteMediaType reports whether a Content-Type header names a media type a FHIR write accepts:
// application/fhir+json (the FHIR JSON media type) or the generic application/json FHIR also permits, a
// charset parameter allowed on either. The header is parsed with mime.ParseMediaType so a parameter or
// casing does not defeat the check; an absent or unparseable Content-Type is not accepted, matching the
// server contract that a write declares its FHIR JSON body.
func isFHIRWriteMediaType(contentType string) bool {
	return mediaTypeEquals(contentType, mediaTypeFHIRJSON) || mediaTypeEquals(contentType, mediaTypeJSON)
}

// mediaTypeEquals reports whether a Content-Type header names the given media type, ignoring a
// charset (or other) parameter and casing. An absent or unparseable Content-Type never matches, so a
// write that does not declare its body type is rejected rather than guessed. It is the shared
// content-type comparison the write-media and patch-media checks use.
func mediaTypeEquals(contentType, want string) bool {
	if contentType == "" {
		return false
	}
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return strings.EqualFold(mt, want)
}

// writeRepoError maps a repository error to an OperationOutcome: ErrNotFound to a 404, ErrGone to a
// 410 (a read of a deleted resource, FHIR R5 http.html#delete), any other to a 500 with a sanitized,
// PHI-free message. The repository's own errors are PHI-free by contract, but the message is still
// passed through sanitizeRepoMessage so only the structural part reaches the wire.
func (h *fhirHandler) writeRepoError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		h.writeError(w, r, http.StatusNotFound, fhir.IssueTypeNotFound, "resource not found")
		return
	}
	if errors.Is(err, ErrGone) {
		h.writeError(w, r, http.StatusGone, issueTypeDeleted, "the resource is deleted")
		return
	}
	if errors.Is(err, ErrVersionConflict) {
		h.writeError(w, r, http.StatusPreconditionFailed, issueTypeConflict,
			"the If-Match version does not match the current version of the resource")
		return
	}
	h.writeError(w, r, http.StatusInternalServerError, issueTypeException, sanitizeRepoMessage(err))
}

// toOutcomeIssues reduces a release-agnostic validation outcome to the role's outcomeIssue list, so
// the release adapter can render it into a release OperationOutcome. Only the error- and
// warning-severity issues are carried (an information issue is not an error response), all PHI-free.
func toOutcomeIssues(oo *fhir.OperationOutcome) []outcomeIssue {
	if oo == nil {
		return nil
	}
	out := make([]outcomeIssue, 0, len(oo.Issue))
	for i := range oo.Issue {
		iss := &oo.Issue[i]
		out = append(out, outcomeIssue{
			Severity:    iss.Severity,
			Code:        iss.Code,
			Diagnostics: iss.Diagnostics,
			Expression:  iss.Expression,
		})
	}
	return out
}

// bundleTypeTransaction is the one Bundle.type the system endpoint processes. A request Bundle of any
// other type is rejected before the repository is touched, matching the advertised transaction-only
// system interaction. The value is the stable FHIR Bundle-type code, identical across R4 and R5.
const bundleTypeTransaction = "transaction"

// bundleType reads a Bundle's "type" by marshalling it and peeking the top-level "type" key, the same
// release-neutral JSON approach the adapters use for a resource id. A Bundle always serialises its
// type under "type", so this reads an R4 or R5 Bundle's type without a per-release switch. A Bundle
// with no type yields "".
func bundleType(bundle fhir.Resource) string {
	data, err := json.Marshal(bundle)
	if err != nil {
		return ""
	}
	var env struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return ""
	}
	return env.Type
}

// isWorkflowResourceType reports whether resourceType is in the served workflow set, so a request
// for an out-of-scope type is rejected rather than routed to the repository.
func isWorkflowResourceType(resourceType string) bool {
	for _, rt := range workflowResourceTypes {
		if rt == resourceType {
			return true
		}
	}
	return false
}

// sanitizeRepoMessage returns the part of a repository error message that is safe to surface: the
// repository's errors are built from resource types, ids, and rule names, never patient values, so
// the message is passed through unchanged. The function is the single point a future PHI-bearing
// repository error would be scrubbed, keeping the no-PHI contract auditable in one place.
func sanitizeRepoMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
