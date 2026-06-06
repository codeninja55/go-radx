package dicomweb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
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
// iterator that decodes each application/dicom part into a dataset. pathErr carries a
// path-construction error so the caller can build the path inline; when set, the iterator
// yields that single error. The bounded response body and the multipart reader cap the
// transfer against a hostile origin (PRD §9.3).
func (c *Client) retrieveInstances(ctx context.Context, path string, pathErr error) iter.Seq2[*dicom.DataSet, error] {
	return func(yield func(*dicom.DataSet, error) bool) {
		if pathErr != nil {
			yield(nil, pathErr)
			return
		}
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			yield(nil, err)
			return
		}
		req.Header.Set("Accept", acceptInstances(c.transferSyntaxes...))

		resp, err := c.httpClient.Do(req)
		if err != nil {
			yield(nil, c.transportError(http.MethodGet, path, err))
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			yield(nil, c.httpError(http.MethodGet, path, resp))
			return
		}
		if !isMultipartRelated(resp.Header.Get("Content-Type")) {
			yield(nil, fmt.Errorf("%w: WADO-RS response is not multipart/related", ErrNotAcceptable))
			return
		}

		mr, err := NewMultipartReader(c.boundedBody(resp), resp.Header.Get("Content-Type"))
		if err != nil {
			yield(nil, err)
			return
		}
		for {
			ct, part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				yield(nil, err)
				return
			}
			if mt := mediaTypeOf(ct); mt != mediaTypeDICOM {
				yield(nil, fmt.Errorf("%w: WADO-RS part media type %q is not application/dicom", ErrNotAcceptable, mt))
				return
			}
			ds, err := decodeInstance(part)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(ds, nil) {
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(http.MethodGet, path, err)
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

// RetrieveFrames fetches the given 1-based frames of an instance as a multipart/related body
// of application/octet-stream parts and returns each frame's raw octets in request order
// (PS3.18 §10.4.3). A frame number below 1 is rejected before any request is made.
func (c *Client) RetrieveFrames(ctx context.Context, p ResourcePath, frames ...int) ([][]byte, error) {
	path, err := p.Frames(frames...)
	if err != nil {
		return nil, err
	}
	return c.retrieveOctetParts(ctx, path, acceptOctetStream())
}

// RetrieveBulkData fetches every bulk-data value of an instance as a multipart/related body
// of application/octet-stream parts and returns each payload's raw octets (PS3.18 §10.4.4).
func (c *Client) RetrieveBulkData(ctx context.Context, p ResourcePath) ([][]byte, error) {
	path, err := p.BulkData()
	if err != nil {
		return nil, err
	}
	return c.retrieveOctetParts(ctx, path, acceptOctetStream())
}

// ResolveBulkDataURI fetches the octets a BulkDataURI references. WADO-RS returns a bulk-data
// value as a multipart/related body of application/octet-stream parts; the resolved value is
// the first such part. A relative reference is joined to the client's WithBulkDataBaseURL (or
// the origin base URL when none was set), so a BulkDataURI emitted in metadata against this
// origin is resolvable without the caller reconstructing the absolute URL. An absolute
// reference is fetched as given. The reference is never logged, since it carries resource
// UIDs (PRD §9.1).
func (c *Client) ResolveBulkDataURI(ctx context.Context, uri BulkDataURI) ([]byte, error) {
	target := c.absoluteBulkDataURL(string(uri))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: build bulkdata request: %w", err)
	}
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
	req.Header.Set("Accept", acceptOctetStream())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dicomweb: GET bulkdata: %w", sanitizeTransportError(err))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
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

// retrieveOctetParts issues a multipart/related WADO-RS GET against path with the given
// Accept header and returns each application/octet-stream part's raw octets. A non-octet
// part media type is a typed error rather than a silently-dropped part (PRD §9.2).
func (c *Client) retrieveOctetParts(ctx context.Context, path, accept string) ([][]byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(http.MethodGet, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
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
