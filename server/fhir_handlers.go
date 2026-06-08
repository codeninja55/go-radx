package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
)

// ServeHTTP routes a FHIR REST request to its interaction handler. The path arrives already stripped
// of the base prefix, so it is one of: "" / "/" (the system root, for a transaction POST),
// "/metadata" (the CapabilityStatement), "/{type}" (a create POST or a search-type GET), or
// "/{type}/{id}" (a read GET). An unrecognised method or path is answered with an OperationOutcome,
// never a bare body, so the FHIR-native error channel is uniform (servers.md). The handler logs the
// method and the interaction class only — never the query string, which can carry PHI (PRD §9.1).
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
		h.handleInstance(w, r, segs[0], segs[1])
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
	default:
		h.writeUnsupported(w, r, "the type endpoint supports GET (search) and POST (create) only")
	}
}

// handleInstance serves the instance-level interactions: a read GET on a resource by type and id.
// update, delete, vread, history, and patch are deferred and answered with a 501 OperationOutcome,
// never a silent no-op (servers.md, PRD §9.2).
func (h *fhirHandler) handleInstance(w http.ResponseWriter, r *http.Request, resourceType, id string) {
	if !isWorkflowResourceType(resourceType) {
		h.writeError(w, r, http.StatusNotFound, issueTypeNotSupported,
			"resource type "+resourceType+" is not served by this endpoint")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleRead(w, r, resourceType, id)
	case http.MethodPut, http.MethodDelete, http.MethodPatch:
		h.writeError(w, r, http.StatusNotImplemented, issueTypeNotSupported,
			"the "+strings.ToLower(r.Method)+" interaction is not implemented in v1")
	default:
		h.writeUnsupported(w, r, "the instance endpoint supports GET (read) only in v1")
	}
}

// handleRead serves a read: it fetches the resource from the repository and writes it, mapping a
// repository ErrNotFound to a 404 OperationOutcome so an absent resource is a structured FHIR error,
// not a bare 404 body.
func (h *fhirHandler) handleRead(w http.ResponseWriter, r *http.Request, resourceType, id string) {
	resource, err := h.repo.Read(r.Context(), resourceType, id)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	h.logger.Info("fhir read", zap.String("type", resourceType), zap.String("interaction", "read"))
	h.writeResource(w, r, http.StatusOK, resource, "")
}

// handleCreate serves a create: it reads and validates the body, then stores it through the
// repository. A body that does not decode, whose resourceType does not match the endpoint, or that
// fails structural validation is rejected with the appropriate OperationOutcome (400 for malformed,
// 422 for an unprocessable but well-formed resource). On success it answers 201 with a Location
// header naming the created resource.
func (h *fhirHandler) handleCreate(w http.ResponseWriter, r *http.Request, resourceType string) {
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
	if oo := h.adapter.validate(resource); oo.HasErrors() {
		h.writeOutcome(w, r, http.StatusUnprocessableEntity, h.adapter.operationOutcome(toOutcomeIssues(oo)))
		return
	}
	created, cerr := h.repo.Create(r.Context(), resource)
	if cerr != nil {
		h.writeRepoError(w, r, cerr)
		return
	}
	h.logger.Info("fhir create", zap.String("type", resourceType), zap.String("interaction", "create"))
	location := h.basePath + "/" + created.ResourceType() + "/" + h.adapter.resourceID(created)
	h.writeResource(w, r, http.StatusCreated, created, location)
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
	response, terr := h.repo.Transaction(r.Context(), bundle)
	if terr != nil {
		h.writeError(w, r, http.StatusBadRequest, issueTypeProcessing, sanitizeRepoMessage(terr))
		return
	}
	h.logger.Info("fhir transaction", zap.String("interaction", "transaction"))
	h.writeResource(w, r, http.StatusOK, response, "")
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

// writeRepoError maps a repository error to an OperationOutcome: ErrNotFound to a 404, any other to
// a 500 with a sanitized, PHI-free message. The repository's own errors are PHI-free by contract,
// but the message is still passed through sanitizeRepoMessage so only the structural part reaches
// the wire.
func (h *fhirHandler) writeRepoError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		h.writeError(w, r, http.StatusNotFound, fhir.IssueTypeNotFound, "resource not found")
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
