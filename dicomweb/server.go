package dicomweb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	case r.Method == http.MethodPost && isStudiesStore(segs):
		s.handleStore(w, r, targetStudyUID(segs))
	case r.Method == http.MethodGet && isInstanceRetrieve(segs):
		s.handleRetrieveInstance(w, r, segs)
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

	resp := dicom.NewDataSet()
	var referenced, failed []*dicom.DataSet

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
			// A non-application/dicom part cannot be stored in this slice; drain and reject.
			_, _ = io.Copy(io.Discard, part)
			s.writeProblem(w, r, http.StatusUnsupportedMediaType, ErrUnsupported,
				"STOW-RS parts must be application/dicom")
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

		if targetStudy != "" {
			if study, _ := ds.GetString(dicom.TagStudyInstanceUID); study != targetStudy {
				// The instance belongs to a different study than the URL targets; reject it
				// rather than store it under the wrong hierarchy (PS3.18 §10.5.1).
				failed = append(failed, failedItem(ds, failureReasonNotInStudy))
				continue
			}
		}

		if serr := s.store.Store(r.Context(), ds); serr != nil {
			failed = append(failed, failedItem(ds, storeFailureReason(serr)))
			continue
		}
		referenced = append(referenced, referencedItem(ds))
	}

	if len(referenced) > 0 {
		resp.Set(dicom.Element{
			Tag: dicom.TagReferencedSOPSequence, VR: dicom.VRSQ,
			Value: dicom.NewSequenceValue(dicom.NewSequence(referenced...)),
		})
	}
	if len(failed) > 0 {
		resp.Set(dicom.Element{
			Tag: dicom.TagFailedSOPSequence, VR: dicom.VRSQ,
			Value: dicom.NewSequenceValue(dicom.NewSequence(failed...)),
		})
	}

	out, merr := MarshalJSON(resp)
	if merr != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, merr, "cannot encode the store response")
		return
	}

	w.Header().Set("Content-Type", mediaTypeDICOMJSON)
	w.WriteHeader(storeStatus(len(referenced), len(failed)))
	_, _ = w.Write(out)
}

// storeStatus maps the accepted/failed counts to the STOW-RS HTTP status (PS3.18
// §10.5.3): 200 OK when every instance was accepted, 409 Conflict when none was, and 202
// Accepted for a partial store. An empty body (no parts) is reported as 409, since the
// request stored nothing.
func storeStatus(accepted, failed int) int {
	switch {
	case failed == 0 && accepted > 0:
		return http.StatusOK
	case accepted == 0:
		return http.StatusConflict
	default:
		return http.StatusAccepted
	}
}

// handleRetrieveInstance answers a WADO-RS instance GET with a multipart/related body of
// one application/dicom part. Content negotiation that asks for an unservable
// representation answers 406 (PRD §9.7 fail-closed negotiation).
func (s *Server) handleRetrieveInstance(w http.ResponseWriter, r *http.Request, segs []string) {
	if s.retrieve == nil {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "WADO-RS retrieval is not implemented")
		return
	}
	if !negotiateMultipartDICOM(r.Header.Get("Accept"), defaultStorageTransferSyntax) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"WADO-RS instance retrieval serves multipart/related application/dicom only")
		return
	}

	p := NewInstance(dicom.UID(segs[1]), dicom.UID(segs[3]), dicom.UID(segs[5]))
	if _, err := p.Path(); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid resource path")
		return
	}

	ds, err := s.retrieve.RetrieveInstance(r.Context(), p)
	if err != nil {
		s.writeProblem(w, r, http.StatusNotFound, err, "instance not found")
		return
	}

	raw, err := encodeInstance(ds, defaultStorageTransferSyntax)
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
	var fr *FailureReasonError
	if errors.As(err, &fr) {
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
