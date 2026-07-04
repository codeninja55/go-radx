package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/codeninja55/go-radx/fhir"
)

// This file holds the FHIR server-role batch processing at the base endpoint (POST [base] with a
// batch Bundle, FHIR R5 http.html#brules). Batch entries execute independently — a failing entry
// never rolls back or aborts its siblings, the defining contrast with the transaction path's
// atomicity — and the response is a batch-response Bundle with one response entry per request entry.
//
// Every entry is executed by dispatching a synthesized sub-request through the role's own ServeHTTP,
// so a batch entry flows through exactly the same per-interaction pipeline as a standalone request:
// release validation on writes, the conditional-header semantics (If-None-Exist fail-closed,
// If-Match compare-and-swap), the workflow-type whitelist, the media-type gates, and the version
// store. There is no second write path a batch entry could use to bypass a check the direct path
// enforces. The spec guarantees no processing order for a batch; this role processes entries
// sequentially in request order so the outcome is deterministic and the response entries correspond
// one-to-one with the request entries.
//
// Per the spec's batch rules, entries must be independent: this role performs no intra-bundle
// reference resolution (a urn:uuid fullUrl is never rewritten into a created id) and no conditional
// reference resolution — both are transaction-only processing steps in http.html#brules.

// batchEntry is one release-neutral entry of a batch Bundle the adapter extracts for the role to
// execute: the request line (method and base-relative url), the entry's resource (nil when it
// carries none), and the conditional-header fields of entry.request, carried verbatim so the
// sub-request presents exactly the headers a standalone request would. An entry with no request at
// all extracts with an empty method and url, which the executor rejects per-entry (bdl-3a) rather
// than failing the whole batch.
type batchEntry struct {
	method          string
	url             string
	resource        fhir.Resource
	ifNoneExist     string
	ifMatch         string
	ifNoneMatch     string
	ifModifiedSince string
}

// batchEntryResponse is one release-neutral response entry the role hands the adapter to render
// into the batch-response Bundle: the HTTP status line, the Location, weak ETag, and lastModified of
// a write, the resource a successful read/search/vread returns (carried in the entry's resource, not
// its response, per FHIR R5 bundle.html), and the OperationOutcome a failed entry carries in
// response.outcome (nil on success). Every structural field — status line, server-minted location,
// version tag — carries no patient value (PRD §9.1); the resource is the one the client would have
// received from the standalone read it stands in for.
type batchEntryResponse struct {
	status       string
	location     string
	etag         string
	lastModified string
	resource     fhir.Resource
	outcome      fhir.Resource
}

// handleBatch serves a batch Bundle already decoded and type-checked by handleSystem: each entry is
// executed independently through the standalone pipeline and the per-entry results are returned as
// a batch-response Bundle. The batch itself answers 200 whatever the individual entries did — a
// per-entry failure lives in that entry's response.status and response.outcome, never in the outer
// HTTP status (FHIR R5 http.html#brules).
func (h *fhirHandler) handleBatch(w http.ResponseWriter, r *http.Request, bundle fhir.Resource) {
	entries, err := h.adapter.batchEntries(bundle)
	if err != nil {
		h.writeError(w, r, http.StatusBadRequest, fhir.IssueTypeInvalid, sanitizeRepoMessage(err))
		return
	}
	if len(entries) > maxBundleEntries {
		h.writeError(w, r, http.StatusRequestEntityTooLarge, issueTypeProcessing, tooManyBundleEntriesDiagnostics)
		return
	}
	responses := make([]batchEntryResponse, 0, len(entries))
	for _, entry := range entries {
		// A batch runs its entries sequentially; a client that disconnected (or a shutdown that
		// cancelled the request context) must stop the remaining writes rather than drive unbounded
		// work no one will read. The check is between entries so an in-flight entry completes cleanly.
		if err := r.Context().Err(); err != nil {
			h.writeError(w, r, http.StatusInternalServerError, issueTypeProcessing,
				"the batch was cancelled before all entries were processed")
			return
		}
		responses = append(responses, h.executeBatchEntry(r, entry))
	}
	response, berr := h.adapter.newBatchResponse(responses)
	if berr != nil {
		h.writeError(w, r, http.StatusInternalServerError, issueTypeException,
			"the server could not build the batch response bundle")
		return
	}
	h.logger.Info("fhir batch", zap.String("interaction", "batch"), zap.Int("entries", len(entries)))
	h.writeResource(w, r, http.StatusOK, response, "")
}

// executeBatchEntry runs one batch entry through the standalone pipeline and returns its response
// entry. A malformed entry (no request line, a base-endpoint target, an absolute url) is answered
// with a per-entry 400 outcome before any dispatch; everything else is dispatched through ServeHTTP
// so the standalone routing decides — including the per-entry 405 for an unsupported method and the
// 404 for an out-of-scope type — and the recorded response is folded into the entry.
func (h *fhirHandler) executeBatchEntry(outer *http.Request, entry batchEntry) batchEntryResponse {
	if entry.method == "" || entry.url == "" {
		return h.batchEntryError(http.StatusBadRequest, fhir.IssueTypeRequired,
			"a batch entry must carry request.method and request.url (bdl-3a)")
	}
	target, err := url.Parse(entry.url)
	if err != nil {
		return h.batchEntryError(http.StatusBadRequest, fhir.IssueTypeInvalid,
			"a batch entry request.url could not be parsed")
	}
	// Entry urls are relative to the server base (http.html#brules). An absolute url names another
	// server, which this role must not silently reinterpret as local; it fails closed per-entry.
	if target.Scheme != "" || target.Host != "" {
		return h.batchEntryError(http.StatusBadRequest, fhir.IssueTypeInvalid,
			"a batch entry request.url must be relative to the server base")
	}
	// An entry addressing the base endpoint would nest another batch/transaction, which the spec
	// forbids and which would recurse through this handler; it fails closed per-entry.
	if strings.Trim(target.Path, "/") == "" {
		return h.batchEntryError(http.StatusBadRequest, fhir.IssueTypeInvalid,
			"a batch entry cannot target the base endpoint; nested batch/transaction processing is not supported")
	}
	// Dispatching internally bypasses net/http's ServeMux path canonicalisation, so a non-canonical
	// entry path ("Patient/../Secret", "Patient//1") would reach the routing edge un-cleaned, an
	// authorization and routing asymmetry with the standalone request the mux would have normalised or
	// redirected. A path that is not already canonical fails closed per-entry rather than being cleaned
	// silently (a client that meant "Patient/1" writes "Patient/1").
	canonical := "/" + strings.Trim(target.Path, "/")
	if path.Clean(canonical) != canonical {
		return h.batchEntryError(http.StatusBadRequest, fhir.IssueTypeInvalid,
			"a batch entry request.url must be a canonical path")
	}

	sub, serr := h.batchSubRequest(outer, entry, target)
	if serr != nil {
		return h.batchEntryError(http.StatusBadRequest, fhir.IssueTypeStructure,
			"a batch entry resource could not be encoded for processing")
	}
	rec := newBatchRecorder()
	h.ServeHTTP(rec, sub)
	return h.batchRecordedResponse(rec)
}

// batchSubRequest synthesizes the standalone-equivalent request for one batch entry: the entry's
// method against the base-stripped path ServeHTTP routes on, the entry's resource marshalled as the
// FHIR JSON body (with the write media type the standalone gates require), and the entry.request
// conditional fields mapped onto their HTTP headers so the standalone conditional semantics apply
// unchanged. The sub-request is cloned from the outer batch request so the connection facts the
// handlers consult (context, Host, TLS) carry over — the entry is the outer client's request, just
// routed internally.
func (h *fhirHandler) batchSubRequest(outer *http.Request, entry batchEntry, target *url.URL) (*http.Request, error) {
	sub := outer.Clone(outer.Context())
	sub.Method = entry.method
	sub.URL = &url.URL{Path: "/" + strings.Trim(target.Path, "/"), RawQuery: target.RawQuery}
	sub.RequestURI = ""
	sub.Header = make(http.Header)
	sub.Body = http.NoBody
	sub.ContentLength = 0

	if entry.resource != nil {
		body, contentType, err := h.batchEntryBody(entry)
		if err != nil {
			return nil, err
		}
		sub.Body = io.NopCloser(bytes.NewReader(body))
		sub.ContentLength = int64(len(body))
		sub.Header.Set("Content-Type", contentType)
	}
	if entry.ifNoneExist != "" {
		sub.Header.Set("If-None-Exist", entry.ifNoneExist)
	}
	if entry.ifMatch != "" {
		sub.Header.Set("If-Match", entry.ifMatch)
	}
	if entry.ifNoneMatch != "" {
		sub.Header.Set("If-None-Match", entry.ifNoneMatch)
	}
	if entry.ifModifiedSince != "" {
		sub.Header.Set("If-Modified-Since", entry.ifModifiedSince)
	}
	return sub, nil
}

// batchEntryBody produces the sub-request body and Content-Type for one batch entry's resource. A
// PATCH entry conveys a JSON Patch as a Binary (contentType application/json-patch+json, base64 data,
// FHIR R5 http.html#patch inside a bundle), so its body is the decoded patch document under the JSON
// Patch media type the standalone PATCH gate requires — a fhir+json body would never satisfy that
// gate. Every other entry marshals its resource as fhir+json, the media type the write gates require.
func (h *fhirHandler) batchEntryBody(entry batchEntry) (body []byte, contentType string, err error) {
	if strings.EqualFold(entry.method, http.MethodPatch) {
		if payload, ok := h.adapter.patchPayload(entry.resource); ok {
			return payload, mediaTypeJSONPatch, nil
		}
	}
	body, err = json.Marshal(entry.resource)
	if err != nil {
		return nil, "", err
	}
	return body, mediaTypeFHIRJSON, nil
}

// batchRecordedResponse folds a dispatched entry's recorded response into its batch-response entry:
// the status line, the Location/ETag/Last-Modified headers a write emitted, the resource a successful
// read/search/vread returned (2xx with a decodable body — a search decodes to a searchset Bundle,
// correct per the spec), and — for an error status — the OperationOutcome body the standalone error
// path wrote, re-decoded so it rides in response.outcome. A body that is not decodable is dropped
// rather than guessed at; the status line still reports the result.
func (h *fhirHandler) batchRecordedResponse(rec *batchRecorder) batchEntryResponse {
	resp := batchEntryResponse{
		status:   strconv.Itoa(rec.status) + " " + http.StatusText(rec.status),
		location: rec.header.Get("Location"),
		etag:     rec.header.Get("ETag"),
	}
	if lm := rec.header.Get("Last-Modified"); lm != "" {
		if t, perr := http.ParseTime(lm); perr == nil {
			resp.lastModified = t.UTC().Format(time.RFC3339)
		}
	}
	if rec.body.Len() > 0 {
		if decoded, err := h.adapter.unmarshalResource(rec.body.Bytes()); err == nil {
			if rec.status >= http.StatusBadRequest {
				resp.outcome = decoded
			} else {
				resp.resource = decoded
			}
		}
	}
	return resp
}

// batchEntryError builds the per-entry error response for an entry the executor rejects before
// dispatch, carrying the same single-issue OperationOutcome shape the standalone error paths write.
// The diagnostic names the structural rule, never a value (PRD §9.1).
func (h *fhirHandler) batchEntryError(status int, code fhir.IssueType, diagnostics string) batchEntryResponse {
	return batchEntryResponse{
		status:  strconv.Itoa(status) + " " + http.StatusText(status),
		outcome: h.singleIssueOutcome(code, diagnostics),
	}
}

// batchRecorder captures one batch entry's sub-response in memory: the status code, headers, and
// body the standalone handler wrote. It is the minimal http.ResponseWriter the internal dispatch
// needs; only the first WriteHeader is honoured, matching net/http semantics.
type batchRecorder struct {
	header      http.Header
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func newBatchRecorder() *batchRecorder {
	return &batchRecorder{header: make(http.Header), status: http.StatusOK}
}

func (rec *batchRecorder) Header() http.Header { return rec.header }

func (rec *batchRecorder) WriteHeader(code int) {
	if rec.wroteHeader {
		return
	}
	rec.status = code
	rec.wroteHeader = true
}

func (rec *batchRecorder) Write(p []byte) (int, error) {
	rec.wroteHeader = true
	return rec.body.Write(p)
}
