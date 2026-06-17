package dicomweb

import (
	"fmt"
	"mime"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// DICOMweb media types (PS3.18 §8.7). application/dicom carries a Part 10-less raw
// instance in a multipart/related part; application/dicom+json carries metadata and
// store/query responses; application/octet-stream carries raw frame or bulk-data bytes.
const (
	mediaTypeDICOM     = "application/dicom"
	mediaTypeDICOMJSON = "application/dicom+json"
	mediaTypeDICOMXML  = "application/dicom+xml"
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

// negotiateMediaTypeDICOM reports whether an Accept header admits a multipart/related body
// of application/dicom parts, ignoring any transfer-syntax constraint. It is the media-type
// gate the instance/study/series retrieval applies before the stored instance is fetched, so
// an Accept naming a wholly unservable media type (for example application/dicom+xml) fails
// fast as 406 without a backend lookup; the precise transfer-syntax decision is then made
// against the true stored syntax once the instance is in hand.
func negotiateMediaTypeDICOM(accept string) bool {
	return negotiate(accept, func(mt string, params map[string]string) bool {
		switch mt {
		case mediaTypeMultipart:
			if t, ok := params["type"]; ok && t != mediaTypeDICOM && t != "*/*" {
				return false
			}
			return true
		case mediaTypeDICOM, "application/*", "*/*":
			return true
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

// acceptOctetStream is the Accept header a WADO-RS frame or bulkdata retrieval sends: a
// multipart/related body of application/octet-stream parts (PS3.18 §10.4.3, §10.4.4). Raw
// frame and bulk-data octets are carried as octet-stream parts, not application/dicom.
func acceptOctetStream() string {
	return relatedContentType(mediaTypeOctet)
}

// negotiateMultipartOctet reports whether an Accept header admits a multipart/related body
// of application/octet-stream parts, the framing a WADO-RS frame or bulkdata response uses.
// An empty Accept (no preference) is accepted; a present Accept must name multipart/related
// with a compatible type parameter, application/octet-stream, or a wildcard. A
// transfer-syntax parameter is honoured against emitTS for frame retrieval, where the pixel
// data's transfer syntax is meaningful; bulk-data retrieval passes an empty emitTS, which a
// concrete transfer-syntax parameter then cannot match (it answers 406).
func negotiateMultipartOctet(accept string, emitTS dicom.TransferSyntax) bool {
	return negotiate(accept, func(mt string, params map[string]string) bool {
		switch mt {
		case mediaTypeMultipart:
			if t, ok := params["type"]; ok && t != mediaTypeOctet && t != "*/*" {
				return false
			}
			return transferSyntaxAcceptable(params, emitTS)
		case mediaTypeOctet, "application/*", "*/*":
			return transferSyntaxAcceptable(params, emitTS)
		default:
			return false
		}
	})
}

// transferSyntaxDecision is the outcome of the WADO-RS retrieve transfer-syntax policy: the
// transfer syntax the response is encoded in, and whether the stored syntax was served
// unchanged (passthrough) or had to be re-encoded (transcode). When acceptable is false the
// caller answers 406 Not Acceptable rather than serve a syntax the client did not admit.
type transferSyntaxDecision struct {
	syntax      dicom.TransferSyntax
	passthrough bool
	acceptable  bool
}

// negotiateRetrieveTransferSyntax applies the WADO-RS retrieve transfer-syntax policy
// (PS3.18 §8.7.3.3, §10.4): given the syntax an instance is stored in and the syntaxes the
// server can transcode to, it picks the response encoding from the Accept header's
// transfer-syntax parameters.
//
//   - No transfer-syntax constraint (absent parameter, or the "*" wildcard): the stored
//     syntax is served unchanged (passthrough). "*" explicitly means "any syntax you hold",
//     so the origin never transcodes for a wildcard.
//   - A constraint that names the stored syntax: passthrough.
//   - A constraint that names a syntax in transcodable (and not the stored one): transcode.
//   - A constraint that names no servable syntax: not acceptable (the caller answers 406).
//
// A wildcard-or-stored match is preferred over a transcode so a client that accepts the
// stored syntax is never made to pay for a re-encode. transcodable lists the syntaxes the
// server's encoder can actually produce; passing only the stored syntax (or none) makes the
// policy passthrough-or-406, never an unsupported transcode.
func negotiateRetrieveTransferSyntax(accept string, stored dicom.TransferSyntax, transcodable ...dicom.TransferSyntax) transferSyntaxDecision {
	wants := acceptTransferSyntaxes(accept, dicomRangeMatchesMediaType)
	if len(wants) == 0 {
		// No transfer-syntax constraint anywhere in the Accept header: serve what is stored.
		return transferSyntaxDecision{syntax: stored, passthrough: true, acceptable: true}
	}

	var transcodeTo dicom.TransferSyntax
	var haveTranscode bool
	for _, want := range wants {
		if want == "*" || want == string(stored) {
			return transferSyntaxDecision{syntax: stored, passthrough: true, acceptable: true}
		}
		if haveTranscode {
			continue
		}
		for _, t := range transcodable {
			if want == string(t) {
				transcodeTo = t
				haveTranscode = true
				break
			}
		}
	}
	if haveTranscode {
		return transferSyntaxDecision{syntax: transcodeTo, passthrough: false, acceptable: true}
	}
	return transferSyntaxDecision{acceptable: false}
}

// acceptTransferSyntaxes collects the transfer-syntax tokens named across the Accept header's
// media ranges, preserving order and de-duplicating. Only ranges whose media type rangeMatches
// the representation being served contribute their transfer-syntax parameter: a transfer-syntax
// parameter is a property of the media type it qualifies (PS3.18 §8.7.3.3), so a token named on
// an unrelated range (for example application/json) must not constrain the served DICOM part.
// An empty result means no matching range constrained the transfer syntax (the caller serves
// its stored/default syntax). A range that fails to parse is skipped, so a malformed Accept
// never silently widens what is served.
func acceptTransferSyntaxes(accept string, rangeMatches func(mt string, params map[string]string) bool) []string {
	accept = strings.TrimSpace(accept)
	if accept == "" {
		return nil
	}
	var out []string
	seen := make(map[string]struct{})
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mt, params, err := mime.ParseMediaType(part)
		if err != nil {
			continue
		}
		if !rangeMatches(strings.ToLower(mt), params) {
			continue
		}
		ts, ok := params["transfer-syntax"]
		if !ok || ts == "" {
			continue
		}
		for _, want := range strings.Split(ts, ",") {
			want = strings.TrimSpace(want)
			if want == "" {
				continue
			}
			if _, dup := seen[want]; dup {
				continue
			}
			seen[want] = struct{}{}
			out = append(out, want)
		}
	}
	return out
}

// dicomRangeMatchesMediaType reports whether an Accept media range names the WADO-RS
// application/dicom representation, ignoring any transfer-syntax constraint. It is the
// media-type gate acceptTransferSyntaxes uses to decide which ranges may carry a
// transfer-syntax parameter for the served instance: multipart/related with a compatible type
// parameter, application/dicom, or a wildcard. It mirrors negotiateMediaTypeDICOM's media-type
// predicate so the set of ranges admitted there and the set whose transfer-syntax binds here
// stay identical.
func dicomRangeMatchesMediaType(mt string, params map[string]string) bool {
	switch mt {
	case mediaTypeMultipart:
		if t, ok := params["type"]; ok && t != mediaTypeDICOM && t != "*/*" {
			return false
		}
		return true
	case mediaTypeDICOM, "application/*", "*/*":
		return true
	default:
		return false
	}
}

// negotiateDICOMJSON reports whether an Accept header admits application/dicom+json
// (used by QIDO-RS, which serves JSON only). An empty Accept defaults to acceptable.
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

// metadataFormat is the serialization a WADO-RS metadata response uses: DICOM JSON or the
// PS3.19 Native DICOM Model XML.
type metadataFormat int

const (
	// metadataJSON serves application/dicom+json (the default when the Accept expresses no
	// preference between the two metadata media types).
	metadataJSON metadataFormat = iota
	// metadataXML serves application/dicom+xml (the Native DICOM Model).
	metadataXML
	// metadataNotAcceptable means the Accept named neither metadata media type; the caller
	// answers 406.
	metadataNotAcceptable
)

// negotiateMetadataFormat selects the WADO-RS metadata serialization from the Accept header
// (PS3.18 §8.7.3, §10.4.1.1.5). Both application/dicom+json and application/dicom+xml are
// served; an empty Accept, or one naming both forms equally (for example */*), defaults to
// JSON, the prior behaviour and the more compact form. An Accept naming only XML selects XML;
// an Accept naming only JSON selects JSON; an Accept naming neither (and no wildcard) is not
// acceptable. A q=0 refusal of the otherwise-selected form falls through to the other when it
// is acceptable, so "application/dicom+json;q=0, application/dicom+xml" yields XML.
func negotiateMetadataFormat(accept string) metadataFormat {
	if strings.TrimSpace(accept) == "" {
		return metadataJSON
	}
	jsonOK := negotiateDICOMJSON(accept)
	xmlOK := negotiateDICOMXML(accept)
	switch {
	case jsonOK && xmlOK:
		// Both acceptable (a wildcard, or an explicit list of both): prefer JSON.
		return metadataJSON
	case xmlOK:
		return metadataXML
	case jsonOK:
		return metadataJSON
	default:
		return metadataNotAcceptable
	}
}

// negotiateDICOMXML reports whether an Accept header admits the application/dicom+xml Native
// DICOM Model. The model is delivered as a multipart/related body of application/dicom+xml
// parts (PS3.18 §8.7.3.4), so a multipart/related range with a compatible type parameter is
// admitted, as is the bare application/dicom+xml media type, the generic application/xml
// synonym, and the application/* and */* wildcards. An empty Accept defaults to acceptable.
func negotiateDICOMXML(accept string) bool {
	return negotiate(accept, func(mt string, params map[string]string) bool {
		switch mt {
		case mediaTypeMultipart:
			t, ok := params["type"]
			return ok && (t == mediaTypeDICOMXML || t == "*/*")
		case mediaTypeDICOMXML, "application/xml", "application/*", "*/*":
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
// A range carrying q=0 explicitly refuses that representation (RFC 9110 §12.4.2): a matching
// range with q=0 does not admit the representation. HTTP precedence is honoured (RFC 9110
// §12.5.1): the most specific matching range governs, so an Accept of
// "application/dicom+json;q=0, */*" is unacceptable for application/dicom+json because the
// specific q=0 range outranks the */* wildcard that would otherwise admit it. A refusal that
// ties the most specific acceptance also wins, since q=0 is a definite "not acceptable".
// Full quality-ordered selection across competing ranges (q between 0 and 1) is out of scope;
// only the q=0 refusal and its precedence are honoured.
func negotiate(accept string, match func(mt string, params map[string]string) bool) bool {
	accept = strings.TrimSpace(accept)
	if accept == "" {
		return true
	}
	bestAccept, bestRefuse := -1, -1
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		mt, params, err := mime.ParseMediaType(part)
		if err != nil {
			continue
		}
		mt = strings.ToLower(mt)
		if !match(mt, params) {
			continue
		}
		spec := rangeSpecificity(mt, params)
		if isRefused(params) {
			if spec > bestRefuse {
				bestRefuse = spec
			}
			continue
		}
		if spec > bestAccept {
			bestAccept = spec
		}
	}
	// No matching range accepted the representation: unacceptable.
	if bestAccept < 0 {
		return false
	}
	// A matching q=0 refusal at least as specific as the best acceptance vetoes it.
	return bestRefuse < bestAccept
}

// rangeSpecificity ranks a matching media range by HTTP precedence (RFC 9110 §12.5.1): a
// fully specific type/subtype outranks a type/* subtype wildcard, which outranks the */*
// wildcard. A multipart/related range with a concrete type parameter is treated as more
// specific than one without, so a parameterised refusal outranks a bare multipart acceptance.
func rangeSpecificity(mt string, params map[string]string) int {
	switch mt {
	case "*/*":
		return 0
	case "application/*":
		return 1
	}
	if mt == mediaTypeMultipart {
		if t := params["type"]; t != "" && t != "*/*" {
			return 3
		}
		return 2
	}
	return 2
}

// isRefused reports whether a media range's q-value is zero (a q=0 refusal, RFC 9110
// §12.4.2). An absent or unparseable q parameter is treated as q=1 (not refused), so a
// malformed q never silently refuses a representation the client did not refuse.
func isRefused(params map[string]string) bool {
	q, ok := params["q"]
	if !ok {
		return false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(q), 64)
	if err != nil {
		return false
	}
	return v == 0
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
