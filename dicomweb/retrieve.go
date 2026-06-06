package dicomweb

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// RetrievedInstance is one instance a WADO-RS retrieval backend returns: its dataset
// together with the transfer syntax it is stored in. The transfer syntax drives the
// retrieve transfer-syntax policy (passthrough, transcode, or 406); a zero TransferSyntax is
// treated as the universal uncompressed syntax (Explicit VR Little Endian), the safe default
// every DICOMweb origin accepts.
type RetrievedInstance struct {
	DataSet        *dicom.DataSet
	TransferSyntax dicom.TransferSyntax
}

// transferSyntaxOrDefault returns the instance's stored transfer syntax, substituting the
// default uncompressed syntax when none is set so a backend that omits it still negotiates.
func (si RetrievedInstance) transferSyntaxOrDefault() dicom.TransferSyntax {
	if si.TransferSyntax == "" {
		return defaultStorageTransferSyntax
	}
	return si.TransferSyntax
}

// BulkDataObject is one raw bulk-data payload a backend returns for an instance-level
// bulkdata retrieval: the opaque octets and their media type (application/octet-stream by
// default). It carries no attribute identity, since the WADO-RS bulkdata sub-resource
// returns every bulk-data value of the instance as ordered octet-stream parts (PS3.18
// §10.4.4).
type BulkDataObject struct {
	MediaType string
	Data      []byte
}

// StudyRetriever is the optional WADO-RS study-level retrieval backend. A backend that does
// not implement it makes /studies/{study} retrieval answer 501; the base RetrieveBackend
// only mandates instance retrieval (ISP, PRD §8.2).
type StudyRetriever interface {
	RetrieveStudy(ctx context.Context, study dicom.UID) ([]RetrievedInstance, error)
}

// SeriesRetriever is the optional WADO-RS series-level retrieval backend.
type SeriesRetriever interface {
	RetrieveSeries(ctx context.Context, study, series dicom.UID) ([]RetrievedInstance, error)
}

// StoredInstanceRetriever is the optional WADO-RS instance-level retrieval backend that
// reports the instance's stored transfer syntax, so the server's retrieve transfer-syntax
// policy can answer passthrough/transcode/406. A backend that implements only the base
// RetrieveBackend.RetrieveInstance is treated as storing the default uncompressed syntax.
type StoredInstanceRetriever interface {
	RetrieveStoredInstance(ctx context.Context, p ResourcePath) (RetrievedInstance, error)
}

// MetadataRetriever is the optional WADO-RS metadata backend. It returns the datasets whose
// DICOM-JSON metadata the server emits at the requested level; bulk-data values are emitted
// as BulkDataURI references resolvable through the server's own bulkdata sub-resource.
type MetadataRetriever interface {
	RetrieveMetadata(ctx context.Context, p ResourcePath) ([]RetrievedInstance, error)
}

// FrameRetriever is the optional WADO-RS frame backend. It returns the requested 1-based
// frames as raw octet-stream payloads, in the order requested. A frame number outside the
// instance is a typed error mapped to 404 (PS3.18 §10.4.3).
type FrameRetriever interface {
	RetrieveFrames(ctx context.Context, p ResourcePath, frames []int) ([]BulkDataObject, error)
}

// BulkDataRetriever is the optional WADO-RS bulkdata backend. It returns every bulk-data
// value of the instance as ordered octet-stream payloads.
type BulkDataRetriever interface {
	RetrieveBulkData(ctx context.Context, p ResourcePath) ([]BulkDataObject, error)
}

// handleRetrieveStudy answers a WADO-RS study GET with a multipart/related body of
// application/dicom parts, one per instance in the study. It applies the retrieve
// transfer-syntax policy per instance: an instance whose stored syntax the Accept header
// admits is passed through, otherwise the request is unacceptable (the server transcodes no
// pixel data in this slice). A backend that does not implement StudyRetriever answers 501.
func (s *Server) handleRetrieveStudy(w http.ResponseWriter, r *http.Request, study dicom.UID) {
	br, ok := s.retrieve.(StudyRetriever)
	if !ok {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "WADO-RS study retrieval is not implemented")
		return
	}
	if !negotiateMediaTypeDICOM(r.Header.Get("Accept")) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"WADO-RS study retrieval serves multipart/related application/dicom only")
		return
	}
	if err := validateUID(study, "StudyInstanceUID"); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid resource path")
		return
	}
	instances, err := br.RetrieveStudy(r.Context(), study)
	if err != nil {
		s.writeProblem(w, r, http.StatusNotFound, err, "study not found")
		return
	}
	s.writeInstanceParts(w, r, instances)
}

// handleRetrieveSeries answers a WADO-RS series GET with a multipart/related body of
// application/dicom parts, one per instance in the series.
func (s *Server) handleRetrieveSeries(w http.ResponseWriter, r *http.Request, study, series dicom.UID) {
	br, ok := s.retrieve.(SeriesRetriever)
	if !ok {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "WADO-RS series retrieval is not implemented")
		return
	}
	if !negotiateMediaTypeDICOM(r.Header.Get("Accept")) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"WADO-RS series retrieval serves multipart/related application/dicom only")
		return
	}
	if err := validateUID(study, "StudyInstanceUID"); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid resource path")
		return
	}
	if err := validateUID(series, "SeriesInstanceUID"); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid resource path")
		return
	}
	instances, err := br.RetrieveSeries(r.Context(), study, series)
	if err != nil {
		s.writeProblem(w, r, http.StatusNotFound, err, "series not found")
		return
	}
	s.writeInstanceParts(w, r, instances)
}

// writeInstanceParts encodes each RetrievedInstance as an application/dicom part of one
// multipart/related body, applying the retrieve transfer-syntax policy per instance. When
// any instance's stored syntax is not admitted by the Accept header (and the server cannot
// transcode it), the whole request is answered 406, never a partial body that silently
// drops the unservable instances (PRD §9.2 fail-closed). An empty study/series is 404.
func (s *Server) writeInstanceParts(w http.ResponseWriter, r *http.Request, instances []RetrievedInstance) {
	if len(instances) == 0 {
		s.writeProblem(w, r, http.StatusNotFound, ErrInvalidResource, "no instances found")
		return
	}
	accept := r.Header.Get("Accept")
	parts := make([][]byte, 0, len(instances))
	for _, si := range instances {
		stored := si.transferSyntaxOrDefault()
		decision := negotiateRetrieveTransferSyntax(accept, stored)
		if !decision.acceptable {
			s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
				"no acceptable transfer syntax for a retrieved instance")
			return
		}
		raw, err := encodeInstance(si.DataSet, decision.syntax)
		if err != nil {
			s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot encode a retrieved instance")
			return
		}
		parts = append(parts, raw)
	}

	var buf strings.Builder
	mw := NewMultipartWriter(&buf, mediaTypeDICOM)
	for _, raw := range parts {
		if err := mw.AddPart(mediaTypeDICOM, strings.NewReader(string(raw))); err != nil {
			s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot frame a retrieved instance")
			return
		}
	}
	if _, err := mw.Close(); err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot close the multipart body")
		return
	}
	w.Header().Set("Content-Type", mw.ContentType())
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, buf.String())
}

// handleRetrieveMetadata answers a WADO-RS metadata GET with an application/dicom+json
// array of the per-instance metadata at the requested level (study, series, or instance).
// Each instance's bulk-data values are emitted as BulkDataURI references rooted at the
// server's own bulkdata sub-resource, so a client can resolve them through this same origin
// (PS3.18 §10.4.1.1.5). A backend that does not implement MetadataRetriever answers 501.
func (s *Server) handleRetrieveMetadata(w http.ResponseWriter, r *http.Request, p ResourcePath) {
	br, ok := s.retrieve.(MetadataRetriever)
	if !ok {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "WADO-RS metadata retrieval is not implemented")
		return
	}
	if !negotiateDICOMJSON(r.Header.Get("Accept")) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"WADO-RS metadata retrieval serves application/dicom+json only")
		return
	}
	if _, err := p.Path(); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid resource path")
		return
	}

	instances, err := br.RetrieveMetadata(r.Context(), p)
	if err != nil {
		s.writeProblem(w, r, http.StatusNotFound, err, "resource not found")
		return
	}
	if len(instances) == 0 {
		s.writeProblem(w, r, http.StatusNotFound, ErrInvalidResource, "no metadata found")
		return
	}

	datasets := make([]*dicom.DataSet, 0, len(instances))
	for _, si := range instances {
		datasets = append(datasets, si.DataSet)
	}
	body, err := marshalMetadata(datasets, s.originBaseURL(r))
	if err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot encode the metadata")
		return
	}
	w.Header().Set("Content-Type", mediaTypeDICOMJSON)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// routeFrames parses the frame-list segment of a WADO-RS frames target into 1-based frame
// numbers and dispatches to handleRetrieveFrames. A malformed or non-positive frame number
// is a bad-request resource fault, never a per-frame failure (PS3.18 §10.4.3).
func (s *Server) routeFrames(w http.ResponseWriter, r *http.Request, segs []string) {
	p := retrievePath(segs)
	if _, err := p.Path(); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid resource path")
		return
	}
	frames, err := parseFrameList(segs[7])
	if err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid frame list")
		return
	}
	s.handleRetrieveFrames(w, r, p, frames)
}

// parseFrameList parses a comma-separated list of 1-based frame numbers, e.g. "1,4,5". An
// empty list, a non-numeric entry, or a frame below 1 is rejected with ErrInvalidResource;
// the offending text is never echoed, since a malformed path segment is attacker-controlled
// (PRD §9.1).
func parseFrameList(list string) ([]int, error) {
	if strings.TrimSpace(list) == "" {
		return nil, fmt.Errorf("%w: empty frame list", ErrInvalidResource)
	}
	fields := strings.Split(list, ",")
	frames := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return nil, fmt.Errorf("%w: frame number is not an integer", ErrInvalidResource)
		}
		if n < 1 {
			return nil, fmt.Errorf("%w: frame number is below 1 (frames are 1-based)", ErrInvalidResource)
		}
		frames = append(frames, n)
	}
	return frames, nil
}

// handleRetrieveFrames answers a WADO-RS frames GET with a multipart/related body of
// application/octet-stream parts, one per requested 1-based frame, in request order. A
// backend that does not implement FrameRetriever answers 501; a frame outside the instance
// surfaces as the backend's error mapped to 404.
func (s *Server) handleRetrieveFrames(w http.ResponseWriter, r *http.Request, p ResourcePath, frames []int) {
	br, ok := s.retrieve.(FrameRetriever)
	if !ok {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "WADO-RS frame retrieval is not implemented")
		return
	}
	if !negotiateMultipartOctet(r.Header.Get("Accept"), defaultStorageTransferSyntax) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"WADO-RS frame retrieval serves multipart/related application/octet-stream only")
		return
	}
	objs, err := br.RetrieveFrames(r.Context(), p, frames)
	if err != nil {
		s.writeProblem(w, r, http.StatusNotFound, err, "frames not found")
		return
	}
	s.writeOctetParts(w, r, objs)
}

// handleRetrieveBulkData answers a WADO-RS bulkdata GET with a multipart/related body of
// application/octet-stream parts, one per bulk-data value of the instance. A backend that
// does not implement BulkDataRetriever answers 501.
func (s *Server) handleRetrieveBulkData(w http.ResponseWriter, r *http.Request, p ResourcePath) {
	br, ok := s.retrieve.(BulkDataRetriever)
	if !ok {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "WADO-RS bulkdata retrieval is not implemented")
		return
	}
	// Bulk-data octets carry no transfer syntax of their own, so a concrete transfer-syntax
	// Accept parameter cannot be satisfied; passing an empty emitTS makes such a request 406.
	if !negotiateMultipartOctet(r.Header.Get("Accept"), "") {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"WADO-RS bulkdata retrieval serves multipart/related application/octet-stream only")
		return
	}
	objs, err := br.RetrieveBulkData(r.Context(), p)
	if err != nil {
		s.writeProblem(w, r, http.StatusNotFound, err, "bulkdata not found")
		return
	}
	s.writeOctetParts(w, r, objs)
}

// writeOctetParts frames the given payloads as a multipart/related body of
// application/octet-stream parts. An empty payload set is 404, so a request for a resource
// that holds none never reads as an empty success.
func (s *Server) writeOctetParts(w http.ResponseWriter, r *http.Request, objs []BulkDataObject) {
	if len(objs) == 0 {
		s.writeProblem(w, r, http.StatusNotFound, ErrInvalidResource, "no payload found")
		return
	}
	var buf strings.Builder
	mw := NewMultipartWriter(&buf, mediaTypeOctet)
	for _, o := range objs {
		mt := o.MediaType
		if mt == "" {
			mt = mediaTypeOctet
		}
		if err := mw.AddPart(mt, strings.NewReader(string(o.Data))); err != nil {
			s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot frame a payload")
			return
		}
	}
	if _, err := mw.Close(); err != nil {
		s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot close the multipart body")
		return
	}
	w.Header().Set("Content-Type", mw.ContentType())
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, buf.String())
}

// originBaseURL returns the absolute origin base URL ("scheme://authority") the metadata
// response roots its BulkDataURI references at, so a client can resolve them through this
// same origin. The scheme is derived from the request's TLS state and the authority from the
// Host header. A request with no Host yields an empty base, which leaves bulk-data values
// inlined rather than referenced.
func (s *Server) originBaseURL(r *http.Request) string {
	if r.Host == "" {
		return ""
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// marshalMetadata encodes the per-instance metadata datasets as the application/dicom+json
// array a WADO-RS metadata response returns. Each instance's binary values are emitted as
// BulkDataURI references rooted at that instance's own bulkdata sub-resource
// (".../instances/{sop}/bulkdata/"), so a client resolves them through this origin (PS3.18
// §10.4.1.1.5). The threshold is 1 byte so every binary value becomes a reference; an empty
// origin base URL inlines, since no resolvable absolute reference can be built without it.
func marshalMetadata(datasets []*dicom.DataSet, originBaseURL string) ([]byte, error) {
	parts := make([][]byte, 0, len(datasets))
	for _, ds := range datasets {
		opts := metadataJSONOptions(ds, originBaseURL)
		raw, err := MarshalJSON(ds, opts...)
		if err != nil {
			return nil, err
		}
		parts = append(parts, raw)
	}
	return joinJSONArray(parts), nil
}

// metadataJSONOptions returns the DICOM-JSON options for one instance's metadata: when the
// origin base URL is set and the instance carries its identity UIDs, binary values are
// referenced as a BulkDataURI rooted at the instance's bulkdata sub-resource; otherwise they
// are inlined. The per-element locator the codec appends keeps two binary attributes
// distinct under the same instance prefix.
func metadataJSONOptions(ds *dicom.DataSet, originBaseURL string) []JSONOption {
	if originBaseURL == "" {
		return nil
	}
	study, _ := ds.GetString(dicom.TagStudyInstanceUID)
	series, _ := ds.GetString(dicom.TagSeriesInstanceUID)
	sop, _ := ds.GetString(dicom.TagSOPInstanceUID)
	if study == "" || series == "" || sop == "" {
		return nil
	}
	p := NewInstance(dicom.UID(study), dicom.UID(series), dicom.UID(sop))
	bulkPath, err := p.BulkData()
	if err != nil {
		return nil
	}
	base := strings.TrimRight(originBaseURL, "/") + bulkPath + "/"
	return []JSONOption{WithBulkDataThreshold(1), WithBulkDataBaseURL(base)}
}

// joinJSONArray concatenates pre-marshalled JSON objects into a JSON array without
// re-parsing them, preserving each object's canonical key order.
func joinJSONArray(parts [][]byte) []byte {
	var b strings.Builder
	b.WriteByte('[')
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(p)
	}
	b.WriteByte(']')
	return []byte(b.String())
}
