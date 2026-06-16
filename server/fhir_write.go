package server

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/fhir"
)

// This file holds the FHIR server-role write interactions beyond create: update (PUT), patch
// (PATCH), and delete (DELETE), with their conditional forms. The normative anchors are the FHIR R5
// HTTP page (hl7.org/fhir/R5/http.html): #update, #patch, #delete, #cond-update, #cond-patch,
// #cond-delete, and #concurrency for the If-Match precondition. The role matches HAPI's default
// status codes (update-as-create allowed; idempotent delete) and never leaks PHI in an error
// (PRD §9.1): every diagnostic names a resource type, an id, a version, or a structural rule.

// mediaTypeJSONPatch is the Content-Type a JSON Patch (RFC 6902) document carries on a PATCH
// request. It is the one patch format the role supports; a PATCH with any other content type is a
// 415. FHIRPath Patch (a Parameters body) is out of scope (documented in cli-server.md).
const mediaTypeJSONPatch = "application/json-patch+json"

// handleUpdate serves an update (PUT [type]/[id], FHIR R5 http.html#update): a full-resource
// replace. The body is read and decoded, its resourceType and id must match the URL (a mismatch is a
// 400 — the resource must address the endpoint it is PUT to), and the resource is validated through
// the same release validator that gates create. The If-Match precondition, when present, is checked
// against the current version (412 on a stale version, FHIR R5 http.html#concurrency). On success the
// repository bumps the version: an existing resource answers 200, an update that created the resource
// (update-as-create, the FHIR default and HAPI's default) answers 201 with a versioned Location.
func (h *fhirHandler) handleUpdate(w http.ResponseWriter, r *http.Request, resourceType, id string) {
	if !h.requireFHIRWriteMedia(w, r) {
		return
	}
	resource, ok := h.decodeWriteBody(w, r)
	if !ok {
		return
	}
	if !h.checkUpdateIntegrity(w, r, resource, resourceType, id) {
		return
	}
	if oo := h.adapter.validate(resource); oo.HasErrors() {
		h.writeOutcome(w, r, http.StatusUnprocessableEntity, h.adapter.operationOutcome(toOutcomeIssues(oo)))
		return
	}
	if !h.checkIfMatch(w, r, resourceType, id) {
		return
	}
	h.applyUpdate(w, r, resourceType, id, resource, "update")
}

// applyUpdate calls the repository's update and writes the response: 201 with a versioned Location
// when the update created the resource, 200 otherwise. interaction names the originating interaction
// for the log and the audit event ("update" or "conditional-update"); a create-on-update is audited
// as a create, the rest as an update.
func (h *fhirHandler) applyUpdate(w http.ResponseWriter, r *http.Request, resourceType, id string, resource fhir.Resource, interaction string) {
	updated, created, err := h.repo.Update(r.Context(), resourceType, id, resource)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	h.auditWrite(updated, created)
	h.logger.Info("fhir "+interaction, zap.String("type", resourceType), zap.String("interaction", interaction))
	h.setVersionHeaders(w, updated)
	if created {
		h.writeResource(w, r, http.StatusCreated, updated, h.createdLocation(updated))
		return
	}
	h.writeResource(w, r, http.StatusOK, updated, "")
}

// handlePatch serves a patch (PATCH [type]/[id], FHIR R5 http.html#patch): a partial modification.
// The role supports JSON Patch (RFC 6902, Content-Type application/json-patch+json); any other patch
// format is a 415. The patch is applied to the current version's JSON, the result is decoded into the
// release type and validated through the same release validator that gates create, the If-Match
// precondition (when present) is checked, and the result is stored as the next version (answering
// 200). A patch of an absent or deleted resource is a 404/410 (there is nothing to patch); a patch
// that does not apply (a failed test op, a bad path) is a 422.
func (h *fhirHandler) handlePatch(w http.ResponseWriter, r *http.Request, resourceType, id string) {
	if !isJSONPatchMediaType(r.Header.Get("Content-Type")) {
		h.writeError(w, r, http.StatusUnsupportedMediaType, issueTypeNotSupported,
			"patch requires Content-Type "+mediaTypeJSONPatch+" (JSON Patch, RFC 6902); FHIRPath Patch is not supported")
		return
	}
	patchDoc, err := h.readBody(r)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body could not be read or exceeds the size limit")
		return
	}
	if !h.checkIfMatch(w, r, resourceType, id) {
		return
	}
	current, err := h.repo.Read(r.Context(), resourceType, id)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	patched, status, diag := h.applyJSONPatch(current, patchDoc)
	if patched == nil {
		h.writeError(w, r, status, fhir.IssueTypeInvalid, diag)
		return
	}
	if oo := h.adapter.validate(patched); oo.HasErrors() {
		h.writeOutcome(w, r, http.StatusUnprocessableEntity, h.adapter.operationOutcome(toOutcomeIssues(oo)))
		return
	}
	updated, _, uerr := h.repo.Update(r.Context(), resourceType, id, patched)
	if uerr != nil {
		h.writeRepoError(w, r, uerr)
		return
	}
	h.auditWrite(updated, false)
	h.logger.Info("fhir patch", zap.String("type", resourceType), zap.String("interaction", "patch"))
	h.setVersionHeaders(w, updated)
	h.writeResource(w, r, http.StatusOK, updated, "")
}

// handleDelete serves a delete (DELETE [type]/[id], FHIR R5 http.html#delete). The If-Match
// precondition (when present) is checked against the current version. The repository retires the
// current version; the delete is idempotent — deleting an absent or already-deleted resource is a
// success (HAPI answers 200/204, never 404). A resource that existed answers 200 with no body; an
// absent resource answers 204 No Content. After a delete a read answers 410 Gone, prior versions
// stay vread-able, and the history shows the deletion.
func (h *fhirHandler) handleDelete(w http.ResponseWriter, r *http.Request, resourceType, id string) {
	if !h.checkIfMatch(w, r, resourceType, id) {
		return
	}
	existed, err := h.repo.Delete(r.Context(), resourceType, id)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	if existed {
		h.auditDelete(resourceType, id)
	}
	h.logger.Info("fhir delete", zap.String("type", resourceType), zap.String("interaction", "delete"))
	if existed {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// checkUpdateIntegrity enforces the update body's resourceType/id integrity: the body's resourceType
// must match the URL type and its id, when present, must match the URL id (FHIR R5 http.html#update:
// "the id ... in the resource SHALL be the same as the [id] in the URL"). A body with no id is
// accepted — the server adopts the URL id — but a body whose id contradicts the URL is a 400, never a
// silent overwrite of the wrong resource. It reports true to proceed, false when it wrote the error.
func (h *fhirHandler) checkUpdateIntegrity(w http.ResponseWriter, r *http.Request, resource fhir.Resource, resourceType, id string) bool {
	if resource.ResourceType() != resourceType {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid,
			"resource type "+resource.ResourceType()+" does not match the "+resourceType+" endpoint")
		return false
	}
	if bodyID := h.adapter.resourceID(resource); bodyID != "" && bodyID != id {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid,
			"resource id "+bodyID+" does not match the URL id "+id)
		return false
	}
	return true
}

// decodeWriteBody reads and decodes a write body into the release's concrete resource, writing the
// 400 OperationOutcome and reporting false on a read or decode failure. It is the shared body-decode
// step the update path uses, mirroring handleCreate's read/decode prelude.
func (h *fhirHandler) decodeWriteBody(w http.ResponseWriter, r *http.Request) (fhir.Resource, bool) {
	body, err := h.readBody(r)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body could not be read or exceeds the size limit")
		return nil, false
	}
	resource, decErr := h.adapter.unmarshalResource(body)
	if decErr != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeStructure, "request body is not a valid FHIR resource")
		return nil, false
	}
	return resource, true
}

// checkIfMatch evaluates a write's If-Match version precondition against the current version of the
// target resource, per FHIR R5 http.html#concurrency: a version-aware write names the version it was
// based on, and "if the version id given in the If-Match header does not match, the server returns a
// 412 Precondition Failed status code instead of updating the resource". A request with no If-Match
// passes through. An If-Match against a resource that does not exist (or is deleted) cannot be
// satisfied, so it is a 412 — the named version is not the current one. A stale If-Match is the 412
// with a conflict OperationOutcome. It reports true to proceed, false when it wrote the error.
func (h *fhirHandler) checkIfMatch(w http.ResponseWriter, r *http.Request, resourceType, id string) bool {
	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		return true
	}
	current, err := h.repo.Read(r.Context(), resourceType, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrGone) {
			h.writeError(w, r, http.StatusPreconditionFailed, issueTypeConflict,
				"the If-Match version cannot be satisfied: "+resourceType+"/"+id+" has no matching current version")
			return false
		}
		h.writeRepoError(w, r, err)
		return false
	}
	versionID, _ := resourceVersionViaJSON(current)
	if etagVersionID(ifMatch) != versionID {
		h.writeError(w, r, http.StatusPreconditionFailed, issueTypeConflict,
			"the If-Match version does not match the current version of "+resourceType+"/"+id)
		return false
	}
	return true
}

// auditWrite emits the structural audit event for a committed update or patch: the resource type
// plus the server-minted id and version, never a body value (PRD §9.5). A create-on-update is
// audited as a create (AuditOpFHIRCreate); an update of an existing resource as an update. Patch
// always passes created=false, so a patch is audited as an update.
func (h *fhirHandler) auditWrite(updated fhir.Resource, created bool) {
	if h.audit == nil {
		return
	}
	versionID, _ := resourceVersionViaJSON(updated)
	op := AuditOpFHIRUpdate
	if created {
		op = AuditOpFHIRCreate
	}
	h.audit(AuditEvent{
		Op:           op,
		Time:         time.Now().UTC(),
		Outcome:      AuditOutcomeStoredIndexed,
		ResourceType: updated.ResourceType(),
		ResourceID:   h.adapter.resourceID(updated),
		VersionID:    versionID,
	})
}

// auditDelete emits the structural audit event for a committed delete: the resource type and id,
// never a body value (PRD §9.5). The version is left empty — the deletion version carries no
// resource body to mint a version from on the role side.
func (h *fhirHandler) auditDelete(resourceType, id string) {
	if h.audit == nil {
		return
	}
	h.audit(AuditEvent{
		Op:           AuditOpFHIRDelete,
		Time:         time.Now().UTC(),
		Outcome:      AuditOutcomeDeleted,
		ResourceType: resourceType,
		ResourceID:   id,
	})
}

// isJSONPatchMediaType reports whether a Content-Type names the JSON Patch media type the role
// accepts on a PATCH. It is parsed with the same media-type parser the write-media check uses so a
// charset parameter or casing does not defeat the check.
func isJSONPatchMediaType(contentType string) bool {
	return mediaTypeEquals(contentType, mediaTypeJSONPatch)
}

// ---- conditional interactions ----

// conditionalScopeDiagnostics is the PHI-free diagnostic the conditional write paths answer with
// when the search criteria are absent: a conditional write must carry a search query (FHIR R5
// http.html#cond-update). It names the missing precondition, never a search value.
const conditionalScopeDiagnostics = "a conditional interaction requires search parameters in the query string"

// handleConditionalUpdate serves a conditional update (PUT [type]?[search], FHIR R5
// http.html#cond-update): the search criteria resolve to zero, one, or many matches. Zero matches is
// a create (the body's id, if any, is honoured as the new resource's id is server-assigned on a
// create-on-no-match; here the role lets the repository mint one via create), one match is an update
// of that resource, and many matches is a 412 (the criteria are not selective enough — the spec's
// "multiple matches" rule). The body is validated through the create gate first; PHI in the criteria
// never reaches an error (PRD §9.1).
func (h *fhirHandler) handleConditionalUpdate(w http.ResponseWriter, r *http.Request, resourceType string) {
	if r.URL.RawQuery == "" {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid, conditionalScopeDiagnostics)
		return
	}
	if !h.requireFHIRWriteMedia(w, r) {
		return
	}
	resource, ok := h.decodeWriteBody(w, r)
	if !ok {
		return
	}
	if resource.ResourceType() != resourceType {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid,
			"resource type "+resource.ResourceType()+" does not match the "+resourceType+" endpoint")
		return
	}
	if oo := h.adapter.validate(resource); oo.HasErrors() {
		h.writeOutcome(w, r, http.StatusUnprocessableEntity, h.adapter.operationOutcome(toOutcomeIssues(oo)))
		return
	}
	match, count, ok := h.resolveConditional(w, r, resourceType)
	if !ok {
		return
	}
	switch count {
	case 0:
		// No match: create. The repository mints the id, matching the unconditional create.
		h.createResolved(w, r, resourceType, resource)
	case 1:
		h.applyUpdate(w, r, resourceType, match, resource, "conditional-update")
	default:
		h.writeError(w, r, http.StatusPreconditionFailed, issueTypeConflict,
			"the conditional update criteria matched multiple resources; refine the search to a single match")
	}
}

// handleConditionalPatch serves a conditional patch (PATCH [type]?[search], FHIR R5
// http.html#cond-patch): the criteria resolve to one match, which is patched. Zero matches is a 404
// (nothing to patch); multiple matches is a 412 (not selective enough). The patch itself is the same
// JSON Patch the instance patch applies.
func (h *fhirHandler) handleConditionalPatch(w http.ResponseWriter, r *http.Request, resourceType string) {
	if r.URL.RawQuery == "" {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid, conditionalScopeDiagnostics)
		return
	}
	if !isJSONPatchMediaType(r.Header.Get("Content-Type")) {
		h.writeError(w, r, http.StatusUnsupportedMediaType, issueTypeNotSupported,
			"patch requires Content-Type "+mediaTypeJSONPatch+" (JSON Patch, RFC 6902); FHIRPath Patch is not supported")
		return
	}
	match, count, ok := h.resolveConditional(w, r, resourceType)
	if !ok {
		return
	}
	switch count {
	case 0:
		h.writeError(w, r, http.StatusNotFound, fhir.IssueTypeNotFound, "the conditional patch criteria matched no resource")
	case 1:
		h.handlePatch(w, r, resourceType, match)
	default:
		h.writeError(w, r, http.StatusPreconditionFailed, issueTypeConflict,
			"the conditional patch criteria matched multiple resources; refine the search to a single match")
	}
}

// handleConditionalDelete serves a conditional delete (DELETE [type]?[search], FHIR R5
// http.html#cond-delete): the criteria resolve the resource(s) to delete. Zero matches is a no-op
// success (204, idempotent — the same stance the unconditional delete takes); one match is deleted
// (200). Multiple-match deletion is not supported (the role deletes a single resolved resource), so
// many matches is a 412 asking the client to refine — the conservative HAPI stance when
// multiple-delete is disabled, never a bulk delete the client did not unambiguously request.
func (h *fhirHandler) handleConditionalDelete(w http.ResponseWriter, r *http.Request, resourceType string) {
	if r.URL.RawQuery == "" {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid, conditionalScopeDiagnostics)
		return
	}
	match, count, ok := h.resolveConditional(w, r, resourceType)
	if !ok {
		return
	}
	switch count {
	case 0:
		h.logger.Info("fhir conditional-delete", zap.String("type", resourceType), zap.String("interaction", "conditional-delete"))
		w.WriteHeader(http.StatusNoContent)
	case 1:
		h.handleDelete(w, r, resourceType, match)
	default:
		h.writeError(w, r, http.StatusPreconditionFailed, issueTypeConflict,
			"the conditional delete criteria matched multiple resources; refine the search to a single match")
	}
}

// createResolved performs the create half of a conditional update's no-match case: it stores the
// resource through the repository's create (the server mints the id) and answers 201 with a
// versioned Location, the same response the unconditional create gives. It is split out so the
// conditional-update switch reads as create/update/conflict.
func (h *fhirHandler) createResolved(w http.ResponseWriter, r *http.Request, resourceType string, resource fhir.Resource) {
	created, err := h.repo.Create(r.Context(), resource)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	h.auditWrite(created, true)
	h.logger.Info("fhir conditional-update", zap.String("type", resourceType), zap.String("interaction", "conditional-update-create"))
	h.setVersionHeaders(w, created)
	h.writeResource(w, r, http.StatusCreated, created, h.createdLocation(created))
}

// resolveConditional resolves a conditional interaction's search criteria to a single id and a match
// count, reusing the repository's search machinery. It runs the search the same way handleSearch
// does, then counts the matches in the returned searchset Bundle: zero, one (the resolved id), or
// many. It reports (id, count, true) to proceed; on a search error it writes the OperationOutcome and
// reports ok=false. The resolved id is a structural locator (a server-assigned id), never a search
// value — the criteria themselves are never echoed into an error (PRD §9.1).
//
// Scope: resolution is exactly as selective as the configured Repository's search. The development
// MemoryRepository matches the _id parameter and otherwise returns all-of-type, so against it a
// conditional interaction is well-defined for an _id criterion (and for an identifier criterion once
// the Repository supports it); a production Repository with full search resolves any criteria. This
// honest scope is documented in the parity matrix and cli-server.md rather than hidden behind a
// criteria parser the dev repository cannot honour.
func (h *fhirHandler) resolveConditional(w http.ResponseWriter, r *http.Request, resourceType string) (id string, count int, ok bool) {
	bundle, err := h.repo.Search(r.Context(), resourceType, conditionalSearchParams(r.URL.Query()))
	if err != nil {
		h.writeRepoError(w, r, err)
		return "", 0, false
	}
	ids := searchsetMatchIDs(bundle, h.adapter)
	if len(ids) == 1 {
		return ids[0], 1, true
	}
	return "", len(ids), true
}

// conditionalSearchParams strips the FHIR result-shaping parameters (_count, _sort, _include, ...)
// from a conditional interaction's query so the resolution counts MATCHES, not a paged view: a
// conditional update/delete's "how many match" question must not be capped by a _count the client
// happened to send. Only the search-criteria parameters are forwarded to the repository.
func conditionalSearchParams(q url.Values) url.Values {
	out := url.Values{}
	for k, vs := range q {
		switch k {
		case "_count", "_sort", "_include", "_revinclude", "_summary", "_elements", "_format":
			continue
		}
		out[k] = vs
	}
	return out
}
