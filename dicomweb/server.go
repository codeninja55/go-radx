package dicomweb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// StoreBackend persists posted instances and reports each instance's outcome (used by
// STOW-RS). It is segregated from retrieval and query so a store-only deployment
// implements only this interface (ISP, PRD §8.2). A nil error means the instance was
// accepted; a non-nil error rejects it and is mapped to a Failure Reason in the store
// response.
type StoreBackend interface {
	Store(ctx context.Context, ds *dicom.DataSet) error
}

// RetrieveBackend resolves a ResourcePath to instances, metadata, frames, and bulk data
// (used by WADO-RS). A retrieve-only deployment implements only this interface.
// RetrieveInstance returns an error wrapping ErrNotFound when the instance does not exist
// (answered 404); any other error is a backend fault answered 500 (the retriever error
// contract documented on StudyRetriever).
type RetrieveBackend interface {
	RetrieveInstance(ctx context.Context, p ResourcePath) (*dicom.DataSet, error)
}

// defaultServerAddr is the loopback address the reference daemon binds when no address
// is configured. Binding to loopback is the safe default; a non-loopback bind is an
// explicit operator choice (PRD §9.1).
const defaultServerAddr = "127.0.0.1:0"

// Server is the embeddable DICOMweb HTTP server. It wires the implemented backends into
// an http.Handler mounted at the DICOMweb root; an unimplemented service answers 501
// with a typed problem document, never a 200 no-op (PRD §9.2).
type Server struct {
	store    StoreBackend
	retrieve RetrieveBackend
	query    QueryBackend

	addr            string
	maxRequestBytes int64
	maxParts        int
	maxQueryResults int
	retrieveURLBase string
}

// ServerOption configures a Server. There is no global configuration; every knob is an
// option (PRD §8.1).
type ServerOption func(*Server)

// WithStoreBackend registers the STOW-RS storage backend.
func WithStoreBackend(b StoreBackend) ServerOption {
	return func(s *Server) { s.store = b }
}

// WithRetrieveBackend registers the WADO-RS retrieval backend.
func WithRetrieveBackend(b RetrieveBackend) ServerOption {
	return func(s *Server) { s.retrieve = b }
}

// WithQueryBackend registers the QIDO-RS query backend.
func WithQueryBackend(b QueryBackend) ServerOption {
	return func(s *Server) { s.query = b }
}

// WithMaxQueryResults caps the number of results a QIDO-RS search returns (default 5,000).
// A search whose result set exceeds the cap is truncated and the response carries the
// Warning: 299 header (PS3.18 §10.6.1.4), so a truncated page is never read as complete.
func WithMaxQueryResults(n int) ServerOption {
	return func(s *Server) { s.maxQueryResults = n }
}

// WithServerAddr sets the listen address. It defaults to a loopback address; passing a
// non-loopback address is an explicit choice the operator makes (PRD §9.1).
func WithServerAddr(addr string) ServerOption {
	return func(s *Server) { s.addr = addr }
}

// WithMaxRequestBytes caps a request body before allocation (PRD §9.3). The default is
// 512 MiB.
func WithMaxRequestBytes(n int64) ServerOption {
	return func(s *Server) { s.maxRequestBytes = n }
}

// WithMaxMultipartParts caps the number of parts in a STOW-RS body (default 10,000).
func WithMaxMultipartParts(n int) ServerOption {
	return func(s *Server) { s.maxParts = n }
}

// WithStoreRetrieveURLBase sets the absolute base URL the STOW-RS store response's Retrieve
// URLs are rooted at, for example https://pacs.example.org/dicom-web. When unset, the base is
// derived per request from its scheme and host, which is correct for a directly-addressed
// server but wrong behind a reverse proxy that rewrites the public origin; set this to the
// public DICOMweb root in that deployment so a returned Retrieve URL resolves from the client.
func WithStoreRetrieveURLBase(base string) ServerOption {
	return func(s *Server) { s.retrieveURLBase = strings.TrimRight(base, "/") }
}

// NewServer constructs a Server from the given options. With no address it binds
// loopback by default.
func NewServer(opts ...ServerOption) (*Server, error) {
	s := &Server{
		addr:            defaultServerAddr,
		maxRequestBytes: defaultMaxResponseBytes,
		maxParts:        defaultMaxParts,
		maxQueryResults: defaultMaxQIDOResults,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// Addr returns the configured listen address.
func (s *Server) Addr() string { return s.addr }

// Handler returns the http.Handler for the DICOMweb root. Mount it under your own mux
// and middleware; it does not assume a path prefix.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.route)
	return mux
}

// route dispatches a request to the matching DICOMweb service. The router is deliberately
// small: it recognises the STOW-RS POST to /studies, the WADO-RS instance GET, and the
// QIDO-RS search GETs, and answers every other path with a typed 501 rather than a silent
// 200 (PRD §9.2).
func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	segs := splitPath(r.URL.Path)

	switch {
	case r.Method == http.MethodGet && isWADOURI(r):
		s.handleWADOURI(w, r)
	case r.Method == http.MethodPost && isStudiesStore(segs):
		s.handleStore(w, r, targetStudyUID(segs))
	case r.Method == http.MethodGet && isMetadataRetrieve(segs):
		s.handleRetrieveMetadata(w, r, retrievePath(segs))
	case r.Method == http.MethodGet && isBulkDataRetrieve(segs):
		s.handleRetrieveBulkData(w, r, retrievePath(segs), segs[7:])
	case r.Method == http.MethodGet && isFrameRetrieve(segs):
		s.routeFrames(w, r, segs)
	case r.Method == http.MethodGet && isInstanceRetrieve(segs):
		s.handleRetrieveInstance(w, r, segs)
	case r.Method == http.MethodGet && isSeriesRetrieve(segs):
		s.handleRetrieveSeries(w, r, dicom.UID(segs[1]), dicom.UID(segs[3]))
	case r.Method == http.MethodGet && isStudyRetrieve(segs):
		s.handleRetrieveStudy(w, r, dicom.UID(segs[1]))
	case isQIDO(r.Method, segs):
		s.handleQuery(w, r, segs)
	default:
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported,
			"the requested DICOMweb service is not implemented")
	}
}

// splitPath splits a cleaned URL path into its non-empty segments.
func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// isStudiesStore reports a STOW-RS target: /studies or /studies/{study}.
func isStudiesStore(segs []string) bool {
	switch len(segs) {
	case 1, 2:
		return segs[0] == "studies"
	default:
		return false
	}
}

// targetStudyUID returns the StudyInstanceUID a /studies/{study} STOW target names, or
// the empty string for the unconstrained /studies target. When present it constrains
// which instances the request may store (PS3.18 §10.5).
func targetStudyUID(segs []string) string {
	if len(segs) == 2 && segs[0] == "studies" {
		return segs[1]
	}
	return ""
}

// isInstanceRetrieve reports a WADO-RS instance target:
// /studies/{study}/series/{series}/instances/{instance}.
func isInstanceRetrieve(segs []string) bool {
	return len(segs) == 6 &&
		segs[0] == "studies" && segs[2] == "series" && segs[4] == "instances"
}

// isStudyRetrieve reports a WADO-RS study target: /studies/{study}. Its final segment is a
// UID, so it never collides with the QIDO-RS /studies search resource.
func isStudyRetrieve(segs []string) bool {
	return len(segs) == 2 && segs[0] == "studies"
}

// isSeriesRetrieve reports a WADO-RS series target:
// /studies/{study}/series/{series}. Its final segment is a UID, so it never collides with
// the QIDO-RS /studies/{study}/series search resource.
func isSeriesRetrieve(segs []string) bool {
	return len(segs) == 4 && segs[0] == "studies" && segs[2] == "series"
}

// isMetadataRetrieve reports a WADO-RS metadata sub-resource target at the study, series,
// or instance level: a path ending in "/metadata" beneath /studies.
func isMetadataRetrieve(segs []string) bool {
	n := len(segs)
	if n == 0 || segs[n-1] != "metadata" || segs[0] != "studies" {
		return false
	}
	switch n {
	case 3: // /studies/{study}/metadata
		return true
	case 5: // /studies/{study}/series/{series}/metadata
		return segs[2] == "series"
	case 7: // /studies/{study}/series/{series}/instances/{instance}/metadata
		return segs[2] == "series" && segs[4] == "instances"
	default:
		return false
	}
}

// isBulkDataRetrieve reports a WADO-RS instance-level bulkdata sub-resource target:
// /studies/{study}/series/{series}/instances/{instance}/bulkdata.
// isBulkDataRetrieve reports a WADO-RS instance-level bulkdata sub-resource target:
// /studies/{study}/series/{series}/instances/{instance}/bulkdata, optionally followed by a
// bulk-data reference suffix (".../bulkdata/{ref}") the metadata response emits per attribute
// (PS3.18 §10.4.4).
func isBulkDataRetrieve(segs []string) bool {
	return len(segs) >= 7 && segs[0] == "studies" && segs[2] == "series" &&
		segs[4] == "instances" && segs[6] == "bulkdata"
}

// isFrameRetrieve reports a WADO-RS frame target:
// /studies/{study}/series/{series}/instances/{instance}/frames/{frameList}.
func isFrameRetrieve(segs []string) bool {
	return len(segs) == 8 && segs[0] == "studies" && segs[2] == "series" &&
		segs[4] == "instances" && segs[6] == "frames"
}

// retrievePath builds the ResourcePath a metadata or bulkdata sub-resource addresses from
// its path segments. The metadata and bulkdata keywords sit at the tail, so the identifying
// UIDs are at the fixed study/series/instance offsets.
func retrievePath(segs []string) ResourcePath {
	var p ResourcePath
	if len(segs) >= 2 {
		p.Study = dicom.UID(segs[1])
	}
	if len(segs) >= 4 && segs[2] == "series" {
		p.Series = dicom.UID(segs[3])
	}
	if len(segs) >= 6 && segs[4] == "instances" {
		p.Instance = dicom.UID(segs[5])
	}
	return p
}

// isQIDO reports a QIDO-RS search target: a GET whose final segment is a search resource
// (studies, series, or instances) rather than a concrete identified resource.
func isQIDO(method string, segs []string) bool {
	if method != http.MethodGet || len(segs) == 0 {
		return false
	}
	switch segs[len(segs)-1] {
	case "studies", "series", "instances":
		return true
	default:
		return false
	}
}

// failureReasonNotInStudy is the STOW-RS Failure Reason (0008,1197) for an instance
// whose StudyInstanceUID does not match the study named in a /studies/{study} target
// (PS3.18 §10.5.1, "Referenced SOP Instance is not in the requested Study").
const failureReasonNotInStudy uint16 = 0xC120

// handleStore reads a STOW-RS multipart/related body of application/dicom parts, stores
// each accepted instance through the StoreBackend, and writes the application/dicom+json
// store response. When the target is /studies/{study}, an instance whose
// StudyInstanceUID does not match is rejected into the Failed SOP Sequence rather than
// stored under an unrelated hierarchy. The HTTP status follows PS3.18 §10.5.3: 200 when
// every instance was accepted, 202 Accepted for a partial store, and 409 Conflict when
// no instance was accepted — fail-closed so a partial store is never read as a clean
// success (PRD §9.2).
//
// STOW-RS is not transactional: parts are stored as they are parsed, so a hard transport
// or parse failure partway through a batch can leave earlier parts persisted with no
// store response describing them (the request fails before the response is written).
// This matches PS3.18's non-atomic store model.
// TODO: M8 — offer an all-or-nothing store option that buffers and validates the whole
// body before committing any part, for callers that need transactional STOW semantics.
func (s *Server) handleStore(w http.ResponseWriter, r *http.Request, targetStudy string) {
	if s.store == nil {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "STOW-RS storage is not implemented")
		return
	}
	if !isMultipartRelated(r.Header.Get("Content-Type")) {
		s.writeProblem(w, r, http.StatusUnsupportedMediaType, ErrUnsupported,
			"STOW-RS requires a multipart/related body")
		return
	}
	if targetStudy != "" {
		// A /studies/{study} target must name a conformant UID; a malformed target is an
		// invalid resource path, not a per-instance failure (PS3.18 §10.5, PRD §9.1).
		if err := validateUID(dicom.UID(targetStudy), "StudyInstanceUID"); err != nil {
			s.writeProblem(w, r, http.StatusBadRequest, err, "invalid target study in the request URL")
			return
		}
	}

	body := http.MaxBytesReader(w, r.Body, s.maxRequestBytes)
	mr, err := NewMultipartReader(body, r.Header.Get("Content-Type"))
	if err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "cannot read the multipart/related body")
		return
	}
	mr.MaxParts = s.maxParts

	// The multipart "type" parameter selects the STOW-RS variant (PS3.18 §10.5): a
	// type="application/dicom" body carries whole Part 10 objects, while a
	// type="application/dicom+json" body carries one metadata part plus separate bulkdata
	// parts the metadata references. application/dicom+xml metadata is deferred.
	switch storeBodyType(r.Header.Get("Content-Type")) {
	case mediaTypeDICOMJSON:
		s.storeMetadataBulkData(w, r, mr, targetStudy)
	default:
		s.storeDICOMParts(w, r, mr, targetStudy)
	}
}

// storeBodyType returns the related body's root media type from the multipart "type"
// parameter, lower-cased; an absent or unparseable parameter defaults to application/dicom,
// the whole-object STOW variant.
func storeBodyType(contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return mediaTypeDICOM
	}
	t := strings.ToLower(strings.TrimSpace(params["type"]))
	if t == "" {
		return mediaTypeDICOM
	}
	return t
}

// storeDICOMParts processes the whole-object STOW variant: each application/dicom part is a
// complete Part 10 instance. Accepted instances carry their Retrieve URL (and a Warning Reason
// when the backend warned); rejected instances carry their Failure Reason. The HTTP status
// follows PS3.18 §10.5.3 (200/202/409), fail-closed so a partial store is never read as a
// clean success (PRD §9.2).
func (s *Server) storeDICOMParts(w http.ResponseWriter, r *http.Request, mr *MultipartReader, targetStudy string) {
	b := newStoreResponseBuilder(s.storeRetrieveURLBase(r))
	for {
		ct, part, perr := mr.NextPart()
		if errors.Is(perr, io.EOF) {
			break
		}
		if perr != nil {
			s.writeProblem(w, r, statusForError(perr), perr, "cannot read a STOW-RS part")
			return
		}
		if mt := mediaTypeOf(ct); mt != mediaTypeDICOM {
			// A non-application/dicom part cannot be stored in this variant; drain and reject.
			_, _ = io.Copy(io.Discard, part)
			s.writeProblem(w, r, http.StatusUnsupportedMediaType, ErrUnsupported,
				"STOW-RS application/dicom parts must be application/dicom")
			return
		}

		ds, derr := decodeInstance(part)
		if derr != nil {
			// A part that does not parse is a malformed instance; fail the whole request
			// rather than store a partial object (truncation is failure, PRD §9.2).
			s.writeProblem(w, r, statusForError(derr), derr, "a STOW-RS part is not a valid DICOM instance")
			return
		}
		if err := requireSOPIdentity(ds); err != nil {
			// An instance with no SOP Class/Instance UID cannot be referenced in the store
			// response at all; reject the request rather than store an unreferenceable object
			// or emit a Failed item with empty UIDs (PRD §9.2).
			s.writeProblem(w, r, http.StatusBadRequest, err, "a STOW-RS part has no SOP identity")
			return
		}
		s.storeOne(r, b, ds, targetStudy)
	}
	s.writeStoreResponse(w, r, b)
}

// storeOne stores one decoded instance into the response builder: it rejects an instance that
// does not belong to a constrained target study, then stores the rest through the backend,
// recording an accept (with any Warning Reason) or a failure. The caller has already rejected
// an instance with no SOP identity.
func (s *Server) storeOne(r *http.Request, b *storeResponseBuilder, ds *dicom.DataSet, targetStudy string) {
	if targetStudy != "" {
		if study, _ := ds.GetString(dicom.TagStudyInstanceUID); study != targetStudy {
			// The instance belongs to a different study than the URL targets; reject it rather
			// than store it under the wrong hierarchy (PS3.18 §10.5.1).
			b.reject(ds, failureReasonNotInStudy)
			return
		}
	}
	res, serr := s.storeInstance(r.Context(), ds)
	if serr != nil {
		b.reject(ds, storeFailureReason(serr))
		return
	}
	b.accept(ds, res.Warning)
}

// storeMetadataBulkData processes the metadata-plus-bulkdata STOW variant (PS3.18 §10.5): a
// type="application/dicom+json" body carries a metadata part (a DICOM-JSON array of instances)
// plus separate bulkdata parts the metadata references by BulkDataURI. The bulkdata parts are
// collected keyed by their Content-Location (or Content-ID); each metadata instance is then
// reassembled by resolving its references against that map and stored. A reference that names
// no part fails that instance into the Failed SOP Sequence rather than storing a partial
// object (PRD §9.2).
func (s *Server) storeMetadataBulkData(w http.ResponseWriter, r *http.Request, mr *MultipartReader, targetStudy string) {
	metadata, bulk, perr := readMetadataBulkParts(mr)
	if perr != nil {
		s.writeProblem(w, r, statusForError(perr), perr, "cannot read the metadata+bulkdata body")
		return
	}
	if len(metadata) == 0 {
		s.writeProblem(w, r, http.StatusBadRequest, ErrInvalidResource,
			"the metadata+bulkdata body carried no application/dicom+json metadata part")
		return
	}

	docs, derr := splitJSONInstances(metadata)
	if derr != nil {
		s.writeProblem(w, r, statusForError(derr), derr, "the STOW-RS metadata part is not valid DICOM JSON")
		return
	}

	resolver := func(_ context.Context, uri BulkDataURI) ([]byte, error) {
		raw, ok := bulk[string(uri)]
		if !ok {
			return nil, fmt.Errorf("%w: bulkdata reference resolves to no part", ErrInvalidResource)
		}
		return raw, nil
	}

	b := newStoreResponseBuilder(s.storeRetrieveURLBase(r))
	for _, doc := range docs {
		ds, err := UnmarshalJSON(doc, WithBulkDataResolver(resolver), WithResolverContext(r.Context()))
		if err != nil {
			// A metadata instance whose bulkdata reference is missing or whose JSON is
			// malformed cannot be stored intact; reject it without echoing the document
			// (PRD §9.1, §9.2). With no SOP identity it is recorded as a top-level Other
			// failure rather than a Failed item with empty UIDs.
			b.otherFailure(0x0110)
			continue
		}
		if err := requireSOPIdentity(ds); err != nil {
			b.otherFailure(0x0110)
			continue
		}
		s.storeOne(r, b, ds, targetStudy)
	}
	s.writeStoreResponse(w, r, b)
}

// writeStoreResponse renders the accumulated store response and writes it with the STOW-RS
// status. The status follows the accepted/failed counts (PS3.18 §10.5.3); a top-level Other
// failure with nothing accepted is reported as 409 Conflict.
func (s *Server) writeStoreResponse(w http.ResponseWriter, r *http.Request, b *storeResponseBuilder) {
	resp := b.build()
	out, merr := MarshalJSON(resp)
	if merr != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, merr, "cannot encode the store response")
		return
	}
	accepted, failed := b.counts()
	w.Header().Set("Content-Type", mediaTypeDICOMJSON)
	w.WriteHeader(storeStatus(accepted, failed, b.hasOtherFailure()))
	_, _ = w.Write(out) // #nosec G705 -- Content-Type is application/dicom+json (set above), not an HTML sink
}

// storeStatus maps the store outcome to the STOW-RS HTTP status (PS3.18 §10.5.3): 200 OK
// only when every instance was accepted with no failure of any kind, 409 Conflict when
// nothing was accepted, and 202 Accepted for a partial store. A top-level Other failure is a
// failure even when no per-instance Failed item was built, so a store that accepted some
// instances yet carries a top-level Failure Reason is 202, never 200 — the response body's
// failure must never be laundered into a complete success. An empty body (no parts) reports
// 409, since the request stored nothing.
func storeStatus(accepted, failed int, otherFailure bool) int {
	switch {
	case failed == 0 && !otherFailure && accepted > 0:
		return http.StatusOK
	case accepted == 0:
		return http.StatusConflict
	default:
		return http.StatusAccepted
	}
}

// handleRetrieveInstance answers a WADO-RS instance GET with a multipart/related body of
// one application/dicom part. The instance is encoded in the syntax the retrieve
// transfer-syntax policy selects: passthrough when the stored syntax satisfies the Accept
// transfer-syntax parameter, else 406 (the server transcodes no pixel data in this slice).
// A backend that reports its stored transfer syntax (StoredInstanceRetriever) drives the
// policy with the true stored syntax; a base RetrieveBackend is treated as storing the
// default uncompressed syntax. A passthrough of an encapsulated (compressed) syntax is
// served byte-exact from the instance's stored bytes, since go-radx writes only uncompressed
// syntaxes; when those bytes are absent the request answers 406 rather than a 500 from a
// doomed re-encode. Content negotiation that asks for an unservable representation answers
// 406 (PRD §9.7 fail-closed negotiation).
func (s *Server) handleRetrieveInstance(w http.ResponseWriter, r *http.Request, segs []string) {
	if s.retrieve == nil {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "WADO-RS retrieval is not implemented")
		return
	}

	if !negotiateMediaTypeDICOM(r.Header.Get("Accept")) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"WADO-RS instance retrieval serves multipart/related application/dicom only")
		return
	}

	p := NewInstance(dicom.UID(segs[1]), dicom.UID(segs[3]), dicom.UID(segs[5]))
	if _, err := p.Path(); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid resource path")
		return
	}

	si, err := s.retrieveStoredInstance(r.Context(), p)
	if err != nil {
		s.writeRetrieveBackendError(w, r, err, "instance not found")
		return
	}

	decision := negotiateRetrieveTransferSyntax(r.Header.Get("Accept"), si.transferSyntaxOrDefault())
	if !decision.acceptable {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"no acceptable transfer syntax for the retrieved instance")
		return
	}

	raw, err := encodeRetrievedInstance(si, decision)
	if errors.Is(err, ErrNotAcceptable) {
		// The stored syntax is encapsulated and the backend supplied no byte-exact bytes to
		// pass through; go-radx writes no encapsulated syntax, so this is an honest 406 rather
		// than a 500 from a doomed re-encode (PRD §9.7 fail-closed negotiation).
		s.writeProblem(w, r, http.StatusNotAcceptable, err,
			"the stored instance is in a compressed transfer syntax that cannot be served unchanged")
		return
	}
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot encode the retrieved instance")
		return
	}

	var buf strings.Builder
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	if err := mw.AddPart(mediaTypeDICOM, strings.NewReader(string(raw))); err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot frame the retrieved instance")
		return
	}
	if _, err := mw.Close(); err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot close the multipart body")
		return
	}
	w.Header().Set("Content-Type", mw.ContentType())
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, buf.String())
}

// retrieveStoredInstance resolves an instance through the richer StoredInstanceRetriever
// when the backend implements it, so the retrieve transfer-syntax policy sees the true
// stored syntax. A backend that implements only the base RetrieveBackend yields a
// RetrievedInstance with no transfer syntax set, which the policy treats as the default
// uncompressed syntax (its prior behaviour).
func (s *Server) retrieveStoredInstance(ctx context.Context, p ResourcePath) (RetrievedInstance, error) {
	if sr, ok := s.retrieve.(StoredInstanceRetriever); ok {
		return sr.RetrieveStoredInstance(ctx, p)
	}
	ds, err := s.retrieve.RetrieveInstance(ctx, p)
	if err != nil {
		return RetrievedInstance{}, err
	}
	return RetrievedInstance{DataSet: ds}, nil
}

// referencedItem builds a Referenced SOP Sequence (0008,1199) item for an accepted
// instance: its SOP Class UID (0008,1150) and SOP Instance UID (0008,1155).
func referencedItem(ds *dicom.DataSet) *dicom.DataSet {
	item := dicom.NewDataSet()
	sopClass, _ := ds.GetString(dicom.TagSOPClassUID)
	sopInstance, _ := ds.GetString(dicom.TagSOPInstanceUID)
	item.SetString(dicom.TagReferencedSOPClassUID, sopClass)
	item.SetString(dicom.TagReferencedSOPInstanceUID, sopInstance)
	return item
}

// failedItem builds a Failed SOP Sequence (0008,1198) item for a rejected instance,
// carrying the SOP identity plus the Failure Reason (0008,1197).
func failedItem(ds *dicom.DataSet, reason uint16) *dicom.DataSet {
	item := referencedItem(ds)
	item.Set(dicom.Element{Tag: dicom.TagFailureReason, VR: dicom.VRUS, Value: dicom.NewInts(dicom.VRUS, int64(reason))})
	return item
}

// storeFailureReason maps a StoreBackend error to a STOW-RS Failure Reason code. A
// backend that returns a typed FailureReasonError names its own code; any other error is
// reported as the generic processing-failure reason (0x0110), so the failure is recorded
// without inventing a more specific cause.
func storeFailureReason(err error) uint16 {
	if fr, ok := errors.AsType[*FailureReasonError](err); ok {
		return fr.Reason
	}
	return 0x0110
}

// mediaTypeOf returns the bare media type of a Content-Type header value, lower-cased,
// dropping parameters; an unparseable value yields the empty string.
func mediaTypeOf(contentType string) string {
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = contentType[:i]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

// statusForError maps a typed dicomweb error to the HTTP status it should produce. A
// limit breach is 413, a truncated transfer is 400, and anything else is 400 bad
// request: the body never parsed cleanly.
func statusForError(err error) int {
	switch {
	case errors.Is(err, ErrLimitExceeded):
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusBadRequest
	}
}

// problemDocument is an RFC 9457 application/problem+json body. It carries only a typed
// title and a structural detail, never PHI (PRD §9.1).
type problemDocument struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// writeProblem writes a typed problem document with the given status. The title is the
// sentinel error's message (a structural, PHI-free string); the detail is a fixed
// description. It never echoes request bytes or the URL's query, which could carry PHI.
func (s *Server) writeProblem(w http.ResponseWriter, _ *http.Request, status int, cause error, detail string) {
	title := http.StatusText(status)
	if cause != nil {
		title = sentinelTitle(cause)
	}
	doc := problemDocument{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(doc)
}

// sentinelTitle returns a PHI-free title for an error: the matching package sentinel's
// message when the error is one, else a generic phrase. It never returns an arbitrary
// error string, which could embed input bytes.
func sentinelTitle(err error) string {
	switch {
	case errors.Is(err, ErrUnsupported):
		return ErrUnsupported.Error()
	case errors.Is(err, ErrNotAcceptable):
		return ErrNotAcceptable.Error()
	case errors.Is(err, ErrLimitExceeded):
		return ErrLimitExceeded.Error()
	case errors.Is(err, ErrInvalidResource):
		return ErrInvalidResource.Error()
	case errors.Is(err, ErrNotFound):
		return ErrNotFound.Error()
	default:
		return "dicomweb: request could not be processed"
	}
}

// FailureReasonError lets a StoreBackend report a specific STOW-RS Failure Reason for a
// rejected instance. A backend returning it controls the (0008,1197) code recorded in
// the Failed SOP Sequence; any other error defaults to processing-failure (0x0110).
type FailureReasonError struct {
	Reason uint16
	Msg    string
}

func (e *FailureReasonError) Error() string {
	return fmt.Sprintf("dicomweb: store rejected (%s): %s", failureReasonName(e.Reason), e.Msg)
}

// loopbackOnly reports whether addr names a loopback host. It is used by the reference
// daemon's bind path to assert the default is loopback (PRD §9.1).
func loopbackOnly(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
