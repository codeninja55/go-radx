package dicomweb

import (
	"fmt"
	"mime"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// DICOMweb media types (PS3.18 §8.7). application/dicom carries a Part 10-less raw
// instance in a multipart/related part; application/dicom+json carries metadata and
// store/query responses; application/octet-stream carries raw frame or bulk-data bytes.
const (
	mediaTypeDICOM     = "application/dicom"
	mediaTypeDICOMJSON = "application/dicom+json"
	mediaTypeOctet     = "application/octet-stream"
	mediaTypeMultipart = "multipart/related"
)

// relatedContentType builds the multipart/related Content-Type with the given root type
// parameter, e.g. multipart/related; type="application/dicom". The boundary is appended
// by the multipart writer; this is the Accept-header form.
func relatedContentType(rootType string) string {
	return fmt.Sprintf("%s; type=%q", mediaTypeMultipart, rootType)
}

// acceptInstances is the Accept header a WADO-RS instance retrieval sends: a
// multipart/related body of application/dicom parts. When the caller stated transfer
// syntax preferences, each becomes its own media range carrying the transfer-syntax
// media-type parameter (PS3.18 §8.7.3.3), ordered most-preferred first. They are
// comma-separated as distinct ranges — not one comma-joined parameter value — because in
// an HTTP Accept header a comma separates media ranges, so a joined value would hide the
// fallbacks from the origin. With no preference a single unconstrained range is sent and
// the origin uses its default.
func acceptInstances(ts ...dicom.TransferSyntax) string {
	base := relatedContentType(mediaTypeDICOM)
	if len(ts) == 0 {
		return base
	}
	ranges := make([]string, 0, len(ts))
	for _, t := range ts {
		ranges = append(ranges, fmt.Sprintf("%s; transfer-syntax=%s", base, string(t)))
	}
	return strings.Join(ranges, ", ")
}

// negotiateMultipartDICOM reports whether an Accept header admits a multipart/related
// body of application/dicom parts encoded in emitTS, the transfer syntax the server can
// produce. An empty Accept (no preference) is accepted: the origin may return its default
// representation. A present Accept must name multipart/related with a compatible type
// parameter, application/dicom, or a wildcard. When a range further constrains
// transfer-syntax, it is satisfiable only if it names emitTS or the "*" wildcard;
// otherwise the caller answers 406 rather than return a syntax the client did not accept.
func negotiateMultipartDICOM(accept string, emitTS dicom.TransferSyntax) bool {
	return negotiate(accept, func(mt string, params map[string]string) bool {
		switch mt {
		case mediaTypeMultipart:
			if t, ok := params["type"]; ok && t != mediaTypeDICOM && t != "*/*" {
				return false
			}
			return transferSyntaxAcceptable(params, emitTS)
		case mediaTypeDICOM, "application/*", "*/*":
			return transferSyntaxAcceptable(params, emitTS)
		default:
			return false
		}
	})
}

// transferSyntaxAcceptable reports whether a media range's transfer-syntax parameter (if
// any) admits emitTS. An absent parameter or the "*" wildcard accepts any syntax; a
// concrete parameter must name emitTS. The parameter may be a comma-separated list, each
// entry of which is checked.
func transferSyntaxAcceptable(params map[string]string, emitTS dicom.TransferSyntax) bool {
	ts, ok := params["transfer-syntax"]
	if !ok || ts == "" {
		return true
	}
	for _, want := range strings.Split(ts, ",") {
		want = strings.TrimSpace(want)
		if want == "*" || want == string(emitTS) {
			return true
		}
	}
	return false
}

// negotiateDICOMJSON reports whether an Accept header admits application/dicom+json
// (used by WADO-RS metadata and QIDO-RS). An empty Accept defaults to acceptable.
// application/dicom+xml is explicitly deferred, so an Accept naming only XML is
// unacceptable (the caller answers 406).
func negotiateDICOMJSON(accept string) bool {
	return negotiate(accept, func(mt string, _ map[string]string) bool {
		switch mt {
		case mediaTypeDICOMJSON, "application/json", "application/*", "*/*":
			return true
		default:
			return false
		}
	})
}

// negotiate parses a comma-separated Accept header and reports whether any of its media
// ranges satisfies match. An empty header means "no preference" and is always accepted.
// A media range that fails to parse is skipped rather than treated as a match, so a
// malformed Accept never silently widens what is served.
//
// TODO: M8 — honour the q-value: a range with q=0 explicitly refuses that representation
// and full quality-ordered selection across ranges is out of this thin slice's scope.
func negotiate(accept string, match func(mt string, params map[string]string) bool) bool {
	accept = strings.TrimSpace(accept)
	if accept == "" {
		return true
	}
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mt, params, err := mime.ParseMediaType(part)
		if err != nil {
			continue
		}
		if match(strings.ToLower(mt), params) {
			return true
		}
	}
	return false
}

// isMultipartRelated reports whether a Content-Type names multipart/related, the framing
// a STOW-RS request body and a WADO-RS instance/frame response use.
func isMultipartRelated(contentType string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return strings.ToLower(mt) == mediaTypeMultipart
}
