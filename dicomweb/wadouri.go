package dicomweb

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// WADO-URI query parameters (PS3.18 §9.3). The legacy URI service identifies a single object
// through the study/series/object UID triple and selects a representation through contentType.
const (
	wadoParamRequestType = "requestType"
	wadoParamStudyUID    = "studyUID"
	wadoParamSeriesUID   = "seriesUID"
	wadoParamObjectUID   = "objectUID"
	wadoParamContentType = "contentType"

	// wadoRequestTypeWADO is the only requestType the URI service defines (PS3.18 §9.1).
	wadoRequestTypeWADO = "WADO"
)

// isWADOURI reports whether a request is a WADO-URI retrieval (PS3.18 §9): a GET carrying the
// requestType=WADO query parameter. The URI service is addressed by query parameters on the
// service URL rather than a hierarchical path, so it is recognised by the parameter, not the
// path. The match is case-insensitive on the requestType value, as DICOM enumerated values are.
func isWADOURI(r *http.Request) bool {
	return strings.EqualFold(r.URL.Query().Get(wadoParamRequestType), wadoRequestTypeWADO)
}

// handleWADOURI answers a WADO-URI single-instance retrieval (PS3.18 §9): a GET whose query
// parameters identify one object by its study/series/object UID triple. It serves
// contentType=application/dicom (the default), returning the complete Part 10 object as the
// raw response body — not the multipart/related framing WADO-RS uses.
//
// Rendered consumer formats (contentType=image/jpeg and the other image media types of PS3.18
// §9.5) are out of scope: rendering is a separate capability tracked as its own parity item,
// and this server ships no pixel-data renderer, so a rendered contentType answers 406 rather
// than silently returning the unrendered object. Every query parameter is validated
// fail-closed: a missing or malformed required parameter is a 400, an unsupported requestType
// or contentType is the appropriate 4xx, and a genuinely absent object is a 404 — never a
// wrong or empty 200 (PRD §9.2).
func (s *Server) handleWADOURI(w http.ResponseWriter, r *http.Request) {
	if s.retrieve == nil {
		s.writeProblem(w, r, http.StatusNotImplemented, ErrUnsupported, "WADO-URI retrieval is not implemented")
		return
	}

	q := r.URL.Query()

	// requestType is validated by isWADOURI before routing here, but a defensive check keeps
	// the handler correct if it is ever called directly.
	if !strings.EqualFold(q.Get(wadoParamRequestType), wadoRequestTypeWADO) {
		s.writeProblem(w, r, http.StatusBadRequest, ErrInvalidResource,
			"WADO-URI requires requestType=WADO")
		return
	}

	// contentType selects the representation. An absent contentType defaults to
	// application/dicom (PS3.18 §9.3.1). A rendered image media type is a recognised but
	// unsupported representation: answer 406, not 400, because the request is well-formed but
	// the server cannot produce that representation.
	contentType := strings.TrimSpace(q.Get(wadoParamContentType))
	if contentType == "" {
		contentType = mediaTypeDICOM
	}
	if strings.EqualFold(contentType, mediaTypeDICOM) {
		s.handleWADOURIDICOM(w, r, q)
		return
	}
	if isRenderedWADOContentType(contentType) {
		s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
			"WADO-URI rendered retrieval (image media types) is not supported; request contentType=application/dicom")
		return
	}
	s.writeProblem(w, r, http.StatusNotAcceptable, ErrNotAcceptable,
		"WADO-URI serves contentType=application/dicom only")
}

// handleWADOURIDICOM serves the contentType=application/dicom representation: the complete
// Part 10 object for the identified instance, returned as the raw response body. The
// study/series/object UID triple is validated as conformant DICOM UIDs (a malformed UID is a
// 400, never interpolated into a backend lookup); a genuinely absent object is a 404.
func (s *Server) handleWADOURIDICOM(w http.ResponseWriter, r *http.Request, q url.Values) {
	study := q.Get(wadoParamStudyUID)
	series := q.Get(wadoParamSeriesUID)
	object := q.Get(wadoParamObjectUID)

	for field, val := range map[string]string{
		"studyUID":  study,
		"seriesUID": series,
		"objectUID": object,
	} {
		if strings.TrimSpace(val) == "" {
			s.writeProblem(w, r, http.StatusBadRequest, ErrInvalidResource,
				"WADO-URI requires "+field)
			return
		}
	}

	p := NewInstance(dicom.UID(study), dicom.UID(series), dicom.UID(object))
	if _, err := p.Path(); err != nil {
		s.writeProblem(w, r, http.StatusBadRequest, err, "invalid WADO-URI object identifier")
		return
	}

	si, err := s.retrieveStoredInstance(r.Context(), p)
	if err != nil {
		s.writeRetrieveBackendError(w, r, err, "object not found")
		return
	}

	// WADO-URI returns the object in whatever transfer syntax the origin holds it (PS3.18
	// §9.3.1; the transfer-syntax negotiation of WADO-RS is not part of the URI service), and
	// WADORetrieveInstanceObject advertises a byte-exact Part 10 object. When the backend
	// supplied the stored bytes, serve them verbatim regardless of transfer syntax: re-encoding
	// from the DataSet would rewrite the File Meta group, padding, and element order, breaking
	// that byte-exact contract. This mirrors the WADO-RS path's preference for Encoded.
	raw := si.Encoded
	if len(raw) == 0 {
		// No stored bytes: fall back to encoding from the DataSet. Passing only the stored
		// syntax to the policy makes it a pure passthrough; an encapsulated stored syntax with
		// no bytes to pass through is ErrNotAcceptable (406 below), never a 500 from a doomed
		// re-encode through dicom.Write, which emits only the four uncompressed syntaxes.
		decision := negotiateRetrieveTransferSyntax("", si.transferSyntaxOrDefault())
		raw, err = encodeRetrievedInstance(si, decision)
		if err != nil {
			s.writeProblem(w, r, http.StatusInternalServerError, err, "cannot encode the retrieved object")
			return
		}
	}

	w.Header().Set("Content-Type", mediaTypeDICOM)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw) // #nosec G705 -- Content-Type is application/dicom (set above), a binary object, not an HTML sink
}

// renderedWADOContentTypePrefixes are the consumer-format media-type families WADO-URI defines
// for rendered retrieval (PS3.18 §9.5): single-frame and multi-frame image and video formats.
// They are recognised so a rendered request gets an honest 406 (rendering is out of scope and
// tracked as a separate parity item) rather than a generic rejection.
var renderedWADOContentTypePrefixes = []string{
	"image/",
	"video/",
	"text/",
	"application/pdf",
}

// isRenderedWADOContentType reports whether a WADO-URI contentType names a rendered consumer
// format rather than application/dicom. The match is case-insensitive and prefix-based so a
// parameterised media type (for example "image/jpeg; quality=90") is still recognised.
func isRenderedWADOContentType(contentType string) bool {
	lower := strings.ToLower(strings.TrimSpace(contentType))
	for _, prefix := range renderedWADOContentTypePrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
