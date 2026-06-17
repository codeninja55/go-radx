package dicomweb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strings"

	"github.com/codeninja55/go-radx/dicom"
)

// RetrieveStudy retrieves every instance of a study as a streaming iterator over the
// multipart/related application/dicom parts the origin returns (PS3.18 §10.4). Each
// iteration yields one decoded instance or a typed error; the iterator stops at the first
// error so a truncated body is never read as a complete study. The response body stays open
// for the life of the iteration, so the caller must drain the iterator (or stop early) to
// release it.
func (c *Client) RetrieveStudy(ctx context.Context, study dicom.UID) iter.Seq2[*dicom.DataSet, error] {
	path, err := NewStudy(study).Path()
	return c.retrieveInstances(ctx, path, err)
}

// RetrieveSeries retrieves every instance of a series as a streaming iterator over the
// multipart/related application/dicom parts, with the same semantics as RetrieveStudy.
func (c *Client) RetrieveSeries(ctx context.Context, study, series dicom.UID) iter.Seq2[*dicom.DataSet, error] {
	path, err := NewSeries(study, series).Path()
	return c.retrieveInstances(ctx, path, err)
}

// retrieveInstances issues a multipart/related WADO-RS GET against path and returns an
// iterator that decodes each application/dicom part into a dataset. Only the decoded dataset is
// yielded, so each part is streamed straight into the decoder without capturing its raw bytes — the
// common, memory-frugal path for a large study, in contrast to retrieveInstanceObjects which tees
// each part to preserve the byte-exact representation. pathErr carries a path-construction error so
// the caller can build the path inline; when set, the iterator yields that single error. The bounded
// response body and the multipart reader cap the transfer against a hostile origin (PRD §9.3).
func (c *Client) retrieveInstances(ctx context.Context, path string, pathErr error) iter.Seq2[*dicom.DataSet, error] {
	objects := c.retrieveInstanceStream(ctx, path, pathErr, false)
	return func(yield func(*dicom.DataSet, error) bool) {
		for si, err := range objects {
			if err != nil {
				if !yield(nil, err) {
					return
				}
				continue
			}
			if !yield(si.DataSet, nil) {
				return
			}
		}
	}
}

// RetrieveStudyObjects retrieves every instance of a study, preserving each instance's transfer
// syntax and byte-exact Part 10 representation. Unlike RetrieveStudy, which yields only the decoded
// dataset and so discards the transfer syntax, this yields a RetrievedInstance whose TransferSyntax
// and Encoded fields let the caller write the object back in the syntax the origin returned rather
// than transcoding it. It shares the streaming, bounding, and stop-at-first-error semantics of
// RetrieveStudy.
func (c *Client) RetrieveStudyObjects(ctx context.Context, study dicom.UID) iter.Seq2[RetrievedInstance, error] {
	path, err := NewStudy(study).Path()
	return c.retrieveInstanceObjects(ctx, path, err)
}

// RetrieveSeriesObjects retrieves every instance of a series, preserving each instance's transfer
// syntax and byte-exact Part 10 representation, with the same semantics as RetrieveStudyObjects.
func (c *Client) RetrieveSeriesObjects(ctx context.Context, study, series dicom.UID) iter.Seq2[RetrievedInstance, error] {
	path, err := NewSeries(study, series).Path()
	return c.retrieveInstanceObjects(ctx, path, err)
}

// retrieveInstanceObjects issues a multipart/related WADO-RS GET against path and returns an
// iterator that decodes each application/dicom part into a RetrievedInstance, capturing the
// instance's transfer syntax and byte-exact Part 10 bytes so a retrieval preserves the origin's
// representation. pathErr carries a path-construction error so the caller can build the path inline;
// when set, the iterator yields that single error. The bounded response body and the multipart
// reader cap the transfer against a hostile origin (PRD §9.3).
func (c *Client) retrieveInstanceObjects(ctx context.Context, path string, pathErr error) iter.Seq2[RetrievedInstance, error] {
	return c.retrieveInstanceStream(ctx, path, pathErr, true)
}

// retrieveInstanceStream is the shared streaming retrieve over a multipart/related WADO-RS body.
// captureBytes selects the decode path: the object-returning caller (retrieveInstanceObjects)
// captures each part's byte-exact Part 10 representation, while the dataset-only caller
// (retrieveInstances) decodes without teeing, so a large study does not pay the doubled per-instance
// memory of buffering bytes it would only discard. pathErr carries a path-construction error so the
// caller can build the path inline; when set, the iterator yields that single error. The bounded
// response body and the multipart reader cap the transfer against a hostile origin (PRD §9.3).
func (c *Client) retrieveInstanceStream(ctx context.Context, path string, pathErr error, captureBytes bool) iter.Seq2[RetrievedInstance, error] {
	return func(yield func(RetrievedInstance, error) bool) {
		if pathErr != nil {
			yield(RetrievedInstance{}, pathErr)
			return
		}
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			yield(RetrievedInstance{}, err)
			return
		}
		req.Header.Set("Accept", acceptInstances(c.transferSyntaxes...))

		resp, err := c.do(req, http.MethodGet, path)
		if err != nil {
			yield(RetrievedInstance{}, err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			yield(RetrievedInstance{}, c.httpError(http.MethodGet, path, resp))
			return
		}
		if !isMultipartRelated(resp.Header.Get("Content-Type")) {
			yield(RetrievedInstance{}, fmt.Errorf("%w: WADO-RS response is not multipart/related", ErrNotAcceptable))
			return
		}

		mr, err := NewMultipartReader(c.boundedBody(resp), resp.Header.Get("Content-Type"))
		if err != nil {
			yield(RetrievedInstance{}, err)
			return
		}
		for {
			ct, part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				yield(RetrievedInstance{}, err)
				return
			}
			if mt := mediaTypeOf(ct); mt != mediaTypeDICOM {
				yield(RetrievedInstance{}, fmt.Errorf("%w: WADO-RS part media type %q is not application/dicom", ErrNotAcceptable, mt))
				return
			}
			si, err := decodeRetrievedInstance(part, captureBytes)
			if err != nil {
				yield(RetrievedInstance{}, err)
				return
			}
			if !yield(si, nil) {
				return
			}
		}
	}
}

// RetrieveMetadata fetches the application/dicom+json metadata of a study, series, or
// instance and parses each metadata object into a dataset. A binary value the origin emitted
// as a BulkDataURI is left as an unresolved reference: enumerate it with BulkDataURIs and
// fetch its octets with ResolveBulkDataURI, which honours the client's WithBulkDataBaseURL
// when the origin emitted a relative reference.
func (c *Client) RetrieveMetadata(ctx context.Context, p ResourcePath) ([]*dicom.DataSet, error) {
	path, err := p.Metadata()
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", mediaTypeDICOMJSON)

	resp, err := c.do(req, http.MethodGet, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, c.httpError(http.MethodGet, path, resp)
	}

	raw, err := io.ReadAll(c.boundedBody(resp))
	if err != nil {
		return nil, c.readError(http.MethodGet, path, err)
	}
	return parseMetadataArray(raw)
}

// RetrieveMetadataXML fetches the metadata of a study, series, or instance as the PS3.19
// Native DICOM Model (application/dicom+xml) and parses each instance into a dataset. It is
// the XML twin of RetrieveMetadata: the origin returns a multipart/related body of
// application/dicom+xml parts, one per instance, which this method decodes (PS3.18 §8.7.3.4,
// §10.4.1.1.5). A binary value the origin emitted as a BulkData reference is left unresolved,
// exactly as in RetrieveMetadata; enumerate it with BulkDataURIs and fetch it with
// ResolveBulkDataURI. Use RetrieveMetadata for the more compact JSON form; this method exists
// for interop with origins or workflows that require the Native DICOM Model.
func (c *Client) RetrieveMetadataXML(ctx context.Context, p ResourcePath) ([]*dicom.DataSet, error) {
	path, err := p.Metadata()
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", relatedContentType(mediaTypeDICOMXML))

	resp, err := c.do(req, http.MethodGet, path) // #nosec G704 -- the URL is joined from the caller-configured base URL (newRequest); requesting the configured service is the client's purpose
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, c.httpError(http.MethodGet, path, resp)
	}
	if !isMultipartRelated(resp.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("%w: WADO-RS XML metadata response is not multipart/related", ErrNotAcceptable)
	}

	mr, err := NewMultipartReader(c.boundedBody(resp), resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	var datasets []*dicom.DataSet
	for {
		ct, part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if mt := mediaTypeOf(ct); mt != mediaTypeDICOMXML {
			return nil, fmt.Errorf("%w: WADO-RS metadata part media type %q is not application/dicom+xml", ErrNotAcceptable, mt)
		}
		raw, err := io.ReadAll(part)
		if err != nil {
			return nil, c.readError(http.MethodGet, path, err)
		}
		ds, err := UnmarshalXML(raw)
		if err != nil {
			return nil, err
		}
		datasets = append(datasets, ds)
	}
	if len(datasets) == 0 {
		return nil, fmt.Errorf("%w: WADO-RS XML metadata response carried no part", ErrNotAcceptable)
	}
	return datasets, nil
}

// RetrieveFrames fetches the given 1-based frames of an instance as a multipart/related body
// of application/octet-stream parts and returns each frame's raw octets in request order
// (PS3.18 §10.4.3). A frame number below 1 is rejected before any request is made.
func (c *Client) RetrieveFrames(ctx context.Context, p ResourcePath, frames ...int) ([][]byte, error) {
	path, err := p.Frames(frames...)
	if err != nil {
		return nil, err
	}
	return c.retrieveOctetParts(ctx, path, acceptOctetStream(), "")
}

// RetrieveBulkData fetches every bulk-data value of an instance as a multipart/related body
// of application/octet-stream parts and returns each payload's raw octets (PS3.18 §10.4.4).
func (c *Client) RetrieveBulkData(ctx context.Context, p ResourcePath) ([][]byte, error) {
	path, err := p.BulkData()
	if err != nil {
		return nil, err
	}
	return c.retrieveOctetParts(ctx, path, acceptOctetStream(), "")
}

// RetrieveBulkDataRange fetches a byte range of an instance's bulk data, sending an HTTP Range
// header (PS3.18 §8.7.4, §10.4.4, matching the dicomweb-client byte_range parameter). It is the
// partial-retrieval form of RetrieveBulkData: an origin that supports range requests returns 206
// Partial Content with only the requested octets, sparing a re-fetch of a large bulk-data value.
// An origin that ignores the Range header returns the full 200 body, which is still correct; the
// caller can compare the returned length against the requested span when partial semantics matter.
// The returned parts are the octet-stream parts of the (possibly partial) response.
func (c *Client) RetrieveBulkDataRange(ctx context.Context, p ResourcePath, br ByteRange) ([][]byte, error) {
	if err := br.validate(); err != nil {
		return nil, err
	}
	path, err := p.BulkData()
	if err != nil {
		return nil, err
	}
	return c.retrieveOctetParts(ctx, path, acceptOctetStream(), br.header())
}

// RetrieveFramesRange fetches a byte range of an instance's frame data, sending an HTTP Range
// header (PS3.18 §8.7.4, §10.4.3). It is the partial-retrieval form of RetrieveFrames; the same
// origin-support caveat as RetrieveBulkDataRange applies.
func (c *Client) RetrieveFramesRange(ctx context.Context, p ResourcePath, br ByteRange, frames ...int) ([][]byte, error) {
	if err := br.validate(); err != nil {
		return nil, err
	}
	path, err := p.Frames(frames...)
	if err != nil {
		return nil, err
	}
	return c.retrieveOctetParts(ctx, path, acceptOctetStream(), br.header())
}

// ByteRange names a byte span for a partial retrieval, rendered as an HTTP Range header
// (RFC 7233 §2.1, PS3.18 §8.7.4). It mirrors the dicomweb-client byte_range parameter: a
// (Start, End) pair selecting bytes Start through End inclusive, or an open-ended suffix/prefix.
// All offsets are zero-based, matching the HTTP Range unit.
type ByteRange struct {
	// Start is the zero-based first byte offset. A negative Start selects the final -Start bytes
	// (a suffix range, "bytes=-N") and End is ignored.
	Start int64
	// End is the zero-based last byte offset, inclusive. A nil End (with a non-negative Start)
	// is open-ended, selecting from Start to the end of the value ("bytes=Start-"). End is a
	// pointer so that an inclusive end of zero (the first-byte probe, "bytes=0-0") is
	// expressible and distinct from the open-ended form; use Int64Ptr to set it.
	End *int64
}

// Int64Ptr returns a pointer to v, a convenience for setting ByteRange.End inline (e.g.
// ByteRange{Start: 0, End: dicomweb.Int64Ptr(0)} for the first-byte probe "bytes=0-0").
func Int64Ptr(v int64) *int64 { return &v }

// validate rejects a degenerate range: a forward range whose End precedes its Start. An
// open-ended range (End nil) and a suffix range (Start < 0) are valid.
func (b ByteRange) validate() error {
	if b.Start >= 0 && b.End != nil && *b.End < b.Start {
		return fmt.Errorf("%w: byte range end precedes start", ErrInvalidResource)
	}
	return nil
}

// header renders the ByteRange as an HTTP Range header value ("bytes=...", RFC 7233 §2.1). A
// suffix range (Start < 0) renders "bytes=-N"; an open-ended range (End nil) renders
// "bytes=Start-"; a closed range renders "bytes=Start-End", including the first-byte probe
// "bytes=0-0" when End points to zero.
func (b ByteRange) header() string {
	switch {
	case b.Start < 0:
		return fmt.Sprintf("bytes=%d", b.Start)
	case b.End == nil:
		return fmt.Sprintf("bytes=%d-", b.Start)
	default:
		return fmt.Sprintf("bytes=%d-%d", b.Start, *b.End)
	}
}

// ResolveBulkDataURI fetches the octets a BulkDataURI references. WADO-RS returns a bulk-data
// value as a multipart/related body of application/octet-stream parts; the resolved value is
// the first such part. A relative reference is joined to the client's WithBulkDataBaseURL (or
// the origin base URL when none was set), so a BulkDataURI emitted in metadata against this
// origin is resolvable without the caller reconstructing the absolute URL. An absolute
// reference is fetched as given. The reference is never logged, since it carries resource
// UIDs (PRD §9.1).
//
// An absolute BulkDataURI on an origin other than the configured one is refused by default
// with ErrCrossOriginBulkData, because server-supplied metadata could steer the client at an
// internal address (169.254.169.254, an internal service) — an SSRF. WithAllowCrossOriginBulkData
// or WithBulkDataHostAllowlist opts a trusting caller back in (PRD §9.8).
//
// The client's credential is attached only when the resolved URL is same-origin with the
// configured base URL (matching scheme, host, and port): the credential transport scopes
// every scheme to the origin. Even under the cross-origin opt-in, sending the Authorization
// header to another host would let a malicious or compromised origin harvest the PACS
// credential, so a cross-origin reference is fetched without credentials (PRD §9.8).
func (c *Client) ResolveBulkDataURI(ctx context.Context, uri BulkDataURI) ([]byte, error) {
	return c.resolveBulkDataURI(ctx, uri, "")
}

// ResolveBulkDataURIRange fetches a byte range of the octets a BulkDataURI references, sending an
// HTTP Range header (PS3.18 §8.7.4, matching the dicomweb-client byte_range parameter). It shares
// the same-origin and cross-origin-allowlist policy as ResolveBulkDataURI; an origin that supports
// ranges returns 206 Partial Content with only the requested span, while one that ignores the
// header returns the full body.
func (c *Client) ResolveBulkDataURIRange(ctx context.Context, uri BulkDataURI, br ByteRange) ([]byte, error) {
	if err := br.validate(); err != nil {
		return nil, err
	}
	return c.resolveBulkDataURI(ctx, uri, br.header())
}

// resolveBulkDataURI is the shared bulk-data resolve, with an optional Range header for a partial
// retrieval. It enforces the same-origin/allowlist SSRF guard, then fetches the first
// octet-stream part of the (possibly partial) multipart/related response.
func (c *Client) resolveBulkDataURI(ctx context.Context, uri BulkDataURI, rangeHeader string) ([]byte, error) {
	target := c.absoluteBulkDataURL(string(uri))
	if err := c.checkBulkDataOrigin(target); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: build bulkdata request: %w", err)
	}
	req.Header.Set("Accept", acceptOctetStream())
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: GET bulkdata: %w", sanitizeTransportError(err))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Method: http.MethodGet, URL: "/bulkdata"}
	}
	if !isMultipartRelated(resp.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("%w: bulkdata response is not multipart/related", ErrNotAcceptable)
	}

	mr, err := NewMultipartReader(c.boundedBody(resp), resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	_, part, err := mr.NextPart()
	if errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: bulkdata response carried no part", ErrNotAcceptable)
	}
	if err != nil {
		return nil, err
	}
	return io.ReadAll(part)
}

// absoluteBulkDataURL resolves a possibly relative BulkDataURI against the client's
// configured bulk-data base URL, falling back to the origin base URL. An already-absolute
// reference (one with a scheme) is returned unchanged.
func (c *Client) absoluteBulkDataURL(ref string) string {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	base := c.bulkDataBaseURL
	if base == "" {
		base = c.baseURL
	}
	base = strings.TrimRight(base, "/")
	if ref == "" {
		return base
	}
	if strings.HasPrefix(ref, "/") {
		return base + ref
	}
	return base + "/" + ref
}

// checkBulkDataOrigin enforces the same-origin default on a resolved BulkDataURI. A target that
// is same-origin with the configured base URL always passes; a relative reference resolves
// against that base and so is same-origin by construction. A cross-origin target is refused with
// a CrossOriginBulkDataError unless cross-origin fetching is enabled wholesale or the target's
// host is on the allowlist. The error names only the host, never the UID-bearing path
// (PRD §9.1, §9.8). A target or base that does not parse fails closed: an unparseable target is
// refused, and an unparseable base means no target can be proven same-origin.
func (c *Client) checkBulkDataOrigin(target string) error {
	targetURL, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("dicomweb: %w: bulk-data reference is not a valid URL", ErrInvalidResource)
	}
	base := c.bulkDataBaseURL
	if base == "" {
		base = c.baseURL
	}
	baseURL, baseErr := url.Parse(base)
	if baseErr == nil && sameOrigin(targetURL, baseURL) {
		return nil
	}
	if c.allowCrossOriginBulkData {
		return nil
	}
	// Match the allowlist on the full host (with port, e.g. "host:8443") and on the bare
	// hostname, so an entry given without a port still admits the default-port form.
	if _, ok := c.bulkDataHostAllowlist[strings.ToLower(targetURL.Host)]; ok {
		return nil
	}
	if _, ok := c.bulkDataHostAllowlist[strings.ToLower(targetURL.Hostname())]; ok {
		return nil
	}
	return &CrossOriginBulkDataError{Host: targetURL.Host}
}

// sameOrigin reports whether two URLs share scheme, host, and port. The comparison is
// case-insensitive on scheme and host (per RFC 3986) and normalises the default port so that,
// for example, https without an explicit port matches https on 443.
func sameOrigin(a, b *url.URL) bool {
	if !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	if !strings.EqualFold(a.Hostname(), b.Hostname()) {
		return false
	}
	return originPort(a) == originPort(b)
}

// originPort returns a URL's effective port, substituting the scheme's default when the URL
// names none, so an explicit and an implicit default port compare equal.
func originPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

// retrieveOctetParts issues a multipart/related WADO-RS GET against path with the given
// Accept header and returns each application/octet-stream part's raw octets. A non-octet
// part media type is a typed error rather than a silently-dropped part (PRD §9.2). A non-empty
// rangeHeader sets the HTTP Range header for a byte-range (partial) retrieval; the response may
// then carry 206 Partial Content, which the status check below admits alongside 200.
func (c *Client) retrieveOctetParts(ctx context.Context, path, accept, rangeHeader string) ([][]byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := c.do(req, http.MethodGet, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, c.httpError(http.MethodGet, path, resp)
	}
	if !isMultipartRelated(resp.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("%w: WADO-RS response is not multipart/related", ErrNotAcceptable)
	}

	mr, err := NewMultipartReader(c.boundedBody(resp), resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	var parts [][]byte
	for {
		ct, body, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if mt := mediaTypeOf(ct); mt != mediaTypeOctet {
			return nil, fmt.Errorf("%w: WADO-RS part media type %q is not application/octet-stream", ErrNotAcceptable, mt)
		}
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, c.readError(http.MethodGet, path, err)
		}
		parts = append(parts, data)
	}
	return parts, nil
}

// parseMetadataArray decodes a WADO-RS metadata response, a JSON array of DICOM-JSON
// attribute objects, into datasets (PS3.18 §F.2.2). It mirrors parseSearchResults: a body
// that is not a JSON array is a typed decode error, never a silent empty result.
func parseMetadataArray(raw []byte) ([]*dicom.DataSet, error) {
	results, err := parseSearchResults(raw)
	if err != nil {
		return nil, err
	}
	out := make([]*dicom.DataSet, 0, len(results))
	for _, r := range results {
		out = append(out, r.DataSet)
	}
	return out, nil
}
